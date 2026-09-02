package saveengine

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// This file owns the shared mutation contract of SaveForge 2.0: the stable kind
// of every save-session mutation, the closed vocabulary of read scopes such a
// mutation invalidates, the unpredictable identifier of one concrete execution,
// and the receipt that carries all of it back to the caller.
//
// Two identifiers exist and they are not interchangeable:
//
//   - operationKind is the stable kind of the mutation. It is exactly the
//     EndpointID of the public endpoint that initiated the change, it is the
//     same value for every execution of that endpoint, and it is what an undo
//     point reports about the mutation it would revert.
//   - operationID names one concrete execution. Two successful executions of the
//     same operationKind always carry different operationIDs. It is opaque,
//     unpredictable and never derived from the kind, the revision, an index or a
//     clock, so no caller can construct the identifier of an execution it never
//     performed.

// Operation kinds of the save-session mutations. They are the EndpointIDs of the
// public endpoints in snake_case, so a shared writer must never report the wrong
// one.
//
// setOwnedWeaponGameID, setCharacterAppearance and setNetworkSettings serve more
// than one public endpoint, so they receive their operation kind as a parameter.
// All three are private writers, and every public entry point picks one of these
// constants itself: no caller outside this package can name an operation.
const (
	kindAddItemToInventory          = "add_item_to_inventory"
	kindAddItemToStorage            = "add_item_to_storage"
	kindApplyAppearancePreset       = "apply_appearance_preset"
	kindApplyBuildTemplate          = "apply_build_template"
	kindApplyFavoritePreset         = "apply_favorite_preset"
	kindApplyNetworkPreset          = "apply_network_preset"
	kindApplyRepairs                = "apply_repairs"
	kindCloneCharacter              = "clone_character"
	kindDeleteCharacter             = "delete_character"
	kindDeleteFavoritePreset        = "delete_favorite_preset"
	kindLockAllSpectralSteedAttires = "lock_all_spectral_steed_attires"
	kindMoveOwnedItemToInventory    = "move_owned_item_to_inventory"
	kindMoveOwnedItemToStorage      = "move_owned_item_to_storage"
	kindRemoveOwnedItem             = "remove_owned_item"
	kindSetBellBearingUnlocked      = "set_bell_bearing_unlocked"
	kindSetBossDefeated             = "set_boss_defeated"
	kindSetCharacterActive          = "set_character_active"
	kindSetCharacterAppearance      = "set_character_appearance"
	kindSetCharacterGender          = "set_character_gender"
	kindSetCharacterName            = "set_character_name"
	kindSetCharacterRunes           = "set_character_runes"
	kindSetCharacterStartingClass   = "set_character_starting_class"
	kindSetCharacterStats           = "set_character_stats"
	kindSetColosseumUnlocked        = "set_colosseum_unlocked"
	kindSetCookbookUnlocked         = "set_cookbook_unlocked"
	kindSetEquippedArmaments        = "set_equipped_armaments"
	kindSetEquippedArmor            = "set_equipped_armor"
	kindSetEquippedSpells           = "set_equipped_spells"
	kindSetEquippedTalismans        = "set_equipped_talismans"
	kindSetFavoritePreset           = "set_favorite_preset"
	kindSetFogOfWarRemoved          = "set_fog_of_war_removed"
	kindSetGestureUnlocked          = "set_gesture_unlocked"
	kindSetGraceVisited             = "set_grace_visited"
	kindSetInventoryOrder           = "set_inventory_order"
	kindSetMapRegionRevealed        = "set_map_region_revealed"
	kindSetNetworkSettings          = "set_network_settings"
	kindSetOwnedItemQuantity        = "set_owned_item_quantity"
	kindSetPhysickMixture           = "set_physick_mixture"
	kindSetPouchItems               = "set_pouch_items"
	kindSetQuestStep                = "set_quest_step"
	kindSetQuickItems               = "set_quick_items"
	kindSetRegionUnlocked           = "set_region_unlocked"
	kindSetSaveAccountID            = "set_save_account_id"
	kindSetSpectralSteedAttire      = "set_spectral_steed_attire"
	kindSetSpiritAshUpgradeLevel    = "set_spirit_ash_upgrade_level"
	kindSetStorageOrder             = "set_storage_order"
	kindSetSummoningPoolActivated   = "set_summoning_pool_activated"
	kindSetTutorialUnlocked         = "set_tutorial_unlocked"
	kindSetWeaponAshOfWar           = "set_weapon_ash_of_war"
	kindSetWeaponInfusion           = "set_weapon_infusion"
	kindSetWeaponUpgradeLevel       = "set_weapon_upgrade_level"
	kindSetWhetbladeUnlocked        = "set_whetblade_unlocked"
	kindUndoCharacterChanges        = "undo_character_changes"
	kindWriteSave                   = "write_save"
)

