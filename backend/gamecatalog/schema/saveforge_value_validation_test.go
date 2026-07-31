package schema_test

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestValidateResourceAcceptsDifferentSaveForgeValue(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	resource := detachedResource(resources[0])
	resource.Item.Storage.MaxInventory = knownFact(
		testProvenance(manifest),
		uint32(99),
	)
	resource.Item.Storage.MaxInventorySFV = saveForgeFact(uint32(1))

	if err := schema.ValidateResource(resource, sources); err != nil {
		t.Fatalf("ValidateResource: %v", err)
	}
}

func TestValidateResourceRejectsInvalidSaveForgeValue(t *testing.T) {
	manifest, resources := prototype.Data()
	sources := mustValidateManifest(t, manifest)
	tests := []struct {
		name   string
		mutate func(*schema.ItemDocument)
		want   string
	}{
		{
			name: "duplicates authoritative value",
			mutate: func(item *schema.ItemDocument) {
				item.Storage.MaxInventorySFV = saveForgeFact(
					item.Storage.MaxInventory.Value,
				)
			},
			want: "duplicates the authoritative value",
		},
		{
			name: "wrong source",
			mutate: func(item *schema.ItemDocument) {
				value := saveForgeFact(item.Storage.MaxInventory.Value + 1)
				value.Provenance.Source = "regulation_equip_param_weapon"
				item.Storage.MaxInventorySFV = value
			},
			want: "want \"legacy_db_data\"",
		},
		{
			name: "unknown authoritative sibling",
			mutate: func(item *schema.ItemDocument) {
				item.Storage.MaxInventory = schema.Fact[uint32]{
					Provenance: item.Storage.MaxInventory.Provenance,
				}
				item.Storage.MaxInventorySFV = saveForgeFact(uint32(1))
			},
			want: "authoritative sibling must be known",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resource := detachedResource(resources[0])
			test.mutate(resource.Item)
			err := schema.ValidateResource(resource, sources)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateResource error = %v, want %q", err, test.want)
			}
		})
	}
}

func detachedResource(resource schema.Resource) schema.Resource {
	item := *resource.Item
	resource.Item = &item
	return resource
}

func saveForgeFact[T any](value T) *schema.Fact[T] {
	return &schema.Fact[T]{
		Known: true,
		Value: value,
		Provenance: schema.Provenance{
			Source: schema.SourceSaveForgeLegacy,
			Method: "test SaveForge value",
		},
	}
}
