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

// The prototype catalog holds real, schema-valid resources, so the getter is
// exercised against real data instead of a mock.
const daggerResourceKey = "item:000F4240"

func newPrototypeCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("gamecatalog.NewPrototype: %v", err)
	}
	return gameCatalog
}

func TestGetResourceReturnsResourceByKey(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResource(newPrototypeCatalog(t), daggerResourceKey)
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}

	if result.Resource.Key != daggerResourceKey {
		t.Errorf("Resource.Key = %q, want %q", result.Resource.Key, daggerResourceKey)
	}
	if result.Resource.Kind != schema.ResourceKindItem {
		t.Errorf("Resource.Kind = %q, want %q", result.Resource.Kind, schema.ResourceKindItem)
	}
	if result.Resource.Item == nil {
		t.Fatal("Resource.Item = nil, want the full item document")
	}
	if result.Resource.Item.GameID.Value != prototype.DaggerGameID {
		t.Errorf("Item.GameID = 0x%08X, want 0x%08X", result.Resource.Item.GameID.Value, prototype.DaggerGameID)
	}
	if result.Resource.Item.Family.Value != schema.ItemFamilyWeapon {
		t.Errorf("Item.Family = %q, want %q", result.Resource.Item.Family.Value, schema.ItemFamilyWeapon)
	}
}

func TestGetResourceReturnsFullItemDocument(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResource(newPrototypeCatalog(t), daggerResourceKey)
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	item := result.Resource.Item

	presentation := item.Presentation
	if !presentation.Name.Known || presentation.Name.Value != "Dagger" {
		t.Errorf("Presentation.Name = %+v, want a known %q", presentation.Name, "Dagger")
	}
	if presentation.Name.Provenance.Source == "" {
		t.Error("Presentation.Name provenance source is empty; provenance must survive the getter")
	}

	if !item.Capabilities.Upgrade.Known || !item.Capabilities.Upgrade.Enabled {
		t.Error("Capabilities.Upgrade is not known and enabled")
	}
	if item.Capabilities.Upgrade.Rules == nil {
		t.Fatal("Capabilities.Upgrade.Rules = nil, want the upgrade rules of the Dagger")
	}
	if got := item.Capabilities.Upgrade.Rules.MaxLevel; got != 25 {
		t.Errorf("Capabilities.Upgrade.Rules.MaxLevel = %d, want 25", got)
	}
	if item.Capabilities.Infusion.Rules == nil || len(item.Capabilities.Infusion.Rules.AllowedAffinities) != 13 {
		t.Errorf("Capabilities.Infusion.Rules = %+v, want 13 allowed affinities", item.Capabilities.Infusion.Rules)
	}

	if len(item.Variants) == 0 {
		t.Fatal("Item.Variants is empty, want the Dagger variants")
	}
	if item.Variants[0].GameID.Provenance.Source == "" {
		t.Error("Variants[0].GameID provenance source is empty; variant provenance must survive the getter")
	}
	if len(item.SourceRecords) == 0 {
		t.Error("Item.SourceRecords is empty, want the parameter records backing the document")
	}
}

// The JSON contract is part of the public getter: the result carries the
// resource and nothing else, in particular no relations.
func TestGetResourceResultSerialisesOnlyTheResource(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResource(newPrototypeCatalog(t), daggerResourceKey)
	if err != nil {
		t.Fatalf("GetResource: %v", err)
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
		t.Errorf("top-level JSON keys = %v, want only [resource]", sortedKeys(fields))
	}
	if _, exists := fields["resource"]; !exists {
		t.Errorf("top-level JSON keys = %v, want a %q key", sortedKeys(fields), "resource")
	}
	for _, removed := range []string{"outgoingRelations", "incomingRelations", "relatedResources"} {
		if _, exists := fields[removed]; exists {
			t.Errorf("top-level JSON contains %q; relations belong to GetResourceRelations", removed)
		}
	}
}

