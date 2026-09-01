package saveengine

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// loadReceiptSession loads the shared undo fixture, whose slot setRunesTestSlot
// is active and mutable through SetCharacterRunes. It is the smallest fixture
// that lets a receipt test drive a real committed mutation.
func loadReceiptSession(t *testing.T) (*Engine, string) {
	t.Helper()

	source, _ := writeUndoFixture(t, PlatformPC)
	engine := New()
	loaded, err := engine.LoadSave(source, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

// commitReceipt runs one no-op mutation of operationKind and returns its
// receipt. A no-op still advances the revision under the existing contract, so
// this exercises the whole commit path without depending on a domain writer.
func commitReceipt(t *testing.T, engine *Engine, saveSessionID, operationKind string) MutationReceipt {
	t.Helper()

	receipt, err := engine.commitRevision(saveSessionID, operationKind, func(*loadedSave) error {
		return nil
	})
	if err != nil {
		t.Fatalf("commitRevision(%s): %v", operationKind, err)
	}
	return receipt
}

// operationID names one execution, not a kind: two successful executions of the
// same operationKind must never share it.
func TestTwoExecutionsOfOneOperationKindGetDifferentOperationIDs(t *testing.T) {
	engine, saveSessionID := loadReceiptSession(t)

	first := commitReceipt(t, engine, saveSessionID, kindSetSaveAccountID)
	second := commitReceipt(t, engine, saveSessionID, kindSetSaveAccountID)

	if first.OperationID == "" || second.OperationID == "" {
		t.Fatalf("operationIDs = %q and %q, want two non-empty identifiers",
			first.OperationID, second.OperationID)
	}
	if first.OperationID == second.OperationID {
		t.Fatalf("two executions of %s shared operationID %q",
			kindSetSaveAccountID, first.OperationID)
	}
	if first.OperationKind != kindSetSaveAccountID || second.OperationKind != kindSetSaveAccountID {
		t.Fatalf("operationKinds = %q and %q, want both %q",
			first.OperationKind, second.OperationKind, kindSetSaveAccountID)
	}

	// The identifier is opaque and unpredictable: it is neither the kind, nor the
	// revision, nor anything a caller could construct from what it already knows.
	for _, receipt := range []MutationReceipt{first, second} {
		if !strings.HasPrefix(receipt.OperationID, operationIDPrefix) {
			t.Errorf("operationID %q does not carry the %q prefix", receipt.OperationID, operationIDPrefix)
		}
		hexPart := strings.TrimPrefix(receipt.OperationID, operationIDPrefix)
		if len(hexPart) != 32 {
			t.Errorf("operationID %q carries %d hex characters, want 32", receipt.OperationID, len(hexPart))
		}
		for _, character := range hexPart {
			if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
				t.Errorf("operationID %q contains non-hex character %q", receipt.OperationID, character)
				break
			}
		}
		if receipt.OperationID == receipt.OperationKind ||
			receipt.OperationID == receipt.SaveRevision ||
			receipt.OperationID == receipt.SaveSessionID {
			t.Errorf("operationID %q repeats another receipt field", receipt.OperationID)
		}
	}
}

// The receipt describes the revision the commit path created, for the session it
// was asked to mutate.
func TestReceiptReportsTheCommittedSessionAndRevision(t *testing.T) {
	engine, saveSessionID := loadReceiptSession(t)

	for _, want := range []string{"1", "2", "3"} {
		receipt := commitReceipt(t, engine, saveSessionID, kindSetSaveAccountID)
		if receipt.SaveSessionID != saveSessionID {
			t.Fatalf("receipt saveSessionID = %q, want %q", receipt.SaveSessionID, saveSessionID)
		}
		if receipt.SaveRevision != want {
			t.Fatalf("receipt saveRevision = %q, want %q", receipt.SaveRevision, want)
		}
		if !IsCanonicalRevision(receipt.SaveRevision) {
			t.Fatalf("receipt saveRevision %q is not canonical", receipt.SaveRevision)
		}

		info, err := engine.GetSessionInfo(saveSessionID)
		if err != nil {
			t.Fatalf("GetSessionInfo: %v", err)
		}
		if info.SaveRevision != receipt.SaveRevision {
			t.Fatalf("session saveRevision = %q, want the revision the receipt reported %q",
				info.SaveRevision, receipt.SaveRevision)
		}
	}
}

// One representative mutation of every area must report exactly the scopes its
// readers depend on: no catch-all, no scope narrowed to the setter's own file.
func TestRepresentativeMutationsReportTheirExactChangedScopes(t *testing.T) {
	testCases := []struct {
		operationKind string
		want          []string
	}{
		{kindAddItemToInventory, []string{"save.session", "inventory", "diagnostics.report"}},
		{kindAddItemToStorage, []string{"save.session", "storage", "diagnostics.report"}},
		{kindRemoveOwnedItem, []string{"save.session", "inventory", "storage", "diagnostics.report"}},
		{kindSetOwnedItemQuantity, []string{
			"save.session", "inventory", "storage", "equipment.loadout", "diagnostics.report"}},
		{kindSetInventoryOrder, []string{"save.session", "inventory", "diagnostics.report"}},
		{kindSetStorageOrder, []string{"save.session", "storage", "diagnostics.report"}},
		// Both moves change the source and the destination container.
		{kindMoveOwnedItemToInventory, []string{
			"save.session", "inventory", "storage", "diagnostics.report"}},
		{kindMoveOwnedItemToStorage, []string{
			"save.session", "inventory", "storage", "diagnostics.report"}},
		// All four weapon writers resolve one common record through an opaque
		// OwnedItemID, so each of them accepts an Inventory or a Storage record and
		// must report both containers. The four *SupportsStorageCommon tests of
		// those writers are the evidence.
		{kindSetWeaponAshOfWar, []string{
			"save.session", "inventory", "storage", "equipment.loadout", "diagnostics.report"}},
		{kindSetWeaponInfusion, []string{
			"save.session", "inventory", "storage", "equipment.loadout", "diagnostics.report"}},
		{kindSetWeaponUpgradeLevel, []string{
			"save.session", "inventory", "storage", "equipment.loadout", "diagnostics.report"}},
		{kindSetSpiritAshUpgradeLevel, []string{
			"save.session", "inventory", "storage", "equipment.loadout", "diagnostics.report"}},
		{kindSetEquippedArmor, []string{"save.session", "equipment.loadout", "diagnostics.report"}},
		{kindSetCharacterStats, []string{
			"save.session", "character.list", "character.profile", "character.stats",
			"diagnostics.report"}},
		{kindSetCharacterAppearance, []string{
			"save.session", "character.profile", "character.appearance", "diagnostics.report"}},
		{kindSetCharacterName, []string{
			"save.session", "character.list", "character.profile", "diagnostics.report"}},
		{kindCloneCharacter, []string{
			"save.session", "character.list", "character.profile", "character.stats",
			"character.appearance", "inventory", "storage", "equipment.loadout", "world.flags",
			"diagnostics.report"}},
		{kindSetGraceVisited, []string{"save.session", "world.flags", "diagnostics.report"}},
		{kindSetSpectralSteedAttire, []string{
			"save.session", "inventory", "world.flags", "diagnostics.report"}},
		{kindSetNetworkSettings, []string{"save.session", "network", "diagnostics.report"}},
		{kindApplyNetworkPreset, []string{"save.session", "network", "diagnostics.report"}},
		{kindSetFavoritePreset, []string{"save.session", "favorites", "diagnostics.report"}},
		{kindApplyRepairs, []string{
			"save.session", "character.list", "character.profile", "character.stats",
			"inventory", "storage", "equipment.loadout", "diagnostics.report"}},
		{kindApplyBuildTemplate, []string{
			"save.session", "character.list", "character.profile", "character.stats",
			"equipment.loadout", "diagnostics.report"}},
		// WriteSave, the account identifier and held runes reach no domain getter
		// today: they invalidate the session and the pinned validation report only.
		{kindWriteSave, []string{"save.session", "diagnostics.report"}},
		{kindSetSaveAccountID, []string{"save.session", "diagnostics.report"}},
		{kindSetCharacterRunes, []string{"save.session", "diagnostics.report"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.operationKind, func(t *testing.T) {
			engine, saveSessionID := loadReceiptSession(t)
			receipt := commitReceipt(t, engine, saveSessionID, testCase.operationKind)

			if strings.Join(receipt.ChangedScopes, ",") != strings.Join(testCase.want, ",") {
				t.Fatalf("changedScopes = %v, want exactly %v", receipt.ChangedScopes, testCase.want)
			}
		})
	}
}

// The scope order is the canonical vocabulary order and never map iteration
// order, so the same kind always reports the same sequence.
func TestChangedScopesAreDeterministicNonEmptyAndFreeOfDuplicates(t *testing.T) {
	for _, kind := range MutationKinds() {
		scopes, err := ChangedScopesForMutationKind(kind)
		if err != nil {
			t.Fatalf("ChangedScopesForMutationKind(%q): %v", kind, err)
		}
		if len(scopes) == 0 {
			t.Errorf("kind %q resolves to no scope", kind)
		}

		position := 0
		seen := make(map[string]bool, len(scopes))
		for _, scope := range scopes {
			if scope == "" {
				t.Errorf("kind %q resolves an empty scope", kind)
			}
			if seen[scope] {
				t.Errorf("kind %q repeats scope %q", kind, scope)
			}
			seen[scope] = true
			for position < len(changedScopeOrder) && changedScopeOrder[position] != scope {
				position++
			}
			if position == len(changedScopeOrder) {
				t.Fatalf("kind %q reported %v, which is not in canonical order %v",
					kind, scopes, changedScopeOrder)
			}
		}

		// Every committed save-session mutation moves the revision and invalidates a
		// validation report pinned to the previous one.
		if !seen[ScopeSaveSession] || !seen[ScopeDiagnosticsReport] {
			t.Errorf("kind %q reported %v without the universal scopes", kind, scopes)
		}
		// There is no catch-all scope in the vocabulary.
		for _, scope := range scopes {
			if scope == "all" || scope == "*" {
				t.Errorf("kind %q reported catch-all scope %q", kind, scope)
			}
		}
	}
}

// The receipt owns its scope slice: a caller can neither reorder nor extend the
// list a later receipt would report.
func TestReceiptChangedScopesAreCopiedPerExecution(t *testing.T) {
	engine, saveSessionID := loadReceiptSession(t)

	first := commitReceipt(t, engine, saveSessionID, kindSetGraceVisited)
	first.ChangedScopes[0] = "tampered"
	second := commitReceipt(t, engine, saveSessionID, kindSetGraceVisited)

	want := []string{"save.session", "world.flags", "diagnostics.report"}
	if strings.Join(second.ChangedScopes, ",") != strings.Join(want, ",") {
		t.Fatalf("changedScopes after tampering with an earlier receipt = %v, want %v",
			second.ChangedScopes, want)
	}
}

// An unknown or empty operationKind is a programming error and must be refused
// before the mutation callback can touch the snapshot.
func TestUnknownOperationKindIsRejectedBeforeTheSnapshotChanges(t *testing.T) {
	testCases := []struct {
		name          string
		operationKind string
		wantError     string
	}{
		{"empty", "", "operationKind is required"},
		{"unknown", "definitely_not_a_mutation", `unknown operationKind "definitely_not_a_mutation"`},
		{"not a save-session mutation", "create_build_template",
			`unknown operationKind "create_build_template"`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			engine, saveSessionID := loadReceiptSession(t)
			before, err := engine.GetSessionInfo(saveSessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}

			ran := false
			receipt, err := engine.commitRevision(
				saveSessionID, testCase.operationKind, func(*loadedSave) error {
					ran = true
					return nil
				})
			if err == nil {
				t.Fatal("commitRevision accepted an unregistered operationKind")
			}
			if err.Error() != testCase.wantError {
				t.Fatalf("error = %q, want %q", err.Error(), testCase.wantError)
			}
			if ran {
				t.Error("the mutation callback ran for an unregistered operationKind")
			}
			if !isZeroReceipt(receipt) {
				t.Errorf("receipt = %+v, want the zero receipt", receipt)
			}

			after, err := engine.GetSessionInfo(saveSessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo after the rejection: %v", err)
			}
			if after != before {
				t.Errorf("session = %+v, want the unchanged %+v", after, before)
			}
		})
	}
}

// The identifier is minted before the first change, so a generator failure
// rejects the mutation instead of surfacing after the revision was committed.
func TestOperationIDGeneratorFailureLeavesTheSessionUntouched(t *testing.T) {
	engine, saveSessionID := loadReceiptSession(t)
	if _, err := engine.SetCharacterRunes(saveSessionID, setRunesTestSlot, 500, "0"); err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}

	before, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	undoBefore, err := engine.GetUndoState(saveSessionID, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	held := engine.sessions[saveSessionID]
	snapshotBefore := string(held.snapshot.data)

	failure := errors.New("no entropy available")
	engine.newOperationID = func() (string, error) { return "", failure }

	ran := false
	receipt, err := engine.commitCharacterRevision(
		saveSessionID, kindSetCharacterRunes, setRunesTestSlot, func(*loadedSave) error {
			ran = true
			return nil
		})
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v, want the generator failure", err)
	}
	if ran {
		t.Error("the mutation callback ran after the identifier could not be minted")
	}
	if !isZeroReceipt(receipt) {
		t.Errorf("receipt = %+v, want the zero receipt", receipt)
	}

	// A refused mutation leaves the snapshot, the revision, the dirty flag and the
	// undo point exactly as they were.
	if string(held.snapshot.data) != snapshotBefore {
		t.Error("a refused mutation changed the snapshot")
	}
	after, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after the rejection: %v", err)
	}
	if after != before {
		t.Errorf("session = %+v, want the unchanged %+v", after, before)
	}
	undoAfter, err := engine.GetUndoState(saveSessionID, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the rejection: %v", err)
	}
	if undoAfter != undoBefore {
		t.Errorf("undo state = %+v, want the unchanged %+v", undoAfter, undoBefore)
	}
	if len(held.session.ownedByID) != len(held.session.ownedByLocator) {
		t.Error("a refused mutation left the identity registry inconsistent")
	}
}

