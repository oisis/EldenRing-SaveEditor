package schema_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestValidateResourceAcceptsFullTypedVariantDocuments(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	p := testProvenance(manifest)
	resource := resources[0]
	resource.Item.Variants = []schema.ItemVariant{
		validVariant(p, resources[0].Item, schema.ItemVariantAffinity, 100, true, false),
		validVariant(p, resources[0].Item, schema.ItemVariantUpgrade, 101, false, true),
		validVariant(p, resources[0].Item, schema.ItemVariantAffinityUpgrade, 102, true, true),
	}

	if err := schema.ValidateResource(resource, sources); err != nil {
		t.Fatalf("ValidateResource: %v", err)
	}
}

func TestValidateResourceRejectsVariantMissingKindField(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	p := testProvenance(manifest)
	resource := resources[0]
	variant := validVariant(p, resources[0].Item, schema.ItemVariantUpgrade, 101, false, true)
	variant.UpgradeLevel = schema.Fact[uint8]{}
	resource.Item.Variants = []schema.ItemVariant{variant}

	err := schema.ValidateResource(resource, sources)
	if err == nil || !strings.Contains(err.Error(), "upgradeLevel") {
		t.Fatalf("ValidateResource error = %v, want missing upgrade-level rejection", err)
	}
}

func TestValidateResourceRejectsVariantMissingFamilyData(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	p := testProvenance(manifest)
	resource := resources[0]
	variant := validVariant(p, resources[0].Item, schema.ItemVariantAffinity, 100, true, false)
	variant.Data = schema.VariantDocumentData{}
	resource.Item.Variants = []schema.ItemVariant{variant}

	err := schema.ValidateResource(resource, sources)
	if err == nil || !strings.Contains(err.Error(), "family must be known") {
		t.Fatalf("ValidateResource error = %v, want missing family-data rejection", err)
	}
}

func TestValidateResourceRejectsRawRecordWithoutProvenance(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	p := testProvenance(manifest)
	resource := resources[0]
	variant := validVariant(p, resources[0].Item, schema.ItemVariantAffinity, 100, true, false)
	variant.SourceRecords[0].Provenance = schema.Provenance{}
	resource.Item.Variants = []schema.ItemVariant{variant}

	err := schema.ValidateResource(resource, sources)
	if err == nil || !strings.Contains(err.Error(), "provenance source is required") {
		t.Fatalf("ValidateResource error = %v, want raw-record provenance rejection", err)
	}
}

func validVariant(
	p schema.Provenance,
	item *schema.ItemDocument,
	kind schema.ItemVariantKind,
	gameID uint32,
	withAffinity bool,
	withUpgrade bool,
) schema.ItemVariant {
	item.Weapon = validExtendedWeaponData(p, *item.Weapon)
	variantWeapon := *item.Weapon
	variantWeapon.AttackPhysical = schema.Fact[int32]{
		Known:      true,
		Value:      75,
		Provenance: p,
	}
	variant := schema.ItemVariant{
		GameID:      knownFact(p, gameID),
		Kind:        knownFact(p, kind),
		SourceRowID: knownFact(p, gameID),
		Data: schema.VariantDocumentData{
			Family:                  item.Family,
			Category:                item.Category,
			Subcategory:             item.Subcategory,
			Presentation:            item.Presentation,
			Storage:                 item.Storage,
			Capabilities:            item.Capabilities,
			Safety:                  item.Safety,
			Acquisition:             item.Acquisition,
			Modifiers:               item.Modifiers,
			Links:                   item.Links,
			Unlocks:                 item.Unlocks,
			RelatedTechnicalRecords: item.RelatedTechnicalRecords,
			Weapon:                  &variantWeapon,
		},
		SourceRecords: append(
			append([]schema.ParameterRecord(nil), item.SourceRecords...),
			schema.ParameterRecord{
				Table:      "EquipParamWeapon",
				RowID:      int64(gameID),
				Provenance: p,
				Fields: []schema.ParameterField{{
					Name: "nameId",
				}},
			},
		),
	}
	if withAffinity {
		variant.Affinity = knownFact(p, schema.AffinityStandard)
	}
	if withUpgrade {
		variant.UpgradeLevel = knownFact(p, uint8(1))
	}
	return variant
}

func validExtendedWeaponData(p schema.Provenance, data schema.WeaponData) *schema.WeaponData {
	data.IconID = knownFact(p, uint32(1))
	data.SortID = knownFact(p, uint32(1))
	data.SortGroupID = knownFact(p, uint8(1))
	data.ReinforceTypeID = knownFact(p, int32(1))
	data.GemMountType = knownFact(p, uint8(1))
	data.AttackMagic = knownFact(p, int32(0))
	data.AttackFire = knownFact(p, int32(0))
	data.AttackLightning = knownFact(p, int32(0))
	data.AttackHoly = knownFact(p, int32(0))
	data.AttackStamina = knownFact(p, int32(1))
	data.GuardPhysical = knownFact(p, int32(1))
	data.GuardMagic = knownFact(p, int32(1))
	data.GuardFire = knownFact(p, int32(1))
	data.GuardLightning = knownFact(p, int32(1))
	data.GuardHoly = knownFact(p, int32(1))
	data.GuardBoost = knownFact(p, int32(1))
	data.RequiredIntelligence = knownFact(p, int32(0))
	data.RequiredFaith = knownFact(p, int32(0))
	data.RequiredArcane = knownFact(p, int32(0))
	data.ScalingStrengthRaw = knownFact(p, float64(1))
	data.ScalingDexterityRaw = knownFact(p, float64(1))
	data.ScalingIntelligenceRaw = knownFact(p, float64(0))
	data.ScalingFaithRaw = knownFact(p, float64(0))
	data.ScalingArcaneRaw = knownFact(p, float64(0))
	data.DefaultAshOfWarID = knownFact(p, int32(1))
	data.SwordArtsName = knownFact(p, "Quickstep")
	data.IsInfusable = knownFact(p, true)
	data.IsSomber = knownFact(p, false)
	data.MaxUpgrade = knownFact(p, int32(25))
	data.Warnings = knownFact(p, []string{"status-deferred"})
	data.RightHandEquipable = knownFact(p, true)
	data.LeftHandEquipable = knownFact(p, true)
	data.BothHandEquipable = knownFact(p, true)
	data.ArrowSlotEquipable = knownFact(p, false)
	data.BoltSlotEquipable = knownFact(p, false)
	return &data
}
