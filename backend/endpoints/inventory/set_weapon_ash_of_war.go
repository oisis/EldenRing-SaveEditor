/*
Endpoint: SetWeaponAshOfWar
EndpointID: set_weapon_ash_of_war
Purpose: Montuje, zmienia lub zdejmuje Ash of War po walidacji wszystkich relacji i skutków dla powiązanych instancji.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: Weapon, AshOfWar.
Input variables: characterID, weaponOwnedItemID, ashOfWarKind, ashOfWarKey, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetWeaponAshOfWarEndpointID is the stable backend identifier of SetWeaponAshOfWar.
const SetWeaponAshOfWarEndpointID = "set_weapon_ash_of_war"

// SetWeaponAshOfWarDefinition describes the public mutation contract.
var SetWeaponAshOfWarDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetWeaponAshOfWar",
	ID:                         SetWeaponAshOfWarEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Weapon, AshOfWar",
	SupportedResourceVariables: []string{"characterID", "weaponOwnedItemID", "ashOfWarKind", "ashOfWarKey", "expectedRevision"},
	Description:                "Montuje, zmienia lub zdejmuje Ash of War po walidacji wszystkich relacji i skutków dla powiązanych instancji.",
})
