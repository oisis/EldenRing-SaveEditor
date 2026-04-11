package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strconv"

	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"github.com/oisis/EldenRing-SaveEditor/backend/db"
	"github.com/oisis/EldenRing-SaveEditor/backend/vm"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxUndoDepth = 5

// slotSnapshot holds a deep copy of a SaveSlot for undo purposes.
type slotSnapshot struct {
	Data              []byte
	Version           uint32
	Player            core.PlayerGameData
	GaMap             map[uint32]uint32
	Inventory         core.EquipInventoryData
	Storage           core.EquipInventoryData
	Warnings          []string
	MagicOffset       int
	InventoryEnd      int
	EventFlagsOffset  int
	PlayerDataOffset  int
	FaceDataOffset    int
	StorageBoxOffset  int
	IngameTimerOffset int
	GaItemDataOffset  int
}

// App struct
type App struct {
	ctx        context.Context
	save       *core.SaveFile
	sourceSave *core.SaveFile
	undoStacks [10][]slotSnapshot
}

// NewApp creates a new App struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SelectAndOpenSave opens a native file dialog and loads the selected save
func (a *App) SelectAndOpenSave() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Elden Ring Save File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Elden Ring Save (*.sl2;*.dat;*.txt)", Pattern: "*.sl2;*.dat;*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("no file selected")
	}

	save, err := core.LoadSave(path)
	if err != nil {
		return "", err
	}
	a.save = save
	a.clearAllUndoStacks()
	return string(save.Platform), nil
}

// SelectAndOpenSourceSave opens a native file dialog and loads the selected source save for import
func (a *App) SelectAndOpenSourceSave() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select SOURCE Elden Ring Save File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Elden Ring Save (*.sl2;*.dat;*.txt)", Pattern: "*.sl2;*.dat;*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("no file selected")
	}

	save, err := core.LoadSave(path)
	if err != nil {
		return "", err
	}
	a.sourceSave = save
	return string(save.Platform), nil
}

// GetCharacter returns the ViewModel for a specific slot
func (a *App) GetCharacter(index int) (*vm.CharacterViewModel, error) {
	if a.save == nil {
		return nil, fmt.Errorf("no save loaded")
	}
	if index < 0 || index >= 10 {
		return nil, fmt.Errorf("invalid slot index")
	}

	slot := a.save.Slots[index]
	return vm.MapParsedSlotToVM(&slot)
}

// SaveCharacter updates the raw slot data from the ViewModel
func (a *App) SaveCharacter(index int, charVM vm.CharacterViewModel) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if index < 0 || index >= 10 {
		return fmt.Errorf("invalid slot index")
	}

	a.pushUndo(index)

	// 1. Update the slot data
	if err := vm.ApplyVMToParsedSlot(&charVM, &a.save.Slots[index]); err != nil {
		return err
	}

	// 2. Update ProfileSummary (for the menu)
	a.save.ProfileSummaries[index].Level = a.save.Slots[index].Player.Level
	copy(a.save.ProfileSummaries[index].CharacterName[:], a.save.Slots[index].Player.CharacterName[:])

	return nil
}

// WriteSave writes the current save state to a file
func (a *App) WriteSave(platform string) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Save Elden Ring Save File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Elden Ring Save (*.sl2;*.dat;*.txt)", Pattern: "*.sl2;*.dat;*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("no file selected")
	}

	// Backup only when the target file already exists (nothing to protect otherwise).
	if _, statErr := os.Stat(path); statErr == nil {
		if _, err := core.CreateBackup(path); err != nil {
			return fmt.Errorf("backup failed, save aborted: %w", err)
		}
		if err := core.PruneBackups(path, 10); err != nil {
			fmt.Printf("Warning: failed to prune old backups: %v\n", err)
		}
	}

	// Apply target platform — enables cross-platform conversion.
	origPlatform := a.save.Platform
	a.save.Platform = core.Platform(platform)
	if platform == "PC" && origPlatform == core.PlatformPS {
		// PS4 → PC: enable AES encryption with a fresh random IV.
		iv := make([]byte, 16)
		if _, err := rand.Read(iv); err != nil {
			return fmt.Errorf("failed to generate IV for encryption: %w", err)
		}
		a.save.IV = iv
		a.save.Encrypted = true
	}
	if platform == "PS4" {
		a.save.Encrypted = false
	}

	if err := a.save.SaveFile(path); err != nil {
		return err
	}
	a.clearAllUndoStacks()
	return nil
}