// scriptedOperationIDs hands out the given identifiers in order and repeats the
// last one forever. It drives the collision path deterministically instead of
// waiting for the real generator to repeat itself, which it never will.
func scriptedOperationIDs(values ...string) func() (string, error) {
	index := 0
	return func() (string, error) {
		value := values[index]
		if index < len(values)-1 {
			index++
		}
		return value, nil
	}
}

const (
	firstScriptedOperationID  = operationIDPrefix + "000000000000000000000000000000aa"
	secondScriptedOperationID = operationIDPrefix + "000000000000000000000000000000bb"
)

// Uniqueness of the identifiers one engine issues is a hard guarantee, not a
// probability: a generator that repeats an identifier the engine already
// reserved is asked again until it produces a fresh one.
func TestARepeatedOperationIDIsRetriedUntilItIsUnique(t *testing.T) {
	engine, saveSessionID := loadReceiptSession(t)
	engine.newOperationID = scriptedOperationIDs(
		firstScriptedOperationID, firstScriptedOperationID, secondScriptedOperationID)

	first := commitReceipt(t, engine, saveSessionID, kindSetSaveAccountID)
	if first.OperationID != firstScriptedOperationID {
		t.Fatalf("first operationID = %q, want %q", first.OperationID, firstScriptedOperationID)
	}

	// The generator offers the reserved identifier again before it yields a fresh
	// one. The second mutation must succeed, and never with the repeated value.
	second := commitReceipt(t, engine, saveSessionID, kindSetSaveAccountID)
	if second.OperationID == first.OperationID {
		t.Fatalf("the second execution reused operationID %q", second.OperationID)
	}
	if second.OperationID != secondScriptedOperationID {
		t.Fatalf("second operationID = %q, want %q", second.OperationID, secondScriptedOperationID)
	}
	if second.SaveRevision != "2" {
		t.Fatalf("second saveRevision = %q, want %q", second.SaveRevision, "2")
	}
}

