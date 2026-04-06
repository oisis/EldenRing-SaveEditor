package vm

import (
	"fmt"
	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"github.com/oisis/EldenRing-SaveEditor/backend/db"
	"unicode/utf16"
)

type ItemViewModel struct {
	Handle       uint32 `json:"handle"`
	ID           uint32 `json:"id"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	SubCategory  string `json:"subCategory"`
	Quantity     uint32 `json:"quantity"`
	MaxInventory uint32 `json:"maxInventory"`
	MaxStorage   uint32 `json:"maxStorage"`
	MaxUpgrade   uint32 `json:"maxUpgrade"`
	IconPath     string `json:"iconPath"`
}

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

	vm.Name = core.UTF16ToString(data.CharacterName[:])

	// Map Inventory
	vm.Inventory = mapItems(slot.Inventory, slot.GaMap)

	// Map Storage
	vm.Storage = mapItems(slot.Storage, slot.GaMap)

	return vm, nil
}

func mapItems(data core.EquipInventoryData, gaMap map[uint32]uint32) []ItemViewModel {
	items := []ItemViewModel{}

	processItem := func(item core.InventoryItem) {
		if item.GaItemHandle == 0 || item.GaItemHandle == 0xFFFFFFFF {
			return
		}

		category := db.GetItemCategoryFromHandle(item.GaItemHandle)
		var itemID uint32
		var ok bool

		if category == "Weapon" || category == "Armor" || category == "Ash of War" {
			// For Weapons, Armor, and AoW, we MUST use the GaMap to find the real ItemID.
			itemID, ok = gaMap[item.GaItemHandle]
		} else if category != "Unknown" {
			// For others (Talisman, Item), the handle IS the ID.
			itemID = item.GaItemHandle
			ok = true
		}

		if ok {
			// Filter Unarmed and Empty
			if itemID == 0 || itemID == 110000 {
				return
			}

			itemData := db.GetItemData(itemID, category)
			name := itemData.Name

			// Strict filtering: skip items that are not in our database (Unknown)
			// to avoid garbage data from misaligned offsets.
			if name == "" || name == fmt.Sprintf("Unknown Item (0x%X)", itemID) ||
				name == fmt.Sprintf("Unknown Weapon (0x%X)", itemID) ||
				name == fmt.Sprintf("Unknown Armor (0x%X)", itemID) ||
				name == fmt.Sprintf("Unknown Talisman (0x%X)", itemID) ||
				name == fmt.Sprintf("Unknown Ash of War (0x%X)", itemID) {
				return
			}

			displayQuantity := item.Quantity
			// For non-stackable items, force quantity to 1.
			if category == "Weapon" || category == "Armor" || category == "Talisman" || category == "Ash of War" {
				displayQuantity = 1
			} else {
				// For stackable items, mask the high bit which is often used by the engine
				displayQuantity = item.Quantity & 0x7FFFFFFF
			}

			items = append(items, ItemViewModel{
				Handle:       item.GaItemHandle,
				ID:           itemID,
				Name:         name,
				Category:     category,
				SubCategory:  db.GetItemSubCategory(itemID, itemData, category),
				Quantity:     displayQuantity,
				MaxInventory: itemData.MaxInventory,
				MaxStorage:   itemData.MaxStorage,
				MaxUpgrade:   itemData.MaxUpgrade,
				IconPath:     itemData.IconPath,
			})
		}
	}

	// Common Items
	for _, item := range data.CommonItems {
		processItem(item)
	}

	// Key Items
	for _, item := range data.KeyItems {
		processItem(item)
	}

	return items
}
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

	u16 := utf16.Encode([]rune(vm.Name))
	for i := 0; i < 16; i++ {
		if i < len(u16) {
			data.CharacterName[i] = u16[i]
		} else {
			data.CharacterName[i] = 0
		}
	}

	// Update Inventory
	updateItems(vm.Inventory, &slot.Inventory)

	// Update Storage
	updateItems(vm.Storage, &slot.Storage)

	return nil
}

func updateItems(vmItems []ItemViewModel, data *core.EquipInventoryData) {
	// Create a map for quick lookup of VM items by handle
	vmMap := make(map[uint32]ItemViewModel)
	for _, item := range vmItems {
		vmMap[item.Handle] = item
	}

	// Update Common Items
	for i := range data.CommonItems {
		handle := data.CommonItems[i].GaItemHandle
		if handle == 0 || handle == 0xFFFFFFFF {
			continue
		}
		if vmItem, ok := vmMap[handle]; ok {
			// Apply quantity limits if necessary (though UI should handle this)
			qty := vmItem.Quantity
			if vmItem.MaxInventory > 0 && qty > vmItem.MaxInventory {
				qty = vmItem.MaxInventory
			}
			data.CommonItems[i].Quantity = qty
		}
	}

	// Update Key Items
	for i := range data.KeyItems {
		handle := data.KeyItems[i].GaItemHandle
		if handle == 0 || handle == 0xFFFFFFFF {
			continue
		}
		if vmItem, ok := vmMap[handle]; ok {
			qty := vmItem.Quantity
			if vmItem.MaxInventory > 0 && qty > vmItem.MaxInventory {
				qty = vmItem.MaxInventory
			}
			data.KeyItems[i].Quantity = qty
		}
	}
}

func MapSlotToVM(slotData []byte) (*CharacterViewModel, error) {
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
	updated := slot.Write("PC")
	copy(slotData, updated)
	return nil
}
