package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

const (
	testDeleteFavSlotCount   = 15
	testDeleteFavSlotSize    = 0x130
	testDeleteFavBaseOffset  = 0x154
	testDeleteFavMagicOffset = 0x18
)

func writeDeleteFavoritesFixture(t *testing.T, platform Platform, activeSlots map[int]bool) string {
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

	for slot := 0; slot < testDeleteFavSlotCount; slot++ {
		slotAt := base + testDeleteFavBaseOffset + int64(slot)*testDeleteFavSlotSize
		if activeSlots != nil && activeSlots[slot] {
			binary.LittleEndian.PutUint16(data[slotAt:], 0xFACE)
			binary.LittleEndian.PutUint32(data[slotAt+4:], 0x11D0)
			copy(data[slotAt+testDeleteFavMagicOffset:], "FACE")
			binary.LittleEndian.PutUint32(data[slotAt+0x1C:], 4)
			binary.LittleEndian.PutUint32(data[slotAt+0x20:], 0x120)
			// fill the rest of the 0x130 slot with distinctive non-zero bytes
			for b := 0x24; b < testDeleteFavSlotSize; b++ {
				data[slotAt+int64(b)] = byte(0xA0 + (slot+b)%32)
			}
		}
	}

	path := filepath.Join(t.TempDir(), "delete_favorites.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestDeleteFavoritePresetTablePCAndPS4(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			active := map[int]bool{
				0:  true,
				3:  true,
				4:  true,
				14: true,
			}
			path := writeDeleteFavoritesFixture(t, platform, active)
			engine := New()
			session, err := engine.LoadSave(path, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			engine.mutex.Lock()
			loaded := engine.sessions[session.SaveSessionID]
			snapshotBefore := append([]byte(nil), loaded.snapshot.data...)
			base := userData10Base(platform)
			engine.mutex.Unlock()

			// Delete slot 3 (active)
			targetSlot := 3
			res, err := engine.DeleteFavoritePreset(session.SaveSessionID, targetSlot, "0")
			if err != nil {
				t.Fatalf("DeleteFavoritePreset: %v", err)
			}

			if res.SaveSessionID != session.SaveSessionID {
				t.Errorf("SaveSessionID = %q, want %q", res.SaveSessionID, session.SaveSessionID)
			}
			if res.SaveRevision != "1" {
				t.Errorf("SaveRevision = %q, want \"1\"", res.SaveRevision)
			}
			if res.FavoriteSlotID != targetSlot {
				t.Errorf("FavoriteSlotID = %d, want %d", res.FavoriteSlotID, targetSlot)
			}

			engine.mutex.Lock()
			loaded = engine.sessions[session.SaveSessionID]
			snapshotAfter := loaded.snapshot.data
			dirtyAfter := loaded.session.dirty
			engine.mutex.Unlock()

			if !dirtyAfter {
				t.Error("expected dirty = true after successful delete")
			}

			targetOffset := base + testDeleteFavBaseOffset + int64(targetSlot)*testDeleteFavSlotSize

			// 1. Target slot must be all zeros in its full 0x130 range
			targetBytes := snapshotAfter[targetOffset : targetOffset+testDeleteFavSlotSize]
			expectedZeros := make([]byte, testDeleteFavSlotSize)
			if !bytes.Equal(targetBytes, expectedZeros) {
				t.Errorf("target slot %d was not fully zeroed: got %x", targetSlot, targetBytes[:32])
			}

			// 2. All bytes before target slot must be identical
			if !bytes.Equal(snapshotBefore[:targetOffset], snapshotAfter[:targetOffset]) {
				t.Error("bytes before target slot were modified")
			}

			// 3. All bytes after target slot must be identical
			afterTarget := targetOffset + testDeleteFavSlotSize
			if !bytes.Equal(snapshotBefore[afterTarget:], snapshotAfter[afterTarget:]) {
				t.Error("bytes after target slot were modified")
			}
		})
	}
}

func TestDeleteFavoritePresetInactiveSlotIsNoOpMutation(t *testing.T) {
	// Slot 1 is inactive (empty)
	active := map[int]bool{
		0: true,
		2: true,
	}
	path := writeDeleteFavoritesFixture(t, PlatformPC, active)
	engine := New()
	session, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	loaded := engine.sessions[session.SaveSessionID]
	snapshotBefore := append([]byte(nil), loaded.snapshot.data...)
	engine.mutex.Unlock()

	// Deleting inactive slot 1
	res, err := engine.DeleteFavoritePreset(session.SaveSessionID, 1, "0")
	if err != nil {
		t.Fatalf("DeleteFavoritePreset: %v", err)
	}

	if res.SaveRevision != "1" {
		t.Errorf("SaveRevision = %q, want \"1\"", res.SaveRevision)
	}
	if res.FavoriteSlotID != 1 {
		t.Errorf("FavoriteSlotID = %d, want 1", res.FavoriteSlotID)
	}

	engine.mutex.Lock()
	loaded = engine.sessions[session.SaveSessionID]
	snapshotAfter := loaded.snapshot.data
	dirtyAfter := loaded.session.dirty
	engine.mutex.Unlock()

	// Zero byte changes
	if !bytes.Equal(snapshotBefore, snapshotAfter) {
		t.Error("snapshot bytes were modified on inactive slot deletion")
	}
	// But dirty flag and revision are updated by global commitRevision
	if !dirtyAfter {
		t.Error("expected dirty = true after successful commitRevision of inactive slot")
	}
}

func TestDeleteFavoritePresetRejectionsDoNotMutateState(t *testing.T) {
	path := writeDeleteFavoritesFixture(t, PlatformPC, map[int]bool{0: true})
	engine := New()
	session, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	loaded := engine.sessions[session.SaveSessionID]
	snapshotBefore := append([]byte(nil), loaded.snapshot.data...)
	revisionBefore := loaded.session.revisionString()
	dirtyBefore := loaded.session.dirty
	engine.mutex.Unlock()

	testCases := []struct {
		name             string
		sessionID        string
		slotID           int
		expectedRevision string
	}{
		{"empty session", "", 0, "0"},
		{"unknown session", "unknown-session-id", 0, "0"},
		{"negative slot", session.SaveSessionID, -1, "0"},
		{"slot 15 out of range", session.SaveSessionID, 15, "0"},
		{"slot 99 out of range", session.SaveSessionID, 99, "0"},
		{"non-canonical revision negative", session.SaveSessionID, 0, "-1"},
		{"non-canonical revision leading zero", session.SaveSessionID, 0, "01"},
		{"non-canonical revision hex", session.SaveSessionID, 0, "0x0"},
		{"stale revision mismatch", session.SaveSessionID, 0, "1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := engine.DeleteFavoritePreset(tc.sessionID, tc.slotID, tc.expectedRevision)
			if err == nil {
				t.Fatalf("expected error for case %q, got nil", tc.name)
			}

			engine.mutex.Lock()
			loaded := engine.sessions[session.SaveSessionID]
			snapshotAfter := loaded.snapshot.data
			revisionAfter := loaded.session.revisionString()
			dirtyAfter := loaded.session.dirty
			engine.mutex.Unlock()

			if !bytes.Equal(snapshotBefore, snapshotAfter) {
				t.Errorf("snapshot was modified after rejection %q", tc.name)
			}
			if revisionAfter != revisionBefore {
				t.Errorf("revision changed from %q to %q after rejection %q", revisionBefore, revisionAfter, tc.name)
			}
			if dirtyAfter != dirtyBefore {
				t.Errorf("dirty changed from %v to %v after rejection %q", dirtyBefore, dirtyAfter, tc.name)
			}
		})
	}
}
