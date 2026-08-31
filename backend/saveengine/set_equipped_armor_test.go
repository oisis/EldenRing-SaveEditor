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
	setArmorSlot            = 1
	setArmorRecordCount     = 8
	setArmorAnchorAt        = 0x20 + setArmorRecordCount*16 + (5120-setArmorRecordCount)*8
	setArmorIndexesAt       = 0x101
	setArmorItemIDsAt       = 0x175
	setArmorHandlesAt       = 0x1CD
	setArmorProjectileAt    = 0x931D
	setArmorInventoryAt     = setArmorAnchorAt + 505
	setArmorInventoryStride = 12
	setArmorFirstSlot       = 12
	setArmorInventoryBase   = 0x180
)

var (
	setArmorEmptyGameIDs  = [4]uint32{0x10002710, 0x10002774, 0x100027D8, 0x1000283C}
	setArmorActualGameIDs = [4]uint32{0x10009C40, 0x10009CA4, 0x10009D08, 0x10009D6C}
)

func writeSetEquippedArmorFixture(t *testing.T, platform Platform) string {
	t.Helper()
	var data []byte
	var userData10Base, slotBase int64
	if platform == PlatformPS4 {
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = inventoryTestPS4SlotDataBase + setArmorSlot*inventoryTestPS4SlotStride
	} else {
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = inventoryTestPCSlotDataBase + setArmorSlot*inventoryTestPCSlotStride
	}
	data[userData10Base+userData10ActiveFlagsOffset+setArmorSlot] = 1
	binary.LittleEndian.PutUint32(data[slotBase:], 83)

	gameIDs := append(setArmorEmptyGameIDs[:], setArmorActualGameIDs[:]...)
	position := int64(0x20)
	for index, gameID := range gameIDs {
		handle := uint32(0x90000100 + index)
		binary.LittleEndian.PutUint32(data[slotBase+position:], handle)
		binary.LittleEndian.PutUint32(data[slotBase+position+4:], gameID)
		position += 16

		rowAt := slotBase + setArmorInventoryAt + int64(index*setArmorInventoryStride)
		binary.LittleEndian.PutUint32(data[rowAt:], handle)
		binary.LittleEndian.PutUint32(data[rowAt+4:], 1)
		binary.LittleEndian.PutUint32(data[rowAt+8:], uint32(index+1))
	}
	copy(data[slotBase+setArmorAnchorAt:], inventoryTestAnchor)

	anchor := slotBase + setArmorAnchorAt
	armamentsAt := anchor + setArmorProjectileAt + 4
	for slot, gameID := range setArmorEmptyGameIDs {
		handle := uint32(0x90000100 + slot)
		binary.LittleEndian.PutUint32(
			data[anchor+setArmorIndexesAt+int64(slot*4):],
			setArmorInventoryBase+uint32(slot))
		binary.LittleEndian.PutUint32(
			data[anchor+setArmorItemIDsAt+int64(slot*4):], gameID&0x0FFFFFFF)
		binary.LittleEndian.PutUint32(
			data[anchor+setArmorHandlesAt+int64(slot*4):], handle)
		binary.LittleEndian.PutUint32(
			data[armamentsAt+int64((setArmorFirstSlot+slot)*4):], gameID)
	}

	path := filepath.Join(t.TempDir(), "equipped-armor.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func validateSetArmorGameID(slot int, gameID uint32) error {
	if setArmorActualGameIDs[slot] != gameID {
		return os.ErrInvalid
	}
	return nil
}

func TestSetEquippedArmorWritesAndClearsFourRepresentationsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeSetEquippedArmorFixture(t, platform), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			inventory, err := engine.GetInventory(
				loaded.SaveSessionID, setArmorSlot, InventorySectionCommon, 1, 50)
			if err != nil || len(inventory.Records) != setArmorRecordCount {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
			}
			var assignments [4]*string
			for slot := range assignments {
				token := inventory.Records[slot+4].OwnedItemID
				assignments[slot] = &token
			}

			before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)
			result, err := engine.SetEquippedArmor(
				loaded.SaveSessionID, setArmorSlot, assignments, "0", validateSetArmorGameID)
			if err != nil {
				t.Fatalf("SetEquippedArmor: %v", err)
			}
			if result.SaveRevision != "1" || result.GameIDs != setArmorActualGameIDs {
				t.Fatalf("result = %+v", result)
			}

			var slotBase int64
			if platform == PlatformPS4 {
				slotBase = inventoryTestPS4SlotDataBase + setArmorSlot*inventoryTestPS4SlotStride
			} else {
				slotBase = inventoryTestPCSlotDataBase + setArmorSlot*inventoryTestPCSlotStride
			}
			anchor := slotBase + setArmorAnchorAt
			armamentsAt := anchor + setArmorProjectileAt + 4
			allowedRanges := []int64{
				anchor + setArmorIndexesAt,
				anchor + setArmorItemIDsAt,
				anchor + setArmorHandlesAt,
				armamentsAt + setArmorFirstSlot*4,
			}
			after := engine.sessions[loaded.SaveSessionID].snapshot.data
			for offset := range before {
				if before[offset] == after[offset] {
					continue
				}
				changedAt := int64(offset)
				allowed := false
				for _, start := range allowedRanges {
					allowed = allowed || changedAt >= start && changedAt < start+16
				}
				if !allowed {
					t.Fatalf("unexpected byte modified at 0x%X", changedAt)
				}
			}

			var cleared [4]*string
			clearedResult, err := engine.SetEquippedArmor(
				loaded.SaveSessionID, setArmorSlot, cleared, "1", validateSetArmorGameID)
			if err != nil {
				t.Fatalf("clear SetEquippedArmor: %v", err)
			}
			if clearedResult.SaveRevision != "2" || clearedResult.GameIDs != ([4]uint32{}) {
				t.Fatalf("clear result = %+v", clearedResult)
			}

			target := filepath.Join(t.TempDir(), "equipped-armor-reloaded.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "2", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			equipment, err := reloadedEngine.GetEquipment(reloaded.SaveSessionID, setArmorSlot)
			if err != nil {
				t.Fatalf("GetEquipment after reload: %v", err)
			}
			for slot, gameID := range setArmorEmptyGameIDs {
				if equipment.Slots[setArmorFirstSlot+slot] != gameID {
					t.Fatalf("reloaded armor slot %d = 0x%08X, want 0x%08X",
						slot, equipment.Slots[setArmorFirstSlot+slot], gameID)
				}
			}
		})
	}
}

