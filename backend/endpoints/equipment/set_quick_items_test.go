package equipment

import (
	"encoding/binary"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func writeSetQuickItemsEndpointFixture(t *testing.T) (string, string) {
	t.Helper()
	path, platform := writeSetPouchEndpointFixture(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	anchorBase := int64(0x310 + 0x10020)
	pairAt := anchorBase + 0x9279
	for i := 0; i < 10; i++ {
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8:], 0)
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8+4:], 0xFFFFFFFF)
	}
	countAt := anchorBase + 0x931D
	tailAt := countAt + 4 + 17*8 + 0x58
	for i := 0; i < 10; i++ {
		binary.LittleEndian.PutUint32(data[tailAt+int64(i)*4:], 0xFFFFFFFF)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path, platform
}

func TestSetQuickItemsDefinitionMatchesRuntimeContract(t *testing.T) {
	want := []string{"saveSessionID", "characterID", "slotAssignments", "expectedRevision"}
	if !reflect.DeepEqual(SetQuickItemsDefinition.SupportedResourceVariables, want) {
		t.Errorf("variables = %#v, want %#v",
			SetQuickItemsDefinition.SupportedResourceVariables, want)
	}
}

func TestSetQuickItemsRejectsInvalidSelections(t *testing.T) {
	engine := saveengine.New()
	catalog := &gamecatalog.Catalog{}
	tests := []struct {
		name        string
		engine      *saveengine.Engine
		catalog     *gamecatalog.Catalog
		assignments []*string
		want        string
	}{
		{"nil engine", nil, catalog, make([]*string, 10), "save engine is not available"},
		{"nil catalog", engine, nil, make([]*string, 10), "game catalog is not available"},
		{"wrong position count", engine, catalog, make([]*string, 9), "exactly 10 positions"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := SetQuickItems(
				testCase.engine, testCase.catalog, "unused", 0,
				testCase.assignments, "0")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want containing %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, SetQuickItemsResult{}) {
				t.Errorf("result = %+v, want zero", result)
			}
		})
	}
}

func TestSetQuickItemsValidatesCatalogAndCommits(t *testing.T) {
	path, platform := writeSetQuickItemsEndpointFixture(t)
	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, platform, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := newPouchCatalog(t)
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(inventory.Records) < 3 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}

	for _, testCase := range []struct {
		name  string
		token string
		want  string
	}{
		{"goods without capability", inventory.Records[1].OwnedItemID, "quick-item"},
		{"wrong family", inventory.Records[2].OwnedItemID, "item family"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assignments := make([]*string, 10)
			assignments[0] = &testCase.token
			result, err := SetQuickItems(
				engine, catalog, loaded.SaveSessionID, 0, assignments, "0")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want containing %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, SetQuickItemsResult{}) {
				t.Errorf("result = %+v, want zero", result)
			}
		})
	}

	token := inventory.Records[0].OwnedItemID
	assignments := make([]*string, 10)
	assignments[0] = &token
	result, err := SetQuickItems(
		engine, catalog, loaded.SaveSessionID, 0, assignments, "0")
	if err != nil {
		t.Fatalf("SetQuickItems: %v", err)
	}
	if result.SaveRevision != "1" || result.CharacterID != 0 ||
		result.SlotAssignments[0] == nil ||
		result.SlotAssignments[0].Kind != schema.ResourceKindItem ||
		result.SlotAssignments[0].Key != "400006A4" {
		t.Fatalf("result = %+v", result)
	}
	for i := 1; i < 10; i++ {
		if result.SlotAssignments[i] != nil {
			t.Errorf("slotAssignments[%d] = %+v, want nil", i, result.SlotAssignments[i])
		}
	}
}