// GetItemList returns a list of items for a given category, filtered by the loaded save's platform.
func (a *App) GetItemList(category string) []db.ItemEntry {
	platform := "PS4"
	if a.save != nil {
		platform = string(a.save.Platform)
	}
	return db.GetItemsByCategory(category, platform)
}

// AddItemsToCharacter adds multiple items from the database to a character slot.
// upgrade25 applies to weapons/bows/shields/staffs/seals with maxUpgrade=25.
// upgrade10 applies to weapons with maxUpgrade=10 (boss weapons, cannot be infused).
// infuseOffset is added to infusable weapons (maxUpgrade=25) as a weapon affinity offset.
// upgradeAsh applies to spirit ashes (category="ashes").
// invQty / storageQty: 0 = skip, -1 = item's MaxInventory/MaxStorage, >0 = exact qty (capped to max).
func (a *App) AddItemsToCharacter(charIdx int, itemIDs []uint32, upgrade25, upgrade10, infuseOffset, upgradeAsh, invQty, storageQty int) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if charIdx < 0 || charIdx >= 10 {
		return fmt.Errorf("invalid character index")
	}

	a.pushUndo(charIdx)

	slot := &a.save.Slots[charIdx]

	for _, id := range itemIDs {
		itemData, _ := db.GetItemDataFuzzy(id)
		finalID := id
		switch {
		case itemData.Category == "ashes":
			finalID = id + uint32(upgradeAsh)
		case itemData.MaxUpgrade == 25:
			finalID = id + uint32(infuseOffset) + uint32(upgrade25)
		case itemData.MaxUpgrade == 10:
			finalID = id + uint32(upgrade10)
		}

		// Resolve actual quantities for this item
		actualInv := resolveQty(invQty, int(itemData.MaxInventory))
		actualStorage := resolveQty(storageQty, int(itemData.MaxStorage))

		// Arrows/bolts are stackable despite weapon-like IDs (0x02.../0x03...).
		forceStackable := db.IsArrowID(finalID)

		if err := core.AddItemsToSlot(slot, []uint32{finalID}, actualInv, actualStorage, forceStackable); err != nil {
			return err
		}
	}
	return nil
}

// RemoveItemsFromCharacter removes items by handle from inventory, storage, or both.
func (a *App) RemoveItemsFromCharacter(charIdx int, handles []uint32, fromInventory, fromStorage bool) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if charIdx < 0 || charIdx >= 10 {
		return fmt.Errorf("invalid character index")
	}
	a.pushUndo(charIdx)

	slot := &a.save.Slots[charIdx]
	for _, handle := range handles {
		if err := core.RemoveItemFromSlot(slot, handle, fromInventory, fromStorage); err != nil {
			return err
		}
	}
	return nil
}

// resolveQty converts a qty directive into an actual quantity.
// qty=0 → 0 (skip); qty=-1 → max; qty>0 → min(qty, max).
func resolveQty(qty, max int) int {
	if qty == 0 || max == 0 {
		return 0
	}
	if qty < 0 {
		return max
	}
	if qty > max {
		return max
	}
	return qty
}

// GetInfuseTypes returns all weapon infusion types with their ID offsets
func (a *App) GetInfuseTypes() []db.InfuseType {
	return db.GetInfuseTypes()
}

// GetAllGraces returns all Sites of Grace (no visited state)
func (a *App) GetAllGraces() []db.GraceEntry {
	return db.GetAllGraces()
}

