package appearance

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Stage 3b.2 embeds the shared MutationReceipt in the public result of
// ApplyAppearancePreset. The receipt is not reassembled here: it is exactly the
// one the central commit path produced.
//
// The scope list is written out literally instead of being read back from
// saveengine.ChangedScopesForMutationKind, so it states the reviewed public
// contract of the endpoint rather than echoing the implementation table.
var appearancePresetScopes = []string{
	"save.session", "character.profile", "character.appearance", "diagnostics.report",
}

func TestApplyAppearancePresetResultCarriesItsCommitReceipt(t *testing.T) {
	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}
	engine := saveengine.New()
	session, err := engine.LoadSave(writeApplyAppearanceFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	const presetID = "geralt-of-rivia-the-witcher"
	result, err := ApplyAppearancePreset(engine, gameCatalog, session.SaveSessionID, 0, presetID, "0")
	if err != nil {
		t.Fatalf("ApplyAppearancePreset: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, session.SaveSessionID,
		ApplyAppearancePresetEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, appearancePresetScopes)
	assertFlatReceiptJSON(t, result, []string{"characterID", "presetID", "appearance"})
	if result.CharacterID != 0 || result.PresetID != presetID {
		t.Errorf("result = %+v, want the domain fields preserved", result)
	}

	// The endpoint shares SaveEngine's single private appearance writer with
	// SetCharacterAppearance and SetCharacterGender. The writer receives its kind
	// from the public entry point, so the preset can never report one of theirs.
	if result.OperationKind != ApplyAppearancePresetEndpointID {
		t.Fatalf("operationKind = %q, want the calling endpoint's own kind",
			result.OperationKind)
	}
	if result.OperationKind == "set_character_appearance" ||
		result.OperationKind == "set_character_gender" {
		t.Fatalf("the shared writer leaked the kind %q of an inner setter",
			result.OperationKind)
	}

	// An adjacent execution of the same endpoint gets its own identifier and the
	// next revision, so the receipt describes one execution and not the endpoint.
	second, err := ApplyAppearancePreset(
		engine, gameCatalog, session.SaveSessionID, 0,
		"ciri-the-princess-of-cintra-from-witcher", "1")
	if err != nil {
		t.Fatalf("second ApplyAppearancePreset: %v", err)
	}
	assertMutationReceipt(t, second.MutationReceipt, session.SaveSessionID,
		ApplyAppearancePresetEndpointID, "2")
	if second.OperationID == result.OperationID {
		t.Errorf("two executions shared operationID %q", result.OperationID)
	}
}

// A rejected mutation returns the complete zero result: no operationID, no
// partial receipt and no partially filled domain field.
func TestRejectedApplyAppearancePresetExposesNoPartialResult(t *testing.T) {
	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("NewPrototype: %v", err)
	}
	engine := saveengine.New()
	session, err := engine.LoadSave(writeApplyAppearanceFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	stale, err := ApplyAppearancePreset(
		engine, gameCatalog, session.SaveSessionID, 0, "geralt-of-rivia-the-witcher", "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", stale)
	}
	if !reflect.DeepEqual(stale, ApplyAppearancePresetResult{}) {
		t.Errorf("rejected result = %+v, want the complete zero result", stale)
	}

	// Adjacent negative case: an unknown preset is refused by catalog resolution
	// before the commit path, which is the other way a receipt could leak.
	unknown, err := ApplyAppearancePreset(
		engine, gameCatalog, session.SaveSessionID, 0, "no-such-preset", "0")
	if err == nil {
		t.Fatalf("an unknown presetID was accepted: %+v", unknown)
	}
	if !reflect.DeepEqual(unknown, ApplyAppearancePresetResult{}) {
		t.Errorf("rejected result = %+v, want the complete zero result", unknown)
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
