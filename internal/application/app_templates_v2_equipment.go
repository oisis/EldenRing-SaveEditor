package application

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/db"
	"github.com/oisis/EldenRing-SaveForge/backend/editor"
	"github.com/oisis/EldenRing-SaveForge/backend/templates"
)

// equipmentSlotChrAsmIndex maps a canonical slot key to the corresponding
// index inside the 22-entry core.ChrAsmEquipment array. Keys mirror
// templates.EquipmentSlotOrder. talisman5 (index 21) is present for read
// completeness only — it has no native write contract, so it is excluded
// from live-character export and from apply (see equipmentSlotKindForKey,
// which is the single source of truth for the writable slot set).
var equipmentSlotChrAsmIndex = map[string]int{
	"weaponLeftHand1":  0,
	"weaponRightHand1": 1,
	"weaponLeftHand2":  2,
	"weaponRightHand2": 3,
	"weaponLeftHand3":  4,
	"weaponRightHand3": 5,
	"arrows1":          6,
	"bolts1":           7,
	"arrows2":          8,
	"bolts2":           9,
	"armorHead":        12,
	"armorChest":       13,
	"armorArms":        14,
	"armorLegs":        15,
	"talisman1":        17,
	"talisman2":        18,
	"talisman3":        19,
	"talisman4":        20,
	"talisman5":        21,
}

// equipmentSlotKindForKey returns the Phase 7b.0 core writer slot kind
// for a Phase 7b.1 canonical slot key. Both maps are stable allowlists;
// an unknown key returns (0, false).
func equipmentSlotKindForKey(slotKey string) (core.EquipmentSlotKind, bool) {
	switch slotKey {
	case "weaponLeftHand1":
		return core.EquipSlotLeftHandArmament1, true
	case "weaponRightHand1":
		return core.EquipSlotRightHandArmament1, true
	case "weaponLeftHand2":
		return core.EquipSlotLeftHandArmament2, true
	case "weaponRightHand2":
		return core.EquipSlotRightHandArmament2, true
	case "weaponLeftHand3":
		return core.EquipSlotLeftHandArmament3, true
	case "weaponRightHand3":
		return core.EquipSlotRightHandArmament3, true
	case "arrows1":
		return core.EquipSlotArrows1, true
	case "bolts1":
		return core.EquipSlotBolts1, true
	case "arrows2":
		return core.EquipSlotArrows2, true
	case "bolts2":
		return core.EquipSlotBolts2, true
	case "armorHead":
		return core.EquipSlotHead, true
	case "armorChest":
		return core.EquipSlotChest, true
	case "armorArms":
		return core.EquipSlotArms, true
	case "armorLegs":
		return core.EquipSlotLegs, true
	case "talisman1":
		return core.EquipSlotTalisman1, true
	case "talisman2":
		return core.EquipSlotTalisman2, true
	case "talisman3":
		return core.EquipSlotTalisman3, true
	case "talisman4":
		return core.EquipSlotTalisman4, true
	}
	return 0, false
}

// talismanOrdinal returns the zero-based pouch ordinal for a talisman slot
// key (talisman1 → 0 … talisman4 → 3) and whether the key is one of the four
// writable talisman slots. talisman5 has no native write contract and is not
// writable, so it returns (0, false).
func talismanOrdinal(slotKey string) (int, bool) {
	switch slotKey {
	case "talisman1":
		return 0, true
	case "talisman2":
		return 1, true
	case "talisman3":
		return 2, true
	case "talisman4":
		return 3, true
	}
	return 0, false
}

// equipmentSlotEquipClass maps a writable slot key to the equipClass that the
// Equipment-tab reader (isEmptyEquipSlot / normalizeEquipItemID) uses. This is
// the single source of truth for how a raw equipped-armaments value is decoded
// per slot family — the export path must not re-invent it.
func equipmentSlotEquipClass(slotKey string) equipClass {
	switch slotKey {
	case "arrows1", "bolts1", "arrows2", "bolts2":
		return classAmmo
	case "armorHead", "armorChest", "armorArms", "armorLegs":
		return classArmor
	case "talisman1", "talisman2", "talisman3", "talisman4", "talisman5":
		return classTalisman
	default:
		return classHandArmament
	}
}

