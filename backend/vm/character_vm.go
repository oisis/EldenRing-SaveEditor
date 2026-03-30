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
	Storage      []ItemViewModel `json:"storage"`
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

	// Map Inventory using GaItems
	handleToID := make(map[uint32]uint32)
	for _, gi := range slot.GaItems {
		if gi.Handle != 0 && gi.Handle != 0xFFFFFFFF {
			handleToID[gi.Handle] = gi.ItemID
		}
	}

	// Process Inventory (Common + Key)
	vm.Inventory = mapItems(slot.EquipInventoryData, handleToID)

	// Process Storage (Common + Key)
	vm.Storage = mapItems(slot.StorageInventoryData, handleToID)

	return vm, nil
}

func mapItems(data core.EquipInventoryData, handleToID map[uint32]uint32) []ItemViewModel {
	items := []ItemViewModel{}
	
	// Common Items
	for _, item := range data.CommonItems {
		if item.GaItemHandle == 0 || item.GaItemHandle == 0xFFFFFFFF {
			continue
		}
		if itemID, ok := handleToID[item.GaItemHandle]; ok {
			if itemID == 110000 { continue } // Filter Unarmed
			items = append(items, ItemViewModel{
				Handle:   item.GaItemHandle,
				ID:       itemID,
				Name:     db.GetItemName(itemID),
				Category: db.GetItemCategory(itemID),
				Quantity: item.Quantity,
			})
		}
	}

	// Key Items
	for _, item := range data.KeyItems {
		if item.GaItemHandle == 0 || item.GaItemHandle == 0xFFFFFFFF {
			continue
		}
		if itemID, ok := handleToID[item.GaItemHandle]; ok {
			items = append(items, ItemViewModel{
				Handle:   item.GaItemHandle,
				ID:       itemID,
				Name:     db.GetItemName(itemID),
				Category: db.GetItemCategory(itemID),
				Quantity: item.Quantity,
			})
		}
	}

	return items
}

// ApplyVMToParsedSlot updates the core.SaveSlot structure from the ViewModel.
func ApplyVMToParsedSlot(vm *CharacterViewModel, slot *core.SaveSlot) error {
	data := &slot.PlayerGameData
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

	// Inventory updates are more complex (adding/removing items).
	// For now, we only update quantities of existing items.
	updateQuantities(vm.Inventory, &slot.EquipInventoryData)
	updateQuantities(vm.Storage, &slot.StorageInventoryData)

	return nil
}

func updateQuantities(vmItems []ItemViewModel, data *core.EquipInventoryData) {
	handleToQty := make(map[uint32]uint32)
	for _, item := range vmItems {
		handleToQty[item.Handle] = item.Quantity
	}

	for i := range data.CommonItems {
		if qty, ok := handleToQty[data.CommonItems[i].GaItemHandle]; ok {
			data.CommonItems[i].Quantity = qty
		}
	}
	for i := range data.KeyItems {
		if qty, ok := handleToQty[data.KeyItems[i].GaItemHandle]; ok {
			data.KeyItems[i].Quantity = qty
		}
	}
}

// Placeholder for old method to avoid compilation errors during refactor
func MapSlotToVM(slotData []byte) (*CharacterViewModel, error) {
	return nil, fmt.Errorf("use MapParsedSlotToVM instead")
}

func ApplyVMToSlot(vm *CharacterViewModel, slotData []byte) error {
	return fmt.Errorf("use ApplyVMToParsedSlot instead")
}
