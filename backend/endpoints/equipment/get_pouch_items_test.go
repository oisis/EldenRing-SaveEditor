package equipment

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Synthetic PC container layout used only by this test. The endpoint owns none
// of these values; they are duplicated here so the fixture is accepted by
// SaveEngine without sharing anything with another test file.
const (
	getPouchItemsHeaderSize       = 0x300
	getPouchItemsEntryCountOffset = 0x0C
	getPouchItemsEntryCount       = 12
	getPouchItemsSlotBlockSize    = 0x280010
	getPouchItemsFixtureSize      = int64(getPouchItemsHeaderSize) + 10*getPouchItemsSlotBlockSize + 0x60010

	getPouchItemsUserData10Offset = int64(getPouchItemsHeaderSize) + 10*getPouchItemsSlotBlockSize + 0x10
	getPouchItemsFlagsOffset      = 0x1954

	getPouchItemsSlot      = 3
	getPouchItemsAnchorAt  = 0x0640
	getPouchItemsSectionAt = 0x92CD
)

// getPouchItemsAnchor is the 65-byte anchor the pouch chain is measured from,
// restated here independently of the implementation: one leading 0x00 byte, then
// four full repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed
// by twelve 0x00 bytes.
var getPouchItemsAnchor = []byte{
	0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

	0xFF, 0xFF, 0xFF, 0xFF,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// writeGetPouchItemsFixture writes a minimal synthetic PC save into t.TempDir()
// with one active character carrying the given raw pouch records, and returns
// its path.
func writeGetPouchItemsFixture(t *testing.T, items [6]saveengine.PouchItemSlot) string {
	t.Helper()

	data := make([]byte, getPouchItemsFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getPouchItemsEntryCountOffset:], getPouchItemsEntryCount)

	data[getPouchItemsUserData10Offset+getPouchItemsFlagsOffset+getPouchItemsSlot] = 1

	slotBase := int64(getPouchItemsHeaderSize) + 0x10 + getPouchItemsSlot*getPouchItemsSlotBlockSize
	copy(data[slotBase+getPouchItemsAnchorAt:], getPouchItemsAnchor)

	sectionAt := slotBase + getPouchItemsAnchorAt + getPouchItemsSectionAt
	for index, item := range items {
		binary.LittleEndian.PutUint32(data[sectionAt+int64(index)*8:], item.ItemID)
		binary.LittleEndian.PutUint32(data[sectionAt+int64(index)*8+4:], item.EquipIndex)
	}

	path := filepath.Join(t.TempDir(), "get-pouch-items.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetPouchItemsReturnsTheActivePouchItemsOfALoadedSession(t *testing.T) {
	items := [6]saveengine.PouchItemSlot{
		{ItemID: 0, EquipIndex: 0xFFFFFFFF},
		{ItemID: 0x40002AF9, EquipIndex: 1},
		{ItemID: 0xFFFFFFFF, EquipIndex: 0xFFFFFFFF},
		{ItemID: 0x80000A01, EquipIndex: 0x80000004},
		{ItemID: 0, EquipIndex: 0},
		{ItemID: 0x90000B02, EquipIndex: 0xFFFFFFF6},
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetPouchItemsFixture(t, items), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetPouchItems(engine, loaded.SaveSessionID, getPouchItemsSlot)
	if err != nil {
		t.Fatalf("GetPouchItems: %v", err)
	}

	want := GetPouchItemsResult{
		SaveSessionID: loaded.SaveSessionID,
		SaveRevision:  "0",
		CharacterID:   getPouchItemsSlot,
		Active:        true,
		Items:         items,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetPouchItemsRejectsMissingEngine(t *testing.T) {
	result, err := GetPouchItems(nil, "any-session", 0)
	if err == nil {
		t.Fatal("GetPouchItems accepted a nil engine")
	}
	if err.Error() != "save engine is not available" {
		t.Errorf("error = %q, want %q", err, "save engine is not available")
	}
	if !reflect.DeepEqual(result, GetPouchItemsResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
