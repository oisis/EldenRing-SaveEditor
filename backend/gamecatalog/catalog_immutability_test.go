package gamecatalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestCatalogQueriesReturnIndependentCopies(t *testing.T) {
	manifest, resources := prototype.Data()
	p := schema.Provenance{Source: manifest.Sources[0].ID, Method: "test fixture"}
	daggerIndex := -1
	for index := range resources {
		if resources[index].Item != nil &&
			resources[index].Item.GameID.Value == prototype.DaggerGameID {
			daggerIndex = index
			break
		}
	}
	if daggerIndex < 0 {
		t.Fatal("Dagger fixture not found")
	}
	resources[daggerIndex].Item.Flags = schema.Fact[[]string]{
		Known: true, Value: []string{"stackable"}, Provenance: p,
	}
	resources[daggerIndex].Item.Aliases = []schema.ItemAlias{{
		GameID: schema.Fact[uint32]{Known: true, Value: 0x7F000001, Provenance: p},
		SourceRecords: []schema.ParameterRecord{{
			Table: "EquipParamWeapon", RowID: 1, Provenance: p,
			Fields: []schema.ParameterField{{Name: "nameId"}},
		}},
	}}
	catalog, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}

	first, ok := catalog.ItemByGameID(prototype.DaggerGameID)
	if !ok {
		t.Fatal("Dagger not found")
	}
	first.Label.Value = "Mutated"
	first.Item.Capabilities.Infusion.Rules.AllowedAffinities[0] = schema.AffinityOccult
	originalVariantID := first.Item.Variants[0].GameID.Value
	first.Item.Variants[0].GameID.Value = 1
	first.Item.Flags.Value[0] = "mutated"
	first.Item.Aliases[0].SourceRecords[0].Fields[0].Name = "mutated"

	second, ok := catalog.ItemByGameID(prototype.DaggerGameID)
	if !ok {
		t.Fatal("Dagger not found after mutation")
	}
	if second.Label.Value != "Dagger" {
		t.Errorf("catalog label was mutated to %q", second.Label.Value)
	}
	if second.Item.Capabilities.Infusion.Rules.AllowedAffinities[0] != schema.AffinityStandard {
		t.Error("catalog affinity slice was mutated")
	}
	if second.Item.Variants[0].GameID.Value != originalVariantID {
		t.Error("catalog variant slice was mutated")
	}
	if second.Item.Flags.Value[0] != "stackable" {
		t.Error("catalog legacy flags were mutated")
	}
	if second.Item.Aliases[0].SourceRecords[0].Fields[0].Name != "nameId" {
		t.Error("catalog parameter field name was mutated")
	}

	manifestCopy := catalog.Manifest()
	manifestCopy.Sources[0].Location = "mutated"
	if catalog.Manifest().Sources[0].Location == "mutated" {
		t.Error("catalog manifest was mutated")
	}
}
