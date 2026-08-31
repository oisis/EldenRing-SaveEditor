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
	getPhysickMixtureHeaderSize       = 0x300
	getPhysickMixtureEntryCountOffset = 0x0C
	getPhysickMixtureEntryCount       = 12
	getPhysickMixtureSlotBlockSize    = 0x280010
	getPhysickMixtureFixtureSize      = int64(getPhysickMixtureHeaderSize) + 10*getPhysickMixtureSlotBlockSize + 0x60010

	getPhysickMixtureUserData10Offset = int64(getPhysickMixtureHeaderSize) + 10*getPhysickMixtureSlotBlockSize + 0x10
	getPhysickMixtureFlagsOffset      = 0x1954

	getPhysickMixtureSlot            = 3
	getPhysickMixtureAnchorAt        = 0x0640
	getPhysickMixtureCountAt         = 0x931D
	getPhysickMixtureArmamentsSize   = 0x9C
	getPhysickMixtureProjectileCount = 17
)

// getPhysickMixtureAnchor is the 65-byte anchor the Physick chain is measured
// from, restated here independently of the implementation: one leading 0x00
// byte, then four full repetitions of a 16-byte block made of 0xFF 0xFF 0xFF
// 0xFF followed by twelve 0x00 bytes.
var getPhysickMixtureAnchor = []byte{
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

// writeGetPhysickMixtureFixture writes a minimal synthetic PC save into
// t.TempDir() with one active character carrying the given raw Tear
// identifiers, and returns its path. The slot declares a non-zero projectile
// count, so the block is only found by following the dynamic length.
func writeGetPhysickMixtureFixture(t *testing.T, tears [2]uint32) string {
	t.Helper()

	data := make([]byte, getPhysickMixtureFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[getPhysickMixtureEntryCountOffset:], getPhysickMixtureEntryCount)

	data[getPhysickMixtureUserData10Offset+getPhysickMixtureFlagsOffset+getPhysickMixtureSlot] = 1

	slotBase := int64(getPhysickMixtureHeaderSize) + 0x10 + getPhysickMixtureSlot*getPhysickMixtureSlotBlockSize
	copy(data[slotBase+getPhysickMixtureAnchorAt:], getPhysickMixtureAnchor)

	countAt := slotBase + getPhysickMixtureAnchorAt + getPhysickMixtureCountAt
	binary.LittleEndian.PutUint32(data[countAt:], getPhysickMixtureProjectileCount)

	blockAt := countAt + 4 + getPhysickMixtureProjectileCount*8 + getPhysickMixtureArmamentsSize
	binary.LittleEndian.PutUint32(data[blockAt:], tears[0])
	binary.LittleEndian.PutUint32(data[blockAt+4:], tears[1])

	path := filepath.Join(t.TempDir(), "get-physick-mixture.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetPhysickMixtureReturnsTheActiveMixtureOfALoadedSession(t *testing.T) {
	tears := [2]uint32{0x40002AF9, 0xFFFFFFFF}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetPhysickMixtureFixture(t, tears), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetPhysickMixture(engine, loaded.SaveSessionID, getPhysickMixtureSlot)
	if err != nil {
		t.Fatalf("GetPhysickMixture: %v", err)
	}

	want := GetPhysickMixtureResult{
		SaveSessionID: loaded.SaveSessionID,
		SaveRevision:  "0",
		CharacterID:   getPhysickMixtureSlot,
		Active:        true,
		Tears:         tears,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetPhysickMixtureRejectsMissingEngine(t *testing.T) {
	result, err := GetPhysickMixture(nil, "any-session", 0)
	if err == nil {
		t.Fatal("GetPhysickMixture accepted a nil engine")
	}
	if err.Error() != "save engine is not available" {
		t.Errorf("error = %q, want %q", err, "save engine is not available")
	}
	if !reflect.DeepEqual(result, GetPhysickMixtureResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
}
