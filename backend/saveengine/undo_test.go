package saveengine

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeUndoFixture builds a complete container whose slot setRunesTestSlot is
// active and carries a usable statistics anchor, so SetCharacterRunes is a real
// mutation of that slot and the container still survives WriteSave. It returns
// the fixture path and the absolute offset of the held-runes field.
func writeUndoFixture(t *testing.T, platform Platform) (string, int64) {
	t.Helper()

	content := gestureTestActiveFixture(
		platform, setRunesTestSlot, setRunesTestFullAnchorAt, 0)
	content.records = setGestureTestRecords()
	path := writeGestureFixture(t, content)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read undo fixture: %v", err)
	}
	slotBase := int64(setRunesTestPCSlotBase) + setRunesTestSlot*setRunesTestPCSlotStride
	if platform == PlatformPS4 {
		slotBase = setRunesTestPS4SlotBase + setRunesTestSlot*setRunesTestPS4SlotStride
	}
	binary.LittleEndian.PutUint32(data[slotBase:], 0x6E)
	runesAt := setRunesTestFieldAt(platform, setRunesTestFullAnchorAt)
	binary.LittleEndian.PutUint32(data[runesAt:], 123)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite undo fixture: %v", err)
	}
	return path, runesAt
}

func undoTestRunes(t *testing.T, engine *Engine, saveSessionID string, at int64) uint32 {
	t.Helper()

	raw, err := engine.sessions[saveSessionID].snapshot.readAt(at, 4)
	if err != nil {
		t.Fatalf("read runes at 0x%X: %v", at, err)
	}
	return binary.LittleEndian.Uint32(raw)
}

// One real mutation has to create the session's single undo point, the getter
// has to report it without changing anything, and the undo has to restore the
// bytes, consume the point, advance the revision by one and put the dirty flag
// back to what it was before the undone mutation. Both platforms run the same
// case because the undo scope is expressed in platform-specific bases.
func TestUndoRestoresOneCharacterMutationOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			source, runesAt := writeUndoFixture(t, platform)
			engine := New()
			loaded, err := engine.LoadSave(source, string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			session := loaded.SaveSessionID

			fresh, err := engine.GetUndoState(session, setRunesTestSlot)
			if err != nil {
				t.Fatalf("GetUndoState before any mutation: %v", err)
			}
			if fresh.Available || fresh.UndoToken != "" || fresh.OperationID != "" {
				t.Fatalf("undo state of a fresh session = %+v, want an unavailable point", fresh)
			}

			if _, err := engine.SetCharacterRunes(
				session, setRunesTestSlot, setRunesTestMaximum, "0"); err != nil {
				t.Fatalf("SetCharacterRunes: %v", err)
			}

			state, err := engine.GetUndoState(session, setRunesTestSlot)
			if err != nil {
				t.Fatalf("GetUndoState: %v", err)
			}
			if !state.Available || state.UndoToken == "" {
				t.Fatalf("undo state = %+v, want an available point with a token", state)
			}
			if state.OperationID != "set_character_runes" {
				t.Errorf("operationID = %q, want set_character_runes", state.OperationID)
			}
			if state.SaveRevision != "1" {
				t.Errorf("saveRevision = %q, want 1", state.SaveRevision)
			}

			// The getter is non-mutating: repeating it changes neither the point
			// nor the revision nor the unsaved-changes flag.
			again, err := engine.GetUndoState(session, setRunesTestSlot)
			if err != nil {
				t.Fatalf("second GetUndoState: %v", err)
			}
			if again != state {
				t.Errorf("second undo state = %+v, want the identical %+v", again, state)
			}
			info, err := engine.GetSessionInfo(session)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if !info.UnsavedChanges {
				t.Error("session after the mutation is clean, want dirty")
			}

			result, err := engine.UndoCharacterChanges(
				session, setRunesTestSlot, state.UndoToken, "1")
			if err != nil {
				t.Fatalf("UndoCharacterChanges: %v", err)
			}
			want := UndoCharacterChangesResult{
				SaveSessionID:     session,
				SaveRevision:      "2",
				CharacterID:       setRunesTestSlot,
				UndoneOperationID: "set_character_runes",
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}
			if got := undoTestRunes(t, engine, session, runesAt); got != 123 {
				t.Errorf("runes after undo = %d, want the pre-mutation 123", got)
			}
			info, err = engine.GetSessionInfo(session)
			if err != nil {
				t.Fatalf("GetSessionInfo after undo: %v", err)
			}
			if info.UnsavedChanges {
				t.Error("session after undoing its only mutation is dirty, want clean")
			}

			// The point is consumed and the undo creates no redo and no new point.
			consumed, err := engine.GetUndoState(session, setRunesTestSlot)
			if err != nil {
				t.Fatalf("GetUndoState after undo: %v", err)
			}
			if consumed.Available {
				t.Errorf("undo state after undo = %+v, want an unavailable point", consumed)
			}
			if _, err := engine.UndoCharacterChanges(
				session, setRunesTestSlot, state.UndoToken, "2"); err == nil {
				t.Error("a second undo succeeded, want the consumed point to be gone")
			}
		})
	}
}

