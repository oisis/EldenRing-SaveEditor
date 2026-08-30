// Package desktop holds the Wails host bridge of SaveForge 2.0.
//
// The bridge is the only place where a public desktop method is declared. It
// owns no domain logic: every method delegates to a public backend endpoint and
// returns the endpoint result unchanged. The application root wires the bridge
// and stays a bootstrap and composition root only.
package desktop

import (
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/inventory"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Bridge is the struct bound to Wails. Its exported methods are the desktop
// bridge surface reachable from the frontend.
type Bridge struct {
	// applicationVersion is injected by the composition root, which owns the
	// single source of the release version. The bridge neither reads a build
	// file nor defines a version constant of its own.
	applicationVersion string
	// saveEngine is the single process-wide engine supplied by the composition
	// root. The bridge only passes it to public endpoints and owns no session or
	// save-data behavior of its own.
	saveEngine *saveengine.Engine
	// gameCatalog is the single process-wide catalog supplied by the composition
	// root. The bridge only passes it to public endpoints and resolves nothing
	// through it itself.
	gameCatalog *gamecatalog.Catalog
}

// NewBridge builds the bridge with the application version, SaveEngine and
// GameCatalog supplied by its caller. No dependency is validated or replaced
// here: the public endpoints own their validation and the bridge must not
// duplicate it or create fallback dependencies.
func NewBridge(
	applicationVersion string,
	saveEngine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
) *Bridge {
	return &Bridge{
		applicationVersion: applicationVersion,
		saveEngine:         saveEngine,
		gameCatalog:        gameCatalog,
	}
}

// GetApplicationInfo delegates to the GetApplicationInfo endpoint and returns
// its result and error unchanged. It declares no capability, no schema version
// and no fallback version of its own.
func (b *Bridge) GetApplicationInfo() (application.GetApplicationInfoResult, error) {
	return application.GetApplicationInfo(b.applicationVersion)
}

// LoadSave delegates to the LoadSave endpoint and returns its result and error
// unchanged. The endpoint and SaveEngine own all file and platform behavior.
func (b *Bridge) LoadSave(source string, expectedPlatform string) (savesession.LoadSaveResult, error) {
	return savesession.LoadSave(b.saveEngine, source, expectedPlatform)
}

// GetLoadedSave delegates to the GetLoadedSave endpoint and returns its result
// and error unchanged.
func (b *Bridge) GetLoadedSave(saveSessionID string) (savesession.GetLoadedSaveResult, error) {
	return savesession.GetLoadedSave(b.saveEngine, saveSessionID)
}

// CloseSave delegates to the CloseSave endpoint and returns its error
// unchanged.
func (b *Bridge) CloseSave(saveSessionID string) error {
	return savesession.CloseSave(b.saveEngine, saveSessionID)
}

// GetSaveCharacters delegates to the GetSaveCharacters endpoint and returns
// its result and error unchanged.
func (b *Bridge) GetSaveCharacters(saveSessionID string) (character.GetSaveCharactersResult, error) {
	return character.GetSaveCharacters(b.saveEngine, saveSessionID)
}

// GetCharacterProfile delegates to the GetCharacterProfile endpoint and
// returns its result and error unchanged.
func (b *Bridge) GetCharacterProfile(
	saveSessionID string,
	characterID int,
) (character.GetCharacterProfileResult, error) {
	return character.GetCharacterProfile(b.saveEngine, saveSessionID, characterID)
}

// GetCharacterStats delegates to the GetCharacterStats endpoint and returns
// its result and error unchanged.
func (b *Bridge) GetCharacterStats(
	saveSessionID string,
	characterID int,
) (character.GetCharacterStatsResult, error) {
	return character.GetCharacterStats(b.saveEngine, saveSessionID, characterID)
}

// GetInventory delegates to the GetInventory endpoint and returns its result
// and error unchanged. Section, page and page size are forwarded exactly as
// received: paging and section resolution are the endpoint's contract.
func (b *Bridge) GetInventory(
	saveSessionID string,
	characterID int,
	containerSection string,
	page int,
	pageSize int,
) (inventory.GetInventoryResult, error) {
	return inventory.GetInventory(
		b.saveEngine, b.gameCatalog, saveSessionID, characterID, containerSection, page, pageSize)
}

// GetStorage delegates to the GetStorage endpoint and returns its result and
// error unchanged. It neither merges Storage with Inventory nor shares any
// state with GetInventory.
func (b *Bridge) GetStorage(
	saveSessionID string,
	characterID int,
	containerSection string,
	page int,
	pageSize int,
) (inventory.GetStorageResult, error) {
	return inventory.GetStorage(
		b.saveEngine, b.gameCatalog, saveSessionID, characterID, containerSection, page, pageSize)
}

// GetResources delegates to the GetResources endpoint and returns its result
// and error unchanged. Every filter, the search text and the paging are
// forwarded exactly as received: which values are accepted, which are rejected,
// what an empty filter means and which defaults apply are the endpoint's
// contract, and the bridge must not restate any of it.
func (b *Bridge) GetResources(
	resourceType string,
	family string,
	capability string,
	endpointID string,
	search string,
	page int,
	pageSize int,
) (catalog.GetResourcesResult, error) {
	return catalog.GetResources(
		b.gameCatalog, resourceType, family, capability, endpointID, search, page, pageSize)
}
