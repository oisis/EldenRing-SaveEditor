package vm

import (
	"unicode/utf16"
	"github.com/oisis/EldenRing-SaveEditor/backend/core"
)

// ItemViewModel represents an item in the inventory for the UI.
type ItemViewModel struct {
	Handle   uint32 `json:"handle"`
	ID       uint32 `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Quantity uint32 `json:"quantity"`
}

// CharacterViewModel represents the editable character data for the UI.
type CharacterViewModel struct {
	Name         string          `json:"name"`
	Level        uint32          `json:"level"`
	Souls        uint32          `json:"souls"`
	Vigor        uint32          `json:"vigor"`
	Mind         uint32          `json:"mind"`
	Endurance    uint32          `json:"endurance"`
	Strength     uint32          `json:"strength"`
	Dexterity    uint32          `json:"dexterity"`
	Intelligence uint32          `json:"intelligence"`
	Faith        uint32          `json:"faith"`
	Arcane       uint32          `json:"arcane"`
	Inventory    []ItemViewModel `json:"inventory"`
	Storage      []ItemViewModel `json:"storage"`
}

// MapParsedSlotToVM extracts character data from the parsed core.SaveSlot structure.
func MapParsedSlotToVM(slot *core.SaveSlot) (*CharacterViewModel, error) {
	data := slot.Player
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
		Inventory:    []ItemViewModel{},
		Storage:      []ItemViewModel{},
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

	// TODO: Implement inventory mapping using the new dynamic offset logic from Python
	// For now, we focus on stats parity.

	return vm, nil
}

// ApplyVMToParsedSlot updates the core.SaveSlot structure from the ViewModel.
func ApplyVMToParsedSlot(vm *CharacterViewModel, slot *core.SaveSlot) error {
	data := &slot.Player
	data.Level = vm.Level
	data.Souls = vm.Souls
	data.Vigor = vm.Vigor
	data.Mind = vm.Mind
	data.Endurance = vm.Endurance
	data.Strength = vm.Strength
	data.Dexterity = vm.Dexterity
	data.Intelligence = vm.Intelligence
	data.Faith = vm.Faith
	data.Arcane = vm.Arcane

	// Encode Name
	u16 := utf16.Encode([]rune(vm.Name))
	for i := 0; i < 16; i++ {
		if i < len(u16) {
			data.CharacterName[i] = u16[i]
		} else {
			data.CharacterName[i] = 0
		}
	}

	return nil
}

// MapSlotToVM and ApplyVMToSlot are kept for compatibility with App.go
func MapSlotToVM(slotData []byte) (*CharacterViewModel, error) {
	// This is a bridge for the new architecture
	r := core.NewReader(slotData)
	slot := &core.SaveSlot{}
	if err := slot.Read(r, "PC"); err != nil {
		return nil, err
	}
	return MapParsedSlotToVM(slot)
}

func ApplyVMToSlot(vm *CharacterViewModel, slotData []byte) error {
	r := core.NewReader(slotData)
	slot := &core.SaveSlot{}
	if err := slot.Read(r, "PC"); err != nil {
		return err
	}
	if err := ApplyVMToParsedSlot(vm, slot); err != nil {
		return err
	}
	// Write changes back to the byte slice
	updated := slot.Write("PC")
	copy(slotData, updated)
	return nil
}
