package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"noti/auth"
	"noti/db"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events" // We need this for the startup event!
	"golang.design/x/hotkey"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	f, _ := os.Create("debug.log")
	log.SetOutput(f)
	defer f.Close()

	godotenv.Load()

	// Initialize the JSON File System
	if err := db.InitStore(); err != nil {
		log.Println("Failed to initialize storage:", err.Error())
	}

	auth.Init()
	auth.TryRestoreSession()

	noteService := &NoteService{
		activeWindows: make(map[string]*application.WebviewWindow),
	}

	app := application.New(application.Options{
		Name:        "Noti",
		Description: "Floating notebook app",
		Services: []application.Service{
			application.NewService(noteService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// CRITICAL: Set to false so the tray daemon stays alive when all notes are hidden
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	noteService.app = app
	startGlobalHotkey(noteService)

	// SYSTEM TRAY SETUP
	tray := app.SystemTray.New()
	menu := app.NewMenu()

	menu.Add("New Note").OnClick(func(ctx *application.Context) {
		if auth.CurrentUser != nil {
			noteService.CreateNote(auth.CurrentUser.ID)
		}
	})

	menu.Add("Show All Notes").OnClick(func(ctx *application.Context) {
		if auth.CurrentUser != nil {
			noteService.RestoreWindows(auth.CurrentUser.ID)
		}
	})

	menu.AddSeparator()

	menu.Add("Quit Noti").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	tray.SetMenu(menu)

	// LIFECYCLE: ON STARTUP
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(e *application.ApplicationEvent) {
		if auth.CurrentUser == nil {
			// User is NOT logged in: Only spawn the Login Window
			loginWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
				Title:     "Noti — Sign In",
				Width:     400,
				Height:    500,
				URL:       "/login",
				Frameless: false,
			})
			loginWindow.Show()
		} else {
			// User IS logged in: Spawn the notes safely
			notes, _ := db.GetNotes(auth.CurrentUser.ID)
			if len(notes) == 0 {
				noteService.CreateNote(auth.CurrentUser.ID)
			} else {
				noteService.RestoreWindows(auth.CurrentUser.ID)
			}
		}
	})

	// Start the engine
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func startGlobalHotkey(noteService *NoteService) {
	go func() {
		// Register Ctrl + Alt + N
		hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModAlt}, hotkey.KeyN)
		err := hk.Register()
		if err != nil {
			fmt.Println("Hotkey registration failed:", err)
			return
		}

		// Infinite loop listening for the hotkey
		for range hk.Keydown() {
			if auth.CurrentUser != nil {
				// Spawn a new note safely on the main thread
				application.InvokeSync(func() {
					noteService.CreateNote(auth.CurrentUser.ID)
				})
			}
		}
	}()
}