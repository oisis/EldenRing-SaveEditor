// Package apperror owns the one public error model of SaveForge 2.0.
//
// It is the single shape every transport reports a failure in: the HTTP
// explorer, the Wails bridge and the frontend all carry these members and
// nothing else. The model exists so a consumer can decide what to do from a
// stable code instead of from the wording of an English sentence: no layer of
// this application classifies a failure by matching text.
//
// Two rules make the model safe to hand to a user interface:
//
//   - a value of this type carries only what the backend vouches for. A raw Go
//     error, a wrapped OS failure, a private path or a stack trace never becomes
//     part of one; Internal replaces such a cause with a fixed English sentence.
//   - the private cause is still reachable for an internal log through Cause,
//     and the log line is correlated with the response through the same
//     DiagnosticID.
//
// The message is a fallback, not the user-facing text. The frontend owns the
// final localized wording and resolves it from Code and Params; Message is what
// it falls back to for a code it does not know.
package apperror

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// Stable error codes. They are a closed, public vocabulary: a new failure gets
// a new constant here, never a new sentence in an existing code.
const (
	// CodeInvalidRequest is a malformed or missing request value the endpoint
	// rejected before doing any work. It carries the offending fields in
	// FieldErrors when the endpoint named them.
	CodeInvalidRequest = "invalid_request"
	// CodeInvalidRevision is an expectedRevision that is not a canonical decimal
	// saveRevision. It is a request defect and never a conflict: no comparison
	// against the session happened.
	CodeInvalidRevision = "invalid_revision"
	// CodeRevisionConflict is a well-formed expectedRevision that does not match
	// the session's current revision. Nothing was mutated, and CurrentRevision
	// reports the revision the caller has to reconcile against.
	CodeRevisionConflict = "revision_conflict"
	// CodeUnknownSaveSession names a saveSessionID no session is registered
	// under.
	CodeUnknownSaveSession = "unknown_save_session"
	// CodeOperationFailed is the safe generic classification of a domain failure
	// that has no confirmed finer code yet. It exists so an unclassified failure
	// stays a stable, boring code instead of inviting a heuristic taxonomy built
	// from error text.
	CodeOperationFailed = "operation_failed"
	// CodeInternalError is an unexpected failure. Its message is fixed and its
	// cause only ever reaches the internal log.
	CodeInternalError = "internal_error"
)

// Severity of one failure. Only these two values exist.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// Stage names where a failure happened. It is deliberately coarse: it tells a
// consumer whether the request itself was wrong, whether the addressed session
// was, whether the mutation was refused, or whether the backend failed.
const (
	StageRequest  = "request"
	StageSession  = "session"
	StageMutation = "mutation"
	StageInternal = "internal"
)

// internalMessage is the only wording an unexpected failure ever reports. The
// real cause stays in the internal log beside the same DiagnosticID.
const internalMessage = "The operation could not be completed."

// FieldError names one rejected request field. It is a separate, deliberately
// small model: a field failure needs the field, a stable code and a safe
// fallback, and nothing else.
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error is the public error model. Every member except cause is safe to send to
// a transport and to a user interface.
type Error struct {
	Code            string            `json:"code"`
	Message         string            `json:"message"`
	Params          map[string]string `json:"params,omitempty"`
	Severity        string            `json:"severity"`
	Stage           string            `json:"stage"`
	Retryable       bool              `json:"retryable"`
	FieldErrors     []FieldError      `json:"fieldErrors,omitempty"`
	CurrentRevision string            `json:"currentRevision,omitempty"`
	DiagnosticID    string            `json:"diagnosticID"`

	// cause is the original failure. It is never serialized and never reaches a
	// transport; only an internal log reads it, through Cause.
	cause error
}

// Error makes the model an ordinary Go error, so it can be returned by any
// existing signature and travels through errors.Is and errors.As untouched.
func (e *Error) Error() string { return e.Message }

// Unwrap exposes the private cause to errors.Is and errors.As only.
func (e *Error) Unwrap() error { return e.cause }

// Cause returns the private failure for an internal log. A transport logs it
// together with DiagnosticID and never puts it into a response.
func (e *Error) Cause() error { return e.cause }

// As reports whether err is or wraps a public error, and returns it.
func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// diagnosticIDPrefix marks an identifier minted for one reported failure, so it
// can never be mistaken for a saveSessionID, an operationID or an undo token.
const diagnosticIDPrefix = "diag-"

