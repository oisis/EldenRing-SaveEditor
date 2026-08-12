package inventory

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// The two containers of this package already own a synthetic fixture each, and
// both carry catalog-resolvable records. GetOwnedItem is verified against those
// same fixtures, so it proves the resolution of a token this package itself
// issued instead of inventing a third container layout.

// Every record of one container read, expressed as the result GetOwnedItem has
// to return for its own token.
func wantOwnedInventory(saveSessionID string, record InventoryRecord) GetOwnedItemResult {
	return GetOwnedItemResult{
		SaveSessionID:    saveSessionID,
		SaveRevision:     "0",
		OwnedItemID:      record.OwnedItemID,
		CharacterID:      getInventorySlot,
		Kind:             record.Kind,
		Key:              record.Key,
		GameID:           record.GameID,
		Container:        "inventory",
		ContainerSection: record.ContainerSection,
		PhysicalIndex:    record.PhysicalIndex,
		GaItemHandle:     record.GaItemHandle,
		Quantity:         record.Quantity,
		AcquisitionIndex: record.AcquisitionIndex,
	}
}

func wantOwnedStorage(saveSessionID string, record StorageRecord) GetOwnedItemResult {
	return GetOwnedItemResult{
		SaveSessionID:    saveSessionID,
		SaveRevision:     "0",
		OwnedItemID:      record.OwnedItemID,
		CharacterID:      getStorageSlot,
		Kind:             record.Kind,
		Key:              record.Key,
		GameID:           record.GameID,
		Container:        "storage",
		ContainerSection: record.ContainerSection,
		PhysicalIndex:    record.PhysicalIndex,
		GaItemHandle:     record.GaItemHandle,
		Quantity:         record.Quantity,
		AcquisitionIndex: record.AcquisitionIndex,
	}
}

func TestGetOwnedItemResolvesInventoryTokensOnBothPlatforms(t *testing.T) {
	for _, platform := range []string{"pc", "ps4"} {
		t.Run(platform, func(t *testing.T) {
			engine, sessionID := loadGetInventorySession(t, platform, true, getInventoryAnchorAt)
			gameCatalog := inventoryCatalog(t)

			listed, err := GetInventory(engine, gameCatalog, sessionID, getInventorySlot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}
			if len(listed.Records) != 3 {
				t.Fatalf("fixture listed %d records, want 3", len(listed.Records))
			}

			for _, record := range listed.Records {
				row := record.ContainerSection + "#" + strconv.Itoa(record.PhysicalIndex)
				t.Run(row, func(t *testing.T) {
					result, err := GetOwnedItem(
						engine, gameCatalog, sessionID, getInventorySlot, record.OwnedItemID)
					if err != nil {
						t.Fatalf("GetOwnedItem: %v", err)
					}
					want := wantOwnedInventory(sessionID, record)
					if !reflect.DeepEqual(result, want) {
						t.Errorf("result = %+v, want %+v", result, want)
					}
				})
			}
		})
	}
}

func TestGetOwnedItemResolvesStorageTokensOnBothPlatforms(t *testing.T) {
	for _, platform := range []string{"pc", "ps4"} {
		t.Run(platform, func(t *testing.T) {
			engine, sessionID := loadGetStorageSession(t, platform, true, getStorageAnchorAt)
			gameCatalog := inventoryCatalog(t)

			listed, err := GetStorage(engine, gameCatalog, sessionID, getStorageSlot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetStorage: %v", err)
			}
			if len(listed.Records) != 3 {
				t.Fatalf("fixture listed %d records, want 3", len(listed.Records))
			}

			for _, record := range listed.Records {
				row := record.ContainerSection + "#" + strconv.Itoa(record.PhysicalIndex)
				t.Run(row, func(t *testing.T) {
					result, err := GetOwnedItem(
						engine, gameCatalog, sessionID, getStorageSlot, record.OwnedItemID)
					if err != nil {
						t.Fatalf("GetOwnedItem: %v", err)
					}
					want := wantOwnedStorage(sessionID, record)
					if !reflect.DeepEqual(result, want) {
						t.Errorf("result = %+v, want %+v", result, want)
					}
				})
			}
		})
	}
}

