package inventory

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Stage 3b.3a embeds the shared MutationReceipt in the public results of the
// twelve Inventory mutations. The interesting invariants are that every result
// carries the receipt the central commit path produced, that the three weapon
// endpoints sharing one SaveEngine writer keep their own operationKind, and
// that a rejected mutation returns nothing at all.
//
// The expected scope lists below are stated literally on purpose: reading them
// back from the SaveEngine table would compare the implementation with itself.
var (
	inventoryScopes     = []string{"save.session", "inventory", "diagnostics.report"}
	storageScopes       = []string{"save.session", "storage", "diagnostics.report"}
	bothContainerScopes = []string{
		"save.session", "inventory", "storage", "diagnostics.report"}
	containerAndLoadoutScopes = []string{
		"save.session", "inventory", "storage", "equipment.loadout", "diagnostics.report"}
)

func TestAddItemToInventoryCarriesItsCommitReceipt(t *testing.T) {
	catalog := quantityTestCatalog(t, addItemTestDocument("tools", 600, 40))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	result, err := AddItemToInventory(
		engine, catalog, sessionID, getInventorySlot, addItemTestKind, addItemTestKey, nil, 5, "0")
	if err != nil {
		t.Fatalf("AddItemToInventory: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID, AddItemToInventoryEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, inventoryScopes)
	assertFlatReceiptJSON(t, result, []string{
		"characterID", "gameID", "added", "quantity", "createdRecord",
		"containerSection", "physicalIndex"})
	if result.GameID != addItemTestEndpointGameID || result.Added != 5 || result.Quantity != 8 {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

func TestAddItemToStorageCarriesItsCommitReceipt(t *testing.T) {
	catalog := quantityTestCatalog(t, addStorageTestDocument(true, 600))
	engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)

	result, err := AddItemToStorage(
		engine, catalog, sessionID, getInventorySlot, addItemTestKind, addItemTestKey, nil, 5, "0")
	if err != nil {
		t.Fatalf("AddItemToStorage: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID, AddItemToStorageEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, storageScopes)
	assertFlatReceiptJSON(t, result, []string{
		"characterID", "gameID", "added", "quantity", "createdRecord",
		"containerSection", "physicalIndex"})
	if result.GameID != addItemTestEndpointGameID || result.Added != 5 || !result.CreatedRecord {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

// The two move endpoints cross the container boundary in both directions, so
// each of them invalidates Inventory and Storage together.
func TestMoveOwnedItemToInventoryCarriesItsCommitReceipt(t *testing.T) {
	engine, catalog, sessionID, ownedItemID := moveInventoryEndpointTarget(t, 6)

	result, err := MoveOwnedItemToInventory(
		engine, catalog, sessionID, getStorageSlot, ownedItemID, 0, "0")
	if err != nil {
		t.Fatalf("MoveOwnedItemToInventory: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID,
		MoveOwnedItemToInventoryEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, bothContainerScopes)
	assertFlatReceiptJSON(t, result, []string{
		"ownedItemID", "characterID", "gameID", "quantity", "containerSection",
		"targetPosition", "physicalIndex", "acquisitionIndex"})
	if result.OwnedItemID != ownedItemID || result.Quantity != 1 {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

func TestMoveOwnedItemToStorageCarriesItsCommitReceipt(t *testing.T) {
	engine, catalog, sessionID, ownedItemID := moveStorageEndpointTarget(t, 6)

	result, err := MoveOwnedItemToStorage(
		engine, catalog, sessionID, getInventorySlot, ownedItemID, 0, "0")
	if err != nil {
		t.Fatalf("MoveOwnedItemToStorage: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID,
		MoveOwnedItemToStorageEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, bothContainerScopes)
	assertFlatReceiptJSON(t, result, []string{
		"ownedItemID", "characterID", "gameID", "quantity", "containerSection",
		"targetPosition", "physicalIndex", "acquisitionIndex"})
	if result.OwnedItemID != ownedItemID || result.Quantity != 1 {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

// RemoveOwnedItem and SetOwnedItemQuantity address either container through one
// opaque OwnedItemID, so the receipt of a record that happens to sit in Storage
// must report exactly the same scopes as the Inventory case.
func TestRemoveOwnedItemCarriesItsCommitReceiptInBothContainers(t *testing.T) {
	for _, container := range []string{"inventory", "storage"} {
		t.Run(container, func(t *testing.T) {
			catalog := inventoryCatalog(t)
			engine, sessionID, ownedItemID := removeTestTarget(t, container, catalog)

			result, err := RemoveOwnedItem(
				engine, sessionID, removeTestSlot(container), ownedItemID, "0")
			if err != nil {
				t.Fatalf("RemoveOwnedItem: %v", err)
			}
			assertMutationReceipt(t, result.MutationReceipt, sessionID,
				RemoveOwnedItemEndpointID, "1")
			assertChangedScopes(t, result.ChangedScopes, bothContainerScopes)
			assertFlatReceiptJSON(t, result, []string{"ownedItemID", "characterID", "gameID"})
			if result.OwnedItemID != ownedItemID {
				t.Errorf("ownedItemID = %q, want %q", result.OwnedItemID, ownedItemID)
			}
		})
	}
}

func TestSetOwnedItemQuantityCarriesItsCommitReceiptInBothContainers(t *testing.T) {
	for _, container := range []string{"inventory", "storage"} {
		t.Run(container, func(t *testing.T) {
			catalog := quantityTestCatalog(t, quantityTestItem(99, 600, 600))
			engine, sessionID, ownedItemID := quantityTestTarget(t, container, catalog)

			result, err := SetOwnedItemQuantity(
				engine, catalog, sessionID, quantityTestSlot(container), ownedItemID, 42, "0")
			if err != nil {
				t.Fatalf("SetOwnedItemQuantity: %v", err)
			}
			assertMutationReceipt(t, result.MutationReceipt, sessionID,
				SetOwnedItemQuantityEndpointID, "1")
			// A quantity change can address a record a Quick Item or Pouch slot
			// reports, so the loadout is invalidated as well.
			assertChangedScopes(t, result.ChangedScopes, containerAndLoadoutScopes)
			assertFlatReceiptJSON(t, result,
				[]string{"ownedItemID", "characterID", "quantity"})
			if result.Quantity != 42 {
				t.Errorf("quantity = %d, want the committed 42", result.Quantity)
			}
		})
	}
}

func TestSetInventoryOrderCarriesItsCommitReceipt(t *testing.T) {
	engine, sessionID, records := setInventoryOrderEndpointTarget(t)

	result, err := SetInventoryOrder(
		engine, inventoryCatalog(t), sessionID, getInventorySlot,
		[]string{records[1].OwnedItemID}, "0")
	if err != nil {
		t.Fatalf("SetInventoryOrder: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID, SetInventoryOrderEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, inventoryScopes)
	assertFlatReceiptJSON(t, result,
		[]string{"characterID", "orderedResources", "acquisitionIndices"})
	if len(result.OrderedResources) != 1 ||
		!reflect.DeepEqual(result.AcquisitionIndices, []uint32{434}) {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

func TestSetStorageOrderCarriesItsCommitReceipt(t *testing.T) {
	engine, sessionID, records := setStorageOrderEndpointTarget(t)

	result, err := SetStorageOrder(
		engine, inventoryCatalog(t), sessionID, getStorageSlot,
		[]string{records[1].OwnedItemID}, "0")
	if err != nil {
		t.Fatalf("SetStorageOrder: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID, SetStorageOrderEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, storageScopes)
	assertFlatReceiptJSON(t, result,
		[]string{"characterID", "orderedResources", "acquisitionIndices"})
	if len(result.OrderedResources) != 1 ||
		!reflect.DeepEqual(result.AcquisitionIndices, []uint32{14}) {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

func TestSetWeaponAshOfWarCarriesItsCommitReceipt(t *testing.T) {
	engine, sessionID, token := setWeaponAshOfWarEndpointTarget(t)
	kind, key := setWeaponAoWEndpointKind, setWeaponAoWEndpointKey

	result, err := SetWeaponAshOfWar(
		engine, inventoryCatalog(t), sessionID, 0, token, &kind, &key, "0")
	if err != nil {
		t.Fatalf("SetWeaponAshOfWar: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID, SetWeaponAshOfWarEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, containerAndLoadoutScopes)
	assertFlatReceiptJSON(t, result, []string{
		"weaponOwnedItemID", "characterID", "container", "weaponGameID",
		"previousAshOfWarGameID", "ashOfWarGameID"})
	if result.WeaponOwnedItemID != token ||
		result.AshOfWarGameID != setWeaponAoWEndpointGameID {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

func TestSetWeaponInfusionCarriesItsCommitReceipt(t *testing.T) {
	const currentGameID = setWeaponEndpointDaggerID + 5
	engine, sessionID, token := setWeaponUpgradeEndpointTarget(t, currentGameID)

	result, err := SetWeaponInfusion(
		engine, inventoryCatalog(t), sessionID, 0, token, schema.AffinityHeavy, "0")
	if err != nil {
		t.Fatalf("SetWeaponInfusion: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID, SetWeaponInfusionEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, containerAndLoadoutScopes)
	assertFlatReceiptJSON(t, result, []string{
		"ownedItemID", "characterID", "container", "previousGameID", "gameID",
		"affinity", "upgradeLevel"})
	if result.PreviousGameID != currentGameID || result.UpgradeLevel != 5 ||
		result.Affinity != schema.AffinityHeavy {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

func TestSetWeaponUpgradeLevelCarriesItsCommitReceipt(t *testing.T) {
	engine, sessionID, token := setWeaponUpgradeEndpointTarget(t, setWeaponEndpointDaggerID)

	result, err := SetWeaponUpgradeLevel(
		engine, inventoryCatalog(t), sessionID, 0, token, 25, "0")
	if err != nil {
		t.Fatalf("SetWeaponUpgradeLevel: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID,
		SetWeaponUpgradeLevelEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, containerAndLoadoutScopes)
	assertFlatReceiptJSON(t, result, []string{
		"ownedItemID", "characterID", "container", "previousGameID", "gameID", "upgradeLevel"})
	if result.PreviousGameID != setWeaponEndpointDaggerID ||
		result.GameID != setWeaponEndpointDaggerID+25 || result.UpgradeLevel != 25 {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

func TestSetSpiritAshUpgradeLevelCarriesItsCommitReceipt(t *testing.T) {
	engine, sessionID, token := setSpiritAshEndpointTarget(t)

	result, err := SetSpiritAshUpgradeLevel(
		engine, inventoryCatalog(t), sessionID, 0, token, 10, "0")
	if err != nil {
		t.Fatalf("SetSpiritAshUpgradeLevel: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, sessionID,
		SetSpiritAshUpgradeLevelEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, containerAndLoadoutScopes)
	assertFlatReceiptJSON(t, result, []string{
		"ownedItemID", "characterID", "container", "previousGameID", "gameID", "upgradeLevel"})
	if result.PreviousGameID != setSpiritAshEndpointCurrent ||
		result.GameID != setSpiritAshEndpointTargetID || result.UpgradeLevel != 10 {
		t.Errorf("result = %+v, want the unchanged domain contract", result)
	}
}

// The three public weapon mutations must report three different operationKinds.
// They do not share one SaveEngine writer: SetWeaponAshOfWar owns its own
// mutation path and its own commit, while SetWeaponInfusion and
// SetWeaponUpgradeLevel are the two callers of setOwnedWeaponGameID. Either way,
// no public kind may be merged into another.
func TestWeaponMutationEndpointsKeepDistinctOperationKinds(t *testing.T) {
	aowEngine, aowSession, aowToken := setWeaponAshOfWarEndpointTarget(t)
	kind, key := setWeaponAoWEndpointKind, setWeaponAoWEndpointKey
	ashOfWar, err := SetWeaponAshOfWar(
		aowEngine, inventoryCatalog(t), aowSession, 0, aowToken, &kind, &key, "0")
	if err != nil {
		t.Fatalf("SetWeaponAshOfWar: %v", err)
	}

	infusionEngine, infusionSession, infusionToken := setWeaponUpgradeEndpointTarget(
		t, setWeaponEndpointDaggerID+5)
	infusion, err := SetWeaponInfusion(
		infusionEngine, inventoryCatalog(t), infusionSession, 0, infusionToken,
		schema.AffinityHeavy, "0")
	if err != nil {
		t.Fatalf("SetWeaponInfusion: %v", err)
	}

	upgradeEngine, upgradeSession, upgradeToken := setWeaponUpgradeEndpointTarget(
		t, setWeaponEndpointDaggerID)
	upgrade, err := SetWeaponUpgradeLevel(
		upgradeEngine, inventoryCatalog(t), upgradeSession, 0, upgradeToken, 25, "0")
	if err != nil {
		t.Fatalf("SetWeaponUpgradeLevel: %v", err)
	}

	kinds := map[string]string{
		"SetWeaponAshOfWar":     ashOfWar.OperationKind,
		"SetWeaponInfusion":     infusion.OperationKind,
		"SetWeaponUpgradeLevel": upgrade.OperationKind,
	}
	want := map[string]string{
		"SetWeaponAshOfWar":     SetWeaponAshOfWarEndpointID,
		"SetWeaponInfusion":     SetWeaponInfusionEndpointID,
		"SetWeaponUpgradeLevel": SetWeaponUpgradeLevelEndpointID,
	}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("operationKinds = %v, want each endpoint's own kind %v", kinds, want)
	}
}

// Two executions of one Inventory mutation share the kind and never the
// identifier of the execution.
func TestTwoExecutionsOfOneInventoryMutationGetDifferentOperationIDs(t *testing.T) {
	catalog := quantityTestCatalog(t, quantityTestItem(99, 600, 600))
	engine, sessionID, ownedItemID := quantityTestTarget(t, "inventory", catalog)

	first, err := SetOwnedItemQuantity(
		engine, catalog, sessionID, getInventorySlot, ownedItemID, 42, "0")
	if err != nil {
		t.Fatalf("first SetOwnedItemQuantity: %v", err)
	}
	listed, err := GetInventory(engine, catalog, sessionID, getInventorySlot, "", 0, 0)
	if err != nil {
		t.Fatalf("GetInventory under the new revision: %v", err)
	}
	second, err := SetOwnedItemQuantity(engine, catalog, sessionID, getInventorySlot,
		quantityTestIdentity(t, getInventoryIdentities(t, listed.Records)), 7, "1")
	if err != nil {
		t.Fatalf("second SetOwnedItemQuantity: %v", err)
	}
	if first.OperationID == second.OperationID {
		t.Fatalf("two executions shared operationID %q", first.OperationID)
	}
	if first.OperationKind != second.OperationKind {
		t.Fatalf("operationKinds = %q and %q, want one stable kind",
			first.OperationKind, second.OperationKind)
	}
}

// A rejected Inventory mutation returns the exact zero result: no operationID
// of an execution that never happened and no partial domain field either.
func TestRejectedInventoryMutationsExposeNoReceipt(t *testing.T) {
	t.Run("AddItemToInventory/stale expectedRevision", func(t *testing.T) {
		catalog := quantityTestCatalog(t, addItemTestDocument("tools", 600, 40))
		engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
		result, err := AddItemToInventory(engine, catalog, sessionID, getInventorySlot,
			addItemTestKind, addItemTestKey, nil, 5, "7")
		assertZeroResult(t, err, result, AddItemToInventoryResult{})
	})
	t.Run("AddItemToInventory/quantity above the catalog stack limit", func(t *testing.T) {
		catalog := quantityTestCatalog(t, addItemTestDocument("tools", 600, 40))
		engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
		result, err := AddItemToInventory(engine, catalog, sessionID, getInventorySlot,
			addItemTestKind, addItemTestKey, nil, 41, "0")
		assertZeroResult(t, err, result, AddItemToInventoryResult{})
	})
	t.Run("AddItemToStorage/stale expectedRevision", func(t *testing.T) {
		catalog := quantityTestCatalog(t, addStorageTestDocument(true, 600))
		engine, sessionID := loadGetInventorySession(t, "pc", true, getInventoryAnchorAt)
		result, err := AddItemToStorage(engine, catalog, sessionID, getInventorySlot,
			addItemTestKind, addItemTestKey, nil, 5, "7")
		assertZeroResult(t, err, result, AddItemToStorageResult{})
	})
	t.Run("MoveOwnedItemToInventory/unknown ownedItemID", func(t *testing.T) {
		engine, catalog, sessionID, _ := moveInventoryEndpointTarget(t, 6)
		result, err := MoveOwnedItemToInventory(
			engine, catalog, sessionID, getStorageSlot, "oi-not-a-record", 0, "0")
		assertZeroResult(t, err, result, MoveOwnedItemToInventoryResult{})
	})
	t.Run("MoveOwnedItemToStorage/non-canonical expectedRevision", func(t *testing.T) {
		engine, catalog, sessionID, ownedItemID := moveStorageEndpointTarget(t, 6)
		result, err := MoveOwnedItemToStorage(
			engine, catalog, sessionID, getInventorySlot, ownedItemID, 0, "00")
		assertZeroResult(t, err, result, MoveOwnedItemToStorageResult{})
	})
	t.Run("RemoveOwnedItem/unknown ownedItemID", func(t *testing.T) {
		catalog := inventoryCatalog(t)
		engine, sessionID, _ := removeTestTarget(t, "inventory", catalog)
		result, err := RemoveOwnedItem(
			engine, sessionID, getInventorySlot, "oi-not-a-record", "0")
		assertZeroResult(t, err, result, RemoveOwnedItemResult{})
	})
	t.Run("SetOwnedItemQuantity/quantity above the catalog limit", func(t *testing.T) {
		catalog := quantityTestCatalog(t, quantityTestItem(99, 600, 600))
		engine, sessionID, ownedItemID := quantityTestTarget(t, "inventory", catalog)
		result, err := SetOwnedItemQuantity(
			engine, catalog, sessionID, getInventorySlot, ownedItemID, 100, "0")
		assertZeroResult(t, err, result, SetOwnedItemQuantityResult{})
	})
	t.Run("SetInventoryOrder/stale expectedRevision", func(t *testing.T) {
		engine, sessionID, records := setInventoryOrderEndpointTarget(t)
		result, err := SetInventoryOrder(engine, inventoryCatalog(t), sessionID,
			getInventorySlot, []string{records[1].OwnedItemID}, "7")
		assertZeroResult(t, err, result, SetInventoryOrderResult{})
	})
	t.Run("SetStorageOrder/stale expectedRevision", func(t *testing.T) {
		engine, sessionID, records := setStorageOrderEndpointTarget(t)
		result, err := SetStorageOrder(engine, inventoryCatalog(t), sessionID,
			getStorageSlot, []string{records[1].OwnedItemID}, "7")
		assertZeroResult(t, err, result, SetStorageOrderResult{})
	})
	t.Run("SetWeaponAshOfWar/stale expectedRevision", func(t *testing.T) {
		engine, sessionID, token := setWeaponAshOfWarEndpointTarget(t)
		kind, key := setWeaponAoWEndpointKind, setWeaponAoWEndpointKey
		result, err := SetWeaponAshOfWar(
			engine, inventoryCatalog(t), sessionID, 0, token, &kind, &key, "7")
		assertZeroResult(t, err, result, SetWeaponAshOfWarResult{})
	})
	t.Run("SetWeaponInfusion/non-canonical expectedRevision", func(t *testing.T) {
		engine, sessionID, token := setWeaponUpgradeEndpointTarget(
			t, setWeaponEndpointDaggerID+5)
		result, err := SetWeaponInfusion(
			engine, inventoryCatalog(t), sessionID, 0, token, schema.AffinityHeavy, "00")
		assertZeroResult(t, err, result, SetWeaponInfusionResult{})
	})
	t.Run("SetWeaponUpgradeLevel/level above the catalog maximum", func(t *testing.T) {
		engine, sessionID, token := setWeaponUpgradeEndpointTarget(t, setWeaponEndpointDaggerID)
		result, err := SetWeaponUpgradeLevel(
			engine, inventoryCatalog(t), sessionID, 0, token, 26, "0")
		assertZeroResult(t, err, result, SetWeaponUpgradeLevelResult{})
	})
	t.Run("SetSpiritAshUpgradeLevel/level above the catalog maximum", func(t *testing.T) {
		engine, sessionID, token := setSpiritAshEndpointTarget(t)
		result, err := SetSpiritAshUpgradeLevel(
			engine, inventoryCatalog(t), sessionID, 0, token, 11, "0")
		assertZeroResult(t, err, result, SetSpiritAshUpgradeLevelResult{})
	})
}

// assertZeroResult fails unless the rejected call returned an error and the
// complete zero result, receipt and domain fields alike.
func assertZeroResult[Result any](t *testing.T, err error, got Result, zero Result) {
	t.Helper()

	if err == nil {
		t.Fatalf("the rejected call was accepted: %+v", got)
	}
	if !reflect.DeepEqual(got, zero) {
		t.Errorf("result = %+v, want the complete zero result", got)
	}
}

// assertMutationReceipt checks the four scalar receipt fields of one committed
// mutation. The scopes are checked separately, because their exact value is a
// per-endpoint contract.
func assertMutationReceipt(
	t *testing.T,
	receipt saveengine.MutationReceipt,
	saveSessionID string,
	operationKind string,
	saveRevision string,
) {
	t.Helper()

	if receipt.OperationID == "" {
		t.Errorf("receipt = %+v, want a minted operationID", receipt)
	}
	if receipt.OperationKind != operationKind {
		t.Errorf("operationKind = %q, want the EndpointID %q", receipt.OperationKind, operationKind)
	}
	if receipt.SaveSessionID != saveSessionID {
		t.Errorf("saveSessionID = %q, want %q", receipt.SaveSessionID, saveSessionID)
	}
	if receipt.SaveRevision != saveRevision {
		t.Errorf("saveRevision = %q, want %q", receipt.SaveRevision, saveRevision)
	}
}

func assertChangedScopes(t *testing.T, got []string, want []string) {
	t.Helper()

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("changedScopes = %v, want exactly %v in canonical order", got, want)
	}
}

// assertFlatReceiptJSON proves the embedding is flat: the five receipt fields
// are top-level keys of the payload, each key appears exactly once, there is no
// nested "receipt" object, and the domain fields of the endpoint survive.
func assertFlatReceiptJSON(t *testing.T, result any, domainKeys []string) {
	t.Helper()

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal %T: %v", result, err)
	}
	keys := jsonTopLevelKeys(t, encoded)

	counts := make(map[string]int, len(keys))
	for _, key := range keys {
		counts[key]++
	}
	want := append([]string{
		"operationID", "operationKind", "saveSessionID", "saveRevision", "changedScopes",
	}, domainKeys...)
	for _, key := range want {
		if counts[key] != 1 {
			t.Errorf("%T JSON carries key %q %d times, want exactly once: %s",
				result, key, counts[key], encoded)
		}
	}
	if len(keys) != len(want) {
		t.Errorf("%T JSON keys = %v, want exactly %v", result, keys, want)
	}
	if counts["receipt"] != 0 {
		t.Errorf("%T JSON nests the receipt instead of flattening it: %s", result, encoded)
	}
}

// jsonTopLevelKeys returns the member names of a JSON object in document order,
// repeats included. A map would silently collapse a duplicated key, which is
// exactly the defect this check exists to catch.
func jsonTopLevelKeys(t *testing.T, encoded []byte) []string {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		t.Fatalf("payload is not a JSON object: %s (%v)", encoded, err)
	}
	var keys []string
	for decoder.More() {
		name, err := decoder.Token()
		if err != nil {
			t.Fatalf("read member name of %s: %v", encoded, err)
		}
		key, isString := name.(string)
		if !isString {
			t.Fatalf("member name %v of %s is not a string", name, encoded)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("read member %q of %s: %v", key, encoded, err)
		}
		keys = append(keys, key)
	}
	return keys
}
