package backend

import (
	"context"
	"er-save-editor/backend/core"
	"er-save-editor/backend/db"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"unicode/utf16"
)

// EventItem represents a grace or boss with its current status
type EventItem struct {
	ID      uint32 `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// GetGraces returns all graces with their status for a specific slot
func (a *App) GetGraces(slotIndex int) ([]EventItem, error) {
	if a.saveManager.CurrentSave == nil {
		return nil, fmt.Errorf("no save file loaded")
	}

	slot := &a.saveManager.CurrentSave.Slots[slotIndex].Slot
	database := db.GetInstance()
	
	var graces []EventItem
	for _, g := range database.Graces {
		graces = append(graces, EventItem{
			ID:      g.ID,
			Name:    g.Name,
			Enabled: slot.GetEventFlag(g.ID),
		})
	}
	return graces, nil
}

// GetBosses returns all bosses with their status for a specific slot
func (a *App) GetBosses(slotIndex int) ([]EventItem, error) {
	if a.saveManager.CurrentSave == nil {
		return nil, fmt.Errorf("no save file loaded")
	}

	slot := &a.saveManager.CurrentSave.Slots[slotIndex].Slot
	database := db.GetInstance()
	
	var bosses []EventItem
	for _, b := range database.Bosses {
		bosses = append(bosses, EventItem{
			ID:      b.ID,
			Name:    b.Name,
			Enabled: slot.GetEventFlag(b.ID),
		})
	}
	return bosses, nil
}

// SetEventFlag updates a grace or boss status and saves the file
func (a *App) SetEventFlag(slotIndex int, flagID uint32, enabled bool) error {
	if a.saveManager.CurrentSave == nil {
		return fmt.Errorf("no save file loaded")
	}

	slot := &a.saveManager.CurrentSave.Slots[slotIndex].Slot
	slot.SetEventFlag(flagID, enabled)
	
	return a.saveManager.SaveFile()
}

// CharacterInfo represents basic character data for the UI
type CharacterInfo struct {
	SlotIndex int    `json:"slotIndex"`
	Name      string `json:"name"`
	Level     uint32 `json:"level"`
	IsActive  bool   `json:"isActive"`
}

// App struct
type App struct {
	ctx         context.Context
	saveManager *core.SaveManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		saveManager: core.NewSaveManager(),
	}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// OpenSaveFile opens a file dialog and loads the selected save file
func (a *App) OpenSaveFile() (string, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Elden Ring Save File (.sl2)",
		Filters: []runtime.FileFilter{
			{DisplayName: "Elden Ring Save (*.sl2)", Pattern: "*.sl2"},
		},
	})

	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "", nil
	}

	if err := a.saveManager.LoadSave(filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

// GetCharacters returns a list of all characters in the loaded save
func (a *App) GetCharacters() ([]CharacterInfo, error) {
	if a.saveManager.CurrentSave == nil {
		return nil, fmt.Errorf("no save file loaded")
	}

	var characters []CharacterInfo
	save := a.saveManager.CurrentSave

	for i := 0; i < 10; i++ {
		isActive := save.UserData10.ActiveSlots[i] == 0x01
		summary := save.UserData10.ProfileSummary[i]

		// Convert UTF-16 name to string
		name := decodeUTF16(summary.CharacterName[:])

		characters = append(characters, CharacterInfo{
			SlotIndex: i,
			Name:      name,
			Level:     summary.Level,
			IsActive:  isActive,
		})
	}

	return characters, nil
}

func decodeUTF16(b []byte) string {
	u16s := make([]uint16, len(b)/2)
	for i := range u16s {
		u16s[i] = uint16(b[i*2]) | uint16(b[i*2+1])<<8
	}
	// Trim null terminators
	for i, v := range u16s {
		if v == 0 {
			u16s = u16s[:i]
			break
		}
	}
	return string(utf16.Decode(u16s))
}

// CharacterDetails represents full character stats for editing
type CharacterDetails struct {
	SlotIndex    int    `json:"slotIndex"`
	Name         string `json:"name"`
	Level        uint32 `json:"level"`
	Vigor        uint32 `json:"vigor"`
	Mind         uint32 `json:"mind"`
	Endurance    uint32 `json:"endurance"`
	Strength     uint32 `json:"strength"`
	Dexterity    uint32 `json:"dexterity"`
	Intelligence uint32 `json:"intelligence"`
	Faith        uint32 `json:"faith"`
	Arcane       uint32 `json:"arcane"`
	Souls        uint32 `json:"souls"`
}

// GetCharacterDetails returns full stats for a specific slot
func (a *App) GetCharacterDetails(slotIndex int) (*CharacterDetails, error) {
	if a.saveManager.CurrentSave == nil {
		return nil, fmt.Errorf("no save file loaded")
	}

	slot := &a.saveManager.CurrentSave.Slots[slotIndex].Slot
	pgd, err := slot.GetPlayerGameData()
	if err != nil {
		return nil, err
	}

	return &CharacterDetails{
		SlotIndex:    slotIndex,
		Name:         decodeUTF16(pgd.CharacterName[:]),
		Level:        pgd.Level,
		Vigor:        pgd.Vigor,
		Mind:         pgd.Mind,
		Endurance:    pgd.Endurance,
		Strength:     pgd.Strength,
		Dexterity:    pgd.Dexterity,
		Intelligence: pgd.Intelligence,
		Faith:        pgd.Faith,
		Arcane:       pgd.Arcane,
		Souls:        pgd.Souls,
	}, nil
}

// SaveCharacterDetails updates character stats and writes the file
func (a *App) SaveCharacterDetails(details CharacterDetails) error {
	if a.saveManager.CurrentSave == nil {
		return fmt.Errorf("no save file loaded")
	}

	slot := &a.saveManager.CurrentSave.Slots[details.SlotIndex].Slot
	pgd, err := slot.GetPlayerGameData()
	if err != nil {
		return err
	}

	// Update stats
	pgd.Vigor = details.Vigor
	pgd.Mind = details.Mind
	pgd.Endurance = details.Endurance
	pgd.Strength = details.Strength
	pgd.Dexterity = details.Dexterity
	pgd.Intelligence = details.Intelligence
	pgd.Faith = details.Faith
	pgd.Arcane = details.Arcane
	pgd.Souls = details.Souls

	// Recalculate level (Level = Sum of attributes - 79)
	pgd.Level = pgd.Vigor + pgd.Mind + pgd.Endurance + pgd.Strength + pgd.Dexterity + pgd.Intelligence + pgd.Faith + pgd.Arcane - 79

	// Write back to slot
	if err := slot.SetPlayerGameData(pgd); err != nil {
		return err
	}

	// Update ProfileSummary in UserData10 so the menu shows new level
	a.saveManager.CurrentSave.UserData10.ProfileSummary[details.SlotIndex].Level = pgd.Level

	// Save the entire file (includes backup and checksum updates)
	return a.saveManager.SaveFile()
}

// GetSteamID returns the SteamID from the current save
func (a *App) GetSteamID() (uint64, error) {
	if a.saveManager.CurrentSave == nil {
		return 0, fmt.Errorf("no save file loaded")
	}
	return a.saveManager.CurrentSave.UserData10.SteamID, nil
}

// SaveSteamID updates the SteamID and saves the file
func (a *App) SaveSteamID(steamID uint64) error {
	if a.saveManager.CurrentSave == nil {
		return fmt.Errorf("no save file loaded")
	}

	a.saveManager.CurrentSave.UserData10.SteamID = steamID
	return a.saveManager.SaveFile()
}

// AddBulkItems adds all items from a specific category to a character slot
func (a *App) AddBulkItems(slotIndex int, category string) (int, error) {
	if a.saveManager.CurrentSave == nil {
		return 0, fmt.Errorf("no save file loaded")
	}

	database := db.GetInstance()
	var itemIDs []uint32

	// Find items in the requested category
	var groups []db.ItemGroup
	switch category {
	case "Weapons":
		groups = database.Weapons
	case "Armors":
		groups = database.Armors
	case "Items":
		groups = database.Items
	case "Talismans":
		groups = database.Talismans
	}

	for _, group := range groups {
		for _, item := range group.Items {
			itemIDs = append(itemIDs, item.ID)
		}
	}

	if len(itemIDs) == 0 {
		return 0, fmt.Errorf("no items found in category %s", category)
	}

	slot := &a.saveManager.CurrentSave.Slots[slotIndex].Slot
	count := slot.AddBulkItems(itemIDs)

	// Save the file to apply changes
	if err := a.saveManager.SaveFile(); err != nil {
		return 0, err
	}

	return count, nil
}

// ImportCharacter opens a file dialog to select a source save and imports a character slot
func (a *App) ImportCharacter(targetSlotIndex int) error {
	if a.saveManager.CurrentSave == nil {
		return fmt.Errorf("no save file loaded")
	}

	// 1. Select source file
	sourcePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Source Elden Ring Save File (.sl2)",
		Filters: []runtime.FileFilter{
			{DisplayName: "Elden Ring Save (*.sl2)", Pattern: "*.sl2"},
		},
	})
	if err != nil || sourcePath == "" {
		return err
	}

	// 2. Load source save
	sourceManager := core.NewSaveManager()
	if err := sourceManager.LoadSave(sourcePath); err != nil {
		return fmt.Errorf("failed to load source save: %v", err)
	}

	// 3. For now, we'll just take the first active slot from source
	// In a full version, we could show a dialog to pick the source slot
	sourceSlotIndex := -1
	for i := 0; i < 10; i++ {
		if sourceManager.CurrentSave.UserData10.ActiveSlots[i] == 0x01 {
			sourceSlotIndex = i
			break
		}
	}

	if sourceSlotIndex == -1 {
		return fmt.Errorf("source save has no active characters")
	}

	// 4. Copy the slot data
	sourceSlot := sourceManager.CurrentSave.Slots[sourceSlotIndex].Slot
	a.saveManager.CurrentSave.Slots[targetSlotIndex].Slot = sourceSlot

	// 5. Copy ProfileSummary and set active flag
	sourceSummary := sourceManager.CurrentSave.UserData10.ProfileSummary[sourceSlotIndex]
	a.saveManager.CurrentSave.UserData10.ProfileSummary[targetSlotIndex] = sourceSummary
	a.saveManager.CurrentSave.UserData10.ActiveSlots[targetSlotIndex] = 0x01

	// 6. Save the file
	return a.saveManager.SaveFile()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
