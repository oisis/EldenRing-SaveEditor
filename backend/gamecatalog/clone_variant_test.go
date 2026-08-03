package gamecatalog

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestCloneResourceDeepCopiesFullVariantDocument(t *testing.T) {
	saveForgeStorage := schema.Fact[uint32]{Value: 1}
	saveForgeWeight := schema.Fact[float64]{Value: 2}
	saveForgeEndurance := schema.Fact[int32]{Value: 3}
	warnings := schema.Fact[[]string]{Value: []string{"warning"}}
	passiveEffects := []schema.WeaponPassiveEffectData{{
		Kind: schema.Fact[string]{Value: "on_hit"},
	}}
	resource := schema.Resource{Item: &schema.ItemDocument{
		Variants: []schema.ItemVariant{{
			Data: schema.VariantDocumentData{
				Flags: schema.Fact[[]string]{Value: []string{"variant"}},
				Storage: schema.ItemStorage{
					SafeModeMaxInventory: &saveForgeStorage,
					MaxInventorySFV:      &saveForgeStorage,
				},
				Modifiers: schema.ItemModifiers{
					EquipLoad: &schema.EquipLoadModifier{
						EnduranceBonusSFV: &saveForgeEndurance,
					},
				},
				Weapon: &schema.WeaponData{
					Warnings:       warnings,
					PassiveEffects: passiveEffects,
					WeightSFV:      &saveForgeWeight,
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
	variant.Data.Storage.SafeModeMaxInventory.Value = 98
	variant.Data.Storage.MaxInventorySFV.Value = 99
	variant.Data.Modifiers.EquipLoad.EnduranceBonusSFV.Value = 99
	variant.Data.Weapon.Warnings.Value[0] = "mutated"
	variant.Data.Weapon.PassiveEffects[0].Kind.Value = "mutated"
	variant.Data.Weapon.WeightSFV.Value = 99
	variant.Data.SpiritAsh.IconID.Value = 99
	variant.SourceRecords[0].Fields[0].Name = "mutated"

	original := resource.Item.Variants[0]
	if original.Data.Flags.Value[0] != "variant" ||
		original.Data.Storage.SafeModeMaxInventory.Value != 1 ||
		original.Data.Storage.MaxInventorySFV.Value != 1 ||
		original.Data.Modifiers.EquipLoad.EnduranceBonusSFV.Value != 3 ||
		original.Data.Weapon.Warnings.Value[0] != "warning" ||
		original.Data.Weapon.PassiveEffects[0].Kind.Value != "on_hit" ||
		original.Data.Weapon.WeightSFV.Value != 2 ||
		original.Data.SpiritAsh.IconID.Value != 1 ||
		original.SourceRecords[0].Fields[0].Name != "nameId" {
		t.Fatal("mutating cloned full variant changed source data")
	}
}
