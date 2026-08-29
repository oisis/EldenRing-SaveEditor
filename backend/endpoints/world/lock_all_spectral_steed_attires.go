/*
Endpoint: LockAllSpectralSteedAttires
EndpointID: lock_all_spectral_steed_attires
Purpose: Removes every Spectral Steed Attire item from Inventory and restores the default appearance of Torrent.
How it works: The handler resolves the shared appearance table against GameCatalog and delegates one atomic mutation to SaveEngine, which plans the removal of the three attire Inventory records and the appearance flags together and applies them as a single verified byte plan.
Supported resource types: ItemDocument: Spectral Steed Attire.
Input variables: saveSessionID, characterID, expectedRevision.
GameCatalog variables read: resource kind and key plus item family and gameID of the three Regulation 1.17 attire items.
Save variables processed: the Inventory records of the three attire items and the four mutually exclusive Spectral Steed Attire event flags. Unrelated Inventory records are never touched.
Implementation status: implemented
*/
package world

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// LockAllSpectralSteedAttiresEndpointID is the stable backend identifier of LockAllSpectralSteedAttires.
const LockAllSpectralSteedAttiresEndpointID = "lock_all_spectral_steed_attires"

// LockAllSpectralSteedAttiresDefinition describes the public mutation contract.
var LockAllSpectralSteedAttiresDefinition = contract.MustDefine(contract.Definition{
	Name:                       "LockAllSpectralSteedAttires",
	ID:                         LockAllSpectralSteedAttiresEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "ItemDocument: Spectral Steed Attire",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "expectedRevision"},
	Description:                "Removes every Spectral Steed Attire item from Inventory and restores the default appearance of Torrent.",
})

// LockAllSpectralSteedAttiresResult reports the committed reset in public terms.
// AttireKey is always the default appearance, which is the only state this
// operation can leave behind.
type LockAllSpectralSteedAttiresResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	AttireKey     string `json:"attireKey"`
}

// LockAllSpectralSteedAttires removes the three attire items from Inventory and
// selects the default appearance in one revision.
//
// It is deliberately a dedicated endpoint rather than a client-side sequence of
// RemoveOwnedItem and SetSpectralSteedAttire calls: a failure between those calls
// would leave the save wearing an appearance whose item is gone. Inventory and
// event flags therefore move together or not at all. Inventory records that
// belong to any other item are preserved.
func LockAllSpectralSteedAttires(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	expectedRevision string,
) (LockAllSpectralSteedAttiresResult, error) {
	if engine == nil {
		return LockAllSpectralSteedAttiresResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return LockAllSpectralSteedAttiresResult{}, errors.New("game catalog is not available")
	}

	declared, err := catalogSpectralSteedAttires(gameCatalog)
	if err != nil {
		return LockAllSpectralSteedAttiresResult{}, err
	}
	mutation, err := engine.LockAllSpectralSteedAttires(
		saveSessionID, characterID, spectralSteedAttireStates(declared), expectedRevision)
	if err != nil {
		return LockAllSpectralSteedAttiresResult{}, err
	}
	return LockAllSpectralSteedAttiresResult{
		SaveSessionID: mutation.SaveSessionID,
		SaveRevision:  mutation.SaveRevision,
		CharacterID:   mutation.CharacterID,
		AttireKey:     SpectralSteedAttireKeyDefault,
	}, nil
}
