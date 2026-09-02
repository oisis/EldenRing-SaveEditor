// Package desktop holds the Wails host bridge of SaveForge 2.0.
//
// The bridge is the only place where a public desktop method is declared. It
// owns no domain logic: every method delegates to a public backend endpoint and
// returns the endpoint result unchanged. The application root wires the bridge
// and stays a bootstrap and composition root only.
package desktop

import (
	"context"
	"errors"
	"sync"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/diagnostics"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/equipment"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/inventory"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const applicationCloseRequestedEvent = "application.close-requested"

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
	// chooseSaveFile is the host's native file dialog, injected by the
	// composition root so the bridge can be exercised without a real window. The
	// bridge holds no dialog implementation of its own.
	chooseSaveFile   SaveFileChooser
	chooseSaveTarget SaveTargetChooser

	// emitEvent delivers a backend event to the frontend. A nil value selects the
	// Wails runtime; only a test replaces it, so the bridge stays exercisable
	// without a real window.
	emitEvent eventEmitter

	// mutex guards hostContext only. It is not the session lock: SaveEngine owns
	// that, and no save state lives here.
	mutex sync.Mutex
	// hostContext is the Wails context handed to Startup by the application
	// lifecycle. It is per-bridge state rather than a package-level variable, so
	// two bridges never share one host, and it is only ever read through
	// hostContextOrNil.
	hostContext context.Context
}

// NewBridge builds the bridge with the application version, SaveEngine and
// GameCatalog supplied by its caller. No dependency is validated or replaced
// here: the public endpoints own their validation and the bridge must not
// duplicate it or create fallback dependencies.
func NewBridge(
	applicationVersion string,
	saveEngine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	chooseSaveFile SaveFileChooser,
	chooseSaveTarget ...SaveTargetChooser,
) *Bridge {
	bridge := &Bridge{
		applicationVersion: applicationVersion,
		saveEngine:         saveEngine,
		gameCatalog:        gameCatalog,
		chooseSaveFile:     chooseSaveFile,
	}
	if len(chooseSaveTarget) > 0 {
		bridge.chooseSaveTarget = chooseSaveTarget[0]
	}
	// The bridge is the only host-aware layer, so it is the one that subscribes
	// to SaveEngine's committed mutations and turns them into Wails events. The
	// engine never learns what a window is.
	if saveEngine != nil {
		saveEngine.SetSessionChangedSink(bridge.publishSessionChanged)
	}
	return bridge
}

// SelectSaveTarget opens the native Save As dialog after Review Changes has
// authorized the current revision. It returns the host path unchanged.
func (b *Bridge) SelectSaveTarget(suggestedName string) (string, error) {
	if b.chooseSaveTarget == nil {
		return "", bridgeError(errors.New("the native Save As dialog is not available"))
	}
	ctx := b.hostContextOrNil()
	if ctx == nil {
		return "", bridgeError(errors.New("the desktop host is not started yet"))
	}
	path, err := b.chooseSaveTarget(ctx, suggestedName)
	if err != nil {
		return "", bridgeError(err)
	}
	return path, nil
}

// Startup receives the Wails context from the application lifecycle. It is
// wired as OnStartup by the composition root and is the only way the host
// context enters the bridge; nothing here reads a package-level context.
func (b *Bridge) Startup(ctx context.Context) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.hostContext = ctx
}

// BeforeClose intercepts only a window close that would abandon unsaved work.
// React owns the Save / Discard / Cancel decision and receives a notification;
// a clean process is allowed to close immediately.
func (b *Bridge) BeforeClose(ctx context.Context) bool {
	if b.saveEngine == nil || !b.saveEngine.HasUnsavedChanges() {
		return false
	}
	emit := b.emitEvent
	if emit == nil {
		emit = runtime.EventsEmit
	}
	emit(ctx, applicationCloseRequestedEvent)
	return true
}

// QuitApplication completes a close request after the frontend has saved or
// discarded the active changes. The host lifecycle check still runs and will
// refuse the quit if another dirty session somehow remains.
func (b *Bridge) QuitApplication() error {
	ctx := b.hostContextOrNil()
	if ctx == nil {
		return bridgeError(errors.New("the desktop host is not started yet"))
	}
	runtime.Quit(ctx)
	return nil
}

func (b *Bridge) hostContextOrNil() context.Context {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.hostContext
}

