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
	// Debug log
	f, _ := os.Create("debug.log")
	log.SetOutput(f)
	defer f.Close()

	godotenv.Load()
	db.Init("./noti.db")
	auth.Init()
	auth.TryRestoreSession()

	noteService := &NoteService{}

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

	// Login window — shown first if not authenticated
	loginWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
					Title: "Noti",
					Width: 1024,
					Height: 768,
				})
	loginWindow.SetTitle("Noti — Sign In")
	loginWindow.SetSize(400, 500)
	loginWindow.SetFrameless(false)
	loginWindow.SetURL("/login")
	loginWindow.Show()

	// If already logged in, open existing notes immediately
	if auth.CurrentUser != nil {
		loginWindow.Hide()
		notes, _ := db.GetNotes(auth.CurrentUser.ID)
		if len(notes) == 0 {
			// First time — create one default note
			noteService.CreateNote(auth.CurrentUser.ID)
		} else {
			for _, n := range notes {
				app.Window.NewWithOptions(application.WebviewWindowOptions{
					Title: "Noti",
					Width: 1024,
					Height: 768,
				}).
					SetFrameless(true).
					SetAlwaysOnTop(n.Pinned).
					SetURL("/?noteId=" + n.ID).
					Show()
			}
		}
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}