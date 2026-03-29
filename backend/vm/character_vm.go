package vm

import (
	"fmt"
	"unicode/utf16"
	"github.com/oisis/EldenRing-SaveEditor/backend/core"
)

// CharacterViewModel represents the editable character data for the UI.
type CharacterViewModel struct {
	Name         string `json:"name"`
	Level        uint32 `json:"level"`
	Souls        uint32 `json:"souls"`
	Vigor        uint32 `json:"vigor"`
	Mind         uint32 `json:"mind"`
	Endurance    uint32 `json:"endurance"`
	Strength     uint32 `json:"strength"`
	Dexterity    uint32 `json:"dexterity"`
	Intelligence uint32 `json:"intelligence"`
	Faith        uint32 `json:"faith"`
	Arcane       uint32 `json:"arcane"`
}

// MapParsedSlotToVM extracts character data from the parsed core.SaveSlot structure.
func MapParsedSlotToVM(slot *core.SaveSlot) (*CharacterViewModel, error) {
	data := slot.PlayerGameData
	vm := &CharacterViewModel{
		Level:        data.Level,
		Souls:        data.Souls,
		Vigor:        data.Vigor,
		Mind:         data.Mind,
		Endurance:    data.Endurance,
		Strength:     data.Strength,
		Dexterity:    data.Dexterity,
		Intelligence: data.Intelligence,
		Faith:        data.Faith,
		Arcane:       data.Arcane,
	}

	// Decode Name
	u16 := data.CharacterName[:]
	// Trim null terminator
	for i, v := range u16 {
		if v == 0 {
			u16 = u16[:i]
			break
		}
	}
	vm.Name = string(utf16.Decode(u16))

	return vm, nil
}

// Placeholder for old method to avoid compilation errors during refactor
func MapSlotToVM(slotData []byte) (*CharacterViewModel, error) {
	return nil, fmt.Errorf("use MapParsedSlotToVM instead")
}

func ApplyVMToSlot(vm *CharacterViewModel, slotData []byte) error {
	return fmt.Errorf("apply not implemented in sequential mode yet")
}
