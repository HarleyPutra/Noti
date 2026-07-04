package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	"noti/auth"
	"noti/db"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"golang.design/x/hotkey"
)

//go:embed frontend/dist
var assets embed.FS
//go:embed .env
var envText string


func main() {
	fmt.Println("--- DEBUG: ENV FILE LENGTH ---", len(envText))

	envMap, err := godotenv.Unmarshal(envText)
	if err != nil {
		// If it crashes, scream loudly in the terminal
		fmt.Println("--- CRITICAL ENV ERROR ---:", err)
	} else {
		for key, value := range envMap {
			os.Setenv(key, value)
		}
		// Prove that it made it into the system environment
		fmt.Println("--- DEBUG: LOADED CLIENT ID ---", os.Getenv("GOOGLE_CLIENT_ID"))
	}

	configDir, _ := os.UserConfigDir()
    appDir := configDir + "/Noti"
    os.MkdirAll(appDir, os.ModePerm)
    f, _ := os.OpenFile(appDir+"/debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    log.SetOutput(f)
    defer f.Close()

	if err := db.InitStore(); err != nil {
		log.Println("Failed to initialize storage:", err.Error())
	}

	auth.Init()
	auth.TryRestoreSession()

	noteService := &NoteService{
		activeWindows: make(map[string]*application.WebviewWindow),
		activeTimers:  make(map[string]context.CancelFunc), // <-- ADD THIS LINE!
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
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	noteService.app = app
	startGlobalHotkey(noteService)

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

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(e *application.ApplicationEvent) {
		if auth.CurrentUser == nil {
			loginWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
				Title:     "Noti — Sign In",
				Width:     400,
				Height:    500,
				URL:       "/#/login",
				Frameless: false,
			})
			loginWindow.Show()
		} else {
			notes, _ := db.GetNotes(auth.CurrentUser.ID)
			if len(notes) == 0 {
				noteService.CreateNote(auth.CurrentUser.ID)
			} else {
				noteService.RestoreWindows(auth.CurrentUser.ID)
			}
		}
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "ContextMenu",
		URL: "/#/menu", 
		Frameless: true,
		AlwaysOnTop: true,
		Hidden: true,
		Width: 230,
		Height: 400, 
		
		BackgroundType: application.BackgroundTypeTransparent, 
		
		BackgroundColour: application.NewRGBA(0, 0, 0, 0), 

		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true, 
		},
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func startGlobalHotkey(noteService *NoteService) {
	go func() {
		hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModAlt}, hotkey.KeyN)
		err := hk.Register()
		if err != nil {
			fmt.Println("Hotkey registration failed:", err)
			return
		}
		for range hk.Keydown() {
			if auth.CurrentUser != nil {
				application.InvokeSync(func() {
					noteService.CreateNote(auth.CurrentUser.ID)
				})
			}
		}
	}()
}