/*
Endpoint: SetEquippedSpells
EndpointID: set_equipped_spells
Purpose: Atomically sets spell order and validates the total Memory Slots cost.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: Spell z capability equipment.
Input variables: characterID, orderedResources, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetEquippedSpellsEndpointID is the stable backend identifier of SetEquippedSpells.
const SetEquippedSpellsEndpointID = "set_equipped_spells"

// SetEquippedSpellsDefinition describes the public mutation contract.
var SetEquippedSpellsDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetEquippedSpells",
	ID:                         SetEquippedSpellsEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Spell z capability equipment",
	SupportedResourceVariables: []string{"characterID", "orderedResources", "expectedRevision"},
	Description:                "Atomically sets spell order and validates the total Memory Slots cost.",
})
