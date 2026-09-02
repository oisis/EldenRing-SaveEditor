package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// Wails 2 carries only the string of a returned error, so the bridge encodes
// the shared error model into that one string. These tests own that envelope:
// what it must contain, and what it must never contain.

func TestBridgeErrorCarriesTheClassificationOfATypedFailure(t *testing.T) {
	encoded := bridgeError(apperror.RevisionConflict("1", "4"))

	public, decoded := DecodeBridgeError(encoded.Error())
	if !decoded {
		t.Fatalf("envelope %q could not be decoded", encoded.Error())
	}
	if public.Code != apperror.CodeRevisionConflict {
		t.Errorf("code = %q, want %q", public.Code, apperror.CodeRevisionConflict)
	}
	if public.CurrentRevision != "4" {
		t.Errorf("currentRevision = %q, want the session's current revision",
			public.CurrentRevision)
	}
	if public.Retryable {
		t.Error("a revision conflict must not be reported as retryable")
	}
}

func TestBridgeErrorLogsEveryReportedDiagnosticID(t *testing.T) {
	var output bytes.Buffer
	previousWriter := log.Writer()
	previousFlags := log.Flags()
	log.SetOutput(&output)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
	})

	failure := apperror.RevisionConflict("1", "4")
	bridgeError(failure)

	line := output.String()
	if !strings.Contains(line, failure.DiagnosticID) ||
		!strings.Contains(line, apperror.CodeRevisionConflict) {
		t.Fatalf("log = %q, want diagnosticID %q and code %q",
			line, failure.DiagnosticID, apperror.CodeRevisionConflict)
	}
}

func TestBridgeErrorCarriesFieldErrors(t *testing.T) {
	encoded := bridgeError(apperror.MissingField("saveSessionID"))

	public, decoded := DecodeBridgeError(encoded.Error())
	if !decoded {
		t.Fatalf("envelope %q could not be decoded", encoded.Error())
	}
	if public.Code != apperror.CodeInvalidRequest {
		t.Errorf("code = %q, want %q", public.Code, apperror.CodeInvalidRequest)
	}
	if len(public.FieldErrors) != 1 || public.FieldErrors[0].Field != "saveSessionID" {
		t.Errorf("fieldErrors = %+v, want exactly the missing field", public.FieldErrors)
	}
}

// The envelope is the boundary that must not leak. An unclassified failure
// carrying a private path crosses it as a stable code and a fixed sentence.
func TestBridgeErrorNeverCarriesTheRawCause(t *testing.T) {
	cause := fmt.Errorf(
		"cannot open /Users/someone/Documents/ER0000.sl2: %w", errors.New("permission denied"))

	message := bridgeError(cause).Error()

	for _, secret := range []string{"/Users/someone", "permission denied", "cannot open"} {
		if strings.Contains(message, secret) {
			t.Errorf("envelope leaks %q: %s", secret, message)
		}
	}
	public, decoded := DecodeBridgeError(message)
	if !decoded || public.Code != apperror.CodeOperationFailed {
		t.Fatalf("envelope = %s, want a decodable operation_failed failure", message)
	}
	if public.DiagnosticID == "" {
		t.Error("the envelope carries no diagnosticID to correlate the backend log with")
	}
}

func TestBridgeErrorIsNilForSuccess(t *testing.T) {
	if err := bridgeError(nil); err != nil {
		t.Errorf("bridgeError(nil) = %v, want nil", err)
	}
}