// Changed scopes are the closed, stable vocabulary of the backend read surfaces
// one committed mutation invalidates. They are a public contract, so they are
// named here once and never assembled from raw strings at a setter.
//
// Every scope owns concrete getters:
//
//   - ScopeSaveSession: GetLoadedSave and GetUndoState, which report the
//     revision, the unsaved-changes flag and the single undo point.
//   - ScopeCharacterList: GetSaveCharacters.
//   - ScopeCharacterProfile: GetCharacterProfile.
//   - ScopeCharacterStats: GetCharacterStats.
//   - ScopeCharacterAppearance: GetCharacterAppearance.
//   - ScopeInventory: GetInventory, GetOwnedItem and GetItemCapacity.
//   - ScopeStorage: GetStorage.
//   - ScopeEquipmentLoadout: GetEquipment, GetCharacterLoadout, GetEquippedSpells,
//     GetPhysickMixture, GetPouchItems and GetQuickItems.
//   - ScopeWorldFlags: every World getter answered from event flags.
//   - ScopeNetwork: GetNetworkSettings.
//   - ScopeFavorites: GetFavoritePresets.
//   - ScopeDiagnosticsReport: GetSaveValidationReport, GetRepairPlan and
//     GetDiagnosticLog, whose results are pinned to one saveRevision.
//
// There is deliberately no "all" and no catch-all scope.
const (
	ScopeSaveSession         = "save.session"
	ScopeCharacterList       = "character.list"
	ScopeCharacterProfile    = "character.profile"
	ScopeCharacterStats      = "character.stats"
	ScopeCharacterAppearance = "character.appearance"
	ScopeInventory           = "inventory"
	ScopeStorage             = "storage"
	ScopeEquipmentLoadout    = "equipment.loadout"
	ScopeWorldFlags          = "world.flags"
	ScopeNetwork             = "network"
	ScopeFavorites           = "favorites"
	ScopeDiagnosticsReport   = "diagnostics.report"
)

// changedScopeOrder is the one canonical order every changedScopes list is
// rendered in. Order comes from this slice and never from map iteration, so the
// same mutation always reports the same sequence.
var changedScopeOrder = []string{
	ScopeSaveSession,
	ScopeCharacterList,
	ScopeCharacterProfile,
	ScopeCharacterStats,
	ScopeCharacterAppearance,
	ScopeInventory,
	ScopeStorage,
	ScopeEquipmentLoadout,
	ScopeWorldFlags,
	ScopeNetwork,
	ScopeFavorites,
	ScopeDiagnosticsReport,
}

// universalChangedScopes are invalidated by every committed save-session
// mutation without exception: the revision and the unsaved-changes flag move,
// and a validation report pinned to the previous revision stops describing the
// session.
//
// They are added by changedScopesForMutationKind instead of being repeated in
// every row below, so a row states only what is specific to its mutation.
var universalChangedScopes = []string{ScopeSaveSession, ScopeDiagnosticsReport}

