package world

import (
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

var getBellBearingsSetFlags = []uint32{11109710, 11109751, 11109790}

func writeGetBellBearingsFixture(t *testing.T, active bool) string {
	t.Helper()
	path := writeGetCookbooksFixture(t, nil, active)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	slotBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize
	sectionAt := slotBase + getCookbooksAnchorAt + getCookbooksSectionAt
	for _, id := range getBellBearingsSetFlags {
		index := int64(id % 1000)
		offset := int64(11129)*getCookbooksBlockSize + index/8
		data[sectionAt+offset] |= 1 << uint8(7-index%8)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadBellBearingsSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetBellBearingsFixture(t, active), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestGetBellBearingsReadsTheStoredCatalog(t *testing.T) {
	engine, sessionID := loadBellBearingsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	result, err := GetBellBearings(engine, gameCatalog, sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetBellBearings: %v", err)
	}
	if !result.Active || len(result.BellBearings) != 62 {
		t.Fatalf("result active/count = %t/%d, want true/62",
			result.Active, len(result.BellBearings))
	}
	unlocked := 0
	for index, entry := range result.BellBearings {
		if entry.Kind != schema.ResourceKindItem || entry.Key == "" ||
			entry.Name == "" || entry.Category == "" {
			t.Fatalf("entry has incomplete public data: %+v", entry)
		}
		if index > 0 {
			previous := result.BellBearings[index-1]
			if previous.Category > entry.Category ||
				(previous.Category == entry.Category && previous.Name > entry.Name) ||
				(previous.Category == entry.Category && previous.Name == entry.Name && previous.Key > entry.Key) {
				t.Fatalf("entries are not ordered at %d: %+v before %+v", index, previous, entry)
			}
		}
		if entry.Unlocked {
			unlocked++
		}
	}
	if unlocked != len(getBellBearingsSetFlags) {
		t.Errorf("unlocked entries = %d, want %d", unlocked, len(getBellBearingsSetFlags))
	}

	for filter, wantCount := range map[string]int{
		BellBearingAvailabilityUnlocked: 3,
		BellBearingAvailabilityLocked:   59,
	} {
		filtered, err := GetBellBearings(
			engine, gameCatalog, sessionID, getCookbooksSlot, filter)
		if err != nil {
			t.Fatalf("GetBellBearings(%q): %v", filter, err)
		}
		if len(filtered.BellBearings) != wantCount {
			t.Errorf("filter %q returned %d entries, want %d",
				filter, len(filtered.BellBearings), wantCount)
		}
	}
}

func TestGetBellBearingsDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadBellBearingsSession(t, false)
	result, err := GetBellBearings(
		engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetBellBearings: %v", err)
	}
	if result.Active || len(result.BellBearings) != 62 {
		t.Fatalf("result active/count = %t/%d, want false/62",
			result.Active, len(result.BellBearings))
	}
	for _, entry := range result.BellBearings {
		if entry.Unlocked {
			t.Errorf("residual slot reports unlocked entry %+v", entry)
		}
	}
}

func TestGetBellBearingsRejectsInvalidInputAndForeignBlocks(t *testing.T) {
	engine, sessionID := loadBellBearingsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	if _, err := GetBellBearings(
		engine, gameCatalog, sessionID, getCookbooksSlot, "Unlocked"); err == nil ||
		!strings.Contains(err.Error(), "availabilityFilter") {
		t.Fatalf("invalid filter error = %v", err)
	}
	if _, err := GetBellBearings(
		nil, gameCatalog, sessionID, getCookbooksSlot, ""); err == nil {
		t.Fatal("missing SaveEngine was accepted")
	}
	if _, err := GetBellBearings(
		engine, nil, sessionID, getCookbooksSlot, ""); err == nil {
		t.Fatal("missing GameCatalog was accepted")
	}

	resources := patchCookbookDocument(t, storedCookbookResources(t), "400022CE",
		func(document *schema.ItemDocument) {
			for index := range document.Unlocks {
				if document.Unlocks[index].Kind.Known &&
					document.Unlocks[index].Kind.Value == bellBearingUnlockKind {
					document.Unlocks[index].EventFlagID.Value = 67000
				}
			}
		})
	result, err := GetBellBearings(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot, "")
	if err == nil {
		t.Fatal("GetBellBearings accepted a cookbook event flag")
	}
	want := "event flag 67000 lies in block 67, which this reader does not support"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if len(result.BellBearings) != 0 || result.Active {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
