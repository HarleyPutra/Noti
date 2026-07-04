package main

import (
	"context"
	"noti/auth"
	"noti/db"
	"noti/models"
	gosync "noti/sync"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type NoteService struct {
	app           *application.App
	TargetNoteID  string
	activeWindows map[string]*application.WebviewWindow
	mu            sync.Mutex
	trackerOnce   sync.Once

	activeTimers map[string]context.CancelFunc
	timerMu      sync.Mutex
	lastMenuTime time.Time
}

func NewNoteService(app *application.App) *NoteService {
	s := &NoteService{
		app:           app,
		activeWindows: make(map[string]*application.WebviewWindow),
		activeTimers:  make(map[string]context.CancelFunc),
	}
	go s.positionTracker()
	s.StartBackgroundSync()
	return s
}

func (s *NoteService) startTrackerSafely() {
	s.trackerOnce.Do(func() {
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

		for id, g := range currentGeoms {
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

func (n *NoteService) Login() (*auth.UserInfo, error) {
	return auth.Login()
}

func (s *NoteService) GetCurrentUser() *auth.UserInfo {
	return auth.CurrentUser
}

func (n *NoteService) Logout() {
	auth.Logout()

	for id, window := range n.activeWindows {
		window.Close()
		delete(n.activeWindows, id)
	}

	loginWindow := n.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "Noti — Sign In",
		Width:     400,
		Height:    500,
		URL:       "/#/login",
		Frameless: false,
	})
	loginWindow.Show()
}

func (s *NoteService) GetNotes(userID string) ([]models.Note, error) {
	s.startTrackerSafely()
	return db.GetNotes(userID)
}

func (s *NoteService) GetNote(noteID string) (*models.Note, error) {
	s.startTrackerSafely()
	return db.GetNoteByID(noteID)
}

func (s *NoteService) CreateNote(userID string) (*models.Note, error) {
	s.startTrackerSafely()

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
		URL:         "/#/?noteId=" + note.ID,
	})

	// THIS is what the Find & Replace accidentally broke! It's fixed now.
	win.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		s.mu.Lock()
		delete(s.activeWindows, note.ID)
		s.mu.Unlock()
	})

	s.mu.Lock()
	s.activeWindows[note.ID] = win
	s.mu.Unlock()

	win.Show()

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
		application.InvokeSync(func() {
			x, y := win.Position()
			w, h := win.Size()

			if w > 0 && h > 0 {
				note.PosX = x
				note.PosY = y
				note.Width = w
				note.Height = h
			}
		})
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
				application.InvokeSync(func() {
					win.Show()
					win.SetSize(n.Width, n.Height)
					win.SetPosition(n.PosX, n.PosY)
				})
				continue
			}

			win = s.app.Window.NewWithOptions(application.WebviewWindowOptions{
				Title:       "",
				Width:       n.Width,
				Height:      n.Height,
				X:           n.PosX,
				Y:           n.PosY,
				Frameless:   true,
				AlwaysOnTop: n.Pinned,
				URL:         "/#/?noteId=" + n.ID,
			})

			s.mu.Lock()
			s.activeWindows[n.ID] = win
			s.mu.Unlock()

			win.Show()

			application.InvokeSync(func() {
				win.SetSize(n.Width, n.Height)
				win.SetPosition(n.PosX, n.PosY)
			})
		}
	}
	return nil
}

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

func (s *NoteService) StartTimer(noteID string, minutes int) {
	s.timerMu.Lock()
	if cancel, exists := s.activeTimers[noteID]; exists {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.activeTimers[noteID] = cancel
	s.timerMu.Unlock()

	totalSeconds := minutes * 60

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				totalSeconds--

				s.app.Event.Emit("timer-tick-"+noteID, totalSeconds)

				if totalSeconds <= 0 {
					s.timerMu.Lock()
					delete(s.activeTimers, noteID)
					s.timerMu.Unlock()
					return
				}
			}
		}
	}()
}

func (s *NoteService) StopTimer(noteID string) {
	s.timerMu.Lock()
	defer s.timerMu.Unlock()

	if cancel, exists := s.activeTimers[noteID]; exists {
		cancel()
		delete(s.activeTimers, noteID)
	}
}