// buildEquipmentSectionFromEquipped scans the writable ChrAsmEquipment slots
// of one character's equipped-armaments block and returns a
// templates.EquipmentSection whose pointer fields are populated for every
// occupied slot.
//
// equipped is RawEquippedState.Equipped — the 22 direct item-ID values read by
// core.SaveSlot.ReadEquippedState from the equipped-armaments block (past the
// variable-length projectiles section). It is NOT the ChrAsm2 GaItem-handle
// header at EquipItemsIDOffset; that header holds encoded handles the item DB
// cannot resolve and must never be used as the equipment source.
//
// Per slot:
//   - isEmptyEquipSlot decides empty (raw 0 / GaHandleInvalid / Unarmed for a
//     hand slot / a bare-armor ID for an armor slot) — the same contract the
//     Equipment tab uses.
//   - Otherwise normalizeEquipItemID resolves the DB item ID, matched against
//     inventoryItems (to carry upgrade / infusion / AoW metadata) and finally
//     the DB.
//
// Only writable slots (equipmentSlotKindForKey ok — weapons, ammo, armor,
// talisman1..4) are scanned; talisman5 is never exported from a live
// character. Talisman slots past activeTalismanSlots stay nil (the source has
// not unlocked them) — they are never emitted as clears.
//
// emitEmptyAsClear selects the export semantics for an empty writable slot:
//   - false → the slot is omitted (occupied-only export); a fully-empty
//     loadout yields nil.
//   - true → the slot is emitted as an explicit clear (EquipmentItemRef with
//     BaseItemID == 0), so applying a "full loadout" template removes whatever
//     gear the target currently equips there rather than leaving stale
//     equipment behind.
func buildEquipmentSectionFromEquipped(equipped [core.ChrAsmFieldCount]uint32, inventoryItems []editor.EditableItem, activeTalismanSlots int, emitEmptyAsClear bool) *templates.EquipmentSection {
	// Index editable inventory by the equipped-form encoded value so we
	// can do a single O(1) lookup per slot. Weapons/armor encode as
	// `ItemID | 0x80000000`; ammo / talismans (goods-like) encode as the
	// bare ItemID.
	byEquipped := make(map[uint32]*editor.EditableItem, len(inventoryItems))
	for i := range inventoryItems {
		it := &inventoryItems[i]
		byEquipped[it.ItemID|core.ItemTypeWeapon] = it
		byEquipped[it.ItemID] = it
	}

	out := &templates.EquipmentSection{}
	any := false
	for _, slotKey := range templates.EquipmentSlotOrder {
		// equipmentSlotKindForKey is the single source of truth for the
		// writable / exportable slot set: weapons, ammo, armor, and
		// talisman1..4. talisman5 has no native write contract, so it is
		// never exported from a live character (stays nil in the section).
		if _, ok := equipmentSlotKindForKey(slotKey); !ok {
			continue
		}
		// Talisman slots the source has not unlocked stay nil — the source
		// character never had that pouch capacity, so there is nothing to
		// clear on it.
		if ordinal, isTal := talismanOrdinal(slotKey); isTal && ordinal >= activeTalismanSlots {
			continue
		}
		class := equipmentSlotEquipClass(slotKey)
		raw := equipped[equipmentSlotChrAsmIndex[slotKey]]

		if isEmptyEquipSlot(raw, class) {
			if emitEmptyAsClear {
				templates.SetEquipmentSlotRef(out, slotKey, &templates.EquipmentItemRef{})
				any = true
			}
			continue
		}

		templates.SetEquipmentSlotRef(out, slotKey, decodeEquipmentSlotToRef(raw, class, byEquipped))
		any = true
	}
	if !any {
		return nil
	}
	return out
}

// decodeEquipmentSlotToRef builds the EquipmentItemRef for an occupied slot.
// raw is a direct equipped-armaments item value; class selects the
// normalizeEquipItemID contract. byEquipped is the inventory lookup indexed by
// ItemID|0x80000000 (weapons/armor) and bare ItemID (ammo/talismans). The
// caller has already excluded empty slots via isEmptyEquipSlot.
func decodeEquipmentSlotToRef(raw uint32, class equipClass, byEquipped map[uint32]*editor.EditableItem) *templates.EquipmentItemRef {
	// Exact encoded-form match first, then the normalized DB item ID — both
	// carry the inventory item's upgrade / infusion / AoW metadata.
	if it, ok := byEquipped[raw]; ok {
		return itemToEquipmentRef(it)
	}
	normID := normalizeEquipItemID(raw, class)
	if it, ok := byEquipped[normID]; ok {
		return itemToEquipmentRef(it)
	}

	// Not in editable inventory — fall back to a DB-derived ref so the export
	// still records which item the slot holds.
	itemData, baseID := db.GetItemDataFuzzy(normID)
	if itemData.Name != "" {
		return &templates.EquipmentItemRef{
			BaseItemID: baseID,
			Name:       itemData.Name,
		}
	}
	// Last resort: unknown item, emit a minimal ref with the normalized ID so
	// the user at least sees "there is something here we could not resolve".
	return &templates.EquipmentItemRef{BaseItemID: normID}
}

