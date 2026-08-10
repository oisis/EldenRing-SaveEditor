/*
Endpoint: GetResourceRelations
EndpointID: get_resource_relations
Purpose: Returns the outgoing and incoming relations of the specified resource.
How it works: The runtime handler validates gameCatalog, kind, key, relationType and direction, resolves the exact (kind, key) pair against the already loaded GameCatalog, and returns the outgoing and incoming relations of that resource as stored, in catalog order, taken from the independent copies the catalog returns; it filters only by the exact relationType and direction it was given and never sorts, normalises, deduplicates, or synthesises a relation, never returns the documents of related resources, and never loads, reloads or modifies the catalog.
Supported resource types: GameResource.
Input variables: kind, key, relationType, direction.
GameCatalog variables read: Resource.Kind and Resource.Key to resolve the resource, and the outgoing and incoming schema.Relation entries of that resource with their From, To, Kind and Provenance.
Save variables read: none; the endpoint never opens or reads a save.
Implementation status: implemented; GetResourceRelations is the runtime handler of this contract.
*/
package catalog

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// GetResourceRelationsEndpointID is the stable backend identifier of GetResourceRelations.
const GetResourceRelationsEndpointID = "get_resource_relations"

// The two accepted direction filters. An empty direction means both directions;
// there is deliberately no "both" value, because the empty filter already is it.
const (
	relationDirectionOutgoing = "outgoing"
	relationDirectionIncoming = "incoming"
)

// GetResourceRelationsDefinition describes the public getter contract.
var GetResourceRelationsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetResourceRelations",
	ID:                         GetResourceRelationsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource",
	SupportedResourceVariables: []string{"kind", "key", "relationType", "direction"},
	Description:                "Returns the outgoing and incoming relations of the specified resource.",
})

// GetResourceRelationsResult is the typed result of GetResourceRelations.
type GetResourceRelationsResult struct {
	Outgoing []schema.Relation `json:"outgoing"`
	Incoming []schema.Relation `json:"incoming"`
}

// GetResourceRelations returns the relations of one catalog resource. The
// resource identity is the pair (kind, key), for example kind "item" and key
// "000F4240": the kind is resolved first and the key is matched exactly inside
// that kind only. Neither value is trimmed, normalised, parsed, or retried under
// another kind, and an unknown kind and an unknown key stay two distinct errors.
//
// relationType filters by an exact schema.RelationKind; an empty relationType
// means every kind. direction is "outgoing", "incoming", or empty for both; the
// direction that is filtered out stays an empty array. Both filters are matched
// exactly and case-sensitively and are never trimmed or normalised.
//
// A resource without relations is a valid case and returns two empty arrays,
// never null. The relations are returned in catalog order and carry their full
// provenance, including the ResourceRef in From and To; the documents of the
// related resources are not part of this result and belong to GetResource. The
// relations are the independent copies the catalog returns, so mutating them
// never reaches the catalog.
func GetResourceRelations(
	gameCatalog *gamecatalog.Catalog,
	kind string,
	key string,
	relationType string,
	direction string,
) (GetResourceRelationsResult, error) {
	if gameCatalog == nil {
		return GetResourceRelationsResult{}, errors.New("game catalog is not loaded")
	}
	if kind == "" {
		return GetResourceRelationsResult{}, errors.New("resource kind is required")
	}
	if key == "" {
		return GetResourceRelationsResult{}, errors.New("resource key is required")
	}
	switch schema.RelationKind(relationType) {
	case "", schema.RelationCompatibleWithAshOfWar, schema.RelationRequiresContainer:
	default:
		return GetResourceRelationsResult{}, fmt.Errorf(
			"relation type %q is not supported; use %q, %q or an empty value for every relation type",
			relationType,
			schema.RelationCompatibleWithAshOfWar,
			schema.RelationRequiresContainer,
		)
	}
	switch direction {
	case "", relationDirectionOutgoing, relationDirectionIncoming:
	default:
		return GetResourceRelationsResult{}, fmt.Errorf(
			"direction %q is not supported; use %q, %q or an empty value for both directions",
			direction,
			relationDirectionOutgoing,
			relationDirectionIncoming,
		)
	}

	outgoing, incoming, err := gameCatalog.RelationsByKindAndKey(schema.ResourceKind(kind), key)
	if err != nil {
		return GetResourceRelationsResult{}, err
	}

	result := GetResourceRelationsResult{
		Outgoing: []schema.Relation{},
		Incoming: []schema.Relation{},
	}
	if direction != relationDirectionIncoming {
		for _, relation := range outgoing {
			if relationType == "" || string(relation.Kind) == relationType {
				result.Outgoing = append(result.Outgoing, relation)
			}
		}
	}
	if direction != relationDirectionOutgoing {
		for _, relation := range incoming {
			if relationType == "" || string(relation.Kind) == relationType {
				result.Incoming = append(result.Incoming, relation)
			}
		}
	}
	return result, nil
}
