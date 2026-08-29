package catalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// The three Regulation 1.17 Spectral Steed Attire goods are the current
// noDatabase items: the World Spectral Steed Attire endpoints own them, so the
// general item list must not offer them while an exact lookup must still work.
var noDatabaseItemKeys = []string{"401EAA00", "401EAA0A", "401EAA14"}

func TestGetResourcesHidesNoDatabaseItems(t *testing.T) {
	t.Parallel()

	gameCatalog := newStoredCatalog(t)
	hidden := make(map[string]struct{}, len(noDatabaseItemKeys))
	for _, key := range noDatabaseItemKeys {
		hidden[key] = struct{}{}
	}

	// The unfiltered list, the item list and a search that matches the shared
	// product name are the three ways a picker reaches the general catalog.
	for name, call := range map[string]func() (catalog.GetResourcesResult, error){
		"every resource": func() (catalog.GetResourcesResult, error) {
			return catalog.GetResources(gameCatalog, "", "", "", "", "", 0, 100000)
		},
		"items": func() (catalog.GetResourcesResult, error) {
			return catalog.GetResources(
				gameCatalog, string(schema.ResourceKindItem), "", "", "", "", 0, 100000)
		},
		"search": func() (catalog.GetResourcesResult, error) {
			return catalog.GetResources(
				gameCatalog, "", "", "", "", "Spectral Steed Regalia", 0, 100000)
		},
	} {
		t.Run(name, func(t *testing.T) {
			result, err := call()
			if err != nil {
				t.Fatalf("GetResources: %v", err)
			}
			if result.Total != len(result.Resources) {
				t.Fatalf("Total = %d but the page carries %d rows; widen the page size",
					result.Total, len(result.Resources))
			}
			for _, entry := range result.Resources {
				if _, forbidden := hidden[entry.Key]; forbidden {
					t.Errorf("the general catalog offers noDatabase item %q", entry.Key)
				}
			}
		})
	}
}

func TestGetResourceStillResolvesNoDatabaseItems(t *testing.T) {
	t.Parallel()

	gameCatalog := newStoredCatalog(t)
	for _, key := range noDatabaseItemKeys {
		result, err := catalog.GetResource(gameCatalog, string(schema.ResourceKindItem), key)
		if err != nil {
			t.Fatalf("GetResource(%q): %v", key, err)
		}
		if result.Resource.Key != key || result.Resource.Item == nil {
			t.Fatalf("GetResource(%q) = %+v, want the full item document", key, result.Resource)
		}
		if !result.Resource.Item.Safety.NoDatabase.Known ||
			!result.Resource.Item.Safety.NoDatabase.Value {
			t.Errorf("%q is no longer a noDatabase item; this test guards the wrong resource", key)
		}
	}
}