// resolveEquipmentWrites walks the selected slots in
// templates.EquipmentSection and produces a []core.EquipmentWrite batch
// ready for SaveSlot.WriteEquipment.
//
// Talisman1..4 route through the same core.WriteEquipment path as weapons,
// ammo, and armor. Slots past the target's active pouch capacity are skipped
// (equip and clear alike) so a locked-slot write can never make the core
// writer reject the whole Equipment batch. talisman5 has no native write
// contract, so it is skipped with a warning.
//
// Phase 7b.1 strict-existing-only policy:
//   - sel must be non-nil and HasAny == true at the call site.
//   - sec may be nil only when no slot is selected (defensive — the
//     caller checks hasEquipment before invoking us).
//   - For each selected + populated slot:
//   - BaseItemID == 0 → emit EquipmentWrite{Handle: 0} (explicit clear).
//   - BaseItemID > 0 → search slot.Inventory.CommonItems for a
//     matching item. Storage is NOT searched. Match keys: BaseItemID
//     (required), Upgrade (optional disambiguator), InfusionName
//     (optional), AoWItemID (optional). Multi-match resolves to the
//     first hit + emits equipment_item_ambiguous warning. No match
//     emits equipment_item_not_in_inventory warning and the slot is
//     skipped. Ammo slots (arrows/bolts) are a pass-through category, so
//     they resolve against the GaMap-backed ammo candidate list
//     (collectAmmoCandidates) instead of the editable inventory, but keep
//     the same first-wins / not-in-inventory semantics.
//
// Returned warnings carry the canonical slot key in Container (reusing
// the existing optional string field on ImportPreviewIssue so the UI
// can deep-link to the affected slot without a new field).
//
// The Go error return is reserved for infrastructure problems (nil
// slot, nil section pointer where the caller expected one); per-slot
// resolution issues never surface as a Go error.
func resolveEquipmentWrites(slot *core.SaveSlot, effectiveTalismanSlots int, sel *templates.SectionSelection, sec *templates.EquipmentSection) ([]core.EquipmentWrite, []templates.ImportPreviewIssue, error) {
	if slot == nil {
		return nil, nil, fmt.Errorf("resolveEquipmentWrites: nil slot")
	}
	// Snapshot the editable inventory once so per-slot lookups are
	// against a single consistent view. BuildSnapshot does the GaMap +
	// DB resolution + AoW current-state lookups the resolver needs to
	// match the optional disambiguators (Upgrade, InfusionName,
	// AoWItemID).
	snap, err := editor.BuildSnapshot(slot, "", -1)
	if err != nil {
		return nil, nil, fmt.Errorf("resolveEquipmentWrites: BuildSnapshot: %w", err)
	}
	// Ammo (arrows / bolts) is a pass-through category (not in
	// editor.SupportedCategories), so it never appears in snap.InventoryItems.
	// Collect owned ammo directly from the parsed inventory + GaMap — the same
	// two structures WriteEquipment consults — so an occupied arrows/bolts ref
	// can still resolve to a real native handle. Storage is never scanned.
	ammo := collectAmmoCandidates(slot.Inventory.CommonItems, slot.GaMap)
	return resolveEquipmentWritesFromItems(snap.InventoryItems, ammo, effectiveTalismanSlots, sel, sec)
}

// ammoCandidate is one owned-ammo record the equipment resolver can equip into
// an arrows/bolts slot. handle is the original native inventory GaItem handle —
// a real ammo handle already accepted by WriteEquipment (arrows/bolts use
// 0x80/0xB0 GaItem records) — and is the value written into the slot. itemID is
// the canonical BaseItemID the DB resolver returns for that handle, so it
// matches against templates.EquipmentItemRef.BaseItemID (never the raw GaMap
// value, which may differ from the resolver's canonical base).
type ammoCandidate struct {
	handle uint32
	itemID uint32
}

