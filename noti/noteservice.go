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
	trackerOnce	  sync.Once
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

func (s *NoteService) startTrackerSafely() {
	s.trackerOnce.Do(func() {
		// 1. The app is officially alive! Angular just connected.
		// Let's force all restored windows to snap to their saved JSON positions.
		s.mu.Lock()
		windowsCopy := make(map[string]*application.WebviewWindow)
		for id, win := range s.activeWindows {
			windowsCopy[id] = win
		}
		s.mu.Unlock()

		for id, win := range windowsCopy {
			if note, err := db.GetNoteByID(id); err == nil {
				application.InvokeSync(func() {
					win.SetSize(note.Width, note.Height)
					win.SetPosition(note.PosX, note.PosY)
				})
			}
		}

		// 2. Start the background ghost tracker
		go s.positionTracker()
	})
}

func (s *NoteService) positionTracker() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		type geom struct{ x, y, w, h int }
		currentGeoms := make(map[string]geom)

		windowsCopy := make(map[string]*application.WebviewWindow)
		s.mu.Lock()
		for id, win := range s.activeWindows {
			windowsCopy[id] = win
		}
		s.mu.Unlock()

		for id, win := range windowsCopy {
			if win != nil {
				application.InvokeSync(func() {
					x, y := win.Position()
					w, h := win.Size()
					currentGeoms[id] = geom{x, y, w, h}
				})
			}
		}

		// 3. Save to JSON if anything actually moved
		for id, g := range currentGeoms {
			// Prevent saving 0,0 if the window was closed during the check
			if g.w > 0 && g.h > 0 {
				note, err := db.GetNoteByID(id)
				if err == nil {
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
	s.startTrackerSafely()
	return db.GetNotes(userID)
}

// GetNote fetches a single note directly for Angular
func (s *NoteService) GetNote(noteID string) (*models.Note, error) {
	s.startTrackerSafely()
	return db.GetNoteByID(noteID)
}

func (s *NoteService) CreateNote(userID string) (*models.Note, error) {
	s.startTrackerSafely() // Wake up tracker

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

	// Safe to do instantly, because Angular triggered this function.
	application.InvokeSync(func() {
		win.SetSize(note.Width, note.Height)
		win.SetPosition(note.PosX, note.PosY)
	})

	return &note, nil
}

func (s *NoteService) UpdateNote(note models.Note) error {
	s.mu.Lock()
	win, ok := s.activeWindows[note.ID]
	s.mu.Unlock()

	if ok && win != nil {
		// Override Angular's stale coordinates with the live window coordinates
		application.InvokeSync(func() {
			x, y := win.Position()
			w, h := win.Size()
			
			// Only apply if Windows gave us real numbers
			if w > 0 && h > 0 {
				note.PosX = x
				note.PosY = y
				note.Width = w
				note.Height = h
			}
		})
	} else {
		// Fallback: If the window is missing, grab the last known position from JSON
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
	return db.UpsertNote(note)
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
			s.mu.Lock()
			win, exists := s.activeWindows[n.ID]
			s.mu.Unlock()

			if exists && win != nil {
				// 1. The window already exists! (It was just hidden).
				// Show it, and force it back to its exact saved coordinates.
				application.InvokeSync(func() {
					win.Show()
					win.SetSize(n.Width, n.Height)
					win.SetPosition(n.PosX, n.PosY)
				})
				continue // Skip the rest of the loop so we don't duplicate it!
			}

			// 2. The window doesn't exist yet, so let's create it.
			win = s.app.Window.NewWithOptions(application.WebviewWindowOptions{
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

			// 3. Force Windows OS to obey the coordinates instantly
			application.InvokeSync(func() {
				win.SetSize(n.Width, n.Height)
				win.SetPosition(n.PosX, n.PosY)
			})
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

// HideWindow turns the window invisible instead of killing the Angular process.
// This allows it to instantly reappear later without recompiling the UI.
func (s *NoteService) HideWindow(noteID string) {
	s.mu.Lock()
	win, ok := s.activeWindows[noteID]
	s.mu.Unlock()

	if ok && win != nil {
		application.InvokeSync(func() {
			win.Hide()
		})
	}
}