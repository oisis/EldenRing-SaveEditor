package main

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/db"
	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
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
	MaxEquipLoad        float64               `json:"maxEquipLoad"`
	CurrentEquipLoad    float64               `json:"currentEquipLoad"`
	EquipLoadKnown      bool                  `json:"equipLoadKnown"`
	EquipLoadClass      string                `json:"equipLoadClass"`
	ActiveTalismanSlots int                   `json:"activeTalismanSlots"`
	ActiveSpellSlots    int                   `json:"activeSpellSlots"`
	RightHandArmaments  [3]EquipmentSlotView  `json:"rightHandArmaments"`
	LeftHandArmaments   [3]EquipmentSlotView  `json:"leftHandArmaments"`
	Arrows              [2]EquipmentSlotView  `json:"arrows"`
	Bolts               [2]EquipmentSlotView  `json:"bolts"`
	Armor               [4]EquipmentSlotView  `json:"armor"`
	Talismans           [4]EquipmentSlotView  `json:"talismans"`
	QuickItems          [10]EquipmentSlotView `json:"quickItems"`
	Pouch               [6]EquipmentSlotView  `json:"pouch"`
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

	baseSpellSlotCount    = 2
	maxSpellSlotCount     = 12
	moonOfNokstellaItemID = 0x20000474
)

// bareArmorItemIDs are the four technical ProtectorParam rows Elden Ring uses
// for an unequipped head, chest, arms, or legs slot. They are not database
// items, so treating them as occupied would make the Equip Load total unknown.
var bareArmorItemIDs = map[uint32]struct{}{
	0x10002710: {},
	0x10002774: {},
	0x100027D8: {},
	0x1000283C: {},
}

// isEmptyEquipSlot reports whether an equipped-armaments slot is empty. The
// game represents an empty hand with the real "Unarmed" weapon item; it must
// render as an empty equipment slot rather than as an equipped item.
func isEmptyEquipSlot(v uint32, class equipClass) bool {
	return v == 0 || v == core.GaHandleInvalid ||
		(class == classHandArmament && v&0x7FFFFFFF == unarmedItemID) ||
		(class == classArmor && isBareArmorItem(v))
}

func isBareArmorItem(v uint32) bool {
	_, ok := bareArmorItemIDs[v&0x7FFFFFFF]
	return ok
}