// SelectSaveFile opens the host's native dialog and returns the path the user
// chose, exactly as the host reported it. Nothing here trims, resolves,
// recases or validates the path: recognising a save is SaveEngine's contract,
// reached later through LoadSave.
//
// Cancelling is an ordinary outcome, not an error: it returns an empty path and
// a nil error, and no session is created for it. This method opens no file,
// loads nothing and touches no session, so a cancelled or failed dialog leaves
// the application exactly as it was.
func (b *Bridge) SelectSaveFile() (string, error) {
	if b.chooseSaveFile == nil {
		return "", bridgeError(errors.New("the native file dialog is not available"))
	}
	ctx := b.hostContextOrNil()
	if ctx == nil {
		return "", bridgeError(errors.New("the desktop host is not started yet"))
	}
	path, err := b.chooseSaveFile(ctx)
	if err != nil {
		// Fail closed: a failed dialog yields no path at all, so a caller can
		// never load whatever partial value the host reported alongside its
		// error. The failure itself is reported in the shared error model.
		return "", bridgeError(err)
	}
	return path, nil
}

// GetApplicationInfo delegates to the GetApplicationInfo endpoint and returns
// its result and error unchanged. It declares no capability, no schema version
// and no fallback version of its own.
func (b *Bridge) GetApplicationInfo() (application.GetApplicationInfoResult, error) {
	return bridged(application.GetApplicationInfo(b.applicationVersion))
}

// LoadSave delegates to the LoadSave endpoint and returns its result and error
// unchanged. The endpoint and SaveEngine own all file, platform and source-kind
// behavior. All three values are forwarded exactly as received: the bridge
// supplies no default source kind, so a caller that states none is rejected by
// the endpoint rather than silently given "local" here.
func (b *Bridge) LoadSave(
	source string,
	expectedPlatform string,
	sourceKind string,
) (savesession.LoadSaveResult, error) {
	return bridged(savesession.LoadSave(b.saveEngine, source, expectedPlatform, sourceKind))
}

// GetLoadedSave delegates to the GetLoadedSave endpoint and returns its result
// and error unchanged.
func (b *Bridge) GetLoadedSave(saveSessionID string) (savesession.GetLoadedSaveResult, error) {
	return bridged(savesession.GetLoadedSave(b.saveEngine, saveSessionID))
}

// CloseSave delegates to the CloseSave endpoint and returns its error
// unchanged.
func (b *Bridge) CloseSave(saveSessionID string) error {
	return bridgeError(savesession.CloseSave(b.saveEngine, saveSessionID))
}

func (b *Bridge) GetOperationHistory(saveSessionID string) (savesession.GetOperationHistoryResult, error) {
	return bridged(savesession.GetOperationHistory(b.saveEngine, saveSessionID))
}

func (b *Bridge) UndoLastOperation(saveSessionID, expectedRevision string) (savesession.UndoLastOperationResult, error) {
	return bridged(savesession.UndoLastOperation(b.saveEngine, saveSessionID, expectedRevision))
}

func (b *Bridge) RedoLastOperation(saveSessionID, expectedRevision string) (savesession.RedoLastOperationResult, error) {
	return bridged(savesession.RedoLastOperation(b.saveEngine, saveSessionID, expectedRevision))
}

func (b *Bridge) RevertOperation(saveSessionID, operationID, expectedRevision string) (savesession.RevertOperationResult, error) {
	return bridged(savesession.RevertOperation(
		b.saveEngine, saveSessionID, operationID, expectedRevision))
}

func (b *Bridge) DiscardChanges(saveSessionID, expectedRevision string) (savesession.DiscardChangesResult, error) {
	return bridged(savesession.DiscardChanges(b.saveEngine, saveSessionID, expectedRevision))
}

func (b *Bridge) ValidateReviewChanges(saveSessionID, expectedRevision string) (savesession.ValidateReviewChangesResult, error) {
	return bridged(savesession.ValidateReviewChanges(b.saveEngine, saveSessionID, expectedRevision))
}

func (b *Bridge) Save(
	saveSessionID, expectedRevision, validationToken string,
	confirmWarnings, confirmBanRisk bool,
) (savesession.SaveResult, error) {
	return bridged(savesession.Save(
		b.saveEngine, saveSessionID, expectedRevision, validationToken,
		confirmWarnings, confirmBanRisk))
}

