package favorites

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

var endpointAppearanceAnchor = []byte{
	0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

func writeEndpointSetFavoritesFixture(t *testing.T, charSlot int) string {
	t.Helper()

	data := make([]byte, pcUserData10Offset+pcUserData10BlockSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[0x0C:], 12)
	base := pcUserData10DataOffset

	// Active flag
	data[base+0x1954+int64(charSlot)] = 1

	// Character slot data
	slotBase := int64(pcHeaderSize) + 0x10 + int64(charSlot)*pcSlotBlockSize

	// Anchor + gender
	anchor := slotBase + 0x1000
	copy(data[anchor:], endpointAppearanceAnchor)
	data[anchor-249] = 1 // Male
	data[anchor-245] = 1

	// Face data header
	faceAt := slotBase + 0x2000
	copy(data[faceAt:], []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'})
	binary.LittleEndian.PutUint32(data[faceAt+0x08:], 4)
	binary.LittleEndian.PutUint32(data[faceAt+0x0C:], 0x120)

	path := filepath.Join(t.TempDir(), "endpoint_set_favorites.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestSetFavoritePresetEndpointSuccess(t *testing.T) {
	path := writeEndpointSetFavoritesFixture(t, 0)
	engine := saveengine.New()
	session, err := engine.LoadSave(path, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetFavoritePreset(engine, session.SaveSessionID, 3, 0, "0")
	if err != nil {
		t.Fatalf("SetFavoritePreset: %v", err)
	}

	if result.SaveSessionID != session.SaveSessionID {
		t.Errorf("SaveSessionID = %q, want %q", result.SaveSessionID, session.SaveSessionID)
	}
	if result.SaveRevision != "1" {
		t.Errorf("SaveRevision = %q, want \"1\"", result.SaveRevision)
	}
	if result.FavoriteSlotID != 3 {
		t.Errorf("FavoriteSlotID = %d, want 3", result.FavoriteSlotID)
	}
	if result.SourceCharacterID != 0 {
		t.Errorf("SourceCharacterID = %d, want 0", result.SourceCharacterID)
	}
}

func TestSetFavoritePresetEndpointRejectsMissingEngine(t *testing.T) {
	if _, err := SetFavoritePreset(nil, "session", 0, 0, "0"); err == nil {
		t.Fatal("expected error for nil engine, got nil")
	}
}

func TestSetFavoritePresetEndpointDelegatesValidation(t *testing.T) {
	path := writeEndpointSetFavoritesFixture(t, 0)
	engine := saveengine.New()
	session, err := engine.LoadSave(path, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	if _, err := SetFavoritePreset(engine, session.SaveSessionID, 15, 0, "0"); err == nil {
		t.Fatal("expected error for out-of-range favoriteSlotID, got nil")
	}
}
