package schema_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestValidateResourceAcceptsEveryItemFamily(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := testProvenance(manifest)

	tests := []struct {
		name   string
		family schema.ItemFamily
		set    func(*schema.ItemDocument)
	}{
		{"weapon", schema.ItemFamilyWeapon, func(item *schema.ItemDocument) {
			item.Weapon = resources[0].Item.Weapon
		}},
		{"armor", schema.ItemFamilyArmor, func(item *schema.ItemDocument) {
			item.Armor = validArmorData(provenance)
		}},
		{"talisman", schema.ItemFamilyTalisman, func(item *schema.ItemDocument) {
			item.Talisman = validTalismanData(provenance)
		}},
		{"ash_of_war", schema.ItemFamilyAshOfWar, func(item *schema.ItemDocument) {
			item.AshOfWar = validAshOfWarData(provenance)
		}},
		{"spell", schema.ItemFamilySpell, func(item *schema.ItemDocument) {
			item.Spell = validSpellData(provenance)
		}},
		{"spirit_ash", schema.ItemFamilySpiritAsh, func(item *schema.ItemDocument) {
			item.SpiritAsh = validSpiritAshData(provenance)
		}},
		{"goods", schema.ItemFamilyGoods, func(item *schema.ItemDocument) {
			item.Goods = validGoodsData(provenance)
		}},
		{"gesture", schema.ItemFamilyGesture, func(item *schema.ItemDocument) {
			item.Gesture = validGestureData(provenance)
		}},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resource := familyResource(resources[0], uint32(index+10), test.family)
			test.set(resource.Item)
			if err := schema.ValidateResource(resource, sources); err != nil {
				t.Fatalf("ValidateResource: %v", err)
			}
		})
	}
}

func TestValidateResourceRequiresExactlyOneFamilySection(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	resource := resources[0]
	resource.Item.Armor = validArmorData(testProvenance(manifest))

	err := schema.ValidateResource(resource, sources)
	if err == nil || !strings.Contains(err.Error(), "exactly one family data section") {
		t.Fatalf("ValidateResource error = %v, want exactly-one rejection", err)
	}
}

func TestValidateGestureAllowsDuplicateItemAcrossDifferentSlots(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := testProvenance(manifest)
	resource := familyResource(resources[0], 30, schema.ItemFamilyGesture)
	resource.Item.Gesture = validGestureData(provenance)
	resource.Item.Gesture.Slots = append(
		resource.Item.Gesture.Slots,
		schema.GestureSlotRecord{
			SlotID:   knownFact(provenance, uint32(2)),
			ItemID:   knownFact(provenance, uint32(0x401EA7A8)),
			Name:     knownFact(provenance, "Fetal Position"),
			Category: knownFact(provenance, "gesture"),
			Flags:    knownFact(provenance, []string{"cut_content"}),
		},
	)

	if err := schema.ValidateResource(resource, sources); err != nil {
		t.Fatalf("ValidateResource with shared gesture item ID: %v", err)
	}
}

func TestValidateGestureRejectsDuplicateSlotID(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	provenance := testProvenance(manifest)
	resource := familyResource(resources[0], 31, schema.ItemFamilyGesture)
	resource.Item.Gesture = validGestureData(provenance)
	duplicate := resource.Item.Gesture.Slots[0]
	resource.Item.Gesture.Slots = append(resource.Item.Gesture.Slots, duplicate)

	err := schema.ValidateResource(resource, sources)
	if err == nil || !strings.Contains(err.Error(), "duplicate slot ID") {
		t.Fatalf("ValidateResource error = %v, want duplicate slot rejection", err)
	}
}

func familyResource(base schema.Resource, offset uint32, family schema.ItemFamily) schema.Resource {
	resource := base
	item := *base.Item
	item.GameID.Value += offset
	resource.Key = fmt.Sprintf("%08X", item.GameID.Value)
	item.Family.Value = family
	item.Variants = nil
	item.Weapon = nil
	item.Armor = nil
	item.Talisman = nil
	item.AshOfWar = nil
	item.Spell = nil
	item.SpiritAsh = nil
	item.Goods = nil
	item.Gesture = nil
	resource.Item = &item
	return resource
}

func testProvenance(manifest schema.Manifest) schema.Provenance {
	return schema.Provenance{Source: manifest.Sources[0].ID, Method: "test fixture"}
}

func knownFact[T any](provenance schema.Provenance, value T) schema.Fact[T] {
	return schema.Fact[T]{Known: true, Value: value, Provenance: provenance}
}

