package application

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/db"
	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

// spectralSteedAttirePlatform is the only platform whose 6700-6703 contract is
// confirmed from native saves. PS4 mutation stays fail-closed until the same
// evidence exists there.
const spectralSteedAttirePlatform = "PC"

// spectralSteedItemOwned reports whether the attire key item is physically in
// the character's inventory. Both the editor-computed handle and the raw item ID
// route through db.HandleToItemID, so a game-placed record matches too.
func spectralSteedItemOwned(slot *core.SaveSlot, itemID uint32) bool {
	has := func(items []core.InventoryItem) bool {
		for _, it := range items {
			if it.GaItemHandle == 0 || it.GaItemHandle == core.GaHandleEmpty || it.Quantity == 0 {
				continue
			}
			if db.HandleToItemID(it.GaItemHandle) == itemID {
				return true
			}
		}
		return false
	}
	return has(slot.Inventory.KeyItems) || has(slot.Inventory.CommonItems)
}

// GetSpectralSteedAttire returns all four Torrent appearances with per-attire
// item ownership and the currently active appearance.
//
// It never mutates and never repairs: when all four flags are readable and
// cleared (a save that predates Regulation 1.17) or when two or more are set,
// Status reports the ambiguity and ActiveID stays 0. An unreadable flag region
// is an error, never "legacy". The default appearance is always listed and is
// always available — it needs no inventory item.
func (a *App) GetSpectralSteedAttire(slotIndex int) (db.SpectralSteedAttireState, error) {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()
	if a.save == nil {
		return db.SpectralSteedAttireState{}, fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return db.SpectralSteedAttireState{}, fmt.Errorf("invalid slot index")
	}
	a.slotMu[slotIndex].Lock()
	defer a.slotMu[slotIndex].Unlock()

	slot := &a.save.Slots[slotIndex]
	if slot.EventFlagsOffset <= 0 || slot.EventFlagsOffset >= len(slot.Data) {
		return db.SpectralSteedAttireState{}, fmt.Errorf("event flags offset not computed for slot %d", slotIndex)
	}
	flags := slot.Data[slot.EventFlagsOffset:]

	entries := db.GetAllSpectralSteedAttires()
	state := db.SpectralSteedAttireState{Status: db.SpectralSteedAttireLegacy}
	setCount := 0
	for i := range entries {
		if entries[i].ItemID == 0 {
			entries[i].Owned = true
		} else {
			entries[i].Owned = spectralSteedItemOwned(slot, entries[i].ItemID)
		}
		on, err := db.GetEventFlag(flags, entries[i].ID)
		if err != nil {
			return db.SpectralSteedAttireState{}, fmt.Errorf("read spectral steed attire flag %d: %w", entries[i].ID, err)
		}
		if on {
			setCount++
			state.ActiveID = entries[i].ID
		}
	}
	switch {
	case setCount == 1:
		state.Status = db.SpectralSteedAttireResolved
	case setCount > 1:
		state.Status = db.SpectralSteedAttireConflict
		state.ActiveID = 0
	}
	state.Entries = entries
	return state, nil
}

// SetSpectralSteedAttire activates exactly one Torrent appearance.
//
// Accepts only flags 6700-6703. An attire (6701-6703) additionally requires its
// key item in inventory; the default appearance (6700) requires nothing. The
// method never adds an item — the caller adds it through the shared
// AddItemsToCharacter path first.
//
// All validation runs before the first mutation, and the mutation clears
// 6700-6703 before setting the requested flag, so on success exactly one of the
// four flags is set. A failure mid-write restores the pre-call slot state.
func (a *App) SetSpectralSteedAttire(slotIndex int, flagID uint32) error {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return fmt.Errorf("invalid slot index")
	}
	if a.save.Platform != spectralSteedAttirePlatform {
		return fmt.Errorf("spectral steed attire is not yet verified for %s saves", a.save.Platform)
	}
	attire, ok := data.FindSpectralSteedAttire(flagID)
	if !ok {
		return fmt.Errorf("unknown spectral steed attire")
	}

	a.slotMu[slotIndex].Lock()
	defer a.slotMu[slotIndex].Unlock()

	slot := &a.save.Slots[slotIndex]
	if slot.EventFlagsOffset <= 0 || slot.EventFlagsOffset >= len(slot.Data) {
		return fmt.Errorf("event flags offset not computed for slot %d", slotIndex)
	}
	if attire.ItemID != 0 && !spectralSteedItemOwned(slot, attire.ItemID) {
		return fmt.Errorf("%s requires its key item in inventory", attire.Name)
	}

	a.pushUndoLocked(slotIndex)
	return a.journalUnlockMutation(actionGameItemsSetSpectralSteedAttire, slotIndex, slot, spectralSteedAttireFlagIDs(), func(s *core.SaveSlot) error {
		return applySpectralSteedAttire(s, flagID)
	})
}

// spectralSteedAttireFlagIDs returns the flags the operation owns.
func spectralSteedAttireFlagIDs() []uint32 {
	ids := make([]uint32, 0, len(data.SpectralSteedAttires))
	for _, a := range data.SpectralSteedAttires {
		ids = append(ids, a.FlagID)
	}
	return ids
}

// applySpectralSteedAttire clears the mutually exclusive appearance flags and
// sets the requested one. Shared by the real slot and Debug Mode's clone so a
// planned diff cannot drift from the applied mutation. Caller holds the slot
// lock and has validated EventFlagsOffset and flagID.
//
// ponytail: 6700-6703 currently resolve into a single flag byte, so a partial
// write is not reachable today; the snapshot keeps the no-partial-mutation
// contract true if the flag mapping ever splits them across bytes.
func applySpectralSteedAttire(slot *core.SaveSlot, flagID uint32) error {
	snapshot := core.SnapshotSlot(slot)
	flags := slot.Data[slot.EventFlagsOffset:]
	for _, a := range data.SpectralSteedAttires {
		if err := db.SetEventFlag(flags, a.FlagID, false); err != nil {
			core.RestoreSlot(slot, snapshot)
			return err
		}
	}
	if err := db.SetEventFlag(flags, flagID, true); err != nil {
		core.RestoreSlot(slot, snapshot)
		return err
	}
	return nil
}
