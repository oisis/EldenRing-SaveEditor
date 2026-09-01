package favorites

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Stage 3b.2 embeds the shared MutationReceipt in the public results of the
// three Favorites mutations. The receipt is not reassembled here: it is exactly
// the one the central commit path produced.
//
// The scope lists are written out literally instead of being read back from
// saveengine.ChangedScopesForMutationKind, so they state the reviewed public
// contract of each endpoint rather than echoing the implementation table.
var (
	// A preset slot lives in UserData10 and is read through GetFavoritePresets.
	favoritesScopes = []string{"save.session", "favorites", "diagnostics.report"}
	// Applying a preset writes the appearance of one character instead.
	favoritesAppearanceScopes = []string{
		"save.session", "character.profile", "character.appearance", "diagnostics.report",
	}
)

func TestSetFavoritePresetResultCarriesItsCommitReceipt(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(writeEndpointSetFavoritesFixture(t, 0), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetFavoritePreset(engine, session.SaveSessionID, 4, 0, "0")
	if err != nil {
		t.Fatalf("SetFavoritePreset: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, session.SaveSessionID,
		SetFavoritePresetEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, favoritesScopes)
	assertFlatReceiptJSON(t, result, []string{"favoriteSlotID", "sourceCharacterID"})
	if result.FavoriteSlotID != 4 || result.SourceCharacterID != 0 {
		t.Errorf("result = %+v, want the domain fields preserved", result)
	}

	// An adjacent execution of the same endpoint on another slot gets its own
	// identifier and the next revision.
	second, err := SetFavoritePreset(engine, session.SaveSessionID, 5, 0, "1")
	if err != nil {
		t.Fatalf("second SetFavoritePreset: %v", err)
	}
	assertMutationReceipt(t, second.MutationReceipt, session.SaveSessionID,
		SetFavoritePresetEndpointID, "2")
	if second.OperationID == result.OperationID {
		t.Errorf("two executions shared operationID %q", result.OperationID)
	}
}

func TestDeleteFavoritePresetResultCarriesItsCommitReceipt(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(
		writeEndpointFavoritesFixture(t, "pc", map[int]bool{2: true}), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := DeleteFavoritePreset(engine, session.SaveSessionID, 2, "0")
	if err != nil {
		t.Fatalf("DeleteFavoritePreset: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, session.SaveSessionID,
		DeleteFavoritePresetEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, favoritesScopes)
	assertFlatReceiptJSON(t, result, []string{"favoriteSlotID"})
	if result.FavoriteSlotID != 2 {
		t.Errorf("favoriteSlotID = %d, want 2", result.FavoriteSlotID)
	}

	// Boundary case: an already empty slot is still a committed mutation, so it
	// carries a complete receipt of its own rather than a partial one.
	empty, err := DeleteFavoritePreset(engine, session.SaveSessionID, 7, "1")
	if err != nil {
		t.Fatalf("DeleteFavoritePreset on an inactive slot: %v", err)
	}
	assertMutationReceipt(t, empty.MutationReceipt, session.SaveSessionID,
		DeleteFavoritePresetEndpointID, "2")
	assertChangedScopes(t, empty.ChangedScopes, favoritesScopes)
	if empty.OperationID == result.OperationID {
		t.Errorf("two executions shared operationID %q", result.OperationID)
	}
}

// ApplyFavoritePreset writes appearance through a character-scoped commit, so
// its kind and its scopes must be its own and never those of the two preset-slot
// mutations that share the Favorites domain.
func TestApplyFavoritePresetResultCarriesItsCommitReceipt(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(
		writeEndpointApplyFavoritesFixture(t, 0, 3), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := ApplyFavoritePreset(engine, session.SaveSessionID, 0, 3, "0")
	if err != nil {
		t.Fatalf("ApplyFavoritePreset: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, session.SaveSessionID,
		ApplyFavoritePresetEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, favoritesAppearanceScopes)
	assertFlatReceiptJSON(t, result, []string{"characterID", "favoriteSlotID"})
	if result.CharacterID != 0 || result.FavoriteSlotID != 3 {
		t.Errorf("result = %+v, want the domain fields preserved", result)
	}
	if result.OperationKind == SetFavoritePresetEndpointID ||
		result.OperationKind == DeleteFavoritePresetEndpointID {
		t.Errorf("operationKind = %q, want the applying endpoint's own kind",
			result.OperationKind)
	}
}

// A rejected mutation returns the complete zero result: no operationID, no
// partial receipt and no partially filled domain field.
func TestRejectedFavoritesMutationsExposeNoPartialResult(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(
		writeEndpointApplyFavoritesFixture(t, 0, 3), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// A stale expectedRevision is the rejection all three mutations share.
	applied, err := ApplyFavoritePreset(engine, session.SaveSessionID, 0, 3, "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", applied)
	}
	if !reflect.DeepEqual(applied, ApplyFavoritePresetResult{}) {
		t.Errorf("rejected ApplyFavoritePreset = %+v, want the complete zero result", applied)
	}

	stored, err := SetFavoritePreset(engine, session.SaveSessionID, 4, 0, "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", stored)
	}
	if !reflect.DeepEqual(stored, SetFavoritePresetResult{}) {
		t.Errorf("rejected SetFavoritePreset = %+v, want the complete zero result", stored)
	}

	deleted, err := DeleteFavoritePreset(engine, session.SaveSessionID, 2, "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", deleted)
	}
	if !reflect.DeepEqual(deleted, DeleteFavoritePresetResult{}) {
		t.Errorf("rejected DeleteFavoritePreset = %+v, want the complete zero result", deleted)
	}

	// Adjacent negative case: an out-of-range slot is rejected by validation
	// before the commit path, which is the other way a receipt could leak.
	outOfRange, err := ApplyFavoritePreset(engine, session.SaveSessionID, 0, 15, "0")
	if err == nil {
		t.Fatalf("an out-of-range favoriteSlotID was accepted: %+v", outOfRange)
	}
	if !reflect.DeepEqual(outOfRange, ApplyFavoritePresetResult{}) {
		t.Errorf("rejected ApplyFavoritePreset = %+v, want the complete zero result", outOfRange)
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
