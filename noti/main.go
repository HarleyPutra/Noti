package main

import (
	"embed"
	"log"
	"noti/auth"
	"noti/db"
	"os"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	f, _ := os.Create("debug.log")
	log.SetOutput(f)
	defer f.Close()

	godotenv.Load()
	
	// Initialize the JSON File System
	err := db.InitStore()
	if err != nil {
		println("Failed to initialize storage:", err.Error())
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
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	noteService.app = app
	go noteService.positionTracker()

	loginWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Noti",
		Width:  400,
		Height: 500,
	})
	loginWindow.SetTitle("Noti — Sign In")
	loginWindow.SetFrameless(false)
	loginWindow.SetURL("/login")
	loginWindow.Show()

	if auth.CurrentUser != nil {
		loginWindow.Hide()
		notes, _ := db.GetNotes(auth.CurrentUser.ID)
		
		if len(notes) == 0 {
			noteService.CreateNote(auth.CurrentUser.ID)
		} else {
			noteService.RestoreWindows(auth.CurrentUser.ID)
		}
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}