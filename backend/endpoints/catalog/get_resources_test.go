package catalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// The prototype catalog holds real, schema-valid resources, so GetResources is
// exercised against real data instead of a mock. It contains exactly two items:
// the Dagger (weapon, upgrade/infusion/ashOfWarMount/equipment enabled, stack
// disabled) and Ash of War: Determination (ash_of_war, no capability enabled).
const (
	getResourcesKindItem       = "item"
	getResourcesDaggerKey      = "000F4240"
	getResourcesDaggerName     = "Dagger"
	getResourcesDeterminationK = "8000EA60"
	getResourcesPrototypeTotal = 2
)

func newGetResourcesCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("gamecatalog.NewPrototype: %v", err)
	}
	return gameCatalog
}

// getResourcesKeys reduces a result to its keys, which is enough to assert both
// the selection and the deterministic (kind, key) order.
func getResourcesKeys(result catalog.GetResourcesResult) []string {
	keys := make([]string, 0, len(result.Resources))
	for _, entry := range result.Resources {
		keys = append(keys, entry.Key)
	}
	return keys
}

func assertGetResourcesKeys(t *testing.T, result catalog.GetResourcesResult, want ...string) {
	t.Helper()

	got := getResourcesKeys(result)
	if len(got) != len(want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("keys = %v, want %v", got, want)
		}
	}
}

func TestGetResourcesReturnsEveryResourceInCatalogOrder(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResources(newGetResourcesCatalog(t), "", "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources: %v", err)
	}

	assertGetResourcesKeys(t, result, getResourcesDaggerKey, getResourcesDeterminationK)
	if result.Total != getResourcesPrototypeTotal {
		t.Errorf("Total = %d, want %d", result.Total, getResourcesPrototypeTotal)
	}
	if result.Page != 1 || result.PageSize != catalog.GetResourcesDefaultPageSize {
		t.Errorf("Page/PageSize = %d/%d, want 1/%d", result.Page, result.PageSize, catalog.GetResourcesDefaultPageSize)
	}
}

func TestGetResourcesProjectsOnlyPickerFields(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResources(newGetResourcesCatalog(t), "", "", "", "", getResourcesDaggerKey, 0, 0)
	if err != nil {
		t.Fatalf("GetResources: %v", err)
	}
	if len(result.Resources) != 1 {
		t.Fatalf("Resources = %v, want exactly the dagger", getResourcesKeys(result))
	}

	entry := result.Resources[0]
	want := catalog.GetResourcesEntry{
		Kind:   schema.ResourceKindItem,
		Key:    getResourcesDaggerKey,
		Family: schema.ItemFamilyWeapon,
		Name:   getResourcesDaggerName,
	}
	if entry != want {
		t.Errorf("entry = %+v, want %+v", entry, want)
	}
}

func TestGetResourcesFiltersByResourceTypeAndFamily(t *testing.T) {
	t.Parallel()

	gameCatalog := newGetResourcesCatalog(t)

	byKind, err := catalog.GetResources(gameCatalog, getResourcesKindItem, "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources by resourceType: %v", err)
	}
	assertGetResourcesKeys(t, byKind, getResourcesDaggerKey, getResourcesDeterminationK)

	byFamily, err := catalog.GetResources(gameCatalog, "", string(schema.ItemFamilyWeapon), "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources by family: %v", err)
	}
	assertGetResourcesKeys(t, byFamily, getResourcesDaggerKey)
	if byFamily.Total != 1 {
		t.Errorf("Total = %d, want 1", byFamily.Total)
	}

	// A valid family that no prototype resource carries must filter everything
	// out instead of falling back to the unfiltered list.
	empty, err := catalog.GetResources(gameCatalog, "", string(schema.ItemFamilyGesture), "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources by unused family: %v", err)
	}
	assertGetResourcesKeys(t, empty)
	if empty.Total != 0 {
		t.Errorf("Total = %d, want 0", empty.Total)
	}
}

func TestGetResourcesFiltersByEnabledCapability(t *testing.T) {
	t.Parallel()

	gameCatalog := newGetResourcesCatalog(t)

	enabled, err := catalog.GetResources(gameCatalog, "", "", catalog.GetResourcesCapabilityInfusion, "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources by infusion: %v", err)
	}
	assertGetResourcesKeys(t, enabled, getResourcesDaggerKey)

	// Stack is known on both prototype items but enabled on neither, so a known
	// yet disabled capability must not match.
	disabled, err := catalog.GetResources(gameCatalog, "", "", catalog.GetResourcesCapabilityStack, "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources by stack: %v", err)
	}
	assertGetResourcesKeys(t, disabled)
	if disabled.Total != 0 {
		t.Errorf("Total = %d, want 0", disabled.Total)
	}
}

