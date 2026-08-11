package dbviewer

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestUnknownCountTraversesNewTopLevelFamilyVariantAndAliasFacts(t *testing.T) {
	resource := schema.Resource{
		Item: &schema.ItemDocument{
			Armor:    &schema.ArmorData{},
			Variants: []schema.ItemVariant{{}},
			Aliases:  []schema.ItemAlias{{}},
		},
	}
	before := countUnknownFacts(resource)

	resource.Item.Category.Known = true
	resource.Item.Armor.Weight.Known = true
	resource.Item.Variants[0].Affinity.Known = true
	resource.Item.Aliases[0].GameID.Known = true

	after := countUnknownFacts(resource)
	if after != before-4 {
		t.Fatalf("unknown count changed from %d to %d, want %d", before, after, before-4)
	}
}

func TestCanonicalItemUnknownCountExcludesVariants(t *testing.T) {
	item := &schema.ItemDocument{
		Family: schema.Fact[schema.ItemFamily]{
			Known: true,
			Value: schema.ItemFamilyWeapon,
		},
		Variants: []schema.ItemVariant{{}},
	}

	before := countUnknownFactsForCanonicalItem(item)
	item.Variants[0].Affinity.Known = true
	after := countUnknownFactsForCanonicalItem(item)
	if after != before {
		t.Fatalf("canonical unknown count changed from %d to %d after changing only a variant", before, after)
	}
}

func TestUnknownCountSkipsNotApplicableMetadataAndUpgradeAffinity(t *testing.T) {
	type optionalMetadata struct {
		RequiredContainerID schema.Fact[uint32]
	}
	notApplicable := optionalMetadata{
		RequiredContainerID: schema.Fact[uint32]{
			Provenance: notApplicableProvenance("never held in a legacy RequiredContainer"),
		},
	}
	if got := countUnknownFactsForFamily(notApplicable, schema.ItemFamilySpiritAsh); got != 0 {
		t.Fatalf("not-applicable metadata unknown count = %d, want 0", got)
	}
	if got := countUnknownFactsForFamily(notApplicable, schema.ItemFamilyWeapon); got != 0 {
		t.Fatalf("weapon not-applicable metadata unknown count = %d, want 0", got)
	}

	unresolved := optionalMetadata{
		RequiredContainerID: schema.Fact[uint32]{
			Provenance: schema.Provenance{
				Source: schema.SourceSaveForgeLegacy,
				Method: "legacy RequiredContainer has no entry for this item",
			},
		},
	}
	if got := countUnknownFactsForFamily(unresolved, schema.ItemFamilySpiritAsh); got != 1 {
		t.Fatalf("unresolved metadata unknown count = %d, want 1", got)
	}

	variant := schema.ItemVariant{
		Kind: schema.Fact[schema.ItemVariantKind]{
			Known: true,
			Value: schema.ItemVariantUpgrade,
		},
		Affinity: schema.Fact[schema.Affinity]{
			Provenance: notApplicableProvenance(
				"spirit ash upgrade variants do not have an affinity",
			),
		},
	}
	before := countUnknownFactsForFamily(variant, schema.ItemFamilySpiritAsh)
	variant.Affinity.Known = true
	after := countUnknownFactsForFamily(variant, schema.ItemFamilySpiritAsh)
	if after != before {
		t.Fatalf("upgrade affinity changed unknown count from %d to %d, want unchanged", before, after)
	}
}

func notApplicableProvenance(reason string) schema.Provenance {
	return schema.Provenance{
		Source: schema.SourceSaveForgeLegacy,
		Method: schema.NotApplicableMethodPrefix + ": " + reason,
	}
}
