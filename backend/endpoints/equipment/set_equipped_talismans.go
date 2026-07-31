/*
Endpoint: SetEquippedTalismans
EndpointID: set_equipped_talismans
Purpose: Atomowo ustawia talismany z uwzględnieniem liczby odblokowanych slotów.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: Talisman z capability equipment.
Input variables: characterID, orderedOwnedItemIDs, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetEquippedTalismansEndpointID is the stable backend identifier of SetEquippedTalismans.
const SetEquippedTalismansEndpointID = "set_equipped_talismans"

// SetEquippedTalismansDefinition describes the public mutation contract.
var SetEquippedTalismansDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetEquippedTalismans",
	ID:                         SetEquippedTalismansEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Talisman z capability equipment",
	SupportedResourceVariables: []string{"characterID", "orderedOwnedItemIDs", "expectedRevision"},
	Description:                "Atomowo ustawia talismany z uwzględnieniem liczby odblokowanych slotów.",
})