// resolveEquipView normalizes a raw stored value per its slot class and
// resolves display metadata via existing DB helpers. Resolution is independent
// of GaMap. Unknown IDs fail soft: Occupied stays true, Resolved is false, and
// Name carries an "Unknown item (0x…)" label for the tooltip.
func resolveEquipView(raw uint32, class equipClass) EquipmentSlotView {
	view := EquipmentSlotView{Occupied: true, RawID: raw}

	normID := normalizeEquipItemID(raw, class)
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

func normalizeEquipItemID(raw uint32, class equipClass) uint32 {
	switch class {
	case classHandArmament, classArmor:
		return raw & 0x7FFFFFFF
	case classAmmo:
		return raw
	case classTalisman:
		return (raw & 0x0FFFFFFF) | 0x20000000
	case classGoods:
		return db.HandleToItemID(raw)
	}
	return 0
}

// equippedItemWeight returns the weight of a load-bearing equipped item. An
// empty slot has zero weight; a resolved item without an ItemWeights entry is
// weightless. An unresolved item makes the total unavailable rather than
// returning a misleading partial value.
func equippedItemWeight(raw uint32, class equipClass) (float64, bool) {
	if isEmptyEquipSlot(raw, class) {
		return 0, true
	}
	item, baseID := db.GetItemDataFuzzy(normalizeEquipItemID(raw, class))
	if item.Name == "" {
		return 0, false
	}
	return data.ItemWeights[baseID], true
}

// currentEquipLoad sums all load-bearing equipment: six hand slots, four armor
// pieces, and the character's active talismans. Ammunition, quick items, pouch
// items, spells, and Physick tears do not contribute to the game's Equip Load.
func currentEquipLoad(raw core.RawEquippedState, activeTalismanSlots int) (float64, bool) {
	total := 0.0
	add := func(index int, class equipClass) bool {
		weight, ok := equippedItemWeight(raw.Equipped[index], class)
		if !ok {
			return false
		}
		total += weight
		return true
	}
	for _, index := range []int{0, 1, 2, 3, 4, 5} {
		if !add(index, classHandArmament) {
			return 0, false
		}
	}
	for _, index := range []int{12, 13, 14, 15} {
		if !add(index, classArmor) {
			return 0, false
		}
	}
	for i := 0; i < activeTalismanSlots; i++ {
		if !add(17+i, classTalisman) {
			return 0, false
		}
	}
	return total, true
}

// maxEquipLoad applies permanent modifiers from equipped head armor and active
// talismans. The modifier table is keyed by canonical item ID, so this avoids
// matching localized names or parsing item descriptions.
func maxEquipLoad(endurance uint32, raw core.RawEquippedState, activeTalismanSlots int) float64 {
	enduranceBonus := 0
	equipLoadRate := 0.0
	add := func(rawID uint32, class equipClass) {
		if isEmptyEquipSlot(rawID, class) {
			return
		}
		modifier, ok := data.EquipLoadModifiers[normalizeEquipItemID(rawID, class)]
		if !ok {
			return
		}
		enduranceBonus += modifier.EnduranceBonus
		equipLoadRate += modifier.EquipLoadRate
	}

	// Only the head armor slot has a permanent Equip Load modifier in the
	// current regulation data; other armor slots have none.
	add(raw.Equipped[12], classArmor)
	for i := 0; i < activeTalismanSlots; i++ {
		add(raw.Equipped[17+i], classTalisman)
	}
	return core.MaxEquipLoad(endurance, enduranceBonus, equipLoadRate)
}

// activeTalismanSlotCount converts the save's additional-slot field (0–3) to
// the total number of usable talisman slots (1–4). The parser already clamps
// this field, but the local clamp keeps the read-only UI projection safe for
// synthetic or future save data as well.
func activeTalismanSlotCount(additional uint8) int {
	if additional > 3 {
		additional = 3
	}
	return int(additional) + 1
}

// activeSpellSlotCount combines the character's permanent Memory Stones with
// Moon of Nokstella's equipped-only two-slot bonus. Memory Stones are read via
// the same effective inventory source as the Character tab, and only unlocked
// talisman fields can activate the Moon.
func activeSpellSlotCount(slot *core.SaveSlot, raw core.RawEquippedState, activeTalismanSlots int) int {
	slots := baseSpellSlotCount + int(normalizeMemoryStones(memoryStonesEffective(slot)))
	for i := 0; i < activeTalismanSlots; i++ {
		if normalizeEquipItemID(raw.Equipped[17+i], classTalisman) == moonOfNokstellaItemID {
			slots += 2
			break
		}
	}
	if slots > maxSpellSlotCount {
		return maxSpellSlotCount
	}
	return slots
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
	snap.ActiveTalismanSlots = activeTalismanSlotCount(slot.Player.TalismanSlots)
	snap.ActiveSpellSlots = activeSpellSlotCount(&slot, raw, snap.ActiveTalismanSlots)
	snap.MaxEquipLoad = maxEquipLoad(slot.Player.Endurance, raw, snap.ActiveTalismanSlots)
	snap.CurrentEquipLoad, snap.EquipLoadKnown = currentEquipLoad(raw, snap.ActiveTalismanSlots)
	if snap.EquipLoadKnown {
		snap.EquipLoadClass = string(core.ClassifyEquipLoad(snap.CurrentEquipLoad, snap.MaxEquipLoad))
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
	// Talismans: active slots are always contiguous from the left. Slots beyond
	// the character's unlock count stay empty and are not rendered by the UI.
	// Talisman5 / index 21 remains outside this UI.
	for i := 0; i < snap.ActiveTalismanSlots; i++ {
		snap.Talismans[i] = equipSlotView(raw.Equipped[17+i], classTalisman)
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