// GetGraces returns all Sites of Grace with visited state from the specified character slot
func (a *App) GetGraces(slotIndex int) ([]db.GraceEntry, error) {
	if a.save == nil {
		return nil, fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return nil, fmt.Errorf("invalid slot index")
	}

	slot := &a.save.Slots[slotIndex]
	graces := db.GetAllGraces()

	if slot.EventFlagsOffset > 0 && slot.EventFlagsOffset < len(slot.Data) {
		flags := slot.Data[slot.EventFlagsOffset:]
		for i := range graces {
			visited, err := db.GetEventFlag(flags, graces[i].ID)
			if err != nil {
				fmt.Printf("Warning: grace %d (%s): %v\n", graces[i].ID, graces[i].Name, err)
				continue
			}
			graces[i].Visited = visited
		}
	}

	return graces, nil
}

// SetGraceVisited sets or clears the visited flag for a Site of Grace in the specified character slot
func (a *App) SetGraceVisited(slotIndex int, graceID uint32, visited bool) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return fmt.Errorf("invalid slot index")
	}

	a.pushUndo(slotIndex)

	slot := &a.save.Slots[slotIndex]
	if slot.EventFlagsOffset <= 0 || slot.EventFlagsOffset >= len(slot.Data) {
		return fmt.Errorf("event flags offset not computed for slot %d", slotIndex)
	}

	flags := slot.Data[slot.EventFlagsOffset:]
	if err := db.SetEventFlag(flags, graceID, visited); err != nil {
		return fmt.Errorf("failed to set grace %d: %w", graceID, err)
	}
	return nil
}

// GetBosses returns all boss encounters with defeated state from the specified character slot
func (a *App) GetBosses(slotIndex int) ([]db.BossEntry, error) {
	if a.save == nil {
		return nil, fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return nil, fmt.Errorf("invalid slot index")
	}

	slot := &a.save.Slots[slotIndex]
	bosses := db.GetAllBosses()

	if slot.EventFlagsOffset > 0 && slot.EventFlagsOffset < len(slot.Data) {
		flags := slot.Data[slot.EventFlagsOffset:]
		for i := range bosses {
			defeated, err := db.GetEventFlag(flags, bosses[i].ID)
			if err != nil {
				continue
			}
			bosses[i].Defeated = defeated
		}
	}

	return bosses, nil
}

// SetBossDefeated sets or clears the defeated flag for a boss in the specified character slot
func (a *App) SetBossDefeated(slotIndex int, bossID uint32, defeated bool) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return fmt.Errorf("invalid slot index")
	}

	a.pushUndo(slotIndex)

	slot := &a.save.Slots[slotIndex]
	if slot.EventFlagsOffset <= 0 || slot.EventFlagsOffset >= len(slot.Data) {
		return fmt.Errorf("event flags offset not computed for slot %d", slotIndex)
	}

	flags := slot.Data[slot.EventFlagsOffset:]
	if err := db.SetEventFlag(flags, bossID, defeated); err != nil {
		return fmt.Errorf("failed to set boss %d: %w", bossID, err)
	}
	return nil
}

// GetSummoningPools returns all summoning pools with activation state from the specified character slot
func (a *App) GetSummoningPools(slotIndex int) ([]db.SummoningPoolEntry, error) {
	if a.save == nil {
		return nil, fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return nil, fmt.Errorf("invalid slot index")
	}

	slot := &a.save.Slots[slotIndex]
	pools := db.GetAllSummoningPools()

	if slot.EventFlagsOffset > 0 && slot.EventFlagsOffset < len(slot.Data) {
		flags := slot.Data[slot.EventFlagsOffset:]
		for i := range pools {
			activated, err := db.GetEventFlag(flags, pools[i].ID)
			if err != nil {
				continue
			}
			pools[i].Activated = activated
		}
	}

	return pools, nil
}

