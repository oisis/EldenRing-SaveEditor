package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

const waitGestureItemID uint32 = 0x40002337

func discardReviewedStorageSaveForgeValues(
	storage *schema.ItemStorage,
	item seed,
	family schema.ItemFamily,
) {
	if family == schema.ItemFamilyGesture && item.ID == waitGestureItemID {
		storage.MaxInventorySFV = nil
	}
}
