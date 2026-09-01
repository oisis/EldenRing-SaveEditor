package saveengine

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// This file owns the single undo point of a save session.
//
// SaveForge 2.0 deliberately keeps one point, not the per-slot stack of depth
// five that SaveForge 1.5.8 and 1.6.8 carried. The saveRevision of this engine
// is global: as soon as any mutation commits, every older point describes a
// revision that no longer exists, so a stack could never offer more than its
// newest entry anyway.
//
// The point stores the three physical ranges one character-slot mutation may
// touch — the slot data, the slot's ProfileSummary and its one activity byte in
// UserData10 — plus the dirty flag that was in force before the undone
// mutation. It never stores the container, is never serialized and never
// reaches a save file. The PC MD5 prefixes are outside its scope because
// WriteSave regenerates all eleven of them from the data it is about to write.

// undoPoint is the private, non-serializable restore point of one committed
// character mutation. It belongs to exactly one session, one characterID and
// one revision.
type undoPoint struct {
	characterID   int
	operationKind string
	token         string
	// revision is the revision this point restores away from. It is the
	// revision the mutation created, so a later mutation makes it unusable.
	revision uint64
	// dirtyBefore is the session's dirty flag as it was before the undone
	// mutation, so undoing the first mutation of a clean session leaves the
	// session clean again.
	dirtyBefore bool

	slotAt    int64
	slot      []byte
	summaryAt int64
	summary   []byte
	flagAt    int64
	flag      []byte
}

// captureUndoPoint copies the three ranges of characterID before a mutation
// runs: the slot data, the slot's ProfileSummary and its activity byte. This is
// the only place the undo scope is resolved; the restore path replays the
// offsets stored here.
//
// It fails closed: when the slot index, a range or the token source is
// unusable it returns an error, and the caller must abandon the mutation
// instead of running it without the undo point it promised.
//
// The caller must already hold Engine.mutex.
func captureUndoPoint(loaded *loadedSave, characterID int, operationKind string) (*undoPoint, error) {
	if characterID < 0 || characterID >= characterSlotCount {
		return nil, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}
	userData10At := userData10Base(loaded.session.platform)
	slotAt := slotDataBase(loaded.session.platform, characterID)
	summaryAt := userData10At + userData10SummaryOffset +
		int64(characterID)*userData10SummaryStride
	flagAt := userData10At + userData10ActiveFlagsOffset + int64(characterID)

	// ponytail: this copies 0x280000 + 0x24C + 1 bytes on every character
	// mutation, accepted or rejected. It is the price of a pre-image; a
	// cheaper point would have to know which sub-range each mutation writes,
	// which is exactly the per-mutation duplication this helper exists to
	// avoid.
	slot, slotErr := loaded.snapshot.readAt(slotAt, characterSlotDataSize)
	summary, summaryErr := loaded.snapshot.readAt(summaryAt, userData10SummaryStride)
	flag, flagErr := loaded.snapshot.readAt(flagAt, 1)
	if slotErr != nil || summaryErr != nil || flagErr != nil {
		return nil, fmt.Errorf(
			"cannot prepare an undo point for character %d: %w",
			characterID, errors.Join(slotErr, summaryErr, flagErr))
	}

	token, err := newUndoToken()
	if err != nil {
		return nil, fmt.Errorf("cannot prepare an undo point for character %d: %w", characterID, err)
	}

	return &undoPoint{
		characterID:   characterID,
		operationKind: operationKind,
		token:         token,
		slotAt:        slotAt,
		slot:          slot,
		summaryAt:     summaryAt,
		summary:       summary,
		flagAt:        flagAt,
		flag:          flag,
	}, nil
}

// changedIn reports whether the mutation actually altered one of the three
// ranges. A successful call that changed none of them still advances the
// revision under the existing contract, but it creates no empty undo point.
func (point *undoPoint) changedIn(snapshot *codec) bool {
	return !snapshot.sameAt(point.slotAt, point.slot) ||
		!snapshot.sameAt(point.summaryAt, point.summary) ||
		!snapshot.sameAt(point.flagAt, point.flag)
}

// newUndoToken mints an unpredictable identifier for one undo point. It is
// random rather than derived so a caller cannot construct the token of a point
// it never saw.
func newUndoToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("cannot create undo token: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// CharacterUndoState reports whether the session currently holds a usable undo
// point for one character. It returns no snapshot byte and no private session
// state.
type CharacterUndoState struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Available     bool   `json:"available"`
	UndoToken     string `json:"undoToken,omitempty"`
	OperationKind string `json:"operationKind,omitempty"`
}