func (s *NoteService) ShowContextMenu(x int, y int, noteID string) {
	s.mu.Lock()
	s.lastMenuTime = time.Now()
	s.mu.Unlock()

	s.TargetNoteID = noteID
	windows := s.app.Window.GetAll()
	for _, win := range windows {
		if win.Name() == "ContextMenu" {
			win.SetPosition(x, y)
			win.Show()
			win.Focus()
			s.app.Event.Emit("menu-opened")
			break
		}
	}
}

func (s *NoteService) HideContextMenu() {
	s.mu.Lock()
	if time.Since(s.lastMenuTime) < 150*time.Millisecond {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	windows := s.app.Window.GetAll()
	for _, win := range windows {
		if win.Name() == "ContextMenu" {
			win.Hide()
			break
		}
	}
}

// Dynamically resizes the context menu to eliminate empty space!
func (s *NoteService) ResizeContextMenu(width int, height int) {
	windows := s.app.Window.GetAll()
	for _, win := range windows {
		if win.Name() == "ContextMenu" {
			win.SetSize(width, height)
			break
		}
	}
}

// Routes actions from the Ghost Window safely to the Main Note
func (s *NoteService) TriggerMenuAction(actionType string, actionValue string) {
	// Include the target ID as the 3rd argument in the Wails event array!
	s.app.Event.Emit("menu-action", actionType, actionValue, s.TargetNoteID)

	windows := s.app.Window.GetAll()
	for _, win := range windows {
		if win.Name() == "ContextMenu" {
			win.Hide()
			break
		}
	}
}

// Debug Tool: Scans the DB and purges corrupted/empty ghost notes
func (s *NoteService) PruneArtifactNotes() (int, error) {
	deletedCount := 0

	// 1. Figure out who is currently logged in
	user := s.GetCurrentUser()
	if user == nil {
		return 0, nil // If nobody is logged in, there's nothing to prune
	}

	// 2. Fetch only their notes
	allNotes, err := s.GetNotes(user.ID)
	if err != nil {
		return 0, err
	}

	// 3. Scan for and destroy the artifacts
	for _, note := range allNotes {
		// Identify artifacts: empty content, pure whitespace, or broken JSON brackets
		if note.Content == "" || note.Content == "{}" || note.Content == "null" {
			s.DeleteNote(note.ID)
			deletedCount++
		}
	}

	return deletedCount, nil
}

// StartBackgroundSync runs invisibly in the background, pushing changes to Google Drive
func (s *NoteService) StartBackgroundSync() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			user := s.GetCurrentUser()
			if user == nil {
				continue // Nobody logged in, skip sync
			}

			// 1. Check if there are any pending mutations
			queue, err := db.GetSyncQueue()
			if err != nil || len(queue) == 0 {
				continue // Nothing to do
			}

			// 2. Trigger the actual sync process
			// Note: because it's in a Goroutine, it will not freeze the Angular frontend while waiting for Google Drive
			err = s.SyncNow(user.ID)
			
			// 3. If the network didn't fail and Google Drive accepted it, clear the queue!
			if err == nil {
				for _, event := range queue {
					db.ClearFromQueue(event.ID)
					db.MarkSynced(event.NoteID)
				}
			} else {
				// If network fails it stays in the SQLite queue and 
				// we will try again in exactly 30 seconds.
				println("Background Sync Delayed (Network/Drive Error): ", err.Error())
			}
		}
	}()
}

func (s *NoteService) SyncNow(userID string) error {
	// 1. "The Hands" pull from Google Drive
	remoteNotes, err := gosync.Pull()
	if err != nil {
		return err // Network failure, background worker will try again later!
	}

	// 2. "The Hands" pull from SQLite
	localNotes, err := db.GetNotes(userID)
	if err != nil {
		return err
	}

	// 3. "The Brain" runs the Vector Clock math!
	finalMergedNotes := gosync.MergeNotes(localNotes, remoteNotes)

	// 4. "The Hands" save the mathematically perfect results back to SQLite
	for _, n := range finalMergedNotes {
		n.Synced = true // Everything in this list is about to go to the cloud
		db.UpsertNote(n) 
	}

	// 5. "The Hands" upload the perfect list back to Google Drive
	if len(finalMergedNotes) > 0 {
		err = gosync.Push(userID, finalMergedNotes) 
		if err != nil {
			return err
		}
	}

	// 6. Tell the Angular UI to stop spinning!
	s.app.Event.Emit("sync-complete")

	return nil
}