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

// Session is the read-only model of one loaded save. It deliberately holds no
// file path, no handle, no offsets, no raw bytes and no character data: the
// current stage only recognises and validates a container.
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
	// ownedByLocator and ownedByID are the two directions of one identity
	// registry, valid for the current revision only.
	ownedByLocator map[ownedItemLocator]string
	ownedByID      map[string]ownedItemLocator
	// ownedSeq numbers the tokens minted by this session.
	ownedSeq uint64
}

// SessionInfo is the safe, public metadata of a session. It is the only session
// representation that leaves the package.
type SessionInfo struct {
	SaveSessionID string `json:"saveSessionID"`
	Platform      string `json:"platform"`
	Format        string `json:"format"`
	// UnsavedChanges is always false at this stage: a read-only session cannot
	// change anything.
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
		UnsavedChanges: false,
	}
}
