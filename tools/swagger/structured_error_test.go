package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// The local explorer answers every failure in the shared error model and
// derives the status from the classification, so no route restates the rule.

func TestStructuredErrorStatusesComeFromTheClassification(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	base := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/"

	tests := map[string]struct {
		target     string
		body       string
		wantStatus int
		wantCode   string
	}{
		"malformed revision": {
			target:     base + "active",
			body:       `{"active":false,"expectedRevision":"00"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   apperror.CodeInvalidRevision,
		},
		"stale revision": {
			target:     base + "active",
			body:       `{"active":false,"expectedRevision":"9"}`,
			wantStatus: http.StatusConflict,
			wantCode:   apperror.CodeRevisionConflict,
		},
		"unknown session": {
			target:     "/api/v1/save-sessions/nope/characters/0/active",
			body:       `{"active":false,"expectedRevision":"0"}`,
			wantStatus: http.StatusNotFound,
			wantCode:   apperror.CodeUnknownSaveSession,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			recorder := doSave(t, saveEngine, http.MethodPatch, test.target, test.body)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d (body %q)",
					recorder.Code, test.wantStatus, recorder.Body.String())
			}
			reported := decodeError(t, recorder)
			if reported.Code != test.wantCode {
				t.Fatalf("code = %q, want %q", reported.Code, test.wantCode)
			}
			if reported.Severity != apperror.SeverityError {
				t.Errorf("severity = %q, want %q", reported.Severity, apperror.SeverityError)
			}
		})
	}
}

// A revision conflict must hand the caller the revision to reconcile against,
// and must never present itself as retryable.
func TestRevisionConflictReportsTheCurrentRevision(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/active"

	// One accepted mutation moves the session to revision 1.
	accepted := doSave(t, saveEngine, http.MethodPatch, target,
		`{"active":false,"expectedRevision":"0"}`)
	assertOK(t, accepted, target)

	conflicted := doSave(t, saveEngine, http.MethodPatch, target,
		`{"active":true,"expectedRevision":"0"}`)
	if conflicted.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body %q)", conflicted.Code, conflicted.Body.String())
	}
	reported := decodeError(t, conflicted)
	if reported.CurrentRevision != "1" {
		t.Errorf("currentRevision = %q, want the session's current revision %q",
			reported.CurrentRevision, "1")
	}
	if reported.Retryable {
		t.Error("a revision conflict must never be reported as retryable")
	}
	if reported.Params["expectedRevision"] != "0" {
		t.Errorf("params = %v, want the rejected expectation", reported.Params)
	}
}

// A missing required request value is reported as a named field failure.
func TestMissingSessionIsReportedAsAFieldError(t *testing.T) {
	saveEngine := saveengine.New()
	if _, err := savesession.LoadSave(
		saveEngine, writeActiveSpellsFixture(t), "", "local"); err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	recorder := doSave(t, saveEngine, http.MethodGet, "/api/v1/save-sessions/%20", "")
	if recorder.Code == http.StatusOK {
		t.Fatalf("an empty session identifier was accepted: %q", recorder.Body.String())
	}
	reported := decodeError(t, recorder)
	if reported.Code != apperror.CodeInvalidRequest &&
		reported.Code != apperror.CodeUnknownSaveSession {
		t.Fatalf("code = %q, want a request or session classification", reported.Code)
	}
	if reported.Code == apperror.CodeInvalidRequest && len(reported.FieldErrors) == 0 {
		t.Errorf("invalid_request without fieldErrors: %+v", reported)
	}
}

// No response may carry a raw backend sentence for an unclassified failure.
func TestUnclassifiedFailuresDoNotLeakTheirMessage(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	// Slot 99 is outside the supported range; the endpoint rejects it with its
	// own sentence, which the transport must not repeat.
	recorder := doSave(t, saveEngine, http.MethodPatch,
		"/api/v1/save-sessions/"+session.SaveSessionID+"/characters/99/active",
		`{"active":false,"expectedRevision":"0"}`)
	if recorder.Code == http.StatusOK {
		t.Fatalf("an out-of-range slot was accepted: %q", recorder.Body.String())
	}
	reported := decodeError(t, recorder)
	if reported.Code != apperror.CodeOperationFailed {
		t.Fatalf("code = %q, want %q", reported.Code, apperror.CodeOperationFailed)
	}
	body := recorder.Body.String()
	for _, leak := range []string{"outside the range", "characterID 99", "goroutine"} {
		if strings.Contains(body, leak) {
			t.Errorf("the response leaks %q: %s", leak, body)
		}
	}
}

func TestUnexpectedServerFailureUsesInternalError(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, http.StatusInternalServerError,
		apperror.Internal(errors.New("private server failure")))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	reported := decodeError(t, recorder)
	if reported.Code != apperror.CodeInternalError {
		t.Fatalf("code = %q, want %q", reported.Code, apperror.CodeInternalError)
	}
	if strings.Contains(recorder.Body.String(), "private server failure") {
		t.Fatalf("response leaked the private cause: %s", recorder.Body.String())
	}
}

// Both success variants of SetCharacterActive travel over the transport
// unchanged: the committed one with the complete receipt, the idempotent one
// without any of the three execution members.
func TestSetCharacterActiveRouteReportsBothSuccessVariants(t *testing.T) {
	saveEngine := saveengine.New()
	session, err := savesession.LoadSave(saveEngine, writeActiveSpellsFixture(t), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	target := "/api/v1/save-sessions/" + session.SaveSessionID + "/characters/0/active"

	// Slot 0 is active in the fixture, so requesting active changes nothing.
	idempotent := doSave(t, saveEngine, http.MethodPatch, target,
		`{"active":true,"expectedRevision":"0"}`)
	assertOK(t, idempotent, target)
	var unchanged character.SetCharacterActiveResult
	if err := json.Unmarshal(idempotent.Body.Bytes(), &unchanged); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if unchanged.Changed || unchanged.SaveRevision != "0" {
		t.Errorf("result = %+v, want an unchanged request at revision 0", unchanged)
	}
	assertAbsentPayloadMembers(t, idempotent.Body.Bytes(),
		[]string{"operationID", "operationKind", "changedScopes", "receipt"})

	committed := doSave(t, saveEngine, http.MethodPatch, target,
		`{"active":false,"expectedRevision":"0"}`)
	assertOK(t, committed, target)
	var changed character.SetCharacterActiveResult
	if err := json.Unmarshal(committed.Body.Bytes(), &changed); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !changed.Changed {
		t.Fatalf("result = %+v, want a committed mutation", changed)
	}
	assertRouteReceipt(t, changed.MutationReceipt, session.SaveSessionID,
		character.SetCharacterActiveEndpointID, "1")
}

func assertAbsentPayloadMembers(t *testing.T, body []byte, absent []string) {
	t.Helper()

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	for _, key := range absent {
		if _, present := payload[key]; present {
			t.Errorf("payload carries %q, want it absent: %s", key, body)
		}
	}
}