// A generator that can only repeat a reserved identifier exhausts the bounded
// number of attempts and refuses the mutation, exactly like a generator that
// errors: before the callback, and with nothing moved.
func TestAnExhaustedOperationIDGeneratorRejectsTheMutationBeforeItRuns(t *testing.T) {
	engine, saveSessionID := loadReceiptSession(t)
	if _, err := engine.SetCharacterRunes(saveSessionID, setRunesTestSlot, 500, "0"); err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}

	engine.newOperationID = func() (string, error) { return firstScriptedOperationID, nil }
	accepted, err := engine.commitCharacterRevision(
		saveSessionID, kindSetCharacterRunes, setRunesTestSlot, func(*loadedSave) error {
			return nil
		})
	if err != nil {
		t.Fatalf("the first execution of a repeating generator failed: %v", err)
	}
	if accepted.OperationID != firstScriptedOperationID {
		t.Fatalf("accepted operationID = %q, want %q", accepted.OperationID, firstScriptedOperationID)
	}

	before, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	undoBefore, err := engine.GetUndoState(saveSessionID, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	held := engine.sessions[saveSessionID]
	snapshotBefore := string(held.snapshot.data)
	ownedByIDBefore := len(held.session.ownedByID)
	ownedByLocatorBefore := len(held.session.ownedByLocator)

	ran := false
	receipt, err := engine.commitCharacterRevision(
		saveSessionID, kindSetCharacterRunes, setRunesTestSlot, func(*loadedSave) error {
			ran = true
			return nil
		})
	wantError := fmt.Sprintf(
		"cannot create a unique mutation operation identifier after %d attempts",
		operationIDMintAttempts)
	if err == nil || err.Error() != wantError {
		t.Fatalf("error = %v, want %q", err, wantError)
	}
	if ran {
		t.Error("the mutation callback ran after the identifier could not be reserved")
	}
	if !isZeroReceipt(receipt) {
		t.Errorf("receipt = %+v, want the zero receipt", receipt)
	}

	// A refused mutation leaves the snapshot, the revision, the dirty flag, the
	// undo point and the identity registry exactly as they were.
	if string(held.snapshot.data) != snapshotBefore {
		t.Error("a refused mutation changed the snapshot")
	}
	after, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after the rejection: %v", err)
	}
	if after != before {
		t.Errorf("session = %+v, want the unchanged %+v", after, before)
	}
	undoAfter, err := engine.GetUndoState(saveSessionID, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the rejection: %v", err)
	}
	if undoAfter != undoBefore {
		t.Errorf("undo state = %+v, want the unchanged %+v", undoAfter, undoBefore)
	}
	if len(held.session.ownedByID) != ownedByIDBefore ||
		len(held.session.ownedByLocator) != ownedByLocatorBefore {
		t.Error("a refused mutation changed the identity registry")
	}
}

