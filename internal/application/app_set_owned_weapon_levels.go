package application

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/editor"
)

// SetOwnedWeaponLevels sets every OWNED, editable weapon in the character's
// Inventory to the given upgrade levels: standard weapons (MaxUpgrade==25)
// to upgrade25, special/somber weapons and somber shields (MaxUpgrade==10)
// to upgrade10. It is an absolute SET — a weapon's level may go UP or DOWN —
// and is clamped per item to that item's MaxUpgrade. Storage, Spirit Ashes,
// non-weapons, non-upgradeable weapons and pass-through records are never
// touched (editor.SetOwnedWeaponLevels walks InventoryItems only and filters
// on IsWeapon + exact MaxUpgrade).
//
// The whole batch is atomic: it runs through the standard, safe workspace
// path (StartInventoryEditSession → editor.SetOwnedWeaponLevels →
// SaveInventoryWorkspaceChanges), which commits every weapon or nothing and
// pushes exactly ONE undo snapshot for the batch. On any error the transient
// session is discarded and slot.Data is left untouched — no partial write.
//
// Active-session contract: if the character already owns an inventory edit
// session (the Inventory/Equipment editor is open, possibly with pending
// edits), the call is REJECTED rather than silently replacing that session
// and losing the user's pending work. This mirrors the schema-v2 template
// apply gate in app_templates_v2_apply.go. The user must close their session
// first, which is a deliberate, explicit decision.
//
// Returns the number of weapons changed.
func (a *App) SetOwnedWeaponLevels(charIdx, upgrade25, upgrade10 int) (int, error) {
	// Collision-safe start: rejects (evicting nothing) if a session is
	// already active for this character, atomically under lifecycleMu so
	// there is no TOCTOU window in which a concurrently created session
	// could be clobbered. Never call the replacing StartInventoryEditSession
	// here — its replace semantics would silently discard a peer's session.
	snap, err := a.startInventoryEditSessionIfNone(charIdx)
	if err != nil {
		return 0, fmt.Errorf("SetOwnedWeaponLevels: %w", err)
	}
	sessionID := snap.SessionID

	// Apply the batch RAM-only on the session workspace.
	var changed int
	{
		sess, err := a.acquireSession(sessionID)
		if err != nil {
			return 0, fmt.Errorf("SetOwnedWeaponLevels: %w", err)
		}
		applyErr := a.journalWorkspaceMutation(actionGameItemsWorkspaceWeapon, sess.CharacterIndex, &sess.Workspace, func(ws *editor.InventoryWorkspaceSnapshot) error {
			n, e := editor.SetOwnedWeaponLevels(ws, upgrade25, upgrade10)
			changed = n
			return e
		})
		sess.Unlock()
		if applyErr != nil {
			// Partial RAM edits only — nothing committed to slot.Data.
			// Drop the transient session so none is left behind.
			_ = a.DiscardInventoryEditSession(sessionID)
			return 0, fmt.Errorf("SetOwnedWeaponLevels: %w", applyErr)
		}
	}

	// No matching weapons → nothing to commit. Discard the transient
	// session so no empty undo snapshot is pushed and no dangling session
	// is left for the UI.
	if changed == 0 {
		_ = a.DiscardInventoryEditSession(sessionID)
		return 0, nil
	}

	// Commit the whole batch as one workspace save (one undo snapshot).
	if _, err := a.SaveInventoryWorkspaceChanges(sessionID); err != nil {
		_ = a.DiscardInventoryEditSession(sessionID)
		return 0, fmt.Errorf("SetOwnedWeaponLevels: workspace save: %w", err)
	}

	// Batch committed; drop the transient session (this endpoint owns its
	// whole lifecycle — the UI never holds this session's ID).
	_ = a.DiscardInventoryEditSession(sessionID)
	return changed, nil
}
