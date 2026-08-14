package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSetCookbookUnlockedMutatesEventFlagBitAndAdvancesRevision(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)

			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			snapshotBefore := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

			// 67010 is clear in fixture; set it to unlocked
			result, err := engine.SetCookbookUnlocked(
				loaded.SaveSessionID, content.slot, 67010, true, "0")
			if err != nil {
				t.Fatalf("SetCookbookUnlocked: %v", err)
			}

			want := SetCookbookUnlockedResult{
				SaveSessionID: loaded.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   content.slot,
				Unlocked:      true,
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			// Re-read event flags to confirm 67010 is now set, while 67000 remains set
			flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, []uint32{67000, 67010})
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}
			if !flags.Flags[67010] {
				t.Error("flag 67010 is false, want true")
			}
			if !flags.Flags[67000] {
				t.Error("flag 67000 is false, want true")
			}

			snapshotAfter := engine.sessions[loaded.SaveSessionID].snapshot.data
			changed := make([]int, 0, 1)
			for index := range snapshotBefore {
				if snapshotBefore[index] != snapshotAfter[index] {
					changed = append(changed, index)
				}
			}
			if len(changed) != 1 {
				t.Fatalf("changed snapshot bytes = %v, want exactly one", changed)
			}
			_, bit := eventFlagTestPosition(t, 67010)
			if delta := snapshotBefore[changed[0]] ^ snapshotAfter[changed[0]]; delta != 1<<bit {
				t.Errorf("changed bit mask = 0x%02X, want 0x%02X", delta, byte(1<<bit))
			}

			info, err := engine.GetSessionInfo(loaded.SaveSessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if !info.UnsavedChanges {
				t.Error("UnsavedChanges = false, want true after mutation")
			}
		})
	}
}

func TestSetCookbookUnlockedClearsUnlockedFlag(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// 67000 is set in fixture; set unlocked=false to clear it
	result, err := engine.SetCookbookUnlocked(
		loaded.SaveSessionID, content.slot, 67000, false, "0")
	if err != nil {
		t.Fatalf("SetCookbookUnlocked: %v", err)
	}

	if result.SaveRevision != "1" || result.Unlocked != false {
		t.Errorf("result = %+v, want revision 1 and unlocked false", result)
	}

	flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, []uint32{67000, 67999})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if flags.Flags[67000] {
		t.Error("flag 67000 is true, want false")
	}
	if !flags.Flags[67999] {
		t.Error("flag 67999 is false, want true (adjacent set flag preserved)")
	}
}

func TestSetCookbookUnlockedPreservesAdjacentBitsInSameByte(t *testing.T) {
	// Set 67000 (bit 7 of first byte of block 67). 67001 is bit 6 of same byte.
	content := eventFlagTestContent(PlatformPC)
	content.set = []uint32{67000, 67001}

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// Clear 67000, keep 67001
	_, err = engine.SetCookbookUnlocked(
		loaded.SaveSessionID, content.slot, 67000, false, "0")
	if err != nil {
		t.Fatalf("SetCookbookUnlocked: %v", err)
	}

	flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, []uint32{67000, 67001})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if flags.Flags[67000] {
		t.Error("flag 67000 is true, want false")
	}
	if !flags.Flags[67001] {
		t.Error("flag 67001 is false, want true (adjacent bit in same byte preserved)")
	}
}

func TestSetCookbookUnlockedHandlesBlockBoundaries(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	content.set = nil

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	boundaries := []uint32{67000, 67999, 68000, 68999}

	revision := "0"
	for _, flag := range boundaries {
		res, err := engine.SetCookbookUnlocked(
			loaded.SaveSessionID, content.slot, flag, true, revision)
		if err != nil {
			t.Fatalf("SetCookbookUnlocked(%d): %v", flag, err)
		}
		revision = res.SaveRevision

		flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, []uint32{flag})
		if err != nil {
			t.Fatalf("GetEventFlags(%d): %v", flag, err)
		}
		if !flags.Flags[flag] {
			t.Errorf("flag %d is false, want true", flag)
		}
	}
}