// collectAmmoCandidates is the single source of ammo candidates for the
// resolver. It is fail-closed: a hand-authored template must never be able to
// smuggle a weapon, armor, talisman, ordinary goods, or an unknown/technical
// placeholder into an arrows/bolts slot by pointing the ref at its BaseItemID.
//
// A record qualifies only when ALL of the following hold:
//   - positive quantity (Quantity & 0x7FFFFFFF > 0),
//   - a GaMap entry resolving the handle to an item ID that is neither 0 nor
//     GaHandleInvalid,
//   - the native handle carries a weapon (0x80) or goods/item (0xB0) prefix —
//     the only two families real arrows/bolts GaItem records use; armor,
//     accessory/talisman, AoW, and any other prefix are rejected outright,
//   - db.GetItemDataFuzzy resolves that item ID to exactly the
//     arrows_and_bolts category.
//
// The candidate stores the canonical BaseItemID the resolver returns (for
// matching against the ref) but keeps the ORIGINAL native handle (for
// WriteEquipment). Only the passed inventory slice is scanned; the caller never
// passes Storage, so ammo present only in Storage is invisible here.
func collectAmmoCandidates(inv []core.InventoryItem, gaMap map[uint32]uint32) []ammoCandidate {
	out := make([]ammoCandidate, 0, len(inv))
	for _, it := range inv {
		if it.Quantity&0x7FFFFFFF == 0 {
			continue
		}
		itemID, ok := gaMap[it.GaItemHandle]
		if !ok || itemID == 0 || itemID == core.GaHandleInvalid {
			continue
		}
		switch it.GaItemHandle & core.GaHandleTypeMask {
		case core.ItemTypeWeapon, core.ItemTypeItem:
		default:
			continue
		}
		itemData, baseID := db.GetItemDataFuzzy(itemID)
		if itemData.Category != templates.ItemCategoryArrowsAndBolts {
			continue
		}
		out = append(out, ammoCandidate{handle: it.GaItemHandle, itemID: baseID})
	}
	return out
}

// lookupAmmoHandle returns (handle, ambiguous, found) for the first ammo
// candidate whose resolved item ID matches baseItemID. Duplicate records that
// share the same handle collapse to one match (positional first-wins); two
// DIFFERENT handles resolving to the same item ID are a genuine ambiguity and
// set ambiguous=true, mirroring the weapon/armor/talisman first-wins contract.
func lookupAmmoHandle(candidates []ammoCandidate, baseItemID uint32) (uint32, bool, bool) {
	var winner uint32
	found := false
	seen := map[uint32]struct{}{}
	for _, c := range candidates {
		if c.itemID != baseItemID {
			continue
		}
		if !found {
			winner = c.handle
			found = true
		}
		seen[c.handle] = struct{}{}
	}
	if !found {
		return 0, false, false
	}
	return winner, len(seen) > 1, true
}