func TestSetEquippedArmorRejectsInvalidPlansWithoutMutation(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeSetEquippedArmorFixture(t, PlatformPC), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmorSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	token := inventory.Records[4].OwnedItemID
	assignments := [4]*string{&token, nil, nil, nil}
	before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)

	_, err = engine.SetEquippedArmor(
		loaded.SaveSessionID, setArmorSlot, assignments, "0",
		func(int, uint32) error { return os.ErrPermission })
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("catalog validation error = %v", err)
	}
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) ||
		engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" {
		t.Fatal("rejected mutation changed snapshot or revision")
	}
}

func TestSetEquippedArmorRejectsClearWithoutNativeEmptyRecord(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeSetEquippedArmorFixture(t, PlatformPC), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmorSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	head := inventory.Records[4].OwnedItemID
	assignments := [4]*string{&head, nil, nil, nil}
	if _, err := engine.SetEquippedArmor(
		loaded.SaveSessionID, setArmorSlot, assignments, "0", validateSetArmorGameID); err != nil {
		t.Fatalf("equip head: %v", err)
	}

	session := engine.sessions[loaded.SaveSessionID]
	slotBase := int64(inventoryTestPCSlotDataBase + setArmorSlot*inventoryTestPCSlotStride)
	bareHeadRowAt := slotBase + setArmorInventoryAt
	for offset := int64(0); offset < setArmorInventoryStride; offset++ {
		session.snapshot.data[bareHeadRowAt+offset] = 0
	}
	before := append([]byte(nil), session.snapshot.data...)
	_, err = engine.SetEquippedArmor(
		loaded.SaveSessionID, setArmorSlot, [4]*string{}, "1", validateSetArmorGameID)
	if err == nil || !strings.Contains(err.Error(), "GaItem allocation is unsupported") {
		t.Fatalf("missing-placeholder error = %v", err)
	}
	if !bytes.Equal(before, session.snapshot.data) || session.session.revisionString() != "1" {
		t.Fatal("rejected clear changed snapshot or revision")
	}
}
