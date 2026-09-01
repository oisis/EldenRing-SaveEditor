package network

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Stage 3b.1 embeds the shared MutationReceipt in the public results of the two
// Network mutations. Both run through one SaveEngine writer, so the interesting
// invariant is that they still report their own operationKind.

// networkScopes are the changed scopes of both Network mutations: the session
// baseline plus the one getter they invalidate.
var networkScopes = []string{"save.session", "network", "diagnostics.report"}

func TestNetworkMutationsCarryTheirCommitReceiptWithDistinctKinds(t *testing.T) {
	gameCatalog := newCatalog(t)
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSettingsFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	settings := validSettingsForSet()

	set, err := SetNetworkSettings(engine, loaded.SaveSessionID, settings, "0")
	if err != nil {
		t.Fatalf("SetNetworkSettings: %v", err)
	}
	assertMutationReceipt(t, set.MutationReceipt, loaded.SaveSessionID,
		SetNetworkSettingsEndpointID, "1")
	assertChangedScopes(t, set.ChangedScopes, networkScopes)
	if set.NetworkSettings != settings {
		t.Errorf("networkSettings = %+v, want the committed set", set.NetworkSettings)
	}
	assertFlatReceiptJSON(t, set, []string{"networkSettings"})

	applied, err := ApplyNetworkPreset(engine, gameCatalog, loaded.SaveSessionID, "faster-reds", "1")
	if err != nil {
		t.Fatalf("ApplyNetworkPreset: %v", err)
	}
	assertMutationReceipt(t, applied.MutationReceipt, loaded.SaveSessionID,
		ApplyNetworkPresetEndpointID, "2")
	assertChangedScopes(t, applied.ChangedScopes, networkScopes)
	if applied.PresetID != "faster-reds" {
		t.Errorf("presetID = %q, want the resolved preset", applied.PresetID)
	}
	presets, err := GetNetworkPresets(gameCatalog, "faster-reds")
	if err != nil {
		t.Fatalf("GetNetworkPresets: %v", err)
	}
	if applied.NetworkSettings != presets.Presets[0].Parameters {
		t.Errorf("networkSettings = %+v, want the complete preset", applied.NetworkSettings)
	}
	assertFlatReceiptJSON(t, applied, []string{"presetID", "networkSettings"})

	// The shared writer must never report the plain settings kind for a preset.
	if applied.OperationKind == set.OperationKind {
		t.Fatalf("both mutations reported operationKind %q", applied.OperationKind)
	}
	if applied.OperationID == set.OperationID {
		t.Fatalf("two executions shared operationID %q", set.OperationID)
	}
}

func TestTwoExecutionsOfOneNetworkMutationGetDifferentOperationIDs(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSettingsFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	settings := validSettingsForSet()

	first, err := SetNetworkSettings(engine, loaded.SaveSessionID, settings, "0")
	if err != nil {
		t.Fatalf("first SetNetworkSettings: %v", err)
	}
	second, err := SetNetworkSettings(engine, loaded.SaveSessionID, settings, "1")
	if err != nil {
		t.Fatalf("second SetNetworkSettings: %v", err)
	}
	if first.OperationID == second.OperationID {
		t.Fatalf("two executions shared operationID %q", first.OperationID)
	}
	if first.OperationKind != second.OperationKind {
		t.Fatalf("operationKinds = %q and %q, want one stable kind",
			first.OperationKind, second.OperationKind)
	}
}

// A rejected mutation returns the exact zero result: no operationID of an
// execution that never happened, and no domain field either.
func TestRejectedNetworkMutationsExposeNoOperationID(t *testing.T) {
	engine := saveengine.New()
	gameCatalog := newCatalog(t)
	loaded, err := engine.LoadSave(writeSettingsFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	for _, test := range []struct {
		name string
		call func() (SetNetworkSettingsResult, error)
	}{
		{
			name: "stale expectedRevision",
			call: func() (SetNetworkSettingsResult, error) {
				return SetNetworkSettings(engine, loaded.SaveSessionID, validSettingsForSet(), "7")
			},
		},
		{
			name: "missing engine",
			call: func() (SetNetworkSettingsResult, error) {
				return SetNetworkSettings(nil, loaded.SaveSessionID, validSettingsForSet(), "0")
			},
		},
	} {
		t.Run("SetNetworkSettings/"+test.name, func(t *testing.T) {
			result, err := test.call()
			if err == nil {
				t.Fatalf("the rejected call was accepted: %+v", result)
			}
			if !reflect.DeepEqual(result, SetNetworkSettingsResult{}) {
				t.Errorf("result = %+v, want the complete zero result", result)
			}
		})
	}

	for _, test := range []struct {
		name string
		call func() (ApplyNetworkPresetResult, error)
	}{
		{
			name: "stale expectedRevision",
			call: func() (ApplyNetworkPresetResult, error) {
				return ApplyNetworkPreset(engine, gameCatalog, loaded.SaveSessionID, "vanilla", "7")
			},
		},
		{
			name: "unknown preset",
			call: func() (ApplyNetworkPresetResult, error) {
				return ApplyNetworkPreset(engine, gameCatalog, loaded.SaveSessionID, "unknown", "0")
			},
		},
		{
			name: "missing engine",
			call: func() (ApplyNetworkPresetResult, error) {
				return ApplyNetworkPreset(nil, gameCatalog, loaded.SaveSessionID, "vanilla", "0")
			},
		},
		{
			name: "missing catalog",
			call: func() (ApplyNetworkPresetResult, error) {
				return ApplyNetworkPreset(engine, nil, loaded.SaveSessionID, "vanilla", "0")
			},
		},
	} {
		t.Run("ApplyNetworkPreset/"+test.name, func(t *testing.T) {
			result, err := test.call()
			if err == nil {
				t.Fatalf("the rejected call was accepted: %+v", result)
			}
			if !reflect.DeepEqual(result, ApplyNetworkPresetResult{}) {
				t.Errorf("result = %+v, want the complete zero result", result)
			}
		})
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
