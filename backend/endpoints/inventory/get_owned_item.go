/*
Endpoint: GetOwnedItem
EndpointID: get_owned_item
Purpose: Returns one owned item instance addressed by its opaque, revision-scoped OwnedItemID, resolved to its ItemDocument identity.
How it works: The runtime handler resolves the identity and reads the single physical record through SaveEngine, then resolves its one save-side game ID through the already loaded GameCatalog. The endpoint opens no file, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument.
Input variables: saveSessionID, characterID, ownedItemID.
GameCatalog variables read: the kind and key of the one ItemDocument resolved by the record's save-side game ID.
Save variables read: the physical record the identity was minted for, taken from the container that minted it, plus the GaItem table of that slot; the getter is non-mutating, keeps gaItemHandle and acquisitionIndex raw and masks only the documented high bit of quantity.
Implementation status: implemented
*/
package inventory

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetOwnedItemEndpointID is the stable backend identifier of GetOwnedItem.
const GetOwnedItemEndpointID = "get_owned_item"

// GetOwnedItemDefinition describes the public getter contract.
var GetOwnedItemDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetOwnedItem",
	ID:                         GetOwnedItemEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "ownedItemID"},
	Description: "Returns details of one owned item instance addressed by its opaque, " +
		"session- and revision-scoped OwnedItemID.",
})

// GetOwnedItemResult is one owned physical record plus its public catalog
// identity. GameID is the exact catalog game ID resolved from the save. Stored
// variants preserve affinity and a confirmed weapon-upgrade range resolves its
// level; Kind and Key remain the canonical resource reference.
//
// OwnedItemID is echoed back exactly as supplied and stays opaque: it is valid
// only inside the session and the SaveRevision that issued it.
type GetOwnedItemResult struct {
	SaveSessionID    string              `json:"saveSessionID"`
	SaveRevision     string              `json:"saveRevision"`
	OwnedItemID      string              `json:"ownedItemID"`
	CharacterID      int                 `json:"characterID"`
	Kind             schema.ResourceKind `json:"kind"`
	Key              string              `json:"key"`
	GameID           uint32              `json:"gameID"`
	Container        string              `json:"container"`
	ContainerSection string              `json:"containerSection"`
	PhysicalIndex    int                 `json:"physicalIndex"`
	GaItemHandle     uint32              `json:"gaItemHandle"`
	Quantity         uint32              `json:"quantity"`
	AcquisitionIndex uint32              `json:"acquisitionIndex"`
}

// GetOwnedItem returns the one owned item instance ownedItemID was minted for,
// resolved to one ItemDocument by its GaItem-backed game ID. The result retains
// the physical fields of the record, but no name, family filter, equipped state
// or capacity is added here.
//
// An unresolvable handle or a game ID absent from GameCatalog rejects the whole
// request: no placeholder document, partial identity or substitute item is ever
// returned.
func GetOwnedItem(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	ownedItemID string,
) (GetOwnedItemResult, error) {
	if engine == nil {
		return GetOwnedItemResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetOwnedItemResult{}, errors.New("game catalog is not available")
	}

	owned, err := engine.GetOwnedItem(saveSessionID, characterID, ownedItemID)
	if err != nil {
		return GetOwnedItemResult{}, err
	}

	gameIDs, err := engine.ResolveGaItemIDs(saveSessionID, characterID, []uint32{owned.GaItemHandle})
	if err != nil {
		return GetOwnedItemResult{}, err
	}
	resource, exists := gameCatalog.ItemByGameID(gameIDs[0])
	if !exists || resource.Kind != schema.ResourceKindItem || resource.Item == nil || resource.Key == "" {
		return GetOwnedItemResult{}, fmt.Errorf("owned item %q: game ID 0x%08X is not a known item",
			ownedItemID, gameIDs[0])
	}

	return GetOwnedItemResult{
		SaveSessionID:    owned.SaveSessionID,
		SaveRevision:     owned.SaveRevision,
		OwnedItemID:      owned.OwnedItemID,
		CharacterID:      owned.CharacterID,
		Kind:             resource.Kind,
		Key:              resource.Key,
		GameID:           gameIDs[0],
		Container:        owned.Container,
		ContainerSection: owned.ContainerSection,
		PhysicalIndex:    owned.PhysicalIndex,
		GaItemHandle:     owned.GaItemHandle,
		Quantity:         owned.Quantity,
		AcquisitionIndex: owned.AcquisitionIndex,
	}, nil
}
