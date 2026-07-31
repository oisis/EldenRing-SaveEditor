/*
Endpoint: GetItemVariants
EndpointID: get_item_variants
Purpose: Zwraca wszystkie dozwolone warianty itemu, w tym poziomy ulepszenia i infusion, bez wyliczania ich po stronie konsumenta.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument.
Input variables: resourceID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package catalog

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

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
