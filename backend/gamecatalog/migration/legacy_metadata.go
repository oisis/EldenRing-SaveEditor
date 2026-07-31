package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func emptyDescriptionRecord() schema.ItemDescriptionRecord {
	return schema.ItemDescriptionRecord{
		Description: unknownCatalogFact[string]("legacy Descriptions has no entry for this item"),
		Location:    unknownCatalogFact[string]("legacy Descriptions has no entry for this item"),
		Weight:      unknownCatalogFact[float64]("legacy Descriptions has no entry for this item"),
	}
}

func buildDescriptionRecord(value descriptionSeed) schema.ItemDescriptionRecord {
	record := schema.ItemDescriptionRecord{
		Description: optionalLegacyString(value.Description, "copied from legacy Descriptions.Description"),
		Location:    optionalLegacyString(value.Location, "copied from legacy Descriptions.Location"),
		Weight:      knownLegacyFact(value.Weight, "copied from legacy Descriptions.Weight"),
	}
	if value.Weapon != nil {
		stats := value.Weapon
		record.Weapon = &schema.CuratedWeaponStats{
			Weight:               knownLegacyFact(stats.Weight, "copied from legacy Descriptions.Weapon.Weight"),
			AttackPhysical:       knownLegacyFact(stats.PhysDamage, "copied from legacy Descriptions.Weapon.PhysDamage"),
			AttackMagic:          knownLegacyFact(stats.MagDamage, "copied from legacy Descriptions.Weapon.MagDamage"),
			AttackFire:           knownLegacyFact(stats.FireDamage, "copied from legacy Descriptions.Weapon.FireDamage"),
			AttackLightning:      knownLegacyFact(stats.LitDamage, "copied from legacy Descriptions.Weapon.LitDamage"),
			AttackHoly:           knownLegacyFact(stats.HolyDamage, "copied from legacy Descriptions.Weapon.HolyDamage"),
			ScalingStrength:      knownLegacyFact(stats.ScaleStr, "copied from legacy Descriptions.Weapon.ScaleStr"),
			ScalingDexterity:     knownLegacyFact(stats.ScaleDex, "copied from legacy Descriptions.Weapon.ScaleDex"),
			ScalingIntelligence:  knownLegacyFact(stats.ScaleInt, "copied from legacy Descriptions.Weapon.ScaleInt"),
			ScalingFaith:         knownLegacyFact(stats.ScaleFai, "copied from legacy Descriptions.Weapon.ScaleFai"),
			RequiredStrength:     knownLegacyFact(stats.ReqStr, "copied from legacy Descriptions.Weapon.ReqStr"),
			RequiredDexterity:    knownLegacyFact(stats.ReqDex, "copied from legacy Descriptions.Weapon.ReqDex"),
			RequiredIntelligence: knownLegacyFact(stats.ReqInt, "copied from legacy Descriptions.Weapon.ReqInt"),
			RequiredFaith:        knownLegacyFact(stats.ReqFai, "copied from legacy Descriptions.Weapon.ReqFai"),
			RequiredArcane:       knownLegacyFact(stats.ReqArc, "copied from legacy Descriptions.Weapon.ReqArc"),
		}
	}
	if value.Armor != nil {
		stats := value.Armor
		record.Armor = &schema.CuratedArmorStats{
			Weight: knownLegacyFact(stats.Weight, "copied from legacy Descriptions.Armor.Weight"), Physical: knownLegacyFact(stats.Physical, "copied from legacy Descriptions.Armor.Physical"),
			Strike: knownLegacyFact(stats.Strike, "copied from legacy Descriptions.Armor.Strike"), Slash: knownLegacyFact(stats.Slash, "copied from legacy Descriptions.Armor.Slash"),
			Pierce: knownLegacyFact(stats.Pierce, "copied from legacy Descriptions.Armor.Pierce"), Magic: knownLegacyFact(stats.Magic, "copied from legacy Descriptions.Armor.Magic"),
			Fire: knownLegacyFact(stats.Fire, "copied from legacy Descriptions.Armor.Fire"), Lightning: knownLegacyFact(stats.Lightning, "copied from legacy Descriptions.Armor.Lightning"),
			Holy: knownLegacyFact(stats.Holy, "copied from legacy Descriptions.Armor.Holy"), Immunity: knownLegacyFact(stats.Immunity, "copied from legacy Descriptions.Armor.Immunity"),
			Robustness: knownLegacyFact(stats.Robustness, "copied from legacy Descriptions.Armor.Robustness"), Focus: knownLegacyFact(stats.Focus, "copied from legacy Descriptions.Armor.Focus"),
			Vitality: knownLegacyFact(stats.Vitality, "copied from legacy Descriptions.Armor.Vitality"), Poise: knownLegacyFact(stats.Poise, "copied from legacy Descriptions.Armor.Poise"),
		}
	}
	if value.Spell != nil {
		stats := value.Spell
		record.Spell = &schema.CuratedSpellStats{
			FPCost: knownLegacyFact(stats.FPCost, "copied from legacy Descriptions.Spell.FPCost"), MemorySlots: knownLegacyFact(stats.Slots, "copied from legacy Descriptions.Spell.Slots"),
			RequiredIntelligence: knownLegacyFact(stats.ReqInt, "copied from legacy Descriptions.Spell.ReqInt"), RequiredFaith: knownLegacyFact(stats.ReqFai, "copied from legacy Descriptions.Spell.ReqFai"),
			RequiredArcane: knownLegacyFact(stats.ReqArc, "copied from legacy Descriptions.Spell.ReqArc"),
		}
	}
	return record
}
