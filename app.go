package main

import (
	"context"
	"fmt"
	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"github.com/oisis/EldenRing-SaveEditor/backend/db"
	"github.com/oisis/EldenRing-SaveEditor/backend/vm"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	save       *core.SaveFile
	sourceSave *core.SaveFile
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

	// Create backup before writing
	if _, err := core.CreateBackup(path); err != nil {
		fmt.Printf("Warning: failed to create backup: %v\n", err)
	}

	return a.save.SaveFile(path)
}

// GetItemList returns a list of items for a given category
func (a *App) GetItemList(category string) []db.ItemEntry {
	return db.GetItemsByCategory(category)
}

// AddItemsToCharacter adds multiple items from the database to a character slot
func (a *App) AddItemsToCharacter(charIdx int, itemIDs []uint32, upgradeLevel int, invMax, storageMax bool) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	if charIdx < 0 || charIdx >= 10 {
		return fmt.Errorf("invalid character index")
	}

	slot := &a.save.Slots[charIdx]
	return core.AddItemsToSlot(slot, itemIDs, upgradeLevel, invMax, storageMax)
}

// GetAllGraces returns all Sites of Grace
func (a *App) GetAllGraces() []db.GraceEntry {
	return db.GetAllGraces()
}

// ImportCharacter copies a slot from the source save file to the destination save file
func (a *App) ImportCharacter(srcIdx, destIdx int) error {
	return fmt.Errorf("ImportCharacter is temporarily disabled during architecture refactor")
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

// Dummy method to force Wails to export types
func (a *App) _forceExportTypes() (db.GraceEntry, db.ItemEntry) {
	return db.GraceEntry{}, db.ItemEntry{}
}