// A rejected undo must change nothing at all: not the snapshot, not the
// revision, not the dirty flag and not the undo point itself.
func TestUndoRejectsAWrongTokenCharacterOrRevisionWithoutChangingAnything(t *testing.T) {
	source, runesAt := writeUndoFixture(t, PlatformPC)
	engine := New()
	loaded, err := engine.LoadSave(source, string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	session := loaded.SaveSessionID
	if _, err := engine.SetCharacterRunes(
		session, setRunesTestSlot, setRunesTestMaximum, "0"); err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}
	state, err := engine.GetUndoState(session, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}

	for _, rejection := range []struct {
		name        string
		characterID int
		token       string
		revision    string
	}{
		{"wrong token", setRunesTestSlot, "not-the-token", "1"},
		{"empty token", setRunesTestSlot, "", "1"},
		{"wrong character", setRunesTestSlot + 1, state.UndoToken, "1"},
		{"stale revision", setRunesTestSlot, state.UndoToken, "0"},
		{"non-canonical revision", setRunesTestSlot, state.UndoToken, "01"},
	} {
		t.Run(rejection.name, func(t *testing.T) {
			result, err := engine.UndoCharacterChanges(
				session, rejection.characterID, rejection.token, rejection.revision)
			if err == nil {
				t.Fatalf("UndoCharacterChanges succeeded with %+v, want a rejection", result)
			}
			if result != (UndoCharacterChangesResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
			if got := undoTestRunes(t, engine, session, runesAt); got != setRunesTestMaximum {
				t.Errorf("runes after the rejection = %d, want the mutated %d",
					got, setRunesTestMaximum)
			}
			unchanged, err := engine.GetUndoState(session, setRunesTestSlot)
			if err != nil {
				t.Fatalf("GetUndoState after the rejection: %v", err)
			}
			if unchanged != state {
				t.Errorf("undo state after the rejection = %+v, want the untouched %+v",
					unchanged, state)
			}
		})
	}
}

// The session keeps one point. A second changing mutation replaces it, a
// successful call that changed none of the three ranges drops it without
// creating an empty one, and a global commit drops it too.
func TestTheNextCommitReplacesOrInvalidatesTheUndoPoint(t *testing.T) {
	source, _ := writeUndoFixture(t, PlatformPC)
	engine := New()
	loaded, err := engine.LoadSave(source, string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	session := loaded.SaveSessionID

	if _, err := engine.SetCharacterRunes(session, setRunesTestSlot, 500, "0"); err != nil {
		t.Fatalf("first SetCharacterRunes: %v", err)
	}
	first, err := engine.GetUndoState(session, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the first mutation: %v", err)
	}

	if _, err := engine.SetCharacterRunes(session, setRunesTestSlot, 700, "1"); err != nil {
		t.Fatalf("second SetCharacterRunes: %v", err)
	}
	second, err := engine.GetUndoState(session, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the second mutation: %v", err)
	}
	if !second.Available || second.UndoToken == first.UndoToken {
		t.Fatalf("undo state after the second mutation = %+v, want a new point replacing %+v",
			second, first)
	}
	if _, err := engine.UndoCharacterChanges(
		session, setRunesTestSlot, first.UndoToken, "2"); err == nil {
		t.Error("the replaced token still undoes, want it retired")
	}

	// An idempotent assignment advances the revision under the existing
	// contract, but it changed no byte of the three ranges, so it must leave no
	// point behind at all.
	if _, err := engine.SetCharacterRunes(session, setRunesTestSlot, 700, "2"); err != nil {
		t.Fatalf("idempotent SetCharacterRunes: %v", err)
	}
	after, err := engine.GetUndoState(session, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the idempotent mutation: %v", err)
	}
	if after.Available {
		t.Errorf("undo state after a no-op mutation = %+v, want no point", after)
	}

	// A global commit records no point and drops the existing one.
	if _, err := engine.SetCharacterRunes(session, setRunesTestSlot, 900, "3"); err != nil {
		t.Fatalf("SetCharacterRunes before the global commit: %v", err)
	}
	if _, err := engine.commitRevision(session, func(*loadedSave) error { return nil }); err != nil {
		t.Fatalf("global commitRevision: %v", err)
	}
	global, err := engine.GetUndoState(session, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the global commit: %v", err)
	}
	if global.Available {
		t.Errorf("undo state after a global mutation = %+v, want no point", global)
	}
}

// A successful WriteSave makes the persisted file the new baseline, so the point
// goes with the revision it belonged to. A rejected WriteSave keeps it.
func TestWriteSaveClearsTheUndoPointOnlyWhenItSucceeds(t *testing.T) {
	source, _ := writeUndoFixture(t, PlatformPC)
	engine := New()
	loaded, err := engine.LoadSave(source, string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	session := loaded.SaveSessionID
	if _, err := engine.SetCharacterRunes(
		session, setRunesTestSlot, setRunesTestMaximum, "0"); err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}
	state, err := engine.GetUndoState(session, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if !state.Available {
		t.Fatal("no undo point after the mutation")
	}

	if _, err := engine.WriteSave(session, "0", filepath.Join(t.TempDir(), "stale.sl2")); err == nil {
		t.Fatal("WriteSave with a stale expectedRevision succeeded, want a rejection")
	}
	kept, err := engine.GetUndoState(session, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the rejected write: %v", err)
	}
	if kept != state {
		t.Errorf("undo state after a rejected write = %+v, want the untouched %+v", kept, state)
	}

	if _, err := engine.WriteSave(session, "1", filepath.Join(t.TempDir(), "written.sl2")); err != nil {
		t.Fatalf("WriteSave: %v", err)
	}
	cleared, err := engine.GetUndoState(session, setRunesTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState after the write: %v", err)
	}
	if cleared.Available {
		t.Errorf("undo state after a successful write = %+v, want no point", cleared)
	}
}

// CloneCharacter is the mutation that writes all three ranges of its target
// slot, so it proves that the activity flag and the ProfileSummary are restored
// and that the point belongs to the target, never to the source.
func TestUndoRestoresTheClonedTargetSlotFlagAndProfileSummary(t *testing.T) {
	source := writeCloneCharacterFixture(t, PlatformPC, "Ranni")
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	engine := New()
	loaded, err := engine.LoadSave(source, string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	session := loaded.SaveSessionID

	if _, err := engine.CloneCharacter(
		session, cloneTestSourceSlot, cloneTestTargetSlot, "0"); err != nil {
		t.Fatalf("CloneCharacter: %v", err)
	}

	if state, err := engine.GetUndoState(session, cloneTestSourceSlot); err != nil {
		t.Fatalf("GetUndoState for the source slot: %v", err)
	} else if state.Available {
		t.Errorf("source slot undo state = %+v, want the point on the target slot only", state)
	}
	state, err := engine.GetUndoState(session, cloneTestTargetSlot)
	if err != nil {
		t.Fatalf("GetUndoState for the target slot: %v", err)
	}
	if !state.Available || state.OperationID != "clone_character" {
		t.Fatalf("target slot undo state = %+v, want an available clone_character point", state)
	}

	if _, err := engine.UndoCharacterChanges(
		session, cloneTestTargetSlot, state.UndoToken, "1"); err != nil {
		t.Fatalf("UndoCharacterChanges: %v", err)
	}

	snapshot := engine.sessions[session].snapshot
	targetSlotAt := cloneTestSlotAt(PlatformPC, cloneTestTargetSlot)
	targetSummaryAt := cloneTestSummaryAt(PlatformPC, cloneTestTargetSlot)
	targetFlagAt := cloneTestFlagAt(PlatformPC, cloneTestTargetSlot)
	for _, restored := range []struct {
		name   string
		at     int64
		length int
	}{
		{"slot data", targetSlotAt, cloneTestSlotSize},
		{"profile summary", targetSummaryAt, cloneTestSummaryStride},
		{"activity flag", targetFlagAt, 1},
	} {
		got, err := snapshot.readAt(restored.at, restored.length)
		if err != nil {
			t.Fatalf("read restored %s: %v", restored.name, err)
		}
		if !bytes.Equal(got, before[restored.at:restored.at+int64(restored.length)]) {
			t.Errorf("restored %s of the target slot differs from the pre-clone bytes", restored.name)
		}
	}
	if !bytes.Equal(
		engine.sessions[session].snapshot.data[cloneTestSlotAt(PlatformPC, cloneTestSourceSlot):][:cloneTestSlotSize],
		before[cloneTestSlotAt(PlatformPC, cloneTestSourceSlot):][:cloneTestSlotSize],
	) {
		t.Error("the source slot changed, want it byte-exact through clone and undo")
	}
}

// The getter and the mutation both reject an unusable session or slot index
// before they look at any undo state.
func TestUndoEndpointsRejectAnUnknownSessionOrSlot(t *testing.T) {
	engine := New()
	if _, err := engine.GetUndoState("", 0); err == nil {
		t.Error("GetUndoState accepted an empty saveSessionID")
	}
	if _, err := engine.GetUndoState("unknown", 0); err == nil {
		t.Error("GetUndoState accepted an unknown saveSessionID")
	}
	if _, err := engine.UndoCharacterChanges("unknown", 0, "token", "0"); err == nil {
		t.Error("UndoCharacterChanges accepted an unknown saveSessionID")
	}

	source, _ := writeUndoFixture(t, PlatformPC)
	loaded, err := engine.LoadSave(source, string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	for _, characterID := range []int{-1, characterSlotCount} {
		if _, err := engine.GetUndoState(loaded.SaveSessionID, characterID); err == nil {
			t.Errorf("GetUndoState accepted characterID %d", characterID)
		}
		if _, err := engine.UndoCharacterChanges(
			loaded.SaveSessionID, characterID, "token", "0"); err == nil {
			t.Errorf("UndoCharacterChanges accepted characterID %d", characterID)
		}
	}
}

// The three public appearance entry points share one private writer, so each of
// them has to name its own endpoint on the undo point that writer creates. Only
// the operation identifier is asserted here; the bytes are covered by the
// dedicated appearance tests.
func TestEachAppearanceEntryPointRecordsItsOwnOperationID(t *testing.T) {
	after := setAppearanceTestValues(0x53)
	after.Gender = 0
	after.VoiceType = 5

	testCases := []struct {
		name        string
		operationID string
		mutate      func(engine *Engine, saveSessionID string) error
	}{
		{
			name:        "SetCharacterAppearance",
			operationID: "set_character_appearance",
			mutate: func(engine *Engine, saveSessionID string) error {
				_, err := engine.SetCharacterAppearance(
					saveSessionID, setAppearanceTestSlot, after, "0")
				return err
			},
		},
		{
			name:        "SetCharacterGenderAppearance",
			operationID: "set_character_gender",
			mutate: func(engine *Engine, saveSessionID string) error {
				_, err := engine.SetCharacterGenderAppearance(
					saveSessionID, setAppearanceTestSlot, after, "0")
				return err
			},
		},
		{
			name:        "ApplyCharacterAppearancePreset",
			operationID: "apply_appearance_preset",
			mutate: func(engine *Engine, saveSessionID string) error {
				_, err := engine.ApplyCharacterAppearancePreset(
					saveSessionID, setAppearanceTestSlot, after, "0")
				return err
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			path, _, _, _ := writeSetAppearanceFixture(
				t, PlatformPC, setAppearanceTestValues(0x11))
			engine := New()
			loaded, err := engine.LoadSave(path, string(PlatformPC))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			if err := testCase.mutate(engine, loaded.SaveSessionID); err != nil {
				t.Fatalf("%s: %v", testCase.name, err)
			}

			state, err := engine.GetUndoState(loaded.SaveSessionID, setAppearanceTestSlot)
			if err != nil {
				t.Fatalf("GetUndoState: %v", err)
			}
			if !state.Available {
				t.Fatalf("state = %+v, want an available undo point", state)
			}
			if state.UndoToken == "" {
				t.Error("the undo point reports no token")
			}
			if state.SaveRevision != "1" {
				t.Errorf("saveRevision = %q, want \"1\"", state.SaveRevision)
			}
			if state.OperationID != testCase.operationID {
				t.Errorf("operationID = %q, want %q", state.OperationID, testCase.operationID)
			}
		})
	}
}

// commitCharacterRevision is the one hook every character mutation goes through,
// so an unnamed operation is a programming error rather than an anonymous point.
func TestCommitCharacterRevisionRequiresAnOperationID(t *testing.T) {
	engine := New()
	source, _ := writeUndoFixture(t, PlatformPC)
	loaded, err := engine.LoadSave(source, string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = engine.commitCharacterRevision(
		loaded.SaveSessionID, "", 0, func(*loadedSave) error { return errors.New("must not run") })
	if err == nil || err.Error() != "operationID is required" {
		t.Fatalf("error = %v, want the missing-operationID rejection", err)
	}
}

// A character mutation that cannot capture its undo point must not run at all.
// Truncating the snapshot makes the three ranges unreadable, which is the only
// failure the capture can hit without a fault-injection seam.
func TestCommitCharacterRevisionRefusesTheMutationWhenTheUndoPointCannotBeCaptured(t *testing.T) {
	engine := New()
	source, _ := writeUndoFixture(t, PlatformPC)
	loaded, err := engine.LoadSave(source, string(PlatformPC))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if _, err := engine.SetCharacterRunes(
		loaded.SaveSessionID, setRunesTestSlot, 500, "0"); err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}

	held := engine.sessions[loaded.SaveSessionID]
	point := held.session.undo
	if point == nil {
		t.Fatal("the first mutation recorded no undo point")
	}
	revision := held.session.revisionString()
	dirty := held.session.dirty
	held.snapshot.data = held.snapshot.data[:1]

	ran := false
	_, err = engine.commitCharacterRevision(
		loaded.SaveSessionID, opSetCharacterRunes, setRunesTestSlot,
		func(*loadedSave) error {
			ran = true
			return nil
		})
	if err == nil {
		t.Fatal("commitCharacterRevision accepted a mutation without an undo point")
	}
	if !strings.Contains(err.Error(), "cannot prepare an undo point") {
		t.Fatalf("error = %v, want the undo-point preparation failure", err)
	}
	if ran {
		t.Error("the mutation callback ran after the undo point could not be captured")
	}
	if got := held.session.revisionString(); got != revision {
		t.Errorf("saveRevision = %s, want the unchanged %s", got, revision)
	}
	if held.session.dirty != dirty {
		t.Errorf("dirty = %t, want the unchanged %t", held.session.dirty, dirty)
	}
	if held.session.undo != point {
		t.Error("the earlier undo point was replaced by a refused mutation")
	}
}