func TestSetCookbookUnlockedIsIdempotentRegardingBitState(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// 67000 is already set in fixture. Setting it to true again should succeed and advance revision.
	res1, err := engine.SetCookbookUnlocked(
		loaded.SaveSessionID, content.slot, 67000, true, "0")
	if err != nil {
		t.Fatalf("SetCookbookUnlocked: %v", err)
	}
	if res1.SaveRevision != "1" {
		t.Errorf("revision = %q, want 1", res1.SaveRevision)
	}

	// Setting it to true again under revision 1 advances to revision 2.
	res2, err := engine.SetCookbookUnlocked(
		loaded.SaveSessionID, content.slot, 67000, true, "1")
	if err != nil {
		t.Fatalf("SetCookbookUnlocked: %v", err)
	}
	if res2.SaveRevision != "2" {
		t.Errorf("revision = %q, want 2", res2.SaveRevision)
	}
}

func TestSetCookbookUnlockedRejectsInactiveSlot(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	content.flag = 0 // Inactive slot

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	_, err = engine.SetCookbookUnlocked(
		loaded.SaveSessionID, content.slot, 67000, true, "0")
	if err == nil {
		t.Fatal("SetCookbookUnlocked accepted an inactive slot")
	}
	want := "character 3 is not active"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}

	// Prove revision and dirty state are untouched
	info, _ := engine.GetSessionInfo(loaded.SaveSessionID)
	if info.UnsavedChanges {
		t.Error("UnsavedChanges = true, want false after failed mutation")
	}
}

