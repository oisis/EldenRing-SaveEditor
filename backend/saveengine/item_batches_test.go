package saveengine

import (
	"strings"
	"testing"
)

// The anchored group move is the one genuinely new placement rule of the batch
// mutations, and it is a pure function of the current order, so it is stated
// here directly instead of through a snapshot fixture.

func TestPlanAnchoredInventoryOrderKeepsEachSideOfTheAnchor(t *testing.T) {
	t.Parallel()

	current := []string{"a", "b", "c", "d", "e"}
	group := map[string]int{"a": 0, "c": 1, "e": 2}

	// The anchor is "c": "a" was in front of it and stays in front, "e" was
	// behind it and stays behind, and the anchor lands exactly on position 3.
	planned, err := planAnchoredInventoryOrder(current, "c", group, 3)
	if err != nil {
		t.Fatalf("planAnchoredInventoryOrder: %v", err)
	}
	want := []string{"b", "d", "a", "c", "e"}
	if strings.Join(planned, ",") != strings.Join(want, ",") {
		t.Fatalf("planned = %v, want %v", planned, want)
	}
	if planned[3] != "c" {
		t.Errorf("anchor landed on %q at position 3, want %q", planned[3], "c")
	}
	// Every record appears exactly once: the plan is a permutation, never a
	// partial order that would drop or duplicate a record.
	if len(planned) != len(current) {
		t.Fatalf("planned has %d records, want %d", len(planned), len(current))
	}
	seen := make(map[string]bool, len(planned))
	for _, ownedItemID := range planned {
		if seen[ownedItemID] {
			t.Fatalf("planned repeats %q", ownedItemID)
		}
		seen[ownedItemID] = true
	}
}

func TestPlanAnchoredInventoryOrderMovesOneRecord(t *testing.T) {
	t.Parallel()

	current := []string{"a", "b", "c"}
	planned, err := planAnchoredInventoryOrder(current, "c", map[string]int{"c": 0}, 0)
	if err != nil {
		t.Fatalf("planAnchoredInventoryOrder: %v", err)
	}
	if strings.Join(planned, ",") != "c,a,b" {
		t.Fatalf("planned = %v, want [c a b]", planned)
	}
}

func TestPlanAnchoredInventoryOrderRejectsImpossiblePlacements(t *testing.T) {
	t.Parallel()

	current := []string{"a", "b", "c", "d"}

	// The group needs one record in front of the anchor, so position 0 cannot
	// hold it. The plan is refused rather than clamped.
	if _, err := planAnchoredInventoryOrder(
		current, "c", map[string]int{"a": 0, "c": 1}, 0); err == nil {
		t.Error("a target position the group cannot occupy was accepted")
	}
	// A position outside the supported order.
	if _, err := planAnchoredInventoryOrder(current, "a", map[string]int{"a": 0}, 4); err == nil {
		t.Error("a target position outside the order was accepted")
	}
	// An identity that is not part of the supported order at all.
	if _, err := planAnchoredInventoryOrder(
		current, "a", map[string]int{"a": 0, "z": 1}, 0); err == nil {
		t.Error("an identity outside the supported order was accepted")
	}
	// The anchor has to be a member of the order it anchors.
	if _, err := planAnchoredInventoryOrder(current, "z", map[string]int{"a": 0}, 0); err == nil {
		t.Error("an anchor outside the supported order was accepted")
	}
}