// SetSummoningPoolActivated sets or clears the activation flag for a summoning pool
func (a *App) SetSummoningPoolActivated(slotIndex int, poolID uint32, activated bool) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return fmt.Errorf("invalid slot index")
	}

	a.pushUndo(slotIndex)

	slot := &a.save.Slots[slotIndex]
	if slot.EventFlagsOffset <= 0 || slot.EventFlagsOffset >= len(slot.Data) {
		return fmt.Errorf("event flags offset not computed for slot %d", slotIndex)
	}

	flags := slot.Data[slot.EventFlagsOffset:]
	if err := db.SetEventFlag(flags, poolID, activated); err != nil {
		return fmt.Errorf("failed to set summoning pool %d: %w", poolID, err)
	}
	return nil
}

// GetColosseums returns all colosseums with unlock state from the specified character slot
func (a *App) GetColosseums(slotIndex int) ([]db.ColosseumEntry, error) {
	if a.save == nil {
		return nil, fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return nil, fmt.Errorf("invalid slot index")
	}

	slot := &a.save.Slots[slotIndex]
	colosseums := db.GetAllColosseums()

	if slot.EventFlagsOffset > 0 && slot.EventFlagsOffset < len(slot.Data) {
		flags := slot.Data[slot.EventFlagsOffset:]
		for i := range colosseums {
			unlocked, err := db.GetEventFlag(flags, colosseums[i].ID)
			if err != nil {
				continue
			}
			colosseums[i].Unlocked = unlocked
		}
	}

	return colosseums, nil
}

// SetColosseumUnlocked sets or clears the unlock flag for a colosseum
func (a *App) SetColosseumUnlocked(slotIndex int, colosseumID uint32, unlocked bool) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if slotIndex < 0 || slotIndex >= 10 {
		return fmt.Errorf("invalid slot index")
	}

	a.pushUndo(slotIndex)

	slot := &a.save.Slots[slotIndex]
	if slot.EventFlagsOffset <= 0 || slot.EventFlagsOffset >= len(slot.Data) {
		return fmt.Errorf("event flags offset not computed for slot %d", slotIndex)
	}

	flags := slot.Data[slot.EventFlagsOffset:]
	if err := db.SetEventFlag(flags, colosseumID, unlocked); err != nil {
		return fmt.Errorf("failed to set colosseum %d: %w", colosseumID, err)
	}
	return nil
}

// ImportCharacter copies a slot from the source save file to the destination save file
func (a *App) ImportCharacter(srcIdx, destIdx int) error {
	return fmt.Errorf("ImportCharacter is temporarily disabled during architecture refactor")
}

// CloneSlot copies an existing character slot to an empty destination slot within the same save.
func (a *App) CloneSlot(srcIdx, destIdx int) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if srcIdx < 0 || srcIdx >= 10 || destIdx < 0 || destIdx >= 10 {
		return fmt.Errorf("invalid slot index")
	}
	if srcIdx == destIdx {
		return fmt.Errorf("source and destination must differ")
	}
	srcName := core.UTF16ToString(a.save.Slots[srcIdx].Player.CharacterName[:])
	if srcName == "" {
		return fmt.Errorf("source slot %d is empty", srcIdx)
	}
	destName := core.UTF16ToString(a.save.Slots[destIdx].Player.CharacterName[:])
	if destName != "" {
		return fmt.Errorf("destination slot %d is not empty", destIdx)
	}

	a.pushUndo(destIdx)

	src := a.save.Slots[srcIdx]

	// Deep copy Data
	newData := make([]byte, len(src.Data))
	copy(newData, src.Data)
	src.Data = newData

	// Deep copy GaMap
	newGaMap := make(map[uint32]uint32, len(src.GaMap))
	for k, v := range src.GaMap {
		newGaMap[k] = v
	}
	src.GaMap = newGaMap

	a.save.Slots[destIdx] = src
	a.save.ActiveSlots[destIdx] = true
	a.save.ProfileSummaries[destIdx] = a.save.ProfileSummaries[srcIdx]

	return nil
}

