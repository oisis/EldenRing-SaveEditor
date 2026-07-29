package application

import (
	"encoding/binary"
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
	Handle   uint32 `json:"handle"`
	Quantity uint32 `json:"quantity"`
	Name     string `json:"name"`
	IconPath string `json:"iconPath"`
	Resolved bool   `json:"resolved"`
	// MemorySlots is the equipped spell's Memory Slot cost (1-3). It is only
	// meaningful for spell slots and stays 0 for every other slot family and for
	// a spell whose cost is unknown; the UI sums it to show real memory usage.
	MemorySlots int `json:"memorySlots,omitempty"`
}

// EquipmentSnapshot is the read-only projection of one character's equipped
// items, grouped per UI slot family. Physick holds the two active Wondrous
// Physick tears (raw IDs preserved, resolved via the local item DB).
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
	Physick             [2]EquipmentSlotView  `json:"physick"`
	Spells              [14]EquipmentSlotView `json:"spells"`
	ActiveSpellIndex    int                   `json:"activeSpellIndex"`
}

// EquipmentChange is one writable Equipment slot request from the frontend.
// Handle is the owned inventory handle selected in the picker; zero clears the
// slot. Quick Items, Pouch, Physick and spells have separate contracts.
type EquipmentChange struct {
	Slot   core.EquipmentSlotKind `json:"slot"`
	Handle uint32                 `json:"handle"`
}

// QuickPouchChange is one writable Quick Item / Pouch request. Handle is the
// exact owned goods handle; zero clears the slot.
type QuickPouchChange struct {
	Slot   core.QuickPouchSlotKind `json:"slot"`
	Handle uint32                  `json:"handle"`
}

// PhysickChange is one writable Wondrous Physick tear request. Handle is the
// exact owned goods handle of a crystal tear; zero clears the slot.
type PhysickChange struct {
	Slot   core.PhysickSlotKind `json:"slot"`
	Handle uint32               `json:"handle"`
}

// equipClass selects how a raw stored value is normalized to a DB item ID.
type equipClass int

