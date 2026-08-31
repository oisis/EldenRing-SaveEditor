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
	setArmamentsSlot            = 1
	setArmamentsRecordCount     = 7
	setArmamentsAnchorAt        = 0x20 + setArmamentsRecordCount*21 + (5120-setArmamentsRecordCount)*8
	setArmamentsIndexesAt       = 0xD1
	setArmamentsItemIDsAt       = 0x145
	setArmamentsHandlesAt       = 0x19D
	setArmamentsProjectileAt    = 0x931D
	setArmamentsInventoryAt     = setArmamentsAnchorAt + 505
	setArmamentsInventoryStride = 12
	setArmamentsInventoryBase   = 0x180
	setArmamentsUnarmedGameID   = uint32(0x0001ADB0)
	setArmamentsWeaponGameID    = uint32(0x000F4240)
)

func writeSetEquippedArmamentsFixture(t *testing.T, platform Platform) string {
	t.Helper()
	var data []byte
	var userData10Base, slotBase int64
	if platform == PlatformPS4 {
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = inventoryTestPS4SlotDataBase + setArmamentsSlot*inventoryTestPS4SlotStride
	} else {
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = inventoryTestPCSlotDataBase + setArmamentsSlot*inventoryTestPCSlotStride
	}
	data[userData10Base+userData10ActiveFlagsOffset+setArmamentsSlot] = 1
	binary.LittleEndian.PutUint32(data[slotBase:], 83)

	for index := 0; index < setArmamentsRecordCount; index++ {
		handle := uint32(0x80000100 + index)
		gameID := setArmamentsWeaponGameID
		if index == 0 {
			gameID = setArmamentsUnarmedGameID
		}
		position := slotBase + 0x20 + int64(index*21)
		binary.LittleEndian.PutUint32(data[position:], handle)
		binary.LittleEndian.PutUint32(data[position+4:], gameID)

		rowAt := slotBase + setArmamentsInventoryAt + int64(index*setArmamentsInventoryStride)
		binary.LittleEndian.PutUint32(data[rowAt:], handle)
		binary.LittleEndian.PutUint32(data[rowAt+4:], 1)
		binary.LittleEndian.PutUint32(data[rowAt+8:], uint32(index+1))
	}
	copy(data[slotBase+setArmamentsAnchorAt:], inventoryTestAnchor)

	anchor := slotBase + setArmamentsAnchorAt
	armamentsAt := anchor + setArmamentsProjectileAt + 4
	for slot := 0; slot < 6; slot++ {
		binary.LittleEndian.PutUint32(
			data[anchor+setArmamentsIndexesAt+int64(slot*4):], setArmamentsInventoryBase)
		binary.LittleEndian.PutUint32(
			data[anchor+setArmamentsItemIDsAt+int64(slot*4):], setArmamentsUnarmedGameID)
		binary.LittleEndian.PutUint32(
			data[anchor+setArmamentsHandlesAt+int64(slot*4):], 0x80000100)
		binary.LittleEndian.PutUint32(
			data[armamentsAt+int64(slot*4):], setArmamentsUnarmedGameID)
	}

	path := filepath.Join(t.TempDir(), "equipped-armaments.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func validateSetArmamentGameID(_ int, gameID uint32) error {
	if gameID != setArmamentsWeaponGameID {
		return os.ErrInvalid
	}
	return nil
}

func TestSetEquippedArmamentsWritesAndClearsFourRepresentationsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeSetEquippedArmamentsFixture(t, platform), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			inventory, err := engine.GetInventory(
				loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
			if err != nil || len(inventory.Records) != setArmamentsRecordCount {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
			}
			var assignments [6]*string
			for slot := range assignments {
				token := inventory.Records[slot+1].OwnedItemID
				assignments[slot] = &token
			}

			before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)
			result, err := engine.SetEquippedArmaments(
				loaded.SaveSessionID, setArmamentsSlot, assignments, "0", validateSetArmamentGameID)
			if err != nil {
				t.Fatalf("SetEquippedArmaments: %v", err)
			}
			if result.SaveRevision != "1" {
				t.Fatalf("result = %+v", result)
			}
			for slot, gameID := range result.GameIDs {
				if gameID != setArmamentsWeaponGameID {
					t.Fatalf("gameIDs[%d] = 0x%08X", slot, gameID)
				}
			}

			var slotBase int64
			if platform == PlatformPS4 {
				slotBase = inventoryTestPS4SlotDataBase + setArmamentsSlot*inventoryTestPS4SlotStride
			} else {
				slotBase = inventoryTestPCSlotDataBase + setArmamentsSlot*inventoryTestPCSlotStride
			}
			anchor := slotBase + setArmamentsAnchorAt
			armamentsAt := anchor + setArmamentsProjectileAt + 4
			allowedRanges := []int64{
				anchor + setArmamentsIndexesAt,
				anchor + setArmamentsItemIDsAt,
				anchor + setArmamentsHandlesAt,
				armamentsAt,
			}
			after := engine.sessions[loaded.SaveSessionID].snapshot.data
			for offset := range before {
				if before[offset] == after[offset] {
					continue
				}
				changedAt := int64(offset)
				allowed := false
				for _, start := range allowedRanges {
					allowed = allowed || changedAt >= start && changedAt < start+24
				}
				if !allowed {
					t.Fatalf("unexpected byte modified at 0x%X", changedAt)
				}
			}

			clearedResult, err := engine.SetEquippedArmaments(
				loaded.SaveSessionID, setArmamentsSlot, [6]*string{}, "1", validateSetArmamentGameID)
			if err != nil {
				t.Fatalf("clear SetEquippedArmaments: %v", err)
			}
			if clearedResult.SaveRevision != "2" || clearedResult.GameIDs != ([6]uint32{}) {
				t.Fatalf("clear result = %+v", clearedResult)
			}

			target := filepath.Join(t.TempDir(), "equipped-armaments-reloaded.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "2", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			equipment, err := reloadedEngine.GetEquipment(reloaded.SaveSessionID, setArmamentsSlot)
			if err != nil {
				t.Fatalf("GetEquipment after reload: %v", err)
			}
			for slot := 0; slot < 6; slot++ {
				if equipment.Slots[slot] != setArmamentsUnarmedGameID {
					t.Fatalf("reloaded armament slot %d = 0x%08X, want Unarmed",
						slot, equipment.Slots[slot])
				}
			}
		})
	}
}

