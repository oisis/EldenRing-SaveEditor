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
	"runtime"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
	"github.com/oisis/EldenRing-SaveForge/backend/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/hostsettings"
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
	// The two Save behavior preferences live beside the Safety Profile: they are
	// host settings, so they stay outside SaveEngine and every save snapshot.
	hostSettings := hostsettings.NewStore(stateDirectory)
	// The Build Templates library and the deployment configuration each get one
	// process-wide store rooted in the same host state directory. Creating either
	// one per call would give the library a second index and the deployment
	// configuration a second set of targets and trusted host keys.
	buildTemplateStore := buildtemplates.NewStore(filepath.Join(stateDirectory, "templates"))
	deploymentStore := deployment.NewStore(stateDirectory)
	// The single process-wide diagnostic service. It owns Debug Mode, the safe
	// record buffer the console reads and the local JSONL sink, which lives in a
	// logs subdirectory of the same host state directory and never beside a save
	// or inside the installation.
	diagnosticService := diagnostics.NewService(diagnostics.Options{
		Directory: filepath.Join(stateDirectory, "logs"),
	})
	defer diagnosticService.Close()
	// The single process-wide GameCatalog, built from the embedded catalog data
	// the backend already ships. A failure here is a build or data defect, not a
	// user condition: the application stops instead of starting with a partial
	// or empty catalog that would silently change endpoint results.
	catalogFiles := catalogdata.Files()
	catalogData, err := loader.LoadFS(catalogFiles)
	if err != nil {
		log.Fatalf("load game catalog data: %v", err)
	}
	// The network parameters and the appearance presets are data sets of their
	// own rather than catalog resources, so they are loaded beside the documents
	// and handed to the single catalog constructor. Without them the Advanced and
	// Appearance endpoints answer from an empty catalog.
	networkPresets, err := gamecatalog.LoadNetworkParams(catalogFiles)
	if err != nil {
		log.Fatalf("load network parameters: %v", err)
	}
	appearancePresets, err := gamecatalog.LoadAppearancePresets(catalogFiles)
	if err != nil {
		log.Fatalf("load appearance presets: %v", err)
	}
	gameCatalog, err := gamecatalog.NewWithData(gamecatalog.CatalogData{
		Manifest:          catalogData.Manifest,
		Resources:         catalogData.Resources(),
		NetworkPresets:    networkPresets,
		AppearancePresets: appearancePresets,
	})
	if err != nil {
		log.Fatalf("build game catalog: %v", err)
	}
	// The native dialog is injected rather than reached for inside the bridge, so
	// the host capability has one owner and the bridge stays testable without a
	// real window.
	bridge := desktop.NewBridgeWithDependencies(desktop.Dependencies{
		ApplicationVersion: applicationVersion,
		SaveEngine:         saveEngine,
		GameCatalog:        gameCatalog,
		SafetyProfiles:     safetyProfiles,
		HostSettings:       hostSettings,
		BuildTemplates:     buildTemplateStore,
		DeploymentStore:    deploymentStore,
		Diagnostics:        diagnosticService,
		ChooseSaveFile:     desktop.NewWailsSaveFileChooser(),
		ChooseSaveTarget:   desktop.NewWailsSaveTargetChooser(),
		ChooseDocument:     desktop.NewWailsDocumentChooser(),
	})

	// The first record of every launch. Debug Mode starts disabled on every
	// launch, so this info record is what a fresh log begins with.
	applicationInfo, _ := application.GetApplicationInfo(applicationVersion)
	diagnosticService.Log(diagnostics.Entry{
		Event:    diagnostics.EventApplicationStarted,
		Version:  applicationVersion,
		Build:    applicationInfo.Build,
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	})

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
		OnShutdown:    bridge.Shutdown,
		Bind:          []any{bridge},
	})
	if err != nil {
		log.Fatal(err)
	}
}
