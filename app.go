package main

import (
	"context"
	"fmt"
	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"github.com/oisis/EldenRing-SaveEditor/backend/db"
	"github.com/oisis/EldenRing-SaveEditor/backend/vm"
)

// App struct
type App struct {
	ctx  context.Context
	save *core.SaveFile
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

// OpenSave loads a save file and returns basic info
func (a *App) OpenSave(path string) (string, error) {
	save, err := core.LoadSave(path)
	if err != nil {
		return "", err
	}
	a.save = save
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
	return vm.MapSlotToVM(a.save.Slots[index])
}

// SaveCharacter updates the raw slot data from the ViewModel
func (a *App) SaveCharacter(index int, charVM vm.CharacterViewModel) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	
	// Validate and recalculate level before saving
	charVM.ValidateStats()
	charVM.RecalculateLevel()
	
	return vm.ApplyVMToSlot(&charVM, a.save.Slots[index])
}

// GetItemList returns a list of items for a given category
func (a *App) GetItemList(category string) []db.ItemEntry {
	return db.GetItemsByCategory(category)
}

// GetAllGraces returns all Sites of Grace
func (a *App) GetAllGraces() []db.GraceEntry {
	return db.GetAllGraces()
}

// ImportSlot copies a slot from another save file
func (a *App) ImportSlot(sourcePath string, srcIdx, destIdx int) error {
	if a.save == nil {
		return fmt.Errorf("no destination save loaded")
	}
	
	sourceSave, err := core.LoadSave(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to load source save: %w", err)
	}
	
	return a.save.ImportSlot(sourceSave, srcIdx, destIdx)
}

// GetActiveSlots returns the activity status of all 10 slots
func (a *App) GetActiveSlots() []bool {
	if a.save == nil {
		return make([]bool, 10)
	}
	return a.save.GetActiveSlots()
}

// SetSlotActivity toggles a slot's active status
func (a *App) SetSlotActivity(index int, active bool) error {
	if a.save == nil {
		return fmt.Errorf("no save loaded")
	}
	return a.save.SetSlotActivity(index, active)
}

// GetSteamID returns the global SteamID from UserData10
func (a *App) GetSteamID() uint64 {
	if a.save == nil {
		return 0
	}
	return 0 
}

// Dummy method to force Wails to export types
func (a *App) _forceExportTypes() (db.GraceEntry, db.ItemEntry) {
	return db.GraceEntry{}, db.ItemEntry{}
}
