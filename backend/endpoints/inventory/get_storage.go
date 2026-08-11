/*
Endpoint: GetStorage
EndpointID: get_storage
Purpose: Returns raw native Storage Box records from one character slot without resolving them through GameCatalog.
How it works: The runtime handler passes saveSessionID, characterID, containerSection and the paging values to SaveEngine, which reads one slot of the private snapshot of an already loaded session. The endpoint opens no file, reads no snapshot, parses no save data of its own and calls no other endpoint.
Supported resource types: —.
Input variables: saveSessionID, characterID, containerSection, page, pageSize.
GameCatalog variables read: none; this phase returns raw state and resolves no ItemDocument.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the physical Storage Box records of the requested section; the getter is non-mutating, keeps gaItemHandle and acquisitionIndex raw, masks only the documented high bit of quantity and applies paging.
Implementation status: implemented
*/
package inventory

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetStorageEndpointID is the stable backend identifier of GetStorage.
const GetStorageEndpointID = "get_storage"

// GetStorageDefinition describes the public getter contract. This is phase 1:
// the contract carries no GameCatalog resource type and no family variable,
// because a raw record proves neither an item identity nor a family.
var GetStorageDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetStorage",
	ID:                         GetStorageEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "containerSection", "page", "pageSize"},
	Description:                "Returns raw native Storage Box records from one character slot without resolving them through GameCatalog.",
})

// GetStorageResult is the typed result of GetStorage. The shape is owned by
// SaveEngine, so the endpoint neither reshapes nor duplicates it. GaItemHandle
// and AcquisitionIndex are returned exactly as stored, Quantity is returned with
// the high bit removed by the 0x7FFFFFFF mask, and no other value is normalised
// or resolved against GameCatalog.
type GetStorageResult = saveengine.CharacterStorage

// GetStorage returns one page of the raw Storage Box records stored in one
// character slot of an existing save session.
//
// This is the first phase of the Storage surface. The result carries no name, no
// kind, no key, no family, no variant, no stable owned-item identity, no
// capacity and no Inventory record: those need a verified GaItem parser and
// belong to a later phase.
//
// The endpoint is thin: it rejects a missing engine and delegates everything
// else. Validating saveSessionID, characterID, containerSection and the paging
// values, reading the snapshot and deciding what an active, inactive or residual
// slot exposes belong to SaveEngine. The session must already exist; this
// endpoint never creates one, so it calls neither LoadSave, nor GetInventory,
// nor any other endpoint, opens no file, reads no GameCatalog and returns no raw
// save byte.
func GetStorage(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	containerSection string,
	page int,
	pageSize int,
) (GetStorageResult, error) {
	if engine == nil {
		return GetStorageResult{}, errors.New("save engine is not available")
	}
	return engine.GetStorage(saveSessionID, characterID, containerSection, page, pageSize)
}