const (
	classHandArmament equipClass = iota // hand armaments: itemID | 0x80000000
	classArmor                          // armor: itemID | 0x80000000
	classAmmo                           // arrows/bolts: bare item ID
	classTalisman                       // talismans: bare lower bits, canonical 0x20 prefix
	classGoods                          // quick items / pouch: 0xB0 goods handle
	classPhysickTear                    // physick tears: bare GoodsParam item ID, resolved as-is
	classSpell                          // spells: bare MagicParam ID, canonical 0x40 item ID

	unarmedItemID uint32 = 0x0001ADB0

	baseSpellSlotCount     = 2
	standardSpellSlotLimit = 10
	maxSpellSlotCount      = 12
	moonOfNokstellaItemID  = 0x20000474
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
	// The filled Physick flask has a distinct saved ID but shares the empty
	// flask's display metadata. Keep RawID above for fidelity while resolving
	// both variants to the one DB-backed presentation entry.
	displayID := db.WondrousPhysickDisplayID(normID)
	item, baseID := db.GetItemDataFuzzy(displayID)
	if item.Name == "" {
		view.Name = fmt.Sprintf("Unknown item (0x%08X)", raw)
		return view
	}
	name := item.Name
	// An infused weapon's ID encodes both its affinity (hundreds) and upgrade
	// level (remainder). Display only the latter: e.g. an offset of 604 is a
	// +4 weapon with the affinity at offset 600, not a +604 weapon.
	if class == classHandArmament && isWeaponLikeCategory(item.Category) {
		if upgrade, _ := decodeWeaponUpgradeInfusion(displayID, baseID); upgrade > 0 {
			name = fmt.Sprintf("%s +%d", name, upgrade)
		}
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
	case classPhysickTear:
		// Tears are stored as bare item IDs (prefix 0x40). Display aliases, where
		// explicitly confirmed, are resolved by PhysickTearDisplayID; raw IDs stay
		// distinct. In particular, Crimson 0x40002AFA and 0x40002AFB are separate
		// tears with the same player-facing name.
		return db.PhysickTearDisplayID(raw)
	case classSpell:
		return raw | 0x40000000
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

// templateSpellSlotLimit is the player-visible hard limit used by character
// templates. Elden Ring exposes at most 10 spell slots normally and 12 only
// while Moon of Nokstella is equipped in an unlocked talisman pouch. The save
// keeps 14 physical records, but records 13-14 are not playable spell slots
// and must not leak into newly created templates.
func templateSpellSlotLimit(slot *core.SaveSlot, raw core.RawEquippedState) int {
	activeTalismanSlots := activeTalismanSlotCount(slot.Player.TalismanSlots)
	for i := 0; i < activeTalismanSlots; i++ {
		if normalizeEquipItemID(raw.Equipped[17+i], classTalisman) == moonOfNokstellaItemID {
			return maxSpellSlotCount
		}
	}
	return standardSpellSlotLimit
}

// equipSlotView builds a view for an equipped-armaments slot, returning an
// empty (unoccupied) view for sentinel values.
func equipSlotView(raw uint32, class equipClass) EquipmentSlotView {
	if isEmptyEquipSlot(raw, class) {
		return EquipmentSlotView{}
	}
	return resolveEquipView(raw, class)
}

func equippedSlotHandle(slot *core.SaveSlot, index int, raw uint32, class equipClass, occupied bool) uint32 {
	if !occupied || slot == nil || index < 0 {
		return 0
	}
	if class == classTalisman {
		handle := core.ItemTypeAccessory | (normalizeEquipItemID(raw, classTalisman) & 0x0FFFFFFF)
		for _, item := range slot.Inventory.CommonItems {
			if item.GaItemHandle == handle && item.Quantity&0x7FFFFFFF != 0 {
				return handle
			}
		}
		return 0
	}
	if slot.EquipItemsIDOffset <= 0 || slot.EquipItemsIDOffset+(index+1)*4 > len(slot.Data) {
		return 0
	}
	header := binary.LittleEndian.Uint32(slot.Data[slot.EquipItemsIDOffset+index*4:])
	key := header >> 8
	switch class {
	case classHandArmament, classAmmo:
		return core.ItemTypeWeapon | key
	case classArmor:
		return core.ItemTypeArmor | key
	default:
		return 0
	}
}

func equippedView(slot *core.SaveSlot, index int, raw uint32, class equipClass) EquipmentSlotView {
	view := equipSlotView(raw, class)
	view.Handle = equippedSlotHandle(slot, index, raw, class, view.Occupied)
	return view
}

// goodsView builds a view for a quick-item / pouch pair. Empty slots use the
// {item_id: 0, equip_index: 0xFFFFFFFF} sentinel; item_id of 0 / 0xFFFFFFFF is
// treated as empty regardless of equip_index.
func goodsView(slot *core.SaveSlot, pair core.RawEquipItem) EquipmentSlotView {
	if pair.ItemID == 0 || pair.ItemID == core.GaHandleInvalid {
		return EquipmentSlotView{}
	}
	view := resolveEquipView(pair.ItemID, classGoods)
	view.Handle = pair.ItemID
	for _, item := range slot.Inventory.CommonItems {
		if item.GaItemHandle == pair.ItemID {
			view.Quantity = item.Quantity & 0x7FFFFFFF
			break
		}
	}
	return view
}

// physickSlotView projects one Wondrous Physick tear field (T545). 0xFFFFFFFF is
// the confirmed native sentinel for an empty tear field on both slots: it yields
// an empty slot (Occupied=false, Resolved=false) with the raw sentinel preserved
// per EquipmentSlotView's contract, and is never resolved against the item DB.
// Every other value — including 0, which is NOT a confirmed sentinel — falls
// through the normal unresolved path and stays visible as "Unknown item (0x…)".
func physickSlotView(slot *core.SaveSlot, raw uint32) EquipmentSlotView {
	if raw == core.GaHandleInvalid {
		return EquipmentSlotView{RawID: raw}
	}
	view := resolveEquipView(raw, classPhysickTear)
	view.Handle = physickTearHandle(slot, raw)
	return view
}

// physickTearHandle returns the exact goods handle the character owns whose raw
// GoodsParam ID equals rawID, so the picker can recognize the currently mixed
// tear (clear-on-reselect) and block it in the other slot. It searches only the
// two carried containers proper to tears (CommonItems, KeyItems) — never Storage
// — and requires a positive quantity. Crimson 0x40002AFA and 0x40002AFB remain
// separate exact handles. Returns 0 when no exact owned handle exists, leaving
// the view's Handle unset.
func physickTearHandle(slot *core.SaveSlot, rawID uint32) uint32 {
	if slot == nil {
		return 0
	}
	for _, container := range [][]core.InventoryItem{slot.Inventory.CommonItems, slot.Inventory.KeyItems} {
		for _, item := range container {
			handle := item.GaItemHandle
			if handle&core.GaHandleTypeMask != core.ItemTypeItem || item.Quantity&0x7FFFFFFF == 0 {
				continue
			}
			if db.HandleToItemID(handle) == rawID {
				return handle
			}
		}
	}
	return 0
}

func spellSlotView(raw uint32) EquipmentSlotView {
	if raw == core.EquippedSpellEmptySentinel {
		return EquipmentSlotView{RawID: raw}
	}
	view := resolveEquipView(raw, classSpell)
	if cost, ok := db.SpellMemorySlotCost(normalizeEquipItemID(raw, classSpell)); ok {
		view.MemorySlots = int(cost)
	}
	return view
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
		snap.RightHandArmaments[i] = equippedView(&slot, rightIdx[i], raw.Equipped[rightIdx[i]], classHandArmament)
		snap.LeftHandArmaments[i] = equippedView(&slot, leftIdx[i], raw.Equipped[leftIdx[i]], classHandArmament)
	}
	// Arrows: slots 6/8. Bolts: slots 7/9.
	snap.Arrows[0] = equippedView(&slot, 6, raw.Equipped[6], classAmmo)
	snap.Arrows[1] = equippedView(&slot, 8, raw.Equipped[8], classAmmo)
	snap.Bolts[0] = equippedView(&slot, 7, raw.Equipped[7], classAmmo)
	snap.Bolts[1] = equippedView(&slot, 9, raw.Equipped[9], classAmmo)
	// Armor: slots 12/13/14/15.
	armorIdx := [4]int{12, 13, 14, 15}
	for i := 0; i < 4; i++ {
		snap.Armor[i] = equippedView(&slot, armorIdx[i], raw.Equipped[armorIdx[i]], classArmor)
	}
	// Talismans: active slots are always contiguous from the left. Slots beyond
	// the character's unlock count stay empty and are not rendered by the UI.
	// Talisman5 / index 21 remains outside this UI.
	for i := 0; i < snap.ActiveTalismanSlots; i++ {
		snap.Talismans[i] = equippedView(&slot, 17+i, raw.Equipped[17+i], classTalisman)
	}
	// Quick items (10) and pouch (6).
	for i := 0; i < 10; i++ {
		snap.QuickItems[i] = goodsView(&slot, raw.QuickItems[i])
	}
	for i := 0; i < 6; i++ {
		snap.Pouch[i] = goodsView(&slot, raw.Pouch[i])
	}
	// Wondrous Physick: two active tears (T545). physicsOff+0 is screen slot 1,
	// physicsOff+4 is slot 2; the game does not left-pack, so a single tear lives
	// only in slot 2. Raw IDs preserved; display metadata follows explicit
	// technical aliases via classPhysickTear.
	for i := 0; i < 2; i++ {
		snap.Physick[i] = physickSlotView(&slot, raw.Physick[i])
	}
	for i := 0; i < core.EquippedSpellSlotCount; i++ {
		snap.Spells[i] = spellSlotView(raw.Spells[i])
	}
	if raw.ActiveSpellIndex < core.EquippedSpellSlotCount {
		snap.ActiveSpellIndex = int(raw.ActiveSpellIndex)
	} else {
		// A malformed save must not select an arbitrary UI slot. The raw reader
		// preserves the value, while the presentation falls back safely.
		snap.ActiveSpellIndex = -1
	}

	return snap, nil
}

// SaveEquippedSpells replaces the compact equipped spell list in memory. It
// does not write a save file; the normal application-level Write Save action is
// still the only disk write. The native active-index rule is deliberately
// narrow: retain a still-valid index; when compaction removes its last record,
// reset it to zero. This is established by T119/T121/T123.
func (a *App) SaveEquippedSpells(charIdx int, itemIDs []uint32) error {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if charIdx < 0 || charIdx >= len(a.save.Slots) || !a.save.ActiveSlots[charIdx] {
		return fmt.Errorf("invalid active character slot %d", charIdx)
	}

	a.slotMu[charIdx].Lock()
	defer a.slotMu[charIdx].Unlock()

	slot := &a.save.Slots[charIdx]
	raw, err := slot.ReadEquippedState()
	if err != nil {
		return fmt.Errorf("SaveEquippedSpells: read equipped state: %w", err)
	}

	// An empty loadout is a legitimate native state (all 14 records cleared,
	// active_index = 0xFFFFFFFF). It has its own atomic writer and skips the
	// per-spell / capacity checks entirely.
	if len(itemIDs) == 0 {
		a.pushUndoLocked(charIdx)
		if err := slot.ClearCompactSpells(); err != nil {
			return fmt.Errorf("SaveEquippedSpells: %w", err)
		}
		return nil
	}

	// The physical section holds at most 14 spell records regardless of memory
	// capacity; reject before doing any per-spell work.
	if len(itemIDs) > core.EquippedSpellSlotCount {
		return fmt.Errorf("SaveEquippedSpells: spell count %d exceeds the %d available records", len(itemIDs), core.EquippedSpellSlotCount)
	}

	// Memory capacity is measured in slots, not records: a single spell can cost
	// 1-3 slots, so the sum of per-spell costs — never len(itemIDs) — is what must
	// fit within the character's active capacity.
	capacity := activeSpellSlotCount(slot, raw, activeTalismanSlotCount(slot.Player.TalismanSlots))
	rawIDs := make([]uint32, len(itemIDs))
	seen := make(map[uint32]int, len(itemIDs))
	totalCost := 0
	for i, itemID := range itemIDs {
		if itemID&0xF0000000 != 0x40000000 {
			return fmt.Errorf("SaveEquippedSpells[%d]: item ID 0x%08X is not a spell", i, itemID)
		}
		item := db.GetItemData(itemID)
		if item.Category != "sorceries" && item.Category != "incantations" {
			return fmt.Errorf("SaveEquippedSpells[%d]: item ID 0x%08X is not a known spell", i, itemID)
		}
		if prev, dup := seen[itemID]; dup {
			return fmt.Errorf("SaveEquippedSpells[%d]: spell %q (0x%08X) is already equipped at position %d", i, item.Name, itemID, prev)
		}
		seen[itemID] = i
		cost, ok := db.SpellMemorySlotCost(itemID)
		if !ok {
			return fmt.Errorf("SaveEquippedSpells[%d]: no Memory Slot cost for spell %q (0x%08X); refusing to guess and write an over-capacity loadout", i, item.Name, itemID)
		}
		totalCost += int(cost)
		rawIDs[i] = db.ItemIDToMagicParamID(itemID)
	}
	if totalCost > capacity {
		return fmt.Errorf("SaveEquippedSpells: loadout costs %d Memory Slots but only %d are available", totalCost, capacity)
	}

	a.pushUndoLocked(charIdx)
	if raw.ActiveSpellIndex >= uint32(len(rawIDs)) {
		// Native removal leaves a selected middle position unchanged (T119), but
		// resets a now-out-of-range selected final position to Pebble/index zero
		// (T121). Do not synthesize any other index transformation here.
		if err := slot.WriteCompactSpellsWithActiveIndex(rawIDs, 0); err != nil {
			return fmt.Errorf("SaveEquippedSpells: %w", err)
		}
		return nil
	}
	if err := slot.WriteCompactSpells(rawIDs); err != nil {
		return fmt.Errorf("SaveEquippedSpells: %w", err)
	}
	return nil
}

// SaveEquipment applies one atomic Equipment batch in memory. The normal
// Save/Save As action remains responsible for writing the .sl2 file.
func (a *App) SaveEquipment(charIdx int, changes []EquipmentChange) error {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if charIdx < 0 || charIdx >= len(a.save.Slots) || !a.save.ActiveSlots[charIdx] {
		return fmt.Errorf("invalid active character slot %d", charIdx)
	}
	if len(changes) == 0 {
		return nil
	}

	a.slotMu[charIdx].Lock()
	defer a.slotMu[charIdx].Unlock()
	slot := &a.save.Slots[charIdx]
	writes := make([]core.EquipmentWrite, len(changes))
	for i, change := range changes {
		writes[i] = core.EquipmentWrite{Slot: change.Slot, Handle: change.Handle}
	}

	before := a.buildSlotSnapshotLocked(charIdx)
	if err := slot.WriteEquipment(writes); err != nil {
		return fmt.Errorf("SaveEquipment: %w", err)
	}
	a.pushUndoSnapshotLocked(charIdx, before)
	return nil
}

// SaveQuickPouchItems applies one atomic Quick Item / Pouch batch in memory.
// The normal Save/Save As action remains responsible for writing the .sl2 file.
func (a *App) SaveQuickPouchItems(charIdx int, changes []QuickPouchChange) error {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if charIdx < 0 || charIdx >= len(a.save.Slots) || !a.save.ActiveSlots[charIdx] {
		return fmt.Errorf("invalid active character slot %d", charIdx)
	}
	if len(changes) == 0 {
		return nil
	}

	a.slotMu[charIdx].Lock()
	defer a.slotMu[charIdx].Unlock()
	slot := &a.save.Slots[charIdx]
	writes := make([]core.QuickPouchWrite, len(changes))
	for i, change := range changes {
		if change.Handle != 0 {
			itemID := db.HandleToItemID(change.Handle)
			// The filled Physick flask (0x400000FA) has no category of its own;
			// its "tools" metadata lives under the canonical empty variant
			// (0x400000FB). Resolve eligibility via the display ID while leaving
			// the raw handle untouched for the writer.
			item := db.GetItemData(db.WondrousPhysickDisplayID(itemID))
			if item.Category != "tools" && item.Category != "ashes" {
				return fmt.Errorf("SaveQuickPouchItems[%d]: item 0x%08X is not eligible for Quick Items or Pouch", i, itemID)
			}
		}
		writes[i] = core.QuickPouchWrite{Slot: change.Slot, Handle: change.Handle}
	}

	before := a.buildSlotSnapshotLocked(charIdx)
	if err := slot.WriteQuickPouch(writes); err != nil {
		return fmt.Errorf("SaveQuickPouchItems: %w", err)
	}
	a.pushUndoSnapshotLocked(charIdx, before)
	return nil
}

// ownedTearQuantity returns the positive quantity of the exact goods handle if
// the character owns it in either inventory container proper to crystal tears
// (CommonItems or KeyItems). Storage is intentionally excluded: only items the
// character carries can be mixed. A zero or missing quantity yields ok=false.
func ownedTearQuantity(slot *core.SaveSlot, handle uint32) (uint32, bool) {
	for _, container := range [][]core.InventoryItem{slot.Inventory.CommonItems, slot.Inventory.KeyItems} {
		for _, item := range container {
			if item.GaItemHandle != handle {
				continue
			}
			if qty := item.Quantity & 0x7FFFFFFF; qty != 0 {
				return qty, true
			}
			return 0, false
		}
	}
	return 0, false
}

// hasWondrousPhysickFlask reports whether the character carries the Flask of
// Wondrous Physick (either raw variant) with a positive quantity. A mixture is
// only meaningful when the flask itself is owned.
func hasWondrousPhysickFlask(slot *core.SaveSlot) bool {
	for _, container := range [][]core.InventoryItem{slot.Inventory.CommonItems, slot.Inventory.KeyItems} {
		for _, item := range container {
			if item.Quantity&0x7FFFFFFF != 0 && db.IsWondrousPhysick(item.GaItemHandle) {
				return true
			}
		}
	}
	return false
}

// SavePhysickMixture applies one atomic Wondrous Physick tear batch in memory.
// The normal Save/Save As action remains responsible for writing the .sl2 file.
//
// It writes only the two active-mixture u32 in EquipPhysicsData. Each non-empty
// tear must be a crystal tear the character actually owns (CommonItems/KeyItems,
// positive quantity); the exact raw ID of the owned record — derived via the
// repository SSOT db.HandleToItemID — is written through unchanged, so distinct
// Crimson tears such as 0x40002AFA are preserved without a reverse map. A
// Flask of Wondrous Physick must be owned. Handle 0 clears a slot (0xFFFFFFFF);
// the surviving tear is never left-packed.
func (a *App) SavePhysickMixture(charIdx int, changes []PhysickChange) error {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if charIdx < 0 || charIdx >= len(a.save.Slots) || !a.save.ActiveSlots[charIdx] {
		return fmt.Errorf("invalid active character slot %d", charIdx)
	}
	if len(changes) == 0 {
		return nil
	}

	a.slotMu[charIdx].Lock()
	defer a.slotMu[charIdx].Unlock()
	slot := &a.save.Slots[charIdx]

	if !hasWondrousPhysickFlask(slot) {
		return fmt.Errorf("SavePhysickMixture: character does not own a Flask of Wondrous Physick")
	}

	writes := make([]core.PhysickWrite, len(changes))
	for i, change := range changes {
		if change.Handle == 0 {
			writes[i] = core.PhysickWrite{Slot: change.Slot, Value: core.GaHandleInvalid}
			continue
		}
		if change.Handle == core.GaHandleInvalid {
			return fmt.Errorf("SavePhysickMixture[%d]: handle 0xFFFFFFFF is invalid; use Handle=0 to clear a slot", i)
		}
		if change.Handle&core.GaHandleTypeMask != core.ItemTypeItem {
			return fmt.Errorf("SavePhysickMixture[%d]: handle 0x%08X is not a goods handle", i, change.Handle)
		}
		if _, ok := ownedTearQuantity(slot, change.Handle); !ok {
			return fmt.Errorf("SavePhysickMixture[%d]: handle 0x%08X is not an owned tear with a positive quantity", i, change.Handle)
		}
		// Repository SSOT for handle → raw item ID. The exact owned ID is written
		// through, so the two distinct Crimson tears are never collapsed.
		rawID := db.HandleToItemID(change.Handle)
		if db.GetItemData(rawID).SubCategory != data.SubcatKeyCrystalTears {
			return fmt.Errorf("SavePhysickMixture[%d]: item 0x%08X is not a Wondrous Physick crystal tear", i, rawID)
		}
		writes[i] = core.PhysickWrite{Slot: change.Slot, Value: rawID}
	}

	before := a.buildSlotSnapshotLocked(charIdx)
	if err := slot.WritePhysick(writes); err != nil {
		return fmt.Errorf("SavePhysickMixture: %w", err)
	}
	a.pushUndoSnapshotLocked(charIdx, before)
	return nil
}
