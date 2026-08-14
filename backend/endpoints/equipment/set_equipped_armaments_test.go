package equipment

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	setArmamentsEndpointSlot         = 0
	setArmamentsEndpointSlotBase     = 0x310
	setArmamentsEndpointAnchorAt     = 0xA07B
	setArmamentsEndpointInventoryAt  = setArmamentsEndpointSlotBase + setArmamentsEndpointAnchorAt + 505
	setArmamentsEndpointIndexesAt    = setArmamentsEndpointSlotBase + setArmamentsEndpointAnchorAt + 0xD1
	setArmamentsEndpointItemIDsAt    = setArmamentsEndpointSlotBase + setArmamentsEndpointAnchorAt + 0x145
	setArmamentsEndpointHandlesAt    = setArmamentsEndpointSlotBase + setArmamentsEndpointAnchorAt + 0x19D
	setArmamentsEndpointDynamicAt    = setArmamentsEndpointSlotBase + setArmamentsEndpointAnchorAt + 0x931D + 4
	setArmamentsEndpointRecordStride = 12
	setArmamentsEndpointWeaponID     = uint32(0x000F4240)
	setArmamentsEndpointUnarmedID    = uint32(0x0001ADB0)
)

var setArmamentsEndpointAnchor = []byte{
	0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

func writeSetEquippedArmamentsEndpointFixture(t *testing.T, selectedGameID uint32) string {
	t.Helper()
	const fixtureSize = int64(0x300) + 10*0x280010 + 0x60010
	data := make([]byte, fixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[0x0C:], 12)
	data[int64(0x300)+10*0x280010+0x10+0x1954+setArmamentsEndpointSlot] = 1
	binary.LittleEndian.PutUint32(data[setArmamentsEndpointSlotBase:], 83)

	for index := 0; index < 7; index++ {
		handle := uint32(0x80000100 + index)
		gameID := selectedGameID
		if index == 0 {
			gameID = setArmamentsEndpointUnarmedID
		}
		position := int64(setArmamentsEndpointSlotBase + 0x20 + index*21)
		binary.LittleEndian.PutUint32(data[position:], handle)
		binary.LittleEndian.PutUint32(data[position+4:], gameID)
		rowAt := int64(setArmamentsEndpointInventoryAt + index*setArmamentsEndpointRecordStride)
		binary.LittleEndian.PutUint32(data[rowAt:], handle)
		binary.LittleEndian.PutUint32(data[rowAt+4:], 1)
		binary.LittleEndian.PutUint32(data[rowAt+8:], uint32(index+1))
	}
	copy(data[setArmamentsEndpointSlotBase+setArmamentsEndpointAnchorAt:],
		setArmamentsEndpointAnchor)

	for slot := 0; slot < 6; slot++ {
		binary.LittleEndian.PutUint32(data[setArmamentsEndpointIndexesAt+slot*4:], 0x180)
		binary.LittleEndian.PutUint32(
			data[setArmamentsEndpointItemIDsAt+slot*4:], setArmamentsEndpointUnarmedID)
		binary.LittleEndian.PutUint32(data[setArmamentsEndpointHandlesAt+slot*4:], 0x80000100)
		binary.LittleEndian.PutUint32(
			data[setArmamentsEndpointDynamicAt+slot*4:], setArmamentsEndpointUnarmedID)
	}

	path := filepath.Join(t.TempDir(), "set-equipped-armaments.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestSetEquippedArmamentsDefinitionMatchesRuntimeContract(t *testing.T) {
	want := []string{"saveSessionID", "characterID", "slotAssignments", "expectedRevision"}
	if !reflect.DeepEqual(SetEquippedArmamentsDefinition.SupportedResourceVariables, want) {
		t.Errorf("variables = %#v, want %#v",
			SetEquippedArmamentsDefinition.SupportedResourceVariables, want)
	}
}

func TestSetEquippedArmamentsValidatesCatalogAndCommits(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeSetEquippedArmamentsEndpointFixture(t, setArmamentsEndpointWeaponID), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmamentsEndpointSlot, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(inventory.Records) != 7 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}

	assignments := make([]*string, 6)
	for slot := range assignments {
		token := inventory.Records[slot+1].OwnedItemID
		assignments[slot] = &token
	}
	result, err := SetEquippedArmaments(
		engine, newPouchCatalog(t), loaded.SaveSessionID,
		setArmamentsEndpointSlot, assignments, "0")
	if err != nil {
		t.Fatalf("SetEquippedArmaments: %v", err)
	}
	if result.SaveRevision != "1" || result.CharacterID != setArmamentsEndpointSlot {
		t.Fatalf("result = %+v", result)
	}
	want := EquippedArmamentAssignment{
		Kind: schema.ResourceKindItem, Key: "000F4240", GameID: setArmamentsEndpointWeaponID}
	for slot, assignment := range result.SlotAssignments {
		if assignment == nil || *assignment != want {
			t.Errorf("slotAssignments[%d] = %+v, want %+v", slot, assignment, want)
		}
	}
}

func TestSetEquippedArmamentsRejectsInvalidRequestAndCatalogData(t *testing.T) {
	engine := saveengine.New()
	catalog := newPouchCatalog(t)
	if _, err := SetEquippedArmaments(
		nil, catalog, "unused", 0, make([]*string, 6), "0"); err == nil {
		t.Fatal("SetEquippedArmaments accepted a nil engine")
	}
	if _, err := SetEquippedArmaments(
		engine, nil, "unused", 0, make([]*string, 6), "0"); err == nil {
		t.Fatal("SetEquippedArmaments accepted a nil catalog")
	}
	if _, err := SetEquippedArmaments(
		engine, catalog, "unused", 0, make([]*string, 5), "0"); err == nil {
		t.Fatal("SetEquippedArmaments accepted five positions")
	}

	const disabledEquipmentGameID = uint32(0x02FAF080)
	loaded, err := engine.LoadSave(
		writeSetEquippedArmamentsEndpointFixture(t, disabledEquipmentGameID), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	token := inventory.Records[1].OwnedItemID
	assignments := make([]*string, 6)
	assignments[0] = &token
	if _, err := SetEquippedArmaments(
		engine, catalog, loaded.SaveSessionID, 0, assignments, "0"); err == nil ||
		!strings.Contains(err.Error(), "no confirmed hand equipment capability") {
		t.Fatalf("catalog capability error = %v", err)
	}
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil || info.UnsavedChanges {
		t.Fatalf("rejected mutation changed session: %+v, %v", info, err)
	}
}
