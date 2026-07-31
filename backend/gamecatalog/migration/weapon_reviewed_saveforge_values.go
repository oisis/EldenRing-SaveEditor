package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

const (
	giantsRedBraidItemID          uint32 = 0x01321760
	magmaWhipCandlestickItemID    uint32 = 0x0131A230
	meteoriteStaffItemID          uint32 = 0x01FB5AD0
	velvetSwordOfSaintTrinaItemID uint32 = 0x00264CB0
)

func discardReviewedWeaponSaveForgeValues(data *schema.WeaponData, item seed) {
	data.SwordArtsNameSFV = nil
	data.AttackPhysicalSFV = nil

	if item.Category == "arrows_and_bolts" {
		data.WeightSFV = nil
		data.AttackMagicSFV = nil
		data.AttackFireSFV = nil
		data.AttackLightningSFV = nil
		data.AttackHolySFV = nil
	}

	switch item.ID {
	case giantsRedBraidItemID, magmaWhipCandlestickItemID:
		data.AttackFireSFV = nil
	case meteoriteStaffItemID:
		data.MaxUpgradeSFV = nil
	case velvetSwordOfSaintTrinaItemID:
		data.WeightSFV = nil
		data.AttackMagicSFV = nil
		data.RequiredStrengthSFV = nil
		data.RequiredDexteritySFV = nil
		data.RequiredIntelligenceSFV = nil
	}
}
