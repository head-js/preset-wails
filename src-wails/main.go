package main

import (
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type emptyFS struct{}

func (emptyFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }

func main() {
	app := application.New(application.Options{
		Name:        "preset-wails",
		Description: "Minimal Wails v3 app",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(emptyFS{}),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Hello Wails V3",
		Width:  800,
		Height: 600,
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
