/*
Endpoint: SetWeaponUpgradeLevel
EndpointID: set_weapon_upgrade_level
Purpose: Sets the upgrade level of an owned weapon according to its upgrade model and catalog limit.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: Weapon z capability upgrade.
Input variables: characterID, ownedItemID, upgradeLevel, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetWeaponUpgradeLevelEndpointID is the stable backend identifier of SetWeaponUpgradeLevel.
const SetWeaponUpgradeLevelEndpointID = "set_weapon_upgrade_level"

// SetWeaponUpgradeLevelDefinition describes the public mutation contract.
var SetWeaponUpgradeLevelDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetWeaponUpgradeLevel",
	ID:                         SetWeaponUpgradeLevelEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Weapon z capability upgrade",
	SupportedResourceVariables: []string{"characterID", "ownedItemID", "upgradeLevel", "expectedRevision"},
	Description:                "Sets the upgrade level of an owned weapon according to its upgrade model and catalog limit.",
})
