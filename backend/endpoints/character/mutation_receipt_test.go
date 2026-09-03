package character

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Stage 3b.2 embeds the shared MutationReceipt in the public results of the
// Character mutations. The receipt is not reassembled here: it is exactly the
// one the central commit path produced, so these tests assert the public shape
// and the values only that path can know.
//
// The scope lists below are written out literally instead of being read back
// from saveengine.ChangedScopesForMutationKind. A test that asks the same table
// the implementation asks proves only that the table was read; these lists are
// the reviewed public contract of each endpoint.

var (
	// A slot-wide rewrite invalidates every per-character read surface.
	characterSlotScopes = []string{
		"save.session", "character.list", "character.profile", "character.stats",
		"character.appearance", "inventory", "storage", "equipment.loadout",
		"world.flags", "diagnostics.report",
	}
	// Gender lives in the appearance block and in the profile.
	characterAppearanceScopes = []string{
		"save.session", "character.profile", "character.appearance", "diagnostics.report",
	}
	// The name is part of the list summary as well as of the profile.
	characterNameScopes = []string{
		"save.session", "character.list", "character.profile", "diagnostics.report",
	}
	// Attributes move the level, which the list summary reports too.
	characterBuildScopes = []string{
		"save.session", "character.list", "character.profile", "character.stats",
		"diagnostics.report",
	}
	// Held runes are reported by GetCharacterStats, so the stats scope moves with
	// the universal ones.
	characterRunesScopes = []string{"save.session", "character.stats", "diagnostics.report"}
)

