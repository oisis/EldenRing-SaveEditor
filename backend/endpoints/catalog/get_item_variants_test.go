package catalog_test

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const itemVariantsDaggerResourceKey = "item:000F4240"

// The Determination Ash of War is a real prototype resource whose item document
// stores no variants, so the empty case is covered by real data too.
const determinationResourceKey = "item:8000EA60"

func newItemVariantsPrototypeCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("gamecatalog.NewPrototype: %v", err)
	}
	return gameCatalog
}

func itemVariantsSortedKeys(fields map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestGetItemVariantsReturnsEveryStoredVariant(t *testing.T) {
	t.Parallel()

	gameCatalog := newItemVariantsPrototypeCatalog(t)
	result, err := catalog.GetItemVariants(gameCatalog, itemVariantsDaggerResourceKey)
	if err != nil {
		t.Fatalf("GetItemVariants: %v", err)
	}

	resource, exists := gameCatalog.ResourceByKey(itemVariantsDaggerResourceKey)
	if !exists {
		t.Fatalf("ResourceByKey(%q) is missing from the prototype catalog", itemVariantsDaggerResourceKey)
	}
	if len(result.Variants) != len(resource.Item.Variants) {
		t.Fatalf("len(Variants) = %d, want %d", len(result.Variants), len(resource.Item.Variants))
	}
	if len(result.Variants) == 0 {
		t.Fatal("Variants is empty, want the Dagger variants")
	}
}

func TestGetItemVariantsPreservesVariantDataAndProvenance(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetItemVariants(newItemVariantsPrototypeCatalog(t), itemVariantsDaggerResourceKey)
	if err != nil {
		t.Fatalf("GetItemVariants: %v", err)
	}
	variant := result.Variants[0]

	if !variant.GameID.Known || variant.GameID.Value != 1000100 {
		t.Errorf("Variants[0].GameID = %+v, want a known 1000100", variant.GameID)
	}
	if variant.GameID.Provenance.Source == "" {
		t.Error("Variants[0].GameID provenance source is empty; provenance must survive the getter")
	}
	if variant.Kind.Value != schema.ItemVariantAffinity {
		t.Errorf("Variants[0].Kind = %q, want %q", variant.Kind.Value, schema.ItemVariantAffinity)
	}
	if variant.Affinity.Value != schema.AffinityHeavy {
		t.Errorf("Variants[0].Affinity = %q, want %q", variant.Affinity.Value, schema.AffinityHeavy)
	}
	if variant.Affinity.Provenance.Source == "" {
		t.Error("Variants[0].Affinity provenance source is empty; provenance must survive the getter")
	}

	displayName := variant.Data.Presentation.DisplayName
	if !displayName.Known || displayName.Value != "Dagger" {
		t.Errorf("Variants[0].Data.Presentation.DisplayName = %+v, want a known %q", displayName, "Dagger")
	}
	if displayName.Provenance.Source == "" {
		t.Error("Variants[0].Data display name provenance source is empty; provenance must survive the getter")
	}
	if variant.Data.Weapon == nil {
		t.Error("Variants[0].Data.Weapon = nil, want the weapon block of the variant")
	}
	if len(variant.SourceRecords) == 0 {
		t.Error("Variants[0].SourceRecords is empty, want the parameter records backing the variant")
	}
}

// The catalog order is the contract: the getter never sorts or reorders.
func TestGetItemVariantsPreservesCatalogOrder(t *testing.T) {
	t.Parallel()

	gameCatalog := newItemVariantsPrototypeCatalog(t)
	result, err := catalog.GetItemVariants(gameCatalog, itemVariantsDaggerResourceKey)
	if err != nil {
		t.Fatalf("GetItemVariants: %v", err)
	}

	resource, exists := gameCatalog.ResourceByKey(itemVariantsDaggerResourceKey)
	if !exists {
		t.Fatalf("ResourceByKey(%q) is missing from the prototype catalog", itemVariantsDaggerResourceKey)
	}
	for index, stored := range resource.Item.Variants {
		if got := result.Variants[index].GameID.Value; got != stored.GameID.Value {
			t.Errorf("Variants[%d].GameID = 0x%08X, want 0x%08X", index, got, stored.GameID.Value)
		}
	}
}

// An item without variants is a valid case: an empty JSON array, never null and
// never an error. The base item is not synthesised into a variant.
func TestGetItemVariantsReturnsEmptyArrayForItemWithoutVariants(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetItemVariants(newItemVariantsPrototypeCatalog(t), determinationResourceKey)
	if err != nil {
		t.Fatalf("GetItemVariants: %v", err)
	}
	if result.Variants == nil {
		t.Fatal("Variants = nil, want an empty non-nil slice")
	}
	if len(result.Variants) != 0 {
		t.Fatalf("len(Variants) = %d, want 0", len(result.Variants))
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if got := string(encoded); got != `{"variants":[]}` {
		t.Errorf("encoded result = %s, want %s", got, `{"variants":[]}`)
	}
}

// The JSON contract is part of the public getter: the result carries the
// variants and nothing else, in particular no resource document.
func TestGetItemVariantsResultSerialisesOnlyTheVariants(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetItemVariants(newItemVariantsPrototypeCatalog(t), itemVariantsDaggerResourceKey)
	if err != nil {
		t.Fatalf("GetItemVariants: %v", err)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if len(fields) != 1 {
		t.Errorf("top-level JSON keys = %v, want only [variants]", itemVariantsSortedKeys(fields))
	}
	if _, exists := fields["variants"]; !exists {
		t.Errorf("top-level JSON keys = %v, want a %q key", itemVariantsSortedKeys(fields), "variants")
	}
	if _, exists := fields["resource"]; exists {
		t.Error("top-level JSON contains \"resource\"; the resource document belongs to GetResource")
	}
}

func TestGetItemVariantsRejectsMissingCatalog(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetItemVariants(nil, itemVariantsDaggerResourceKey)
	if err == nil {
		t.Fatalf("GetItemVariants(nil, %q) = %+v, nil error; want error", itemVariantsDaggerResourceKey, result)
	}
	assertEmptyVariantsResult(t, result)
}

func TestGetItemVariantsRejectsInvalidResourceID(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"empty":               "",
		"whitespace only":     "   ",
		"tab only":            "\t",
		"leading whitespace":  " " + itemVariantsDaggerResourceKey,
		"trailing whitespace": itemVariantsDaggerResourceKey + " ",
		"unknown key":         "item:DEADBEEF",
		"unknown key kind":    "gesture:000F4240",
		// The public parameter is Resource.Key, never the numeric ResourceID.
		"numeric resource ID":  "1",
		"numeric game ID":      strconv.FormatUint(uint64(prototype.DaggerGameID), 10),
		"hex key without kind": "000F4240",
	}

	gameCatalog := newItemVariantsPrototypeCatalog(t)
	for name, resourceID := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := catalog.GetItemVariants(gameCatalog, resourceID)
			if err == nil {
				t.Fatalf("GetItemVariants(catalog, %q) = %+v, nil error; want error", resourceID, result)
			}
			assertEmptyVariantsResult(t, result)
		})
	}
}

