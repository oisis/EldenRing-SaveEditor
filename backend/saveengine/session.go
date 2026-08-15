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

// Session is the model of one loaded save. It deliberately holds no file path,
// no handle, no offsets, no raw bytes and no character data: it recognises and
// validates a container and carries the identity state of the records read from
// it.
//
// revision, ownedByLocator, ownedByID and ownedSeq are the private owned-item
// identity state described in owned_item_id.go. None of them is part of
// SessionInfo, so none of them leaves the package.
type Session struct {
	id       string
	platform Platform
	format   string

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
}

// SessionInfo is the safe, public metadata of a session. It is the only session
// representation that leaves the package.
type SessionInfo struct {
	SaveSessionID string `json:"saveSessionID"`
	Platform      string `json:"platform"`
	Format        string `json:"format"`
	// UnsavedChanges reports whether the private snapshot of the session carries a
	// committed mutation. It is false for a freshly loaded session and stays false
	// while every mutation is rejected or rolled back. A successful WriteSave
	// clears it after persisting the validated snapshot.
	UnsavedChanges bool `json:"unsavedChanges"`
}

// newSession creates a session with a fresh, non-empty identifier.
func newSession(platform Platform, format string) (*Session, error) {
	id, err := newSessionID()
	if err != nil {
		return nil, err
	}
	return &Session{
		id:             id,
		platform:       platform,
		format:         format,
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

// Info returns the metadata a caller may see.
func (session *Session) Info() SessionInfo {
	return SessionInfo{
		SaveSessionID:  session.id,
		Platform:       string(session.platform),
		Format:         session.format,
		UnsavedChanges: session.dirty,
	}
}
