package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:dist
var distFS embed.FS

func main() {
	if err := run(); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	srv, err := startAPIServer(apiListenAddr, newAPIHandler())
	if err != nil {
		return fmt.Errorf("start API server on %s: %w", apiListenAddr, err)
	}
	defer srv.Stop()
	log.Printf("API server listening on %s (local client: %s)", apiListenAddr, apiClientURL)

	app := application.New(application.Options{
		Name:        "preset-wails",
		Description: "Minimal Wails v3 app",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(distFS),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		OnShutdown: srv.Stop,
	})

	go monitorAPIServer(app, srv)

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:        "Hello Wails V3",
		Width:        800,
		Height:       600,
		AlwaysOnTop: true,
	})

	setupTray(app, window)

	return app.Run()
}

func monitorAPIServer(app *application.App, srv *apiServer) {
	err := <-srv.Err()
	log.Printf("API server failed: %v; quitting application", err)
	app.Quit()
}