// WriteSave prepares its receipt before it serializes, validates or replaces the
// target, so a generator failure can never leave a written file behind an error.
func TestWriteSaveRefusesBeforeWritingWhenTheOperationIDCannotBeMinted(t *testing.T) {
	engine, saveSessionID := loadReceiptSession(t)
	target := writeFixture(t, "write-target.sl2", pcHeader(), pcFixtureSize)

	before, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	original := readTargetForTest(t, target)

	failure := errors.New("no entropy available")
	engine.newOperationID = func() (string, error) { return "", failure }

	if _, err := engine.WriteSave(saveSessionID, before.SaveRevision, target); !errors.Is(err, failure) {
		t.Fatalf("WriteSave error = %v, want the generator failure", err)
	}
	if string(readTargetForTest(t, target)) != string(original) {
		t.Error("a refused WriteSave replaced the target file")
	}
	after, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after the refusal: %v", err)
	}
	if after != before {
		t.Errorf("session = %+v, want the unchanged %+v", after, before)
	}
}

// An undo reports the scopes of the mutation it reverts, never a catch-all and
// never only the undo baseline.
func TestUndoResolvesTheScopesOfTheRevertedMutation(t *testing.T) {
	testCases := []struct {
		undoneKind string
		want       []string
	}{
		{kindSetCharacterStats, []string{
			"save.session", "character.list", "character.profile", "character.stats",
			"diagnostics.report"}},
		{kindSetInventoryOrder, []string{"save.session", "inventory", "diagnostics.report"}},
		{kindSetGraceVisited, []string{"save.session", "world.flags", "diagnostics.report"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.undoneKind, func(t *testing.T) {
			scopes, err := changedScopesForMutationKind(
				kindUndoCharacterChanges, undoneChangedScopes(testCase.undoneKind)...)
			if err != nil {
				t.Fatalf("undo scopes for %q: %v", testCase.undoneKind, err)
			}
			if strings.Join(scopes, ",") != strings.Join(testCase.want, ",") {
				t.Fatalf("undo changedScopes = %v, want exactly %v", scopes, testCase.want)
			}
		})
	}
}

// The undo path prepares its receipt before the first restore write, so a
// generator failure refuses the undo instead of surfacing after the ranges were
// already restored and a revision committed.
func TestUndoRefusesBeforeRestoringWhenTheOperationIDCannotBeMinted(t *testing.T) {
	engine, saveSessionID := loadReceiptSession(t)
	if _, err := engine.SetCharacterRunes(saveSessionID, setRunesTestSlot, 500, "0"); err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}
	state, err := engine.GetUndoState(saveSessionID, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if !state.Available || state.OperationKind != kindSetCharacterRunes {
		t.Fatalf("undo state = %+v, want an available point for %q", state, kindSetCharacterRunes)
	}

	held := engine.sessions[saveSessionID]
	snapshotBefore := string(held.snapshot.data)
	before, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}

	failure := errors.New("no entropy available")
	engine.newOperationID = func() (string, error) { return "", failure }

	result, err := engine.UndoCharacterChanges(
		saveSessionID, setRunesTestSlot, state.UndoToken, state.SaveRevision)
	if !errors.Is(err, failure) {
		t.Fatalf("UndoCharacterChanges error = %v, want the generator failure", err)
	}
	if !reflect.DeepEqual(result, UndoCharacterChangesResult{}) {
		t.Errorf("result = %+v, want the zero result", result)
	}
	if string(held.snapshot.data) != snapshotBefore {
		t.Error("a refused undo changed the snapshot")
	}
	after, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after the refusal: %v", err)
	}
	if after != before {
		t.Errorf("session = %+v, want the unchanged %+v", after, before)
	}

	// The point survives untouched, so a later undo with a working generator can
	// still consume it.
	engine.newOperationID = nil
	stillThere, err := engine.GetUndoState(saveSessionID, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the refusal: %v", err)
	}
	if stillThere != state {
		t.Fatalf("undo state = %+v, want the unchanged %+v", stillThere, state)
	}
	undone, err := engine.UndoCharacterChanges(
		saveSessionID, setRunesTestSlot, state.UndoToken, state.SaveRevision)
	if err != nil {
		t.Fatalf("UndoCharacterChanges after restoring the generator: %v", err)
	}
	if undone.UndoneOperationKind != kindSetCharacterRunes {
		t.Errorf("undoneOperationKind = %q, want %q", undone.UndoneOperationKind, kindSetCharacterRunes)
	}
}

