package equipment

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func newPouchCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("loader.LoadFS: %v", err)
	}
	cat, err := gamecatalog.New(data.Manifest, data.Resources())
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	return cat
}

func writeSetPouchEndpointFixture(t *testing.T) (string, string) {
	t.Helper()
	// Create PC fixture with item 0x400006A4 (Throwing Dagger, handle 0xB00006A4) in inventory common
	header := make([]byte, 0x300)
	copy(header, []byte("BND4"))
	binary.LittleEndian.PutUint32(header[0x0C:], 12)

	path := filepath.Join(t.TempDir(), "save_pouch_endpoint.sl2")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := file.Write(header); err != nil {
		t.Fatalf("Write: %v", err)
	}
	fixtureSize := int64(0x300) + 10*0x280010 + 0x60010
	if err := file.Truncate(fixtureSize); err != nil {
		t.Fatalf("Truncate: %v", err)
	}
	file.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	slotBase := int64(0x300) + 0x10
	data[int64(0x300)+10*0x280010+0x10+0x1954] = 1 // active character 0
	binary.LittleEndian.PutUint32(data[slotBase:], 82)

	anchorAt := int64(0x10020)
	pouchAnchor := []byte{
		0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	copy(data[slotBase+anchorAt:], pouchAnchor)

	pairAt := slotBase + anchorAt + 0x92CD
	for i := 0; i < 6; i++ {
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8:], 0)
		binary.LittleEndian.PutUint32(data[pairAt+int64(i)*8+4:], 0xFFFFFFFF)
	}

	countAt := slotBase + anchorAt + 0x931D
	binary.LittleEndian.PutUint32(data[countAt:], 17)

	tailAt := countAt + 4 + 17*8 + 0x80
	for i := 0; i < 6; i++ {
		binary.LittleEndian.PutUint32(data[tailAt+int64(i)*4:], 0xFFFFFFFF)
	}

	inventoryAt := slotBase + anchorAt + 505
	binary.LittleEndian.PutUint32(data[inventoryAt-4:], 3)
	// Row 0: Throwing Dagger (goods with pouch capability)
	binary.LittleEndian.PutUint32(data[inventoryAt:], 0xB00006A4)
	binary.LittleEndian.PutUint32(data[inventoryAt+4:], 10)
	binary.LittleEndian.PutUint32(data[inventoryAt+8:], 1)
	// Row 1: Memory Stone 0x4000272E (goods without pouch capability)
	binary.LittleEndian.PutUint32(data[inventoryAt+12:], 0xB000272E)
	binary.LittleEndian.PutUint32(data[inventoryAt+16:], 1)
	binary.LittleEndian.PutUint32(data[inventoryAt+20:], 2)
	// Row 2: Moon of Nokstella 0x20000474 (talisman, wrong item family)
	binary.LittleEndian.PutUint32(data[inventoryAt+24:], 0xA0000474)
	binary.LittleEndian.PutUint32(data[inventoryAt+28:], 1)
	binary.LittleEndian.PutUint32(data[inventoryAt+32:], 3)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	return path, "pc"
}

func TestSetPouchItemsDefinitionMatchesRuntimeContract(t *testing.T) {
	want := []string{"saveSessionID", "characterID", "slotAssignments", "expectedRevision"}
	if !reflect.DeepEqual(SetPouchItemsDefinition.SupportedResourceVariables, want) {
		t.Errorf("variables = %#v, want %#v",
			SetPouchItemsDefinition.SupportedResourceVariables, want)
	}
}

func TestSetPouchItemsRejectsInvalidSelections(t *testing.T) {
	engine := saveengine.New()
	catalog := &gamecatalog.Catalog{}

	tests := []struct {
		name        string
		engine      *saveengine.Engine
		catalog     *gamecatalog.Catalog
		assignments []*string
		want        string
	}{
		{
			name:        "nil engine",
			engine:      nil,
			catalog:     catalog,
			assignments: make([]*string, 6),
			want:        "save engine is not available",
		},
		{
			name:        "nil catalog",
			engine:      engine,
			catalog:     nil,
			assignments: make([]*string, 6),
			want:        "game catalog is not available",
		},
		{
			name:        "wrong position count",
			engine:      engine,
			catalog:     catalog,
			assignments: make([]*string, 5),
			want:        "slotAssignments must contain exactly 6 positions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SetPouchItems(tt.engine, tt.catalog, "unused", 0, tt.assignments, "0")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
			if !reflect.DeepEqual(result, SetPouchItemsResult{}) {
				t.Errorf("result = %+v, want zero value", result)
			}
		})
	}
}

func TestSetPouchItemsRejectsInvalidCatalogData(t *testing.T) {
	path, platform := writeSetPouchEndpointFixture(t)
	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, platform, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	cat := newPouchCatalog(t)
	inv, err := engine.GetInventory(loaded.SaveSessionID, 0, "common", 1, 50)
	if err != nil || len(inv.Records) < 3 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inv.Records))
	}

	tokNoPouchCap := inv.Records[1].OwnedItemID
	tokWrongFamily := inv.Records[2].OwnedItemID

	tests := []struct {
		name      string
		token     string
		wantError string
	}{
		{
			name:      "goods without pouch capability",
			token:     tokNoPouchCap,
			wantError: "pouch",
		},
		{
			name:      "wrong item family",
			token:     tokWrongFamily,
			wantError: "item family",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignments := []*string{&tt.token, nil, nil, nil, nil, nil}
			result, err := SetPouchItems(engine, cat, loaded.SaveSessionID, 0, assignments, "0")
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want error containing %q", err, tt.wantError)
			}
			if !reflect.DeepEqual(result, SetPouchItemsResult{}) {
				t.Errorf("result = %+v, want zero value", result)
			}
		})
	}
}

func TestSetPouchItemsSuccessWithCatalog(t *testing.T) {
	path, platform := writeSetPouchEndpointFixture(t)
	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, platform, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	cat := newPouchCatalog(t)
	inv, err := engine.GetInventory(loaded.SaveSessionID, 0, "common", 1, 50)
	if err != nil || len(inv.Records) == 0 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inv.Records))
	}
	tok := inv.Records[0].OwnedItemID

	assignments := []*string{&tok, nil, nil, nil, nil, nil}
	result, err := SetPouchItems(engine, cat, loaded.SaveSessionID, 0, assignments, "0")
	if err != nil {
		t.Fatalf("SetPouchItems: %v", err)
	}
	if result.SaveSessionID != loaded.SaveSessionID || result.SaveRevision != "1" || result.CharacterID != 0 {
		t.Fatalf("result = %+v", result)
	}
	if result.SlotAssignments[0] == nil || result.SlotAssignments[0].Kind != schema.ResourceKindItem || result.SlotAssignments[0].Key != "400006A4" {
		t.Errorf("slotAssignments[0] = %+v, want item 400006A4", result.SlotAssignments[0])
	}
	for i := 1; i < 6; i++ {
		if result.SlotAssignments[i] != nil {
			t.Errorf("slotAssignments[%d] = %+v, want nil", i, result.SlotAssignments[i])
		}
	}
}
