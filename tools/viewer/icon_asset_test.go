package dbviewer

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestVariantIconUsesCanonicalAsset(t *testing.T) {
	item := &schema.ItemDocument{
		Presentation: schema.ItemPresentation{
			IconPath: schema.Fact[string]{
				Known: true,
				Value: "assets/icons/items/weapon/canonical.png",
			},
		},
	}
	variant := schema.ItemVariant{}
	if got := variantIconURL(item, variant); got != "/catalog-assets/icons/items/weapon/canonical.png" {
		t.Fatalf("variant icon URL = %q", got)
	}
}
