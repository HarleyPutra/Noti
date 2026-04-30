package main

import (
	"noti/auth"
	"noti/db"
	"noti/models"
	gosync "noti/sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type NoteService struct {
	app *application.App
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
		Color:       "#6b3f3f",
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

	// Spawn a new window for this note
	win := s.app.Window.New()
	win.SetTitle("")
	win.SetSize(note.Width, note.Height)
	win.SetPosition(note.PosX, note.PosY)
	win.SetFrameless(true)
	win.SetAlwaysOnTop(note.Pinned)
	win.SetURL("/?noteId=" + note.ID)
	win.Show()

	return &note, nil
}

func (s *NoteService) UpdateNote(note models.Note) error {
	note.UpdatedAt = time.Now().UnixMilli()
	note.Synced = false
	return db.UpsertNote(note)
}

func (s *NoteService) DeleteNote(id string) error {
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
	// Safely unpack the window and the boolean 'ok' status
	if win, ok := s.app.Window.GetByName(noteID); ok {
		win.SetAlwaysOnTop(pinned)
	}
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