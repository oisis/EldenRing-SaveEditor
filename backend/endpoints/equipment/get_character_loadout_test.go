package equipment

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	loadoutEndpointDagger      = uint32(0x000F4240)
	loadoutEndpointUnarmed     = uint32(0x0001ADB0)
	loadoutEndpointMoon        = uint32(0x20000474)
	loadoutEndpointMemoryStone = uint32(0x4000272E)
	loadoutEndpointAlias       = uint32(0x400000FA)
	loadoutEndpointTear        = uint32(0x40002AF9)
)

func writeGetCharacterLoadoutFixture(t *testing.T) string {
	t.Helper()
	path := writeGetEquippedSpellsFixture(t, []uint32{rawGlintstonePebble})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	slotBase := int64(getEquippedSpellsHeaderSize) + 0x10 +
		getEquippedSpellsSlot*getEquippedSpellsSlotBlockSize
	anchor := slotBase + getEquippedSpellsAnchorAt
	put := func(at int64, value uint32) {
		binary.LittleEndian.PutUint32(data[anchor+at:], value)
	}
	count := binary.LittleEndian.Uint32(data[anchor+getEquippedSpellsCountAt:])
	blockAt := int64(getEquippedSpellsCountAt) + 4 + int64(count)*8

	equipment := [22]uint32{
		loadoutEndpointUnarmed, loadoutEndpointDagger,
		loadoutEndpointUnarmed, loadoutEndpointUnarmed,
		loadoutEndpointUnarmed, loadoutEndpointUnarmed,
		0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF,
		0xFFFFFFFF, 0xFFFFFFFF,
		0x10002710, 0x10002774, 0x100027D8, 0x1000283C,
		0xFFFFFFFF,
		loadoutEndpointMoon, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF,
		0xFFFFFFFF,
	}
	for index, value := range equipment {
		put(blockAt+int64(index*4), value)
	}

	// Native empty pairs and tails first.
	for index := 0; index < 10; index++ {
		put(0x9279+int64(index*8), 0)
		put(0x9279+int64(index*8+4), 0xFFFFFFFF)
		put(blockAt+0x58+int64(index*4), 0xFFFFFFFF)
	}
	for index := 0; index < 6; index++ {
		put(0x92CD+int64(index*8), 0)
		put(0x92CD+int64(index*8+4), 0xFFFFFFFF)
		put(blockAt+0x80+int64(index*4), 0xFFFFFFFF)
	}

	// Memory Stone lives at common row 97 in the shared spell fixture.
	put(0x9279, 0xB000272E)
	put(0x9279+4, 0x180+97)
	put(blockAt+0x58, loadoutEndpointMemoryStone)
	put(0x92CD, 0xB000272E)
	put(0x92CD+4, 0x180+97)
	put(blockAt+0x80, loadoutEndpointMemoryStone)

	// A second Quick Items record exercises the catalog alias imported from the
	// confirmed native save: the public ResourceRef must be canonical.
	aliasRow := int64(getEquippedSpellsInventoryAt + 98*12)
	put(aliasRow, 0xB00000FA)
	put(aliasRow+4, 1)
	put(aliasRow+8, 98)
	put(0x9279+8, 0xB00000FA)
	put(0x9279+12, 0x180+98)
	put(blockAt+0x58+4, loadoutEndpointAlias)

	put(blockAt+0x9C, loadoutEndpointTear)
	put(blockAt+0xA0, saveengine.PhysickEmptyTearID)
	put(getEquippedSpellsSectionAt+14*8, 0)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	return path
}

func TestGetCharacterLoadoutResolvesEveryConfirmedGroup(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetCharacterLoadoutFixture(t), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	result, err := GetCharacterLoadout(
		engine, newEquippedSpellsCatalog(t), loaded.SaveSessionID, getEquippedSpellsSlot)
	if err != nil {
		t.Fatalf("GetCharacterLoadout: %v", err)
	}

	if !result.Active || result.SaveRevision != "0" || result.CharacterID != getEquippedSpellsSlot {
		t.Fatalf("identity = %+v", result)
	}
	if len(result.RightHand) != 3 || len(result.LeftHand) != 3 ||
		len(result.Arrows) != 2 || len(result.Bolts) != 2 || len(result.Armor) != 4 ||
		len(result.Talismans) != 4 || len(result.QuickItems) != 10 || len(result.Pouch) != 6 ||
		len(result.Physick) != 2 || len(result.Spells) != 12 {
		t.Fatalf("unexpected group lengths: %+v", result)
	}
	if result.RightHand[0].State != LoadoutSlotOccupied ||
		result.RightHand[0].Resource == nil || result.RightHand[0].Resource.Key == "" ||
		result.LeftHand[0].State != LoadoutSlotEmpty {
		t.Errorf("hands = right %+v left %+v", result.RightHand[0], result.LeftHand[0])
	}
	for index, slot := range result.Armor {
		if slot.State != LoadoutSlotEmpty || slot.Resource != nil {
			t.Errorf("Armor[%d] = %+v, want technical empty", index, slot)
		}
	}
	if result.Talismans[0].State != LoadoutSlotOccupied ||
		result.Talismans[0].Resource == nil || result.Talismans[0].Resource.Key != "20000474" {
		t.Errorf("Talismans[0] = %+v", result.Talismans[0])
	}
	for index := 1; index < 4; index++ {
		if result.Talismans[index].State != LoadoutSlotLocked {
			t.Errorf("Talismans[%d] = %+v, want locked", index, result.Talismans[index])
		}
	}
	if result.QuickItems[0].Quantity != 3 || result.QuickItems[0].OwnedItemID == "" ||
		result.Pouch[0].Quantity != 3 || result.Pouch[0].OwnedItemID == "" {
		t.Errorf("owned slots = quick %+v pouch %+v", result.QuickItems[0], result.Pouch[0])
	}
	if result.QuickItems[1].Resource == nil ||
		*result.QuickItems[1].Resource != (schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "400000FB"}) {
		t.Errorf("alias slot = %+v, want canonical 400000FB", result.QuickItems[1])
	}
	if result.Physick[0].State != LoadoutSlotOccupied || result.Physick[1].State != LoadoutSlotEmpty {
		t.Errorf("Physick = %+v", result.Physick)
	}
	if result.Spells[0].State != LoadoutSlotOccupied || result.Spells[0].MemorySlots != 1 ||
		result.ActiveSpellIndex != 0 || result.UsedMemorySlots != 1 ||
		result.AvailableMemorySlots != 7 || result.UnlockedTalismanSlots != 1 {
		t.Errorf("spell/capacity = %+v active=%d used=%d available=%d talismans=%d",
			result.Spells[0], result.ActiveSpellIndex, result.UsedMemorySlots,
			result.AvailableMemorySlots, result.UnlockedTalismanSlots)
	}
}

