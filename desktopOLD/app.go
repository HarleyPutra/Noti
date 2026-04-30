package main

import (
	"context"
	"desktop/auth"
	"desktop/db"
	"desktop/models"
	"desktop/sync"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	db.Init("./todos.db")
	auth.Init()
	auth.TryRestoreSession()
}

// ── Auth ──────────────────────────────────────────

func (a *App) Login() (*auth.UserInfo, error) {
	return auth.Login()
}

func (a *App) GetCurrentUser() *auth.UserInfo {
	return auth.CurrentUser
}

func (a *App) Logout() {
	auth.Logout()
}

// ── Dashboard Settings ─────────────────────────────────────────

// SavePresets receives the JSON string from Angular and saves it to SQLite
func (a *App) SavePresets(presetsJSON string) error {
	return db.SaveSetting("dashboard_presets", presetsJSON)
}

// LoadPresets reads the JSON string from SQLite and sends it back to Angular
func (a *App) LoadPresets() (string, error) {
	return db.GetSetting("dashboard_presets")
}

// ── Todos ─────────────────────────────────────────

func (a *App) GetTodos(userID string) ([]models.Todo, error) {
	return db.GetTodos(userID)
}

func (a *App) UpsertTodo(todo models.Todo) error {
	return db.UpsertTodo(todo)
}

// ── Sync ──────────────────────────────────────────

func (a *App) SyncNow(userID string) error {
	// Pull from Drive
	remote, err := sync.Pull()
	if err != nil {
		return err
	}

	if len(remote) > 0 {
		// Merge with local
		local, err := db.GetTodos(userID)
		if err != nil {
			return err
		}
		merged := sync.MergeTodos(local, remote)
		for _, t := range merged {
			db.UpsertTodo(t)
		}
	}

	// Push local changes up
	return sync.Push(userID)
}