package world

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	// Weeping Peninsula and Limgrave, West both own a Map Fragment; Murkwater
	// Catacombs is a dungeon map of the same curated table and owns none.
	setMapRegionFragmentKey     = "limgrave_weeping_peninsula"
	setMapRegionFragmentGameID  = uint32(0x40002199)
	setMapRegionRevealedWestKey = "limgrave_limgrave_west"
	setMapRegionWestGameID      = uint32(0x40002198)
	setMapRegionNoFragmentKey   = "limgrave_murkwater_catacombs"
)

func TestSetMapRegionRevealedCommitsFlagAndFragment(t *testing.T) {
	// The fixture leaves 62010 set and every other region clear, so revealing one
	// region and hiding another proves the endpoint resolves the exact key.
	engine, sessionID := loadMapRegionsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	result, err := SetMapRegionRevealed(engine, gameCatalog, sessionID, getCookbooksSlot,
		"map_region", setMapRegionFragmentKey, true, "0")
	if err != nil {
		t.Fatalf("SetMapRegionRevealed: %v", err)
	}
	want := SetMapRegionRevealedResult{
		MutationReceipt: wantWorldReceipt(
			t, result.MutationReceipt, SetMapRegionRevealedEndpointID, sessionID, "1"),
		CharacterID:   getCookbooksSlot,
		MapRegionKind: schema.ResourceKindMapRegion,
		MapRegionKey:  setMapRegionFragmentKey,
		Revealed:      true,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	if _, err := SetMapRegionRevealed(engine, gameCatalog, sessionID, getCookbooksSlot,
		"map_region", setMapRegionRevealedWestKey, false, "1"); err != nil {
		t.Fatalf("SetMapRegionRevealed hide: %v", err)
	}
	// A region without a fragment must commit on the flag alone.
	if _, err := SetMapRegionRevealed(engine, gameCatalog, sessionID, getCookbooksSlot,
		"map_region", setMapRegionNoFragmentKey, true, "2"); err != nil {
		t.Fatalf("SetMapRegionRevealed dungeon: %v", err)
	}

	state, err := GetMapRegions(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetMapRegions: %v", err)
	}
	visible := make(map[string]bool, len(state.MapRegions))
	for _, entry := range state.MapRegions {
		visible[entry.Key] = entry.Visible
	}
	for key, expected := range map[string]bool{
		setMapRegionFragmentKey:     true,
		setMapRegionRevealedWestKey: false,
		setMapRegionNoFragmentKey:   true,
	} {
		if visible[key] != expected {
			t.Errorf("map region %q visible = %t, want %t", key, visible[key], expected)
		}
	}

	present, err := engine.GetInventoryGoodsPresence(sessionID, getCookbooksSlot,
		[]uint32{setMapRegionFragmentGameID, setMapRegionWestGameID})
	if err != nil {
		t.Fatalf("GetInventoryGoodsPresence: %v", err)
	}
	if !present.Presence[setMapRegionFragmentGameID] {
		t.Error("revealing a region did not add its Map Fragment")
	}
	if present.Presence[setMapRegionWestGameID] {
		t.Error("hiding a region left its Map Fragment in the inventory")
	}
}

func TestSetMapRegionRevealedRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadMapRegionsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := SetMapRegionRevealed(nil, gameCatalog, sessionID, getCookbooksSlot,
		"map_region", setMapRegionFragmentKey, true, "0"); err == nil ||
		err.Error() != "save engine is not available" {
		t.Errorf("nil SaveEngine error = %v", err)
	}
	if _, err := SetMapRegionRevealed(engine, nil, sessionID, getCookbooksSlot,
		"map_region", setMapRegionFragmentKey, true, "0"); err == nil ||
		err.Error() != "game catalog is not available" {
		t.Errorf("nil GameCatalog error = %v", err)
	}

	for name, testCase := range map[string]struct {
		kind     string
		key      string
		revision string
		want     string
	}{
		// A valid map region key under another kind must fail, which proves the
		// kind is checked before the key is looked up.
		"wrong kind": {
			"item", setMapRegionFragmentKey, "0",
			`resource kind "item" is not "map_region"`,
		},
		"unknown key": {
			"map_region", "not_a_map_region", "0",
			`unknown resource key "not_a_map_region" in kind "map_region"`,
		},
		"stale revision": {
			"map_region", setMapRegionFragmentKey, "9",
			`does not match the current saveRevision`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := SetMapRegionRevealed(engine, gameCatalog, sessionID,
				getCookbooksSlot, testCase.kind, testCase.key, true, testCase.revision)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want one containing %q", err, testCase.want)
			}
		})
	}

	// An inconsistent catalog — a second map region claiming the flag of the
	// requested one — must be refused before a save byte is touched.
	resources := storedCookbookResources(t)
	patched := 0
	for index := range resources {
		if resources[index].Kind != schema.ResourceKindMapRegion ||
			resources[index].Key != setMapRegionNoFragmentKey {
			continue
		}
		document := *resources[index].MapRegion
		document.VisibleEventFlagID.Value = 62011
		resources[index].MapRegion = &document
		patched++
	}
	if patched != 1 {
		t.Fatalf("patched %d map regions, want 1", patched)
	}
	if _, err := SetMapRegionRevealed(engine, cookbooksCatalogOf(t, resources), sessionID,
		getCookbooksSlot, "map_region", setMapRegionFragmentKey, true, "0"); err == nil ||
		!strings.Contains(err.Error(), "both declare event flag 62011") {
		t.Fatalf("duplicate flag error = %v", err)
	}

	assertMapRegionSessionUnchanged(t, engine, sessionID)
}

func assertMapRegionSessionUnchanged(t *testing.T, engine *saveengine.Engine, sessionID string) {
	t.Helper()
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejections = %+v, want clean", info)
	}
}