func (b *Bridge) SaveAs(
	saveSessionID, expectedRevision, validationToken string,
	confirmWarnings, confirmBanRisk bool,
	target string,
) (savesession.SaveAsResult, error) {
	return bridged(savesession.SaveAs(
		b.saveEngine, saveSessionID, expectedRevision, validationToken,
		confirmWarnings, confirmBanRisk, target))
}

func (b *Bridge) GetRecentFiles() (savesession.GetRecentFilesResult, error) {
	return bridged(savesession.GetRecentFiles(b.saveEngine))
}

func (b *Bridge) RecordRecentFile(saveSessionID string) (savesession.RecordRecentFileResult, error) {
	return bridged(savesession.RecordRecentFile(b.saveEngine, saveSessionID))
}

func (b *Bridge) RemoveRecentFile(path string) (savesession.RemoveRecentFileResult, error) {
	return bridged(savesession.RemoveRecentFile(b.saveEngine, path))
}

func (b *Bridge) ClearRecentFiles() error {
	return bridgeError(savesession.ClearRecentFiles(b.saveEngine))
}

func (b *Bridge) GetSaveLifecycleSettings() (savesession.GetSaveLifecycleSettingsResult, error) {
	return bridged(savesession.GetSaveLifecycleSettings(b.saveEngine))
}

func (b *Bridge) SetSaveLifecycleSettings(backupRetention int) (savesession.SetSaveLifecycleSettingsResult, error) {
	return bridged(savesession.SetSaveLifecycleSettings(b.saveEngine, backupRetention))
}

func (b *Bridge) GetRecoveryJournals() (savesession.GetRecoveryJournalsResult, error) {
	return bridged(savesession.GetRecoveryJournals(b.saveEngine))
}

func (b *Bridge) GetRecoveryJournal(journalID string) (savesession.GetRecoveryJournalResult, error) {
	return bridged(savesession.GetRecoveryJournal(b.saveEngine, journalID))
}

func (b *Bridge) RestoreRecoveryJournal(journalID string) (savesession.RestoreRecoveryJournalResult, error) {
	return bridged(savesession.RestoreRecoveryJournal(b.saveEngine, journalID))
}

func (b *Bridge) DiscardRecoveryJournal(journalID string) error {
	return bridgeError(savesession.DiscardRecoveryJournal(b.saveEngine, journalID))
}

func (b *Bridge) ExportRecoveryJournal(journalID, target string) error {
	return bridgeError(savesession.ExportRecoveryJournal(b.saveEngine, journalID, target))
}

// GetSaveValidationReport delegates to the GetSaveValidationReport endpoint and
// returns its result and error unchanged. The session, the slot and the scope
// are forwarded exactly as received: which scopes exist, how a scope is judged,
// what coverage means and which findings a report may carry are the endpoint's
// contract, and the bridge neither restates nor aggregates any of it.
func (b *Bridge) GetSaveValidationReport(
	saveSessionID string,
	characterID int,
	scope string,
) (diagnostics.GetSaveValidationReportResult, error) {
	return bridged(diagnostics.GetSaveValidationReport(
		b.saveEngine, b.gameCatalog, saveSessionID, characterID, scope))
}

// GetSaveCharacters delegates to the GetSaveCharacters endpoint and returns
// its result and error unchanged.
func (b *Bridge) GetSaveCharacters(saveSessionID string) (character.GetSaveCharactersResult, error) {
	return bridged(character.GetSaveCharacters(b.saveEngine, saveSessionID))
}

// GetCharacterProfile delegates to the GetCharacterProfile endpoint and
// returns its result and error unchanged.
func (b *Bridge) GetCharacterProfile(
	saveSessionID string,
	characterID int,
) (character.GetCharacterProfileResult, error) {
	return bridged(character.GetCharacterProfile(b.saveEngine, saveSessionID, characterID))
}

// GetCharacterStats delegates to the GetCharacterStats endpoint and returns
// its result and error unchanged.
func (b *Bridge) GetCharacterStats(
	saveSessionID string,
	characterID int,
) (character.GetCharacterStatsResult, error) {
	return bridged(character.GetCharacterStats(b.saveEngine, saveSessionID, characterID))
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
	return bridged(inventory.GetInventory(
		b.saveEngine, b.gameCatalog, saveSessionID, characterID, containerSection, page, pageSize))
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
	return bridged(inventory.GetStorage(
		b.saveEngine, b.gameCatalog, saveSessionID, characterID, containerSection, page, pageSize))
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
	return bridged(catalog.GetResources(
		b.gameCatalog, resourceType, family, capability, endpointID, search, page, pageSize))
}

