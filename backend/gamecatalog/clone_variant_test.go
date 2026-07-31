package gamecatalog

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestCloneResourceDeepCopiesFullVariantDocument(t *testing.T) {
	warnings := schema.Fact[[]string]{Value: []string{"warning"}}
	passiveEffects := []schema.WeaponPassiveEffectData{{
		Kind: schema.Fact[string]{Value: "on_hit"},
	}}
	resource := schema.Resource{Item: &schema.ItemDocument{
		Variants: []schema.ItemVariant{{
			Data: schema.VariantDocumentData{
				Flags: schema.Fact[[]string]{Value: []string{"variant"}},
				Weapon: &schema.WeaponData{
					Warnings:       warnings,
					PassiveEffects: passiveEffects,
				},
				SpiritAsh: &schema.SpiritAshData{
					IconID: schema.Fact[uint32]{Value: 1},
				},
			},
			SourceRecords: []schema.ParameterRecord{{
				Fields: []schema.ParameterField{{Name: "nameId"}},
			}},
		}},
	}}

	cloned := cloneResource(resource)
	variant := &cloned.Item.Variants[0]
	variant.Data.Flags.Value[0] = "mutated"
	variant.Data.Weapon.Warnings.Value[0] = "mutated"
	variant.Data.Weapon.PassiveEffects[0].Kind.Value = "mutated"
	variant.Data.SpiritAsh.IconID.Value = 99
	variant.SourceRecords[0].Fields[0].Name = "mutated"

	original := resource.Item.Variants[0]
	if original.Data.Flags.Value[0] != "variant" ||
		original.Data.Weapon.Warnings.Value[0] != "warning" ||
		original.Data.Weapon.PassiveEffects[0].Kind.Value != "on_hit" ||
		original.Data.SpiritAsh.IconID.Value != 1 ||
		original.SourceRecords[0].Fields[0].Name != "nameId" {
		t.Fatal("mutating cloned full variant changed source data")
	}
}
