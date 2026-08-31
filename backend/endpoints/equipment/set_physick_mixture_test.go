package equipment

import (
	"encoding/binary"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	setPhysickMixtureFlaskHandle = uint32(0xB00000FA)
	setPhysickMixtureTearHandle  = uint32(0xB0002AF9)
	setPhysickMixtureTearID      = uint32(0x40002AF9)
)

func loadSetPhysickMixtureSession(t *testing.T) (*saveengine.Engine, string) {
	t.Helper()

	path := writeGetPhysickMixtureFixture(
		t, [2]uint32{saveengine.PhysickEmptyTearID, saveengine.PhysickEmptyTearID})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	slotBase := int64(getPhysickMixtureHeaderSize) + 0x10 +
		getPhysickMixtureSlot*getPhysickMixtureSlotBlockSize
	inventoryAt := slotBase + getPhysickMixtureAnchorAt + 505
	binary.LittleEndian.PutUint32(data[inventoryAt-4:], 1)
	binary.LittleEndian.PutUint32(data[inventoryAt:], setPhysickMixtureFlaskHandle)
	binary.LittleEndian.PutUint32(data[inventoryAt+4:], 1)

	keyAt := inventoryAt + 0xA80*12 + 4
	binary.LittleEndian.PutUint32(data[keyAt-4:], 1)
	binary.LittleEndian.PutUint32(data[keyAt:], setPhysickMixtureTearHandle)
	binary.LittleEndian.PutUint32(data[keyAt+4:], 1)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestSetPhysickMixtureResolvesResourcesAndClearsOnePosition(t *testing.T) {
	engine, sessionID := loadSetPhysickMixtureSession(t)
	resource := &schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "40002AF9"}

	result, err := SetPhysickMixture(
		engine,
		newEquippedSpellsCatalog(t),
		sessionID,
		getPhysickMixtureSlot,
		[]*schema.ResourceRef{nil, resource},
		"0",
	)
	if err != nil {
		t.Fatalf("SetPhysickMixture: %v", err)
	}
	if result.SaveSessionID != sessionID || result.SaveRevision != "1" ||
		result.CharacterID != getPhysickMixtureSlot {
		t.Fatalf("result header = %+v", result)
	}
	wantRefs := [2]*schema.ResourceRef{nil, {Kind: schema.ResourceKindItem, Key: "40002AF9"}}
	if !reflect.DeepEqual(result.CrystalTearResources, wantRefs) {
		t.Errorf("resources = %+v, want %+v", result.CrystalTearResources, wantRefs)
	}

	mixture, err := engine.GetPhysickMixture(sessionID, getPhysickMixtureSlot)
	if err != nil {
		t.Fatalf("GetPhysickMixture: %v", err)
	}
	wantTears := [2]uint32{saveengine.PhysickEmptyTearID, setPhysickMixtureTearID}
	if mixture.Tears != wantTears {
		t.Errorf("tears = %08X/%08X, want %08X/%08X",
			mixture.Tears[0], mixture.Tears[1], wantTears[0], wantTears[1])
	}
}

func TestSetPhysickMixtureRejectsInvalidPublicSelections(t *testing.T) {
	gameCatalog := newEquippedSpellsCatalog(t)
	engine := saveengine.New()

	tests := []struct {
		name      string
		resources []*schema.ResourceRef
		want      string
	}{
		{
			name:      "wrong number of positions",
			resources: []*schema.ResourceRef{{Kind: schema.ResourceKindItem, Key: "40002AF9"}},
			want:      "must contain exactly 2 positions",
		},
		{
			name: "non-tear goods",
			resources: []*schema.ResourceRef{
				{Kind: schema.ResourceKindItem, Key: "4000272E"}, nil,
			},
			want: "no confirmed Physick equipment capability",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := SetPhysickMixture(
				engine, gameCatalog, "unused", 0, testCase.resources, "0")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want containing %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, SetPhysickMixtureResult{}) {
				t.Errorf("result = %+v, want zero value", result)
			}
		})
	}
}

func TestSetPhysickMixtureDefinitionMatchesRuntimeContract(t *testing.T) {
	want := []string{"saveSessionID", "characterID", "crystalTearResources", "expectedRevision"}
	if !reflect.DeepEqual(SetPhysickMixtureDefinition.SupportedResourceVariables, want) {
		t.Errorf("variables = %#v, want %#v",
			SetPhysickMixtureDefinition.SupportedResourceVariables, want)
	}
}
