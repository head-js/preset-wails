package main

import (
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const apiVersion = "0.1.0"

type emptyFS struct{}

func (emptyFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }

func newAPIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"` + apiVersion + `"}`))
	})
	return mux
}

func newAppHandler(asset, api http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		asset.ServeHTTP(w, r)
	})
}

func main() {
	app := application.New(application.Options{
		Name:        "preset-wails",
		Description: "Minimal Wails v3 app",
		Assets: application.AssetOptions{
			Handler: newAppHandler(
				application.AssetFileServerFS(emptyFS{}),
				newAPIHandler(),
			),
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