// DeleteSlot removes a character from a slot and shifts all subsequent slots down by one.
func (a *App) DeleteSlot(idx int) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if idx < 0 || idx >= 10 {
		return fmt.Errorf("invalid slot index")
	}
	name := core.UTF16ToString(a.save.Slots[idx].Player.CharacterName[:])
	if name == "" {
		return fmt.Errorf("slot %d is already empty", idx)
	}

	// Snapshot all affected slots (idx..9) since delete shifts them down
	for i := idx; i < 10; i++ {
		a.pushUndo(i)
	}

	for i := idx; i < 9; i++ {
		a.save.Slots[i] = a.save.Slots[i+1]
		a.save.ActiveSlots[i] = a.save.ActiveSlots[i+1]
		a.save.ProfileSummaries[i] = a.save.ProfileSummaries[i+1]
	}

	// Zero out the last slot with a valid MagicOffset to prevent Write() from panicking
	a.save.Slots[9] = core.SaveSlot{
		Data:        make([]byte, 0x280000),
		GaMap:       make(map[uint32]uint32),
		MagicOffset: 0x15420 + 432,
	}
	a.save.ActiveSlots[9] = false
	a.save.ProfileSummaries[9] = core.ProfileSummary{}

	return nil
}

// GetActiveSlots returns the activity status of all 10 slots
func (a *App) GetActiveSlots() []bool {
	active := make([]bool, 10)
	if a.save == nil {
		return active
	}
	for i := 0; i < 10; i++ {
		// Slot is active if it has a name (Python method)
		name := core.UTF16ToString(a.save.Slots[i].Player.CharacterName[:])
		active[i] = name != ""
	}
	return active
}

// GetCharacterNames returns the names of all 10 characters
func (a *App) GetCharacterNames() []string {
	names := make([]string, 10)
	if a.save == nil {
		for i := 0; i < 10; i++ {
			names[i] = "Empty Slot"
		}
		return names
	}
	for i := 0; i < 10; i++ {
		// Get name directly from the character slot (Python method)
		name := core.UTF16ToString(a.save.Slots[i].Player.CharacterName[:])
		if name == "" {
			names[i] = "Empty Slot"
		} else {
			names[i] = name
		}
	}
	return names
}

// GetSourceActiveSlots returns the activity status of all 10 slots in the source file
func (a *App) GetSourceActiveSlots() []bool {
	active := make([]bool, 10)
	if a.sourceSave == nil {
		return active
	}
	for i := 0; i < 10; i++ {
		name := core.UTF16ToString(a.sourceSave.Slots[i].Player.CharacterName[:])
		active[i] = name != ""
	}
	return active
}

// SetSlotActivity toggles a slot's active status
func (a *App) SetSlotActivity(index int, active bool) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	a.save.ActiveSlots[index] = active
	return nil
}

// GetSteamID returns the global SteamID from UserData10
func (a *App) GetSteamID() uint64 {
	if a.save == nil {
		return 0
	}
	return a.save.SteamID
}

// SetSteamID updates the global SteamID
func (a *App) SetSteamID(id uint64) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	a.save.SteamID = id
	return nil
}

// GetSteamIDString returns the SteamID as a decimal string to avoid JS float64 precision loss.
func (a *App) GetSteamIDString() string {
	if a.save == nil {
		return ""
	}
	return strconv.FormatUint(a.save.SteamID, 10)
}

// SetSteamIDFromString parses a decimal string and updates the SteamID.
func (a *App) SetSteamIDFromString(s string) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid SteamID: %w", err)
	}
	a.save.SteamID = id
	return nil
}

