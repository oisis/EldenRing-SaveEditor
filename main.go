// Command saveforge is the SaveForge 2.0 desktop application.
//
// This file is the bootstrap and composition root only: it owns the embedded
// frontend assets, the release version and the Wails window options, and wires
// the desktop bridge. It contains no endpoint logic.
package main

import (
	"embed"
	"log"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
	"github.com/oisis/EldenRing-SaveForge/internal/desktop"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// applicationVersion is the single Go-side hand-off point for the release
// version. A production build overrides it from the Makefile VERSION with
// -ldflags "-X main.applicationVersion=$(VERSION)"; a development build keeps
// "dev". No release number is written into Go or TypeScript source.
var applicationVersion = "dev"

func main() {
	saveEngine := saveengine.New()
	bridge := desktop.NewBridge(applicationVersion, saveEngine)

	err := wails.Run(&options.App{
		Title:       "Elden Ring SaveForge",
		Width:       1280,
		Height:      820,
		MinWidth:    960,
		MinHeight:   640,
		AssetServer: &assetserver.Options{Assets: assets},
		Bind:        []any{bridge},
	})
	if err != nil {
		log.Fatal(err)
	}
}
