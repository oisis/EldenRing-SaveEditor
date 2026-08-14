package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	setBellBearingTestFlag       = uint32(11109710)
	setBellBearingTestGameID     = uint32(0x400022CE)
	setBellBearingTestHandle     = uint32(0xB00022CE)
	setBellBearingInventoryAt    = int64(505)
	setBellBearingCommonRecords  = int64(0xA80)
	setBellBearingStorageRecords = int64(0x780)
	setBellBearingRecordSize     = int64(12)
)

type setBellBearingTestLayout struct {
	inventoryCommon int64
	inventoryKey    int64
	storageCommon   int64
	storageKey      int64
}

func setBellBearingFixture(
	t *testing.T,
	platform Platform,
	setFlag bool,
	full bool,
) (string, eventFlagTestFixture, setBellBearingTestLayout) {
	t.Helper()
	content := eventFlagTestContent(platform)
	if setFlag {
		content.set = append(content.set, setBellBearingTestFlag)
	}
	if full {
		// One shared marker terminates the empty GaItem table and anchors all
		// later slot sections. Keeping one occurrence makes every reader choose
		// the same position.
		content.anchorAt = 0x20 + 5120*8
	}
	path := writeEventFlagFixture(t, content)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	slotBase := eventFlagTestPCSlotDataBase + int64(content.slot)*eventFlagTestPCSlotStride
	if platform == PlatformPS4 {
		slotBase = eventFlagTestPS4SlotDataBase + int64(content.slot)*eventFlagTestPS4SlotStride
	}
	if full {
		binary.LittleEndian.PutUint32(data[slotBase:], 0x6E)
	}
	anchor := slotBase + content.anchorAt
	storage := anchor + eventFlagTestProjectileCountAt + 4 +
		int64(content.projectiles)*8 + eventFlagTestBlocksBeforeStorage
	layout := setBellBearingTestLayout{
		inventoryCommon: anchor + setBellBearingInventoryAt,
		inventoryKey: anchor + setBellBearingInventoryAt +
			setBellBearingCommonRecords*setBellBearingRecordSize + 4,
		storageCommon: storage + 4,
		storageKey: storage + 4 +
			setBellBearingStorageRecords*setBellBearingRecordSize + 4,
	}

	putRecord := func(at int64, handle uint32, index uint32) {
		binary.LittleEndian.PutUint32(data[at:], handle)
		binary.LittleEndian.PutUint32(data[at+4:], 1)
		binary.LittleEndian.PutUint32(data[at+8:], index)
	}
	putRecord(layout.inventoryCommon+2*setBellBearingRecordSize, setBellBearingTestHandle, 2)
	putRecord(layout.inventoryKey+3*setBellBearingRecordSize, setBellBearingTestGameID, 3)
	putRecord(layout.storageCommon+4*setBellBearingRecordSize, setBellBearingTestGameID, 4)
	binary.LittleEndian.PutUint32(data[layout.inventoryCommon-4:], 1)
	binary.LittleEndian.PutUint32(data[layout.inventoryKey-4:], 1)
	binary.LittleEndian.PutUint32(data[layout.storageCommon-4:], 1)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path, content, layout
}

func TestSetBellBearingUnlockedConsumesConfirmedRecordsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			path, content, layout := setBellBearingFixture(t, platform, false, false)
			engine := New()
			loaded, err := engine.LoadSave(path, string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.SetBellBearingUnlocked(
				loaded.SaveSessionID, content.slot, setBellBearingTestFlag,
				setBellBearingTestGameID, true, "0")
			if err != nil {
				t.Fatalf("SetBellBearingUnlocked: %v", err)
			}
			if result.SaveRevision != "1" || !result.Unlocked {
				t.Errorf("result = %+v, want revision 1 and unlocked", result)
			}

			flags, err := engine.GetEventFlags(
				loaded.SaveSessionID, content.slot, []uint32{setBellBearingTestFlag})
			if err != nil || !flags.Flags[setBellBearingTestFlag] {
				t.Fatalf("flag after mutation = %+v, err %v", flags.Flags, err)
			}
			snapshot := engine.sessions[loaded.SaveSessionID].snapshot
			for name, at := range map[string]int64{
				"inventory common": layout.inventoryCommon + 2*setBellBearingRecordSize,
				"inventory key":    layout.inventoryKey + 3*setBellBearingRecordSize,
				"storage common":   layout.storageCommon + 4*setBellBearingRecordSize,
			} {
				raw, err := snapshot.readAt(at, int(setBellBearingRecordSize))
				if err != nil {
					t.Fatalf("read %s: %v", name, err)
				}
				if binary.LittleEndian.Uint32(raw) != 0 || binary.LittleEndian.Uint32(raw[4:]) != 0 {
					t.Errorf("%s was not cleared: % X", name, raw)
				}
			}
			if count, _ := snapshot.uint32At(layout.inventoryCommon - 4); count != 0 {
				t.Errorf("inventory common count = %d, want 0", count)
			}
			if count, _ := snapshot.uint32At(layout.inventoryKey - 4); count != 1 {
				t.Errorf("inventory key count = %d, want unchanged 1", count)
			}
			if count, _ := snapshot.uint32At(layout.storageCommon - 4); count != 0 {
				t.Errorf("storage common count = %d, want 0", count)
			}
		})
	}
}

