/*
Endpoint: GetProjectLinks
EndpointID: get_project_links
Purpose: Returns the closed set of approved project links the About & Updates screen may open.
How it works: The runtime handler returns the compile-time allowlist owned by the projectlinks package: link identifiers and their approved absolute URLs. Nothing is read from configuration, from a save or from the frontend.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: none.
Implementation status: implemented
*/
package application

import (
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/projectlinks"
)

// GetProjectLinksEndpointID is the stable backend identifier of GetProjectLinks.
const GetProjectLinksEndpointID = "get_project_links"

// GetProjectLinksDefinition describes the public getter contract.
var GetProjectLinksDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetProjectLinks",
	ID:                         GetProjectLinksEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: nil,
	Description:                "Returns the closed set of approved project links the About & Updates screen may open.",
})

// ProjectLink is one approved destination.
type ProjectLink struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// GetProjectLinksResult is the typed result of GetProjectLinks.
type GetProjectLinksResult struct {
	Links []ProjectLink `json:"links"`
}

// GetProjectLinks reports the approved links in their presentation order.
func GetProjectLinks() (GetProjectLinksResult, error) {
	approved := projectlinks.All()
	links := make([]ProjectLink, 0, len(approved))
	for _, link := range approved {
		links = append(links, ProjectLink{ID: link.ID, URL: link.URL})
	}
	return GetProjectLinksResult{Links: links}, nil
}
