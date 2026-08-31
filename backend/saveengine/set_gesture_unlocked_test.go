package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const setGestureTestSlot = 3

func setGestureTestRecords(values ...uint32) []uint32 {
	records := make([]uint32, gestureTestSlotCount)
	for index := range records {
		records[index] = gestureTestEmptySentinel
	}
	copy(records, values)
	return records
}

func loadSetGestureSession(
	t *testing.T,
	platform Platform,
	records []uint32,
) (*Engine, string) {
	t.Helper()
	content := gestureTestActiveFixture(platform, setGestureTestSlot, 0xB000, 0)
	content.records = records
	engine := New()
	loaded, err := engine.LoadSave(writeGestureFixture(t, content), string(platform), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func readSetGestureRecords(t *testing.T, engine *Engine, sessionID string) []uint32 {
	t.Helper()
	result, err := engine.GetGestures(sessionID, setGestureTestSlot)
	if err != nil {
		t.Fatalf("GetGestures: %v", err)
	}
	return result.Slots
}

func assertRejectedGestureMutationUnchanged(
	t *testing.T,
	engine *Engine,
	sessionID string,
	before []byte,
) {
	t.Helper()
	if !bytes.Equal(before, engine.sessions[sessionID].snapshot.data) {
		t.Error("rejected mutation changed the private snapshot")
	}
	if revision := engine.sessions[sessionID].session.revisionString(); revision != "0" {
		t.Errorf("revision after rejection = %q, want 0", revision)
	}
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejection = %+v, want clean", info)
	}
}

func TestSetGestureUnlockedAssignsNativeRecordsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, sessionID := loadSetGestureSession(
				t, platform, setGestureTestRecords(1, 228, 4242, 0))

			result, err := engine.SetGestureUnlocked(
				sessionID, setGestureTestSlot, 229, true, "0")
			if err != nil {
				t.Fatalf("unlock: %v", err)
			}
			if result.SaveSessionID != sessionID || result.SaveRevision != "1" ||
				result.CharacterID != setGestureTestSlot || !result.Unlocked {
				t.Errorf("unlock result = %+v", result)
			}
			records := readSetGestureRecords(t, engine, sessionID)
			if records[1] != 229 {
				t.Errorf("unlocked record = %d, want 229", records[1])
			}
			if records[0] != 1 || records[2] != 4242 || records[3] != 0 {
				t.Errorf("unrelated records changed: %v", records[:4])
			}

			result, err = engine.SetGestureUnlocked(
				sessionID, setGestureTestSlot, 229, false, "1")
			if err != nil {
				t.Fatalf("lock: %v", err)
			}
			if result.SaveRevision != "2" || result.Unlocked {
				t.Errorf("lock result = %+v", result)
			}
			if got := readSetGestureRecords(t, engine, sessionID)[1]; got != 228 {
				t.Errorf("locked record = %d, want 228", got)
			}
		})
	}
}

func TestSetGestureUnlockedPreservesUnrelatedRecordsAndHandlesCapacity(t *testing.T) {
	t.Run("uses first sentinel", func(t *testing.T) {
		engine, sessionID := loadSetGestureSession(
			t, PlatformPC, setGestureTestRecords(1, 4242, 0))
		if _, err := engine.SetGestureUnlocked(
			sessionID, setGestureTestSlot, 229, true, "0"); err != nil {
			t.Fatalf("SetGestureUnlocked: %v", err)
		}
		records := readSetGestureRecords(t, engine, sessionID)
		if records[3] != 229 {
			t.Errorf("first sentinel = %d, want 229", records[3])
		}
		if records[0] != 1 || records[1] != 4242 || records[2] != 0 {
			t.Errorf("unrelated records changed: %v", records[:4])
		}
	})

	t.Run("locks every duplicate", func(t *testing.T) {
		engine, sessionID := loadSetGestureSession(
			t, PlatformPC, setGestureTestRecords(229, 4242, 229, 0))
		if _, err := engine.SetGestureUnlocked(
			sessionID, setGestureTestSlot, 229, false, "0"); err != nil {
			t.Fatalf("SetGestureUnlocked: %v", err)
		}
		records := readSetGestureRecords(t, engine, sessionID)
		if records[0] != 228 || records[2] != 228 {
			t.Errorf("duplicate records = %d, %d, want 228, 228", records[0], records[2])
		}
		if records[1] != 4242 || records[3] != 0 {
			t.Errorf("unrelated records changed: %v", records[:4])
		}
	})

	t.Run("idempotent assignment advances revision", func(t *testing.T) {
		engine, sessionID := loadSetGestureSession(
			t, PlatformPC, setGestureTestRecords(229))
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)
		result, err := engine.SetGestureUnlocked(
			sessionID, setGestureTestSlot, 229, true, "0")
		if err != nil {
			t.Fatalf("SetGestureUnlocked: %v", err)
		}
		if result.SaveRevision != "1" {
			t.Errorf("revision = %q, want 1", result.SaveRevision)
		}
		if !bytes.Equal(before, engine.sessions[sessionID].snapshot.data) {
			t.Error("idempotent assignment changed the snapshot")
		}
	})

	t.Run("rejects a full block", func(t *testing.T) {
		records := make([]uint32, gestureTestSlotCount)
		for index := range records {
			records[index] = uint32(1001 + index*2)
		}
		engine, sessionID := loadSetGestureSession(t, PlatformPC, records)
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)
		_, err := engine.SetGestureUnlocked(
			sessionID, setGestureTestSlot, 229, true, "0")
		if err == nil || err.Error() != "GestureGameData has no slot available for gesture 229" {
			t.Fatalf("error = %v", err)
		}
		assertRejectedGestureMutationUnchanged(t, engine, sessionID, before)
	})
}

