package world

import (
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	getMapRegionsCount    = 263
	getMapRegionsSetFlag  = 62010
	getMapRegionsSetKey   = "limgrave_limgrave_west"
	getMapRegionsClearKey = "limgrave_weeping_peninsula"

	// Block 62 occupies this BST position in the confirmed bitfield layout.
	getMapRegionsBlockPosition = 12
)

func writeGetMapRegionsFixture(t *testing.T, active bool) string {
	t.Helper()

	path := writeGetCookbooksFixture(t, nil, active)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchorBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize + getCookbooksAnchorAt
	index := int64(getMapRegionsSetFlag % 1000)
	offset := getMapRegionsBlockPosition*getCookbooksBlockSize + index/8
	data[anchorBase+getCookbooksSectionAt+offset] |= 1 << uint8(7-index%8)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadMapRegionsSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetMapRegionsFixture(t, active), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestGetMapRegionsReturnsTheCuratedVisibilityState(t *testing.T) {
	engine, sessionID := loadMapRegionsSession(t, true)
	result, err := GetMapRegions(
		engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetMapRegions: %v", err)
	}
	if !result.Active || result.SaveSessionID != sessionID ||
		result.CharacterID != getCookbooksSlot {
		t.Fatalf("result identity = %+v, want the active requested slot", result)
	}
	if len(result.MapRegions) != getMapRegionsCount {
		t.Fatalf("map region count = %d, want %d",
			len(result.MapRegions), getMapRegionsCount)
	}

	want := map[string]bool{getMapRegionsSetKey: true, getMapRegionsClearKey: false}
	found := make(map[string]bool, len(want))
	for index, entry := range result.MapRegions {
		if entry.Kind != schema.ResourceKindMapRegion {
			t.Fatalf("entry %q has kind %q", entry.Key, entry.Kind)
		}
		if entry.Name == "" || entry.AreaLabel == "" {
			t.Fatalf("entry %q = %+v, want a name and area label", entry.Key, entry)
		}
		if strings.Contains(entry.Key, "62010") {
			t.Fatalf("entry %q leaks a raw event flag", entry.Key)
		}
		if index > 0 {
			previous := result.MapRegions[index-1]
			if previous.AreaLabel > entry.AreaLabel ||
				(previous.AreaLabel == entry.AreaLabel && previous.Name > entry.Name) ||
				(previous.AreaLabel == entry.AreaLabel && previous.Name == entry.Name &&
					previous.Key > entry.Key) {
				t.Fatalf("map regions are not ordered at %q then %q", previous.Key, entry.Key)
			}
		}
		if visible, exists := want[entry.Key]; exists {
			found[entry.Key] = true
			if entry.Visible != visible {
				t.Errorf("map region %q visible = %t, want %t", entry.Key, entry.Visible, visible)
			}
		}
	}
	if len(found) != len(want) {
		t.Fatalf("found map region keys = %v, want all %v", found, want)
	}
}

func TestGetMapRegionsDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadMapRegionsSession(t, false)
	result, err := GetMapRegions(
		engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetMapRegions: %v", err)
	}
	if result.Active || len(result.MapRegions) != getMapRegionsCount {
		t.Fatalf("active/count = %t/%d, want false/%d",
			result.Active, len(result.MapRegions), getMapRegionsCount)
	}
	for _, entry := range result.MapRegions {
		if entry.Visible {
			t.Fatalf("residual slot reports %q visible", entry.Key)
		}
	}
}

func TestGetMapRegionsRejectsInvalidInputAndDuplicateFlag(t *testing.T) {
	engine, sessionID := loadMapRegionsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	if _, err := GetMapRegions(nil, gameCatalog, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil SaveEngine was accepted")
	}
	if _, err := GetMapRegions(engine, nil, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil GameCatalog was accepted")
	}
	if _, err := GetMapRegions(engine, gameCatalog, "missing", getCookbooksSlot); err == nil {
		t.Error("unknown session was accepted")
	}

	resources := storedCookbookResources(t)
	patched := 0
	for index := range resources {
		if resources[index].Kind != schema.ResourceKindMapRegion ||
			resources[index].Key != getMapRegionsClearKey {
			continue
		}
		document := *resources[index].MapRegion
		document.VisibleEventFlagID.Value = getMapRegionsSetFlag
		resources[index].MapRegion = &document
		patched++
	}
	if patched != 1 {
		t.Fatalf("patched %d map regions, want 1", patched)
	}
	_, err := GetMapRegions(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot)
	if err == nil || !strings.Contains(err.Error(), "both declare event flag") {
		t.Fatalf("duplicate visibility flag error = %v", err)
	}
}
