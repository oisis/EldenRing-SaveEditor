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
	setTalismanIndexesAt     = 0x115
	setTalismanItemIDsAt     = 0x189
	setTalismanHandlesAt     = 0x1E1
	setTalismanSlotsAt       = -241
	setTalismanProjectileAt  = 0x931D
	setTalismanGameID1       = uint32(0x20000064)
	setTalismanGameID2       = uint32(0x20000065)
	setTalismanTechnicalSlot = uint32(0xDEADBEEF)
)

func writeSetEquippedTalismansFixture(
	t *testing.T,
	platform Platform,
	slot int,
	additionalSlots byte,
) (string, string) {
	t.Helper()
	path, platformName := writeSetQuickItemsFixture(t, platform, slot, []struct {
		gameID   uint32
		quantity uint32
	}{{setTalismanGameID1, 1}, {setTalismanGameID2, 1}})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var slotBase int64
	if platform == PlatformPS4 {
		slotBase = quickItemsPS4SlotDataBase + int64(slot)*quickItemsPS4SlotStride
	} else {
		slotBase = quickItemsPCSlotDataBase + int64(slot)*quickItemsPCSlotStride
	}
	anchor := slotBase + setQuickAnchorAt
	data[anchor+setTalismanSlotsAt] = additionalSlots

	for _, blockAt := range []int64{
		setTalismanIndexesAt, setTalismanItemIDsAt, setTalismanHandlesAt,
	} {
		for index := 17; index <= 20; index++ {
			value := uint32(0xFFFFFFFF)
			if blockAt == setTalismanHandlesAt {
				value = 0
			}
			binary.LittleEndian.PutUint32(data[anchor+blockAt+int64((index-17)*4):], value)
		}
		binary.LittleEndian.PutUint32(data[anchor+blockAt+16:], setTalismanTechnicalSlot)
	}

	countAt := anchor + setTalismanProjectileAt
	armamentsAt := countAt + 4 + int64(binary.LittleEndian.Uint32(data[countAt:]))*8
	for index := 17; index <= 20; index++ {
		binary.LittleEndian.PutUint32(data[armamentsAt+int64(index*4):], 0xFFFFFFFF)
	}
	binary.LittleEndian.PutUint32(data[armamentsAt+21*4:], setTalismanTechnicalSlot)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path, platformName
}

func validateTalismanTestGameID(gameID uint32) error {
	if gameID == setTalismanGameID1 || gameID == setTalismanGameID2 {
		return nil
	}
	return os.ErrInvalid
}

func TestSetEquippedTalismansWritesFourRepresentationsAndReloads(t *testing.T) {
	for _, testCase := range []struct {
		platform Platform
		slot     int
	}{{PlatformPC, 0}, {PlatformPS4, 5}} {
		t.Run(string(testCase.platform), func(t *testing.T) {
			path, platform := writeSetEquippedTalismansFixture(
				t, testCase.platform, testCase.slot, 3)
			engine := New()
			loaded, err := engine.LoadSave(path, platform)
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			inventory, err := engine.GetInventory(
				loaded.SaveSessionID, testCase.slot, InventorySectionCommon, 1, 50)
			if err != nil || len(inventory.Records) != 2 {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
			}
			before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)
			result, err := engine.SetEquippedTalismans(
				loaded.SaveSessionID,
				testCase.slot,
				[]string{inventory.Records[1].OwnedItemID, inventory.Records[0].OwnedItemID},
				"0",
				validateTalismanTestGameID,
			)
			if err != nil {
				t.Fatalf("SetEquippedTalismans: %v", err)
			}
			if result.SaveRevision != "1" || result.UnlockedSlots != 4 ||
				len(result.GameIDs) != 2 || result.GameIDs[0] != setTalismanGameID2 {
				t.Fatalf("result = %+v", result)
			}

			var slotBase int64
			if testCase.platform == PlatformPS4 {
				slotBase = quickItemsPS4SlotDataBase + int64(testCase.slot)*quickItemsPS4SlotStride
			} else {
				slotBase = quickItemsPCSlotDataBase + int64(testCase.slot)*quickItemsPCSlotStride
			}
			anchor := slotBase + setQuickAnchorAt
			countAt := anchor + setTalismanProjectileAt
			armamentsAt := countAt + 4 + 17*8
			want := [4][4]uint32{
				{0x181, 0x180, 0xFFFFFFFF, 0xFFFFFFFF},
				{0x65, 0x64, 0xFFFFFFFF, 0xFFFFFFFF},
				{0xA0000065, 0xA0000064, 0, 0},
				{setTalismanGameID2, setTalismanGameID1, 0xFFFFFFFF, 0xFFFFFFFF},
			}
			for block, at := range []int64{
				anchor + setTalismanIndexesAt,
				anchor + setTalismanItemIDsAt,
				anchor + setTalismanHandlesAt,
				armamentsAt + 17*4,
			} {
				for index := 0; index < 4; index++ {
					if got := binary.LittleEndian.Uint32(
						engine.sessions[loaded.SaveSessionID].snapshot.data[at+int64(index*4):]); got != want[block][index] {
						t.Errorf("representation %d slot %d = 0x%08X, want 0x%08X",
							block, index, got, want[block][index])
					}
				}
				if got := binary.LittleEndian.Uint32(
					engine.sessions[loaded.SaveSessionID].snapshot.data[at+16:]); got != setTalismanTechnicalSlot {
					t.Errorf("technical field 21 changed in representation %d: 0x%08X", block, got)
				}
			}

			after := engine.sessions[loaded.SaveSessionID].snapshot.data
			for offset := range before {
				if before[offset] == after[offset] {
					continue
				}
				at := int64(offset)
				allowed := false
				for _, start := range []int64{
					anchor + setTalismanIndexesAt,
					anchor + setTalismanItemIDsAt,
					anchor + setTalismanHandlesAt,
					armamentsAt + 17*4,
				} {
					allowed = allowed || at >= start && at < start+16
				}
				if !allowed {
					t.Fatalf("unexpected byte modified at 0x%X", at)
				}
			}

			target := filepath.Join(t.TempDir(), "talismans.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, platform)
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			equipment, err := reloadedEngine.GetEquipment(reloaded.SaveSessionID, testCase.slot)
			if err != nil || equipment.Slots[17] != setTalismanGameID2 ||
				equipment.Slots[18] != setTalismanGameID1 || equipment.Slots[21] != setTalismanTechnicalSlot {
				t.Fatalf("GetEquipment after reload: %v, slots=%#v", err, equipment.Slots)
			}
		})
	}
}

func TestSetEquippedTalismansRejectsInvalidPlansWithoutMutation(t *testing.T) {
	path, platform := writeSetEquippedTalismansFixture(t, PlatformPC, 0, 1)
	engine := New()
	loaded, err := engine.LoadSave(path, platform)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, 0, InventorySectionCommon, 1, 50)
	if err != nil || len(inventory.Records) != 2 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}
	token := inventory.Records[0].OwnedItemID
	before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)

	for _, testCase := range []struct {
		name   string
		values []string
		want   string
	}{
		{"locked slot", []string{token, inventory.Records[1].OwnedItemID, token}, "2 unlocked"},
		{"duplicate", []string{token, token}, "both slot"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := engine.SetEquippedTalismans(
				loaded.SaveSessionID, 0, testCase.values, "0", validateTalismanTestGameID)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want containing %q", err, testCase.want)
			}
			if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) ||
				engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" {
				t.Fatal("rejected mutation changed snapshot or revision")
			}
		})
	}
}
