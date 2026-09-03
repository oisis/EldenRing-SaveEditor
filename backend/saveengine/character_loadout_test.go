package saveengine

import (
	"encoding/binary"
	"os"
	"reflect"
	"strings"
	"testing"
)

const (
	loadoutTestSlot            = 2
	loadoutTestAnchorAt        = int64(setQuickAnchorAt)
	loadoutTestDagger          = uint32(0x000F4240)
	loadoutTestMemoryStone     = uint32(0x4000272E)
	loadoutTestCrimsonTear     = uint32(0x40002AF9)
	loadoutTestSpell           = uint32(0x00000FA0)
	loadoutTestMoonOfNokstella = uint32(0x20000474)
)

func writeCharacterLoadoutFixture(t *testing.T, platform Platform) string {
	t.Helper()
	path, _ := writeSetQuickItemsFixture(t, platform, loadoutTestSlot, []struct {
		gameID   uint32
		quantity uint32
	}{
		{gameID: loadoutTestMemoryStone, quantity: 3},
		{gameID: loadoutTestCrimsonTear, quantity: 2},
	})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var slotBase int64
	if platform == PlatformPS4 {
		slotBase = equipmentPS4SlotDataBase + int64(loadoutTestSlot)*equipmentPS4SlotStride
	} else {
		slotBase = equipmentPCSlotDataBase + int64(loadoutTestSlot)*equipmentPCSlotStride
	}
	anchor := slotBase + loadoutTestAnchorAt
	put := func(at int64, value uint32) {
		binary.LittleEndian.PutUint32(data[anchor+at:], value)
	}

	count := binary.LittleEndian.Uint32(data[anchor+equipmentProjectileCountOffset:])
	blockAt := equipmentProjectileCountOffset + 4 + int64(count)*equipmentProjectileRecordSize
	equipment := [equipmentSlotCount]uint32{
		unarmedEquipmentGameID, loadoutTestDagger,
		unarmedEquipmentGameID, unarmedEquipmentGameID,
		unarmedEquipmentGameID, unarmedEquipmentGameID,
		0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF,
		0xFFFFFFFF, 0xFFFFFFFF,
		equippedArmorEmptyGameIDs[0], equippedArmorEmptyGameIDs[1],
		equippedArmorEmptyGameIDs[2], equippedArmorEmptyGameIDs[3],
		0xFFFFFFFF,
		loadoutTestMoonOfNokstella, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF,
		0xFFFFFFFF,
	}
	for index, value := range equipment {
		put(blockAt+int64(index*4), value)
	}

	// Quick Items position 1 and Pouch position 1 both reference the first
	// Inventory common row. The fixture helper already created that row and the
	// Pouch reference; this adds the corresponding Quick Items triple.
	handle, err := gaItemHandleForGameID(loadoutTestMemoryStone)
	if err != nil {
		t.Fatalf("gaItemHandleForGameID: %v", err)
	}
	put(quickItemsSectionOffset, handle)
	put(quickItemsSectionOffset+4, removeReferenceInventoryRowBase)
	put(blockAt+quickItemsTailOffset, loadoutTestMemoryStone)
	for index := 0; index < pouchItemSlotCount; index++ {
		put(pouchItemsSectionOffset+int64(index*pouchItemRecordSize), pouchEmptyItemID)
		put(pouchItemsSectionOffset+int64(index*pouchItemRecordSize+4), pouchEmptyEquipIndex)
		put(blockAt+pouchItemsTailOffset+int64(index*4), PouchEmptyGameID)
	}
	put(pouchItemsSectionOffset, handle)
	put(pouchItemsSectionOffset+4, removeReferenceInventoryRowBase)
	put(blockAt+pouchItemsTailOffset, loadoutTestMemoryStone)

	// The equipped Dagger and talisman are addressed the way the three armament
	// writers address them: the physical Inventory common row in EquipedItemIndex
	// and the exact GaItem handle of that row in ActiveEquipedItemsGa.
	const (
		loadoutTestDaggerRow    = 100
		loadoutTestDaggerHandle = uint32(0x80000064)
		loadoutTestMoonRow      = 101
		loadoutTestMoonHandle   = uint32(0xA0000474)
	)
	// A weapon handle carries no game ID of its own, so the equipped Dagger also
	// needs its record in the GaItem table the writers resolve through.
	binary.LittleEndian.PutUint32(data[slotBase+gaItemTableOffset:], loadoutTestDaggerHandle)
	binary.LittleEndian.PutUint32(data[slotBase+gaItemTableOffset+4:], loadoutTestDagger)

	daggerRowAt := inventoryHeldCommonOffset + int64(loadoutTestDaggerRow*inventoryHeldRecordSize)
	put(daggerRowAt, loadoutTestDaggerHandle)
	put(daggerRowAt+4, 1)
	put(daggerRowAt+8, uint32(loadoutTestDaggerRow))
	put(removeEquipmentIndexesOffset+1*4, removeReferenceInventoryRowBase+loadoutTestDaggerRow)
	put(removeEquipmentHandlesOffset+1*4, loadoutTestDaggerHandle)

	moonRowAt := inventoryHeldCommonOffset + int64(loadoutTestMoonRow*inventoryHeldRecordSize)
	put(moonRowAt, loadoutTestMoonHandle)
	put(moonRowAt+4, 1)
	put(moonRowAt+8, uint32(loadoutTestMoonRow))
	put(removeEquipmentIndexesOffset+17*4, removeReferenceInventoryRowBase+loadoutTestMoonRow)
	put(removeEquipmentHandlesOffset+17*4, loadoutTestMoonHandle)

	// The mixture is positional and keeps its second slot empty.
	put(blockAt+physickArmamentsBlockSize, loadoutTestCrimsonTear)
	put(blockAt+physickArmamentsBlockSize+4, PhysickEmptyTearID)

	for index := 0; index < equippedSpellSlotCount; index++ {
		put(equippedSpellsSectionOffset+int64(index*equippedSpellRecordSize), equippedSpellEmptyID)
		put(equippedSpellsSectionOffset+int64(index*equippedSpellRecordSize+4), equippedSpellEmptyFollower)
	}
	put(equippedSpellsSectionOffset, loadoutTestSpell)
	put(equippedSpellsSectionOffset+4, equippedSpellOccupiedFollower)
	put(equippedSpellsSectionOffset+equippedSpellsActiveAt, 0)
	data[anchor+talismanSlotsOffset] = 0

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	return path
}

