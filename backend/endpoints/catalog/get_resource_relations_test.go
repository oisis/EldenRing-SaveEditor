package catalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// The prototype catalog holds the Dagger and Determination, which the catalog
// links with exactly one derived compatible_with_aow relation. That is real,
// schema-valid data, so the getter is exercised against the relation the catalog
// itself derived instead of a mock.
const (
	relationsKindItem            = "item"
	relationsDaggerKey           = "000F4240"
	relationsDeterminationKey    = "8000EA60"
	relationsUnknownKey          = "DEADBEEF"
	relationsUnknownKind         = "gesture"
	relationsCompatibleWithAshes = "compatible_with_aow"
	relationsRequiresContainer   = "requires_container"
)

func newRelationsCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("gamecatalog.NewPrototype: %v", err)
	}
	return gameCatalog
}

func relationsOf(
	t *testing.T,
	gameCatalog *gamecatalog.Catalog,
	key string,
	relationType string,
	direction string,
) catalog.GetResourceRelationsResult {
	t.Helper()

	result, err := catalog.GetResourceRelations(gameCatalog, relationsKindItem, key, relationType, direction)
	if err != nil {
		t.Fatalf(
			"GetResourceRelations(catalog, %q, %q, %q, %q): %v",
			relationsKindItem,
			key,
			relationType,
			direction,
			err,
		)
	}
	return result
}

func assertRelationCounts(t *testing.T, result catalog.GetResourceRelationsResult, outgoing, incoming int) {
	t.Helper()

	if len(result.Outgoing) != outgoing {
		t.Errorf("len(Outgoing) = %d, want %d", len(result.Outgoing), outgoing)
	}
	if len(result.Incoming) != incoming {
		t.Errorf("len(Incoming) = %d, want %d", len(result.Incoming), incoming)
	}
}

func TestGetResourceRelationsReturnsOutgoingRelations(t *testing.T) {
	result := relationsOf(t, newRelationsCatalog(t), relationsDaggerKey, "", "")

	assertRelationCounts(t, result, 1, 0)
	relation := result.Outgoing[0]
	if relation.Kind != schema.RelationCompatibleWithAshOfWar {
		t.Errorf("Kind = %q, want %q", relation.Kind, schema.RelationCompatibleWithAshOfWar)
	}
	if relation.From.Kind != schema.ResourceKindItem || relation.From.Key != relationsDaggerKey {
		t.Errorf("From = %+v, want kind %q key %q", relation.From, schema.ResourceKindItem, relationsDaggerKey)
	}
	if relation.To.Kind != schema.ResourceKindItem || relation.To.Key != relationsDeterminationKey {
		t.Errorf("To = %+v, want kind %q key %q", relation.To, schema.ResourceKindItem, relationsDeterminationKey)
	}
	if relation.Provenance.Source == "" {
		t.Error("Provenance.Source is empty; the derived provenance must survive the getter")
	}
}

func TestGetResourceRelationsReturnsIncomingRelations(t *testing.T) {
	result := relationsOf(t, newRelationsCatalog(t), relationsDeterminationKey, "", "")

	assertRelationCounts(t, result, 0, 1)
	relation := result.Incoming[0]
	if relation.From.Key != relationsDaggerKey || relation.To.Key != relationsDeterminationKey {
		t.Errorf(
			"relation = %q -> %q, want %q -> %q",
			relation.From.Key,
			relation.To.Key,
			relationsDaggerKey,
			relationsDeterminationKey,
		)
	}
}

func TestGetResourceRelationsFiltersByRelationType(t *testing.T) {
	gameCatalog := newRelationsCatalog(t)

	matching := relationsOf(t, gameCatalog, relationsDaggerKey, relationsCompatibleWithAshes, "")
	assertRelationCounts(t, matching, 1, 0)

	// A supported type that matches nothing is an empty result, not an error.
	other := relationsOf(t, gameCatalog, relationsDaggerKey, relationsRequiresContainer, "")
	assertRelationCounts(t, other, 0, 0)
}

