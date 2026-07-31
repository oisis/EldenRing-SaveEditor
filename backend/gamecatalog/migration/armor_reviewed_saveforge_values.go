package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

const perfumerRobeAlteredItemID uint32 = 0x100163DC

func discardReviewedArmorSaveForgeValues(data *schema.ArmorData, item seed) {
	if item.ID == perfumerRobeAlteredItemID {
		data.FocusSFV = nil
	}
}
