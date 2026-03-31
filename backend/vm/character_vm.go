package vm

import (
	"unicode/utf16"
	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"github.com/oisis/EldenRing-SaveEditor/backend/db"
)

type ItemViewModel struct {
	Handle   uint32 `json:"handle"`
	ID       uint32 `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Quantity uint32 `json:"quantity"`
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

		var itemID uint32
		var ok bool

		typeBits := item.GaItemHandle & 0xF0000000

		// For Accessories and Items, the handle often contains the ID information.
		// We normalize it to the standard item prefixes (0x2 for Talisman, 0x4 for Item).
		if typeBits == core.ItemTypeAccessory {
			itemID = (item.GaItemHandle & 0x0FFFFFFF) | 0x20000000
			ok = true
		} else if typeBits == core.ItemTypeItem {
			itemID = (item.GaItemHandle & 0x0FFFFFFF) | 0x40000000
			ok = true
		} else {
			// For Weapons and Armor, we MUST use the GaMap to find the real ItemID.
			itemID, ok = gaMap[item.GaItemHandle]
		}

		if ok {
			if itemID == 0 || itemID == 110000 {
				return
			} // Filter Unarmed and Empty
			items = append(items, ItemViewModel{
				Handle:   item.GaItemHandle,
				ID:       itemID,
				Name:     db.GetItemName(itemID),
				Category: db.GetItemCategory(itemID),
				Quantity: item.Quantity,
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
	return nil
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
