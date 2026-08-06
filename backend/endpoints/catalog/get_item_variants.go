/*
Endpoint: GetItemVariants
EndpointID: get_item_variants
Purpose: Zwraca wszystkie dozwolone warianty itemu, w tym poziomy ulepszenia i infusion, bez wyliczania ich po stronie konsumenta.
How it works: The runtime handler validates resourceID, resolves it against the already loaded GameCatalog as an exact schema.Resource.Key, and returns the variants stored in that item document in catalog order, taken from the independent deep copy the catalog returns; it never materialises, filters, sorts, normalises, or synthesises a variant, and it never loads or modifies the catalog.
Supported resource types: ItemDocument.
Input variables: resourceID.
GameCatalog variables read: Resource.Key to resolve the resource, and Item.Variants of that resource with every variant Fact, provenance, Data, and SourceRecords.
Save variables read: none; the endpoint never opens or reads a save.
Implementation status: implemented; GetItemVariants is the runtime handler of this contract.
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

// GetItemVariantsEndpointID is the stable backend identifier of GetItemVariants.
const GetItemVariantsEndpointID = "get_item_variants"

// GetItemVariantsDefinition describes the public getter contract.
var GetItemVariantsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetItemVariants",
	ID:                         GetItemVariantsEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"resourceID"},
	Description:                "Zwraca wszystkie dozwolone warianty itemu, w tym poziomy ulepszenia i infusion, bez wyliczania ich po stronie konsumenta.",
})

// GetItemVariantsResult is the typed result of GetItemVariants.
type GetItemVariantsResult struct {
	Variants []schema.ItemVariant `json:"variants"`
}

// GetItemVariants returns the variants stored in one item document. resourceID
// is the stable schema.Resource.Key, for example "item:000F4240". It is neither
// a numeric schema.ResourceID, an Item.GameID, a variant ID, nor a data file
// name, and it is matched exactly: the endpoint declares no key format of its
// own and never parses, derives, or normalises the key. The result carries
// Item.Variants as stored, in catalog order and with full provenance; the base
// item is not part of Item.Variants and is never synthesised into an extra
// variant. An item without variants is a valid case and returns an empty slice.
// The variants are the deep copy the catalog already returns, so mutating them
// never reaches the catalog.
func GetItemVariants(gameCatalog *gamecatalog.Catalog, resourceID string) (GetItemVariantsResult, error) {
	if gameCatalog == nil {
		return GetItemVariantsResult{}, errors.New("game catalog is not loaded")
	}
	if resourceID == "" {
		return GetItemVariantsResult{}, errors.New("resource ID is required")
	}
	if strings.TrimSpace(resourceID) != resourceID {
		return GetItemVariantsResult{}, fmt.Errorf(
			"resource ID %q must not contain leading or trailing whitespace",
			resourceID,
		)
	}

	resource, exists := gameCatalog.ResourceByKey(resourceID)
	if !exists {
		return GetItemVariantsResult{}, fmt.Errorf("resource ID %q was not found in the game catalog", resourceID)
	}
	if resource.Item == nil {
		return GetItemVariantsResult{}, fmt.Errorf("resource ID %q is not an item and has no variants", resourceID)
	}
	if resource.Item.Variants == nil {
		return GetItemVariantsResult{Variants: []schema.ItemVariant{}}, nil
	}
	return GetItemVariantsResult{Variants: resource.Item.Variants}, nil
}