func validArmorData(p schema.Provenance) *schema.ArmorData {
	return &schema.ArmorData{
		SourceRowID: knownFact(p, uint32(1)), IconIDMale: knownFact(p, uint32(2)),
		IconIDFemale: knownFact(p, uint32(3)), SortID: knownFact(p, uint32(4)),
		SortGroupID: knownFact(p, uint8(5)), Weight: knownFact(p, 1.0),
		Physical: knownFact(p, 1.0), Strike: knownFact(p, 1.0),
		Slash: knownFact(p, 1.0), Pierce: knownFact(p, 1.0),
		Magic: knownFact(p, 1.0), Fire: knownFact(p, 1.0),
		Lightning: knownFact(p, 1.0), Holy: knownFact(p, 1.0),
		Immunity: knownFact(p, uint32(1)), Robustness: knownFact(p, uint32(1)),
		Focus: knownFact(p, uint32(1)), Vitality: knownFact(p, uint32(1)),
		Poise: knownFact(p, 1.0), HeadEquipable: knownFact(p, true),
		BodyEquipable: knownFact(p, false), ArmEquipable: knownFact(p, false),
		LegEquipable: knownFact(p, false),
	}
}

func validTalismanData(p schema.Provenance) *schema.TalismanData {
	return &schema.TalismanData{
		SourceRowID: knownFact(p, uint32(1)), IconID: knownFact(p, uint32(2)),
		SortID: knownFact(p, uint32(3)), SortGroupID: knownFact(p, uint8(4)),
		Weight: knownFact(p, 1.0),
	}
}

func validAshOfWarData(p schema.Provenance) *schema.AshOfWarData {
	return &schema.AshOfWarData{
		SourceRowID:          knownFact(p, uint32(1)),
		IconID:               knownFact(p, uint32(2)),
		SortID:               knownFact(p, uint32(3)),
		SortGroupID:          knownFact(p, uint8(4)),
		SwordArtsParamID:     knownFact(p, int32(5)),
		SwordArtsName:        knownFact(p, "Test skill"),
		DefaultAffinity:      knownFact(p, uint8(0)),
		CompatibilityMask:    knownFact(p, uint64(1)),
		CompatibleClassNames: knownFact(p, []string{"dagger"}),
	}
}

func validSpellData(p schema.Provenance) *schema.SpellData {
	return &schema.SpellData{
		SourceRowID: knownFact(p, uint32(1)), IconID: knownFact(p, uint32(2)),
		SortID: knownFact(p, uint32(3)), FPCost: knownFact(p, uint32(4)),
		StaminaCost: knownFact(p, uint32(5)), MemorySlots: knownFact(p, uint8(1)),
		RequiredIntelligence: knownFact(p, uint32(6)), RequiredFaith: knownFact(p, uint32(7)),
		RequiredArcane: knownFact(p, uint32(8)),
	}
}

func validSpiritAshData(p schema.Provenance) *schema.SpiritAshData {
	return &schema.SpiritAshData{
		SourceRowID: knownFact(p, uint32(1)), IconID: knownFact(p, uint32(2)),
		SortID: knownFact(p, uint32(3)), SortGroupID: knownFact(p, uint8(4)),
		ReinforceGoodsID: knownFact(p, int32(3)), ReinforceMaterialID: knownFact(p, int32(4)),
		ReinforcePrice: knownFact(p, uint32(5)),
	}
}

func validGoodsData(p schema.Provenance) *schema.GoodsData {
	return &schema.GoodsData{
		SourceRowID: knownFact(p, uint32(1)), IconID: knownFact(p, uint32(2)),
		SortID: knownFact(p, uint32(3)), SortGroupID: knownFact(p, uint8(4)),
		GoodsType: knownFact(p, uint16(5)), Weight: knownFact(p, 1.0),
		MaxQuantity: knownFact(p, uint32(10)), MaxRepository: knownFact(p, uint32(20)),
		TutorialFlagID: knownFact(p, uint32(30)), IsEquipable: knownFact(p, true),
		IsConsumable: knownFact(p, true), IsDiscardable: knownFact(p, true),
		IsDepositable: knownFact(p, true), IsDroppable: knownFact(p, true),
	}
}

func validGestureData(p schema.Provenance) *schema.GestureData {
	return &schema.GestureData{
		GoodsSourceRowID: knownFact(p, uint32(1)),
		IconID:           knownFact(p, uint32(2)),
		Slots: []schema.GestureSlotRecord{{
			SlotID:   knownFact(p, uint32(1)),
			ItemID:   knownFact(p, uint32(0x401EA7A8)),
			Name:     knownFact(p, "The Carian Oath"),
			Category: knownFact(p, "gesture"),
			Flags:    knownFact(p, []string{"cut_content"}),
		}},
	}
}
