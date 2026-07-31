package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func attachUpgradeSaveForgeValue(
	capability *schema.Capability[schema.UpgradeRules],
	item seed,
	family schema.ItemFamily,
) error {
	if family == schema.ItemFamilySpiritAsh {
		return nil
	}
	if !item.HasLegacyItem || capability == nil || capability.Rules == nil {
		return nil
	}
	if item.MaxUpgrade > 0xFF {
		return fmt.Errorf(
			"legacy ItemData.MaxUpgrade %d exceeds uint8",
			item.MaxUpgrade,
		)
	}
	candidates := []legacyCandidate[uint8]{{
		available: true,
		value:     uint8(item.MaxUpgrade),
		source:    "ItemData.MaxUpgrade",
	}}
	if item.WeaponStats != nil {
		if item.WeaponStats.MaxUpgrade < 0 ||
			item.WeaponStats.MaxUpgrade > 0xFF {
			return fmt.Errorf(
				"legacy WeaponStatsV1.MaxUpgrade %d exceeds uint8",
				item.WeaponStats.MaxUpgrade,
			)
		}
		candidates = append(candidates, legacyCandidate[uint8]{
			available: true,
			value:     uint8(item.WeaponStats.MaxUpgrade),
			source:    "WeaponStatsV1.MaxUpgrade",
		})
	}
	value, err := saveForgeConsensusValue(
		"maxLevel",
		capability.Rules.MaxLevel,
		candidates...,
	)
	if err != nil {
		return err
	}
	capability.Rules.MaxLevelSFV = value
	return nil
}
