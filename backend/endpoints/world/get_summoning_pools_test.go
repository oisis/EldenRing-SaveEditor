package world

import (
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	// The number of summoning pools the stored catalog declares, and two
	// neighbouring activation flags: the fixture sets the first and leaves the
	// second clear, so a shifted byte or an inverted bit direction fails here.
	getSummoningPoolsCount     = 213
	getSummoningPoolsSetFlag   = 670130
	getSummoningPoolsClearFlag = 670131
	getSummoningPoolsSetKey    = "stormveil_castle_gateside_chamber"
	getSummoningPoolsClearKey  = "stormveil_castle_liftside_chamber"

	// Block 670 occupies this BST position in the confirmed bitfield layout.
	getSummoningPoolsBlockPosition = 107
)

// writeGetSummoningPoolsFixture reuses the synthetic PC container of the
// cookbook tests and sets one block-670 flag directly, because the cookbook
// fixture only places the blocks its own flags live in.
func writeGetSummoningPoolsFixture(t *testing.T, active bool) string {
	t.Helper()

	path := writeGetCookbooksFixture(t, nil, active)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchorBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize + getCookbooksAnchorAt
	index := int64(getSummoningPoolsSetFlag % 1000)
	offset := getSummoningPoolsBlockPosition*getCookbooksBlockSize + index/8
	data[anchorBase+getCookbooksSectionAt+offset] |= 1 << uint8(7-index%8)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadSummoningPoolsSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetSummoningPoolsFixture(t, active), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestGetSummoningPoolsReturnsTheCuratedCatalogState(t *testing.T) {
	engine, sessionID := loadSummoningPoolsSession(t, true)
	result, err := GetSummoningPools(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSummoningPools: %v", err)
	}
	if !result.Active || result.SaveSessionID != sessionID ||
		result.CharacterID != getCookbooksSlot {
		t.Fatalf("result identity = %+v, want the active requested slot", result)
	}
	if len(result.SummoningPools) != getSummoningPoolsCount {
		t.Fatalf("summoning pool count = %d, want %d",
			len(result.SummoningPools), getSummoningPoolsCount)
	}

	// The three arenas of the separate colosseum list must not appear here.
	colosseumKeys := map[string]struct{}{
		"caelid_colosseum": {}, "limgrave_colosseum": {}, "royal_colosseum": {},
	}
	want := map[string]bool{
		getSummoningPoolsSetKey:   true,
		getSummoningPoolsClearKey: false,
	}
	found := make(map[string]bool, len(want))
	for index, entry := range result.SummoningPools {
		if entry.Kind != schema.ResourceKindSummoningPool {
			t.Fatalf("entry %q has kind %q", entry.Key, entry.Kind)
		}
		if _, isColosseum := colosseumKeys[entry.Key]; isColosseum {
			t.Fatalf("colosseum %q is reported as a summoning pool", entry.Key)
		}
		if entry.Name == "" || entry.RegionLabel == "" {
			t.Fatalf("entry %q = %+v, want a name and a region label", entry.Key, entry)
		}
		if index > 0 {
			previous := result.SummoningPools[index-1]
			if previous.RegionLabel > entry.RegionLabel ||
				(previous.RegionLabel == entry.RegionLabel && previous.Name > entry.Name) ||
				(previous.RegionLabel == entry.RegionLabel && previous.Name == entry.Name &&
					previous.Key > entry.Key) {
				t.Fatalf("summoning pools are not ordered at %q then %q", previous.Key, entry.Key)
			}
		}
		if activated, exists := want[entry.Key]; exists {
			found[entry.Key] = true
			if entry.Activated != activated {
				t.Errorf("summoning pool %q activated = %t, want %t",
					entry.Key, entry.Activated, activated)
			}
		}
	}
	if len(found) != len(want) {
		t.Fatalf("found summoning pool keys = %v, want all %v", found, want)
	}
}

func TestGetSummoningPoolsDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadSummoningPoolsSession(t, false)
	result, err := GetSummoningPools(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSummoningPools: %v", err)
	}
	if result.Active || len(result.SummoningPools) != getSummoningPoolsCount {
		t.Fatalf("active/count = %t/%d, want false/%d",
			result.Active, len(result.SummoningPools), getSummoningPoolsCount)
	}
	// The bitfield of the deleted character still carries the set flag, so an
	// entirely deactivated result proves the slot data was never decoded.
	for _, entry := range result.SummoningPools {
		if entry.Activated {
			t.Fatalf("residual slot reports %q activated", entry.Key)
		}
	}
}

func TestGetSummoningPoolsRejectsInvalidInputAndDuplicateFlag(t *testing.T) {
	engine, sessionID := loadSummoningPoolsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	if _, err := GetSummoningPools(nil, gameCatalog, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil SaveEngine was accepted")
	}
	if _, err := GetSummoningPools(engine, nil, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil GameCatalog was accepted")
	}
	if _, err := GetSummoningPools(engine, gameCatalog, "missing", getCookbooksSlot); err == nil {
		t.Error("unknown session was accepted")
	}

	resources := storedCookbookResources(t)
	patched := 0
	for index := range resources {
		if resources[index].Kind != schema.ResourceKindSummoningPool ||
			resources[index].Key != getSummoningPoolsClearKey {
			continue
		}
		document := *resources[index].SummoningPool
		document.ActivationEventFlagID.Value = getSummoningPoolsSetFlag
		resources[index].SummoningPool = &document
		patched++
	}
	if patched != 1 {
		t.Fatalf("patched %d summoning pools, want 1", patched)
	}
	_, err := GetSummoningPools(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot)
	if err == nil || !strings.Contains(err.Error(), "both declare event flag") {
		t.Fatalf("duplicate activation flag error = %v", err)
	}
}