func TestGetItemVariantsReportsTheUnknownResourceID(t *testing.T) {
	t.Parallel()

	const unknown = "item:DEADBEEF"
	_, err := catalog.GetItemVariants(newItemVariantsPrototypeCatalog(t), unknown)
	if err == nil {
		t.Fatalf("GetItemVariants(catalog, %q) = nil error, want error", unknown)
	}
	if !strings.Contains(err.Error(), unknown) {
		t.Errorf("error = %q, want it to name the missing resource ID %q", err, unknown)
	}
}

func TestGetItemVariantsDoesNotMutateCatalog(t *testing.T) {
	t.Parallel()

	gameCatalog := newItemVariantsPrototypeCatalog(t)
	before, err := catalog.GetItemVariants(gameCatalog, itemVariantsDaggerResourceKey)
	if err != nil {
		t.Fatalf("GetItemVariants: %v", err)
	}
	originalGameID := before.Variants[0].GameID.Value
	originalDisplayName := before.Variants[0].Data.Presentation.DisplayName.Value
	originalCount := len(before.Variants)

	before.Variants[0].GameID.Value = 1
	before.Variants[0].Data.Presentation.DisplayName.Value = "mutated"
	before.Variants[0].SourceRecords = nil

	after, err := catalog.GetItemVariants(gameCatalog, itemVariantsDaggerResourceKey)
	if err != nil {
		t.Fatalf("GetItemVariants after mutation: %v", err)
	}
	if len(after.Variants) != originalCount {
		t.Fatalf("len(Variants) = %d, want %d", len(after.Variants), originalCount)
	}
	if got := after.Variants[0].GameID.Value; got != originalGameID {
		t.Errorf("catalog variant game ID = 0x%08X, want 0x%08X", got, originalGameID)
	}
	if got := after.Variants[0].Data.Presentation.DisplayName.Value; got != originalDisplayName {
		t.Errorf("catalog variant display name = %q, want %q", got, originalDisplayName)
	}
	if len(after.Variants[0].SourceRecords) == 0 {
		t.Error("catalog variant source records were cleared; the getter must hand out an independent copy")
	}
}

func assertEmptyVariantsResult(t *testing.T, result catalog.GetItemVariantsResult) {
	t.Helper()

	if result.Variants != nil {
		t.Errorf("Variants = %+v, want an empty result on failure", result.Variants)
	}
}
