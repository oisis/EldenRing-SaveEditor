package equipment

import (
	"encoding/binary"
	"os"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func writeSetEquippedTalismansEndpointFixture(t *testing.T) (string, string) {
	t.Helper()
	path, platform := writeSetPouchEndpointFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	anchor := int64(0x310 + 0x10020)
	for _, blockAt := range []int64{0x115, 0x189, 0x1E1} {
		for index := 0; index < 4; index++ {
			value := uint32(0xFFFFFFFF)
			if blockAt == 0x1E1 {
				value = 0
			}
			binary.LittleEndian.PutUint32(data[anchor+blockAt+int64(index*4):], value)
		}
	}
	countAt := anchor + 0x931D
	armamentsAt := countAt + 4 + 17*8
	for index := 17; index <= 20; index++ {
		binary.LittleEndian.PutUint32(data[armamentsAt+int64(index*4):], 0xFFFFFFFF)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path, platform
}

func TestSetEquippedTalismansDefinitionMatchesRuntimeContract(t *testing.T) {
	want := []string{"saveSessionID", "characterID", "orderedOwnedItemIDs", "expectedRevision"}
	if !reflect.DeepEqual(SetEquippedTalismansDefinition.SupportedResourceVariables, want) {
		t.Errorf("variables = %#v, want %#v",
			SetEquippedTalismansDefinition.SupportedResourceVariables, want)
	}
}

func TestSetEquippedTalismansValidatesCatalogAndCommits(t *testing.T) {
	path, platform := writeSetEquippedTalismansEndpointFixture(t)
	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, platform)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(inventory.Records) < 3 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}

	result, err := SetEquippedTalismans(
		engine,
		newPouchCatalog(t),
		loaded.SaveSessionID,
		0,
		[]string{inventory.Records[2].OwnedItemID},
		"0",
	)
	if err != nil {
		t.Fatalf("SetEquippedTalismans: %v", err)
	}
	if result.SaveRevision != "1" || result.UnlockedSlots != 1 ||
		len(result.OrderedResources) != 1 ||
		result.OrderedResources[0] != (schema.ResourceRef{Kind: "item", Key: "20000474"}) {
		t.Fatalf("result = %+v", result)
	}
}
