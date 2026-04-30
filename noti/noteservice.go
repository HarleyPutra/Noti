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
	// Start the Background Tracker
	go s.positionTracker()
	return s
}

func (s *NoteService) positionTracker() {
	ticker := time.NewTicker(2 * time.Second) 
	for range ticker.C {
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

		for id, g := range currentGeoms {
			note, err := db.GetNoteByID(id)
			if err == nil {
				// Only write to the JSON file if something actually moved
				if note.PosX != g.x || note.PosY != g.y || note.Width != g.w || note.Height != g.h {
					note.PosX = g.x
					note.PosY = g.y
					note.Width = g.w
					note.Height = g.h
					db.UpsertNote(*note) 
				}
			}
		}
	}
}

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

// GetNote fetches a single note directly for Angular
func (s *NoteService) GetNote(noteID string) (*models.Note, error) {
	return db.GetNoteByID(noteID)
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
	
	// Create the JSON file
	if err := db.UpsertNote(note); err != nil {
		return nil, err
	}

	win := s.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       "",
		Width:       note.Width,
		Height:      note.Height,
		X:           note.PosX,
		Y:           note.PosY,
		Frameless:   true,
		AlwaysOnTop: note.Pinned,
		URL:         "/?noteId=" + note.ID,
	})

	s.mu.Lock()
	s.activeWindows[note.ID] = win
	s.mu.Unlock()

	win.Show()
	return &note, nil
}

func (s *NoteService) UpdateNote(note models.Note) error {
	s.mu.Lock()
	win, ok := s.activeWindows[note.ID]
	s.mu.Unlock()

	if ok && win != nil {
		x, y := win.Position()
		w, h := win.Size()
		note.PosX = x
		note.PosY = y
		note.Width = w
		note.Height = h
	} else {
		existing, err := db.GetNoteByID(note.ID)
		if err == nil {
			note.PosX = existing.PosX
			note.PosY = existing.PosY
			note.Width = existing.Width
			note.Height = existing.Height
		}
	}

	note.UpdatedAt = time.Now().UnixMilli()
	note.Synced = false
	return db.UpsertNote(note) // Saves directly to JSON
}

func (s *NoteService) DeleteNote(id string) error {
	s.mu.Lock()
	if win, ok := s.activeWindows[id]; ok {
		win.Close()
		delete(s.activeWindows, id)
	}
	s.mu.Unlock()

	note, err := db.GetNoteByID(id)
	if err == nil {
		// IMPORTANT: We do not delete the JSON file! We mark it deleted.
		// If we delete the file, the sync engine won't know to tell Android to delete it too.
		note.Deleted = true
		note.UpdatedAt = time.Now().UnixMilli()
		note.Synced = false
		return db.UpsertNote(*note)
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
			win := s.app.Window.NewWithOptions(application.WebviewWindowOptions{
				Title:       "",
				Width:       n.Width,
				Height:      n.Height,
				X:           n.PosX,
				Y:           n.PosY,
				Frameless:   true,
				AlwaysOnTop: n.Pinned,
				URL:         "/?noteId=" + n.ID,
			})

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