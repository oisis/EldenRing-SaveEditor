package main

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/editor"
)

// registerLiveSession publishes a real InventoryEditSession object into the
// registry for charIdx and returns it, so tests can assert it survives a
// rejected collision-safe start (identity, not just presence). A non-nil
// (empty) save is installed so the start variants pass their save!=nil /
// range guards and actually reach the session-existence check.
func registerLiveSession(app *App, charIdx int) *editor.InventoryEditSession {
	app.save = &core.SaveFile{}
	sess := &editor.InventoryEditSession{ID: editor.NewSessionID(), CharacterIndex: charIdx}
	app.editSessions[sess.ID] = sess
	app.editSessionByChar[charIdx] = sess.ID
	return sess
}

// TestStartInventoryEditSessionIfNone_RejectsAndKeepsSession exercises the
// collision-safe start variant DIRECTLY (not via the old map pre-check): a
// really-registered session must be rejected and left byte-for-byte in the
// registry — same object, same mapping, not closed.
func TestStartInventoryEditSessionIfNone_RejectsAndKeepsSession(t *testing.T) {
	app := NewApp()
	const idx = 0
	sess := registerLiveSession(app, idx)

	_, err := app.startInventoryEditSessionIfNone(idx)
	if err == nil {
		t.Fatal("expected reject when a session is active, got nil")
	}
	if !strings.Contains(err.Error(), "close the inventory edit session") {
		t.Errorf("error = %q, want it to mention closing the active session", err.Error())
	}
	// The exact session object must still be registered and live.
	if got := app.editSessions[sess.ID]; got != sess {
		t.Error("collision-safe start removed or replaced the live session object")
	}
	if app.editSessionByChar[idx] != sess.ID {
		t.Error("collision-safe start rewrote editSessionByChar mapping")
	}
	if sess.IsClosed() {
		t.Error("collision-safe start closed the live session")
	}
}

// TestSetOwnedWeaponLevels_RejectsActiveSession is fixture-independent: the
// endpoint (through the collision-safe start) refuses to run while a real
// session is registered, clobbers nothing, and returns changed==0.
func TestSetOwnedWeaponLevels_RejectsActiveSession(t *testing.T) {
	app := NewApp()
	const idx = 0
	sess := registerLiveSession(app, idx)

	changed, err := app.SetOwnedWeaponLevels(idx, 25, 10)
	if err == nil {
		t.Fatal("expected error when a session is active, got nil")
	}
	if changed != 0 {
		t.Errorf("changed = %d, want 0 on rejection", changed)
	}
	if !strings.Contains(err.Error(), "close the inventory edit session") {
		t.Errorf("error = %q, want it to mention closing the active session", err.Error())
	}
	// The pre-existing session must be intact — same object, same mapping.
	if got := app.editSessions[sess.ID]; got != sess {
		t.Error("the pre-existing session object was removed or replaced")
	}
	if app.editSessionByChar[idx] != sess.ID {
		t.Error("the pre-existing session mapping was rewritten")
	}
}

// invWeaponLevels returns UID -> CurrentUpgrade for editable inventory
// weapons in the character's current on-disk state, via a throwaway
// session that is discarded before returning (so it never blocks a
// subsequent SetOwnedWeaponLevels call).
func invWeaponLevels(t *testing.T, app *App, idx int) (map[string]editor.EditableItem, string) {
	t.Helper()
	snap, err := app.StartInventoryEditSession(idx)
	if err != nil {
		t.Fatalf("StartInventoryEditSession: %v", err)
	}
	out := make(map[string]editor.EditableItem, len(snap.InventoryItems))
	for _, it := range snap.InventoryItems {
		if it.IsWeapon {
			out[it.UID] = it
		}
	}
	if err := app.DiscardInventoryEditSession(snap.SessionID); err != nil {
		t.Fatalf("DiscardInventoryEditSession: %v", err)
	}
	return out, snap.SessionID
}

