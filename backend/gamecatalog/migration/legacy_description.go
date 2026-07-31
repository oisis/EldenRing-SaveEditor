package migration

import "github.com/oisis/EldenRing-SaveForge/backend/db/data"

func copyLegacyDescription(value data.ItemDescription) *descriptionSeed {
	result := &descriptionSeed{
		Description: value.Description,
		Location:    value.Location,
		Weight:      value.Weight,
	}
	if value.Weapon != nil {
		result.Weapon = &legacyWeaponStats{
			Weight: value.Weapon.Weight, PhysDamage: value.Weapon.PhysDamage,
			MagDamage: value.Weapon.MagDamage, FireDamage: value.Weapon.FireDamage,
			LitDamage: value.Weapon.LitDamage, HolyDamage: value.Weapon.HolyDamage,
			ScaleStr: value.Weapon.ScaleStr, ScaleDex: value.Weapon.ScaleDex,
			ScaleInt: value.Weapon.ScaleInt, ScaleFai: value.Weapon.ScaleFai,
			ReqStr: value.Weapon.ReqStr, ReqDex: value.Weapon.ReqDex,
			ReqInt: value.Weapon.ReqInt, ReqFai: value.Weapon.ReqFai,
			ReqArc: value.Weapon.ReqArc,
		}
	}
	if value.Armor != nil {
		result.Armor = &legacyArmorStats{
			Weight: value.Armor.Weight, Physical: value.Armor.Physical,
			Strike: value.Armor.Strike, Slash: value.Armor.Slash,
			Pierce: value.Armor.Pierce, Magic: value.Armor.Magic,
			Fire: value.Armor.Fire, Lightning: value.Armor.Lightning,
			Holy: value.Armor.Holy, Immunity: value.Armor.Immunity,
			Robustness: value.Armor.Robustness, Focus: value.Armor.Focus,
			Vitality: value.Armor.Vitality, Poise: value.Armor.Poise,
		}
	}
	if value.Spell != nil {
		result.Spell = &legacySpellStats{
			FPCost: value.Spell.FPCost, Slots: value.Spell.Slots,
			ReqInt: value.Spell.ReqInt, ReqFai: value.Spell.ReqFai,
			ReqArc: value.Spell.ReqArc,
		}
	}
	return result
}