// pushUndo takes a deep-copy snapshot of the given slot and pushes it onto the undo stack.
func (a *App) pushUndo(idx int) {
	slot := &a.save.Slots[idx]

	// Deep copy Data
	dataCopy := make([]byte, len(slot.Data))
	copy(dataCopy, slot.Data)

	// Deep copy GaMap
	gaMapCopy := make(map[uint32]uint32, len(slot.GaMap))
	for k, v := range slot.GaMap {
		gaMapCopy[k] = v
	}

	snap := slotSnapshot{
		Data:              dataCopy,
		Version:           slot.Version,
		Player:            slot.Player,
		GaMap:             gaMapCopy,
		Inventory:         slot.Inventory.Clone(),
		Storage:           slot.Storage.Clone(),
		Warnings:          append([]string{}, slot.Warnings...),
		MagicOffset:       slot.MagicOffset,
		InventoryEnd:      slot.InventoryEnd,
		EventFlagsOffset:  slot.EventFlagsOffset,
		PlayerDataOffset:  slot.PlayerDataOffset,
		FaceDataOffset:    slot.FaceDataOffset,
		StorageBoxOffset:  slot.StorageBoxOffset,
		IngameTimerOffset: slot.IngameTimerOffset,
		GaItemDataOffset:  slot.GaItemDataOffset,
	}

	stack := a.undoStacks[idx]
	if len(stack) >= maxUndoDepth {
		stack = stack[1:] // drop oldest
	}
	a.undoStacks[idx] = append(stack, snap)
}

// RevertSlot pops the last snapshot from the undo stack and restores the slot.
func (a *App) RevertSlot(idx int) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if idx < 0 || idx >= 10 {
		return fmt.Errorf("invalid slot index")
	}
	stack := a.undoStacks[idx]
	if len(stack) == 0 {
		return fmt.Errorf("nothing to undo for slot %d", idx)
	}

	snap := stack[len(stack)-1]
	a.undoStacks[idx] = stack[:len(stack)-1]

	slot := &a.save.Slots[idx]
	slot.Data = snap.Data
	slot.Version = snap.Version
	slot.Player = snap.Player
	slot.GaMap = snap.GaMap
	slot.Inventory = snap.Inventory
	slot.Storage = snap.Storage
	slot.Warnings = snap.Warnings
	slot.MagicOffset = snap.MagicOffset
	slot.InventoryEnd = snap.InventoryEnd
	slot.EventFlagsOffset = snap.EventFlagsOffset
	slot.PlayerDataOffset = snap.PlayerDataOffset
	slot.FaceDataOffset = snap.FaceDataOffset
	slot.StorageBoxOffset = snap.StorageBoxOffset
	slot.IngameTimerOffset = snap.IngameTimerOffset
	slot.GaItemDataOffset = snap.GaItemDataOffset

	return nil
}

// GetUndoDepth returns the number of undo snapshots available for a slot.
func (a *App) GetUndoDepth(idx int) int {
	if a.save == nil || idx < 0 || idx >= 10 {
		return 0
	}
	return len(a.undoStacks[idx])
}

// clearAllUndoStacks resets all undo history (called on file load/save).
func (a *App) clearAllUndoStacks() {
	for i := range a.undoStacks {
		a.undoStacks[i] = nil
	}
}

// ---------- 21.4 Save file diffing ----------

// DiffEntry represents a single change between original and current save state.
type DiffEntry struct {
	Category string `json:"category"` // "stat", "item", "storage", "grace"
	Action   string `json:"action"`   // "changed", "added", "removed"
	Field    string `json:"field"`    // field or item name
	OldValue string `json:"oldValue"`
	NewValue string `json:"newValue"`
}

// SlotDiffSummary is a quick overview for one slot.
type SlotDiffSummary struct {
	SlotIndex  int    `json:"slotIndex"`
	CharName   string `json:"charName"`
	ChangeCount int   `json:"changeCount"`
}

