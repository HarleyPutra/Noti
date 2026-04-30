package main

import (
	"embed"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed frontend_placeholder
var assets embed.FS

func main() {
	f, _ := os.Create("debug.log")
	log.SetOutput(f)
	defer f.Close()

	godotenv.Load()
	app := NewApp()

	err := wails.Run(&options.App{
		Title:              "Todo App",
		Width:              1024,
		Height:             768,
		LogLevel:           logger.DEBUG,
		LogLevelProduction: logger.DEBUG,
		OnStartup:          app.startup,
		Bind:               []interface{}{app},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
	})
	if err != nil {
		log.Println("Wails error:", err)
	}
}