func TestGetCharacterLoadoutReturnsEmptyGroupsForInactiveSlot(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetCharacterLoadoutFixture(t), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	result, err := GetCharacterLoadout(engine, newEquippedSpellsCatalog(t), loaded.SaveSessionID, 0)
	if err != nil {
		t.Fatalf("GetCharacterLoadout: %v", err)
	}
	if result.Active || result.ActiveSpellIndex != -1 || result.SaveRevision != "0" ||
		len(result.RightHand) != 0 || len(result.QuickItems) != 0 || len(result.Spells) != 0 {
		t.Fatalf("inactive result = %+v", result)
	}
}

func TestGetCharacterLoadoutFailsClosedForUnknownOccupiedItem(t *testing.T) {
	path := writeGetCharacterLoadoutFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	slotBase := int64(getEquippedSpellsHeaderSize) + 0x10 +
		getEquippedSpellsSlot*getEquippedSpellsSlotBlockSize
	anchor := slotBase + getEquippedSpellsAnchorAt
	count := binary.LittleEndian.Uint32(data[anchor+getEquippedSpellsCountAt:])
	blockAt := anchor + getEquippedSpellsCountAt + 4 + int64(count)*8
	binary.LittleEndian.PutUint32(data[blockAt+4:], 0x0FFFFFFE)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = GetCharacterLoadout(
		engine, newEquippedSpellsCatalog(t), loaded.SaveSessionID, getEquippedSpellsSlot)
	if err == nil || !strings.Contains(err.Error(), "right hand: slot 0: game ID 0x0FFFFFFE is not a known item") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetCharacterLoadoutFailsClosedWhenSpellsExceedAvailableCapacity(t *testing.T) {
	path := writeGetCharacterLoadoutFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	slotBase := int64(getEquippedSpellsHeaderSize) + 0x10 +
		getEquippedSpellsSlot*getEquippedSpellsSlotBlockSize
	anchor := slotBase + getEquippedSpellsAnchorAt
	for index := 0; index < 8; index++ {
		at := anchor + getEquippedSpellsSectionAt + int64(index*getEquippedSpellsRecordSize)
		binary.LittleEndian.PutUint32(data[at:], rawGlintstonePebble)
		binary.LittleEndian.PutUint32(data[at+4:], 0xFFFFFFFF)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = GetCharacterLoadout(
		engine, newEquippedSpellsCatalog(t), loaded.SaveSessionID, getEquippedSpellsSlot)
	if err == nil || !strings.Contains(err.Error(), "equipped spells use 8 memory slots") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetCharacterLoadoutRejectsWrongFamilyInOwnedSlot(t *testing.T) {
	path := writeGetCharacterLoadoutFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	slotBase := int64(getEquippedSpellsHeaderSize) + 0x10 +
		getEquippedSpellsSlot*getEquippedSpellsSlotBlockSize
	anchor := slotBase + getEquippedSpellsAnchorAt
	count := binary.LittleEndian.Uint32(data[anchor+getEquippedSpellsCountAt:])
	blockAt := int64(getEquippedSpellsCountAt) + 4 + int64(count)*8
	put := func(at int64, value uint32) {
		binary.LittleEndian.PutUint32(data[anchor+at:], value)
	}

	const spellHandle = uint32(0xB0000FA0)
	aliasRow := int64(getEquippedSpellsInventoryAt + 98*12)
	put(aliasRow, spellHandle)
	put(aliasRow+4, 1)
	put(aliasRow+8, 98)
	put(0x9279+8, spellHandle)
	put(0x9279+12, 0x180+98)
	put(blockAt+0x58+4, gamecatalog.EquippedSpellGameIDPrefix|rawGlintstonePebble)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = GetCharacterLoadout(
		engine, newEquippedSpellsCatalog(t), loaded.SaveSessionID, getEquippedSpellsSlot)
	if err == nil || !strings.Contains(err.Error(), "quick items: slot 1: game ID 0x40000FA0 has item family \"spell\"") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetCharacterLoadoutRejectsMissingDependencies(t *testing.T) {
	if _, err := GetCharacterLoadout(nil, newEquippedSpellsCatalog(t), "session", 0); err == nil ||
		err.Error() != "save engine is not available" {
		t.Fatalf("nil engine error = %v", err)
	}
	if _, err := GetCharacterLoadout(saveengine.New(), nil, "session", 0); err == nil ||
		err.Error() != "game catalog is not available" {
		t.Fatalf("nil catalog error = %v", err)
	}
}