// GetUndoState returns the undo state of one character slot. It reads the
// session only: the undo point, the snapshot, the revision, the dirty flag and
// the OwnedItemID registries are all left exactly as they are.
//
// saveSessionID is matched exactly. Available is true only when the session's
// single undo point belongs to this character and to the current revision;
// otherwise the token and the operation kind stay empty.
func (engine *Engine) GetUndoState(saveSessionID string, characterID int) (CharacterUndoState, error) {
	if saveSessionID == "" {
		return CharacterUndoState{}, errors.New("saveSessionID is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return CharacterUndoState{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return CharacterUndoState{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	session := loaded.session
	state := CharacterUndoState{
		SaveSessionID: saveSessionID,
		SaveRevision:  session.revisionString(),
		CharacterID:   characterID,
	}
	point := session.undo
	if point == nil || point.characterID != characterID || point.revision != session.revision {
		return state, nil
	}
	state.Available = true
	state.UndoToken = point.token
	state.OperationKind = point.operationKind
	return state, nil
}

// UndoCharacterChangesResult reports one consumed undo point.
type UndoCharacterChangesResult struct {
	SaveSessionID       string `json:"saveSessionID"`
	SaveRevision        string `json:"saveRevision"`
	CharacterID         int    `json:"characterID"`
	UndoneOperationKind string `json:"undoneOperationKind"`
}

// UndoCharacterChanges restores the three ranges the last committed mutation of
// characterID owned. It replays no domain mutation in reverse: it writes back
// the bytes captured before that mutation.
//
// The complete operation runs under Engine.mutex. All three current ranges are
// read before the first byte is written, and all three are verified after the
// last one; a write or verification failure restores the state the call started
// from and leaves the undo point, the revision and the dirty flag untouched.
//
// A successful undo consumes the point, restores the dirty flag from before the
// undone mutation, advances saveRevision by exactly one and retires every
// OwnedItemID of the old revision. It creates no redo and no new undo point.
func (engine *Engine) UndoCharacterChanges(
	saveSessionID string,
	characterID int,
	undoToken string,
	expectedRevision string,
) (UndoCharacterChangesResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return UndoCharacterChangesResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if saveSessionID == "" {
		return UndoCharacterChangesResult{}, errors.New("saveSessionID is required")
	}
	if undoToken == "" {
		return UndoCharacterChangesResult{}, errors.New("undoToken is required")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return UndoCharacterChangesResult{}, fmt.Errorf("unknown save session %q", saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return UndoCharacterChangesResult{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	session := loaded.session
	current := session.revisionString()
	if expectedRevision != current {
		return UndoCharacterChangesResult{}, fmt.Errorf(
			"expectedRevision %q does not match the current saveRevision %q",
			expectedRevision, current)
	}

	point := session.undo
	if point == nil || point.characterID != characterID || point.revision != session.revision {
		return UndoCharacterChangesResult{}, fmt.Errorf(
			"no undo point is available for character %d at saveRevision %s", characterID, current)
	}
	// ponytail: a plain comparison. The token exists so a stale or guessed
	// client cannot undo a mutation it never observed, on a loopback-only local
	// API; it is not a secret defended against an offline attacker.
	if point.token != undoToken {
		return UndoCharacterChangesResult{}, fmt.Errorf(
			"undoToken does not match the undo point of character %d", characterID)
	}

	// The undo produces its own receipt through the shared mutation path. It is
	// prepared here, before the first write: an unknown kind or a failing
	// identifier generator refuses the undo instead of surfacing after the
	// restore has already committed a revision. Its changed scopes are the scopes
	// of the mutation being reverted, taken from the point, so an undo never
	// reports a catch-all.
	pending, err := engine.prepareMutation(
		kindUndoCharacterChanges, undoneChangedScopes(point.operationKind)...)
	if err != nil {
		return UndoCharacterChangesResult{}, err
	}

	// The pre-undo image of all three ranges is read before the first write, so
	// a failure halfway through can be reverted completely.
	slotNow, slotErr := loaded.snapshot.readAt(point.slotAt, len(point.slot))
	summaryNow, summaryErr := loaded.snapshot.readAt(point.summaryAt, len(point.summary))
	flagNow, flagErr := loaded.snapshot.readAt(point.flagAt, len(point.flag))
	if slotErr != nil || summaryErr != nil || flagErr != nil {
		return UndoCharacterChangesResult{}, fmt.Errorf(
			"cannot read the current data of character %d: %w",
			characterID, errors.Join(slotErr, summaryErr, flagErr))
	}

	restoreFailure := func(failure string) error {
		return restoreCharacterSlotRanges(
			loaded.snapshot, characterID,
			point.slotAt, slotNow, point.summaryAt, summaryNow, point.flagAt, flagNow,
			failure)
	}

	if err := loaded.snapshot.writeAt(point.slotAt, point.slot); err != nil {
		return UndoCharacterChangesResult{}, restoreFailure(fmt.Sprintf(
			"character %d was not restored: the slot data could not be written: %v", characterID, err))
	}
	if err := loaded.snapshot.writeAt(point.summaryAt, point.summary); err != nil {
		return UndoCharacterChangesResult{}, restoreFailure(fmt.Sprintf(
			"character %d was not restored: the profile summary could not be written: %v",
			characterID, err))
	}
	if err := loaded.snapshot.writeAt(point.flagAt, point.flag); err != nil {
		return UndoCharacterChangesResult{}, restoreFailure(fmt.Sprintf(
			"character %d was not restored: the activity flag could not be written: %v",
			characterID, err))
	}

	if point.changedIn(loaded.snapshot) {
		return UndoCharacterChangesResult{}, restoreFailure(fmt.Sprintf(
			"the restored data of character %d could not be verified", characterID))
	}

	undoneKind := point.operationKind
	session.undo = nil
	session.dirty = point.dirtyBefore
	receipt := pending.receipt(saveSessionID, session.advanceRevision())
	return UndoCharacterChangesResult{
		SaveSessionID:       receipt.SaveSessionID,
		SaveRevision:        receipt.SaveRevision,
		CharacterID:         characterID,
		UndoneOperationKind: undoneKind,
	}, nil
}

// undoneChangedScopes resolves the scopes the reverted mutation owned. An
// unregistered kind cannot reach a stored undo point, because commitWithHook
// validates the kind before it captures one, so an empty result here only
// happens for a kind that carries no domain scope of its own.
func undoneChangedScopes(undoneKind string) []string {
	scopes, err := changedScopesForMutationKind(undoneKind)
	if err != nil {
		return nil
	}
	return scopes
}