// domainChangedScopes maps one operation kind to the read scopes it invalidates
// beyond the universal ones. A kind absent from this map is not a save-session
// mutation and is refused before anything is touched.
//
// ponytail: one static table, not a per-mutation callback. A mutation's scope
// set is a property of the endpoint, not of its arguments; a row is a superset
// only where the endpoint itself accepts either container, which is exactly what
// the caller has to refresh anyway.
var domainChangedScopes = map[string][]string{
	// Inventory and Storage records.
	kindAddItemToInventory:       {ScopeInventory},
	kindAddItemToStorage:         {ScopeStorage},
	kindMoveOwnedItemToInventory: {ScopeInventory, ScopeStorage},
	kindMoveOwnedItemToStorage:   {ScopeInventory, ScopeStorage},
	// RemoveOwnedItem and SetOwnedItemQuantity address either container through
	// one opaque OwnedItemID, so both are invalidated. A removal cannot target a
	// record an Equipment, Quick Item or Pouch slot references, so it leaves the
	// loadout alone; a quantity change can, and GetCharacterLoadout reports that
	// quantity for its Quick Item and Pouch positions.
	kindRemoveOwnedItem:      {ScopeInventory, ScopeStorage},
	kindSetOwnedItemQuantity: {ScopeInventory, ScopeStorage, ScopeEquipmentLoadout},
	kindSetInventoryOrder:    {ScopeInventory},
	kindSetStorageOrder:      {ScopeStorage},
	// The weapon writers change the game ID of an owned record and keep the
	// equipped references of that record coherent. Each of them addresses one
	// common record through an opaque OwnedItemID, so the record can sit in
	// Inventory or in Storage and both containers are invalidated.
	kindSetWeaponAshOfWar:        {ScopeInventory, ScopeStorage, ScopeEquipmentLoadout},
	kindSetWeaponInfusion:        {ScopeInventory, ScopeStorage, ScopeEquipmentLoadout},
	kindSetWeaponUpgradeLevel:    {ScopeInventory, ScopeStorage, ScopeEquipmentLoadout},
	kindSetSpiritAshUpgradeLevel: {ScopeInventory, ScopeStorage, ScopeEquipmentLoadout},

	// Equipment writes only the loadout fields of the slot; no owned record moves.
	kindSetEquippedArmaments: {ScopeEquipmentLoadout},
	kindSetEquippedArmor:     {ScopeEquipmentLoadout},
	kindSetEquippedTalismans: {ScopeEquipmentLoadout},
	kindSetEquippedSpells:    {ScopeEquipmentLoadout},
	kindSetPhysickMixture:    {ScopeEquipmentLoadout},
	kindSetPouchItems:        {ScopeEquipmentLoadout},
	kindSetQuickItems:        {ScopeEquipmentLoadout},

	// Character identity and build. Name and level are part of the character
	// list summary as well as of the profile.
	kindSetCharacterName:          {ScopeCharacterList, ScopeCharacterProfile},
	kindSetCharacterStats:         {ScopeCharacterList, ScopeCharacterProfile, ScopeCharacterStats},
	kindSetCharacterStartingClass: {ScopeCharacterList, ScopeCharacterProfile, ScopeCharacterStats},
	// Gender is part of both the appearance block and the profile.
	kindSetCharacterAppearance: {ScopeCharacterProfile, ScopeCharacterAppearance},
	kindSetCharacterGender:     {ScopeCharacterProfile, ScopeCharacterAppearance},
	kindApplyAppearancePreset:  {ScopeCharacterProfile, ScopeCharacterAppearance},
	kindApplyFavoritePreset:    {ScopeCharacterProfile, ScopeCharacterAppearance},
	// Held runes have no getter today, so this mutation invalidates the universal
	// scopes only. It gains a domain scope in the same task that exposes runes.
	kindSetCharacterRunes: nil,
	kindApplyBuildTemplate: {
		ScopeCharacterList, ScopeCharacterProfile, ScopeCharacterStats, ScopeEquipmentLoadout,
	},

	// Slot-wide mutations. Activity gates every per-character getter, and cloning
	// or deleting rewrites the complete slot.
	kindSetCharacterActive: {
		ScopeCharacterList, ScopeCharacterProfile, ScopeCharacterStats, ScopeCharacterAppearance,
		ScopeInventory, ScopeStorage, ScopeEquipmentLoadout, ScopeWorldFlags,
	},
	kindCloneCharacter: {
		ScopeCharacterList, ScopeCharacterProfile, ScopeCharacterStats, ScopeCharacterAppearance,
		ScopeInventory, ScopeStorage, ScopeEquipmentLoadout, ScopeWorldFlags,
	},
	kindDeleteCharacter: {
		ScopeCharacterList, ScopeCharacterProfile, ScopeCharacterStats, ScopeCharacterAppearance,
		ScopeInventory, ScopeStorage, ScopeEquipmentLoadout, ScopeWorldFlags,
	},

	// World mutations that write event flags, gesture records or the unlocked
	// region list and nothing else. Every one of them is answered by a World
	// getter, and none of them creates, removes or re-quantifies an owned record.
	kindSetBossDefeated:           {ScopeWorldFlags},
	kindSetColosseumUnlocked:      {ScopeWorldFlags},
	kindSetCookbookUnlocked:       {ScopeWorldFlags},
	kindSetFogOfWarRemoved:        {ScopeWorldFlags},
	kindSetGestureUnlocked:        {ScopeWorldFlags},
	kindSetGraceVisited:           {ScopeWorldFlags},
	kindSetQuestStep:              {ScopeWorldFlags},
	kindSetRegionUnlocked:         {ScopeWorldFlags},
	kindSetSummoningPoolActivated: {ScopeWorldFlags},
	kindSetTutorialUnlocked:       {ScopeWorldFlags},
	// Selecting one Spectral Steed appearance reads the Inventory to prove the
	// item is held and then writes appearance event flags only. It never creates
	// or removes a record, so Inventory is data it depends on, not data it
	// changes, and a scope list is what the endpoint writes.
	kindSetSpectralSteedAttire: {ScopeWorldFlags},

	// World mutations that keep a companion item in step with their flags, so
	// they write InventoryHeld records and the section counters beside the flag
	// byte. None of them can empty a row an Equipment, Quick Item or Pouch slot
	// references: the shared removal planner refuses a referenced record, so the
	// loadout is never invalidated.
	kindSetMapRegionRevealed:        {ScopeInventory, ScopeWorldFlags},
	kindSetWhetbladeUnlocked:        {ScopeInventory, ScopeWorldFlags},
	kindLockAllSpectralSteedAttires: {ScopeInventory, ScopeWorldFlags},
	// Handing a Bell Bearing in consumes every matching record, and it searches
	// Inventory as well as Storage, so both containers are invalidated.
	kindSetBellBearingUnlocked: {ScopeInventory, ScopeStorage, ScopeWorldFlags},

	// Session-wide surfaces stored outside a character slot.
	kindSetNetworkSettings:   {ScopeNetwork},
	kindApplyNetworkPreset:   {ScopeNetwork},
	kindSetFavoritePreset:    {ScopeFavorites},
	kindDeleteFavoritePreset: {ScopeFavorites},
	// The account identifier reaches no getter; it is private account data.
	kindSetSaveAccountID: nil,

	// Repairs remove owned records, correct quantities and rewrite statistics
	// through the same writers as the individual endpoints.
	kindApplyRepairs: {
		ScopeCharacterList, ScopeCharacterProfile, ScopeCharacterStats,
		ScopeInventory, ScopeStorage, ScopeEquipmentLoadout,
	},

	// WriteSave persists the snapshot and clears the unsaved-changes flag. It
	// changes no domain value, so it invalidates the universal scopes only.
	kindWriteSave: nil,
	// Undo restores the ranges of one character mutation. Its concrete scopes are
	// the scopes of the mutation it reverts, resolved at commit time on top of
	// this baseline.
	kindUndoCharacterChanges: nil,
}