func TestSetBellBearingUnlockedClearsOnlyTheFlag(t *testing.T) {
	path, content, _ := setBellBearingFixture(t, PlatformPC, true, false)
	engine := New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	if _, err := engine.SetBellBearingUnlocked(
		loaded.SaveSessionID, content.slot, setBellBearingTestFlag,
		setBellBearingTestGameID, false, "0"); err != nil {
		t.Fatalf("SetBellBearingUnlocked: %v", err)
	}
	flags, err := engine.GetEventFlags(
		loaded.SaveSessionID, content.slot, []uint32{setBellBearingTestFlag})
	if err != nil || flags.Flags[setBellBearingTestFlag] {
		t.Fatalf("flag after relock = %+v, err %v", flags.Flags, err)
	}
	after := engine.sessions[loaded.SaveSessionID].snapshot.data
	changed := 0
	for index := range before {
		if before[index] != after[index] {
			changed++
		}
	}
	if changed != 1 {
		t.Errorf("changed bytes = %d, want only the event flag byte", changed)
	}
}

func TestSetBellBearingUnlockedRejectsStorageKeyBeforeMutation(t *testing.T) {
	path, content, layout := setBellBearingFixture(t, PlatformPC, false, false)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	binary.LittleEndian.PutUint32(data[layout.storageKey:], setBellBearingTestGameID)
	binary.LittleEndian.PutUint32(data[layout.storageKey+4:], 1)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	engine := New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)
	_, err = engine.SetBellBearingUnlocked(
		loaded.SaveSessionID, content.slot, setBellBearingTestFlag,
		setBellBearingTestGameID, true, "0")
	if err == nil || !strings.Contains(err.Error(), "Storage key") {
		t.Fatalf("error = %v, want unsupported Storage key", err)
	}
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) {
		t.Error("rejected mutation changed the snapshot")
	}
	info, _ := engine.GetSessionInfo(loaded.SaveSessionID)
	if info.UnsavedChanges || engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" {
		t.Errorf("rejected session = %+v, revision %q", info,
			engine.sessions[loaded.SaveSessionID].session.revisionString())
	}
}

func TestSetBellBearingUnlockedPersistsAndReloads(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			path, content, _ := setBellBearingFixture(t, platform, false, true)
			engine := New()
			loaded, err := engine.LoadSave(path, string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			if _, err := engine.SetBellBearingUnlocked(
				loaded.SaveSessionID, content.slot, setBellBearingTestFlag,
				setBellBearingTestGameID, true, "0"); err != nil {
				t.Fatalf("SetBellBearingUnlocked: %v", err)
			}
			target := filepath.Join(t.TempDir(), "written.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}

			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform))
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			flags, err := reloadedEngine.GetEventFlags(
				reloaded.SaveSessionID, content.slot, []uint32{setBellBearingTestFlag})
			if err != nil || !flags.Flags[setBellBearingTestFlag] {
				t.Fatalf("reloaded flag = %+v, err %v", flags.Flags, err)
			}
			inventory, err := reloadedEngine.GetInventory(
				reloaded.SaveSessionID, content.slot, "", 1, 50)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}
			for _, record := range inventory.Records {
				if record.GaItemHandle == setBellBearingTestGameID ||
					record.GaItemHandle == setBellBearingTestHandle {
					t.Errorf("reloaded inventory still contains Bell Bearing: %+v", record)
				}
			}
			storage, err := reloadedEngine.GetStorage(
				reloaded.SaveSessionID, content.slot, "", 1, 50)
			if err != nil {
				t.Fatalf("GetStorage: %v", err)
			}
			for _, record := range storage.Records {
				if record.GaItemHandle == setBellBearingTestGameID ||
					record.GaItemHandle == setBellBearingTestHandle {
					t.Errorf("reloaded storage still contains Bell Bearing: %+v", record)
				}
			}
		})
	}
}
