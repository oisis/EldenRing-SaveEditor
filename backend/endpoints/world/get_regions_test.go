package world

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	getRegionsFirstID  = uint32(6100000)
	getRegionsSecondID = uint32(6200000)
	getRegionsUnknown  = uint32(9999999)
	getRegionsCount    = 274
)

func writeGetRegionsFixture(t *testing.T, active bool) string {
	t.Helper()
	path := writeGetCookbooksFixture(t, nil, active)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	slotBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize
	countAt := slotBase + getCookbooksAnchorAt + getCookbooksProjectileCountAt + 4 +
		getCookbooksProjectiles*8 + getCookbooksBlocksBeforeStorage +
		getCookbooksStorageBoxSize + getCookbooksGestureSectionSize
	ids := []uint32{getRegionsFirstID, getRegionsUnknown, getRegionsSecondID}
	binary.LittleEndian.PutUint32(data[countAt:], uint32(len(ids)))
	for index, id := range ids {
		binary.LittleEndian.PutUint32(data[countAt+4+int64(index)*4:], id)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadRegionsSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetRegionsFixture(t, active), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func patchRegionDocument(
	t *testing.T, resources []schema.Resource, key string, change func(*schema.RegionDocument),
) []schema.Resource {
	t.Helper()
	patched := 0
	for index := range resources {
		if resources[index].Kind != schema.ResourceKindRegion || resources[index].Key != key {
			continue
		}
		document := *resources[index].Region
		change(&document)
		resources[index].Region = &document
		patched++
	}
	if patched != 1 {
		t.Fatalf("patched %d regions of %q, want 1", patched, key)
	}
	return resources
}

func TestGetRegionsReturnsCuratedCatalogState(t *testing.T) {
	engine, sessionID := loadRegionsSession(t, true)
	result, err := GetRegions(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetRegions: %v", err)
	}
	if !result.Active || result.SaveSessionID != sessionID || result.CharacterID != getCookbooksSlot {
		t.Fatalf("result identity = %+v, want the active requested slot", result)
	}
	if len(result.Regions) != getRegionsCount {
		t.Fatalf("region count = %d, want %d", len(result.Regions), getRegionsCount)
	}
	want := map[string]bool{
		"limgrave_the_first_step":       true,
		"liurnia_lake_facing_cliffs":    true,
		"liurnia_liurnia_highway_south": false,
	}
	found := make(map[string]bool, len(want))
	for index, entry := range result.Regions {
		if index > 0 {
			previous := result.Regions[index-1]
			if previous.Area > entry.Area ||
				(previous.Area == entry.Area && previous.Name > entry.Name) ||
				(previous.Area == entry.Area && previous.Name == entry.Name && previous.Key > entry.Key) {
				t.Fatalf("regions are not ordered at %q then %q", previous.Key, entry.Key)
			}
		}
		if unlocked, exists := want[entry.Key]; exists {
			found[entry.Key] = true
			if entry.Unlocked != unlocked {
				t.Errorf("region %q unlocked = %t, want %t", entry.Key, entry.Unlocked, unlocked)
			}
		}
	}
	if len(found) != len(want) {
		t.Fatalf("found region keys = %v, want all %v", found, want)
	}
}

func TestGetRegionsDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadRegionsSession(t, false)
	result, err := GetRegions(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetRegions: %v", err)
	}
	if result.Active || len(result.Regions) != getRegionsCount {
		t.Fatalf("active/count = %t/%d, want false/%d",
			result.Active, len(result.Regions), getRegionsCount)
	}
	for _, entry := range result.Regions {
		if entry.Unlocked {
			t.Fatalf("residual slot reports %q unlocked", entry.Key)
		}
	}
}

func TestGetRegionsRejectsInvalidInputAndDuplicateID(t *testing.T) {
	engine, sessionID := loadRegionsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	if _, err := GetRegions(nil, gameCatalog, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil SaveEngine was accepted")
	}
	if _, err := GetRegions(engine, nil, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil GameCatalog was accepted")
	}
	if _, err := GetRegions(engine, gameCatalog, "missing", getCookbooksSlot); err == nil {
		t.Error("unknown session was accepted")
	}
	for _, characterID := range []int{-1, 10} {
		if _, err := GetRegions(engine, gameCatalog, sessionID, characterID); err == nil {
			t.Errorf("characterID %d was accepted", characterID)
		}
	}

	resources := patchRegionDocument(
		t, storedCookbookResources(t), "liurnia_lake_facing_cliffs",
		func(document *schema.RegionDocument) { document.RegionID.Value = getRegionsFirstID },
	)
	_, err := GetRegions(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot)
	if err == nil || !strings.Contains(err.Error(), "both declare region ID") {
		t.Fatalf("duplicate region ID error = %v", err)
	}
}

func TestRegionCatalogFailsClosed(t *testing.T) {
	resources := patchRegionDocument(
		t, storedCookbookResources(t), "limgrave_the_first_step",
		func(document *schema.RegionDocument) {
			document.Area = schema.Fact[string]{Provenance: document.Area.Provenance}
		},
	)
	if _, err := gamecatalog.New(storedCookbookCatalogData(t).Manifest, resources); err == nil {
		t.Fatal("catalog accepted a region without a known area")
	}
}
