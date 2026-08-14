/*
Endpoint: AddItemToStorage
EndpointID: add_item_to_storage
Purpose: Adds the specified resource or variant to the common section of Storage after validating the family, routing, depositability, limits and complete mutation plan.
How it works: The handler resolves the resource through GameCatalog, proves the common-only Storage add contract and delegates one atomic operation to SaveEngine.
Supported resource types: Depositable ItemDocument of family goods or talisman, outside category key_items and the unresolved Flasks subcategory.
Input variables: saveSessionID, characterID, kind, key, variantID, quantity, expectedRevision.
Variant selection: the optional variantID selects the base item document when it is absent and exactly one stored variant of the same (kind, key) pair when it is present; gamecatalog.Catalog.ResourceByKindKeyAndVariant is the single implementation of that rule.
GameCatalog variables read: item.family, item.gameID, item.category, item.subcategory, item.storage.recordMode, item.storage.maxStorage, item.capabilities.stack, and item.goods.isDepositable for goods.
Save variables processed: for a top-up the quantity bytes of the first matching common Storage record; for a new record the first free common row, common count, Storage acquisition allocators and one active GaItemData entry when needed. SaveEngine validates the complete plan and finishes with full success or rollback.
Implementation status: implemented
*/
package inventory

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// AddItemToStorageEndpointID is the stable backend identifier of AddItemToStorage.
const AddItemToStorageEndpointID = "add_item_to_storage"

// AddItemToStorageDefinition describes the public mutation contract.
var AddItemToStorageDefinition = contract.MustDefine(contract.Definition{
	Name:                       "AddItemToStorage",
	ID:                         AddItemToStorageEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "Depositable ItemDocument of family goods or talisman supported by common-only addition",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "kind", "key", "variantID", "quantity", "expectedRevision"},
	Description:                "Adds the specified resource or variant to common Storage after validating routing, depositability, limits and the complete mutation plan.",
})

// AddItemToStorageResult is the SaveEngine receipt without a parallel endpoint
// copy of the same fields.
type AddItemToStorageResult = saveengine.AddItemToStorageResult

// AddItemToStorage adds quantity as a delta to common Storage. Existing
// quantity stacks are topped up on their first physical record; separate
// instances always create one new record and therefore accept quantity 1 only.
func AddItemToStorage(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	kind string,
	key string,
	variantID *uint32,
	quantity uint32,
	expectedRevision string,
) (AddItemToStorageResult, error) {
	if engine == nil {
		return AddItemToStorageResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return AddItemToStorageResult{}, errors.New("game catalog is not available")
	}

	resolved, err := resolveCommonItemAddition(gameCatalog, kind, key, variantID)
	if err != nil {
		return AddItemToStorageResult{}, err
	}
	if err := validateCommonItemAdditionSubcategory(resolved, kind, key); err != nil {
		return AddItemToStorageResult{}, err
	}
	maxStorage, err := storageCommonItemAdditionLimit(resolved, kind, key)
	if err != nil {
		return AddItemToStorageResult{}, err
	}

	return engine.AddItemToStorage(
		saveSessionID,
		characterID,
		resolved.gameID,
		quantity,
		expectedRevision,
		resolved.separateInstances,
		maxStorage,
	)
}
