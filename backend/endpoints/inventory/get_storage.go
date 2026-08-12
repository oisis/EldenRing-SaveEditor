/*
Endpoint: GetStorage
EndpointID: get_storage
Purpose: Returns Storage Box records from one character slot resolved to ItemDocument identities.
How it works: The runtime handler reads the records and their save-side game IDs through SaveEngine, then resolves each ID through the already loaded GameCatalog. The endpoint opens no file, parses no save data of its own and calls no other endpoint.
Supported resource types: ItemDocument.
Input variables: saveSessionID, characterID, containerSection, page, pageSize.
GameCatalog variables read: the kind and key of every ItemDocument resolved by its save-side game ID.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the physical Storage Box records and GaItem table; the getter is non-mutating, keeps gaItemHandle and acquisitionIndex raw, masks only the documented high bit of quantity and applies paging.
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

// GetStorageEndpointID is the stable backend identifier of GetStorage.
const GetStorageEndpointID = "get_storage"

// GetStorageDefinition describes the public getter contract.
var GetStorageDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetStorage",
	ID:                         GetStorageEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "containerSection", "page", "pageSize"},
	Description:                "Returns Storage Box records from one character slot resolved to ItemDocument identities.",
})

// StorageRecord is one physical Storage Box row plus its public catalog
// identity. GameID is the exact catalog game ID resolved from the save; for an
// upgraded or infused item it selects the materialised catalog variant, while
// Kind and Key remain the canonical resource reference.
type StorageRecord struct {
	OwnedItemID      string              `json:"ownedItemID"`
	Kind             schema.ResourceKind `json:"kind"`
	Key              string              `json:"key"`
	GameID           uint32              `json:"gameID"`
	ContainerSection string              `json:"containerSection"`
	PhysicalIndex    int                 `json:"physicalIndex"`
	GaItemHandle     uint32              `json:"gaItemHandle"`
	Quantity         uint32              `json:"quantity"`
	AcquisitionIndex uint32              `json:"acquisitionIndex"`
}

// GetStorageResult is one resolved page of Storage Box records.
type GetStorageResult struct {
	SaveSessionID string          `json:"saveSessionID"`
	SaveRevision  string          `json:"saveRevision"`
	CharacterID   int             `json:"characterID"`
	Active        bool            `json:"active"`
	Records       []StorageRecord `json:"records"`
	Total         int             `json:"total"`
	Page          int             `json:"page"`
	PageSize      int             `json:"pageSize"`
}

// GetStorage returns one page of Storage Box records stored in one character
// slot of an existing save session. Every listed record is resolved to one
// ItemDocument by its GaItem-backed game ID. The result retains the physical
// fields from the raw reader, but no name, family filter, capacity or Inventory
// record is added here.
func GetStorage(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	containerSection string,
	page int,
	pageSize int,
) (GetStorageResult, error) {
	if engine == nil {
		return GetStorageResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetStorageResult{}, errors.New("game catalog is not available")
	}

	stored, err := engine.GetStorage(saveSessionID, characterID, containerSection, page, pageSize)
	if err != nil {
		return GetStorageResult{}, err
	}
	result := GetStorageResult{
		SaveSessionID: stored.SaveSessionID,
		SaveRevision:  stored.SaveRevision,
		CharacterID:   stored.CharacterID,
		Active:        stored.Active,
		Records:       make([]StorageRecord, 0, len(stored.Records)),
		Total:         stored.Total,
		Page:          stored.Page,
		PageSize:      stored.PageSize,
	}
	if !stored.Active {
		return result, nil
	}

	handles := make([]uint32, len(stored.Records))
	for index, record := range stored.Records {
		handles[index] = record.GaItemHandle
	}
	gameIDs, err := engine.ResolveGaItemIDs(saveSessionID, characterID, handles)
	if err != nil {
		return GetStorageResult{}, err
	}
	for index, record := range stored.Records {
		resource, exists := gameCatalog.ItemByGameID(gameIDs[index])
		if !exists || resource.Kind != schema.ResourceKindItem || resource.Item == nil || resource.Key == "" {
			return GetStorageResult{}, fmt.Errorf("storage record %d: game ID 0x%08X is not a known item",
				index, gameIDs[index])
		}
		result.Records = append(result.Records, StorageRecord{
			OwnedItemID:      record.OwnedItemID,
			Kind:             resource.Kind,
			Key:              resource.Key,
			GameID:           gameIDs[index],
			ContainerSection: record.ContainerSection,
			PhysicalIndex:    record.PhysicalIndex,
			GaItemHandle:     record.GaItemHandle,
			Quantity:         record.Quantity,
			AcquisitionIndex: record.AcquisitionIndex,
		})
	}
	return result, nil
}
