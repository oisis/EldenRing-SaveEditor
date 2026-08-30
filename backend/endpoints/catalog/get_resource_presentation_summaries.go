/*
Endpoint: GetResourcePresentationSummaries
EndpointID: get_resource_presentation_summaries
Purpose: Returns lightweight presentation metadata for an ordered batch of exact GameCatalog resource identities.
How it works: The runtime handler resolves each exact kind and key through Catalog.ResourceSummaryByKindAndKey, preserves input order and duplicates, and projects only kind, key, name and iconPath. If any identity is invalid, the whole request fails without returning a partial result.
Supported resource types: GameResource.
Input variables: identities.
GameCatalog variables read: Resource.Kind, Resource.Key, Item.Presentation.Name, Item.Presentation.IconPath, Colosseum.Name, Region.Name, SummoningPool.Name, Grace.Name, Boss.Name, MapRegion.Name, Tutorial.Title, Quest.Name and Class.Name.
Save variables read: none; the endpoint never opens or reads a save.
Implementation status: implemented; GetResourcePresentationSummaries is the runtime handler of this contract.
*/
package catalog

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const GetResourcePresentationSummariesEndpointID = "get_resource_presentation_summaries"

var GetResourcePresentationSummariesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetResourcePresentationSummaries",
	ID:                         GetResourcePresentationSummariesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource",
	SupportedResourceVariables: []string{"identities"},
	Description:                "Returns lightweight presentation metadata for an ordered batch of exact GameCatalog resource identities.",
})

// ResourcePresentationIdentity is one exact catalog identity. Neither field is
// trimmed, normalised, recased or resolved through an alias.
type ResourcePresentationIdentity struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}

// ResourcePresentationSummary is deliberately scalar-only. Unknown names and
// icon paths stay empty; the endpoint never invents display data.
type ResourcePresentationSummary struct {
	Kind     string `json:"kind"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	IconPath string `json:"iconPath"`
}

type GetResourcePresentationSummariesResult struct {
	Resources []ResourcePresentationSummary `json:"resources"`
}

// GetResourcePresentationSummaries resolves every identity atomically and
// preserves the caller's order and duplicates. Exact lookup deliberately keeps
// noDatabase resources reachable for the owning features that already know
// their identities; it does not turn this endpoint into a general catalog.
func GetResourcePresentationSummaries(
	gameCatalog *gamecatalog.Catalog,
	identities []ResourcePresentationIdentity,
) (GetResourcePresentationSummariesResult, error) {
	if gameCatalog == nil {
		return GetResourcePresentationSummariesResult{}, errors.New("game catalog is not loaded")
	}

	resources := make([]ResourcePresentationSummary, 0, len(identities))
	for index, identity := range identities {
		summary, err := gameCatalog.ResourceSummaryByKindAndKey(
			schema.ResourceKind(identity.Kind), identity.Key)
		if err != nil {
			return GetResourcePresentationSummariesResult{}, fmt.Errorf("identity %d: %w", index, err)
		}

		presentation := ResourcePresentationSummary{
			Kind: string(summary.Kind),
			Key:  summary.Key,
		}
		if summary.NameKnown {
			presentation.Name = summary.Name
		}
		if summary.IconPathKnown {
			presentation.IconPath = summary.IconPath
		}
		resources = append(resources, presentation)
	}

	return GetResourcePresentationSummariesResult{Resources: resources}, nil
}
