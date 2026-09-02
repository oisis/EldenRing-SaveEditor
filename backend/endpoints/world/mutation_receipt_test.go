package world

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Stage 3b.3c embeds the shared MutationReceipt in the public results of the
// fifteen World mutations. The interesting invariants are that every result
// carries the receipt the central commit path produced, that each of the fifteen
// keeps its own operationKind — the two Spectral Steed entry points share one
// SaveEngine result type, yet keep separate kinds, separate changedScopes and
// separate attireKey values, and must never be confused — that a commit which
// writes no byte is still a commit with a full receipt and a new revision, and
// that a rejected mutation returns nothing at all.
//
// The expected scope lists are stated literally on purpose: reading them back
// from the SaveEngine table would compare the implementation with itself.
var (
	// The mutations that write event flags, gesture records or the unlocked
	// region list and nothing else.
	worldFlagScopes = []string{"save.session", "world.flags", "diagnostics.report"}
	// The mutations that keep a companion InventoryHeld item in step with their
	// flags. The shared removal planner refuses a record an Equipment, Quick Item
	// or Pouch slot references, so the loadout is never invalidated.
	worldFlagInventoryScopes = []string{
		"save.session", "inventory", "world.flags", "diagnostics.report"}
	// Handing a Bell Bearing in consumes every matching record and searches
	// Inventory as well as Storage.
	worldFlagInventoryStorageScopes = []string{
		"save.session", "inventory", "storage", "world.flags", "diagnostics.report"}
)

// The domain members each result keeps beside the flat receipt.
var (
	bellBearingDomainKeys = []string{
		"characterID", "bellBearingKind", "bellBearingKey", "unlocked"}
	bossDomainKeys      = []string{"characterID", "bossKind", "bossKey", "defeated"}
	colosseumDomainKeys = []string{"characterID", "colosseumKind", "colosseumKey", "unlocked"}
	cookbookDomainKeys  = []string{"characterID", "cookbookKind", "cookbookKey", "unlocked"}
	fogOfWarDomainKeys  = []string{"characterID", "removed"}
	gestureDomainKeys   = []string{"characterID", "gestureKind", "gestureKey", "unlocked"}
	graceDomainKeys     = []string{"characterID", "graceKind", "graceKey", "visited"}
	mapRegionDomainKeys = []string{"characterID", "mapRegionKind", "mapRegionKey", "revealed"}
	questStepDomainKeys = []string{
		"characterID", "questKind", "questKey", "stepKind", "stepKey"}
	regionDomainKeys        = []string{"characterID", "regionKind", "regionKey", "unlocked"}
	spectralSteedDomainKeys = []string{"characterID", "attireKey"}
	summoningPoolDomainKeys = []string{
		"characterID", "summoningPoolKind", "summoningPoolKey", "activated"}
	tutorialDomainKeys  = []string{"characterID", "tutorialKind", "tutorialKey", "unlocked"}
	whetbladeDomainKeys = []string{"characterID", "whetbladeKind", "whetbladeKey", "unlocked"}
)

// worldReceiptCatalog is the stored GameCatalog every case below shares.
//
// ponytail: one sync.Once, not a change to the existing per-test constructors.
// newCookbooksCatalog rebuilds the whole catalog on every call and this file
// would call for it about fifty times. Nothing here patches a document, so one
// shared read-only catalog is enough.
func worldReceiptCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	worldReceiptCatalogOnce.Do(func() {
		worldReceiptCatalogValue = newCookbooksCatalog(t)
	})
	return worldReceiptCatalogValue
}

var (
	worldReceiptCatalogOnce  sync.Once
	worldReceiptCatalogValue *gamecatalog.Catalog
)

