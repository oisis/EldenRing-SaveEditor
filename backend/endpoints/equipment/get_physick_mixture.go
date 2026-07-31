/*
Endpoint: GetPhysickMixture
EndpointID: get_physick_mixture
Purpose: Zwraca obie pozycje aktualnej mieszanki Flask of Wondrous Physick.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument: CrystalTear.
Input variables: characterID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetPhysickMixtureEndpointID is the stable backend identifier of GetPhysickMixture.
const GetPhysickMixtureEndpointID = "get_physick_mixture"

// GetPhysickMixtureDefinition describes the public getter contract.
var GetPhysickMixtureDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetPhysickMixture",
	ID:                         GetPhysickMixtureEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: CrystalTear",
	SupportedResourceVariables: []string{"characterID"},
	Description:                "Zwraca obie pozycje aktualnej mieszanki Flask of Wondrous Physick.",
})
