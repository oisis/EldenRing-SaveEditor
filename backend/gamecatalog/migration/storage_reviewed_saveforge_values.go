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
const cookbooksSubcategory = "Cookbooks"
const crystalTearsSubcategory = "Crystal Tears"
const worldMapsSubcategory = "World Maps"
const spellbooksSubcategory = "Sorcery Scrolls + Incantation Scrolls"

var safeModeNG0InventoryByKeyItemID = map[uint32]uint32{
	0x40000852: 10, // Celestial Dew
	0x4000082A: 9,  // Deathroot
	0x4000274C: 27, // Dragon Heart
	0x401EA3CB: 1,  // Heart of Bayle
	0x40001FFA: 4,  // Imbued Sword Key
	0x40001FF9: 18, // Larval Tear
	0x401EA3E1: 9,  // Larval Tear (DLC)
	0x40002756: 20, // Lost Ashes of War
	0x40000087: 3,  // Phantom Great Rune
	0x40002001: 6,  // Seedbed Curse
	0x400004D8: 3,  // Shabriri Grape
	0x40001F40: 81, // Stonesword Key
}

var nonDepositableKeyItemIDs = map[uint32]struct{}{
	0x40001FFA: {}, // Imbued Sword Key
	0x40001FF9: {}, // Larval Tear
	0x401EA3E1: {}, // Larval Tear (DLC)
	0x40000087: {}, // Phantom Great Rune
	0x40002001: {}, // Seedbed Curse
	0x400004D8: {}, // Shabriri Grape
}

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

func promoteKeyItemNG0InventoryLimits(
	storage *schema.ItemStorage,
	item seed,
) {
	if item.Category != "key_items" {
		return
	}
	limit, ok := safeModeNG0InventoryByKeyItemID[item.ID]
	if !ok {
		return
	}

	storage.SafeModeMaxInventory = storage.MaxInventorySFV
	if storage.SafeModeMaxInventory != nil {
		storage.SafeModeMaxInventory.Value = limit
		storage.SafeModeMaxInventory.Provenance.Method =
			"defined Safe Mode maximum inventory for a single full NG0 playthrough"
	}
	storage.MaxInventorySFV = nil

	if _, ok := nonDepositableKeyItemIDs[item.ID]; ok {
		storage.MaxStorageSFV = nil
	}
}

func discardReviewedStorageSaveForgeValues(
	storage *schema.ItemStorage,
	item seed,
	family schema.ItemFamily,
) {
	if item.Category == toolsCategory {
		storage.MaxInventorySFV = nil
		storage.MaxStorageSFV = nil
	}
	if item.Category == infoCategory {
		storage.MaxStorageSFV = nil
	}
	if item.Category == "key_items" && item.Subcategory == cookbooksSubcategory {
		storage.MaxInventorySFV = nil
		storage.MaxStorageSFV = nil
	}
	if item.Category == "key_items" && item.Subcategory == crystalTearsSubcategory {
		storage.MaxInventorySFV = nil
		storage.MaxStorageSFV = nil
	}
	if item.Category == "key_items" && item.Subcategory == worldMapsSubcategory {
		storage.MaxInventorySFV = nil
		storage.MaxStorageSFV = nil
	}
	if item.Category == "key_items" && item.Subcategory == spellbooksSubcategory {
		storage.MaxInventorySFV = nil
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
