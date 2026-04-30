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
	activeWindows map[string]*application.WebviewWindow // We track windows ourselves
	mu            sync.Mutex                            // Thread safety for the map
}

func NewNoteService(app *application.App) *NoteService {
	return &NoteService{
		app:           app,
		activeWindows: make(map[string]*application.WebviewWindow),
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

func (s *NoteService) CreateNote(userID string) (*models.Note, error) {
	now := time.Now().UnixMilli()
	note := models.Note{
		ID:          uuid.New().String(),
		UserID:      userID,
		Title:       "New Note",
		Content:     "",
		Mode:        "list",
		Color:       "#875c5c",
		BgColor:     "#e8e0d5", // Your new background color field!
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

	// Spawn window EXACTLY how you had it originally
	win := s.app.Window.New()
	win.SetTitle("")
	win.SetSize(note.Width, note.Height)
	win.SetPosition(note.PosX, note.PosY)
	win.SetFrameless(true)
	win.SetAlwaysOnTop(note.Pinned)
	win.SetURL("/?noteId=" + note.ID)

	// Save the window safely into our custom map
	s.mu.Lock()
	s.activeWindows[note.ID] = win
	s.mu.Unlock()

	win.Show()
	return &note, nil
}

func (s *NoteService) UpdateNote(note models.Note) error {
	// THE HIJACK: Grab the real-time position/size right before saving!
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
	}

	note.UpdatedAt = time.Now().UnixMilli()
	note.Synced = false
	return db.UpsertNote(note)
}

func (s *NoteService) DeleteNote(id string) error {
	// Remove from our map and close the physical window
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
	// Safely get from our custom map
	s.mu.Lock()
	win, ok := s.activeWindows[noteID]
	s.mu.Unlock()

	if ok && win != nil {
		win.SetAlwaysOnTop(pinned)
	}
}

// Restore all active notes when the app starts
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