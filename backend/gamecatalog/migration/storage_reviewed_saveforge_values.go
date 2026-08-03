package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

const waitGestureItemID uint32 = 0x40002337

const bolsteringMaterialsCategory = "bolstering_materials"
const incantationsCategory = "incantations"
const sorceriesCategory = "sorceries"

func promoteSafeModeStorageLimits(
	storage *schema.ItemStorage,
	item seed,
) {
	if item.Category != bolsteringMaterialsCategory &&
		item.Category != incantationsCategory &&
		item.Category != sorceriesCategory {
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
	if family == schema.ItemFamilyGesture && item.ID == waitGestureItemID {
		storage.MaxInventorySFV = nil
	}
}
