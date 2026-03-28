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

// GetGracesByRegion returns Sites of Grace grouped by region
func (a *App) GetGracesByRegion() map[string][]db.GraceEntry {
	return db.GetGracesByRegion()
}

// GetSteamID returns the global SteamID from UserData10
func (a *App) GetSteamID() uint64 {
	if a.save == nil {
		return 0
	}
	return 0 
}