func TestSetGestureUnlockedProtectsStartingGestures(t *testing.T) {
	protected := []uint32{1, 13, 15, 41, 43, 45, 47, 49, 101, 141, 161, 185}
	for _, slotID := range protected {
		t.Run(strconv.FormatUint(uint64(slotID), 10), func(t *testing.T) {
			engine, sessionID := loadSetGestureSession(
				t, PlatformPC, setGestureTestRecords(slotID, slotID+1000, 0))
			before := bytes.Clone(engine.sessions[sessionID].snapshot.data)
			_, err := engine.SetGestureUnlocked(
				sessionID, setGestureTestSlot, slotID, false, "0")
			if err == nil || !strings.Contains(err.Error(), "protected starting gesture") {
				t.Fatalf("error = %v", err)
			}
			assertRejectedGestureMutationUnchanged(t, engine, sessionID, before)
		})
	}

	t.Run("missing Bow preserves zero", func(t *testing.T) {
		engine, sessionID := loadSetGestureSession(
			t, PlatformPC, setGestureTestRecords(0, gestureTestEmptySentinel, 4242))
		if _, err := engine.SetGestureUnlocked(
			sessionID, setGestureTestSlot, 1, true, "0"); err != nil {
			t.Fatalf("SetGestureUnlocked: %v", err)
		}
		records := readSetGestureRecords(t, engine, sessionID)
		if records[0] != 0 || records[1] != 1 || records[2] != 4242 {
			t.Errorf("records = %v, want [0 1 4242]", records[:3])
		}
	})
}

func TestSetGestureUnlockedAppliesRingOfMiquellaAlias(t *testing.T) {
	t.Run("pre-order grant makes unlock a no-op", func(t *testing.T) {
		engine, sessionID := loadSetGestureSession(
			t, PlatformPC, setGestureTestRecords(227, 232, 4242))
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)
		result, err := engine.SetGestureUnlocked(
			sessionID, setGestureTestSlot, 233, true, "0")
		if err != nil {
			t.Fatalf("SetGestureUnlocked: %v", err)
		}
		if result.SaveRevision != "1" || !result.Unlocked {
			t.Errorf("result = %+v", result)
		}
		if !bytes.Equal(before, engine.sessions[sessionID].snapshot.data) {
			t.Error("unlock changed a save that already carries slot 227")
		}
	})

	t.Run("unlocks only earned alias", func(t *testing.T) {
		engine, sessionID := loadSetGestureSession(
			t, PlatformPC, setGestureTestRecords(226, 232, 4242))
		if _, err := engine.SetGestureUnlocked(
			sessionID, setGestureTestSlot, 233, true, "0"); err != nil {
			t.Fatalf("SetGestureUnlocked: %v", err)
		}
		records := readSetGestureRecords(t, engine, sessionID)
		if records[0] != 226 || records[1] != 233 || records[2] != 4242 {
			t.Errorf("records = %v, want [226 233 4242]", records[:3])
		}
	})

	t.Run("pre-order grant cannot be locked", func(t *testing.T) {
		engine, sessionID := loadSetGestureSession(
			t, PlatformPS4, setGestureTestRecords(227, 233, 4242))
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)
		_, err := engine.SetGestureUnlocked(
			sessionID, setGestureTestSlot, 233, false, "0")
		if err == nil || err.Error() !=
			"Ring of Miquella is granted by pre-order slot 227 and cannot be locked" {
			t.Fatalf("error = %v", err)
		}
		assertRejectedGestureMutationUnchanged(t, engine, sessionID, before)
	})

	t.Run("locks earned alias without pre-order grant", func(t *testing.T) {
		engine, sessionID := loadSetGestureSession(
			t, PlatformPS4, setGestureTestRecords(233, 4242, 233))
		if _, err := engine.SetGestureUnlocked(
			sessionID, setGestureTestSlot, 233, false, "0"); err != nil {
			t.Fatalf("SetGestureUnlocked: %v", err)
		}
		records := readSetGestureRecords(t, engine, sessionID)
		if records[0] != 232 || records[1] != 4242 || records[2] != 232 {
			t.Errorf("records = %v, want [232 4242 232]", records[:3])
		}
	})
}