// Every registered kind is reachable as a concrete constant of this package and
// the list is stable and ordered, so the endpoint conformance test compares two
// complete sets rather than two partial ones.
func TestMutationKindsIsSortedCompleteAndFreeOfEmptyValues(t *testing.T) {
	kinds := MutationKinds()
	if len(kinds) != len(domainChangedScopes) {
		t.Fatalf("MutationKinds returned %d kinds, want %d", len(kinds), len(domainChangedScopes))
	}
	for index, kind := range kinds {
		if kind == "" {
			t.Fatal("MutationKinds returned an empty kind")
		}
		if index > 0 && kinds[index-1] >= kind {
			t.Fatalf("MutationKinds is not sorted at %q", kind)
		}
		if _, registered := domainChangedScopes[kind]; !registered {
			t.Fatalf("MutationKinds returned unregistered kind %q", kind)
		}
	}
}

func readTargetForTest(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

// isZeroReceipt reports whether a rejected mutation returned nothing at all: no
// revision, no scopes and, above all, no operationID for an execution that never
// happened.
func isZeroReceipt(receipt MutationReceipt) bool {
	return receipt.OperationID == "" && receipt.OperationKind == "" &&
		receipt.SaveSessionID == "" && receipt.SaveRevision == "" &&
		receipt.ChangedScopes == nil
}

// assertCommittedReceipt fails unless receipt is the complete receipt of one
// execution of operationKind, committed for saveSessionID at saveRevision. It
// exists so a result type that embeds MutationReceipt is checked against the
// whole contract instead of the two fields it used to carry.
func assertCommittedReceipt(
	t *testing.T,
	receipt MutationReceipt,
	saveSessionID string,
	operationKind string,
	saveRevision string,
) {
	t.Helper()

	if receipt.OperationID == "" {
		t.Errorf("receipt = %+v, want a minted operationID", receipt)
	}
	if receipt.OperationKind != operationKind {
		t.Errorf("operationKind = %q, want %q", receipt.OperationKind, operationKind)
	}
	if receipt.SaveSessionID != saveSessionID {
		t.Errorf("saveSessionID = %q, want %q", receipt.SaveSessionID, saveSessionID)
	}
	if receipt.SaveRevision != saveRevision {
		t.Errorf("saveRevision = %q, want %q", receipt.SaveRevision, saveRevision)
	}
	wantScopes, err := ChangedScopesForMutationKind(operationKind)
	if err != nil {
		t.Fatalf("ChangedScopesForMutationKind(%q): %v", operationKind, err)
	}
	if strings.Join(receipt.ChangedScopes, ",") != strings.Join(wantScopes, ",") {
		t.Errorf("changedScopes = %v, want %v", receipt.ChangedScopes, wantScopes)
	}
}

// assertUndoReceipt fails unless receipt is the complete receipt of one undo
// execution. Undo carries two kinds: its own operationKind, which is always
// kindUndoCharacterChanges, and the kind of the mutation it reverted, which
// decides the changed scopes. kindUndoCharacterChanges owns no domain scope of
// its own, so the expected list is exactly the scope list of undoneKind.
func assertUndoReceipt(
	t *testing.T,
	receipt MutationReceipt,
	saveSessionID string,
	undoneKind string,
	saveRevision string,
) {
	t.Helper()

	if receipt.OperationID == "" {
		t.Errorf("receipt = %+v, want a minted operationID", receipt)
	}
	if receipt.OperationKind != kindUndoCharacterChanges {
		t.Errorf("operationKind = %q, want %q", receipt.OperationKind, kindUndoCharacterChanges)
	}
	if receipt.OperationKind == undoneKind {
		t.Errorf("undo reported the reverted kind %q as its own operationKind", undoneKind)
	}
	if receipt.SaveSessionID != saveSessionID {
		t.Errorf("saveSessionID = %q, want %q", receipt.SaveSessionID, saveSessionID)
	}
	if receipt.SaveRevision != saveRevision {
		t.Errorf("saveRevision = %q, want %q", receipt.SaveRevision, saveRevision)
	}
	wantScopes, err := ChangedScopesForMutationKind(undoneKind)
	if err != nil {
		t.Fatalf("ChangedScopesForMutationKind(%q): %v", undoneKind, err)
	}
	if strings.Join(receipt.ChangedScopes, ",") != strings.Join(wantScopes, ",") {
		t.Errorf("changedScopes = %v, want the scopes of the reverted %q: %v",
			receipt.ChangedScopes, undoneKind, wantScopes)
	}
}

// setOwnedWeaponGameID is shared by exactly two public setters, SetWeaponInfusion
// and SetWeaponUpgradeLevel, and receives its operation kind from whichever one
// called it. Both therefore commit through one writer and still report their own
// kind. This is the SaveEngine half of that guarantee; the endpoint half, which
// also covers the separately implemented SetWeaponAshOfWar, lives in
// backend/endpoints/inventory/mutation_receipt_test.go.
func TestTheSharedWeaponWriterReportsTheKindItWasGiven(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeSetEquippedArmamentsFixture(t, PlatformPC), string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	equipSetWeaponUpgradeFixture(t, engine, loaded.SaveSessionID, PlatformPC)
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}

	upgrade, err := engine.SetWeaponUpgradeLevel(
		loaded.SaveSessionID, setArmamentsSlot, inventory.Records[1].OwnedItemID, 5, "0",
		setWeaponUpgradeCurrent, setWeaponUpgradeTarget, 5)
	if err != nil {
		t.Fatalf("SetWeaponUpgradeLevel: %v", err)
	}
	assertCommittedReceipt(t, upgrade.MutationReceipt, loaded.SaveSessionID,
		kindSetWeaponUpgradeLevel, "1")

	inventory, err = engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory under the new revision: %v", err)
	}
	infusion, err := engine.SetWeaponInfusion(
		loaded.SaveSessionID, setArmamentsSlot, inventory.Records[1].OwnedItemID, "1",
		setWeaponUpgradeTarget, setWeaponUpgradeTarget)
	if err != nil {
		t.Fatalf("SetWeaponInfusion: %v", err)
	}
	assertCommittedReceipt(t, infusion.MutationReceipt, loaded.SaveSessionID,
		kindSetWeaponInfusion, "2")

	if upgrade.OperationKind == infusion.OperationKind {
		t.Fatalf("both weapon setters reported operationKind %q", upgrade.OperationKind)
	}
	if upgrade.OperationID == infusion.OperationID {
		t.Fatalf("two executions shared operationID %q", upgrade.OperationID)
	}
}

