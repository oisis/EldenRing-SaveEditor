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

	// The shared spells fixture anchors its slot 0x640 bytes in, which leaves no
	// room in front of it for the confirmed GaItem table this loadout is now
	// cross-checked against. This fixture therefore moves its own anchored region
	// far enough into the same slot for the table to fit; every offset below
	// stays relative to the anchor, so nothing else about the layout changes.
	loadoutEndpointAnchorShift = int64(0x10000)
	loadoutEndpointAnchorAt    = getEquippedSpellsAnchorAt + loadoutEndpointAnchorShift
	loadoutEndpointRegionAt    = getEquippedSpellsAnchorAt - 0x100
	loadoutEndpointRegionSize  = int64(0xA000)
	loadoutEndpointSlotVersion = 82

	// The two confirmed reference blocks in front of InventoryHeld, measured
	// from the same anchor the rest of this fixture writes from. They are the
	// distances SaveEngine derives from inventoryHeldCommonOffset: the handle
	// block ends where the common count begins, and the index block sits two
	// 22-field blocks and a 0x1C gap in front of it.
	loadoutEndpointHandlesAt = int64(505 - 4 - 22*4)
	loadoutEndpointIndexesAt = loadoutEndpointHandlesAt - 22*4 - 0x1C - 22*4

	// The two Inventory common rows the equipped Dagger and the equipped
	// talisman are referenced through, with their exact GaItem handles.
	loadoutEndpointDaggerRow    = 100
	loadoutEndpointDaggerHandle = uint32(0x80000064)
	loadoutEndpointMoonRow      = 101
	loadoutEndpointMoonHandle   = uint32(0xA0000474)
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

	// Move the anchored region, then clear the region it came from so the
	// anchor search finds exactly one marker, the new one.
	source := slotBase + loadoutEndpointRegionAt
	copy(data[source+loadoutEndpointAnchorShift:source+loadoutEndpointAnchorShift+loadoutEndpointRegionSize],
		data[source:source+loadoutEndpointRegionSize])
	for index := source; index < source+loadoutEndpointRegionSize; index++ {
		data[index] = 0
	}
	// The slot declares its version, and the equipped Dagger gets the GaItem
	// record its weapon handle resolves through. Every other record of the table
	// stays the native empty eight-byte record.
	binary.LittleEndian.PutUint32(data[slotBase:], loadoutEndpointSlotVersion)
	binary.LittleEndian.PutUint32(data[slotBase+0x20:], loadoutEndpointDaggerHandle)
	binary.LittleEndian.PutUint32(data[slotBase+0x24:], loadoutEndpointDagger)

	anchor := slotBase + loadoutEndpointAnchorAt
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

	// The equipped Dagger and talisman are addressed the way the three armament
	// writers address them: the physical Inventory common row in EquipedItemIndex
	// and the exact GaItem handle of that row in ActiveEquipedItemsGa.
	daggerRowAt := int64(getEquippedSpellsInventoryAt + loadoutEndpointDaggerRow*12)
	put(daggerRowAt, loadoutEndpointDaggerHandle)
	put(daggerRowAt+4, 1)
	put(daggerRowAt+8, uint32(loadoutEndpointDaggerRow))
	put(loadoutEndpointIndexesAt+1*4, 0x180+loadoutEndpointDaggerRow)
	put(loadoutEndpointHandlesAt+1*4, loadoutEndpointDaggerHandle)

	moonRowAt := int64(getEquippedSpellsInventoryAt + loadoutEndpointMoonRow*12)
	put(moonRowAt, loadoutEndpointMoonHandle)
	put(moonRowAt+4, 1)
	put(moonRowAt+8, uint32(loadoutEndpointMoonRow))
	put(loadoutEndpointIndexesAt+17*4, 0x180+loadoutEndpointMoonRow)
	put(loadoutEndpointHandlesAt+17*4, loadoutEndpointMoonHandle)

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
	loaded, err := engine.LoadSave(writeGetCharacterLoadoutFixture(t), "", "local")
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
	// The occupied hand and talisman positions name the exact Inventory common
	// record they reference, and no other position invents one.
	if result.RightHand[0].OwnedItemID == "" ||
		result.RightHand[0].OwnedItemID == result.Talismans[0].OwnedItemID {
		t.Errorf("right hand owned identity = %q", result.RightHand[0].OwnedItemID)
	}
	if result.Talismans[0].OwnedItemID == "" {
		t.Errorf("talisman owned identity = %q", result.Talismans[0].OwnedItemID)
	}
	for _, slot := range append(append([]LoadoutSlot{}, result.LeftHand...),
		append(append([]LoadoutSlot{}, result.Arrows...),
			append(result.Bolts, result.Physick...)...)...) {
		if slot.OwnedItemID != "" {
			t.Errorf("slot %+v must carry no owned identity", slot)
		}
	}
	for index := 1; index < 4; index++ {
		if result.Talismans[index].OwnedItemID != "" {
			t.Errorf("locked talisman %d = %+v", index, result.Talismans[index])
		}
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
	// availableMemorySlots is base 2 + 3 stones + the Moon of Nokstella bonus, so
	// memoryStones must report exactly the 3 stones that capacity was built from.
	if result.Spells[0].State != LoadoutSlotOccupied || result.Spells[0].MemorySlots != 1 ||
		result.ActiveSpellIndex != 0 || result.UsedMemorySlots != 1 ||
		result.AvailableMemorySlots != 7 || result.MemoryStones != 3 ||
		result.UnlockedTalismanSlots != 1 {
		t.Errorf("spell/capacity = %+v active=%d used=%d available=%d stones=%d talismans=%d",
			result.Spells[0], result.ActiveSpellIndex, result.UsedMemorySlots,
			result.AvailableMemorySlots, result.MemoryStones, result.UnlockedTalismanSlots)
	}
}

