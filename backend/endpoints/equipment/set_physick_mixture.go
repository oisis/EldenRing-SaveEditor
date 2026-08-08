/*
Endpoint: SetPhysickMixture
EndpointID: set_physick_mixture
Purpose: Atomowo ustawia obie pozycje Flask of Wondrous Physick.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: CrystalTear.
Input variables: characterID, crystalTearResources, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetPhysickMixtureEndpointID is the stable backend identifier of SetPhysickMixture.
const SetPhysickMixtureEndpointID = "set_physick_mixture"

// SetPhysickMixtureDefinition describes the public mutation contract.
var SetPhysickMixtureDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetPhysickMixture",
	ID:                         SetPhysickMixtureEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: CrystalTear",
	SupportedResourceVariables: []string{"characterID", "crystalTearResources", "expectedRevision"},
	Description:                "Atomowo ustawia obie pozycje Flask of Wondrous Physick.",
})
