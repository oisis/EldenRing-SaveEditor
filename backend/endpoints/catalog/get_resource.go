/*
Endpoint: GetResource
EndpointID: get_resource
Purpose: Zwraca pełny dokument jednego zasobu wraz z capabilities, wariantami, prezentacją i provenance.
How it works: The runtime handler validates resourceID, resolves it against the already loaded GameCatalog as an exact schema.Resource.Key, and returns a typed result built from an independent deep copy without loading, reloading or modifying the catalog.
Supported resource types: GameResource.
Input variables: resourceID.
GameCatalog variables read: the resource stored under the given Resource.Key including its full schema.ItemDocument (presentation, capabilities, safety, storage, acquisition, modifiers, links, variants, aliases, unlocks, related technical records, source records and family data).
Save variables read: none; the endpoint never opens or reads a save.
Implementation status: implemented; GetResource is the runtime handler of this contract.
*/
package catalog

import (
	"errors"
	"fmt"
	"strings"

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
	SupportedResourceVariables: []string{"resourceID"},
	Description:                "Zwraca pełny dokument jednego zasobu wraz z capabilities, wariantami, prezentacją i provenance.",
})

// GetResourceResult is the typed result of GetResource.
type GetResourceResult struct {
	Resource schema.Resource `json:"resource"`
}

// GetResource returns one catalog resource in full. Relations are not part of
// this result; they belong to GetResourceRelations. resourceID is the stable
// schema.Resource.Key, for example "item:80085CA0". It is neither a numeric
// schema.ResourceID, an Item.GameID, a variant ID, nor a data file name, and it
// is matched exactly: the endpoint declares no key format of its own and never
// parses, derives, or normalises the key, because schema declares no such
// format either. The result is the deep copy the catalog already returns, so
// mutating it never reaches the catalog.
func GetResource(gameCatalog *gamecatalog.Catalog, resourceID string) (GetResourceResult, error) {
	if gameCatalog == nil {
		return GetResourceResult{}, errors.New("game catalog is not loaded")
	}
	if resourceID == "" {
		return GetResourceResult{}, errors.New("resource ID is required")
	}
	if strings.TrimSpace(resourceID) != resourceID {
		return GetResourceResult{}, fmt.Errorf(
			"resource ID %q must not contain leading or trailing whitespace",
			resourceID,
		)
	}

	resource, exists := gameCatalog.ResourceByKey(resourceID)
	if !exists {
		return GetResourceResult{}, fmt.Errorf("resource ID %q was not found in the game catalog", resourceID)
	}

	return GetResourceResult{Resource: resource}, nil
}
