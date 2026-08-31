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
	getEquipmentHeaderSize       = 0x300
	getEquipmentEntryCountOffset = 0x0C
	getEquipmentEntryCount       = 12
	getEquipmentSlotBlockSize    = 0x280010
	getEquipmentFixtureSize      = int64(getEquipmentHeaderSize) + 10*getEquipmentSlotBlockSize + 0x60010

	getEquipmentUserData10Offset = int64(getEquipmentHeaderSize) + 10*getEquipmentSlotBlockSize + 0x10
	getEquipmentFlagsOffset      = 0x1954

	getEquipmentSlot            = 5
	getEquipmentAnchorAt        = 0x0640
	getEquipmentCountAt         = 0x931D
	getEquipmentProjectileCount = 9
)

// getEquipmentAnchor is the 65-byte anchor the equipment chain is measured from,
// restated here independently of the implementation: one leading 0x00 byte, then
// four full repetitions of a 16-byte block made of 0xFF 0xFF 0xFF 0xFF followed
// by twelve 0x00 bytes.
var getEquipmentAnchor = []byte{
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

// writeGetEquipmentFixture writes a minimal synthetic PC save into t.TempDir()
// with one active character carrying the given raw equipment, behind a non-zero
// acquired-projectiles section, and returns its path.
func writeGetEquipmentFixture(t *testing.T, slots [22]uint32) string {
	t.Helper()

	data := make([]byte, getEquipmentFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getEquipmentEntryCountOffset:], getEquipmentEntryCount)

	data[getEquipmentUserData10Offset+getEquipmentFlagsOffset+getEquipmentSlot] = 1

	slotBase := int64(getEquipmentHeaderSize) + 0x10 + getEquipmentSlot*getEquipmentSlotBlockSize
	copy(data[slotBase+getEquipmentAnchorAt:], getEquipmentAnchor)

	countAt := slotBase + getEquipmentAnchorAt + getEquipmentCountAt
	binary.LittleEndian.PutUint32(data[countAt:], getEquipmentProjectileCount)

	blockAt := countAt + 4 + getEquipmentProjectileCount*8
	for index, value := range slots {
		binary.LittleEndian.PutUint32(data[blockAt+int64(index)*4:], value)
	}

	path := filepath.Join(t.TempDir(), "get-equipment.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetEquipmentReturnsTheActiveEquipmentOfALoadedSession(t *testing.T) {
	slots := [22]uint32{
		0x8000012C, 0x80000190, 0x000001F4, 0x80000258, 0x000002BC, 0x80000320,
		0x0000012D, 0x00000191, 0xFFFFFFFF, 0x000001F5, 0, 0x0000025A,
		0x90000BB8, 0x90000C1C, 0x90000C80, 0x90000CE4, 0x00000D48,
		0x00002710, 0x00002711, 0x00002712, 0x00002713, 0x00002714,
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetEquipmentFixture(t, slots), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetEquipment(engine, loaded.SaveSessionID, getEquipmentSlot)
	if err != nil {
		t.Fatalf("GetEquipment: %v", err)
	}

	want := GetEquipmentResult{
		SaveSessionID: loaded.SaveSessionID,
		SaveRevision:  "0",
		CharacterID:   getEquipmentSlot,
		Active:        true,
		Slots:         slots,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetEquipmentRejectsMissingEngine(t *testing.T) {
	result, err := GetEquipment(nil, "any-session", 0)
	if err == nil {
		t.Fatal("GetEquipment accepted a nil engine")
	}
	if err.Error() != "save engine is not available" {
		t.Errorf("error = %q, want %q", err, "save engine is not available")
	}
	if !reflect.DeepEqual(result, GetEquipmentResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