// A residual slot mints no identity, so nothing addresses its rows: a token of
// another session stays unknown instead of resolving into the surviving data.
func TestGetOwnedItemFindsNothingInAResidualSlot(t *testing.T) {
	gameCatalog := inventoryCatalog(t)
	activeEngine, activeID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	listed, err := GetInventory(activeEngine, gameCatalog, activeID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	foreign := listed.Records[0].OwnedItemID

	residualEngine, residualID := loadGetInventorySession(t, "pc", false, getInventoryAnchorAt)
	residual, err := GetInventory(residualEngine, gameCatalog, residualID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory of the residual slot: %v", err)
	}
	if residual.Active || len(residual.Records) != 0 {
		t.Fatalf("residual slot = %+v, want an inactive, empty result", residual)
	}

	result, err := GetOwnedItem(residualEngine, gameCatalog, residualID, getInventorySlot, foreign)
	if err == nil {
		t.Fatalf("GetOwnedItem resolved a foreign token in a residual slot: %+v", result)
	}
	if err.Error() != "unknown ownedItemID" {
		t.Errorf("error = %q, want the unknown-token error", err)
	}
	if !reflect.DeepEqual(result, GetOwnedItemResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestGetOwnedItemRejectsInvalidRequests(t *testing.T) {
	gameCatalog := inventoryCatalog(t)
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	listed, err := GetInventory(engine, gameCatalog, sessionID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	valid := listed.Records[0].OwnedItemID

	closedEngine, closedID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	if err := closedEngine.CloseSession(closedID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	cases := map[string]struct {
		engine        *saveengine.Engine
		saveSessionID string
		characterID   int
		ownedItemID   string
		want          string
	}{
		"nil engine":      {nil, sessionID, getInventorySlot, valid, "save engine is not available"},
		"empty session":   {engine, "", getInventorySlot, valid, "saveSessionID is required"},
		"unknown session": {engine, "missing", getInventorySlot, valid, `unknown save session "missing"`},
		"closed session": {closedEngine, closedID, getInventorySlot, valid,
			`unknown save session ` + strconv.Quote(closedID)},
		"characterID -1": {engine, sessionID, -1, valid, "characterID -1 is outside the range 0..9"},
		"characterID 10": {engine, sessionID, 10, valid, "characterID 10 is outside the range 0..9"},
		"empty ownedItemID": {engine, sessionID, getInventorySlot, "",
			"ownedItemID is required"},
		"unknown ownedItemID": {engine, sessionID, getInventorySlot, "whatever",
			"unknown ownedItemID"},
		// The token is never parsed, trimmed or reconstructed, so a padded or
		// extended form of a valid token is a different, unknown string.
		"padded ownedItemID":   {engine, sessionID, getInventorySlot, " " + valid, "unknown ownedItemID"},
		"extended ownedItemID": {engine, sessionID, getInventorySlot, valid + "0", "unknown ownedItemID"},
		"ownedItemID of another character": {engine, sessionID, 4, valid,
			"ownedItemID belongs to character 3, not to character 4"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := GetOwnedItem(
				testCase.engine, gameCatalog, testCase.saveSessionID, testCase.characterID, testCase.ownedItemID)
			if err == nil {
				t.Fatalf("GetOwnedItem accepted %s: %+v", name, result)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, GetOwnedItemResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}

// Unknown data fails the whole request: no placeholder document, no partial
// identity and no substitute item.
func TestGetOwnedItemRejectsANilCatalogAndAnUnknownCatalogItem(t *testing.T) {
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
	listed, err := GetInventory(engine, inventoryCatalog(t), sessionID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	valid := listed.Records[0].OwnedItemID

	result, err := GetOwnedItem(engine, nil, sessionID, getInventorySlot, valid)
	if err == nil || err.Error() != "game catalog is not available" {
		t.Errorf("nil catalog error = %v, want game catalog is not available", err)
	}
	if !reflect.DeepEqual(result, GetOwnedItemResult{}) {
		t.Errorf("nil catalog result = %+v, want the zero value", result)
	}

	prototype, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("gamecatalog.NewPrototype: %v", err)
	}
	result, err = GetOwnedItem(engine, prototype, sessionID, getInventorySlot, valid)
	if err == nil {
		t.Fatalf("GetOwnedItem accepted an item absent from the prototype catalog: %+v", result)
	}
	if !reflect.DeepEqual(result, GetOwnedItemResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}

func TestGetOwnedItemContractResolvesItemDocuments(t *testing.T) {
	if GetOwnedItemDefinition.SupportedResourceTypes != "ItemDocument" {
		t.Errorf("supported resource types = %q, want ItemDocument",
			GetOwnedItemDefinition.SupportedResourceTypes)
	}
	want := []string{"saveSessionID", "characterID", "ownedItemID"}
	if !reflect.DeepEqual(GetOwnedItemDefinition.SupportedResourceVariables, want) {
		t.Errorf("supported resource variables = %v, want %v",
			GetOwnedItemDefinition.SupportedResourceVariables, want)
	}
}