func TestCloneCharacterResultCarriesItsCommitReceipt(t *testing.T) {
	engine, saveSessionID := loadCharacterReceiptSession(t)
	// A clone needs a named source; the naming mutation itself commits revision 1.
	if _, err := engine.SetCharacterName(
		saveSessionID, getCharacterStatsSlot, "Ranni", "0"); err != nil {
		t.Fatalf("SetCharacterName fixture setup: %v", err)
	}

	result, err := CloneCharacter(engine, saveSessionID, getCharacterStatsSlot, 5, "1")
	if err != nil {
		t.Fatalf("CloneCharacter: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, saveSessionID, CloneCharacterEndpointID, "2")
	assertChangedScopes(t, result.ChangedScopes, characterSlotScopes)
	assertFlatReceiptJSON(t, result, []string{"sourceCharacterID", "targetSlotID", "name"})
	if result.SourceCharacterID != getCharacterStatsSlot || result.TargetSlotID != 5 {
		t.Errorf("result = %+v, want the domain fields of the clone preserved", result)
	}

	// An adjacent execution of the same endpoint gets its own identifier and the
	// next revision, so the receipt describes one execution and not the endpoint.
	second, err := CloneCharacter(engine, saveSessionID, getCharacterStatsSlot, 6, "2")
	if err != nil {
		t.Fatalf("second CloneCharacter: %v", err)
	}
	assertMutationReceipt(t, second.MutationReceipt, saveSessionID, CloneCharacterEndpointID, "3")
	if second.OperationID == result.OperationID {
		t.Errorf("two executions shared operationID %q", result.OperationID)
	}
}

func TestDeleteCharacterResultCarriesItsCommitReceipt(t *testing.T) {
	engine, saveSessionID := loadCharacterReceiptSession(t)

	result, err := DeleteCharacter(engine, saveSessionID, getCharacterStatsSlot, "0")
	if err != nil {
		t.Fatalf("DeleteCharacter: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, saveSessionID, DeleteCharacterEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, characterSlotScopes)
	assertFlatReceiptJSON(t, result, []string{"characterID"})
	if result.CharacterID != getCharacterStatsSlot {
		t.Errorf("characterID = %d, want %d", result.CharacterID, getCharacterStatsSlot)
	}
}

func TestSetCharacterNameResultCarriesItsCommitReceipt(t *testing.T) {
	engine, saveSessionID := loadCharacterReceiptSession(t)

	result, err := SetCharacterName(engine, saveSessionID, getCharacterStatsSlot, "Melina", "0")
	if err != nil {
		t.Fatalf("SetCharacterName: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, saveSessionID, SetCharacterNameEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, characterNameScopes)
	assertFlatReceiptJSON(t, result, []string{"characterID", "name"})
	if result.Name != "Melina" {
		t.Errorf("name = %q, want the committed name", result.Name)
	}
}

func TestSetCharacterRunesResultCarriesItsCommitReceipt(t *testing.T) {
	engine, saveSessionID := loadCharacterReceiptSession(t)

	result, err := SetCharacterRunes(engine, saveSessionID, getCharacterStatsSlot, 4_242, "0")
	if err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, saveSessionID, SetCharacterRunesEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, characterRunesScopes)
	assertFlatReceiptJSON(t, result, []string{"characterID", "runes"})
	if result.Runes != 4_242 {
		t.Errorf("runes = %d, want the committed value", result.Runes)
	}
}

func TestSetCharacterStatsResultCarriesItsCommitReceipt(t *testing.T) {
	engine, saveSessionID := loadCharacterReceiptSession(t)

	result, err := SetCharacterStats(
		engine, saveSessionID, getCharacterStatsSlot, setCharacterStatsAttributes,
		"recalculate", "0")
	if err != nil {
		t.Fatalf("SetCharacterStats: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, saveSessionID, SetCharacterStatsEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, characterBuildScopes)
	assertFlatReceiptJSON(t, result,
		[]string{"characterID", "attributes", "level", "soulMemory"})
	if result.Attributes != setCharacterStatsAttributes || result.Level != 44 {
		t.Errorf("result = %+v, want the committed attributes at level 44", result)
	}
}

func TestSetCharacterStartingClassResultCarriesItsCommitReceipt(t *testing.T) {
	engine, saveSessionID := loadCharacterReceiptSession(t)

	result, err := SetCharacterStartingClass(
		engine, saveSessionID, getCharacterStatsSlot, 4, true, "0")
	if err != nil {
		t.Fatalf("SetCharacterStartingClass: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, saveSessionID,
		SetCharacterStartingClassEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, characterBuildScopes)
	assertFlatReceiptJSON(t, result,
		[]string{"characterID", "startingClassID", "attributes", "level", "soulMemory"})
	if result.StartingClassID != 4 || result.Level != 6 {
		t.Errorf("result = %+v, want the Astrologer base at level 6", result)
	}
}

func TestSetCharacterAppearanceResultCarriesItsCommitReceipt(t *testing.T) {
	engine, saveSessionID := loadCharacterAppearanceReceiptSession(t)

	values := setCharacterAppearanceEndpointValues()
	result, err := SetCharacterAppearance(
		engine, saveSessionID, getCharacterAppearanceSlot, values, "0")
	if err != nil {
		t.Fatalf("SetCharacterAppearance: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, saveSessionID,
		SetCharacterAppearanceEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, characterAppearanceScopes)
	assertFlatReceiptJSON(t, result, []string{"characterID", "appearance"})
	if !reflect.DeepEqual(result.Appearance, values) {
		t.Errorf("appearance = %+v, want the committed values", result.Appearance)
	}
}

// SetCharacterGender and SetCharacterAppearance share one private SaveEngine
// appearance writer. The writer must report the public entry point that was
// called, so the two kinds can never collapse into one.
func TestSetCharacterGenderKeepsItsOwnKindOnTheSharedAppearanceWriter(t *testing.T) {
	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}
	engine, saveSessionID := loadCharacterAppearanceReceiptSession(t)

	gender, err := SetCharacterGender(
		engine, gameCatalog, saveSessionID, getCharacterAppearanceSlot, 1, "0")
	if err != nil {
		t.Fatalf("SetCharacterGender: %v", err)
	}
	assertMutationReceipt(t, gender.MutationReceipt, saveSessionID,
		SetCharacterGenderEndpointID, "1")
	assertChangedScopes(t, gender.ChangedScopes, characterAppearanceScopes)
	assertFlatReceiptJSON(t, gender, []string{"characterID", "presetID", "appearance"})
	if gender.PresetID == "" {
		t.Error("presetID is empty, want the confirmed default preset of the body type")
	}

	appearance, err := SetCharacterAppearance(
		engine, saveSessionID, getCharacterAppearanceSlot,
		setCharacterAppearanceEndpointValues(), "1")
	if err != nil {
		t.Fatalf("SetCharacterAppearance: %v", err)
	}
	if appearance.OperationKind == gender.OperationKind {
		t.Fatalf("both appearance entry points reported operationKind %q", gender.OperationKind)
	}
	if appearance.OperationID == gender.OperationID {
		t.Fatalf("two executions shared operationID %q", gender.OperationID)
	}
}

// Undo carries two kinds at once. Its own operationKind is always
// undo_character_changes; undoneOperationKind names the reverted mutation, and
// the changed scopes are the reverted mutation's scopes.
func TestUndoCharacterChangesReportsBothKindsSeparately(t *testing.T) {
	engine, saveSessionID := loadCharacterReceiptSession(t)

	mutated, err := SetCharacterStats(
		engine, saveSessionID, getCharacterStatsSlot, setCharacterStatsAttributes,
		"recalculate", "0")
	if err != nil {
		t.Fatalf("SetCharacterStats: %v", err)
	}
	state, err := GetUndoState(engine, saveSessionID, getCharacterStatsSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}

	result, err := UndoCharacterChanges(
		engine, saveSessionID, getCharacterStatsSlot, state.UndoToken, "1")
	if err != nil {
		t.Fatalf("UndoCharacterChanges: %v", err)
	}
	assertUndoReceipt(t, result.MutationReceipt, saveSessionID, SetCharacterStatsEndpointID, "2")
	// The reverted mutation decides the scopes, so an undo of a statistics change
	// reports the statistics scopes and never a catch-all.
	assertChangedScopes(t, result.ChangedScopes, characterBuildScopes)
	assertFlatReceiptJSON(t, result, []string{"characterID", "undoneOperationKind"})
	if result.UndoneOperationKind != SetCharacterStatsEndpointID {
		t.Errorf("undoneOperationKind = %q, want %q",
			result.UndoneOperationKind, SetCharacterStatsEndpointID)
	}
	if result.OperationKind != UndoCharacterChangesEndpointID {
		t.Errorf("operationKind = %q, want %q",
			result.OperationKind, UndoCharacterChangesEndpointID)
	}
	if result.OperationID == mutated.OperationID {
		t.Errorf("the undo reused the operationID %q of the mutation it reverted",
			mutated.OperationID)
	}
}

// A rejected mutation returns the complete zero result. It must expose neither
// the identifier of an execution that never happened nor a partial receipt or a
// partially filled domain field.
func TestRejectedCharacterMutationsExposeNoPartialResult(t *testing.T) {
	engine, saveSessionID := loadCharacterReceiptSession(t)
	appearanceEngine, appearanceSession := loadCharacterAppearanceReceiptSession(t)
	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}

	// Every case is a stale expectedRevision, the one rejection every mutation of
	// this batch shares, so the assertion covers the general invariant.
	if _, err := engine.SetCharacterName(
		saveSessionID, getCharacterStatsSlot, "Ranni", "0"); err != nil {
		t.Fatalf("SetCharacterName fixture setup: %v", err)
	}
	cloned, err := CloneCharacter(engine, saveSessionID, getCharacterStatsSlot, 5, "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", cloned)
	}
	if !reflect.DeepEqual(cloned, CloneCharacterResult{}) {
		t.Errorf("rejected CloneCharacter = %+v, want the complete zero result", cloned)
	}

	deleted, err := DeleteCharacter(engine, saveSessionID, getCharacterStatsSlot, "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", deleted)
	}
	if !reflect.DeepEqual(deleted, DeleteCharacterResult{}) {
		t.Errorf("rejected DeleteCharacter = %+v, want the complete zero result", deleted)
	}

	named, err := SetCharacterName(engine, saveSessionID, getCharacterStatsSlot, "Melina", "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", named)
	}
	if !reflect.DeepEqual(named, SetCharacterNameResult{}) {
		t.Errorf("rejected SetCharacterName = %+v, want the complete zero result", named)
	}

	runed, err := SetCharacterRunes(engine, saveSessionID, getCharacterStatsSlot, 1, "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", runed)
	}
	if !reflect.DeepEqual(runed, SetCharacterRunesResult{}) {
		t.Errorf("rejected SetCharacterRunes = %+v, want the complete zero result", runed)
	}

	statted, err := SetCharacterStats(
		engine, saveSessionID, getCharacterStatsSlot, setCharacterStatsAttributes,
		"recalculate", "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", statted)
	}
	if !reflect.DeepEqual(statted, SetCharacterStatsResult{}) {
		t.Errorf("rejected SetCharacterStats = %+v, want the complete zero result", statted)
	}

	classed, err := SetCharacterStartingClass(
		engine, saveSessionID, getCharacterStatsSlot, 4, true, "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", classed)
	}
	if !reflect.DeepEqual(classed, SetCharacterStartingClassResult{}) {
		t.Errorf("rejected SetCharacterStartingClass = %+v, want the complete zero result", classed)
	}

	undone, err := UndoCharacterChanges(engine, saveSessionID, getCharacterStatsSlot, "token", "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", undone)
	}
	if !reflect.DeepEqual(undone, UndoCharacterChangesResult{}) {
		t.Errorf("rejected UndoCharacterChanges = %+v, want the complete zero result", undone)
	}

	appearance, err := SetCharacterAppearance(
		appearanceEngine, appearanceSession, getCharacterAppearanceSlot,
		setCharacterAppearanceEndpointValues(), "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", appearance)
	}
	if !reflect.DeepEqual(appearance, SetCharacterAppearanceResult{}) {
		t.Errorf("rejected SetCharacterAppearance = %+v, want the complete zero result", appearance)
	}

	gender, err := SetCharacterGender(
		appearanceEngine, gameCatalog, appearanceSession, getCharacterAppearanceSlot, 1, "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", gender)
	}
	if !reflect.DeepEqual(gender, SetCharacterGenderResult{}) {
		t.Errorf("rejected SetCharacterGender = %+v, want the complete zero result", gender)
	}
}

func loadCharacterReceiptSession(t *testing.T) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterStatsFixture(t, GetCharacterStatsResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func loadCharacterAppearanceReceiptSession(t *testing.T) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeGetCharacterAppearanceFixture(t, GetCharacterAppearanceResult{}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
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

// assertUndoReceipt checks the receipt of one undo execution. Its own kind is
// always undo_character_changes and must never be the kind it reverted.
func assertUndoReceipt(
	t *testing.T,
	receipt saveengine.MutationReceipt,
	saveSessionID string,
	undoneKind string,
	saveRevision string,
) {
	t.Helper()

	assertMutationReceipt(t, receipt, saveSessionID, UndoCharacterChangesEndpointID, saveRevision)
	if receipt.OperationKind == undoneKind {
		t.Errorf("undo reported the reverted kind %q as its own operationKind", undoneKind)
	}
}

func assertChangedScopes(t *testing.T, got []string, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
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