func sortedKeys(fields map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestGetResourceRejectsMissingCatalog(t *testing.T) {
	t.Parallel()

	result, err := catalog.GetResource(nil, daggerResourceKey)
	if err == nil {
		t.Fatalf("GetResource(nil, %q) = %+v, nil error; want error", daggerResourceKey, result)
	}
	assertEmptyResourceResult(t, result)
}

func TestGetResourceRejectsInvalidResourceID(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"empty":               "",
		"whitespace only":     "   ",
		"tab only":            "\t",
		"leading whitespace":  " " + daggerResourceKey,
		"trailing whitespace": daggerResourceKey + " ",
		"unknown key":         "item:DEADBEEF",
		"unknown key kind":    "gesture:000F4240",
		// The public parameter is Resource.Key, never the numeric ResourceID.
		"numeric resource ID":  "1",
		"numeric game ID":      strconv.FormatUint(uint64(prototype.DaggerGameID), 10),
		"hex key without kind": "000F4240",
	}

	gameCatalog := newPrototypeCatalog(t)
	for name, resourceID := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := catalog.GetResource(gameCatalog, resourceID)
			if err == nil {
				t.Fatalf("GetResource(catalog, %q) = %+v, nil error; want error", resourceID, result)
			}
			assertEmptyResourceResult(t, result)
		})
	}
}

func TestGetResourceReportsTheUnknownResourceID(t *testing.T) {
	t.Parallel()

	const unknown = "item:DEADBEEF"
	_, err := catalog.GetResource(newPrototypeCatalog(t), unknown)
	if err == nil {
		t.Fatalf("GetResource(catalog, %q) = nil error, want error", unknown)
	}
	if !strings.Contains(err.Error(), unknown) {
		t.Errorf("error = %q, want it to name the missing resource ID %q", err, unknown)
	}
}

func TestGetResourceDoesNotMutateCatalog(t *testing.T) {
	t.Parallel()

	gameCatalog := newPrototypeCatalog(t)
	before, err := catalog.GetResource(gameCatalog, daggerResourceKey)
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	originalName := before.Resource.Item.Presentation.Name.Value
	originalVariantGameID := before.Resource.Item.Variants[0].GameID.Value
	originalAffinity := before.Resource.Item.Capabilities.Infusion.Rules.AllowedAffinities[0]
	originalResourceCount := gameCatalog.ResourceCount()

	before.Resource.Key = "mutated"
	before.Resource.Item.Presentation.Name.Value = "mutated"
	before.Resource.Item.Variants[0].GameID.Value = 1
	before.Resource.Item.Capabilities.Infusion.Rules.AllowedAffinities[0] = schema.AffinityOccult

	after, err := catalog.GetResource(gameCatalog, daggerResourceKey)
	if err != nil {
		t.Fatalf("GetResource after mutation: %v", err)
	}
	if after.Resource.Key != daggerResourceKey {
		t.Errorf("catalog resource key = %q, want %q", after.Resource.Key, daggerResourceKey)
	}
	if got := after.Resource.Item.Presentation.Name.Value; got != originalName {
		t.Errorf("catalog item name = %q, want %q", got, originalName)
	}
	if got := after.Resource.Item.Variants[0].GameID.Value; got != originalVariantGameID {
		t.Errorf("catalog variant game ID = 0x%08X, want 0x%08X", got, originalVariantGameID)
	}
	if got := after.Resource.Item.Capabilities.Infusion.Rules.AllowedAffinities[0]; got != originalAffinity {
		t.Errorf("catalog affinity = %q, want %q", got, originalAffinity)
	}
	if got := gameCatalog.ResourceCount(); got != originalResourceCount {
		t.Errorf("catalog resource count = %d, want %d; the getter must not add resources", got, originalResourceCount)
	}
}

func assertEmptyResourceResult(t *testing.T, result catalog.GetResourceResult) {
	t.Helper()

	if result.Resource.ID != 0 || result.Resource.Key != "" ||
		result.Resource.Kind != "" || result.Resource.Item != nil {
		t.Errorf("Resource = %+v, want an empty resource on failure", result.Resource)
	}
}
