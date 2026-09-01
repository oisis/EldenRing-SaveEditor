package equipment

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Stage 3b.3b embeds the shared MutationReceipt in the public results of the
// seven Equipment mutations. The interesting invariants are that every result
// carries the receipt the central commit path produced, that each of the seven
// keeps its own operationKind, that a commit which writes no byte is still a
// commit with a full receipt and a new revision, and that a rejected mutation
// returns nothing at all.
//
// The expected scope list is stated literally on purpose: reading it back from
// the SaveEngine table would compare the implementation with itself. All seven
// Equipment mutations write only loadout fields, so all seven share it.
var equipmentLoadoutScopes = []string{
	"save.session", "equipment.loadout", "diagnostics.report"}

// The domain members each result keeps beside the flat receipt.
var (
	armamentDomainKeys = []string{"characterID", "slotAssignments"}
	armorDomainKeys    = []string{"characterID", "slotAssignments"}
	talismanDomainKeys = []string{"characterID", "orderedResources", "unlockedSlots"}
	spellDomainKeys    = []string{
		"characterID", "orderedResources", "usedMemorySlots", "availableMemorySlots"}
	physickDomainKeys = []string{"characterID", "crystalTearResources"}
	pouchDomainKeys   = []string{"characterID", "slotAssignments"}
	quickDomainKeys   = []string{"characterID", "slotAssignments"}
)

// receiptCatalog is the full stored GameCatalog, built once for this file.
//
// ponytail: one sync.Once, not a change to the existing per-test constructors.
// newPouchCatalog rebuilds the whole catalog on every call, and this file calls
// for it about twenty times; under -race that alone pushed the package past the
// ten-minute test timeout. Nothing here patches a document, so one shared
// read-only catalog is enough.
func receiptCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	receiptCatalogOnce.Do(func() {
		data, err := loader.LoadFS(catalogdata.Files())
		if err != nil {
			receiptCatalogErr = err
			return
		}
		receiptCatalogValue, receiptCatalogErr = gamecatalog.New(data.Manifest, data.Resources())
	})
	if receiptCatalogErr != nil {
		t.Fatalf("build the stored catalog: %v", receiptCatalogErr)
	}
	return receiptCatalogValue
}

var (
	receiptCatalogOnce  sync.Once
	receiptCatalogValue *gamecatalog.Catalog
	receiptCatalogErr   error
)

// equipmentReceiptTarget is one prepared Equipment session: the engine, the
// session identifier and a closure that performs the endpoint call under one
// expected revision. Each of the seven endpoints has a different signature, so
// the closure is the smallest shape that lets one table drive all of them.
//
// ponytail: a func value, not a mutation-plan interface. The seven calls have
// nothing in common but "take a revision, return something JSON-encodable".
type equipmentReceiptTarget struct {
	engine    *saveengine.Engine
	sessionID string
	apply     func(expectedRevision string) (any, saveengine.MutationReceipt, error)
}

