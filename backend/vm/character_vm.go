package vm

import (
	"encoding/binary"
	"github.com/oisis/EldenRing-SaveEditor/backend/core"
	"github.com/oisis/EldenRing-SaveEditor/backend/db"
	"unicode/utf16"
)

type ItemViewModel struct {
	Handle         uint32   `json:"handle"`
	ID             uint32   `json:"id"`
	Name           string   `json:"name"`
	Category       string   `json:"category"`
	SubCategory    string   `json:"subCategory"`
	Quantity       uint32   `json:"quantity"`
	MaxInventory   uint32   `json:"maxInventory"`
	MaxStorage     uint32   `json:"maxStorage"`
	MaxUpgrade     uint32   `json:"maxUpgrade"`
	CurrentUpgrade uint32   `json:"currentUpgrade"`
	IconPath       string   `json:"iconPath"`
	Flags          []string `json:"flags"`
}

type CharacterViewModel struct {
	Name                string          `json:"name"`
	Level               uint32          `json:"level"`
	Souls               uint32          `json:"souls"`
	Vigor               uint32          `json:"vigor"`
	Mind                uint32          `json:"mind"`
	Endurance           uint32          `json:"endurance"`
	Strength            uint32          `json:"strength"`
	Dexterity           uint32          `json:"dexterity"`
	Intelligence        uint32          `json:"intelligence"`
	Faith               uint32          `json:"faith"`
	Arcane              uint32          `json:"arcane"`
	ScadutreeBlessing   uint8           `json:"scadutreeBlessing"`
	ShadowRealmBlessing uint8           `json:"shadowRealmBlessing"`
	Inventory           []ItemViewModel `json:"inventory"`
	Storage             []ItemViewModel `json:"storage"`
}

func MapParsedSlotToVM(slot *core.SaveSlot) (*CharacterViewModel, error) {
	data := slot.Player
	vm := &CharacterViewModel{
		Level:               data.Level,
		Souls:               data.Souls,
		Vigor:               data.Vigor,
		Mind:                data.Mind,
		Endurance:           data.Endurance,
		Strength:            data.Strength,
		Dexterity:           data.Dexterity,
		Intelligence:        data.Intelligence,
		Faith:               data.Faith,
		Arcane:              data.Arcane,
		ScadutreeBlessing:   data.ScadutreeBlessing,
		ShadowRealmBlessing: data.ShadowRealmBlessing,
		Inventory:           []ItemViewModel{},
		Storage:             []ItemViewModel{},
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

			itemData, baseID := db.GetItemDataFuzzy(itemID)
			name := itemData.Name

			// Strict filtering: skip items that are not in our database (Unknown)
			// to avoid garbage data from misaligned offsets.
			if name == "" {
				return
			}

			var currentUpgrade uint32
			if baseID != itemID && itemID > baseID {
				currentUpgrade = itemID - baseID
			}

			displayQuantity := item.Quantity
			// For non-stackable items, force quantity to 1.
			// Exception: arrows/bolts have weapon-like handles (0x82...) but are stackable.
			isArrow := itemData.Category == "arrows_and_bolts"
			if (category == "Weapon" || category == "Armor" || category == "Talisman" || category == "Ash of War") && !isArrow {
				displayQuantity = 1
			} else {
				// For stackable items, mask the high bit which is often used by the engine
				displayQuantity = item.Quantity & 0x7FFFFFFF
			}

			items = append(items, ItemViewModel{
				Handle:         item.GaItemHandle,
				ID:             itemID,
				Name:           name,
				Category:       category,
				SubCategory:    db.GetItemSubCategory(itemID, itemData, category),
				Quantity:       displayQuantity,
				MaxInventory:   itemData.MaxInventory,
				MaxStorage:     itemData.MaxStorage,
				MaxUpgrade:     itemData.MaxUpgrade,
				CurrentUpgrade: currentUpgrade,
				IconPath:       itemData.IconPath,
				Flags:          itemData.Flags,
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
	data.ScadutreeBlessing = vm.ScadutreeBlessing
	data.ShadowRealmBlessing = vm.ShadowRealmBlessing

	u16 := utf16.Encode([]rune(vm.Name))
	for i := 0; i < 16; i++ {
		if i < len(u16) {
			data.CharacterName[i] = u16[i]
		} else {
			data.CharacterName[i] = 0
		}
	}

	// Update Inventory (with write-back to slot.Data)
	updateItemsAndSync(vm.Inventory, &slot.Inventory, slot, false)

	// Update Storage (with write-back to slot.Data)
	updateItemsAndSync(vm.Storage, &slot.Storage, slot, true)

	return nil
}

func updateItemsAndSync(vmItems []ItemViewModel, data *core.EquipInventoryData, slot *core.SaveSlot, isStorage bool) {
	vmMap := make(map[uint32]ItemViewModel)
	for _, item := range vmItems {
		vmMap[item.Handle] = item
	}

	var commonStart int
	if isStorage {
		commonStart = slot.StorageBoxOffset + 4
	} else {
		commonStart = slot.MagicOffset + 505
	}

	for i := range data.CommonItems {
		handle := data.CommonItems[i].GaItemHandle
		if handle == 0 || handle == 0xFFFFFFFF {
			continue
		}
		if vmItem, ok := vmMap[handle]; ok {
			qty := vmItem.Quantity
			if isStorage {
				if vmItem.MaxStorage > 0 && qty > vmItem.MaxStorage {
					qty = vmItem.MaxStorage
				}
			} else {
				if vmItem.MaxInventory > 0 && qty > vmItem.MaxInventory {
					qty = vmItem.MaxInventory
				}
			}
			data.CommonItems[i].Quantity = qty
			off := commonStart + i*12 + 4
			if off+4 <= len(slot.Data) {
				binary.LittleEndian.PutUint32(slot.Data[off:], qty)
			}
		}
	}

	// Key Items are only in main inventory (not storage), stored after common items
	if !isStorage {
		keyStart := commonStart + 0xa80*12
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
				off := keyStart + i*12 + 4
				if off+4 <= len(slot.Data) {
					binary.LittleEndian.PutUint32(slot.Data[off:], qty)
				}
			}
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
