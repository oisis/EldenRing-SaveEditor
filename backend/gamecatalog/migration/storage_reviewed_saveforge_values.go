package migration

import (
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const waitGestureItemID uint32 = 0x40002337
const deathrootItemID uint32 = 0x4000082A
const lostAshesOfWarItemID uint32 = 0x40002756

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
const dlcKeysSubcategory = "DLC Keys"

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

var reviewedDepositableKeyItemStorageIDs = map[uint32]struct{}{
	0x40000852: {}, // Celestial Dew
	0x4000274C: {}, // Dragon Heart
	0x40001F40: {}, // Stonesword Key
}

var reviewedNonDepositableKeyItemStorageIDs = map[uint32]struct{}{
	0x40000073: {},
	0x40001F4A: {},
	0x40001FA9: {},
	0x40001FAA: {},
	0x40001FAB: {},
	0x40001FAD: {},
	0x40001FAF: {},
	0x40001FB9: {},
	0x40001FBE: {},
	0x40001FC0: {},
	0x40001FC1: {},
	0x40001FC6: {},
	0x40001FC8: {},
	0x40001FC9: {},
	0x40001FCE: {},
	0x40001FCF: {},
	0x40001FD0: {},
	0x40001FD2: {},
	0x40001FD3: {},
	0x40001FD4: {},
	0x40001FD5: {},
	0x40001FD6: {},
	0x40001FD7: {},
	0x40001FD8: {},
	0x40001FD9: {},
	0x40001FDB: {},
	0x40001FDF: {},
	0x40001FE4: {},
	0x40001FE6: {},
	0x40001FE8: {},
	0x40001FE9: {},
	0x40001FEB: {},
	0x40001FEC: {},
	0x40001FEE: {},
	0x40001FEF: {},
	0x40001FF0: {},
	0x40001FF6: {},
	0x40001FF7: {},
	0x40001FF8: {},
	0x40001FFB: {},
	0x40001FFD: {},
	0x40001FFE: {},
	0x40001FFF: {},
	0x40002000: {},
	0x40002002: {},
	0x40002003: {},
	0x40002005: {},
	0x40002006: {},
	0x40002007: {},
	0x4000201E: {},
	0x4000218E: {},
	0x400021D4: {},
	0x400022CE: {},
	0x400022CF: {},
	0x400022D0: {},
	0x400022D1: {},
	0x400022D3: {},
	0x400022D4: {},
	0x400022D5: {},
	0x400022D6: {},
	0x400022D7: {},
	0x400022D8: {},
	0x400022D9: {}, // Nomadic Merchant's Bell Bearing [1]
	0x400022DA: {}, // Nomadic Merchant's Bell Bearing [2]
	0x400022DB: {}, // Nomadic Merchant's Bell Bearing [3]
	0x400022DC: {}, // Nomadic Merchant's Bell Bearing [4]
	0x400022DD: {}, // Nomadic Merchant's Bell Bearing [5]
	0x400022DE: {}, // Isolated Merchant's Bell Bearing [1]
	0x400022DF: {}, // Isolated Merchant's Bell Bearing [2]
	0x400022E0: {}, // Nomadic Merchant's Bell Bearing [6]
	0x400022E1: {}, // Hermit Merchant's Bell Bearing [1]
	0x400022E2: {}, // Nomadic Merchant's Bell Bearing [7]
	0x400022E3: {}, // Nomadic Merchant's Bell Bearing [8]
	0x400022E4: {}, // Nomadic Merchant's Bell Bearing [9]
	0x400022E5: {}, // Nomadic Merchant's Bell Bearing [10]
	0x400022E6: {}, // Nomadic Merchant's Bell Bearing [11]
	0x400022E7: {}, // Isolated Merchant's Bell Bearing [3]
	0x400022E8: {}, // Hermit Merchant's Bell Bearing [2]
	0x400022E9: {}, // Abandoned Merchant's Bell Bearing
	0x400022EA: {}, // Hermit Merchant's Bell Bearing [3]
	0x400022EB: {}, // Imprisoned Merchant's Bell Bearing
	0x400022EC: {}, // Iji's Bell Bearing
	0x400022ED: {}, // Rogier's Bell Bearing
	0x400022EE: {}, // Blackguard's Bell Bearing
	0x400022EF: {}, // Corhyn's Bell Bearing
	0x400022F0: {}, // Gowry's Bell Bearing
	0x400022F1: {}, // Bone Peddler's Bell Bearing
	0x400022F2: {}, // Meat Peddler's Bell Bearing
	0x400022F3: {}, // Medicine Peddler's Bell Bearing
	0x400022F4: {}, // Gravity Stone Peddler's Bell Bearing
	0x400022F7: {}, // Smithing-Stone Miner's Bell Bearing [1]
	0x400022F8: {}, // Smithing-Stone Miner's Bell Bearing [2]
	0x400022F9: {}, // Smithing-Stone Miner's Bell Bearing [3]
	0x400022FA: {}, // Smithing-Stone Miner's Bell Bearing [4]
	0x400022FB: {}, // Somberstone Miner's Bell Bearing [1]
	0x400022FC: {}, // Somberstone Miner's Bell Bearing [2]
	0x400022FD: {}, // Somberstone Miner's Bell Bearing [3]
	0x400022FE: {}, // Somberstone Miner's Bell Bearing [4]
	0x400022FF: {}, // Somberstone Miner's Bell Bearing [5]
	0x40002300: {}, // Glovewort Picker's Bell Bearing [1]
	0x40002301: {}, // Glovewort Picker's Bell Bearing [2]
	0x40002302: {}, // Glovewort Picker's Bell Bearing [3]
	0x40002303: {}, // Ghost-Glovewort Picker's Bell Bearing [1]
	0x40002304: {}, // Ghost-Glovewort Picker's Bell Bearing [2]
	0x40002305: {}, // Ghost-Glovewort Picker's Bell Bearing [3]
	0x40002310: {}, // Unalloyed Gold Needle
	0x40002311: {}, // Valkyrie's Prosthesis
	0x40002313: {}, // Beast Eye
	0x40002314: {}, // Weathered Dagger
	0x40002458: {}, // Fugitive Warrior's Recipe [5]
	0x40002760: {}, // Great Rune of the Unborn
	0x401EA3C3: {}, // Igon's Furled Finger
	0x401EA3C8: {}, // Hole-Laden Necklace
	0x401EA3D3: {}, // Black Syrup
	0x401EA3D5: {}, // Messmer's Kindling
	0x401EA744: {}, // Moore's Bell Bearing
	0x401EA745: {}, // Ymir's Bell Bearing
	0x401EA746: {}, // Herbalist's Bell Bearing
	0x401EA747: {}, // Mushroom-Seller's Bell Bearing [1]
	0x401EA748: {}, // Mushroom-Seller's Bell Bearing [2]
	0x401EA749: {}, // Greasemonger's Bell Bearing
	0x401EA74A: {}, // Moldmonger's Bell Bearing
	0x401EA74B: {}, // Igon's Bell Bearing
	0x401EA74C: {}, // Spellmachinist's Bell Bearing
	0x401EA74D: {}, // String-Seller's Bell Bearing
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

	if _, ok := reviewedDepositableKeyItemStorageIDs[item.ID]; ok {
		storage.MaxStorageSFV = nil
	}
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
	if item.Category == "key_items" &&
		(item.Subcategory == dlcKeysSubcategory || item.ID == deathrootItemID || item.ID == lostAshesOfWarItemID) {
		storage.MaxStorageSFV = nil
	}
	if _, ok := reviewedNonDepositableKeyItemStorageIDs[item.ID]; ok {
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