// setOwnedWeaponGameID mints its identifier through the same commit path as
// every other mutation, so a generator failure refuses the change before the
// first byte moves. SetWeaponUpgradeLevel is one of its two callers.
func TestTheSharedWeaponWriterRefusesWhenTheOperationIDCannotBeMinted(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeSetEquippedArmamentsFixture(t, PlatformPC), string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	equipSetWeaponUpgradeFixture(t, engine, loaded.SaveSessionID, PlatformPC)
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	held := engine.sessions[loaded.SaveSessionID]
	snapshotBefore := string(held.snapshot.data)
	before, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}

	failure := errors.New("no entropy available")
	engine.newOperationID = func() (string, error) { return "", failure }

	result, err := engine.SetWeaponUpgradeLevel(
		loaded.SaveSessionID, setArmamentsSlot, inventory.Records[1].OwnedItemID, 5, "0",
		setWeaponUpgradeCurrent, setWeaponUpgradeTarget, 5)
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v, want the generator failure", err)
	}
	if !reflect.DeepEqual(result, SetWeaponUpgradeLevelResult{}) {
		t.Errorf("result = %+v, want the complete zero result", result)
	}
	if string(held.snapshot.data) != snapshotBefore {
		t.Error("a refused weapon mutation changed the snapshot")
	}
	after, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo after the rejection: %v", err)
	}
	if after != before {
		t.Errorf("session = %+v, want the unchanged %+v", after, before)
	}
}