func TestSetCookbookUnlockedRejectsInvalidRevisionAndSession(t *testing.T) {
	engine := New()
	path := writeEventFlagFixture(t, eventFlagTestContent(PlatformPC))
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	cases := map[string]struct {
		sessionID string
		slot      int
		rev       string
		want      string
	}{
		"non-canonical revision": {
			loaded.SaveSessionID, 3, "01",
			`expectedRevision must be a canonical decimal saveRevision; got "01"`,
		},
		"mismatched revision": {
			loaded.SaveSessionID, 3, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`,
		},
		"unknown session": {
			"unknown-session", 3, "0",
			`unknown save session "unknown-session"`,
		},
		"characterID -1": {
			loaded.SaveSessionID, -1, "0",
			"characterID -1 is outside the range 0..9",
		},
		"characterID 10": {
			loaded.SaveSessionID, 10, "0",
			"characterID 10 is outside the range 0..9",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := engine.SetCookbookUnlocked(tc.sessionID, tc.slot, 67000, true, tc.rev)
			if err == nil {
				t.Fatalf("accepted %s", name)
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err.Error(), tc.want)
			}
		})
	}

	// Verify session remains unchanged
	info, _ := engine.GetSessionInfo(loaded.SaveSessionID)
	if info.UnsavedChanges {
		t.Error("UnsavedChanges = true, want false after rejected calls")
	}
}

func TestSetCookbookUnlockedRejectsUnsupportedFlagsAndCorruptData(t *testing.T) {
	engine := New()

	load := func(c eventFlagTestFixture) string {
		l, err := engine.LoadSave(writeEventFlagFixture(t, c), "")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return l.SaveSessionID
	}

	normalID := load(eventFlagTestContent(PlatformPC))

	missingAnchor := eventFlagTestContent(PlatformPC)
	missingAnchor.noAnchor = true
	corruptRange := eventFlagTestContent(PlatformPC)
	corruptRange.anchorAt = 0x100000
	corruptProjectiles := eventFlagTestContent(PlatformPC)
	corruptProjectiles.projectiles = 200001
	corruptRegions := eventFlagTestContent(PlatformPC)
	corruptRegions.regions = 20001
	corruptTutorial := eventFlagTestContent(PlatformPC)
	corruptTutorial.tutorialSize = 0x10001
	corruptMenu := eventFlagTestContent(PlatformPC)
	corruptMenu.menuSize = 0x10001

	cases := map[string]struct {
		sessionID string
		flag      uint32
		want      string
	}{
		"block 66": {
			normalID, 66999,
			"event flag 66999 lies in block 66, which this reader does not support",
		},
		"whetblade block 60": {
			normalID, 60130,
			"event flag 60130 lies in block 60, which this reader does not support",
		},
		"block 69": {
			normalID, 69000,
			"event flag 69000 lies in block 69, which this reader does not support",
		},
		"missing anchor": {
			load(missingAnchor), 67000,
			"character 3 carries no event flag anchor",
		},
		"event flags outside slot": {
			load(corruptRange), 67000,
			"event flags of character 3 do not fit into their slot",
		},
		"projectile count above limit": {
			load(corruptProjectiles), 67000,
			"character 3 declares a projectile count of 200001, want at most 200000",
		},
		"region count above limit": {
			load(corruptRegions), 67000,
			"character 3 declares a region count of 20001, want at most 20000",
		},
		"tutorial size above limit": {
			load(corruptTutorial), 67000,
			"character 3 declares a tutorial size of 65537, want at most 65536",
		},
		"menu profile size above limit": {
			load(corruptMenu), 67000,
			"character 3 declares a menu profile size of 65537, want at most 65536",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			snapshotBefore := bytes.Clone(engine.sessions[tc.sessionID].snapshot.data)
			_, err := engine.SetCookbookUnlocked(tc.sessionID, 3, tc.flag, true, "0")
			if err == nil {
				t.Fatalf("accepted %s", name)
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err.Error(), tc.want)
			}
			if !bytes.Equal(snapshotBefore, engine.sessions[tc.sessionID].snapshot.data) {
				t.Error("rejected mutation changed the private snapshot")
			}
			info, err := engine.GetSessionInfo(tc.sessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if revision := engine.sessions[tc.sessionID].session.revisionString(); revision != "0" {
				t.Errorf("revision after rejection = %q, want 0", revision)
			}
			if info.UnsavedChanges {
				t.Errorf("session after rejection = %+v, want clean", info)
			}
		})
	}
}

func writeFullCookbookSaveFixture(t *testing.T, content eventFlagTestFixture) string {
	t.Helper()
	content.anchorAt = 0xB000
	path := writeEventFlagFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	slotBase := eventFlagTestPCSlotDataBase + int64(content.slot)*eventFlagTestPCSlotStride
	if content.platform == PlatformPS4 {
		slotBase = eventFlagTestPS4SlotDataBase + int64(content.slot)*eventFlagTestPS4SlotStride
	}
	binary.LittleEndian.PutUint32(data[slotBase:], 0x6E)
	copy(data[slotBase+0x20+5120*8:], gaItemAnchor)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestSetCookbookUnlockedPersistsToTargetFileOnWriteSave(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			srcPath := writeFullCookbookSaveFixture(t, eventFlagTestContent(platform))

			engine := New()
			loaded, err := engine.LoadSave(srcPath, string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			_, err = engine.SetCookbookUnlocked(loaded.SaveSessionID, 3, 67010, true, "0")
			if err != nil {
				t.Fatalf("SetCookbookUnlocked: %v", err)
			}

			targetPath := filepath.Join(t.TempDir(), "written.sl2")
			writeRes, err := engine.WriteSave(loaded.SaveSessionID, "1", targetPath)
			if err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			if writeRes.SaveRevision != "2" {
				t.Errorf("WriteSave revision = %q, want 2", writeRes.SaveRevision)
			}

			srcBytes, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			targetBytes, err := os.ReadFile(targetPath)
			if err != nil {
				t.Fatalf("read target: %v", err)
			}
			if bytes.Equal(srcBytes, targetBytes) {
				t.Error("written file is byte-identical to source file, want it modified")
			}

			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(targetPath, string(platform))
			if err != nil {
				t.Fatalf("LoadSave target: %v", err)
			}
			flags, err := reloadedEngine.GetEventFlags(reloaded.SaveSessionID, 3, []uint32{67010})
			if err != nil {
				t.Fatalf("GetEventFlags target: %v", err)
			}
			if !flags.Flags[67010] {
				t.Error("flag 67010 in reloaded save is false, want true")
			}
		})
	}
}
