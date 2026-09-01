package savesession

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetSaveAccountIDDelegatesToSaveEngine(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(writePCFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := SetSaveAccountID(engine, session.SaveSessionID, "1311768467463790320", "0")
	if err != nil {
		t.Fatalf("SetSaveAccountID: %v", err)
	}
	assertMutationReceipt(t, result.MutationReceipt, session.SaveSessionID,
		SetSaveAccountIDEndpointID, "1")
}

func TestSetSaveAccountIDRejectsMissingEngineAndForwardsErrors(t *testing.T) {
	if result, err := SetSaveAccountID(nil, "session", "1", "0"); err == nil ||
		err.Error() != "save engine is not available" || !isZeroReceipt(result.MutationReceipt) {
		t.Fatalf("nil-engine result = %+v, error = %v", result, err)
	}

	engine := saveengine.New()
	session, err := engine.LoadSave(writePS4Fixture(t), "ps4", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	result, err := SetSaveAccountID(engine, session.SaveSessionID, "12345", "0")
	if err == nil || !strings.Contains(err.Error(), "PC saves only") {
		t.Fatalf("forwarded PS4 error = %v", err)
	}
	if !isZeroReceipt(result.MutationReceipt) {
		t.Errorf("rejected result = %+v, want zero value", result)
	}
}

func TestSetSaveAccountIDContractMatchesRuntime(t *testing.T) {
	if SetSaveAccountIDDefinition.SupportedResourceTypes != "—" {
		t.Errorf("SupportedResourceTypes = %q, want no catalog resource dependency",
			SetSaveAccountIDDefinition.SupportedResourceTypes)
	}
	wantVariables := []string{"saveSessionID", "accountID", "expectedRevision"}
	if !reflect.DeepEqual(SetSaveAccountIDDefinition.SupportedResourceVariables, wantVariables) {
		t.Errorf("SupportedResourceVariables = %v, want %v",
			SetSaveAccountIDDefinition.SupportedResourceVariables, wantVariables)
	}
}
