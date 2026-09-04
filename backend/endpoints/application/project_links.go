/*
Endpoint: GetProjectLinks
EndpointID: get_project_links
Purpose: Returns the closed set of approved project links the About & Updates screen may open.
How it works: The runtime handler returns a compile-time table of link identifiers and their approved absolute URLs. Nothing is read from configuration, from a save or from the frontend.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: none.
Implementation status: implemented
*/
package application

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
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

// The four approved link identifiers. The frontend asks the host to open one of
// these identifiers and never states a URL: there is deliberately no host action
// that opens an arbitrary address the frontend supplies.
const (
	ProjectLinkRepository     = "repository"
	ProjectLinkReleases       = "releases"
	ProjectLinkSponsorCoffee  = "sponsor_coffee"
	ProjectLinkSponsorBitcoin = "sponsor_bitcoin"
)

// projectLinks is the whole allowlist. It is a compile-time table on purpose:
// an approved address is a product decision, not configuration, and nothing at
// runtime may add an entry to it.
var projectLinks = map[string]string{
	ProjectLinkRepository:     "https://github.com/oisis/EldenRing-SaveEditor",
	ProjectLinkReleases:       "https://github.com/oisis/EldenRing-SaveEditor/releases",
	ProjectLinkSponsorCoffee:  "https://buymeacoffee.com/oisisk",
	ProjectLinkSponsorBitcoin: "https://www.blockonomics.co/#/search?q=18FqJhKioiuxH859LU2pcpas2h46MGr9a2",
}

// projectLinkOrder is the order the About screen presents the links in.
var projectLinkOrder = []string{
	ProjectLinkRepository,
	ProjectLinkReleases,
	ProjectLinkSponsorCoffee,
	ProjectLinkSponsorBitcoin,
}

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
	links := make([]ProjectLink, 0, len(projectLinkOrder))
	for _, id := range projectLinkOrder {
		links = append(links, ProjectLink{ID: id, URL: projectLinks[id]})
	}
	return GetProjectLinksResult{Links: links}, nil
}

// ResolveProjectLink maps one approved identifier onto its absolute URL. An
// unknown identifier is rejected: this is the only way a URL is ever produced
// for the host to open, so it is also the only place that could turn frontend
// input into an outgoing address.
func ResolveProjectLink(linkID string) (string, error) {
	url, known := projectLinks[linkID]
	if !known {
		return "", fmt.Errorf("unknown project link %q", linkID)
	}
	return url, nil
}
