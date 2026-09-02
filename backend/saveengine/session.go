package saveengine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Platform is the recognised save platform. Only these two values exist; a
// container that is neither is rejected instead of being guessed.
type Platform string

const (
	PlatformPC  Platform = "pc"
	PlatformPS4 Platform = "ps4"
)

// SourceKind states what the file a session was created from is. Only these two
// values exist; a caller that supplies neither is rejected instead of being
// defaulted, so a session never claims an origin nobody stated.
//
// SourceKindLocal is a durable file the user owns. SourceKindTemporary is a
// working copy that is not the user's durable save; it exists for the later
// deployment flow and carries no behaviour of its own at this stage.
type SourceKind string

const (
	SourceKindLocal     SourceKind = "local"
	SourceKindTemporary SourceKind = "temporary"
)

// Session is the model of one loaded save. It deliberately holds no handle, no
// offsets, no raw bytes and no character data: it recognises and validates a
// container and carries the identity state of the records read from it.
//
// sourcePath and sourceKind are desktop session metadata describing where the
// private snapshot came from. They are not permission to read that file again:
// the snapshot is taken once during LoadSave and every later read goes to it,
// so removing or replacing the file leaves this session untouched.
//
// revision, ownedByLocator, ownedByID and ownedSeq are the private owned-item
// identity state described in owned_item_id.go. Of these only revision reaches
// SessionInfo, and only through its canonical decimal rendering.
type Session struct {
	id         string
	platform   Platform
	format     string
	sourcePath string
	sourceKind SourceKind

	// revision is the private saveRevision of this session. It starts at 0 and
	// only commitRevision advances it.
	revision uint64
	// dirty reports whether a mutation has been committed into the private
	// snapshot of this session since it was loaded. It starts false and only
	// commitRevision sets it, in the same critical section as the increment, so a
	// rejected or rolled back mutation never marks the session changed. A
	// successful WriteSave clears it in its own commit critical section.
	dirty bool
	// ownedByLocator and ownedByID are the two directions of one identity
	// registry, valid for the current revision only.
	ownedByLocator map[ownedItemLocator]string
	ownedByID      map[string]ownedItemLocator
	// ownedSeq numbers the tokens minted by this session.
	ownedSeq uint64
	// undo is the single private restore point of this session, described in
	// undo.go. It starts nil, holds at most one character mutation, and is
	// never serialized.
	undo *undoPoint
	// reviewAuthorization is the last successful Review Changes validation. It
	// is bound to one exact revision and is invalidated by every later mutation.
	reviewAuthorization *reviewAuthorization
	// journal is the private ring buffer of diagnostic records for this session.
	journal []DiagnosticRecord
	// journalSeq numbers the diagnostic records appended to this session.
	journalSeq uint64
	// eventSeq numbers the session.changed events published for this session. It
	// starts at 0 and only a committed mutation advances it, so it counts
	// committed mutations and never rejections, rollbacks or successes that
	// committed nothing.
	eventSeq uint64
}

// SessionInfo is the safe, public metadata of a session. It is the only session
// representation that leaves the package.
//
// SourcePath is the exact path the snapshot was created from, carried verbatim:
// it is never trimmed, recased, resolved or guessed. It is metadata about the
// origin of the session and nothing more, so it grants no caller the right to
// reopen that file.
//
// SourceKind is the caller's exact statement of what that path is, echoed back
// unchanged.
//
// SaveRevision is the canonical decimal rendering of the private session
// revision, "0" for a freshly loaded session. It is a string and not a number so
// no consumer can round, increment or reorder it.
//
// The struct still exposes no handle, no offset, no snapshot and no save byte.
type SessionInfo struct {
	SaveSessionID string `json:"saveSessionID"`
	Platform      string `json:"platform"`
	Format        string `json:"format"`
	SourcePath    string `json:"sourcePath"`
	SourceKind    string `json:"sourceKind"`
	SaveRevision  string `json:"saveRevision"`
	// UnsavedChanges reports whether the private snapshot of the session carries a
	// committed mutation. It is false for a freshly loaded session and stays false
	// while every mutation is rejected or rolled back. A successful WriteSave
	// clears it after persisting the validated snapshot.
	UnsavedChanges bool `json:"unsavedChanges"`
	// EventSequence is the canonical decimal rendering of the session's
	// session.changed counter, "0" for a session that has committed nothing. It
	// exists so a subscriber that starts late, misses events or reconnects can
	// read the current position of the stream instead of guessing it, and it is a
	// string for the same reason SaveRevision is one.
	EventSequence string `json:"eventSequence"`
}

// newSession creates a session with a fresh, non-empty identifier. sourcePath
// and sourceKind are stored exactly as the caller validated them; this
// constructor neither validates nor rewrites either value.
func newSession(
	platform Platform,
	format string,
	sourcePath string,
	sourceKind SourceKind,
) (*Session, error) {
	id, err := newSessionID()
	if err != nil {
		return nil, err
	}
	return &Session{
		id:             id,
		platform:       platform,
		format:         format,
		sourcePath:     sourcePath,
		sourceKind:     sourceKind,
		ownedByLocator: make(map[ownedItemLocator]string),
		ownedByID:      make(map[string]ownedItemLocator),
	}, nil
}

func newSessionID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("cannot create save session identifier: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// Info returns the metadata a caller may see, as a value: the caller receives
// an independent copy and can reach neither the session nor its snapshot.
//
// The caller must already hold Engine.mutex, like every other session helper:
// the revision is read here.
func (session *Session) Info() SessionInfo {
	return SessionInfo{
		SaveSessionID:  session.id,
		Platform:       string(session.platform),
		Format:         session.format,
		SourcePath:     session.sourcePath,
		SourceKind:     string(session.sourceKind),
		SaveRevision:   session.revisionString(),
		UnsavedChanges: session.dirty,
		EventSequence:  session.eventSequenceString(),
	}
}
