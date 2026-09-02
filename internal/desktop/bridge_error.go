package desktop

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// This file is the error boundary of the desktop bridge.
//
// Wails 2.15 carries exactly one thing back from a failing bound method: the
// string of the returned error. internal/frontend/dispatcher/calls.go assigns
// `callbackMessage.Err = err.Error()`, and the desktop runtime turns that
// string into `new Error(message)` before rejecting the call. There is no
// structured error channel to opt into, so a structured contract has to be
// carried inside that one string.
//
// The envelope below is therefore deliberate and explicit rather than a guess
// about undocumented serialization: every bridge failure is one line consisting
// of a fixed marker and the JSON of the shared public error model. The frontend
// adapter parses exactly that shape, validates it, and reduces anything else —
// a runtime failure, a Wails timeout, a truncated payload — to its safe
// bridge_call_failed code.
//
// Domain results keep their generated types; only the error side is encoded.

// BridgeErrorPrefix marks a bridge error string that carries the public error
// model as JSON. The frontend matches this exact prefix, so it is a shared
// constant and never a literal on either side.
const BridgeErrorPrefix = "saveforge-error:"

// bridgeError converts one endpoint failure into the transportable envelope.
// The private cause is logged here with the same diagnosticID the frontend
// receives and never becomes part of the string that leaves the process.
func bridgeError(err error) error {
	if err == nil {
		return nil
	}
	public := apperror.From(err)
	if cause := public.Cause(); cause != nil {
		log.Printf("%s %s: %v", public.DiagnosticID, public.Code, cause)
	} else {
		log.Printf("%s %s", public.DiagnosticID, public.Code)
	}
	encoded, marshalErr := json.Marshal(public)
	if marshalErr != nil {
		// Fail closed: an envelope that cannot be built carries the marker and no
		// payload, which the frontend reduces to bridge_call_failed. It never
		// falls back to the raw error text.
		log.Printf("%s: cannot encode the bridge error envelope: %v",
			public.DiagnosticID, marshalErr)
		return errors.New(BridgeErrorPrefix)
	}
	return errors.New(BridgeErrorPrefix + string(encoded))
}

// bridged wraps the result of one endpoint call for the bridge: a success is
// returned unchanged, and a failure is replaced by the envelope together with
// the zero result, so a partial value can never travel beside an error.
//
// ponytail: one generic wrapper instead of an if in every bound method. It is
// the whole error boundary; there is no interceptor chain and no middleware.
func bridged[T any](value T, err error) (T, error) {
	if err != nil {
		var zero T
		return zero, bridgeError(err)
	}
	return value, nil
}

// DecodeBridgeError parses an envelope produced by bridgeError. It exists for
// the bridge's own tests and mirrors exactly what the frontend adapter does: an
// unmarked, truncated or malformed string is not an application error.
func DecodeBridgeError(message string) (*apperror.Error, bool) {
	payload, marked := strings.CutPrefix(message, BridgeErrorPrefix)
	if !marked || payload == "" {
		return nil, false
	}
	var required struct {
		Code         string `json:"code"`
		Message      string `json:"message"`
		Severity     string `json:"severity"`
		Stage        string `json:"stage"`
		Retryable    *bool  `json:"retryable"`
		DiagnosticID string `json:"diagnosticID"`
	}
	if err := json.Unmarshal([]byte(payload), &required); err != nil {
		return nil, false
	}
	if required.Code == "" || required.Message == "" || required.Severity == "" ||
		required.Stage == "" || required.Retryable == nil || required.DiagnosticID == "" {
		return nil, false
	}
	var public apperror.Error
	if err := json.Unmarshal([]byte(payload), &public); err != nil {
		return nil, false
	}
	return &public, true
}
