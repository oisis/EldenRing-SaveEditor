package savesession

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Stage 3b.1 embeds the shared MutationReceipt in the public results of the two
// SaveSession mutations. The receipt is not reassembled here: it is exactly the
// one the central commit path produced, so these tests assert the public shape
// and the values only that path can know.

// sessionScopes are the changed scopes of both SaveSession mutations. Neither
// reaches a domain getter, so both invalidate the universal scopes only.
var sessionScopes = []string{"save.session", "diagnostics.report"}

func TestSetSaveAccountIDResultCarriesItsCommitReceipt(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(writePCFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	first, err := SetSaveAccountID(engine, session.SaveSessionID, "0", "0")
	if err != nil {
		t.Fatalf("SetSaveAccountID: %v", err)
	}
	assertMutationReceipt(t, first.MutationReceipt, session.SaveSessionID,
		SetSaveAccountIDEndpointID, "1")
	assertChangedScopes(t, first.ChangedScopes, sessionScopes)
	assertFlatReceiptJSON(t, first, nil)

	second, err := SetSaveAccountID(engine, session.SaveSessionID, "0", "1")
	if err != nil {
		t.Fatalf("second SetSaveAccountID: %v", err)
	}
	assertMutationReceipt(t, second.MutationReceipt, session.SaveSessionID,
		SetSaveAccountIDEndpointID, "2")
	if second.OperationID == first.OperationID {
		t.Fatalf("two executions shared operationID %q", first.OperationID)
	}
}

func TestWriteSaveResultCarriesItsCommitReceipt(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(writePCFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	first, err := WriteSave(engine, session.SaveSessionID, "0",
		filepath.Join(t.TempDir(), "first.sl2"))
	if err != nil {
		t.Fatalf("WriteSave: %v", err)
	}
	assertMutationReceipt(t, first.MutationReceipt, session.SaveSessionID, WriteSaveEndpointID, "1")
	assertChangedScopes(t, first.ChangedScopes, sessionScopes)
	assertFlatReceiptJSON(t, first, nil)

	second, err := WriteSave(engine, session.SaveSessionID, "1",
		filepath.Join(t.TempDir(), "second.sl2"))
	if err != nil {
		t.Fatalf("second WriteSave: %v", err)
	}
	assertMutationReceipt(t, second.MutationReceipt, session.SaveSessionID, WriteSaveEndpointID, "2")
	if second.OperationID == first.OperationID {
		t.Fatalf("two executions shared operationID %q", first.OperationID)
	}
}

// A rejected mutation returns the zero result. It must expose neither the
// identifier of an execution that never happened nor a partial receipt.
func TestRejectedSaveSessionMutationsExposeNoOperationID(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(writePCFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	accountResult, err := SetSaveAccountID(engine, session.SaveSessionID, "0", "7")
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", accountResult)
	}
	if !isZeroReceipt(accountResult.MutationReceipt) {
		t.Errorf("rejected SetSaveAccountID receipt = %+v, want the zero receipt",
			accountResult.MutationReceipt)
	}

	writeResult, err := WriteSave(engine, session.SaveSessionID, "7",
		filepath.Join(t.TempDir(), "rejected.sl2"))
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", writeResult)
	}
	if !isZeroReceipt(writeResult.MutationReceipt) {
		t.Errorf("rejected WriteSave receipt = %+v, want the zero receipt", writeResult.MutationReceipt)
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

func isZeroReceipt(receipt saveengine.MutationReceipt) bool {
	return receipt.OperationID == "" && receipt.OperationKind == "" &&
		receipt.SaveSessionID == "" && receipt.SaveRevision == "" && receipt.ChangedScopes == nil
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
