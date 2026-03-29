package vm

import (
	"fmt"
	"unicode/utf16"
	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"github.com/oisis/EldenRing-SaveEditor/backend/db"
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
		Inventory:    []ItemViewModel{},
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

	// Map Inventory using GaItems
	handleToID := make(map[uint32]uint32)
	for _, gi := range slot.GaItems {
		if gi.Handle != 0 && gi.Handle != 0xFFFFFFFF {
			handleToID[gi.Handle] = gi.ItemID
		}
	}

	// Process Common Items
	for _, item := range slot.EquipInventoryData.CommonItems {
		if item.GaItemHandle == 0 || item.GaItemHandle == 0xFFFFFFFF {
			continue
		}

		itemID, ok := handleToID[item.GaItemHandle]
		if !ok {
			continue
		}

		// Filter out "Unarmed" (110000) if it's noise
		if itemID == 110000 {
			continue
		}

		vm.Inventory = append(vm.Inventory, ItemViewModel{
			Handle:   item.GaItemHandle,
			ID:       itemID,
			Name:     db.GetItemName(itemID),
			Category: db.GetItemCategory(itemID),
			Quantity: item.Quantity,
		})
	}

	// Process Key Items
	for _, item := range slot.EquipInventoryData.KeyItems {
		if item.GaItemHandle == 0 || item.GaItemHandle == 0xFFFFFFFF {
			continue
		}

		itemID, ok := handleToID[item.GaItemHandle]
		if !ok {
			continue
		}

		vm.Inventory = append(vm.Inventory, ItemViewModel{
			Handle:   item.GaItemHandle,
			ID:       itemID,
			Name:     db.GetItemName(itemID),
			Category: db.GetItemCategory(itemID),
			Quantity: item.Quantity,
		})
	}

	return vm, nil
}

// Placeholder for old method to avoid compilation errors during refactor
func MapSlotToVM(slotData []byte) (*CharacterViewModel, error) {
	return nil, fmt.Errorf("use MapParsedSlotToVM instead")
}

func ApplyVMToSlot(vm *CharacterViewModel, slotData []byte) error {
	return fmt.Errorf("apply not implemented in sequential mode yet")
}
