package catalog_test

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// The prototype catalog holds items only, so the colosseum cases run against the
// stored catalog, which declares the three base-game arenas next to the items.
const (
	colosseumKind      = "colosseum"
	colosseumRoyalKey  = "royal_colosseum"
	colosseumRoyalName = "Royal Colosseum"
	colosseumRoyalFlag = 60370
	colosseumCount     = 3
)

var (
	storedCatalogOnce sync.Once
	storedCatalog     *gamecatalog.Catalog
	storedCatalogErr  error
)

// The catalog is read-only and every getter returns copies, so one build is
// shared by the cases below instead of paying for it four times.
func newStoredCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	storedCatalogOnce.Do(func() {
		data, err := loader.LoadFS(catalogdata.Files())
		if err != nil {
			storedCatalogErr = err
			return
		}
		storedCatalog, storedCatalogErr = gamecatalog.New(data.Manifest, data.Resources())
	})
	if storedCatalogErr != nil {
		t.Fatalf("build stored catalog: %v", storedCatalogErr)
	}
	return storedCatalog
}

func TestGetResourcesReturnsColosseumsWithTheirNames(t *testing.T) {
	t.Parallel()

	gameCatalog := newStoredCatalog(t)

	result, err := catalog.GetResources(gameCatalog, colosseumKind, "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(colosseum): %v", err)
	}
	if result.Total != colosseumCount {
		t.Fatalf("total = %d, want %d", result.Total, colosseumCount)
	}
	royal := false
	for _, entry := range result.Resources {
		if entry.Kind != schema.ResourceKindColosseum {
			t.Errorf("entry %q has kind %q, want %q", entry.Key, entry.Kind, schema.ResourceKindColosseum)
		}
		// A colosseum carries no item document, so it must never be reported
		// with a family, which would be an item property it does not have.
		if entry.Family != "" {
			t.Errorf("colosseum %q reports family %q, want empty", entry.Key, entry.Family)
		}
		if entry.Name == "" {
			t.Errorf("colosseum %q reports no name", entry.Key)
		}
		if entry.Key == colosseumRoyalKey {
			royal = true
			if entry.Name != colosseumRoyalName {
				t.Errorf("colosseum %q name = %q, want %q", entry.Key, entry.Name, colosseumRoyalName)
			}
		}
	}
	if !royal {
		t.Fatalf("colosseum %q is missing from the result", colosseumRoyalKey)
	}
}

func TestGetResourcesSearchesColosseumsByName(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResources(
		newStoredCatalog(t), colosseumKind, "", "", "", "royal colosseum", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(colosseum, search): %v", err)
	}
	if result.Total != 1 || len(result.Resources) != 1 {
		t.Fatalf("total = %d, resources = %d, want 1 and 1", result.Total, len(result.Resources))
	}
	if result.Resources[0].Key != colosseumRoyalKey {
		t.Fatalf("key = %q, want %q", result.Resources[0].Key, colosseumRoyalKey)
	}
}

// Family and capability describe items only. Combined with the colosseum type
// they must select nothing instead of widening the page back to the items.
func TestGetResourcesNeverMatchesAColosseumByFamilyOrCapability(t *testing.T) {
	t.Parallel()

	gameCatalog := newStoredCatalog(t)

	for name, filter := range map[string]struct{ family, capability string }{
		"family":     {family: string(schema.ItemFamilyWeapon)},
		"capability": {capability: catalog.GetResourcesCapabilityUpgrade},
	} {
		result, err := catalog.GetResources(
			gameCatalog, colosseumKind, filter.family, filter.capability, "", "", 0, 0)
		if err != nil {
			t.Fatalf("GetResources(colosseum, %s): %v", name, err)
		}
		if result.Total != 0 || len(result.Resources) != 0 {
			t.Fatalf("%s filter: total = %d, resources = %d, want 0 and 0",
				name, result.Total, len(result.Resources))
		}
	}
}

func TestGetResourceReturnsTheColosseumDocument(t *testing.T) {
	t.Parallel()

	gameCatalog := newStoredCatalog(t)

	result, err := catalog.GetResource(gameCatalog, colosseumKind, colosseumRoyalKey)
	if err != nil {
		t.Fatalf("GetResource(colosseum, %q): %v", colosseumRoyalKey, err)
	}
	if result.Resource.Item != nil {
		t.Fatalf("colosseum resource carries an item document")
	}
	if result.Resource.Colosseum == nil {
		t.Fatalf("colosseum resource carries no colosseum document")
	}
	name := result.Resource.Colosseum.Name
	if !name.Known || name.Value != colosseumRoyalName {
		t.Errorf("name = %q (known %t), want %q", name.Value, name.Known, colosseumRoyalName)
	}
	if name.Provenance.Source == "" || name.Provenance.Method == "" {
		t.Errorf("name provenance = %+v, want a source and a method", name.Provenance)
	}
	flag := result.Resource.Colosseum.UnlockEventFlagID
	if !flag.Known || flag.Value != colosseumRoyalFlag {
		t.Errorf("unlockEventFlagID = %d (known %t), want %d", flag.Value, flag.Known, colosseumRoyalFlag)
	}
	if flag.Provenance.Source == "" || flag.Provenance.Method == "" {
		t.Errorf("unlockEventFlagID provenance = %+v, want a source and a method", flag.Provenance)
	}

	// The returned document is a copy: mutating it must not reach the catalog.
	result.Resource.Colosseum.Name.Value = "mutated"
	again, err := catalog.GetResource(gameCatalog, colosseumKind, colosseumRoyalKey)
	if err != nil {
		t.Fatalf("GetResource(colosseum, %q) again: %v", colosseumRoyalKey, err)
	}
	if again.Resource.Colosseum.Name.Value != colosseumRoyalName {
		t.Fatalf("catalog name = %q, want %q", again.Resource.Colosseum.Name.Value, colosseumRoyalName)
	}
}

// The union must not serialise the document of a kind the resource is not, so a
// colosseum response carries no item key at all, not even a null one.
func TestGetResourceColosseumJSONCarriesNoItemField(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResource(newStoredCatalog(t), colosseumKind, colosseumRoyalKey)
	if err != nil {
		t.Fatalf("GetResource(colosseum, %q): %v", colosseumRoyalKey, err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(encoded), `"item"`) {
		t.Fatalf("colosseum JSON contains an item field: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"colosseum"`) {
		t.Fatalf("colosseum JSON contains no colosseum field: %s", encoded)
	}
}