// GetResourcePresentationSummaries delegates to the matching batch endpoint.
// Order, duplicates, exact identity validation and atomic failure all remain
// endpoint concerns; the bridge forwards the slice unchanged.
func (b *Bridge) GetResourcePresentationSummaries(
	identities []catalog.ResourcePresentationIdentity,
) (catalog.GetResourcePresentationSummariesResult, error) {
	return bridged(catalog.GetResourcePresentationSummaries(b.gameCatalog, identities))
}

// GetResource delegates to the GetResource endpoint and returns its result and
// error unchanged. The kind and the key are forwarded exactly as received: what
// a valid kind is, that neither value is trimmed, normalised or retried under
// another kind, and which of the four identity failures applies are the
// endpoint's contract, and the bridge must not restate any of it.
func (b *Bridge) GetResource(kind string, key string) (catalog.GetResourceResult, error) {
	return bridged(catalog.GetResource(b.gameCatalog, kind, key))
}

// GetItemVariants delegates to the GetItemVariants endpoint and returns its
// result and error unchanged. The kind and the key are forwarded exactly as
// received: that only the item kind carries variants, that neither value is
// trimmed, normalised or aliased, that an item without variants is a valid
// empty result and which identity failure applies are the endpoint's contract,
// and the bridge must not restate any of it.
func (b *Bridge) GetItemVariants(kind string, key string) (catalog.GetItemVariantsResult, error) {
	return bridged(catalog.GetItemVariants(b.gameCatalog, kind, key))
}

// GetEquipment delegates to the GetEquipment endpoint and returns its result
// and error unchanged. The session identifier and the slot index are forwarded
// exactly as received: matching a session, the slot range and what an inactive
// or residual slot exposes are the endpoint's contract, and the bridge must not
// restate any of it. The 22 raw fields are carried over without being named,
// reordered or resolved.
func (b *Bridge) GetEquipment(
	saveSessionID string,
	characterID int,
) (equipment.GetEquipmentResult, error) {
	return bridged(equipment.GetEquipment(b.saveEngine, saveSessionID, characterID))
}

// GetCharacterLoadout delegates to the coherent, catalog-resolved loadout
// endpoint and returns its result and error unchanged. Native layout,
// cross-structure validation, catalog resolution and revision capture remain
// backend concerns; the bridge only forwards the exact session and slot.
func (b *Bridge) GetCharacterLoadout(
	saveSessionID string,
	characterID int,
) (equipment.GetCharacterLoadoutResult, error) {
	return bridged(equipment.GetCharacterLoadout(
		b.saveEngine, b.gameCatalog, saveSessionID, characterID))
}

// GetQuickItems delegates to the GetQuickItems endpoint and returns its result
// and error unchanged. The ten raw records and the signed active-slot value are
// carried over exactly as the endpoint reports them.
func (b *Bridge) GetQuickItems(
	saveSessionID string,
	characterID int,
) (equipment.GetQuickItemsResult, error) {
	return bridged(equipment.GetQuickItems(b.saveEngine, saveSessionID, characterID))
}

// GetPouchItems delegates to the GetPouchItems endpoint and returns its result
// and error unchanged. The six raw records are carried over exactly as the
// endpoint reports them.
func (b *Bridge) GetPouchItems(
	saveSessionID string,
	characterID int,
) (equipment.GetPouchItemsResult, error) {
	return bridged(equipment.GetPouchItems(b.saveEngine, saveSessionID, characterID))
}

// GetPhysickMixture delegates to the GetPhysickMixture endpoint and returns its
// result and error unchanged. Both raw Crystal Tear identifiers are carried
// over exactly as the endpoint reports them.
func (b *Bridge) GetPhysickMixture(
	saveSessionID string,
	characterID int,
) (equipment.GetPhysickMixtureResult, error) {
	return bridged(equipment.GetPhysickMixture(b.saveEngine, saveSessionID, characterID))
}

// GetEquippedSpells delegates to the GetEquippedSpells endpoint and returns its
// result and error unchanged. The wired catalog is passed on as it is: the
// bridge builds, loads or substitutes no catalog of its own, and a missing one
// stays the endpoint's rejection.
func (b *Bridge) GetEquippedSpells(
	saveSessionID string,
	characterID int,
) (equipment.GetEquippedSpellsResult, error) {
	return bridged(equipment.GetEquippedSpells(b.saveEngine, b.gameCatalog, saveSessionID, characterID))
}
