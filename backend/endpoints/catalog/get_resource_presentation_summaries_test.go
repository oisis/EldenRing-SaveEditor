package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
)

func TestGetResourcePresentationSummariesPreservesOrderDuplicatesAndPresentation(t *testing.T) {
	t.Parallel()

	identities := []catalog.ResourcePresentationIdentity{
		{Kind: resourceKindItem, Key: daggerResourceKey},
		{Kind: resourceKindItem, Key: getResourcesDeterminationK},
		{Kind: resourceKindItem, Key: daggerResourceKey},
	}
	result, err := catalog.GetResourcePresentationSummaries(newPrototypeCatalog(t), identities)
	if err != nil {
		t.Fatalf("GetResourcePresentationSummaries: %v", err)
	}
	if len(result.Resources) != len(identities) {
		t.Fatalf("Resources length = %d, want %d", len(result.Resources), len(identities))
	}
	for index, identity := range identities {
		if result.Resources[index].Kind != identity.Kind || result.Resources[index].Key != identity.Key {
			t.Fatalf("Resources[%d] identity = %+v, want %+v", index, result.Resources[index], identity)
		}
	}
	if got := result.Resources[0]; got.Name != "Dagger" || got.IconPath != "assets/icons/items/melee_armaments/dagger.png" {
		t.Fatalf("dagger presentation = %+v", got)
	}
	if result.Resources[0] != result.Resources[2] {
		t.Fatalf("duplicate presentations differ: %+v and %+v", result.Resources[0], result.Resources[2])
	}
}

func TestGetResourcePresentationSummariesReturnsAnEmptyArray(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResourcePresentationSummaries(newPrototypeCatalog(t), nil)
	if err != nil {
		t.Fatalf("GetResourcePresentationSummaries: %v", err)
	}
	if result.Resources == nil || len(result.Resources) != 0 {
		t.Fatalf("Resources = %#v, want non-nil empty slice", result.Resources)
	}
}

func TestGetResourcePresentationSummariesIsAtomicAndReportsTheInputIndex(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResourcePresentationSummaries(
		newPrototypeCatalog(t),
		[]catalog.ResourcePresentationIdentity{
			{Kind: resourceKindItem, Key: daggerResourceKey},
			{Kind: resourceKindItem, Key: "UNKNOWN"},
			{Kind: resourceKindItem, Key: daggerResourceKey},
		},
	)
	if err == nil || !strings.Contains(err.Error(), `identity 1: unknown resource key "UNKNOWN"`) {
		t.Fatalf("error = %v, want indexed exact-lookup failure", err)
	}
	if result.Resources != nil {
		t.Fatalf("Resources = %#v, want no partial result", result.Resources)
	}
}

func TestGetResourcePresentationSummariesRejectsANilCatalog(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResourcePresentationSummaries(nil, nil)
	if err == nil || err.Error() != "game catalog is not loaded" {
		t.Fatalf("error = %v, want nil-catalog rejection", err)
	}
	if result.Resources != nil {
		t.Fatalf("Resources = %#v, want zero result", result.Resources)
	}
}

func TestGetResourcePresentationSummariesJSONContainsOnlyScalarPresentationFields(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResourcePresentationSummaries(
		newPrototypeCatalog(t),
		[]catalog.ResourcePresentationIdentity{{Kind: resourceKindItem, Key: daggerResourceKey}},
	)
	if err != nil {
		t.Fatalf("GetResourcePresentationSummaries: %v", err)
	}
	encoded, err := json.Marshal(result.Resources[0])
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(fields) != 4 {
		t.Fatalf("fields = %v, want only kind, key, name and iconPath", fields)
	}
	for _, name := range []string{"kind", "key", "name", "iconPath"} {
		if _, exists := fields[name]; !exists {
			t.Errorf("field %q is missing from %s", name, encoded)
		}
	}
}

func TestGetResourcePresentationSummariesResolvesNoDatabaseItemsByExactIdentity(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResourcePresentationSummaries(
		newStoredCatalog(t),
		[]catalog.ResourcePresentationIdentity{{Kind: resourceKindItem, Key: noDatabaseItemKeys[0]}},
	)
	if err != nil {
		t.Fatalf("GetResourcePresentationSummaries(noDatabase): %v", err)
	}
	if len(result.Resources) != 1 || result.Resources[0].Key != noDatabaseItemKeys[0] {
		t.Fatalf("Resources = %+v, want the exact noDatabase identity", result.Resources)
	}
}
