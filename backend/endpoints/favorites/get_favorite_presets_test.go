package favorites

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	testSlotCount          = 15
	pcHeaderSize           = 0x300
	pcSlotCount            = 10
	pcSlotBlockSize        = 0x280010
	pcUserData10BlockSize  = 0x60010
	pcSlotsOffset          = int64(pcHeaderSize)
	pcSlotsSize            = int64(pcSlotCount) * pcSlotBlockSize
	pcUserData10Offset     = pcSlotsOffset + pcSlotsSize
	pcUserData10DataOffset = pcUserData10Offset + 0x10

	ps4HeaderSize           = 0x70
	ps4SlotCount            = 10
	ps4SlotSize             = 0x280000
	ps4UserData10Size       = 0x60000
	ps4SlotsOffset          = int64(ps4HeaderSize)
	ps4SlotsSize            = int64(ps4SlotCount) * ps4SlotSize
	ps4UserData10Offset     = ps4SlotsOffset + ps4SlotsSize
	ps4UserData10DataOffset = ps4UserData10Offset

	favoriteBaseOffset  = 0x154
	favoriteSlotSize    = 0x130
	favoriteMagicOffset = 0x18
)

func writeEndpointFavoritesFixture(t *testing.T, platform string, activeSlots map[int]bool) string {
	t.Helper()

	var data []byte
	var base int64
	switch platform {
	case "pc":
		data = make([]byte, pcUserData10Offset+pcUserData10BlockSize)
		copy(data, []byte("BND4"))
		binary.LittleEndian.PutUint32(data[0x0C:], 12)
		base = pcUserData10DataOffset
	case "ps4":
		data = make([]byte, ps4UserData10Offset+ps4UserData10Size)
		copy(data, []byte{0xCB, 0x01, 0x9C, 0x2C})
		for entry := 0; entry < 12; entry++ {
			off := 0x10 + entry*8
			binary.LittleEndian.PutUint32(data[off:], uint32(7+entry))
			binary.LittleEndian.PutUint32(data[off+4:], 0x7F7F7F7F)
		}
		base = ps4UserData10DataOffset
	default:
		t.Fatalf("unknown platform %q", platform)
	}

	for slot := 0; slot < testSlotCount; slot++ {
		if activeSlots != nil && activeSlots[slot] {
			slotAt := base + favoriteBaseOffset + int64(slot)*favoriteSlotSize
			binary.LittleEndian.PutUint16(data[slotAt:], 0xFACE)
			binary.LittleEndian.PutUint32(data[slotAt+4:], 0x11D0)
			copy(data[slotAt+favoriteMagicOffset:], []byte("FACE"))
			binary.LittleEndian.PutUint32(data[slotAt+0x1C:], 4)
			binary.LittleEndian.PutUint32(data[slotAt+0x20:], 0x120)
		}
	}

	path := filepath.Join(t.TempDir(), "endpoint_favorites.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func ptr(v int) *int {
	return &v
}

func TestGetFavoritePresetsEndpointReturnsAllSlots(t *testing.T) {
	for _, platform := range []string{"pc", "ps4"} {
		t.Run(platform, func(t *testing.T) {
			active := map[int]bool{
				0:  true,
				2:  true,
				14: true,
			}
			path := writeEndpointFavoritesFixture(t, platform, active)
			engine := saveengine.New()
			session, err := engine.LoadSave(path, platform, "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			res, err := GetFavoritePresets(engine, session.SaveSessionID, nil)
			if err != nil {
				t.Fatalf("GetFavoritePresets: %v", err)
			}

			if res.SaveSessionID != session.SaveSessionID {
				t.Fatalf("SaveSessionID = %q, want %q", res.SaveSessionID, session.SaveSessionID)
			}
			if len(res.Presets) != testSlotCount {
				t.Fatalf("len(Presets) = %d, want %d", len(res.Presets), testSlotCount)
			}

			for i := 0; i < testSlotCount; i++ {
				if res.Presets[i].FavoriteSlotID != i {
					t.Errorf("Presets[%d].FavoriteSlotID = %d, want %d", i, res.Presets[i].FavoriteSlotID, i)
				}
				if res.Presets[i].Active != active[i] {
					t.Errorf("Presets[%d].Active = %v, want %v", i, res.Presets[i].Active, active[i])
				}
			}
		})
	}
}

func TestGetFavoritePresetsEndpointFiltersSingleSlot(t *testing.T) {
	path := writeEndpointFavoritesFixture(t, "pc", map[int]bool{4: true})
	engine := saveengine.New()
	session, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	res, err := GetFavoritePresets(engine, session.SaveSessionID, ptr(4))
	if err != nil {
		t.Fatalf("GetFavoritePresets(4): %v", err)
	}
	if len(res.Presets) != 1 {
		t.Fatalf("len(Presets) = %d, want 1", len(res.Presets))
	}
	if res.Presets[0].FavoriteSlotID != 4 || !res.Presets[0].Active {
		t.Errorf("Presets[0] = %+v, want FavoriteSlotID: 4, Active: true", res.Presets[0])
	}
}

func TestGetFavoritePresetsEndpointRejectsOutOfRangeSlots(t *testing.T) {
	path := writeEndpointFavoritesFixture(t, "pc", nil)
	engine := saveengine.New()
	session, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	for _, invalid := range []int{-1, 15, 99} {
		if _, err := GetFavoritePresets(engine, session.SaveSessionID, ptr(invalid)); err == nil {
			t.Fatalf("GetFavoritePresets(%d) succeeded, want error", invalid)
		}
	}
}

func TestGetFavoritePresetsEndpointRejectsMissingEngine(t *testing.T) {
	if _, err := GetFavoritePresets(nil, "any-session", nil); err == nil {
		t.Fatal("missing engine was accepted, want error")
	}
}

func TestGetFavoritePresetsEndpointDelegatesSessionValidation(t *testing.T) {
	engine := saveengine.New()
	for _, id := range []string{"", "unknown-session"} {
		if _, err := GetFavoritePresets(engine, id, nil); err == nil {
			t.Fatalf("GetFavoritePresets(%q) succeeded, want error", id)
		}
	}
}