// MutationKinds returns every registered save-session mutation kind in
// lexical order. It exists so the endpoint layer can assert that its
// implemented mutation EndpointIDs and these kinds never drift apart.
func MutationKinds() []string {
	kinds := make([]string, 0, len(domainChangedScopes))
	for kind := range domainChangedScopes {
		kinds = append(kinds, kind)
	}
	sortStrings(kinds)
	return kinds
}

// ChangedScopesForMutationKind returns the exact, canonically ordered scopes one
// registered mutation kind invalidates. An unknown or empty kind is an error.
func ChangedScopesForMutationKind(operationKind string) ([]string, error) {
	return changedScopesForMutationKind(operationKind)
}

func changedScopesForMutationKind(operationKind string, extra ...string) ([]string, error) {
	if operationKind == "" {
		return nil, errors.New("operationKind is required")
	}
	domain, registered := domainChangedScopes[operationKind]
	if !registered {
		return nil, fmt.Errorf("unknown operationKind %q", operationKind)
	}

	selected := make(map[string]bool, len(changedScopeOrder))
	for _, scope := range universalChangedScopes {
		selected[scope] = true
	}
	for _, scope := range domain {
		selected[scope] = true
	}
	for _, scope := range extra {
		selected[scope] = true
	}

	// The result is assembled from the canonical order, never from the map, so it
	// carries no duplicate, no empty value and no iteration-dependent sequence.
	scopes := make([]string, 0, len(selected))
	for _, scope := range changedScopeOrder {
		if selected[scope] {
			scopes = append(scopes, scope)
		}
	}
	return scopes, nil
}

// sortStrings is an insertion sort over a list this package only ever builds
// from its own static table.
//
// ponytail: avoids importing sort for one ordering of a few dozen constants.
func sortStrings(values []string) {
	for index := 1; index < len(values); index++ {
		value := values[index]
		position := index
		for position > 0 && values[position-1] > value {
			values[position] = values[position-1]
			position--
		}
		values[position] = value
	}
}