// Every batch entry point validates its request before the session is touched,
// so these rejections happen on an engine that holds no session at all: a
// rejected batch can never reach a snapshot.
func TestBatchRequestsAreRejectedBeforeTheSessionIsTouched(t *testing.T) {
	t.Parallel()

	engine := New()
	duplicate := []string{"oi-1", "oi-1"}

	if _, err := engine.AddItemsToContainers("session", 0, nil, nil, "0"); err == nil {
		t.Error("an empty batch add was accepted")
	}
	repeated := []ItemAddition{
		{GameID: 1, Quantity: 1, MaxPerRecord: 1, MaxContainerTotal: 1},
		{GameID: 1, Quantity: 1, MaxPerRecord: 1, MaxContainerTotal: 1},
	}
	if _, err := engine.AddItemsToContainers("session", 0, repeated, nil, "0"); err == nil {
		t.Error("a batch add repeating one item in one container was accepted")
	}
	if _, err := engine.MoveOwnedItemsToStorage("session", 0, nil, "0"); err == nil {
		t.Error("an empty batch move was accepted")
	}
	moves := []OwnedItemMove{
		{OwnedItemID: "oi-1", MaxQuantity: 1},
		{OwnedItemID: "oi-1", MaxQuantity: 1},
	}
	if _, err := engine.MoveOwnedItemsToInventory("session", 0, moves, "0"); err == nil {
		t.Error("a batch move repeating one ownedItemID was accepted")
	}
	if _, err := engine.RemoveOwnedItems("session", 0, nil, "0"); err == nil {
		t.Error("an empty batch removal was accepted")
	}
	if _, err := engine.RemoveOwnedItems("session", 0, duplicate, "0"); err == nil {
		t.Error("a batch removal repeating one ownedItemID was accepted")
	}
	classify := func(uint32) (bool, error) { return true, nil }
	if _, err := engine.ReorderInventoryItems(
		"session", 0, "oi-1", nil, 0, "0", classify); err == nil {
		t.Error("a reorder without a group was accepted")
	}
	if _, err := engine.ReorderInventoryItems(
		"session", 0, "oi-2", []string{"oi-1"}, 0, "0", classify); err == nil {
		t.Error("a reorder whose anchor is outside the group was accepted")
	}
	if _, err := engine.ReorderInventoryItems(
		"session", 0, "oi-1", duplicate, 0, "0", classify); err == nil {
		t.Error("a reorder repeating one ownedItemID was accepted")
	}
	// A non-canonical revision is refused by every entry point.
	if _, err := engine.RemoveOwnedItems("session", 0, []string{"oi-1"}, "01"); err == nil {
		t.Error("a non-canonical expectedRevision was accepted")
	}
}

// The changed scopes of a batch add are resolved from the request, so a call
// that writes one container never reports the other one.
func TestBatchAddScopesNameOnlyTheContainersItWrites(t *testing.T) {
	t.Parallel()

	inventoryOnly, err := changedScopesForMutationKind(kindAddItemsToContainers, ScopeInventory)
	if err != nil {
		t.Fatalf("changedScopesForMutationKind: %v", err)
	}
	if strings.Join(inventoryOnly, ",") != "save.session,inventory,diagnostics.report" {
		t.Errorf("inventory-only scopes = %v", inventoryOnly)
	}
	both, err := changedScopesForMutationKind(
		kindAddItemsToContainers, ScopeInventory, ScopeStorage)
	if err != nil {
		t.Fatalf("changedScopesForMutationKind: %v", err)
	}
	if strings.Join(both, ",") != "save.session,inventory,storage,diagnostics.report" {
		t.Errorf("both-container scopes = %v", both)
	}
}

// The remaining tests drive real batch mutations against the shared synthetic
// slot of add_item_to_inventory_test.go. They cover what is genuinely new here
// — the atomicity of the candidate image, the per-execution scopes and the
// per-execution risk — and deliberately not the writers, limits or binary
// layouts the single-record tests of that file already prove.

// loadBatchSession builds a slot with one occupied common record in each
// container and returns the engine and the session it was loaded into.
func loadBatchSession(t *testing.T) (*Engine, string, int) {
	t.Helper()

	const slot = 2
	content := addItemTestFixture{
		platform: PlatformPC, slot: slot,
		common:      []addItemTestRow{{index: 0, handle: addItemTestOtherHandle, rawQuantity: 4, acquisition: 11}},
		storage:     []addItemTestRow{{index: 0, handle: addItemTestOtherHandle, rawQuantity: 2, acquisition: 12}},
		commonCount: 1, storageCount: 1, nextEquipIndex: 433, nextAcquisition: 968,
		gaItemData: []uint32{addItemTestOtherID},
	}
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, content), string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID, slot
}

// batchAddition is one ordinary, fully specified addition of the goods item.
func batchAddition(gameID uint32, quantity uint32) ItemAddition {
	return ItemAddition{
		GameID: gameID, Quantity: quantity, MaxPerRecord: 40, MaxContainerTotal: 600,
	}
}

func batchHistory(t *testing.T, engine *Engine, saveSessionID string) OperationHistory {
	t.Helper()

	history, err := engine.GetOperationHistory(saveSessionID)
	if err != nil {
		t.Fatalf("GetOperationHistory: %v", err)
	}
	return history
}