func TestGetCharacterLoadoutSnapshotReadsOneCoherentSnapshotOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeCharacterLoadoutFixture(t, platform), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetCharacterLoadoutSnapshot(loaded.SaveSessionID, loadoutTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterLoadoutSnapshot: %v", err)
			}
			if !result.Active || result.SaveRevision != "0" || result.CharacterID != loadoutTestSlot {
				t.Fatalf("identity = %+v", result)
			}
			if result.Equipment[1] != loadoutTestDagger ||
				result.Equipment[12] != equippedArmorEmptyGameIDs[0] ||
				result.Equipment[17] != loadoutTestMoonOfNokstella {
				t.Errorf("Equipment = %08X / %08X / %08X",
					result.Equipment[1], result.Equipment[12], result.Equipment[17])
			}
			if result.QuickItems[0].GameID != loadoutTestMemoryStone ||
				result.QuickItems[0].Quantity != 3 || result.QuickItems[0].OwnedItemID == "" {
				t.Errorf("QuickItems[0] = %+v", result.QuickItems[0])
			}
			if result.Pouch[0].GameID != loadoutTestMemoryStone ||
				result.Pouch[0].Quantity != 3 || result.Pouch[0].OwnedItemID == "" {
				t.Errorf("Pouch[0] = %+v", result.Pouch[0])
			}
			if result.ActiveQuickItem != 4 || result.Physick != [2]uint32{loadoutTestCrimsonTear, PhysickEmptyTearID} {
				t.Errorf("activeQuick/physick = %d / %08X", result.ActiveQuickItem, result.Physick)
			}
			if result.Spells[0] != loadoutTestSpell || result.ActiveSpellIndex != 0 ||
				result.AvailableMemorySlots != 7 || result.UnlockedTalismanSlots != 1 {
				t.Errorf("spell state = first %08X active %d available %d talismans %d",
					result.Spells[0], result.ActiveSpellIndex, result.AvailableMemorySlots,
					result.UnlockedTalismanSlots)
			}
		})
	}
}

