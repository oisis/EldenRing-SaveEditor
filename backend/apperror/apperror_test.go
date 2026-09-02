package apperror

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestInvalidRevisionNamesTheFieldAndKeepsTheRequestStage(t *testing.T) {
	err := InvalidRevision("00")

	if err.Code != CodeInvalidRevision || err.Stage != StageRequest {
		t.Errorf("error = %+v, want an invalid_revision request failure", err)
	}
	if err.CurrentRevision != "" {
		t.Errorf("currentRevision = %q, want none: no session was consulted",
			err.CurrentRevision)
	}
	if len(err.FieldErrors) != 1 || err.FieldErrors[0].Field != "expectedRevision" {
		t.Errorf("fieldErrors = %+v, want exactly the expectedRevision field", err.FieldErrors)
	}
	if err.Params["value"] != "00" {
		t.Errorf("params = %v, want the rejected value", err.Params)
	}
}

func TestRevisionConflictCarriesTheCurrentRevisionAndIsNotRetryable(t *testing.T) {
	err := RevisionConflict("1", "4")

	if err.Code != CodeRevisionConflict || err.CurrentRevision != "4" {
		t.Errorf("error = %+v, want a conflict reporting the current revision", err)
	}
	if err.Retryable {
		t.Error("a revision conflict must never be reported as retryable")
	}
	if err.Params["expectedRevision"] != "1" || err.Params["currentRevision"] != "4" {
		t.Errorf("params = %v, want both revisions", err.Params)
	}
}

func TestUnknownSaveSessionIsClassifiedAsASessionFailure(t *testing.T) {
	err := UnknownSaveSession("abc")

	if err.Code != CodeUnknownSaveSession || err.Stage != StageSession {
		t.Errorf("error = %+v, want an unknown_save_session failure", err)
	}
	if err.Params["saveSessionID"] != "abc" {
		t.Errorf("params = %v, want the addressed session", err.Params)
	}
}

func TestMissingFieldReportsOneNamedRequestField(t *testing.T) {
	err := MissingField("saveSessionID")

	if err.Code != CodeInvalidRequest || err.Message != "saveSessionID is required" {
		t.Errorf("error = %+v, want the named required field", err)
	}
	if len(err.FieldErrors) != 1 || err.FieldErrors[0].Field != "saveSessionID" ||
		err.FieldErrors[0].Code != CodeInvalidRequest {
		t.Errorf("fieldErrors = %+v, want exactly the missing field", err.FieldErrors)
	}
}

// The whole point of the model: nothing a transport can serialize may contain
// the private cause. A path, a wrapped OS failure and a stack-like sentence all
// have to stay in the log.
func TestUnclassifiedFailuresNeverSerializeTheirCause(t *testing.T) {
	cause := fmt.Errorf(
		"cannot open /Users/someone/Documents/ER0000.sl2: %w", errors.New("permission denied"))

	for name, public := range map[string]*Error{
		"operation_failed": OperationFailed(cause),
		"internal_error":   Internal(cause),
	} {
		encoded, err := json.Marshal(public)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		for _, secret := range []string{"/Users/someone", "permission denied", "cannot open"} {
			if strings.Contains(string(encoded), secret) {
				t.Errorf("%s payload leaks %q: %s", name, secret, encoded)
			}
		}
		if public.Message != internalMessage {
			t.Errorf("%s message = %q, want the fixed safe sentence", name, public.Message)
		}
		if public.DiagnosticID == "" {
			t.Errorf("%s carries no diagnosticID to correlate the log entry with", name)
		}
		// The cause is still reachable for the internal log, and only there.
		if !errors.Is(public, cause) {
			t.Errorf("%s lost its cause, so the log cannot report it", name)
		}
	}
}

// Two failures never share a diagnostic identifier, and it is not derived from
// anything the caller supplied.
func TestDiagnosticIDsAreUnpredictableAndDistinct(t *testing.T) {
	first := RevisionConflict("1", "2")
	second := RevisionConflict("1", "2")

	if first.DiagnosticID == "" || second.DiagnosticID == "" {
		t.Fatal("a reported failure must carry a diagnosticID")
	}
	if first.DiagnosticID == second.DiagnosticID {
		t.Errorf("two failures shared diagnosticID %q", first.DiagnosticID)
	}
	if !strings.HasPrefix(first.DiagnosticID, diagnosticIDPrefix) {
		t.Errorf("diagnosticID = %q, want the %q marker", first.DiagnosticID, diagnosticIDPrefix)
	}
}

// From must never re-derive a classification a lower layer already made, and
// above all never derive one from the text of an error.
func TestFromKeepsAnExistingClassificationAndNormalizesEverythingElse(t *testing.T) {
	if From(nil) != nil {
		t.Error("From(nil) must stay nil so a caller can normalize unconditionally")
	}

	conflict := RevisionConflict("1", "2")
	if From(conflict) != conflict {
		t.Error("From replaced an already classified failure")
	}
	if From(fmt.Errorf("wrapped: %w", conflict)) != conflict {
		t.Error("From lost the classification of a wrapped failure")
	}

	// A message that reads like a known failure must not be classified as one.
	misleading := errors.New(`expectedRevision "1" does not match the current saveRevision "2"`)
	normalized := From(misleading)
	if normalized.Code != CodeOperationFailed {
		t.Errorf("code = %q, want %q: classification must never read the message",
			normalized.Code, CodeOperationFailed)
	}
	if normalized.CurrentRevision != "" {
		t.Errorf("currentRevision = %q, want none for an unclassified failure",
			normalized.CurrentRevision)
	}
}