func TestGetResourcesSearchesKeyAndNameCaseInsensitively(t *testing.T) {
	t.Parallel()

	gameCatalog := newGetResourcesCatalog(t)

	byKey, err := catalog.GetResources(gameCatalog, "", "", "", "", "000f4240", 0, 0)
	if err != nil {
		t.Fatalf("GetResources by key search: %v", err)
	}
	assertGetResourcesKeys(t, byKey, getResourcesDaggerKey)

	byName, err := catalog.GetResources(gameCatalog, "", "", "", "", "dETERMINATION", 0, 0)
	if err != nil {
		t.Fatalf("GetResources by name search: %v", err)
	}
	assertGetResourcesKeys(t, byName, getResourcesDeterminationK)

	missing, err := catalog.GetResources(gameCatalog, "", "", "", "", "no such resource", 0, 0)
	if err != nil {
		t.Fatalf("GetResources by unknown search: %v", err)
	}
	assertGetResourcesKeys(t, missing)
}

func TestGetResourcesPagesDeterministicallyAndReportsTotal(t *testing.T) {
	t.Parallel()

	gameCatalog := newGetResourcesCatalog(t)

	first, err := catalog.GetResources(gameCatalog, "", "", "", "", "", 1, 1)
	if err != nil {
		t.Fatalf("GetResources page 1: %v", err)
	}
	assertGetResourcesKeys(t, first, getResourcesDaggerKey)
	if first.Total != getResourcesPrototypeTotal {
		t.Errorf("page 1 Total = %d, want %d", first.Total, getResourcesPrototypeTotal)
	}

	second, err := catalog.GetResources(gameCatalog, "", "", "", "", "", 2, 1)
	if err != nil {
		t.Fatalf("GetResources page 2: %v", err)
	}
	assertGetResourcesKeys(t, second, getResourcesDeterminationK)
	if second.Total != getResourcesPrototypeTotal {
		t.Errorf("page 2 Total = %d, want %d", second.Total, getResourcesPrototypeTotal)
	}
	if second.Page != 2 || second.PageSize != 1 {
		t.Errorf("page 2 Page/PageSize = %d/%d, want 2/1", second.Page, second.PageSize)
	}
}

func TestGetResourcesReturnsEmptySliceBeyondTheLastPage(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResources(newGetResourcesCatalog(t), "", "", "", "", "", 99, 1)
	if err != nil {
		t.Fatalf("GetResources: %v", err)
	}
	if result.Resources == nil {
		t.Fatal("Resources = nil, want an empty array so the JSON payload is [] and not null")
	}
	if len(result.Resources) != 0 {
		t.Fatalf("Resources = %v, want empty", getResourcesKeys(result))
	}
	if result.Total != getResourcesPrototypeTotal {
		t.Errorf("Total = %d, want %d", result.Total, getResourcesPrototypeTotal)
	}
}

func TestGetResourcesRejectsTheEndpointIDFilter(t *testing.T) {
	t.Parallel()

	_, err := catalog.GetResources(newGetResourcesCatalog(t), "", "", "", "get_resource", "", 0, 0)
	if err == nil {
		t.Fatal("GetResources accepted an endpointId filter the catalog cannot answer")
	}
	const want = "the endpointId filter is not supported because GameCatalog does not declare endpoint relations yet; got \"get_resource\""
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestGetResourcesRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	gameCatalog := newGetResourcesCatalog(t)

	if _, err := catalog.GetResources(nil, "", "", "", "", "", 0, 0); err == nil {
		t.Error("GetResources accepted a nil catalog")
	}

	cases := []struct {
		name         string
		resourceType string
		family       string
		capability   string
		page         int
		pageSize     int
	}{
		{name: "unknown resource type", resourceType: "gesture"},
		{name: "resource type is case sensitive", resourceType: "Item"},
		{name: "unknown family", family: "consumable"},
		{name: "unknown capability", capability: "upgradeable"},
		{name: "negative page", page: -1},
		{name: "negative page size", pageSize: -1},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := catalog.GetResources(
				gameCatalog,
				testCase.resourceType,
				testCase.family,
				testCase.capability,
				"",
				"",
				testCase.page,
				testCase.pageSize,
			)
			if err == nil {
				t.Fatal("GetResources accepted invalid input")
			}
		})
	}
}
