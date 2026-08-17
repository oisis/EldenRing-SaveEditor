package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	testFavoriteSlotCount   = 15
	testFavoriteSlotSize    = 0x130
	testFavoriteBaseOffset  = 0x154
	testFavoriteMagicOffset = 0x18
)

type favoritesFixture struct {
	activeSlots map[int]bool
}

func writeFavoritesFixture(t *testing.T, platform Platform, content favoritesFixture) string {
	t.Helper()

	var data []byte
	var base int64
	switch platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		base = pcUserData10DataOffset
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		base = ps4UserData10DataOffset
	default:
		t.Fatalf("unknown platform %q", platform)
	}

	for slot := 0; slot < testFavoriteSlotCount; slot++ {
		if content.activeSlots != nil && content.activeSlots[slot] {
			slotAt := base + testFavoriteBaseOffset + int64(slot)*testFavoriteSlotSize
			binary.LittleEndian.PutUint16(data[slotAt:], 0xFACE)
			binary.LittleEndian.PutUint32(data[slotAt+4:], 0x11D0)
			copy(data[slotAt+testFavoriteMagicOffset:], "FACE")
			binary.LittleEndian.PutUint32(data[slotAt+0x1C:], 4)
			binary.LittleEndian.PutUint32(data[slotAt+0x20:], 0x120)
		}
	}

	path := filepath.Join(t.TempDir(), "favorites.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func ptr(v int) *int {
	return &v
}

func TestGetFavoritePresetsReads15SlotsOnPCAndPS4(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := favoritesFixture{
				activeSlots: map[int]bool{
					0:  true,
					3:  true,
					14: true,
				},
			}
			path := writeFavoritesFixture(t, platform, content)
			engine := New()
			session, err := engine.LoadSave(path, string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			res, err := engine.GetFavoritePresets(session.SaveSessionID, nil)
			if err != nil {
				t.Fatalf("GetFavoritePresets(nil): %v", err)
			}

			if res.SaveSessionID != session.SaveSessionID {
				t.Fatalf("SaveSessionID = %q, want %q", res.SaveSessionID, session.SaveSessionID)
			}
			if len(res.Presets) != testFavoriteSlotCount {
				t.Fatalf("len(Presets) = %d, want %d", len(res.Presets), testFavoriteSlotCount)
			}

			for i := 0; i < testFavoriteSlotCount; i++ {
				if res.Presets[i].FavoriteSlotID != i {
					t.Errorf("Presets[%d].FavoriteSlotID = %d, want %d", i, res.Presets[i].FavoriteSlotID, i)
				}
				expectedActive := content.activeSlots[i]
				if res.Presets[i].Active != expectedActive {
					t.Errorf("Presets[%d].Active = %v, want %v", i, res.Presets[i].Active, expectedActive)
				}
			}
		})
	}
}

func TestGetFavoritePresetsFiltersSingleSlot(t *testing.T) {
	content := favoritesFixture{
		activeSlots: map[int]bool{
			5: true,
		},
	}
	path := writeFavoritesFixture(t, PlatformPC, content)
	engine := New()
	session, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// Slot 5: active
	res5, err := engine.GetFavoritePresets(session.SaveSessionID, ptr(5))
	if err != nil {
		t.Fatalf("GetFavoritePresets(5): %v", err)
	}
	if len(res5.Presets) != 1 {
		t.Fatalf("len(Presets) = %d, want 1", len(res5.Presets))
	}
	if res5.Presets[0].FavoriteSlotID != 5 || !res5.Presets[0].Active {
		t.Errorf("Presets[0] = %+v, want FavoriteSlotID: 5, Active: true", res5.Presets[0])
	}

	// Slot 2: empty
	res2, err := engine.GetFavoritePresets(session.SaveSessionID, ptr(2))
	if err != nil {
		t.Fatalf("GetFavoritePresets(2): %v", err)
	}
	if len(res2.Presets) != 1 {
		t.Fatalf("len(Presets) = %d, want 1", len(res2.Presets))
	}
	if res2.Presets[0].FavoriteSlotID != 2 || res2.Presets[0].Active {
		t.Errorf("Presets[0] = %+v, want FavoriteSlotID: 2, Active: false", res2.Presets[0])
	}
}

func TestGetFavoritePresetsRejectsOutOfRangeSlotID(t *testing.T) {
	path := writeFavoritesFixture(t, PlatformPC, favoritesFixture{})
	engine := New()
	session, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	for _, invalidSlot := range []int{-1, 15, 100} {
		_, err := engine.GetFavoritePresets(session.SaveSessionID, ptr(invalidSlot))
		if err == nil {
			t.Fatalf("expected error for favoriteSlotID %d, got nil", invalidSlot)
		}
	}
}

func TestGetFavoritePresetsRejectsEmptyAndUnknownSession(t *testing.T) {
	engine := New()

	if _, err := engine.GetFavoritePresets("", nil); err == nil {
		t.Fatal("expected error for empty saveSessionID, got nil")
	}

	if _, err := engine.GetFavoritePresets("unknown-session", nil); err == nil {
		t.Fatal("expected error for unknown saveSessionID, got nil")
	}
}

