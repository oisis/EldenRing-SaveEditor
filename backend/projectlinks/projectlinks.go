// Package projectlinks owns the closed allowlist of approved project addresses
// the host may open.
//
// The allowlist lives outside the endpoint packages because two layers need it:
// the GetProjectLinks endpoint presents it, and the desktop bridge resolves one
// identifier into the address it hands to the host browser. Keeping one table
// here is what makes "the frontend never states a URL" a property of the whole
// application rather than of one endpoint file.
package projectlinks

import "fmt"

// The four approved link identifiers. The frontend asks the host to open one of
// these identifiers and never states a URL: there is deliberately no host action
// that opens an arbitrary address the frontend supplies.
const (
	Repository     = "repository"
	Releases       = "releases"
	SponsorCoffee  = "sponsor_coffee"
	SponsorBitcoin = "sponsor_bitcoin"
)

// Link is one approved destination.
type Link struct {
	ID  string
	URL string
}

// approved is the whole allowlist, in the order the About screen presents it.
// It is a compile-time table on purpose: an approved address is a product
// decision, not configuration, and nothing at runtime may add an entry to it.
var approved = []Link{
	{ID: Repository, URL: "https://github.com/oisis/EldenRing-SaveEditor"},
	{ID: Releases, URL: "https://github.com/oisis/EldenRing-SaveEditor/releases"},
	{ID: SponsorCoffee, URL: "https://buymeacoffee.com/oisisk"},
	{ID: SponsorBitcoin, URL: "https://www.blockonomics.co/#/search?q=18FqJhKioiuxH859LU2pcpas2h46MGr9a2"},
}

// All reports the approved links in their presentation order.
func All() []Link {
	links := make([]Link, len(approved))
	copy(links, approved)
	return links
}

// Resolve maps one approved identifier onto its absolute URL. An unknown
// identifier is rejected: this is the only way a URL is ever produced for the
// host to open, so it is also the only place that could turn frontend input
// into an outgoing address.
func Resolve(linkID string) (string, error) {
	// ponytail: linear scan over four entries; the table is the ordering too.
	for _, link := range approved {
		if link.ID == linkID {
			return link.URL, nil
		}
	}
	return "", fmt.Errorf("unknown project link %q", linkID)
}