func TestSetGestureUnlockedRejectsInvalidRequestsWithoutMutation(t *testing.T) {
	engine, sessionID := loadSetGestureSession(
		t, PlatformPC, setGestureTestRecords(228, 4242))
	before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

	cases := map[string]struct {
		sessionID        string
		characterID      int
		slotID           uint32
		expectedRevision string
		want             string
	}{
		"empty session":   {"", setGestureTestSlot, 229, "0", "saveSessionID is required"},
		"unknown session": {"missing", setGestureTestSlot, 229, "0", `unknown save session "missing"`},
		"character below": {sessionID, -1, 229, "0", "characterID -1 is outside the range 0..9"},
		"character above": {sessionID, 10, 229, "0", "characterID 10 is outside the range 0..9"},
		"noncanonical revision": {sessionID, setGestureTestSlot, 229, "00",
			`expectedRevision must be a canonical decimal saveRevision; got "00"`},
		"stale revision": {sessionID, setGestureTestSlot, 229, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`},
		"even slot": {sessionID, setGestureTestSlot, 228, "0",
			"gesture slot ID 228 is not a supported canonical odd slot ID"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := engine.SetGestureUnlocked(
				testCase.sessionID,
				testCase.characterID,
				testCase.slotID,
				true,
				testCase.expectedRevision,
			)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
	assertRejectedGestureMutationUnchanged(t, engine, sessionID, before)
}

func TestSetGestureUnlockedRejectsInactiveAndMalformedSlots(t *testing.T) {
	cases := map[string]struct {
		content gestureTestFixture
		want    string
	}{
		"inactive slot": {
			content: gestureTestFixture{
				platform: PlatformPC, slot: setGestureTestSlot, noAnchor: true,
			},
			want: "character 3 is not active",
		},
		"missing anchor": {
			content: gestureTestFixture{
				platform: PlatformPS4, slot: setGestureTestSlot,
				flag: userData10ActiveFlagValue, noAnchor: true,
			},
			want: "character 3 carries no gesture anchor",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeGestureFixture(t, testCase.content), string(testCase.content.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)
			_, err = engine.SetGestureUnlocked(
				loaded.SaveSessionID, setGestureTestSlot, 229, true, "0")
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			assertRejectedGestureMutationUnchanged(
				t, engine, loaded.SaveSessionID, before)
		})
	}
}

func writeFullGestureSaveFixture(t *testing.T, platform Platform, records []uint32) string {
	t.Helper()
	// The confirmed PlayerGameData marker terminates the 5120 eight-byte GaItem
	// records that start at slot offset 0x20. The gesture locator deliberately
	// follows that same marker, as a native save does.
	content := gestureTestActiveFixture(
		platform, setGestureTestSlot, 0x20+5120*8, 0)
	content.records = records
	path := writeGestureFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	slotBase := int64(gestureTestPCSlotDataBase) + int64(content.slot)*gestureTestPCSlotStride
	if platform == PlatformPS4 {
		slotBase = gestureTestPS4SlotDataBase + int64(content.slot)*gestureTestPS4SlotStride
	}
	binary.LittleEndian.PutUint32(data[slotBase:], 0x6E)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestSetGestureUnlockedPersistsAndReloadsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			source := writeFullGestureSaveFixture(
				t, platform, setGestureTestRecords(228, 4242, 0))
			sourceBefore, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}

			engine := New()
			loaded, err := engine.LoadSave(source, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			if _, err := engine.SetGestureUnlocked(
				loaded.SaveSessionID, setGestureTestSlot, 229, true, "0"); err != nil {
				t.Fatalf("SetGestureUnlocked: %v", err)
			}

			sourceAfter, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read source after mutation: %v", err)
			}
			if !bytes.Equal(sourceBefore, sourceAfter) {
				t.Error("in-memory mutation changed the source file")
			}

			target := filepath.Join(t.TempDir(), "written.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}

			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave target: %v", err)
			}
			records := readSetGestureRecords(t, reloadedEngine, reloaded.SaveSessionID)
			if records[0] != 229 || records[1] != 4242 || records[2] != 0 {
				t.Errorf("reloaded records = %v, want [229 4242 0]", records[:3])
			}
		})
	}
}