// GetSlotDiff compares the current state of a slot against the original loaded state.
func (a *App) GetSlotDiff(idx int) ([]DiffEntry, error) {
	if a.save == nil || a.sourceSave == nil {
		return nil, fmt.Errorf("no save loaded")
	}
	if idx < 0 || idx >= 10 {
		return nil, fmt.Errorf("invalid slot index")
	}

	cur := &a.save.Slots[idx]
	orig := &a.sourceSave.Slots[idx]
	var diffs []DiffEntry

	// --- Stats ---
	type statField struct {
		name string
		cur  uint32
		orig uint32
	}
	stats := []statField{
		{"Level", cur.Player.Level, orig.Player.Level},
		{"Vigor", cur.Player.Vigor, orig.Player.Vigor},
		{"Mind", cur.Player.Mind, orig.Player.Mind},
		{"Endurance", cur.Player.Endurance, orig.Player.Endurance},
		{"Strength", cur.Player.Strength, orig.Player.Strength},
		{"Dexterity", cur.Player.Dexterity, orig.Player.Dexterity},
		{"Intelligence", cur.Player.Intelligence, orig.Player.Intelligence},
		{"Faith", cur.Player.Faith, orig.Player.Faith},
		{"Arcane", cur.Player.Arcane, orig.Player.Arcane},
		{"Souls", cur.Player.Souls, orig.Player.Souls},
	}
	for _, s := range stats {
		if s.cur != s.orig {
			diffs = append(diffs, DiffEntry{
				Category: "stat",
				Action:   "changed",
				Field:    s.name,
				OldValue: strconv.FormatUint(uint64(s.orig), 10),
				NewValue: strconv.FormatUint(uint64(s.cur), 10),
			})
		}
	}

	curName := core.UTF16ToString(cur.Player.CharacterName[:])
	origName := core.UTF16ToString(orig.Player.CharacterName[:])
	if curName != origName {
		diffs = append(diffs, DiffEntry{
			Category: "stat",
			Action:   "changed",
			Field:    "Name",
			OldValue: origName,
			NewValue: curName,
		})
	}

	// --- Inventory diff ---
	diffs = append(diffs, diffInventory("item", cur.Inventory, orig.Inventory)...)

	// --- Storage diff ---
	diffs = append(diffs, diffInventory("storage", cur.Storage, orig.Storage)...)

	// --- Graces diff ---
	diffs = append(diffs, a.diffGraces(idx)...)

	// --- Boss diff ---
	diffs = append(diffs, a.diffBosses(idx)...)

	return diffs, nil
}

// diffInventory compares two EquipInventoryData and returns DiffEntries.
func diffInventory(category string, cur, orig core.EquipInventoryData) []DiffEntry {
	var diffs []DiffEntry

	// Build maps: GaItemHandle → item for quick lookup
	type itemInfo struct {
		qty   uint32
		name  string
	}
	buildMap := func(items []core.InventoryItem) map[uint32]itemInfo {
		m := make(map[uint32]itemInfo, len(items))
		for _, it := range items {
			if it.GaItemHandle == 0 {
				continue
			}
			name := resolveItemName(it.GaItemHandle)
			existing, ok := m[it.GaItemHandle]
			if ok {
				existing.qty += it.Quantity
				m[it.GaItemHandle] = existing
			} else {
				m[it.GaItemHandle] = itemInfo{qty: it.Quantity, name: name}
			}
		}
		return m
	}

	origAll := append(orig.CommonItems, orig.KeyItems...)
	curAll := append(cur.CommonItems, cur.KeyItems...)
	origMap := buildMap(origAll)
	curMap := buildMap(curAll)

	// Added or changed
	for handle, ci := range curMap {
		oi, existed := origMap[handle]
		if !existed {
			diffs = append(diffs, DiffEntry{
				Category: category,
				Action:   "added",
				Field:    ci.name,
				NewValue: "×" + strconv.FormatUint(uint64(ci.qty), 10),
			})
		} else if ci.qty != oi.qty {
			diffs = append(diffs, DiffEntry{
				Category: category,
				Action:   "changed",
				Field:    ci.name,
				OldValue: "×" + strconv.FormatUint(uint64(oi.qty), 10),
				NewValue: "×" + strconv.FormatUint(uint64(ci.qty), 10),
			})
		}
	}

	// Removed
	for handle, oi := range origMap {
		if _, exists := curMap[handle]; !exists {
			diffs = append(diffs, DiffEntry{
				Category: category,
				Action:   "removed",
				Field:    oi.name,
				OldValue: "×" + strconv.FormatUint(uint64(oi.qty), 10),
			})
		}
	}

	return diffs
}

