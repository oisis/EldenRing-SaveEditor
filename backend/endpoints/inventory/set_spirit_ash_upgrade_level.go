/*
Endpoint: SetSpiritAshUpgradeLevel
EndpointID: set_spirit_ash_upgrade_level
Purpose: Sets the Spirit Ash upgrade level according to its applicable model and limit.
How it works: The runtime handler validates the complete request and expected revision, resolves catalog resources when applicable, and delegates one atomic operation to SaveEngine.
Supported resource types: ItemDocument: SpiritAsh z capability upgrade.
Input variables: characterID, ownedItemID, upgradeLevel, expectedRevision.
GameCatalog variables read: the fields required to resolve and validate the declared resource types; the exact projection belongs to the endpoint runtime specification.
Save variables processed: the state required by the declared variables; the mutation must validate a complete plan and finish with full success or rollback.
Implementation status: contract definition only; no runtime handler is implemented in this file yet.
*/
package inventory

import "github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"

// SetSpiritAshUpgradeLevelEndpointID is the stable backend identifier of SetSpiritAshUpgradeLevel.
const SetSpiritAshUpgradeLevelEndpointID = "set_spirit_ash_upgrade_level"

// SetSpiritAshUpgradeLevelDefinition describes the public mutation contract.
var SetSpiritAshUpgradeLevelDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetSpiritAshUpgradeLevel",
	ID:                         SetSpiritAshUpgradeLevelEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: SpiritAsh z capability upgrade",
	SupportedResourceVariables: []string{"characterID", "ownedItemID", "upgradeLevel", "expectedRevision"},
	Description:                "Sets the Spirit Ash upgrade level according to its applicable model and limit.",
})
