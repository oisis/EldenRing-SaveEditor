package world

import (
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	// The number of bosses the stored catalog declares, and two neighbouring
	// defeat flags: the fixture sets the first and leaves the second clear, so a
	// shifted byte or an inverted bit direction fails here. getBossesClearKey
	// declares the immediate neighbour flag 9101.
	getBossesCount    = 110
	getBossesSetFlag  = 9100
	getBossesSetKey   = "stormveil_castle_margit_the_fell_omen"
	getBossesClearKey = "stormveil_castle_godrick_the_grafted"

	// Block 9 occupies this BST position in the confirmed bitfield layout.
	getBossesBlockPosition = 9
)

// writeGetBossesFixture reuses the synthetic PC container of the cookbook tests
// and sets one block-9 flag directly, because the cookbook fixture only places
// the blocks its own flags live in.
func writeGetBossesFixture(t *testing.T, active bool) string {
	t.Helper()

	path := writeGetCookbooksFixture(t, nil, active)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchorBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize + getCookbooksAnchorAt
	index := int64(getBossesSetFlag % 1000)
	offset := getBossesBlockPosition*getCookbooksBlockSize + index/8
	data[anchorBase+getCookbooksSectionAt+offset] |= 1 << uint8(7-index%8)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadBossesSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetBossesFixture(t, active), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestGetBossesReturnsTheCuratedCatalogState(t *testing.T) {
	engine, sessionID := loadBossesSession(t, true)
	result, err := GetBosses(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetBosses: %v", err)
	}
	if !result.Active || result.SaveSessionID != sessionID ||
		result.CharacterID != getCookbooksSlot {
		t.Fatalf("result identity = %+v, want the active requested slot", result)
	}
	if len(result.Bosses) != getBossesCount {
		t.Fatalf("boss count = %d, want %d", len(result.Bosses), getBossesCount)
	}

	want := map[string]bool{getBossesSetKey: true, getBossesClearKey: false}
	found := make(map[string]bool, len(want))
	encounterTypes := make(map[string]struct{})
	remembrance := false
	for index, entry := range result.Bosses {
		if entry.Kind != schema.ResourceKindBoss {
			t.Fatalf("entry %q has kind %q", entry.Key, entry.Kind)
		}
		if entry.Name == "" || entry.RegionLabel == "" {
			t.Fatalf("entry %q = %+v, want a name and a region label", entry.Key, entry)
		}
		// The raw defeat flag must not leak into any public field.
		if strings.Contains(entry.Key, "9100") {
			t.Fatalf("entry %q leaks a raw event flag", entry.Key)
		}
		encounterTypes[entry.EncounterType] = struct{}{}
		if entry.Remembrance {
			remembrance = true
		}
		if index > 0 {
			previous := result.Bosses[index-1]
			if previous.RegionLabel > entry.RegionLabel ||
				(previous.RegionLabel == entry.RegionLabel && previous.Name > entry.Name) ||
				(previous.RegionLabel == entry.RegionLabel && previous.Name == entry.Name &&
					previous.Key > entry.Key) {
				t.Fatalf("bosses are not ordered at %q then %q", previous.Key, entry.Key)
			}
		}
		if defeated, exists := want[entry.Key]; exists {
			found[entry.Key] = true
			if entry.Defeated != defeated {
				t.Errorf("boss %q defeated = %t, want %t", entry.Key, entry.Defeated, defeated)
			}
		}
	}
	if len(found) != len(want) {
		t.Fatalf("found boss keys = %v, want all %v", found, want)
	}
	// The exact counts are authored-catalog facts; here this endpoint test only
	// has to prove the fields are carried through instead of staying uniform.
	if !remembrance {
		t.Error("no entry carries remembrance")
	}
	for _, encounterType := range []string{
		schema.BossEncounterTypeMain, schema.BossEncounterTypeField,
	} {
		if _, present := encounterTypes[encounterType]; !present {
			t.Errorf("no entry carries encounter type %q", encounterType)
		}
	}
}

func TestGetBossesDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadBossesSession(t, false)
	result, err := GetBosses(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetBosses: %v", err)
	}
	if result.Active || len(result.Bosses) != getBossesCount {
		t.Fatalf("active/count = %t/%d, want false/%d",
			result.Active, len(result.Bosses), getBossesCount)
	}
	// The bitfield of the deleted character still carries the set flag, so an
	// entirely undefeated result proves the slot data was never decoded.
	for _, entry := range result.Bosses {
		if entry.Defeated {
			t.Fatalf("residual slot reports %q defeated", entry.Key)
		}
	}
}

func TestGetBossesRejectsInvalidInputAndDuplicateFlag(t *testing.T) {
	engine, sessionID := loadBossesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	if _, err := GetBosses(nil, gameCatalog, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil SaveEngine was accepted")
	}
	if _, err := GetBosses(engine, nil, sessionID, getCookbooksSlot); err == nil {
		t.Error("nil GameCatalog was accepted")
	}
	if _, err := GetBosses(engine, gameCatalog, "missing", getCookbooksSlot); err == nil {
		t.Error("unknown session was accepted")
	}

	resources := storedCookbookResources(t)
	patched := 0
	for index := range resources {
		if resources[index].Kind != schema.ResourceKindBoss ||
			resources[index].Key != getBossesClearKey {
			continue
		}
		document := *resources[index].Boss
		document.DefeatEventFlagID.Value = getBossesSetFlag
		resources[index].Boss = &document
		patched++
	}
	if patched != 1 {
		t.Fatalf("patched %d bosses, want 1", patched)
	}
	_, err := GetBosses(engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot)
	if err == nil || !strings.Contains(err.Error(), "both declare event flag") {
		t.Fatalf("duplicate defeat flag error = %v", err)
	}
}