// newDiagnosticID mints the unpredictable correlation identifier of one
// reported failure. It is random rather than derived, so it names one concrete
// occurrence and no caller can construct the identifier of a failure it never
// saw.
//
// ponytail: a generator failure degrades to an empty identifier instead of
// replacing the reported failure with a second one. Losing correlation is worse
// reporting; losing the original error would be a lost failure.
func newDiagnosticID() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return ""
	}
	return diagnosticIDPrefix + hex.EncodeToString(raw)
}

// InvalidRevision rejects an expectedRevision that is not a canonical decimal
// saveRevision. No session was consulted, so this is a request defect.
func InvalidRevision(value string) *Error {
	message := fmt.Sprintf(
		"expectedRevision must be a canonical decimal saveRevision; got %q", value)
	return &Error{
		Code:     CodeInvalidRevision,
		Message:  message,
		Params:   map[string]string{"value": value},
		Severity: SeverityError,
		Stage:    StageRequest,
		FieldErrors: []FieldError{{
			Field:   "expectedRevision",
			Code:    CodeInvalidRevision,
			Message: message,
		}},
		DiagnosticID: newDiagnosticID(),
	}
}

// RevisionConflict reports that a well-formed expectedRevision no longer
// matches the session. Nothing was mutated. It is not retryable: the caller has
// to review the change against the current revision, and no layer may repeat
// the mutation on its own.
func RevisionConflict(expectedRevision string, currentRevision string) *Error {
	return &Error{
		Code: CodeRevisionConflict,
		Message: fmt.Sprintf(
			"expectedRevision %q does not match the current saveRevision %q",
			expectedRevision, currentRevision),
		Params: map[string]string{
			"expectedRevision": expectedRevision,
			"currentRevision":  currentRevision,
		},
		Severity:        SeverityError,
		Stage:           StageMutation,
		CurrentRevision: currentRevision,
		DiagnosticID:    newDiagnosticID(),
	}
}

// UnknownSaveSession names a saveSessionID no session is registered under.
func UnknownSaveSession(saveSessionID string) *Error {
	return &Error{
		Code:         CodeUnknownSaveSession,
		Message:      fmt.Sprintf("unknown save session %q", saveSessionID),
		Params:       map[string]string{"saveSessionID": saveSessionID},
		Severity:     SeverityError,
		Stage:        StageSession,
		DiagnosticID: newDiagnosticID(),
	}
}

// InvalidRequest rejects request values the endpoint named itself. message is
// the endpoint's own safe English fallback; fields are the members it rejected.
func InvalidRequest(message string, fields ...FieldError) *Error {
	return &Error{
		Code:         CodeInvalidRequest,
		Message:      message,
		Severity:     SeverityError,
		Stage:        StageRequest,
		FieldErrors:  fields,
		DiagnosticID: newDiagnosticID(),
	}
}

// MissingField rejects a required request value that was not supplied. It is
// the one constructor for the "x is required" class, so that class has a stable
// code and a named field instead of a bare sentence.
func MissingField(name string) *Error {
	message := fmt.Sprintf("%s is required", name)
	return InvalidRequest(message, Field(name, message))
}

// Field builds one rejected request field under CodeInvalidRequest.
func Field(name string, message string) FieldError {
	return FieldError{Field: name, Code: CodeInvalidRequest, Message: message}
}

// Internal reports an unexpected failure. The cause is kept for the internal
// log and deliberately never reaches the reported message, because nothing
// vouches for what an arbitrary Go error contains.
func Internal(cause error) *Error {
	return &Error{
		Code:         CodeInternalError,
		Message:      internalMessage,
		Severity:     SeverityError,
		Stage:        StageInternal,
		DiagnosticID: newDiagnosticID(),
		cause:        cause,
	}
}

// OperationFailed is the safe generic classification of a domain failure with
// no confirmed finer code. Like Internal it reports a fixed message and keeps
// the cause for the log: the failure is real and stable, but its wording is not
// something this layer can vouch for.
func OperationFailed(cause error) *Error {
	return &Error{
		Code:         CodeOperationFailed,
		Message:      internalMessage,
		Severity:     SeverityError,
		Stage:        StageMutation,
		DiagnosticID: newDiagnosticID(),
		cause:        cause,
	}
}

// From normalizes any error into the public model. An error that already is one
// is returned unchanged, so the classification a lower layer made is never
// re-derived, and above all never re-derived from its text. Everything else
// becomes the generic classification, keeping its cause for the log only.
//
// A nil error yields nil, so a caller can normalize unconditionally.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	if public, ok := As(err); ok {
		return public
	}
	return OperationFailed(err)
}
