/*
Endpoint: GetEquipment
EndpointID: get_equipment
Purpose: Zwraca wyposażone armamenty, armor i talismany wraz z ograniczeniami slotów wynikającymi z GameCatalog.
How it works: The runtime handler reads only through the responsible backend owners and returns a typed result without modifying save or application state.
Supported resource types: ItemDocument: Weapon, Armor, Talisman.
Input variables: characterID.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables read: the state required by the declared variables; the getter must remain non-mutating.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package equipment

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// GetEquipmentEndpointID is the stable backend identifier of GetEquipment.
const GetEquipmentEndpointID = "get_equipment"

// GetEquipmentDefinition describes the public getter contract.
var GetEquipmentDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetEquipment",
	ID:                         GetEquipmentEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument: Weapon, Armor, Talisman",
	SupportedResourceVariables: []string{"characterID"},
	Description:                "Zwraca wyposażone armamenty, armor i talismany wraz z ograniczeniami slotów wynikającymi z GameCatalog.",
})