// TestSetOwnedWeaponLevels_BatchSetAndUndo drives the full path on a real
// save fixture (skips when absent): every editable inventory weapon with
// MaxUpgrade 25/10 is SET to the slider levels, infusion is preserved,
// exactly one undo snapshot is pushed, and one revert restores the batch.
func TestSetOwnedWeaponLevels_BatchSetAndUndo(t *testing.T) {
	app, idx := realSaveAppForSave(t)

	before, _ := invWeaponLevels(t, app, idx)
	if len(before) == 0 {
		t.Skip("fixture has no editable inventory weapons")
	}

	depth0 := app.GetUndoDepth(idx)

	changed, err := app.SetOwnedWeaponLevels(idx, 25, 10)
	if err != nil {
		t.Fatalf("SetOwnedWeaponLevels: %v", err)
	}
	if changed == 0 {
		t.Skip("fixture has no MaxUpgrade 25/10 weapons to change")
	}

	// Exactly one undo snapshot for the whole batch.
	depth1 := app.GetUndoDepth(idx)
	if depth1 != depth0+1 {
		t.Fatalf("undo depth = %d, want %d (exactly one snapshot for the batch)", depth1, depth0+1)
	}

	// The endpoint owns its transient session end-to-end — none is left.
	if active, _ := app.GetActiveInventoryEditSessionForCharacter(idx); active.Active {
		t.Error("a session was left registered after the batch committed")
	}

	// Re-run with identical targets: every weapon is already there, so this
	// is a pure no-op — changed==0, NO new undo snapshot, session cleaned.
	noop, err := app.SetOwnedWeaponLevels(idx, 25, 10)
	if err != nil {
		t.Fatalf("SetOwnedWeaponLevels (no-op): %v", err)
	}
	if noop != 0 {
		t.Errorf("no-op re-run changed = %d, want 0", noop)
	}
	if got := app.GetUndoDepth(idx); got != depth1 {
		t.Errorf("no-op re-run pushed an undo: depth %d, want %d", got, depth1)
	}
	if active, _ := app.GetActiveInventoryEditSessionForCharacter(idx); active.Active {
		t.Error("no-op re-run left a session registered")
	}

	after, _ := invWeaponLevels(t, app, idx)
	changedCount := 0
	for uid, b := range before {
		a, ok := after[uid]
		if !ok {
			continue // handle may re-key on save; covered by count check below
		}
		switch b.MaxUpgrade {
		case 25:
			if a.CurrentUpgrade != 25 {
				t.Errorf("%s (%s): +%d, want +25", b.Name, uid, a.CurrentUpgrade)
			}
		case 10:
			if a.CurrentUpgrade != 10 {
				t.Errorf("%s (%s): +%d, want +10", b.Name, uid, a.CurrentUpgrade)
			}
		default:
			if a.CurrentUpgrade != b.CurrentUpgrade {
				t.Errorf("%s (%s): MaxUpgrade %d weapon changed +%d -> +%d", b.Name, uid, b.MaxUpgrade, b.CurrentUpgrade, a.CurrentUpgrade)
			}
			continue
		}
		// Infusion must be preserved across the SET.
		if a.InfusionName != b.InfusionName {
			t.Errorf("%s (%s): infusion %q -> %q", b.Name, uid, b.InfusionName, a.InfusionName)
		}
		changedCount++
	}
	if changedCount == 0 {
		t.Fatal("no eligible weapon was verified as changed")
	}

	// One revert restores the entire batch.
	if err := app.RevertSlot(idx); err != nil {
		t.Fatalf("RevertSlot: %v", err)
	}
	restored, _ := invWeaponLevels(t, app, idx)
	for uid, b := range before {
		if r, ok := restored[uid]; ok && r.CurrentUpgrade != b.CurrentUpgrade {
			t.Errorf("%s (%s): after revert +%d, want +%d (batch not restored)", b.Name, uid, r.CurrentUpgrade, b.CurrentUpgrade)
		}
	}
}
