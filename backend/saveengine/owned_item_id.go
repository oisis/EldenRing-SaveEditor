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

// IsCanonicalRevision reports whether value is a canonical decimal saveRevision
// string with no sign, prefix, padding or separator.
func IsCanonicalRevision(value string) bool {
	return isCanonicalRevision(value)
}

// isCanonicalRevision accepts exactly the decimal representation emitted by
// revisionString. Callers still compare the original string byte for byte; the
// parsed value never becomes part of the public contract.
func isCanonicalRevision(value string) bool {
	parsed, err := strconv.ParseUint(value, 10, 64)
	return err == nil && strconv.FormatUint(parsed, 10) == value
}

// advanceRevision retires every identity of the current revision and returns
// the next revision in its public form. The caller owns the dirty-state outcome
// and must already hold Engine.mutex.
func (session *Session) advanceRevision() string {
	session.revision++
	session.ownedByLocator = make(map[ownedItemLocator]string)
	session.ownedByID = make(map[string]ownedItemLocator)
	return session.revisionString()
}

// commitRevision runs commit as the mutating step of the session registered
// under saveSessionID and, only when it succeeds, advances the revision by one,
// marks the session dirty and drops every identity minted under the previous
// revision. The registry is not rebuilt here: it re-materialises when a
// container is read again. The new revision is returned as its canonical decimal
// string, so the caller reports exactly the value the next request has to match.
//
// A commit error leaves the revision, the dirty flag, both registry maps and the
// sequence exactly as they were and returns an empty revision, so a validation
// failure or a rollback never invalidates an identity, and never claims an
// unsaved change, for a change that did not happen.
//
// commit receives the loaded session it is about to mutate and runs under the
// existing process-wide Engine.mutex, which this function takes exactly once. It
// may therefore use every session and container helper that requires the lock to
// be held already, and must not call a public engine method, which would take
// the same lock again.
//
// A global mutation invalidates the session's single undo point, because the
// point is pinned to a revision this call is about to retire.
//
// ponytail: this is the whole mutation path. The callback is the smallest hook
// that keeps the increment, the non-increment and the invalidation in one place;
// there is no transaction framework, no mutation-plan type and no per-session
// lock. Widen it only when a mutation needs something this signature cannot
// express.
func (engine *Engine) commitRevision(
	saveSessionID string,
	commit func(*loadedSave) error,
) (string, error) {
	return engine.commit(saveSessionID, "", 0, commit)
}

// commitCharacterRevision is commitRevision for a mutation that owns one
// character slot. In addition to the revision contract it records the session's
// single undo point: the three ranges of characterID as they were before
// commit, attributed to operationID.
//
// The point replaces any earlier one. A commit that changes none of the three
// ranges records no point and drops the earlier one, so an undo can never
// restore a revision that is no longer the current one.
//
// When the point cannot be captured the mutation is refused before commit runs,
// so a character mutation never succeeds without the undo point it promises.
func (engine *Engine) commitCharacterRevision(
	saveSessionID string,
	operationID string,
	characterID int,
	commit func(*loadedSave) error,
) (string, error) {
	if operationID == "" {
		return "", errors.New("operationID is required")
	}
	return engine.commit(saveSessionID, operationID, characterID, commit)
}

// commit is the one implementation behind both entry points. An empty
// operationID marks a global mutation, which records no undo point.
func (engine *Engine) commit(
	saveSessionID string,
	operationID string,
	characterID int,
	commit func(*loadedSave) error,
) (string, error) {
	if saveSessionID == "" {
		return "", errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return "", fmt.Errorf("unknown save session %q", saveSessionID)
	}

	// Fail closed: a character mutation that cannot get its undo point does not
	// run at all. Returning here leaves the earlier point, the revision, the
	// dirty flag and both registries exactly as they were.
	var point *undoPoint
	if operationID != "" {
		captured, err := captureUndoPoint(loaded, characterID, operationID)
		if err != nil {
			return "", err
		}
		point = captured
	}

	if err := commit(loaded); err != nil {
		return "", err
	}

	session := loaded.session
	// The previous point cannot survive this commit under any branch: its
	// revision expires below.
	session.undo = nil
	if point != nil && point.changedIn(loaded.snapshot) {
		point.dirtyBefore = session.dirty
		session.undo = point
	}
	session.dirty = true
	revision := session.advanceRevision()
	if session.undo != nil {
		session.undo.revision = session.revision
	}
	return revision, nil
}