func TestGetFavoritePresetsTruncatedContainerFailsClosed(t *testing.T) {
	engine := New()
	session, err := engine.LoadSave(writeFavoritesFixture(t, PlatformPC, favoritesFixture{}), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// Truncate snapshot to verify bounds check in GetFavoritePresets
	engine.mutex.Lock()
	loaded := engine.sessions[session.SaveSessionID]
	loaded.snapshot.data = loaded.snapshot.data[:pcUserData10DataOffset+testFavoriteBaseOffset+100]
	engine.mutex.Unlock()

	if _, err := engine.GetFavoritePresets(session.SaveSessionID, nil); err == nil {
		t.Fatal("expected error for truncated snapshot reading all slots, got nil")
	}

	if _, err := engine.GetFavoritePresets(session.SaveSessionID, ptr(1)); err == nil {
		t.Fatal("expected error for truncated snapshot reading slot 1, got nil")
	}
}

func TestGetFavoritePresetsActiveDependsOnlyOnMagic(t *testing.T) {
	path := writeFavoritesFixture(t, PlatformPC, favoritesFixture{})
	engine := New()
	session, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	loaded := engine.sessions[session.SaveSessionID]
	// Slot 0: exact "FACE" magic, but arbitrary/non-canonical alignment and inner size -> must be active: true
	slot0At := pcUserData10DataOffset + testFavoriteBaseOffset
	copy(loaded.snapshot.data[slot0At+testFavoriteMagicOffset:], "FACE")
	binary.LittleEndian.PutUint32(loaded.snapshot.data[slot0At+0x1C:], 8)
	binary.LittleEndian.PutUint32(loaded.snapshot.data[slot0At+0x20:], 0x999)

	// Slot 1: valid appearance alignment and inner size, but magic is not "FACE" -> must be active: false
	slot1At := pcUserData10DataOffset + testFavoriteBaseOffset + testFavoriteSlotSize
	copy(loaded.snapshot.data[slot1At+testFavoriteMagicOffset:], "NOPE")
	binary.LittleEndian.PutUint32(loaded.snapshot.data[slot1At+0x1C:], 4)
	binary.LittleEndian.PutUint32(loaded.snapshot.data[slot1At+0x20:], 0x120)
	engine.mutex.Unlock()

	res0, err := engine.GetFavoritePresets(session.SaveSessionID, ptr(0))
	if err != nil {
		t.Fatalf("GetFavoritePresets(0): %v", err)
	}
	if len(res0.Presets) != 1 || !res0.Presets[0].Active {
		t.Errorf("slot 0 with FACE magic = %+v, want active: true", res0.Presets)
	}

	res1, err := engine.GetFavoritePresets(session.SaveSessionID, ptr(1))
	if err != nil {
		t.Fatalf("GetFavoritePresets(1): %v", err)
	}
	if len(res1.Presets) != 1 || res1.Presets[0].Active {
		t.Errorf("slot 1 without FACE magic = %+v, want active: false", res1.Presets)
	}
}

func TestGetFavoritePresetsIsStrictlyReadOnly(t *testing.T) {
	content := favoritesFixture{
		activeSlots: map[int]bool{
			0: true,
			1: false,
		},
	}
	path := writeFavoritesFixture(t, PlatformPC, content)
	engine := New()
	session, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	loaded := engine.sessions[session.SaveSessionID]
	snapshotBefore := append([]byte(nil), loaded.snapshot.data...)
	revisionBefore := loaded.session.revisionString()
	dirtyBefore := loaded.session.dirty
	undoBefore := loaded.session.undo
	ownedByIDLenBefore := len(loaded.session.ownedByID)
	ownedByLocatorLenBefore := len(loaded.session.ownedByLocator)
	engine.mutex.Unlock()

	_, err = engine.GetFavoritePresets(session.SaveSessionID, nil)
	if err != nil {
		t.Fatalf("GetFavoritePresets: %v", err)
	}

	engine.mutex.Lock()
	loaded = engine.sessions[session.SaveSessionID]
	snapshotAfter := loaded.snapshot.data
	revisionAfter := loaded.session.revisionString()
	dirtyAfter := loaded.session.dirty
	undoAfter := loaded.session.undo
	ownedByIDLenAfter := len(loaded.session.ownedByID)
	ownedByLocatorLenAfter := len(loaded.session.ownedByLocator)
	engine.mutex.Unlock()

	if !bytes.Equal(snapshotBefore, snapshotAfter) {
		t.Fatal("snapshot data was mutated by GetFavoritePresets")
	}
	if revisionBefore != revisionAfter {
		t.Fatalf("revision changed from %q to %q", revisionBefore, revisionAfter)
	}
	if dirtyBefore != dirtyAfter {
		t.Fatalf("dirty flag changed from %v to %v", dirtyBefore, dirtyAfter)
	}
	if !reflect.DeepEqual(undoBefore, undoAfter) {
		t.Fatalf("undo state changed from %v to %v", undoBefore, undoAfter)
	}
	if ownedByIDLenBefore != ownedByIDLenAfter || ownedByLocatorLenBefore != ownedByLocatorLenAfter {
		t.Fatal("ownedItems registry was mutated")
	}
}
