// Command saveforge is the SaveForge 2.0 desktop application.
//
// This file is the bootstrap and composition root only: it owns the embedded
// frontend assets, the release version and the Wails window options, and wires
// the desktop bridge. It contains no endpoint logic.
package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
	"github.com/oisis/EldenRing-SaveForge/internal/catalogassets"
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
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("locate application data directory: %v", err)
	}
	stateDirectory := filepath.Join(configDirectory, "SaveForge")
	saveEngine := saveengine.NewWithOptions(saveengine.EngineOptions{
		StateDirectory: stateDirectory,
	})
	// The global Safety Profile is a host application setting, so it lives beside
	// the other host state and deliberately outside SaveEngine and every save
	// snapshot.
	safetyProfiles := safetyprofile.NewStore(stateDirectory)
	// The single process-wide GameCatalog, built from the embedded catalog data
	// the backend already ships. A failure here is a build or data defect, not a
	// user condition: the application stops instead of starting with a partial
	// or empty catalog that would silently change endpoint results.
	catalogData, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		log.Fatalf("load game catalog data: %v", err)
	}
	gameCatalog, err := gamecatalog.New(catalogData.Manifest, catalogData.Resources())
	if err != nil {
		log.Fatalf("build game catalog: %v", err)
	}
	// The native dialog is injected rather than reached for inside the bridge, so
	// the host capability has one owner and the bridge stays testable without a
	// real window.
	bridge := desktop.NewBridgeWithSettings(
		applicationVersion,
		saveEngine,
		gameCatalog,
		safetyProfiles,
		desktop.NewWailsSaveFileChooser(),
		desktop.NewWailsSaveTargetChooser(),
	)

	err = wails.Run(&options.App{
		Title:     "Elden Ring SaveForge",
		Width:     1280,
		Height:    820,
		MinWidth:  960,
		MinHeight: 640,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: catalogassets.New(catalogData),
		},
		// The bridge receives the Wails context through the ordinary lifecycle;
		// nothing in the application stores it in a package-level variable.
		OnStartup:     bridge.Startup,
		OnBeforeClose: bridge.BeforeClose,
		Bind:          []any{bridge},
	})
	if err != nil {
		log.Fatal(err)
	}
}