func TestSetEquippedArmamentsRejectsInvalidPlanWithoutMutation(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeSetEquippedArmamentsFixture(t, PlatformPC), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	token := inventory.Records[1].OwnedItemID
	assignments := [6]*string{&token, nil, nil, nil, nil, nil}
	before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)

	_, err = engine.SetEquippedArmaments(
		loaded.SaveSessionID, setArmamentsSlot, assignments, "0",
		func(int, uint32) error { return os.ErrPermission })
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("catalog validation error = %v", err)
	}
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) ||
		engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" {
		t.Fatal("rejected mutation changed snapshot or revision")
	}

	assignments[1] = &token
	_, err = engine.SetEquippedArmaments(
		loaded.SaveSessionID, setArmamentsSlot, assignments, "0", validateSetArmamentGameID)
	if err == nil || !strings.Contains(err.Error(), "assigned to both slot 0 and slot 1") {
		t.Fatalf("duplicate-instance error = %v", err)
	}
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) ||
		engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" {
		t.Fatal("duplicate rejection changed snapshot or revision")
	}
}

func TestSetEquippedArmamentsRejectsClearWithoutNativeUnarmedRecord(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeSetEquippedArmamentsFixture(t, PlatformPC), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	var assignments [6]*string
	for slot := range assignments {
		token := inventory.Records[slot+1].OwnedItemID
		assignments[slot] = &token
	}
	if _, err := engine.SetEquippedArmaments(
		loaded.SaveSessionID, setArmamentsSlot, assignments, "0", validateSetArmamentGameID); err != nil {
		t.Fatalf("equip armaments: %v", err)
	}

	session := engine.sessions[loaded.SaveSessionID]
	slotBase := int64(inventoryTestPCSlotDataBase + setArmamentsSlot*inventoryTestPCSlotStride)
	unarmedRowAt := slotBase + setArmamentsInventoryAt
	for offset := int64(0); offset < setArmamentsInventoryStride; offset++ {
		session.snapshot.data[unarmedRowAt+offset] = 0
	}
	before := append([]byte(nil), session.snapshot.data...)
	_, err = engine.SetEquippedArmaments(
		loaded.SaveSessionID, setArmamentsSlot, [6]*string{}, "1", validateSetArmamentGameID)
	if err == nil || !strings.Contains(err.Error(), "GaItem allocation is unsupported") {
		t.Fatalf("missing-Unarmed error = %v", err)
	}
	if !bytes.Equal(before, session.snapshot.data) || session.session.revisionString() != "1" {
		t.Fatal("rejected clear changed snapshot or revision")
	}
}
