/*
Endpoint: GetResource
EndpointID: get_resource
Purpose: Returns the complete document of one resource, including capabilities, variants, presentation, and provenance.
How it works: The runtime handler validates kind and key, resolves the exact (kind, key) pair against the already loaded GameCatalog, and returns a typed result built from an independent deep copy without loading, reloading or modifying the catalog.
Supported resource types: GameResource.
Input variables: kind, key.
GameCatalog variables read: the resource stored under the given (Resource.Kind, Resource.Key) pair including the full document of its kind: for kind item the schema.ItemDocument (presentation, capabilities, safety, storage, acquisition, modifiers, links, variants, aliases, unlocks, related technical records, source records and family data), for kind colosseum the schema.ColosseumDocument (name and unlock event flag ID, each with its own provenance), for kind region the schema.RegionDocument (region ID, name and area, each with its own provenance), for kind summoning_pool the schema.SummoningPoolDocument (name, region label and activation event flag ID, each with its own provenance).
Save variables read: none; the endpoint never opens or reads a save.
Implementation status: implemented; GetResource is the runtime handler of this contract.
*/
package catalog

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// GetResourceEndpointID is the stable backend identifier of GetResource.
const GetResourceEndpointID = "get_resource"

// GetResourceDefinition describes the public getter contract.
var GetResourceDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetResource",
	ID:                         GetResourceEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource",
	SupportedResourceVariables: []string{"kind", "key"},
	Description:                "Returns the complete document of one resource, including capabilities, variants, presentation, and provenance.",
})

// GetResourceResult is the typed result of GetResource.
type GetResourceResult struct {
	Resource schema.Resource `json:"resource"`
}

// GetResource returns one catalog resource in full, whatever its kind: the
// result carries the document of the resolved kind and no document of any
// other. Relations are not part of this result; they belong to
// GetResourceRelations. The resource identity is the pair (kind, key): the kind
// is resolved first and the key is matched exactly inside that kind only.
// Neither value is trimmed, normalised, parsed, or retried under another kind,
// so the pre-migration key "item:000F4240" is an unknown key and never an alias. A
// missing kind, an unknown kind, a missing key and a key that is unknown inside
// an existing kind are four distinct errors. The result is the deep copy the
// catalog already returns, so mutating it never reaches the catalog.
func GetResource(gameCatalog *gamecatalog.Catalog, kind string, key string) (GetResourceResult, error) {
	if gameCatalog == nil {
		return GetResourceResult{}, errors.New("game catalog is not loaded")
	}
	if kind == "" {
		return GetResourceResult{}, errors.New("resource kind is required")
	}
	if key == "" {
		return GetResourceResult{}, errors.New("resource key is required")
	}

	resource, err := gameCatalog.ResourceByKindAndKey(schema.ResourceKind(kind), key)
	if err != nil {
		return GetResourceResult{}, err
	}

	return GetResourceResult{Resource: resource}, nil
}
