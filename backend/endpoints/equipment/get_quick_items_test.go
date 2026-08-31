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
	getQuickItemsHeaderSize       = 0x300
	getQuickItemsEntryCountOffset = 0x0C
	getQuickItemsEntryCount       = 12
	getQuickItemsSlotBlockSize    = 0x280010
	getQuickItemsFixtureSize      = int64(getQuickItemsHeaderSize) + 10*getQuickItemsSlotBlockSize + 0x60010

	getQuickItemsUserData10Offset = int64(getQuickItemsHeaderSize) + 10*getQuickItemsSlotBlockSize + 0x10
	getQuickItemsFlagsOffset      = 0x1954

	getQuickItemsSlot      = 5
	getQuickItemsAnchorAt  = 0x0640
	getQuickItemsSectionAt = 0x9279
	getQuickItemsActiveAt  = 0x50
)

// getQuickItemsAnchor is the 65-byte anchor the quick-items chain is measured
// from, restated here independently of the implementation: one leading 0x00
// byte, then four full repetitions of a 16-byte block made of 0xFF 0xFF 0xFF
// 0xFF followed by twelve 0x00 bytes.
var getQuickItemsAnchor = []byte{
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

// writeGetQuickItemsFixture writes a minimal synthetic PC save into t.TempDir()
// with one active character carrying the given raw quick items and active-slot
// value, and returns its path.
func writeGetQuickItemsFixture(
	t *testing.T,
	items [10]saveengine.QuickItemSlot,
	activeQuick uint32,
) string {
	t.Helper()

	data := make([]byte, getQuickItemsFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getQuickItemsEntryCountOffset:], getQuickItemsEntryCount)

	data[getQuickItemsUserData10Offset+getQuickItemsFlagsOffset+getQuickItemsSlot] = 1

	slotBase := int64(getQuickItemsHeaderSize) + 0x10 + getQuickItemsSlot*getQuickItemsSlotBlockSize
	copy(data[slotBase+getQuickItemsAnchorAt:], getQuickItemsAnchor)

	sectionAt := slotBase + getQuickItemsAnchorAt + getQuickItemsSectionAt
	for index, item := range items {
		binary.LittleEndian.PutUint32(data[sectionAt+int64(index)*8:], item.ItemID)
		binary.LittleEndian.PutUint32(data[sectionAt+int64(index)*8+4:], item.EquipIndex)
	}
	binary.LittleEndian.PutUint32(data[sectionAt+getQuickItemsActiveAt:], activeQuick)

	path := filepath.Join(t.TempDir(), "get-quick-items.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetQuickItemsReturnsTheActiveQuickItemsOfALoadedSession(t *testing.T) {
	items := [10]saveengine.QuickItemSlot{
		{ItemID: 0, EquipIndex: 0xFFFFFFFF},
		{ItemID: 0x40002AF9, EquipIndex: 1},
		{ItemID: 0x400003E8, EquipIndex: 2},
		{ItemID: 0xFFFFFFFF, EquipIndex: 0xFFFFFFFF},
		{ItemID: 0x80000A01, EquipIndex: 0x80000004},
		{ItemID: 0x400007D0, EquipIndex: 5},
		{ItemID: 0x40000BB8, EquipIndex: 6},
		{ItemID: 0, EquipIndex: 0},
		{ItemID: 0x40000FA0, EquipIndex: 8},
		{ItemID: 0x90000B02, EquipIndex: 0xFFFFFFF6},
	}

	engine := saveengine.New()
	// 0xFFFFFFF6 is -10: the value keeps its sign through the endpoint.
	loaded, err := engine.LoadSave(writeGetQuickItemsFixture(t, items, 0xFFFFFFF6), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetQuickItems(engine, loaded.SaveSessionID, getQuickItemsSlot)
	if err != nil {
		t.Fatalf("GetQuickItems: %v", err)
	}

	want := GetQuickItemsResult{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   getQuickItemsSlot,
		Active:        true,
		Items:         items,
		ActiveQuick:   -10,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetQuickItemsRejectsMissingEngine(t *testing.T) {
	result, err := GetQuickItems(nil, "any-session", 0)
	if err == nil {
		t.Fatal("GetQuickItems accepted a nil engine")
	}
	if err.Error() != "save engine is not available" {
		t.Errorf("error = %q, want %q", err, "save engine is not available")
	}
	if !reflect.DeepEqual(result, GetQuickItemsResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
