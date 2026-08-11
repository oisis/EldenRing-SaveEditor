package saveengine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// This file owns the two private primitives of the owned-item identity: the
// per-session saveRevision and the registry that maps an opaque OwnedItemID to
// the physical record it was minted for. Nothing here is exported: the engine is
// read-only at this stage, so the identity has no producer and no consumer yet.
//
// The token is opaque by contract. Its shape is an implementation detail of this
// file, it carries no GaItem handle, no acquisition index, no physical index and
// no slot address, and no code outside this file parses it.

const (
	// ownedContainerInventory and ownedContainerStorage are the two physical
	// containers a record can live in. They are part of the private locator only,
	// so a record in Inventory and a record at the same coordinates in Storage
	// are two different records.
	ownedContainerInventory = "inventory"
	ownedContainerStorage   = "storage"

	// ownedItemIDPrefix marks a token minted by this package. It exists so a
	// foreign string can be told apart from one of ours without a tombstone map.
	ownedItemIDPrefix = "oi-"
)

// errUnknownOwnedItemID rejects a token this session never minted, including one
// minted by another session. The caller has no record to address.
var errUnknownOwnedItemID = errors.New("unknown ownedItemID")

// errStaleOwnedItemID rejects a token this session minted under an earlier
// revision. It is deliberately distinct from errUnknownOwnedItemID because the
// remedy differs: the caller re-reads the container instead of giving up.
var errStaleOwnedItemID = errors.New("stale ownedItemID")

// ownedItemLocator is the private physical address of one record. It is never
// returned to a caller outside the package.
//
// containerSection is the section the record was read from, which is a property
// of the record and the only way to tell the third row of the common section
// from the third row of the key section. It is never the caller's
// containerSection filter, so reading one section, both sections or a different
// page never changes which token a record gets.
type ownedItemLocator struct {
	characterID      int
	container        string
	containerSection string
	physicalIndex    int
}

// mintOwnedItemID returns the token of locator under the current revision,
// minting one on first use. Minting is idempotent: the same locator always maps
// back to the same token until the revision advances, so one physical record can
// never be reachable through two tokens.
//
// The caller must already hold Engine.mutex; the session carries no lock of its
// own.
func (session *Session) mintOwnedItemID(locator ownedItemLocator) string {
	if existing, minted := session.ownedByLocator[locator]; minted {
		return existing
	}

	session.ownedSeq++
	// ponytail: session + revision + sequence is unique by construction, so no
	// randomness is needed. The revision in the token is what lets resolve tell
	// stale from unknown without keeping every retired token forever.
	token := fmt.Sprintf("%s%d", session.currentOwnedItemIDPrefix(), session.ownedSeq)
	session.ownedByLocator[locator] = token
	session.ownedByID[token] = locator
	return token
}

// resolveOwnedItemID returns the record ownedItemID was minted for, or an error
// that states why it cannot be resolved. It never falls back to a physical
// lookup and never resolves into another character's slot.
//
// The caller must already hold Engine.mutex.
func (session *Session) resolveOwnedItemID(characterID int, ownedItemID string) (ownedItemLocator, error) {
	if ownedItemID == "" {
		return ownedItemLocator{}, errors.New("ownedItemID is required")
	}

	if locator, known := session.ownedByID[ownedItemID]; known {
		if locator.characterID != characterID {
			return ownedItemLocator{}, fmt.Errorf(
				"ownedItemID belongs to character %d, not to character %d",
				locator.characterID, characterID)
		}
		return locator, nil
	}

	// The registry holds exactly the tokens of the current revision, so one of
	// ours that is missing from it is either fabricated at the current revision
	// or genuinely retired by an earlier increment.
	if strings.HasPrefix(ownedItemID, session.currentOwnedItemIDPrefix()) {
		return ownedItemLocator{}, errUnknownOwnedItemID
	}
	if strings.HasPrefix(ownedItemID, session.ownedItemIDPrefix()) {
		return ownedItemLocator{}, errStaleOwnedItemID
	}
	return ownedItemLocator{}, errUnknownOwnedItemID
}

// ownedItemIDPrefix is the part of a token that identifies the minting session,
// and currentOwnedItemIDPrefix additionally pins the current revision.
func (session *Session) ownedItemIDPrefix() string {
	return ownedItemIDPrefix + session.id + "-"
}

func (session *Session) currentOwnedItemIDPrefix() string {
	return session.ownedItemIDPrefix() + session.revisionString() + "-"
}

// revisionString is the public rendering of the internal uint64 revision: a
// non-empty decimal string with no sign, no prefix, no padding and no separator.
// It is the only way the revision leaves the package, so a getter never exposes
// a JSON number that a frontend could round, increment or reorder.
//
// The caller must already hold Engine.mutex, like every other session helper.
func (session *Session) revisionString() string {
	return strconv.FormatUint(session.revision, 10)
}

// commitRevision runs commit as the mutating step of the session registered
// under saveSessionID and, only when it succeeds, advances the revision by one
// and drops every identity minted under the previous one. The registry is not
// rebuilt here: it re-materialises when a container is read again.
//
// A commit error leaves the revision, both registry maps and the sequence
// exactly as they were, so a validation failure or a rollback never invalidates
// an identity for a change that did not happen.
//
// commit runs under the existing process-wide Engine.mutex, so no caller can
// observe a committed change with a stale revision or the reverse.
//
// ponytail: this is the whole mutation path. The engine is read-only today, so
// there is no exported mutation API and no transaction framework — the callback
// is the smallest hook that makes the increment, the non-increment and the
// invalidation testable. Widen it when the first real mutation lands.
func (engine *Engine) commitRevision(saveSessionID string, commit func() error) error {
	if saveSessionID == "" {
		return errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return fmt.Errorf("unknown save session %q", saveSessionID)
	}

	if err := commit(); err != nil {
		return err
	}

	session := loaded.session
	session.revision++
	session.ownedByLocator = make(map[ownedItemLocator]string)
	session.ownedByID = make(map[string]ownedItemLocator)
	return nil
}