// A batch that writes several records in both containers is one mutation: one
// revision, one receipt with one operationID, and one history entry.
func TestBatchAddCommitsOneRevisionOneReceiptAndOneHistoryEntry(t *testing.T) {
	engine, saveSessionID, slot := loadBatchSession(t)

	result, err := engine.AddItemsToContainers(
		saveSessionID, slot,
		[]ItemAddition{batchAddition(addItemTestGoodsID, 5), batchAddition(addItemTestTalismanID, 1)},
		[]ItemAddition{batchAddition(addItemTestGoodsID, 3)},
		"0")
	if err != nil {
		t.Fatalf("AddItemsToContainers: %v", err)
	}

	// Both containers were written, so the receipt of this one execution reports
	// both. The static row of the kind names neither: the scopes are a property
	// of the request, and the receipt is checked against exactly what it wrote.
	assertCommittedReceiptWithScopes(t, result.MutationReceipt, saveSessionID,
		kindAddItemsToContainers, "1",
		[]string{ScopeSaveSession, ScopeInventory, ScopeStorage, ScopeDiagnosticsReport})
	if len(result.Added) != 3 {
		t.Fatalf("Added reports %d entries, want 3: %+v", len(result.Added), result.Added)
	}

	history := batchHistory(t, engine, saveSessionID)
	if len(history.Operations) != 1 {
		t.Fatalf("history holds %d operations, want exactly 1", len(history.Operations))
	}
	entry := history.Operations[0]
	if entry.OperationID != result.OperationID || entry.OperationKind != kindAddItemsToContainers {
		t.Errorf("history entry = %q/%q, want the receipt's %q/%q",
			entry.OperationID, entry.OperationKind, result.OperationID, kindAddItemsToContainers)
	}
	if history.SaveRevision != "1" || history.UndoCount != 1 || history.RedoCount != 0 {
		t.Errorf("history = revision %q, undo %d, redo %d; want 1, 1, 0",
			history.SaveRevision, history.UndoCount, history.RedoCount)
	}
	// Both containers were written, so both are reported, and the area of the
	// entry follows those real scopes rather than the kind.
	if strings.Join(entry.ChangedScopes, ",") != "save.session,inventory,storage,diagnostics.report" {
		t.Errorf("changedScopes = %v", entry.ChangedScopes)
	}
	if entry.Area != "Inventory" {
		t.Errorf("area = %q, want Inventory", entry.Area)
	}
}

// A step that fails after earlier steps already succeeded leaves the session
// exactly as it was: the candidate image is discarded whole.
func TestBatchAddLeavesNothingBehindWhenALaterItemFails(t *testing.T) {
	// The common section is occupied except for its very last row, and the
	// declared count matches those records, so the slot holds room for exactly
	// one more record.
	occupied := make([]addItemTestRow, 0, addItemTestCommonRecords-1)
	for index := 0; index < addItemTestCommonRecords-1; index++ {
		occupied = append(occupied, addItemTestRow{
			index: index, handle: addItemTestFillHandleBase + uint32(index)})
	}
	const slot = 2
	engine := New()
	loaded, err := engine.LoadSave(writeAddItemFixture(t, addItemTestFixture{
		platform: PlatformPC, slot: slot,
		common:         occupied,
		commonCount:    addItemTestCommonRecords - 1,
		nextEquipIndex: 433, nextAcquisition: 968,
	}), string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	saveSessionID := loaded.SaveSessionID
	before := addItemTestSlotData(t, engine, saveSessionID, PlatformPC, slot)

	// The two additions name different resources, so the request passes
	// validation, and each one is acceptable on its own: the container holds
	// neither item and the one free row can carry either record. The first
	// addition takes that row on the candidate image, and the second is refused
	// exactly because the candidate no longer has a free record. Nothing but a
	// whole-batch rollback can leave the session unchanged here.
	_, err = engine.AddItemsToContainers(
		saveSessionID, slot,
		[]ItemAddition{
			batchAddition(addItemTestGoodsID, 5),
			batchAddition(addItemTestTalismanID, 1),
		},
		nil, "0")
	if err == nil {
		t.Fatal("a batch whose second item finds no free record was accepted")
	}

	after := addItemTestSlotData(t, engine, saveSessionID, PlatformPC, slot)
	addItemTestAssertChanged(t, before, after, nil)
	if revision, dirty := addItemTestSessionState(t, engine, saveSessionID); revision != "0" || dirty {
		t.Errorf("the rejected batch left revision %q, dirty %v; want 0, false", revision, dirty)
	}
	if history := batchHistory(t, engine, saveSessionID); len(history.Operations) != 0 {
		t.Errorf("the rejected batch recorded %d history entries, want 0", len(history.Operations))
	}
}

// The dynamic scopes of a batch add stay attributed to that one execution, so
// undoing it reports exactly the containers it wrote and never the other one.
func TestBatchAddUndoReportsTheContainersTheExecutionWrote(t *testing.T) {
	cases := []struct {
		name      string
		inventory []ItemAddition
		storage   []ItemAddition
		want      string
	}{
		{"inventory only", []ItemAddition{batchAddition(addItemTestGoodsID, 5)}, nil,
			"save.session,inventory,diagnostics.report"},
		{"storage only", nil, []ItemAddition{batchAddition(addItemTestGoodsID, 5)},
			"save.session,storage,diagnostics.report"},
		{"both containers",
			[]ItemAddition{batchAddition(addItemTestGoodsID, 5)},
			[]ItemAddition{batchAddition(addItemTestTalismanID, 1)},
			"save.session,inventory,storage,diagnostics.report"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			engine, saveSessionID, slot := loadBatchSession(t)

			added, err := engine.AddItemsToContainers(
				saveSessionID, slot, testCase.inventory, testCase.storage, "0")
			if err != nil {
				t.Fatalf("AddItemsToContainers: %v", err)
			}
			if strings.Join(added.ChangedScopes, ",") != testCase.want {
				t.Errorf("add changedScopes = %v, want %s", added.ChangedScopes, testCase.want)
			}

			state, err := engine.GetUndoState(saveSessionID, slot)
			if err != nil {
				t.Fatalf("GetUndoState: %v", err)
			}
			if !state.Available {
				t.Fatalf("no undo point after the batch add: %+v", state)
			}
			undone, err := engine.UndoCharacterChanges(saveSessionID, slot, state.UndoToken, "1")
			if err != nil {
				t.Fatalf("UndoCharacterChanges: %v", err)
			}
			// The undo carries the scopes of the execution it reverted, not the
			// ones the batch-add kind would statically resolve to.
			if strings.Join(undone.ChangedScopes, ",") != testCase.want {
				t.Errorf("undo changedScopes = %v, want %s", undone.ChangedScopes, testCase.want)
			}
		})
	}
}

