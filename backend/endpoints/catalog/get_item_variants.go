/*
Endpoint: GetItemVariants
EndpointID: get_item_variants
Purpose: Returns all allowed item variants, including upgrade levels and infusions, without requiring consumers to derive them.
How it works: The runtime handler validates kind and key, resolves the exact (kind, key) pair against the already loaded GameCatalog, and returns the variants stored in that item document in catalog order, taken from the independent deep copy the catalog returns; it never materialises, filters, sorts, normalises, or synthesises a variant, and it never loads or modifies the catalog.
Supported resource types: ItemDocument.
Input variables: kind, key.
GameCatalog variables read: Resource.Kind and Resource.Key to resolve the resource, and Item.Variants of that resource with every variant Fact, provenance, Data, and SourceRecords.
Save variables read: none; the endpoint never opens or reads a save.
Implementation status: implemented; GetItemVariants is the runtime handler of this contract.
*/
package catalog

import (
	"errors"
	"fmt"

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
	SupportedResourceVariables: []string{"kind", "key"},
	Description:                "Returns all allowed item variants, including upgrade levels and infusions, without requiring consumers to derive them.",
})

// GetItemVariantsResult is the typed result of GetItemVariants.
type GetItemVariantsResult struct {
	Variants []schema.ItemVariant `json:"variants"`
}

// GetItemVariants returns the variants stored in one item document. The
// resource identity is the pair (kind, key), for example kind "item" and key
// "000F4240": only the item kind carries variants, and the key is matched
// exactly inside that kind. Neither value is trimmed, normalised, parsed, or
// retried under another kind, so the pre-migration key "item:000F4240" is an
// unknown key and never an alias. A missing kind, a kind other than item, a
// missing key and a key that is unknown inside the item kind are four distinct
// errors. The result carries Item.Variants as stored, in catalog order and with
// full provenance; the base item is not part of Item.Variants and is never
// synthesised into an extra variant. An item without variants is a valid case
// and returns an empty slice. The variants are the deep copy the catalog
// already returns, so mutating them never reaches the catalog.
func GetItemVariants(gameCatalog *gamecatalog.Catalog, kind string, key string) (GetItemVariantsResult, error) {
	if gameCatalog == nil {
		return GetItemVariantsResult{}, errors.New("game catalog is not loaded")
	}
	if kind == "" {
		return GetItemVariantsResult{}, errors.New("resource kind is required")
	}
	if kind != string(schema.ResourceKindItem) {
		return GetItemVariantsResult{}, fmt.Errorf(
			"resource kind %q has no item variants; only kind %q is supported",
			kind,
			schema.ResourceKindItem,
		)
	}
	if key == "" {
		return GetItemVariantsResult{}, errors.New("resource key is required")
	}

	resource, err := gameCatalog.ResourceByKindAndKey(schema.ResourceKindItem, key)
	if err != nil {
		return GetItemVariantsResult{}, err
	}
	if resource.Item == nil {
		return GetItemVariantsResult{}, fmt.Errorf(
			"resource kind %q key %q is not an item and has no variants",
			kind,
			key,
		)
	}
	if resource.Item.Variants == nil {
		return GetItemVariantsResult{Variants: []schema.ItemVariant{}}, nil
	}
	return GetItemVariantsResult{Variants: resource.Item.Variants}, nil
}
