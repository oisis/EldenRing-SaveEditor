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

func TestUnknownCountSkipsNotApplicableMetadataAndUpgradeAffinity(t *testing.T) {
	metadata := struct {
		RequiredContainerID schema.Fact[uint32]
		WhetbladeName       schema.Fact[string]
	}{}
	if got := countUnknownFactsForFamily(metadata, schema.ItemFamilySpiritAsh); got != 0 {
		t.Fatalf("not-applicable metadata unknown count = %d, want 0", got)
	}

	variant := schema.ItemVariant{
		Kind: schema.Fact[schema.ItemVariantKind]{
			Known: true,
			Value: schema.ItemVariantUpgrade,
		},
	}
	before := countUnknownFactsForFamily(variant, schema.ItemFamilySpiritAsh)
	variant.Affinity.Known = true
	after := countUnknownFactsForFamily(variant, schema.ItemFamilySpiritAsh)
	if after != before {
		t.Fatalf("upgrade affinity changed unknown count from %d to %d, want unchanged", before, after)
	}
}