func TestGetCharacterLoadoutReturnsEmptyGroupsForInactiveSlot(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetCharacterLoadoutFixture(t), "", "local")
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
	anchor := slotBase + loadoutEndpointAnchorAt
	count := binary.LittleEndian.Uint32(data[anchor+getEquippedSpellsCountAt:])
	blockAt := anchor + getEquippedSpellsCountAt + 4 + int64(count)*8
	// The position and the GaItem record it resolves through are moved together,
	// so the save stays internally consistent and the failure under test is the
	// unknown catalog identity rather than a broken reference.
	binary.LittleEndian.PutUint32(data[blockAt+4:], 0x0FFFFFFE)
	binary.LittleEndian.PutUint32(data[slotBase+0x24:], 0x0FFFFFFE)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = GetCharacterLoadout(
		engine, newEquippedSpellsCatalog(t), loaded.SaveSessionID, getEquippedSpellsSlot)
	if err == nil || !strings.Contains(err.Error(), "right hand: slot 0: game ID 0x0FFFFFFE is not a known item") {
		t.Fatalf("error = %v", err)
	}

	// The adjacent boundary: every reference is structurally valid, but the
	// handle of the equipped hand resolves through the GaItem table to a
	// different item than the one the position presents. Reporting the name of
	// one item next to the owned identity of another is exactly the failure this
	// getter must never produce, so the whole read fails instead.
	inconsistent := writeGetCharacterLoadoutFixture(t)
	data, err = os.ReadFile(inconsistent)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	binary.LittleEndian.PutUint32(data[slotBase+0x24:], loadoutEndpointUnarmed)
	if err := os.WriteFile(inconsistent, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine = saveengine.New()
	loaded, err = engine.LoadSave(inconsistent, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = GetCharacterLoadout(
		engine, newEquippedSpellsCatalog(t), loaded.SaveSessionID, getEquippedSpellsSlot)
	if err == nil ||
		!strings.Contains(err.Error(), "equipment position 1: inconsistent existing save state") {
		t.Fatalf("inconsistent handle error = %v", err)
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
	anchor := slotBase + loadoutEndpointAnchorAt
	for index := 0; index < 8; index++ {
		at := anchor + getEquippedSpellsSectionAt + int64(index*getEquippedSpellsRecordSize)
		binary.LittleEndian.PutUint32(data[at:], rawGlintstonePebble)
		binary.LittleEndian.PutUint32(data[at+4:], 0xFFFFFFFF)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "", "local")
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
	anchor := slotBase + loadoutEndpointAnchorAt
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
	loaded, err := engine.LoadSave(path, "", "local")
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
