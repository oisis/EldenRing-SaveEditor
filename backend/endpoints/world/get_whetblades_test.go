package world

import (
	"encoding/binary"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	getWhetbladesInventoryCommonAt = 505
	getWhetbladesInventoryRecord   = 12
	getWhetbladesInventoryKeyAt    = getWhetbladesInventoryCommonAt + 0xA80*12 + 4
)

func writeGetWhetbladesFixture(t *testing.T, active bool) string {
	t.Helper()
	path := writeGetCookbooksFixture(t, nil, active)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	slotBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize
	anchorBase := slotBase + getCookbooksAnchorAt

	for id, position := range map[uint32]int64{60130: 10, 65610: 15} {
		index := int64(id % 1000)
		offset := position*getCookbooksBlockSize + index/8
		data[anchorBase+getCookbooksSectionAt+offset] |= 1 << uint8(7-index%8)
	}
	putRecord := func(at int64, handle, quantity uint32) {
		binary.LittleEndian.PutUint32(data[anchorBase+at:], handle)
		binary.LittleEndian.PutUint32(data[anchorBase+at+4:], quantity)
	}
	putRecord(getWhetbladesInventoryCommonAt, 0x4000230B, 1)
	putRecord(getWhetbladesInventoryCommonAt+getWhetbladesInventoryRecord, 0xB000230D, 0)
	putRecord(getWhetbladesInventoryKeyAt, 0xB000230E, 1)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadWhetbladesSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetWhetbladesFixture(t, active), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestGetWhetbladesJoinsFlagsAndInventory(t *testing.T) {
	engine, sessionID := loadWhetbladesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	result, err := GetWhetblades(engine, gameCatalog, sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetWhetblades: %v", err)
	}
	if !result.Active || len(result.Whetblades) != 6 {
		t.Fatalf("result active/count = %t/%d, want true/6", result.Active, len(result.Whetblades))
	}
	want := map[string]bool{
		"Black Whetblade": true, "Glintstone Whetblade": false,
		"Iron Whetblade": true, "Red-Hot Whetblade": true,
		"Sanctified Whetblade": false, "Whetstone Knife": true,
	}
	wantOrder := []string{
		"Black Whetblade", "Glintstone Whetblade", "Iron Whetblade",
		"Red-Hot Whetblade", "Sanctified Whetblade", "Whetstone Knife",
	}
	got := make(map[string]bool, len(result.Whetblades))
	for index, entry := range result.Whetblades {
		if entry.Kind != "item" || entry.Key == "" {
			t.Fatalf("entry has invalid identity: %+v", entry)
		}
		if entry.Name != wantOrder[index] {
			t.Errorf("entry %d name = %q, want %q", index, entry.Name, wantOrder[index])
		}
		got[entry.Name] = entry.Unlocked
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("states = %#v, want %#v", got, want)
	}

	for filter, wantCount := range map[string]int{
		WhetbladeAvailabilityUnlocked: 4,
		WhetbladeAvailabilityLocked:   2,
	} {
		filtered, err := GetWhetblades(
			engine, gameCatalog, sessionID, getCookbooksSlot, filter)
		if err != nil {
			t.Fatalf("GetWhetblades(%q): %v", filter, err)
		}
		if len(filtered.Whetblades) != wantCount {
			t.Errorf("filter %q returned %d entries, want %d",
				filter, len(filtered.Whetblades), wantCount)
		}
	}
}

func TestGetWhetbladesDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadWhetbladesSession(t, false)
	result, err := GetWhetblades(
		engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetWhetblades: %v", err)
	}
	if result.Active || len(result.Whetblades) != 6 {
		t.Fatalf("result active/count = %t/%d, want false/6",
			result.Active, len(result.Whetblades))
	}
	for _, entry := range result.Whetblades {
		if entry.Unlocked {
			t.Errorf("residual slot reports unlocked entry %+v", entry)
		}
	}
}

func TestGetWhetbladesRejectsInvalidInput(t *testing.T) {
	engine, sessionID := loadWhetbladesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	if _, err := GetWhetblades(
		engine, gameCatalog, sessionID, getCookbooksSlot, "Unlocked"); err == nil || !strings.Contains(err.Error(), "availabilityFilter") {
		t.Fatalf("invalid filter error = %v", err)
	}
	if _, err := GetWhetblades(
		nil, gameCatalog, sessionID, getCookbooksSlot, ""); err == nil {
		t.Fatal("missing SaveEngine was accepted")
	}
	if _, err := GetWhetblades(
		engine, nil, sessionID, getCookbooksSlot, ""); err == nil {
		t.Fatal("missing GameCatalog was accepted")
	}
}

func TestGetWhetbladesRejectsAFlagFromAnotherSupportedDomain(t *testing.T) {
	engine, sessionID := loadWhetbladesSession(t, true)
	resources := patchCookbookDocument(t, storedCookbookResources(t), "4000218E",
		func(document *schema.ItemDocument) {
			for index := range document.Unlocks {
				if document.Unlocks[index].Kind.Known &&
					document.Unlocks[index].Kind.Value == whetbladeUnlockKind {
					document.Unlocks[index].EventFlagID.Value = 67000
				}
			}
		})

	result, err := GetWhetblades(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot, "")
	if err == nil {
		t.Fatal("GetWhetblades accepted a cookbook event flag for a whetblade")
	}
	want := "event flag 67000 lies in block 67, which this reader does not support"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if len(result.Whetblades) != 0 || result.Active {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