// Everything that is not a well-formed envelope must be rejected rather than
// half-read: the frontend's own fallback is what such a rejection turns into.
func TestDecodeBridgeErrorRejectsAnythingButACompleteEnvelope(t *testing.T) {
	valid, err := json.Marshal(apperror.UnknownSaveSession("abc"))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	for name, message := range map[string]string{
		"unmarked":       string(valid),
		"marker only":    BridgeErrorPrefix,
		"not json":       BridgeErrorPrefix + "{oops",
		"array":          BridgeErrorPrefix + `["code"]`,
		"missing code":   BridgeErrorPrefix + `{"message":"x","severity":"error","stage":"request"}`,
		"missing text":   BridgeErrorPrefix + `{"code":"x","severity":"error","stage":"request"}`,
		"missing retry":  BridgeErrorPrefix + `{"code":"x","message":"x","severity":"error","stage":"request","diagnosticID":"diag-1"}`,
		"missing diag":   BridgeErrorPrefix + `{"code":"x","message":"x","severity":"error","stage":"request","retryable":false}`,
		"empty document": BridgeErrorPrefix + `{}`,
	} {
		if _, decoded := DecodeBridgeError(message); decoded {
			t.Errorf("%s: %q was accepted as an envelope", name, message)
		}
	}

	if _, decoded := DecodeBridgeError(BridgeErrorPrefix + string(valid)); !decoded {
		t.Error("a complete envelope was rejected")
	}
}

// The bridge is the only place a SaveEngine event becomes a host emission, and
// it must emit exactly the published event under the backend's own name.
func TestBridgePublishesSessionChangedToTheHost(t *testing.T) {
	bridge := NewBridge("2.0.0", saveengine.New(), nil, nil)
	type emission struct {
		name string
		data []any
	}
	var emitted []emission
	bridge.emitEvent = func(_ context.Context, name string, data ...any) {
		emitted = append(emitted, emission{name: name, data: data})
	}

	event := saveengine.SessionChangedEvent{
		Sequence:      "1",
		OperationID:   "op-1",
		OperationKind: "set_character_name",
		SaveSessionID: "session",
		SaveRevision:  "1",
		ChangedScopes: []string{"save.session"},
	}

	// Before Startup there is no host to emit to, and the event is dropped
	// rather than buffered: the frontend resynchronises when its listener starts.
	bridge.publishSessionChanged(event)
	if len(emitted) != 0 {
		t.Fatalf("emitted %+v before the host started, want nothing", emitted)
	}

	bridge.Startup(context.Background())
	bridge.publishSessionChanged(event)
	if len(emitted) != 1 {
		t.Fatalf("emitted %d events, want exactly one", len(emitted))
	}
	if emitted[0].name != saveengine.SessionChangedEventName {
		t.Errorf("event name = %q, want %q",
			emitted[0].name, saveengine.SessionChangedEventName)
	}
	if len(emitted[0].data) != 1 || !reflectEqualEvent(emitted[0].data[0], event) {
		t.Errorf("payload = %+v, want exactly the published event", emitted[0].data)
	}
}

// NewBridge subscribes the engine, so a committed mutation reaches the host
// without the composition root wiring a second path.
func TestNewBridgeSubscribesTheEngine(t *testing.T) {
	engine := saveengine.New()
	bridge := NewBridge("2.0.0", engine, nil, nil)
	bridge.Startup(context.Background())
	var names []string
	bridge.emitEvent = func(_ context.Context, name string, _ ...any) {
		names = append(names, name)
	}

	engine.SetSessionChangedSink(nil)
	engine.SetSessionChangedSink(bridge.publishSessionChanged)
	bridge.publishSessionChanged(saveengine.SessionChangedEvent{Sequence: "1"})

	if len(names) != 1 || names[0] != saveengine.SessionChangedEventName {
		t.Errorf("emitted %v, want one %q", names, saveengine.SessionChangedEventName)
	}
}

func reflectEqualEvent(value any, want saveengine.SessionChangedEvent) bool {
	event, ok := value.(saveengine.SessionChangedEvent)
	if !ok {
		return false
	}
	if event.Sequence != want.Sequence || event.OperationID != want.OperationID ||
		event.OperationKind != want.OperationKind || event.SaveSessionID != want.SaveSessionID ||
		event.SaveRevision != want.SaveRevision ||
		len(event.ChangedScopes) != len(want.ChangedScopes) {
		return false
	}
	for index, scope := range want.ChangedScopes {
		if event.ChangedScopes[index] != scope {
			return false
		}
	}
	return true
}