func TestGetResourceRelationsFiltersByDirection(t *testing.T) {
	gameCatalog := newRelationsCatalog(t)

	outgoingOnly := relationsOf(t, gameCatalog, relationsDaggerKey, "", "outgoing")
	assertRelationCounts(t, outgoingOnly, 1, 0)

	// The Dagger has no incoming relation, so the outgoing one must disappear
	// rather than leak into the filtered direction.
	incomingOnly := relationsOf(t, gameCatalog, relationsDaggerKey, "", "incoming")
	assertRelationCounts(t, incomingOnly, 0, 0)

	determinationIncoming := relationsOf(t, gameCatalog, relationsDeterminationKey, "", "incoming")
	assertRelationCounts(t, determinationIncoming, 0, 1)
}

func TestGetResourceRelationsRejectsInvalidInput(t *testing.T) {
	gameCatalog := newRelationsCatalog(t)
	cases := []struct {
		name         string
		gameCatalog  *gamecatalog.Catalog
		kind         string
		key          string
		relationType string
		direction    string
		want         string
	}{
		{"nil catalog", nil, relationsKindItem, relationsDaggerKey, "", "", "game catalog is not loaded"},
		{"empty kind", gameCatalog, "", relationsDaggerKey, "", "", "resource kind is required"},
		{"empty key", gameCatalog, relationsKindItem, "", "", "", "resource key is required"},
		{"unknown kind", gameCatalog, relationsUnknownKind, relationsDaggerKey, "", "", "unknown resource kind"},
		{"unknown key", gameCatalog, relationsKindItem, relationsUnknownKey, "", "", "unknown resource key"},
		{"unknown relation type", gameCatalog, relationsKindItem, relationsDaggerKey, "linked_to", "", "relation type \"linked_to\" is not supported"},
		{"unknown direction", gameCatalog, relationsKindItem, relationsDaggerKey, "", "both", "direction \"both\" is not supported"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := catalog.GetResourceRelations(
				testCase.gameCatalog,
				testCase.kind,
				testCase.key,
				testCase.relationType,
				testCase.direction,
			)
			if err == nil {
				t.Fatalf("GetResourceRelations = %+v, nil error; want error", result)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %q, want it to report %q", err, testCase.want)
			}
			if result.Outgoing != nil || result.Incoming != nil {
				t.Errorf("result = %+v, want an empty result on failure", result)
			}
		})
	}
}

// The JSON contract is part of the public getter: a filtered-out direction is an
// empty array, never null, so a client never has to special-case it.
func TestGetResourceRelationsSerialisesEmptyDirectionsAsArrays(t *testing.T) {
	result := relationsOf(t, newRelationsCatalog(t), relationsDaggerKey, "", "outgoing")

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"incoming":[]`) {
		t.Errorf("encoded = %s, want the filtered direction as an empty array", encoded)
	}
	if strings.Contains(string(encoded), "null") {
		t.Errorf("encoded = %s, want no null direction", encoded)
	}
}

func TestGetResourceRelationsDoesNotMutateCatalog(t *testing.T) {
	gameCatalog := newRelationsCatalog(t)

	before := relationsOf(t, gameCatalog, relationsDaggerKey, "", "")
	originalTo := before.Outgoing[0].To
	before.Outgoing[0].To = schema.ResourceRef{Kind: "mutated", Key: "mutated"}
	before.Outgoing[0].Kind = "mutated"

	after := relationsOf(t, gameCatalog, relationsDaggerKey, "", "")
	if after.Outgoing[0].To != originalTo {
		t.Errorf("To = %+v, want the unmutated %+v", after.Outgoing[0].To, originalTo)
	}
	if after.Outgoing[0].Kind != schema.RelationCompatibleWithAshOfWar {
		t.Errorf("Kind = %q, want %q", after.Outgoing[0].Kind, schema.RelationCompatibleWithAshOfWar)
	}
}
