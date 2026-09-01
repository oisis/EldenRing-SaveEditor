package templates_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Stage 3b.2 embeds the shared MutationReceipt in the public result of
// ApplyBuildTemplate. The receipt is not reassembled here: it is exactly the one
// the central commit path produced, and ApplyBuildTemplate reaches that path
// through a lower SaveEngine writer that must still report the public kind.
//
// The scope list is written out literally instead of being read back from
// saveengine.ChangedScopesForMutationKind, so it states the reviewed public
// contract of the endpoint rather than echoing the implementation table.
var buildTemplateScopes = []string{
	"save.session", "character.list", "character.profile", "character.stats",
	"equipment.loadout", "diagnostics.report",
}

// applyReceiptTemplate is a minimal statistics-only template: it changes vigor
// and strength and the level SaveEngine recalculates from them.
func applyReceiptTemplate(t *testing.T, store *buildtemplates.Store) (string, string) {
	t.Helper()

	vigor := uint32(60)
	strength := uint32(40)
	return createTestTemplate(t, store, &buildtemplates.BuildTemplate{
		Schema:   buildtemplates.SchemaKey,
		Version:  buildtemplates.MaxSchemaVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{Name: "Receipt Build"},
		Selection: &buildtemplates.TemplateSelection{
			Stats: &buildtemplates.SectionSelection{
				Fields: map[string]bool{"vigor": true, "strength": true},
			},
		},
		Sections: buildtemplates.TemplateSections{
			Stats: &buildtemplates.StatsSection{Vigor: &vigor, Strength: &strength},
		},
	})
}

func TestApplyBuildTemplateResultCarriesItsCommitReceipt(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	store := buildtemplates.NewStore(t.TempDir())
	templateID, templateRevision := applyReceiptTemplate(t, store)

	result, err := templates.ApplyBuildTemplate(
		store, engine, newTestCatalog(t), templates.ApplyBuildTemplateRequest{
			SaveSessionID:    loaded.SaveSessionID,
			CharacterID:      testSlotActive,
			TemplateID:       templateID,
			ExpectedRevision: "0",
		})
	if err != nil {
		t.Fatalf("ApplyBuildTemplate: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, loaded.SaveSessionID,
		templates.ApplyBuildTemplateEndpointID, "1")
	assertChangedScopes(t, result.ChangedScopes, buildTemplateScopes)
	assertFlatReceiptJSON(t, result,
		[]string{"templateID", "templateRevision", "characterID", "plan"})

	// The template store revision and the save revision are two different
	// identifiers and must not be confused by the embedding.
	if result.TemplateID != templateID || result.TemplateRevision != templateRevision {
		t.Errorf("result = %+v, want templateID %q at templateRevision %q",
			result, templateID, templateRevision)
	}
	if result.CharacterID != testSlotActive {
		t.Errorf("characterID = %d, want %d", result.CharacterID, testSlotActive)
	}
	if result.Plan.Stats == nil {
		t.Error("plan.stats is nil, want the applied statistics plan preserved")
	}

	// ApplyBuildTemplate commits through a lower SaveEngine character writer. The
	// receipt must keep the kind of the public endpoint that was called.
	if result.OperationKind != "apply_build_template" {
		t.Fatalf("operationKind = %q, want the public endpoint kind", result.OperationKind)
	}
	// The undo point the same commit recorded names the same kind, so the two
	// public surfaces of one mutation cannot disagree.
	undo, err := engine.GetUndoState(loaded.SaveSessionID, testSlotActive)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undo.OperationKind != result.OperationKind {
		t.Errorf("undo operationKind = %q, want the receipt kind %q",
			undo.OperationKind, result.OperationKind)
	}
}

// A rejected apply returns the complete zero result: no operationID, no partial
// receipt and no partially filled domain field.
func TestRejectedApplyBuildTemplateExposesNoPartialResult(t *testing.T) {
	engine := saveengine.New()
	savePath, _ := writeTestSaveFixture(t)
	loaded, err := engine.LoadSave(savePath, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	store := buildtemplates.NewStore(t.TempDir())
	templateID, _ := applyReceiptTemplate(t, store)
	catalog := newTestCatalog(t)

	stale, err := templates.ApplyBuildTemplate(
		store, engine, catalog, templates.ApplyBuildTemplateRequest{
			SaveSessionID:    loaded.SaveSessionID,
			CharacterID:      testSlotActive,
			TemplateID:       templateID,
			ExpectedRevision: "7",
		})
	if err == nil {
		t.Fatalf("a stale revision was accepted: %+v", stale)
	}
	if !reflect.DeepEqual(stale, templates.ApplyBuildTemplateResult{}) {
		t.Errorf("rejected result = %+v, want the complete zero result", stale)
	}

	// Adjacent negative case: a non-canonical expectedRevision is refused before
	// the plan is even resolved, which is the other way a receipt could leak.
	malformed, err := templates.ApplyBuildTemplate(
		store, engine, catalog, templates.ApplyBuildTemplateRequest{
			SaveSessionID:    loaded.SaveSessionID,
			CharacterID:      testSlotActive,
			TemplateID:       templateID,
			ExpectedRevision: "01",
		})
	if err == nil {
		t.Fatalf("a non-canonical revision was accepted: %+v", malformed)
	}
	if !reflect.DeepEqual(malformed, templates.ApplyBuildTemplateResult{}) {
		t.Errorf("rejected result = %+v, want the complete zero result", malformed)
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