// wantWorldReceipt is the receipt one committed World mutation must carry. The
// opaque operationID cannot be predicted, so it is asserted non-empty here and
// then carried into the returned value, which lets a domain test keep comparing
// the complete result in one step instead of dropping the receipt from its
// assertion.
//
// The scopes come from the resolver rather than from a literal, because the
// literal scope contract of all fifteen World kinds is asserted in
// TestWorldEndpointIDsAreTheirRegisteredMutationKinds below and again in
// backend/saveengine/mutation_test.go. Restating it in every domain test would
// only duplicate those tables.
//
// ponytail: a want-builder, not an assertion helper. The caller already owns an
// exact comparison; it only lacks the one member it cannot know in advance.
func wantWorldReceipt(
	t *testing.T,
	got saveengine.MutationReceipt,
	operationKind string,
	saveSessionID string,
	saveRevision string,
) saveengine.MutationReceipt {
	t.Helper()

	if got.OperationID == "" {
		t.Errorf("receipt = %+v, want a minted operationID", got)
	}
	scopes, err := saveengine.ChangedScopesForMutationKind(operationKind)
	if err != nil {
		t.Fatalf("ChangedScopesForMutationKind(%q): %v", operationKind, err)
	}
	return saveengine.MutationReceipt{
		OperationID:   got.OperationID,
		OperationKind: operationKind,
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		ChangedScopes: scopes,
	}
}

// worldReceiptTarget is one prepared World session: the engine, the session
// identifier, a closure that performs the endpoint call under one expected
// revision, and the zero result of that endpoint. The fifteen endpoints have
// different signatures, so the closure is the smallest shape that lets one table
// drive all of them.
//
// ponytail: a func value, not a mutation-plan interface. The fifteen calls have
// nothing in common but "take a revision, return something JSON-encodable".
type worldReceiptTarget struct {
	engine    *saveengine.Engine
	sessionID string
	apply     func(expectedRevision string) (any, saveengine.MutationReceipt, error)
	zero      any
}

func bellBearingReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadBellBearingsSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetBellBearingUnlockedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetBellBearingUnlocked(engine, catalog, sessionID,
				getCookbooksSlot, "item", "400022CF", true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func bossReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadBossesSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetBossDefeatedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetBossDefeated(engine, catalog, sessionID,
				getCookbooksSlot, "boss", getBossesClearKey, true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func colosseumReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadColosseumsSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetColosseumUnlockedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetColosseumUnlocked(engine, catalog, sessionID,
				getCookbooksSlot, "colosseum", "limgrave_colosseum", true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func cookbookReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadCookbooksSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetCookbookUnlockedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetCookbookUnlocked(engine, catalog, sessionID,
				getCookbooksSlot, "item", getCookbooksSecondKey, true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func fogOfWarReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadRegionsSession(t, true)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetFogOfWarRemovedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetFogOfWarRemoved(
				engine, sessionID, getCookbooksSlot, true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func gestureReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	const slotID = uint32(229)
	engine, sessionID := loadGesturesSession(
		t, setGestureEndpointRecords(slotID-1, 4242, 0), true)
	catalog := newGesturesCatalog(t)
	key := gestureKeyForSlot(t, slotID)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetGestureUnlockedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetGestureUnlocked(engine, catalog, sessionID,
				getGesturesSlot, "item", key, true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func graceReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadGracesSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetGraceVisitedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetGraceVisited(engine, catalog, sessionID,
				getCookbooksSlot, "grace", getGracesClearKey, true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func mapRegionReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadMapRegionsSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetMapRegionRevealedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetMapRegionRevealed(engine, catalog, sessionID, getCookbooksSlot,
				"map_region", setMapRegionFragmentKey, true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func questStepReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadBossesSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetQuestStepResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetQuestStep(engine, catalog, sessionID, getCookbooksSlot,
				"quest", "brother_corhyn", "quest_step", "legacy_000", expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func regionReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadSetRegionSession(t, true, []uint32{6200000})
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetRegionUnlockedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetRegionUnlocked(engine, catalog, sessionID, 0,
				"region", setRegionValidKey, true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

// spectralSteedReceiptInventory is the ownership the two Spectral Steed targets
// share: two editor-created common records and one game-placed key record, so
// every appearance of the set is actually held.
func spectralSteedReceiptInventory() spectralSteedInventory {
	return spectralSteedInventory{
		commonHandles: []uint32{
			spectralSteedGoodsHandle(spectralSteedTreeGameID),
			spectralSteedGoodsHandle(spectralSteedSilverGameID),
		},
		keyGameIDs: []uint32{spectralSteedFuneralID},
	}
}

func spectralSteedAttireReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadSpectralSteedSession(
		t, nil, spectralSteedReceiptInventory(), true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetSpectralSteedAttireResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetSpectralSteedAttire(engine, catalog, sessionID,
				getCookbooksSlot, SpectralSteedAttireKeyTreeSentinel, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func lockAllSpectralSteedReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadSpectralSteedSession(
		t, []uint32{6702}, spectralSteedReceiptInventory(), true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: LockAllSpectralSteedAttiresResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := LockAllSpectralSteedAttires(
				engine, catalog, sessionID, getCookbooksSlot, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func summoningPoolReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadSummoningPoolsSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetSummoningPoolActivatedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetSummoningPoolActivated(engine, catalog, sessionID,
				getCookbooksSlot, "summoning_pool", getSummoningPoolsClearKey,
				true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func tutorialReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadTutorialsSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetTutorialUnlockedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetTutorialUnlocked(engine, catalog, sessionID, getCookbooksSlot,
				string(schema.ResourceKindTutorial), getTutorialsLockedKey,
				true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func whetbladeReceiptTarget(t *testing.T) worldReceiptTarget {
	t.Helper()

	engine, sessionID := loadWhetbladesSession(t, true)
	catalog := worldReceiptCatalog(t)
	return worldReceiptTarget{
		engine: engine, sessionID: sessionID, zero: SetWhetbladeUnlockedResult{},
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetWhetbladeUnlocked(engine, catalog, sessionID,
				getCookbooksSlot, "item", "4000230C", true, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

// worldReceiptCases drives the receipt, scope, flat-JSON, committed no-change
// and rejection invariants for all fifteen endpoints from one table, so a new
// World mutation cannot be added with a narrower guarantee.
var worldReceiptCases = []struct {
	name          string
	operationKind string
	scopes        []string
	domainKeys    []string
	target        func(*testing.T) worldReceiptTarget
}{
	{"SetBellBearingUnlocked", SetBellBearingUnlockedEndpointID,
		worldFlagInventoryStorageScopes, bellBearingDomainKeys, bellBearingReceiptTarget},
	{"SetBossDefeated", SetBossDefeatedEndpointID,
		worldFlagScopes, bossDomainKeys, bossReceiptTarget},
	{"SetColosseumUnlocked", SetColosseumUnlockedEndpointID,
		worldFlagScopes, colosseumDomainKeys, colosseumReceiptTarget},
	{"SetCookbookUnlocked", SetCookbookUnlockedEndpointID,
		worldFlagScopes, cookbookDomainKeys, cookbookReceiptTarget},
	{"SetFogOfWarRemoved", SetFogOfWarRemovedEndpointID,
		worldFlagScopes, fogOfWarDomainKeys, fogOfWarReceiptTarget},
	{"SetGestureUnlocked", SetGestureUnlockedEndpointID,
		worldFlagScopes, gestureDomainKeys, gestureReceiptTarget},
	{"SetGraceVisited", SetGraceVisitedEndpointID,
		worldFlagScopes, graceDomainKeys, graceReceiptTarget},
	{"SetMapRegionRevealed", SetMapRegionRevealedEndpointID,
		worldFlagInventoryScopes, mapRegionDomainKeys, mapRegionReceiptTarget},
	{"SetQuestStep", SetQuestStepEndpointID,
		worldFlagScopes, questStepDomainKeys, questStepReceiptTarget},
	{"SetRegionUnlocked", SetRegionUnlockedEndpointID,
		worldFlagScopes, regionDomainKeys, regionReceiptTarget},
	{"SetSpectralSteedAttire", SetSpectralSteedAttireEndpointID,
		worldFlagScopes, spectralSteedDomainKeys, spectralSteedAttireReceiptTarget},
	{"LockAllSpectralSteedAttires", LockAllSpectralSteedAttiresEndpointID,
		worldFlagInventoryScopes, spectralSteedDomainKeys, lockAllSpectralSteedReceiptTarget},
	{"SetSummoningPoolActivated", SetSummoningPoolActivatedEndpointID,
		worldFlagScopes, summoningPoolDomainKeys, summoningPoolReceiptTarget},
	{"SetTutorialUnlocked", SetTutorialUnlockedEndpointID,
		worldFlagScopes, tutorialDomainKeys, tutorialReceiptTarget},
	{"SetWhetbladeUnlocked", SetWhetbladeUnlockedEndpointID,
		worldFlagInventoryScopes, whetbladeDomainKeys, whetbladeReceiptTarget},
}

func TestEveryWorldMutationCarriesItsCommitReceipt(t *testing.T) {
	for _, testCase := range worldReceiptCases {
		t.Run(testCase.name, func(t *testing.T) {
			target := testCase.target(t)
			result, receipt, err := target.apply("0")
			if err != nil {
				t.Fatalf("%s: %v", testCase.name, err)
			}
			assertWorldReceipt(t, receipt, target.sessionID, testCase.operationKind, "1")
			assertWorldChangedScopes(t, receipt.ChangedScopes, testCase.scopes)
			assertFlatWorldReceiptJSON(t, result, testCase.domainKeys)
		})
	}
}

// Every World setter may finish its callback without writing a byte, and the
// central commit path still advances the revision, marks the session dirty and
// issues a receipt. That committed no-change is a commit, so the second
// identical assignment must report a fresh operationID, the same kind, the next
// revision, the same scopes and unchanged domain fields.
func TestEveryWorldMutationCommitsAnIdenticalReassignment(t *testing.T) {
	for _, testCase := range worldReceiptCases {
		t.Run(testCase.name, func(t *testing.T) {
			target := testCase.target(t)
			first, firstReceipt, err := target.apply("0")
			if err != nil {
				t.Fatalf("first %s: %v", testCase.name, err)
			}

			second, secondReceipt, err := target.apply("1")
			if err != nil {
				t.Fatalf("identical second %s: %v", testCase.name, err)
			}
			assertWorldReceipt(t, secondReceipt, target.sessionID, testCase.operationKind, "2")
			assertWorldChangedScopes(t, secondReceipt.ChangedScopes, testCase.scopes)
			assertFlatWorldReceiptJSON(t, second, testCase.domainKeys)
			if secondReceipt.OperationID == firstReceipt.OperationID {
				t.Errorf("two executions shared operationID %q", secondReceipt.OperationID)
			}
			if !reflect.DeepEqual(
				worldDomainMembers(t, second, testCase.domainKeys),
				worldDomainMembers(t, first, testCase.domainKeys)) {
				t.Errorf("committed no-change changed the domain fields: %+v, want %+v",
					second, first)
			}

			// The revision really moved and the session really is dirty.
			info, err := target.engine.GetSessionInfo(target.sessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if info.SaveRevision != "2" || !info.UnsavedChanges {
				t.Errorf("session = %+v, want revision 2 with unsaved changes", info)
			}
		})
	}
}

// A rejected World mutation returns the complete zero result — no receipt member
// and no domain member survives — and leaves the session exactly as it was. A
// stale expectedRevision is the one rejection every one of the fifteen shares,
// so it is the general invariant this table proves.
func TestRejectedWorldMutationsReturnNoReceiptAndNoDomainData(t *testing.T) {
	for _, testCase := range worldReceiptCases {
		t.Run(testCase.name+"/stale expectedRevision", func(t *testing.T) {
			target := testCase.target(t)
			before, err := target.engine.GetSessionInfo(target.sessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}

			result, _, err := target.apply("7")
			if err == nil {
				t.Fatalf("the stale revision was accepted: %+v", result)
			}
			if !reflect.DeepEqual(result, target.zero) {
				t.Errorf("result = %+v, want the complete zero result %+v", result, target.zero)
			}

			after, err := target.engine.GetSessionInfo(target.sessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo after the rejection: %v", err)
			}
			if after != before {
				t.Errorf("session = %+v, want the unchanged %+v", after, before)
			}
		})
		t.Run(testCase.name+"/non-canonical expectedRevision", func(t *testing.T) {
			target := testCase.target(t)
			result, _, err := target.apply("00")
			if err == nil {
				t.Fatalf("the non-canonical revision was accepted: %+v", result)
			}
			if !reflect.DeepEqual(result, target.zero) {
				t.Errorf("result = %+v, want the complete zero result %+v", result, target.zero)
			}
		})
	}
}

// The two Spectral Steed entry points share SpectralSteedAttireMutation as their
// SaveEngine result type, yet each keeps its own operationKind, and their receipts
// also differ in the exact changedScopes they resolve: selecting an appearance
// touches the world flags only, locking every appearance touches Inventory too.
// Their public endpoint results keep their own attireKey as well — the selected
// appearance against the default one. Neither may ever report the other's kind,
// the other's scopes or the other's attireKey.
func TestTheTwoSpectralSteedMutationsKeepSeparateKinds(t *testing.T) {
	selection, selectionReceipt, err := spectralSteedAttireReceiptTarget(t).apply("0")
	if err != nil {
		t.Fatalf("SetSpectralSteedAttire: %v", err)
	}
	reset, resetReceipt, err := lockAllSpectralSteedReceiptTarget(t).apply("0")
	if err != nil {
		t.Fatalf("LockAllSpectralSteedAttires: %v", err)
	}

	if selectionReceipt.OperationKind != SetSpectralSteedAttireEndpointID {
		t.Errorf("selection operationKind = %q, want %q",
			selectionReceipt.OperationKind, SetSpectralSteedAttireEndpointID)
	}
	if resetReceipt.OperationKind != LockAllSpectralSteedAttiresEndpointID {
		t.Errorf("reset operationKind = %q, want %q",
			resetReceipt.OperationKind, LockAllSpectralSteedAttiresEndpointID)
	}
	if selectionReceipt.OperationID == resetReceipt.OperationID {
		t.Errorf("the two executions shared operationID %q", selectionReceipt.OperationID)
	}

	// Selecting an appearance reads the Inventory to prove the item is held and
	// writes appearance flags only. Locking every appearance removes the records,
	// so only the second one invalidates Inventory.
	assertWorldChangedScopes(t, selectionReceipt.ChangedScopes, worldFlagScopes)
	assertWorldChangedScopes(t, resetReceipt.ChangedScopes, worldFlagInventoryScopes)

	// The domain payload keeps telling the two apart as well.
	if selection.(SetSpectralSteedAttireResult).AttireKey !=
		SpectralSteedAttireKeyTreeSentinel {
		t.Errorf("selection = %+v, want the Tree Sentinel appearance", selection)
	}
	if reset.(LockAllSpectralSteedAttiresResult).AttireKey !=
		SpectralSteedAttireKeyDefault {
		t.Errorf("reset = %+v, want the default appearance", reset)
	}
}

// Every World endpoint reports its own EndpointID as operationKind, each of the
// fifteen is a registered SaveEngine mutation kind, and each resolves exactly
// the scopes stated literally in the table above.
func TestWorldEndpointIDsAreTheirRegisteredMutationKinds(t *testing.T) {
	registered := make(map[string]bool)
	for _, kind := range saveengine.MutationKinds() {
		registered[kind] = true
	}
	for _, testCase := range worldReceiptCases {
		t.Run(testCase.operationKind, func(t *testing.T) {
			if !registered[testCase.operationKind] {
				t.Fatalf("SaveEngine registers no operation kind %q", testCase.operationKind)
			}
			scopes, err := saveengine.ChangedScopesForMutationKind(testCase.operationKind)
			if err != nil {
				t.Fatalf("ChangedScopesForMutationKind(%q): %v", testCase.operationKind, err)
			}
			assertWorldChangedScopes(t, scopes, testCase.scopes)
		})
	}
	if len(worldReceiptCases) != 15 {
		t.Errorf("the table drives %d World mutations, want all 15", len(worldReceiptCases))
	}
}

// worldDomainMembers returns the endpoint's own JSON members, receipt excluded,
// so a committed no-change can be compared without the operationID that must
// differ.
func worldDomainMembers(t *testing.T, result any, domainKeys []string) map[string]string {
	t.Helper()

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal %T: %v", result, err)
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &members); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}
	domain := make(map[string]string, len(domainKeys))
	for _, key := range domainKeys {
		raw, present := members[key]
		if !present {
			t.Fatalf("%T JSON carries no domain member %q: %s", result, key, encoded)
		}
		domain[key] = string(raw)
	}
	return domain
}

// assertWorldReceipt checks the four scalar receipt fields of one committed
// mutation. The scopes are checked separately, because their exact value is a
// per-endpoint contract.
func assertWorldReceipt(
	t *testing.T,
	receipt saveengine.MutationReceipt,
	saveSessionID string,
	operationKind string,
	saveRevision string,
) {
	t.Helper()

	if receipt.OperationID == "" {
		t.Errorf("receipt = %+v, want a minted operationID", receipt)
	}
	if receipt.OperationKind != operationKind {
		t.Errorf("operationKind = %q, want the EndpointID %q", receipt.OperationKind, operationKind)
	}
	if receipt.SaveSessionID != saveSessionID {
		t.Errorf("saveSessionID = %q, want %q", receipt.SaveSessionID, saveSessionID)
	}
	if receipt.SaveRevision != saveRevision {
		t.Errorf("saveRevision = %q, want %q", receipt.SaveRevision, saveRevision)
	}
}

func assertWorldChangedScopes(t *testing.T, got []string, want []string) {
	t.Helper()

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("changedScopes = %v, want exactly %v in canonical order", got, want)
	}
}

// assertFlatWorldReceiptJSON proves the embedding is flat: the five receipt
// fields are top-level keys of the payload, each key appears exactly once, there
// is no nested "receipt" object, and the domain fields of the endpoint survive.
func assertFlatWorldReceiptJSON(t *testing.T, result any, domainKeys []string) {
	t.Helper()

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal %T: %v", result, err)
	}
	keys := worldJSONTopLevelKeys(t, encoded)

	counts := make(map[string]int, len(keys))
	for _, key := range keys {
		counts[key]++
	}
	want := append([]string{
		"operationID", "operationKind", "saveSessionID", "saveRevision", "changedScopes",
	}, domainKeys...)
	for _, key := range want {
		if counts[key] != 1 {
			t.Errorf("%T JSON carries key %q %d times, want exactly once: %s",
				result, key, counts[key], encoded)
		}
	}
	if len(keys) != len(want) {
		t.Errorf("%T JSON keys = %v, want exactly %v", result, keys, want)
	}
	if counts["receipt"] != 0 {
		t.Errorf("%T JSON nests the receipt instead of flattening it: %s", result, encoded)
	}
}

// worldJSONTopLevelKeys returns the member names of a JSON object in document
// order, repeats included. A map would silently collapse a duplicated key, which
// is exactly the defect this check exists to catch.
func worldJSONTopLevelKeys(t *testing.T, encoded []byte) []string {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		t.Fatalf("payload is not a JSON object: %s (%v)", encoded, err)
	}
	var keys []string
	for decoder.More() {
		name, err := decoder.Token()
		if err != nil {
			t.Fatalf("read member name of %s: %v", encoded, err)
		}
		key, isString := name.(string)
		if !isString {
			t.Fatalf("member name %v of %s is not a string", name, encoded)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("read member %q of %s: %v", key, encoded, err)
		}
		keys = append(keys, key)
	}
	return keys
}
