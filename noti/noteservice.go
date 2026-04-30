package main

import (
	"noti/auth"
	"noti/db"
	"noti/models"
	gosync "noti/sync"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type NoteService struct {
	app           *application.App
	activeWindows map[string]*application.WebviewWindow
	mu            sync.Mutex
}

func NewNoteService(app *application.App) *NoteService {
	s := &NoteService{
		app:           app,
		activeWindows: make(map[string]*application.WebviewWindow),
	}
	
	// Start the Background Tracker immediately!
	go s.positionTracker()
	
	return s
}

// ---------------------------------------------------------
// THE GHOST TRACKER: Runs silently in the background
// ---------------------------------------------------------
func (s *NoteService) positionTracker() {
	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
	for range ticker.C {
		// Step 1: Safely grab the current coordinates of all windows
		type geom struct{ x, y, w, h int }
		currentGeoms := make(map[string]geom)

		s.mu.Lock()
		for id, win := range s.activeWindows {
			if win != nil {
				x, y := win.Position()
				w, h := win.Size()
				currentGeoms[id] = geom{x, y, w, h}
			}
		}
		s.mu.Unlock()

		// Step 2: Compare against the DB and save ONLY if they changed
		for id, g := range currentGeoms {
			note, err := db.GetNoteByID(id)
			if err == nil {
				if note.PosX != g.x || note.PosY != g.y || note.Width != g.w || note.Height != g.h {
					note.PosX = g.x
					note.PosY = g.y
					note.Width = g.w
					note.Height = g.h
					// Silently save the new coordinates
					db.UpsertNote(*note)
				}
			}
		}
	}
}
// ---------------------------------------------------------

func (s *NoteService) Login() (*auth.UserInfo, error) {
	return auth.Login()
}

func (s *NoteService) GetCurrentUser() *auth.UserInfo {
	return auth.CurrentUser
}

func (s *NoteService) Logout() {
	auth.Logout()
}

func (s *NoteService) GetNotes(userID string) ([]models.Note, error) {
	return db.GetNotes(userID)
}

func (s *NoteService) CreateNote(userID string) (*models.Note, error) {
	now := time.Now().UnixMilli()
	note := models.Note{
		ID:          uuid.New().String(),
		UserID:      userID,
		Title:       "New Note",
		Content:     "",
		Mode:        "list",
		Color:       "#875c5c",
		BgColor:     "#e8e0d5", 
		Pinned:      false,
		Width:       400,
		Height:      500,
		PosX:        100,
		PosY:        100,
		CreatedAt:   now,
		UpdatedAt:   now,
		Synced:      false,
		Version:     1,
		VectorClock: `{"` + userID + `":1}`,
	}
	if err := db.UpsertNote(note); err != nil {
		return nil, err
	}

	win := s.app.Window.New()
	win.SetTitle("")
	win.SetSize(note.Width, note.Height)
	win.SetPosition(note.PosX, note.PosY)
	win.SetFrameless(true)
	win.SetAlwaysOnTop(note.Pinned)
	win.SetURL("/?noteId=" + note.ID)

	// Register window for tracking
	s.mu.Lock()
	s.activeWindows[note.ID] = win
	s.mu.Unlock()

	win.Show()
	return &note, nil
}

func (s *NoteService) UpdateNote(note models.Note) error {
	note.UpdatedAt = time.Now().UnixMilli()
	note.Synced = false
	return db.UpsertNote(note)
}

func (s *NoteService) DeleteNote(id string) error {
	// Deregister window and physically close it
	s.mu.Lock()
	if win, ok := s.activeWindows[id]; ok {
		win.Close()
		delete(s.activeWindows, id)
	}
	s.mu.Unlock()

	notes, err := db.GetNotes("")
	if err != nil {
		return err
	}
	for _, n := range notes {
		if n.ID == id {
			n.Deleted = true
			n.UpdatedAt = time.Now().UnixMilli()
			n.Synced = false
			return db.UpsertNote(n)
		}
	}
	return nil
}

func (s *NoteService) SetAlwaysOnTop(noteID string, pinned bool) {
	s.mu.Lock()
	win, ok := s.activeWindows[noteID]
	s.mu.Unlock()

	if ok && win != nil {
		win.SetAlwaysOnTop(pinned)
	}
}

func (s *NoteService) RestoreWindows(userID string) error {
	notes, err := db.GetNotes(userID)
	if err != nil {
		return err
	}
	for _, n := range notes {
		if !n.Deleted {
			win := s.app.Window.New()
			win.SetTitle("")
			win.SetSize(n.Width, n.Height)
			win.SetPosition(n.PosX, n.PosY)
			win.SetFrameless(true)
			win.SetAlwaysOnTop(n.Pinned)
			win.SetURL("/?noteId=" + n.ID)

			// Register window for tracking
			s.mu.Lock()
			s.activeWindows[n.ID] = win
			s.mu.Unlock()

			win.Show()
		}
	}
	return nil
}

func (s *NoteService) SyncNow(userID string) error {
	remote, err := gosync.Pull()
	if err != nil {
		return err
	}
	if len(remote) > 0 {
		local, _ := db.GetNotes(userID)
		merged := gosync.MergeNotes(local, remote)
		for _, n := range merged {
			db.UpsertNote(n)
		}
	}
	return gosync.Push(userID)
}