// resolveItemName tries to get a human-readable name for an inventory item handle.
func resolveItemName(gaItemHandle uint32) string {
	entry, _ := db.GetItemDataFuzzy(gaItemHandle)
	if entry.Name != "" {
		return entry.Name
	}
	return fmt.Sprintf("Item 0x%X", gaItemHandle)
}

// diffGraces compares grace event flags between source and current save.
func (a *App) diffGraces(idx int) []DiffEntry {
	cur := &a.save.Slots[idx]
	orig := &a.sourceSave.Slots[idx]

	if cur.EventFlagsOffset <= 0 || orig.EventFlagsOffset <= 0 {
		return nil
	}
	if cur.EventFlagsOffset >= len(cur.Data) || orig.EventFlagsOffset >= len(orig.Data) {
		return nil
	}

	curFlags := cur.Data[cur.EventFlagsOffset:]
	origFlags := orig.Data[orig.EventFlagsOffset:]
	graces := db.GetAllGraces()

	var diffs []DiffEntry
	for _, g := range graces {
		curVisited, err1 := db.GetEventFlag(curFlags, g.ID)
		origVisited, err2 := db.GetEventFlag(origFlags, g.ID)
		if err1 != nil || err2 != nil {
			continue
		}
		if curVisited != origVisited {
			action := "added"
			if !curVisited {
				action = "removed"
			}
			diffs = append(diffs, DiffEntry{
				Category: "grace",
				Action:   action,
				Field:    g.Name,
			})
		}
	}
	return diffs
}

// diffBosses compares boss defeat event flags between source and current save.
func (a *App) diffBosses(idx int) []DiffEntry {
	cur := &a.save.Slots[idx]
	orig := &a.sourceSave.Slots[idx]

	if cur.EventFlagsOffset <= 0 || orig.EventFlagsOffset <= 0 {
		return nil
	}
	if cur.EventFlagsOffset >= len(cur.Data) || orig.EventFlagsOffset >= len(orig.Data) {
		return nil
	}

	curFlags := cur.Data[cur.EventFlagsOffset:]
	origFlags := orig.Data[orig.EventFlagsOffset:]
	bosses := db.GetAllBosses()

	var diffs []DiffEntry
	for _, b := range bosses {
		curDefeated, err1 := db.GetEventFlag(curFlags, b.ID)
		origDefeated, err2 := db.GetEventFlag(origFlags, b.ID)
		if err1 != nil || err2 != nil {
			continue
		}
		if curDefeated != origDefeated {
			action := "added"
			if !curDefeated {
				action = "removed"
			}
			diffs = append(diffs, DiffEntry{
				Category: "boss",
				Action:   action,
				Field:    b.Name + " (" + b.Region + ")",
			})
		}
	}
	return diffs
}

// GetSaveDiffSummary returns a quick change-count overview for all active slots.
func (a *App) GetSaveDiffSummary() ([]SlotDiffSummary, error) {
	if a.save == nil || a.sourceSave == nil {
		return nil, fmt.Errorf("no save loaded")
	}

	var summaries []SlotDiffSummary
	for i := 0; i < 10; i++ {
		if !a.save.ActiveSlots[i] {
			continue
		}
		diffs, err := a.GetSlotDiff(i)
		if err != nil {
			continue
		}
		name := core.UTF16ToString(a.save.Slots[i].Player.CharacterName[:])
		summaries = append(summaries, SlotDiffSummary{
			SlotIndex:   i,
			CharName:    name,
			ChangeCount: len(diffs),
		})
	}
	return summaries, nil
}

// Dummy method to force Wails to export types
func (a *App) _forceExportTypes() (db.GraceEntry, db.BossEntry, db.ItemEntry, DiffEntry, SlotDiffSummary) {
	return db.GraceEntry{}, db.BossEntry{}, db.ItemEntry{}, DiffEntry{}, SlotDiffSummary{}
}