// resolveEquipmentWritesFromItems is the pure-logic core of the
// resolver, taking an already-materialised list of editable inventory
// items. Factored out so tests can exercise the matching / warning
// logic without standing up a full SaveSlot that BuildSnapshot can
// parse.
//
// unlockedTalismans is the target character's active pouch capacity (1–4,
// from activeTalismanSlotCount). Talisman writes for ordinals at or beyond
// that capacity are skipped so the core writer never rejects the batch over a
// locked slot.
func resolveEquipmentWritesFromItems(items []editor.EditableItem, ammo []ammoCandidate, unlockedTalismans int, sel *templates.SectionSelection, sec *templates.EquipmentSection) ([]core.EquipmentWrite, []templates.ImportPreviewIssue, error) {
	if sec == nil {
		return nil, nil, fmt.Errorf("resolveEquipmentWrites: nil equipment section")
	}

	var writes []core.EquipmentWrite
	var warnings []templates.ImportPreviewIssue

	for _, slotKey := range templates.EquipmentSlotOrder {
		if !sel.Selected(slotKey) {
			continue
		}
		ref := templates.EquipmentSlotRef(sec, slotKey)
		if ref == nil {
			continue
		}
		kind, ok := equipmentSlotKindForKey(slotKey)
		if !ok {
			// The only non-writable key in EquipmentSlotOrder is talisman5,
			// which has no native write contract. Skip it with a warning
			// rather than a hard error so an imported template that carries
			// talisman5 still applies its other slots.
			warnings = append(warnings, templates.ImportPreviewIssue{
				Severity:  "warning",
				Code:      templates.IssueCodeEquipmentSlotInvalid,
				Message:   fmt.Sprintf("equipment.%s: slot has no native write contract; skipped", slotKey),
				Container: slotKey,
			})
			continue
		}
		// Skip talisman slots the target has not unlocked — both equips
		// (which the writer would reject) and clears (which it would accept),
		// so the batch composition never depends on the writer's clear/equip
		// asymmetry for locked slots.
		if ordinal, isTal := talismanOrdinal(slotKey); isTal && ordinal >= unlockedTalismans {
			warnings = append(warnings, templates.ImportPreviewIssue{
				Severity:  "warning",
				Code:      templates.IssueCodeTalismanSlotPouchInsufficient,
				Message:   fmt.Sprintf("equipment.%s: talisman slot %d exceeds the target's %d unlocked pouch slot(s); skipped", slotKey, ordinal+1, unlockedTalismans),
				Container: slotKey,
			})
			continue
		}

		if ref.BaseItemID == 0 {
			writes = append(writes, core.EquipmentWrite{Slot: kind, Handle: 0})
			continue
		}

		// Ammo is a pass-through category (never in the editable inventory
		// snapshot), so occupied arrows/bolts refs resolve against the ammo
		// candidate list collected from slot.Inventory.CommonItems + GaMap.
		// Weapons/armor/talismans keep the editable-inventory resolution with
		// the optional Upgrade / InfusionName / AoWItemID disambiguators.
		var (
			handle    uint32
			ambiguous bool
			found     bool
		)
		if equipmentSlotEquipClass(slotKey) == classAmmo {
			handle, ambiguous, found = lookupAmmoHandle(ammo, ref.BaseItemID)
		} else {
			handle, ambiguous, found = lookupEquipmentHandle(items, ref)
		}
		if !found {
			warnings = append(warnings, templates.ImportPreviewIssue{
				Severity:   "warning",
				Code:       templates.IssueCodeEquipmentItemNotInInventory,
				Message:    fmt.Sprintf("equipment.%s: baseItemID 0x%08X is not in inventory; slot skipped", slotKey, ref.BaseItemID),
				Container:  slotKey,
				BaseItemID: ref.BaseItemID,
			})
			continue
		}
		if ambiguous {
			warnings = append(warnings, templates.ImportPreviewIssue{
				Severity:   "warning",
				Code:       templates.IssueCodeEquipmentItemAmbiguous,
				Message:    fmt.Sprintf("equipment.%s: multiple inventory matches for baseItemID 0x%08X; first wins", slotKey, ref.BaseItemID),
				Container:  slotKey,
				BaseItemID: ref.BaseItemID,
			})
		}
		writes = append(writes, core.EquipmentWrite{Slot: kind, Handle: handle})
	}
	return writes, warnings, nil
}

// lookupEquipmentHandle returns (handle, ambiguous, found) for the
// first EditableItem in items that matches the ref's BaseItemID and any
// supplied optional disambiguators (Upgrade, InfusionName, AoWItemID).
// ambiguous is true when more than one item satisfied the match.
func lookupEquipmentHandle(items []editor.EditableItem, ref *templates.EquipmentItemRef) (uint32, bool, bool) {
	matches := 0
	var winner uint32
	for i := range items {
		it := &items[i]
		if it.BaseItemID != ref.BaseItemID {
			continue
		}
		if ref.Upgrade != nil && it.CurrentUpgrade != *ref.Upgrade {
			continue
		}
		if ref.InfusionName != "" && it.InfusionName != ref.InfusionName {
			continue
		}
		if ref.AoWItemID != nil {
			if !it.HasCurrentAoW || it.CurrentAoWItemID != *ref.AoWItemID {
				continue
			}
		}
		matches++
		if matches == 1 {
			winner = it.OriginalHandle
		}
	}
	if matches == 0 {
		return 0, false, false
	}
	return winner, matches > 1, true
}

// itemToEquipmentRef projects an EditableItem onto the schema fields
// EquipmentItemRef carries. Upgrade / AoW pointers are heap-allocated so
// the resulting ref does not alias the editor item.
func itemToEquipmentRef(it *editor.EditableItem) *templates.EquipmentItemRef {
	ref := &templates.EquipmentItemRef{
		BaseItemID:   it.BaseItemID,
		Name:         it.Name,
		InfusionName: it.InfusionName,
	}
	if it.IsWeapon || it.IsArmor {
		up := it.CurrentUpgrade
		ref.Upgrade = &up
	}
	if it.HasCurrentAoW && it.CurrentAoWItemID != 0 {
		aow := it.CurrentAoWItemID
		ref.AoWItemID = &aow
	}
	return ref
}
