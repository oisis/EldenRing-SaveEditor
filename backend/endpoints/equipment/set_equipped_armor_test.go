package equipment

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	setArmorEndpointSlot         = 0
	setArmorEndpointSlotBase     = 0x310
	setArmorEndpointAnchorAt     = 0xA060
	setArmorEndpointInventoryAt  = setArmorEndpointSlotBase + setArmorEndpointAnchorAt + 505
	setArmorEndpointIndexesAt    = setArmorEndpointSlotBase + setArmorEndpointAnchorAt + 0x101
	setArmorEndpointItemIDsAt    = setArmorEndpointSlotBase + setArmorEndpointAnchorAt + 0x175
	setArmorEndpointHandlesAt    = setArmorEndpointSlotBase + setArmorEndpointAnchorAt + 0x1CD
	setArmorEndpointArmamentsAt  = setArmorEndpointSlotBase + setArmorEndpointAnchorAt + 0x931D + 4
	setArmorEndpointInventoryRow = 12
)

var (
	setArmorEndpointEmptyIDs  = [4]uint32{0x10002710, 0x10002774, 0x100027D8, 0x1000283C}
	setArmorEndpointActualIDs = [4]uint32{0x10009C40, 0x10009CA4, 0x10009D08, 0x10009D6C}
	setArmorEndpointAnchor    = []byte{
		0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
)

func writeSetEquippedArmorEndpointFixture(t *testing.T) string {
	t.Helper()
	const fixtureSize = int64(0x300) + 10*0x280010 + 0x60010
	data := make([]byte, fixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[0x0C:], 12)
	data[int64(0x300)+10*0x280010+0x10+0x1954+setArmorEndpointSlot] = 1
	binary.LittleEndian.PutUint32(data[setArmorEndpointSlotBase:], 83)

	gameIDs := append(setArmorEndpointEmptyIDs[:], setArmorEndpointActualIDs[:]...)
	position := int64(setArmorEndpointSlotBase + 0x20)
	for index, gameID := range gameIDs {
		handle := uint32(0x90000100 + index)
		binary.LittleEndian.PutUint32(data[position:], handle)
		binary.LittleEndian.PutUint32(data[position+4:], gameID)
		position += 16
		rowAt := int64(setArmorEndpointInventoryAt + index*setArmorEndpointInventoryRow)
		binary.LittleEndian.PutUint32(data[rowAt:], handle)
		binary.LittleEndian.PutUint32(data[rowAt+4:], 1)
		binary.LittleEndian.PutUint32(data[rowAt+8:], uint32(index+1))
	}
	copy(data[setArmorEndpointSlotBase+setArmorEndpointAnchorAt:], setArmorEndpointAnchor)

	for slot, gameID := range setArmorEndpointEmptyIDs {
		handle := uint32(0x90000100 + slot)
		binary.LittleEndian.PutUint32(data[setArmorEndpointIndexesAt+slot*4:], 0x180+uint32(slot))
		binary.LittleEndian.PutUint32(data[setArmorEndpointItemIDsAt+slot*4:], gameID&0x0FFFFFFF)
		binary.LittleEndian.PutUint32(data[setArmorEndpointHandlesAt+slot*4:], handle)
		binary.LittleEndian.PutUint32(data[setArmorEndpointArmamentsAt+(12+slot)*4:], gameID)
	}

	path := filepath.Join(t.TempDir(), "set-equipped-armor.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestSetEquippedArmorDefinitionMatchesRuntimeContract(t *testing.T) {
	want := []string{"saveSessionID", "characterID", "slotAssignments", "expectedRevision"}
	if !reflect.DeepEqual(SetEquippedArmorDefinition.SupportedResourceVariables, want) {
		t.Errorf("variables = %#v, want %#v",
			SetEquippedArmorDefinition.SupportedResourceVariables, want)
	}
}

func TestSetEquippedArmorValidatesCatalogAndCommits(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSetEquippedArmorEndpointFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmorEndpointSlot, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(inventory.Records) != 8 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}

	assignments := make([]*string, 4)
	for slot := range assignments {
		token := inventory.Records[slot+4].OwnedItemID
		assignments[slot] = &token
	}
	result, err := SetEquippedArmor(
		engine, newPouchCatalog(t), loaded.SaveSessionID,
		setArmorEndpointSlot, assignments, "0")
	if err != nil {
		t.Fatalf("SetEquippedArmor: %v", err)
	}
	if result.SaveRevision != "1" || result.CharacterID != setArmorEndpointSlot {
		t.Fatalf("result = %+v", result)
	}
	for slot, gameID := range setArmorEndpointActualIDs {
		want := schema.ResourceRef{Kind: schema.ResourceKindItem, Key: fmt.Sprintf("%08X", gameID)}
		if result.SlotAssignments[slot] == nil || *result.SlotAssignments[slot] != want {
			t.Errorf("slotAssignments[%d] = %+v, want %+v", slot, result.SlotAssignments[slot], want)
		}
	}
}

func TestSetEquippedArmorRejectsWrongSlotAndInvalidDependencies(t *testing.T) {
	engine := saveengine.New()
	catalog := newPouchCatalog(t)
	if _, err := SetEquippedArmor(nil, catalog, "unused", 0, make([]*string, 4), "0"); err == nil {
		t.Fatal("SetEquippedArmor accepted a nil engine")
	}
	if _, err := SetEquippedArmor(engine, nil, "unused", 0, make([]*string, 4), "0"); err == nil {
		t.Fatal("SetEquippedArmor accepted a nil catalog")
	}
	if _, err := SetEquippedArmor(engine, catalog, "unused", 0, make([]*string, 3), "0"); err == nil {
		t.Fatal("SetEquippedArmor accepted three positions")
	}

	loaded, err := engine.LoadSave(writeSetEquippedArmorEndpointFixture(t), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	headToken := inventory.Records[4].OwnedItemID
	assignments := make([]*string, 4)
	assignments[1] = &headToken
	if _, err := SetEquippedArmor(
		engine, catalog, loaded.SaveSessionID, 0, assignments, "0"); err == nil ||
		!strings.Contains(err.Error(), "cannot be equipped in slot \"chest\"") {
		t.Fatalf("wrong-slot error = %v", err)
	}
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil || info.UnsavedChanges {
		t.Fatalf("rejected mutation changed session: %+v, %v", info, err)
	}
}