func TestIsTechnicalEmptyEquipmentSlotUsesTheNativeValueOfEachPublicGroup(t *testing.T) {
	tests := []struct {
		name   string
		index  int
		gameID uint32
		empty  bool
	}{
		{"unarmed hand", 1, unarmedEquipmentGameID, true},
		{"weapon in hand", 1, loadoutTestDagger, false},
		{"empty arrow", 6, loadoutEquipmentEmptyID, true},
		{"wrong empty sentinel in hand", 1, loadoutEquipmentEmptyID, false},
		{"bare head", 12, equippedArmorEmptyGameIDs[0], true},
		{"bare head in chest slot", 13, equippedArmorEmptyGameIDs[0], false},
		{"empty talisman", 17, loadoutEquipmentEmptyID, true},
		{"technical unknown field", 16, loadoutEquipmentEmptyID, false},
		{"outside equipment", 22, loadoutEquipmentEmptyID, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsTechnicalEmptyEquipmentSlot(test.index, test.gameID); got != test.empty {
				t.Fatalf("IsTechnicalEmptyEquipmentSlot(%d, 0x%08X) = %t, want %t",
					test.index, test.gameID, got, test.empty)
			}
		})
	}
}

func TestGetCharacterLoadoutSnapshotDoesNotReadResidualSlotData(t *testing.T) {
	engine, sessionID := loadValidationFixture(t, validationTestFixture{
		platform: PlatformPC,
		inactive: true,
	})
	result, err := engine.GetCharacterLoadoutSnapshot(sessionID, validationTestSlot)
	if err != nil {
		t.Fatalf("GetCharacterLoadoutSnapshot: %v", err)
	}
	if result.Active || result.SaveRevision != "0" || result.ActiveSpellIndex != -1 {
		t.Fatalf("result = %+v", result)
	}
	if result.Equipment != ([equipmentSlotCount]uint32{}) ||
		!reflect.DeepEqual(result.QuickItems, [quickItemSlotCount]CharacterLoadoutOwnedItem{}) {
		t.Fatal("inactive slot exposed residual loadout data")
	}
}

func TestGetCharacterLoadoutSnapshotRejectsInconsistentOwnedReference(t *testing.T) {
	path := writeCharacterLoadoutFixture(t, PlatformPC)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchor := equipmentPCSlotDataBase + int64(loadoutTestSlot)*equipmentPCSlotStride + loadoutTestAnchorAt
	count := binary.LittleEndian.Uint32(data[anchor+equipmentProjectileCountOffset:])
	blockAt := anchor + equipmentProjectileCountOffset + 4 + int64(count)*equipmentProjectileRecordSize
	binary.LittleEndian.PutUint32(data[blockAt+quickItemsTailOffset:], loadoutTestCrimsonTear)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine := New()
	loaded, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = engine.GetCharacterLoadoutSnapshot(loaded.SaveSessionID, loadoutTestSlot)
	if err == nil || !strings.Contains(err.Error(), "quick item slot 0: inconsistent existing save state") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetCharacterLoadoutSnapshotRejectsActiveIndexWithoutOccupiedSpell(t *testing.T) {
	path := writeCharacterLoadoutFixture(t, PlatformPC)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchor := equipmentPCSlotDataBase + int64(loadoutTestSlot)*equipmentPCSlotStride + loadoutTestAnchorAt
	binary.LittleEndian.PutUint32(data[anchor+equippedSpellsSectionOffset+equippedSpellsActiveAt:], 9)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine := New()
	loaded, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = engine.GetCharacterLoadoutSnapshot(loaded.SaveSessionID, loadoutTestSlot)
	if err == nil || !strings.Contains(err.Error(), "does not address an occupied public spell position") {
		t.Fatalf("error = %v", err)
	}
}