func armamentsReceiptTarget(t *testing.T) equipmentReceiptTarget {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeSetEquippedArmamentsEndpointFixture(t, setArmamentsEndpointWeaponID), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := receiptCatalog(t)
	return equipmentReceiptTarget{
		engine:    engine,
		sessionID: loaded.SaveSessionID,
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			// OwnedItemIDs are pinned to one revision, so a repeated call must
			// resolve fresh tokens instead of replaying the stale ones.
			inventory, err := engine.GetInventory(loaded.SaveSessionID,
				setArmamentsEndpointSlot, saveengine.InventorySectionCommon, 1, 50)
			if err != nil || len(inventory.Records) != 7 {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
			}
			assignments := make([]*string, 6)
			for slot := range assignments {
				token := inventory.Records[slot+1].OwnedItemID
				assignments[slot] = &token
			}
			result, err := SetEquippedArmaments(engine, catalog, loaded.SaveSessionID,
				setArmamentsEndpointSlot, assignments, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func armorReceiptTarget(t *testing.T) equipmentReceiptTarget {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSetEquippedArmorEndpointFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := receiptCatalog(t)
	return equipmentReceiptTarget{
		engine:    engine,
		sessionID: loaded.SaveSessionID,
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			inventory, err := engine.GetInventory(loaded.SaveSessionID,
				setArmorEndpointSlot, saveengine.InventorySectionCommon, 1, 50)
			if err != nil || len(inventory.Records) != 8 {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
			}
			assignments := make([]*string, 4)
			for slot := range assignments {
				token := inventory.Records[slot+4].OwnedItemID
				assignments[slot] = &token
			}
			result, err := SetEquippedArmor(engine, catalog, loaded.SaveSessionID,
				setArmorEndpointSlot, assignments, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func talismansReceiptTarget(t *testing.T) equipmentReceiptTarget {
	t.Helper()

	path, platform := writeSetEquippedTalismansEndpointFixture(t)
	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, platform, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := receiptCatalog(t)
	return equipmentReceiptTarget{
		engine:    engine,
		sessionID: loaded.SaveSessionID,
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			inventory, err := engine.GetInventory(
				loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
			if err != nil || len(inventory.Records) < 3 {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
			}
			ordered := []string{inventory.Records[2].OwnedItemID}
			result, err := SetEquippedTalismans(
				engine, catalog, loaded.SaveSessionID, 0, ordered, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func spellsReceiptTarget(t *testing.T) equipmentReceiptTarget {
	t.Helper()

	engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})
	catalog := receiptCatalog(t)
	ordered := []*schema.ResourceRef{
		{Kind: schema.ResourceKindItem, Key: "40000FA0"},
		{Kind: schema.ResourceKindItem, Key: "40001068"},
	}
	return equipmentReceiptTarget{
		engine:    engine,
		sessionID: sessionID,
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetEquippedSpells(
				engine, catalog, sessionID, getEquippedSpellsSlot, ordered, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func physickReceiptTarget(t *testing.T) equipmentReceiptTarget {
	t.Helper()

	engine, sessionID := loadSetPhysickMixtureSession(t)
	catalog := receiptCatalog(t)
	tears := []*schema.ResourceRef{{Kind: schema.ResourceKindItem, Key: "40002AF9"}, nil}
	return equipmentReceiptTarget{
		engine:    engine,
		sessionID: sessionID,
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			result, err := SetPhysickMixture(
				engine, catalog, sessionID, getPhysickMixtureSlot, tears, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func pouchReceiptTarget(t *testing.T) equipmentReceiptTarget {
	t.Helper()

	path, platform := writeSetPouchEndpointFixture(t)
	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, platform, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := receiptCatalog(t)
	return equipmentReceiptTarget{
		engine:    engine,
		sessionID: loaded.SaveSessionID,
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			inventory, err := engine.GetInventory(
				loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
			if err != nil || len(inventory.Records) == 0 {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
			}
			token := inventory.Records[0].OwnedItemID
			result, err := SetPouchItems(engine, catalog, loaded.SaveSessionID, 0,
				[]*string{&token, nil, nil, nil, nil, nil}, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

func quickItemsReceiptTarget(t *testing.T) equipmentReceiptTarget {
	t.Helper()

	path, platform := writeSetQuickItemsEndpointFixture(t)
	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, platform, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	catalog := receiptCatalog(t)
	return equipmentReceiptTarget{
		engine:    engine,
		sessionID: loaded.SaveSessionID,
		apply: func(expectedRevision string) (any, saveengine.MutationReceipt, error) {
			inventory, err := engine.GetInventory(
				loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
			if err != nil || len(inventory.Records) == 0 {
				t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
			}
			token := inventory.Records[0].OwnedItemID
			assignments := make([]*string, 10)
			assignments[0] = &token
			result, err := SetQuickItems(
				engine, catalog, loaded.SaveSessionID, 0, assignments, expectedRevision)
			return result, result.MutationReceipt, err
		},
	}
}

// equipmentReceiptCases drives the receipt, scope, flat-JSON and committed
// no-change invariants for all seven endpoints from one table, so a new
// Equipment mutation cannot be added with a narrower guarantee.
var equipmentReceiptCases = []struct {
	name          string
	operationKind string
	domainKeys    []string
	target        func(*testing.T) equipmentReceiptTarget
}{
	{"SetEquippedArmaments", SetEquippedArmamentsEndpointID, armamentDomainKeys,
		armamentsReceiptTarget},
	{"SetEquippedArmor", SetEquippedArmorEndpointID, armorDomainKeys, armorReceiptTarget},
	{"SetEquippedTalismans", SetEquippedTalismansEndpointID, talismanDomainKeys,
		talismansReceiptTarget},
	{"SetEquippedSpells", SetEquippedSpellsEndpointID, spellDomainKeys, spellsReceiptTarget},
	{"SetPhysickMixture", SetPhysickMixtureEndpointID, physickDomainKeys, physickReceiptTarget},
	{"SetPouchItems", SetPouchItemsEndpointID, pouchDomainKeys, pouchReceiptTarget},
	{"SetQuickItems", SetQuickItemsEndpointID, quickDomainKeys, quickItemsReceiptTarget},
}

func TestEveryEquipmentMutationCarriesItsCommitReceipt(t *testing.T) {
	for _, testCase := range equipmentReceiptCases {
		t.Run(testCase.name, func(t *testing.T) {
			target := testCase.target(t)
			result, receipt, err := target.apply("0")
			if err != nil {
				t.Fatalf("%s: %v", testCase.name, err)
			}
			assertMutationReceipt(t, receipt, target.sessionID, testCase.operationKind, "1")
			assertChangedScopes(t, receipt.ChangedScopes, equipmentLoadoutScopes)
			assertFlatReceiptJSON(t, result, testCase.domainKeys)
		})
	}
}

// Every Equipment setter may finish its callback without writing a byte, and the
// central commit path still advances the revision, marks the session dirty and
// issues a receipt. That committed no-change is a commit, so the second identical
// assignment must report a fresh operationID, the same kind, the next revision,
// the same scopes and unchanged domain fields.
func TestEveryEquipmentMutationCommitsAnIdenticalReassignment(t *testing.T) {
	for _, testCase := range equipmentReceiptCases {
		t.Run(testCase.name, func(t *testing.T) {
			target := testCase.target(t)
			first, firstReceipt, err := target.apply("0")
			if err != nil {
				t.Fatalf("first %s: %v", testCase.name, err)
			}

			second, secondReceipt, err := target.apply("1")
			if err != nil {
				t.Fatalf("identical second %s: %v", testCase.name, err)
			}
			assertMutationReceipt(t, secondReceipt, target.sessionID, testCase.operationKind, "2")
			assertChangedScopes(t, secondReceipt.ChangedScopes, equipmentLoadoutScopes)
			assertFlatReceiptJSON(t, second, testCase.domainKeys)
			if secondReceipt.OperationID == firstReceipt.OperationID {
				t.Errorf("two executions shared operationID %q", secondReceipt.OperationID)
			}
			if !reflect.DeepEqual(
				domainMembers(t, second, testCase.domainKeys),
				domainMembers(t, first, testCase.domainKeys)) {
				t.Errorf("committed no-change changed the domain fields: %+v, want %+v",
					second, first)
			}

			// The revision really moved and the session really is dirty.
			info, err := target.engine.GetSessionInfo(target.sessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if info.SaveRevision != "2" || !info.UnsavedChanges {
				t.Errorf("session = %+v, want revision 2 with unsaved changes", info)
			}
		})
	}
}

// A rejected Equipment mutation returns the complete zero result: no receipt
// member and no domain member survives. The cases below cover the general
// invariants, not one literal item, slot or index.
func TestRejectedEquipmentMutationsReturnNoReceiptAndNoDomainData(t *testing.T) {
	t.Run("SetEquippedArmaments/stale expectedRevision", func(t *testing.T) {
		target := armamentsReceiptTarget(t)
		result, _, err := target.apply("7")
		assertZeroResult(t, err, result.(SetEquippedArmamentsResult), SetEquippedArmamentsResult{})
	})
	t.Run("SetEquippedArmaments/unknown ownedItemID", func(t *testing.T) {
		engine := saveengine.New()
		loaded, err := engine.LoadSave(
			writeSetEquippedArmamentsEndpointFixture(t, setArmamentsEndpointWeaponID), "pc", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		token := "oi-not-a-record"
		assignments := make([]*string, 6)
		assignments[0] = &token
		result, err := SetEquippedArmaments(engine, receiptCatalog(t), loaded.SaveSessionID,
			setArmamentsEndpointSlot, assignments, "0")
		assertZeroResult(t, err, result, SetEquippedArmamentsResult{})
	})
	t.Run("SetEquippedArmor/non-canonical expectedRevision", func(t *testing.T) {
		target := armorReceiptTarget(t)
		result, _, err := target.apply("00")
		assertZeroResult(t, err, result.(SetEquippedArmorResult), SetEquippedArmorResult{})
	})
	t.Run("SetEquippedArmor/wrong position count", func(t *testing.T) {
		result, err := SetEquippedArmor(saveengine.New(), receiptCatalog(t),
			"unused", 0, make([]*string, 3), "0")
		assertZeroResult(t, err, result, SetEquippedArmorResult{})
	})
	t.Run("SetEquippedTalismans/more talismans than unlockedSlots", func(t *testing.T) {
		path, platform := writeSetEquippedTalismansEndpointFixture(t)
		engine := saveengine.New()
		loaded, err := engine.LoadSave(path, platform, "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		inventory, err := engine.GetInventory(
			loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
		if err != nil || len(inventory.Records) < 3 {
			t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
		}
		// The fixture unlocks exactly one talisman field.
		ordered := []string{inventory.Records[2].OwnedItemID, inventory.Records[2].OwnedItemID}
		result, err := SetEquippedTalismans(
			engine, receiptCatalog(t), loaded.SaveSessionID, 0, ordered, "0")
		if err == nil || !strings.Contains(err.Error(), "unlocked talisman slot") {
			t.Fatalf("error = %v, want the unlocked-slot rejection", err)
		}
		assertZeroResult(t, err, result, SetEquippedTalismansResult{})
	})
	t.Run("SetEquippedTalismans/more positions than the endpoint accepts", func(t *testing.T) {
		result, err := SetEquippedTalismans(saveengine.New(), receiptCatalog(t),
			"unused", 0, []string{"a", "b", "c", "d", "e"}, "0")
		assertZeroResult(t, err, result, SetEquippedTalismansResult{})
	})
	t.Run("SetEquippedSpells/memory slots exceeded", func(t *testing.T) {
		engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})
		// 1 + 3 + 2 memory slots against the five the fixture makes available.
		ordered := []*schema.ResourceRef{
			{Kind: schema.ResourceKindItem, Key: "40000FA0"},
			{Kind: schema.ResourceKindItem, Key: "40001068"},
			{Kind: schema.ResourceKindItem, Key: "40001108"},
		}
		result, err := SetEquippedSpells(engine, receiptCatalog(t),
			sessionID, getEquippedSpellsSlot, ordered, "0")
		assertZeroResult(t, err, result, SetEquippedSpellsResult{})
	})
	t.Run("SetEquippedSpells/unknown resource", func(t *testing.T) {
		engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})
		ordered := []*schema.ResourceRef{{Kind: schema.ResourceKindItem, Key: "40000FFF"}}
		result, err := SetEquippedSpells(engine, receiptCatalog(t),
			sessionID, getEquippedSpellsSlot, ordered, "0")
		assertZeroResult(t, err, result, SetEquippedSpellsResult{})
	})
	t.Run("SetPhysickMixture/stale expectedRevision", func(t *testing.T) {
		target := physickReceiptTarget(t)
		result, _, err := target.apply("7")
		assertZeroResult(t, err, result.(SetPhysickMixtureResult), SetPhysickMixtureResult{})
	})
	t.Run("SetPhysickMixture/wrong position count", func(t *testing.T) {
		engine, sessionID := loadSetPhysickMixtureSession(t)
		result, err := SetPhysickMixture(engine, receiptCatalog(t), sessionID,
			getPhysickMixtureSlot,
			[]*schema.ResourceRef{{Kind: schema.ResourceKindItem, Key: "40002AF9"}}, "0")
		assertZeroResult(t, err, result, SetPhysickMixtureResult{})
	})
	t.Run("SetPouchItems/goods without the pouch capability", func(t *testing.T) {
		path, platform := writeSetPouchEndpointFixture(t)
		engine := saveengine.New()
		loaded, err := engine.LoadSave(path, platform, "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		inventory, err := engine.GetInventory(
			loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
		if err != nil || len(inventory.Records) < 2 {
			t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
		}
		token := inventory.Records[1].OwnedItemID
		result, err := SetPouchItems(engine, receiptCatalog(t), loaded.SaveSessionID, 0,
			[]*string{&token, nil, nil, nil, nil, nil}, "0")
		assertZeroResult(t, err, result, SetPouchItemsResult{})
	})
	t.Run("SetQuickItems/wrong item family", func(t *testing.T) {
		path, platform := writeSetQuickItemsEndpointFixture(t)
		engine := saveengine.New()
		loaded, err := engine.LoadSave(path, platform, "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		inventory, err := engine.GetInventory(
			loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
		if err != nil || len(inventory.Records) < 3 {
			t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
		}
		token := inventory.Records[2].OwnedItemID
		assignments := make([]*string, 10)
		assignments[0] = &token
		result, err := SetQuickItems(
			engine, receiptCatalog(t), loaded.SaveSessionID, 0, assignments, "0")
		assertZeroResult(t, err, result, SetQuickItemsResult{})
	})
	t.Run("SetQuickItems/wrong position count", func(t *testing.T) {
		result, err := SetQuickItems(saveengine.New(), receiptCatalog(t),
			"unused", 0, make([]*string, 9), "0")
		assertZeroResult(t, err, result, SetQuickItemsResult{})
	})
}

// Every Equipment endpoint reports its own EndpointID as operationKind, and each
// of the seven is a registered SaveEngine mutation kind.
func TestEquipmentEndpointIDsAreTheirRegisteredMutationKinds(t *testing.T) {
	want := map[string]string{
		"set_equipped_armaments": SetEquippedArmamentsEndpointID,
		"set_equipped_armor":     SetEquippedArmorEndpointID,
		"set_equipped_talismans": SetEquippedTalismansEndpointID,
		"set_equipped_spells":    SetEquippedSpellsEndpointID,
		"set_physick_mixture":    SetPhysickMixtureEndpointID,
		"set_pouch_items":        SetPouchItemsEndpointID,
		"set_quick_items":        SetQuickItemsEndpointID,
	}
	registered := make(map[string]bool)
	for _, kind := range saveengine.MutationKinds() {
		registered[kind] = true
	}
	for kind, endpointID := range want {
		if endpointID != kind {
			t.Errorf("EndpointID = %q, want %q", endpointID, kind)
		}
		if !registered[kind] {
			t.Errorf("SaveEngine registers no operation kind %q", kind)
		}
		scopes, err := saveengine.ChangedScopesForMutationKind(kind)
		if err != nil {
			t.Fatalf("ChangedScopesForMutationKind(%q): %v", kind, err)
		}
		assertChangedScopes(t, scopes, equipmentLoadoutScopes)
	}
}

// domainMembers returns the endpoint's own JSON members, receipt excluded, so a
// committed no-change can be compared without the operationID that must differ.
func domainMembers(t *testing.T, result any, domainKeys []string) map[string]string {
	t.Helper()

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal %T: %v", result, err)
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &members); err != nil {
		t.Fatalf("decode %s: %v", encoded, err)
	}
	domain := make(map[string]string, len(domainKeys))
	for _, key := range domainKeys {
		raw, present := members[key]
		if !present {
			t.Fatalf("%T JSON carries no domain member %q: %s", result, key, encoded)
		}
		domain[key] = string(raw)
	}
	return domain
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
