package migration

import (
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const waitGestureItemID uint32 = 0x40002337

const bolsteringMaterialsCategory = "bolstering_materials"
const incantationsCategory = "incantations"
const sorceriesCategory = "sorceries"
const ashesCategory = "ashes"
const toolsCategory = "tools"
const gesturesCategory = "gestures"
const infoCategory = "info"

func promoteSafeModeStorageLimits(
	storage *schema.ItemStorage,
	item seed,
) {
	isPrattlingPate := item.Category == toolsCategory && strings.HasPrefix(item.Name, "Prattling Pate")
	isRemembrance := item.Category == toolsCategory &&
		(item.Name == "Elden Remembrance" || strings.HasPrefix(item.Name, "Remembrance"))
	isTelescope := item.Category == toolsCategory && item.Name == "Telescope"
	if item.Category != bolsteringMaterialsCategory &&
		item.Category != incantationsCategory &&
		item.Category != sorceriesCategory &&
		item.Category != ashesCategory &&
		!isPrattlingPate &&
		!isRemembrance &&
		!isTelescope {
		return
	}
	storage.SafeModeMaxInventory = storage.MaxInventorySFV
	storage.SafeModeMaxStorage = storage.MaxStorageSFV
	inventoryMethod := "preserved legacy Safe Mode maximum inventory for a single NG0 playthrough"
	storageMethod := "preserved legacy Safe Mode maximum storage for a single NG0 playthrough"
	if item.Category == incantationsCategory || item.Category == sorceriesCategory {
		inventoryMethod = "preserved legacy Safe Mode maximum spell inventory"
		storageMethod = "preserved legacy Safe Mode maximum spell storage"
	}
	if isRemembrance {
		storage.SafeModeMaxInventory.Value = 2
		inventoryMethod = "defined Safe Mode maximum Remembrance inventory for a single NG0 playthrough"
	}
	if isTelescope {
		storage.SafeModeMaxStorage.Value = 0
		storageMethod = "defined Safe Mode maximum Telescope storage"
	}
	if storage.SafeModeMaxInventory != nil {
		storage.SafeModeMaxInventory.Provenance.Method = inventoryMethod
	}
	if storage.SafeModeMaxStorage != nil {
		storage.SafeModeMaxStorage.Provenance.Method = storageMethod
	}
	storage.MaxInventorySFV = nil
	storage.MaxStorageSFV = nil
}

func discardReviewedStorageSaveForgeValues(
	storage *schema.ItemStorage,
	item seed,
	family schema.ItemFamily,
) {
	if item.Category == toolsCategory {
		switch item.Name {
		case "?GoodsName? Holy Water Pot",
			"Deathsbane Jerky",
			"Deathsbane White Jerky",
			"Drawstring Freezing Grease",
			"Holy Water Grease",
			"Roped Freezing Pot":
			storage.MaxInventorySFV = nil
			storage.MaxStorageSFV = nil
		}
		switch item.ID {
		case 0x4000012C, // Fire Pot
			0x40000140, // Lightning Pot
			0x4000014A, // Fetid Pot
			0x40000154, // Swarm Pot
			0x4000015E, // Holy Water Pot
			0x40000172, // Poison Pot
			0x4000017C, // Oil Pot
			0x40000190, // Roped Fire Pot
			0x400001A4, // Roped Lightning Pot
			0x400001AE, // Roped Fetid Pot
			0x400001B8, // Roped Poison Pot
			0x400001C2, // Roped Oil Pot
			0x400001CC, // Roped Magic Pot
			0x400001D6, // Roped Fly Pot
			0x400001EA, // Roped Volcano Pot
			0x400001FE, // Roped Holy Water Pot
			0x40000258, // Volcano Pot
			0x40000280, // Sleep Pot
			0x4000028A, // Rancor Pot
			0x40000294, // Magic Pot
			0x401E873C, // Red Lightning Pot
			0x401E8746, // Frenzied Flame Pot
			0x401E8778: // Roped Frenzied Flame Pot
			storage.MaxInventorySFV = nil
		}
	}
	if item.Category == toolsCategory &&
		(strings.HasPrefix(item.Name, "Flask of Crimson Tears") ||
			strings.HasPrefix(item.Name, "Flask of Cerulean Tears")) {
		storage.MaxInventorySFV = nil
		storage.MaxStorageSFV = nil
	}
	if item.Category == toolsCategory && item.Name == "Flask of Wondrous Physick" {
		storage.MaxStorageSFV = nil
	}
	if item.Category == toolsCategory && item.Name == "Festering Bloody Finger" {
		storage.MaxInventorySFV = nil
	}
	if item.Category == infoCategory {
		storage.MaxStorageSFV = nil
	}
	if family != schema.ItemFamilyGesture {
		return
	}
	storage.MaxStorageSFV = nil
	if item.ID == waitGestureItemID {
		storage.MaxInventorySFV = nil
	}
}