// MutationReceipt is the shared, public description of one committed save
// mutation. It is produced in exactly one place, the session commit path, and
// describes the revision that path created.
//
// OperationID names this single execution; OperationKind names the stable kind
// of mutation it was. A rejected mutation returns the zero receipt and never
// exposes the identifier it had prepared.
type MutationReceipt struct {
	OperationID   string   `json:"operationID"`
	OperationKind string   `json:"operationKind"`
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	ChangedScopes []string `json:"changedScopes"`
}

// operationIDPrefix marks an identifier minted for one mutation execution. It
// exists so such an identifier can never be mistaken for a saveSessionID, an
// OwnedItemID or an undo token.
const operationIDPrefix = "op-"

// pendingMutation is everything the receipt of a mutation needs that can be
// prepared before the first byte changes. It is created by prepareMutation and
// turned into a receipt only after a revision has actually been committed.
type pendingMutation struct {
	operationID   string
	operationKind string
	changedScopes []string
}

// prepareMutation validates operationKind, resolves its changed scopes and mints
// the identifier of this execution. Every one of those steps can fail, and all of
// them run before the mutation touches the session, so a failure here rejects the
// operation without a snapshot, revision, dirty flag, undo point or identity
// registry having moved.
//
// extraScopes are additional registered scopes a special path resolves at commit
// time; the undo path uses them to report the scopes of the mutation it reverts.
//
// The caller must already hold Engine.mutex when it is about to mutate; the
// preparation itself touches no session state.
func (engine *Engine) prepareMutation(
	operationKind string,
	extraScopes ...string,
) (pendingMutation, error) {
	scopes, err := changedScopesForMutationKind(operationKind, extraScopes...)
	if err != nil {
		return pendingMutation{}, err
	}
	operationID, err := engine.mintOperationID()
	if err != nil {
		return pendingMutation{}, err
	}
	return pendingMutation{
		operationID:   operationID,
		operationKind: operationKind,
		changedScopes: scopes,
	}, nil
}

// receipt completes a prepared mutation with the session and revision it
// committed. It cannot fail: everything fallible happened in prepareMutation,
// so no error can appear after the revision is irreversibly advanced.
func (pending pendingMutation) receipt(saveSessionID string, saveRevision string) MutationReceipt {
	// The scope slice is copied so a caller can never alias, reorder or extend the
	// list a later receipt would reuse.
	scopes := make([]string, len(pending.changedScopes))
	copy(scopes, pending.changedScopes)
	return MutationReceipt{
		OperationID:   pending.operationID,
		OperationKind: pending.operationKind,
		SaveSessionID: saveSessionID,
		SaveRevision:  saveRevision,
		ChangedScopes: scopes,
	}
}

// operationIDMintAttempts bounds how often one mutation may ask the generator
// for an identifier that is not already reserved. Random 128-bit values collide
// with negligible probability, so a second attempt is already unreachable in
// practice; the bound exists so a generator that keeps returning the same value
// fails fast and closed instead of looping forever.
const operationIDMintAttempts = 8

// mintOperationID reserves the identifier of one mutation execution. It is
// random rather than derived, so no caller can construct the identifier of an
// execution it never performed, and it is checked against every identifier this
// engine already reserved, so "two executions never share one" is a hard
// guarantee rather than a probability.
//
// The caller holds Engine.mutex, so the check and the reservation are one
// atomic step, and both happen before the mutation touches the session.
//
// ponytail: an in-memory set, one entry per reserved identifier and never
// pruned. That is ~100 bytes per mutation of one running engine, so a session
// would need millions of mutations to matter. Retention becomes a question only
// when a persistent operation log exists, and that log is the thing that would
// own it.
func (engine *Engine) mintOperationID() (string, error) {
	generate := newOperationID
	if engine.newOperationID != nil {
		generate = engine.newOperationID
	}
	for attempt := 0; attempt < operationIDMintAttempts; attempt++ {
		operationID, err := generate()
		if err != nil {
			return "", err
		}
		if engine.reservedOperationIDs[operationID] {
			continue
		}
		if engine.reservedOperationIDs == nil {
			engine.reservedOperationIDs = make(map[string]bool)
		}
		engine.reservedOperationIDs[operationID] = true
		return operationID, nil
	}
	return "", fmt.Errorf(
		"cannot create a unique mutation operation identifier after %d attempts",
		operationIDMintAttempts)
}

func newOperationID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("cannot create mutation operation identifier: %w", err)
	}
	return operationIDPrefix + hex.EncodeToString(raw), nil
}
