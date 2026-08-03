package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

const waitGestureItemID uint32 = 0x40002337

const bolsteringMaterialsCategory = "bolstering_materials"

func promoteBolsteringMaterialSafeModeLimits(
	storage *schema.ItemStorage,
	item seed,
) {
	if item.Category != bolsteringMaterialsCategory {
		return
	}
	storage.SafeModeMaxInventory = storage.MaxInventorySFV
	storage.SafeModeMaxStorage = storage.MaxStorageSFV
	if storage.SafeModeMaxInventory != nil {
		storage.SafeModeMaxInventory.Provenance.Method =
			"preserved legacy Safe Mode maximum inventory for a single NG0 playthrough"
	}
	if storage.SafeModeMaxStorage != nil {
		storage.SafeModeMaxStorage.Provenance.Method =
			"preserved legacy Safe Mode maximum storage for a single NG0 playthrough"
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