// An extra scope resolved at commit time is part of a closed vocabulary. One
// outside it refuses the mutation instead of vanishing from the result.
func TestAnExtraScopeOutsideTheVocabularyIsRejected(t *testing.T) {
	t.Parallel()

	if _, err := changedScopesForMutationKind(kindAddItemsToContainers, "inventory.common"); err == nil {
		t.Error("an unregistered extra scope was accepted")
	}
	if _, err := changedScopesForMutationKind(kindAddItemsToContainers, ""); err == nil {
		t.Error("an empty extra scope was accepted")
	}
}

// The risk of a batch add is a property of the execution: an ordinary batch is
// normal, and one carrying a ban-risk resource is recorded as a ban risk and
// therefore demands its own confirmation in Review Changes.
func TestBatchAddRecordsTheRiskOfTheConcreteExecution(t *testing.T) {
	cases := []struct {
		name    string
		banRisk bool
		want    OperationRisk
		count   int
	}{
		{"ordinary batch", false, OperationRiskNormal, 0},
		{"ban risk batch", true, OperationRiskBanRisk, 1},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			engine, saveSessionID, slot := loadBatchSession(t)

			addition := batchAddition(addItemTestGoodsID, 5)
			addition.BanRisk = testCase.banRisk
			if _, err := engine.AddItemsToContainers(
				saveSessionID, slot, []ItemAddition{addition}, nil, "0"); err != nil {
				t.Fatalf("AddItemsToContainers: %v", err)
			}

			entry := batchHistory(t, engine, saveSessionID).Operations[0]
			if entry.Risk != testCase.want {
				t.Errorf("recorded risk = %q, want %q", entry.Risk, testCase.want)
			}

			validation, err := engine.ValidateReviewChanges(saveSessionID, "1")
			if err != nil {
				t.Fatalf("ValidateReviewChanges: %v", err)
			}
			if validation.BanRiskCount != testCase.count {
				t.Errorf("banRiskCount = %d, want %d", validation.BanRiskCount, testCase.count)
			}
			if !validation.Valid {
				t.Fatalf("validation is not valid: %+v", validation)
			}

			// Review Changes is the second gate. The confirmation the Add itself
			// took is not carried over: saving the session needs its own.
			err = requireReviewAuthorization(
				engine.sessions[saveSessionID].session, "1", validation.ValidationToken, true, false)
			if testCase.banRisk && err == nil {
				t.Error("Save was authorized without a separate ban-risk confirmation")
			}
			if !testCase.banRisk && err != nil {
				t.Errorf("an ordinary batch demanded a ban-risk confirmation: %v", err)
			}
			if err := requireReviewAuthorization(
				engine.sessions[saveSessionID].session, "1", validation.ValidationToken, true, true,
			); err != nil {
				t.Errorf("a confirmed Save was refused: %v", err)
			}
		})
	}
}
