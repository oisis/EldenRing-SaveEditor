package world

import (
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	// The number of graces the stored catalog declares, and two neighbouring
	// visit flags: the fixture sets the first and leaves the second clear, so a
	// shifted byte or an inverted bit direction fails here: getGracesClearKey
	// declares the immediate neighbour flag 71001.
	getGracesCount    = 419
	getGracesSetFlag  = 71000
	getGracesSetKey   = "stormveil_castle_godrick_the_grafted"
	getGracesClearKey = "stormveil_castle_margit_the_fell_omen"

	// Block 71 occupies this BST position in the confirmed bitfield layout.
	getGracesBlockPosition = 21
)

// writeGetGracesFixture reuses the synthetic PC container of the cookbook tests
// and sets one block-71 flag directly, because the cookbook fixture only places
// the blocks its own flags live in.
func writeGetGracesFixture(t *testing.T, active bool) string {
	t.Helper()

	path := writeGetCookbooksFixture(t, nil, active)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchorBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize + getCookbooksAnchorAt
	index := int64(getGracesSetFlag % 1000)
	offset := getGracesBlockPosition*getCookbooksBlockSize + index/8
	data[anchorBase+getCookbooksSectionAt+offset] |= 1 << uint8(7-index%8)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadGracesSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetGracesFixture(t, active), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestGetGracesReturnsTheCuratedCatalogState(t *testing.T) {
	engine, sessionID := loadGracesSession(t, true)
	result, err := GetGraces(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetGraces: %v", err)
	}
	if !result.Active || result.SaveSessionID != sessionID ||
		result.CharacterID != getCookbooksSlot {
		t.Fatalf("result identity = %+v, want the active requested slot", result)
	}
	if len(result.Graces) != getGracesCount {
		t.Fatalf("grace count = %d, want %d", len(result.Graces), getGracesCount)
	}

	want := map[string]bool{getGracesSetKey: true, getGracesClearKey: false}
	found := make(map[string]bool, len(want))
	dungeonTypes := make(map[string]struct{})
	bossArena := false
	for index, entry := range result.Graces {
		if entry.Kind != schema.ResourceKindGrace {
			t.Fatalf("entry %q has kind %q", entry.Key, entry.Kind)
		}
		if entry.Name == "" || entry.RegionLabel == "" {
			t.Fatalf("entry %q = %+v, want a name and a region label", entry.Key, entry)
		}
		// The composed legacy name is split, so no entry may carry the region
		// suffix and the raw event flag must not leak into any public field.
		if strings.Contains(entry.Name, "(") || strings.Contains(entry.Key, "71000") {
			t.Fatalf("entry %q = %+v leaks source formatting or a raw flag", entry.Key, entry)
		}
		dungeonTypes[entry.DungeonType] = struct{}{}
		if entry.BossArena {
			bossArena = true
		}
		if index > 0 {
			previous := result.Graces[index-1]
			if previous.RegionLabel > entry.RegionLabel ||
				(previous.RegionLabel == entry.RegionLabel && previous.Name > entry.Name) ||
				(previous.RegionLabel == entry.RegionLabel && previous.Name == entry.Name &&
					previous.Key > entry.Key) {
				t.Fatalf("graces are not ordered at %q then %q", previous.Key, entry.Key)
			}
		}
		if visited, exists := want[entry.Key]; exists {
			found[entry.Key] = true
			if entry.Visited != visited {
				t.Errorf("grace %q visited = %t, want %t", entry.Key, entry.Visited, visited)
			}
		}
	}
	if len(found) != len(want) {
		t.Fatalf("found grace keys = %v, want all %v", found, want)
	}
	// The exact count is a curated-data fact the migrator owns; here it only has
	// to prove the field is carried through instead of staying uniformly false.
	if !bossArena {
		t.Error("no entry carries bossArena")
	}
	for _, dungeonType := range []string{
		schema.GraceDungeonTypeNone,
		schema.GraceDungeonTypeCatacomb,
		schema.GraceDungeonTypeHeroGrave,
	} {
		if _, present := dungeonTypes[dungeonType]; !present {
			t.Errorf("no entry carries dungeon type %q", dungeonType)
		}
	}
}

func TestGetGracesDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadGracesSession(t, false)
	result, err := GetGraces(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetGraces: %v", err)
	}
	if result.Active || len(result.Graces) != getGracesCount {
		t.Fatalf("active/count = %t/%d, want false/%d",
			result.Active, len(result.Graces), getGracesCount)
	}
	// The bitfield of the deleted character still carries the set flag, so an
	// entirely unvisited result proves the slot data was never decoded.
	for _, entry := range result.Graces {
		if entry.Visited {
			t.Fatalf("residual slot reports %q visited", entry.Key)
		}
	}
}

func TestGetGracesRejectsInvalidInputAndDuplicateFlag(t *testing.T) {
	engine, sessionID := loadGracesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	if _, err := GetGraces(nil, gameCatalog, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil SaveEngine was accepted")
	}
	if _, err := GetGraces(engine, nil, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil GameCatalog was accepted")
	}
	if _, err := GetGraces(engine, gameCatalog, "missing", getCookbooksSlot); err == nil {
		t.Error("unknown session was accepted")
	}

	resources := storedCookbookResources(t)
	patched := 0
	for index := range resources {
		if resources[index].Kind != schema.ResourceKindGrace ||
			resources[index].Key != getGracesClearKey {
			continue
		}
		document := *resources[index].Grace
		document.VisitEventFlagID.Value = getGracesSetFlag
		resources[index].Grace = &document
		patched++
	}
	if patched != 1 {
		t.Fatalf("patched %d graces, want 1", patched)
	}
	_, err := GetGraces(engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot)
	if err == nil || !strings.Contains(err.Error(), "both declare event flag") {
		t.Fatalf("duplicate visit flag error = %v", err)
	}
}
