package main

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/db"
)

// EquipmentSlotView is the frontend-facing projection of a single equipped
// slot. RawID preserves the exact value stored in the save (for future
// write/round-trip work); resolution metadata is derived backend-side so the
// frontend never decodes save IDs or matches names.
type EquipmentSlotView struct {
	Occupied bool   `json:"occupied"`
	RawID    uint32 `json:"rawId"`
	Name     string `json:"name"`
	IconPath string `json:"iconPath"`
	Resolved bool   `json:"resolved"`
}

// EquipmentSnapshot is the read-only projection of one character's equipped
// items, grouped per UI slot family. Physick is intentionally excluded: its
// tear encoding is not yet confirmed and it stays presentation-only.
type EquipmentSnapshot struct {
	RightHandArmaments [3]EquipmentSlotView  `json:"rightHandArmaments"`
	LeftHandArmaments  [3]EquipmentSlotView  `json:"leftHandArmaments"`
	Arrows             [2]EquipmentSlotView  `json:"arrows"`
	Bolts              [2]EquipmentSlotView  `json:"bolts"`
	Armor              [4]EquipmentSlotView  `json:"armor"`
	Talismans          [4]EquipmentSlotView  `json:"talismans"`
	QuickItems         [10]EquipmentSlotView `json:"quickItems"`
	Pouch              [6]EquipmentSlotView  `json:"pouch"`
}

// equipClass selects how a raw stored value is normalized to a DB item ID.
type equipClass int

const (
	classHandArmament equipClass = iota // hand armaments: itemID | 0x80000000
	classArmor                          // armor: itemID | 0x80000000
	classAmmo                           // arrows/bolts: bare item ID
	classTalisman                       // talismans: bare lower bits, canonical 0x20 prefix
	classGoods                          // quick items / pouch: 0xB0 goods handle

	unarmedItemID uint32 = 0x0001ADB0
)

// isEmptyEquipSlot reports whether an equipped-armaments slot is empty. The
// game represents an empty hand with the real "Unarmed" weapon item; it must
// render as an empty equipment slot rather than as an equipped item.
func isEmptyEquipSlot(v uint32, class equipClass) bool {
	return v == 0 || v == core.GaHandleInvalid ||
		(class == classHandArmament && v&0x7FFFFFFF == unarmedItemID)
}

// resolveEquipView normalizes a raw stored value per its slot class and
// resolves display metadata via existing DB helpers. Resolution is independent
// of GaMap. Unknown IDs fail soft: Occupied stays true, Resolved is false, and
// Name carries an "Unknown item (0x…)" label for the tooltip.
func resolveEquipView(raw uint32, class equipClass) EquipmentSlotView {
	view := EquipmentSlotView{Occupied: true, RawID: raw}

	var normID uint32
	switch class {
	case classHandArmament, classArmor:
		normID = raw & 0x7FFFFFFF
	case classAmmo:
		normID = raw
	case classTalisman:
		normID = (raw & 0x0FFFFFFF) | 0x20000000
	case classGoods:
		normID = db.HandleToItemID(raw)
	}

	item, baseID := db.GetItemDataFuzzy(normID)
	if item.Name == "" {
		view.Name = fmt.Sprintf("Unknown item (0x%08X)", raw)
		return view
	}
	name := item.Name
	if upgrade := normID - baseID; upgrade > 0 {
		name = fmt.Sprintf("%s +%d", name, upgrade)
	}
	view.Name = name
	view.IconPath = item.IconPath
	view.Resolved = true
	return view
}

// equipSlotView builds a view for an equipped-armaments slot, returning an
// empty (unoccupied) view for sentinel values.
func equipSlotView(raw uint32, class equipClass) EquipmentSlotView {
	if isEmptyEquipSlot(raw, class) {
		return EquipmentSlotView{}
	}
	return resolveEquipView(raw, class)
}

// goodsView builds a view for a quick-item / pouch pair. Empty slots use the
// {item_id: 0, equip_index: 0xFFFFFFFF} sentinel; item_id of 0 / 0xFFFFFFFF is
// treated as empty regardless of equip_index.
func goodsView(pair core.RawEquipItem) EquipmentSlotView {
	if pair.ItemID == 0 || pair.ItemID == core.GaHandleInvalid {
		return EquipmentSlotView{}
	}
	return resolveEquipView(pair.ItemID, classGoods)
}

// GetEquipmentSnapshot returns the read-only equipped-item projection for the
// given character slot. It never mutates any save state and never calls a
// writer / repack / repair / save path.
func (a *App) GetEquipmentSnapshot(charIdx int) (EquipmentSnapshot, error) {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()

	var snap EquipmentSnapshot
	if a.save == nil {
		return snap, fmt.Errorf("no save loaded")
	}
	if charIdx < 0 || charIdx >= 10 {
		return snap, fmt.Errorf("invalid slot index")
	}
	if !a.save.ActiveSlots[charIdx] {
		return snap, fmt.Errorf("character slot %d is not active", charIdx)
	}

	a.slotMu[charIdx].Lock()
	defer a.slotMu[charIdx].Unlock()

	slot := a.save.Slots[charIdx]
	if slot.Version == 0 {
		return snap, fmt.Errorf("character slot %d is empty", charIdx)
	}

	raw, err := slot.ReadEquippedState()
	if err != nil {
		return snap, err
	}

	// Hand armaments: right hand = slots 1/3/5, left hand = 0/2/4.
	rightIdx := [3]int{1, 3, 5}
	leftIdx := [3]int{0, 2, 4}
	for i := 0; i < 3; i++ {
		snap.RightHandArmaments[i] = equipSlotView(raw.Equipped[rightIdx[i]], classHandArmament)
		snap.LeftHandArmaments[i] = equipSlotView(raw.Equipped[leftIdx[i]], classHandArmament)
	}
	// Arrows: slots 6/8. Bolts: slots 7/9.
	snap.Arrows[0] = equipSlotView(raw.Equipped[6], classAmmo)
	snap.Arrows[1] = equipSlotView(raw.Equipped[8], classAmmo)
	snap.Bolts[0] = equipSlotView(raw.Equipped[7], classAmmo)
	snap.Bolts[1] = equipSlotView(raw.Equipped[9], classAmmo)
	// Armor: slots 12/13/14/15.
	armorIdx := [4]int{12, 13, 14, 15}
	for i := 0; i < 4; i++ {
		snap.Armor[i] = equipSlotView(raw.Equipped[armorIdx[i]], classArmor)
	}
	// Talismans: slots 17/18/19/20 (Talisman5 / index 21 excluded from UI).
	talismanIdx := [4]int{17, 18, 19, 20}
	for i := 0; i < 4; i++ {
		snap.Talismans[i] = equipSlotView(raw.Equipped[talismanIdx[i]], classTalisman)
	}
	// Quick items (10) and pouch (6).
	for i := 0; i < 10; i++ {
		snap.QuickItems[i] = goodsView(raw.QuickItems[i])
	}
	for i := 0; i < 6; i++ {
		snap.Pouch[i] = goodsView(raw.Pouch[i])
	}

	return snap, nil
}
