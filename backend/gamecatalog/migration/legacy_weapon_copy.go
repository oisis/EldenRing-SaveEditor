package migration

import "github.com/oisis/EldenRing-SaveForge/backend/db/data"

func copyWeaponStats(value data.WeaponStatsV1) *weaponStatsSeed {
	effects := make([]passiveEffectSeed, len(value.PassiveEffects))
	for index, effect := range value.PassiveEffects {
		effects[index] = passiveEffectSeed{
			Kind: effect.Kind, Source: effect.Source, SpEffectID: effect.SpEffectID,
			Label: effect.Label, Value: effect.Value, Known: effect.Known,
		}
	}
	return &weaponStatsSeed{
		ItemID: value.ItemID, WepType: value.WepType, SortGroupID: value.SortGroupID,
		ReinforceTypeID: value.ReinforceTypeID, GemMountType: value.GemMountType,
		Weight: value.Weight, AttackPhysical: value.AttackPhysical,
		AttackMagic: value.AttackMagic, AttackFire: value.AttackFire,
		AttackLightning: value.AttackLightning, AttackHoly: value.AttackHoly,
		AttackStamina: value.AttackStamina, GuardPhysical: value.GuardPhysical,
		GuardMagic: value.GuardMagic, GuardFire: value.GuardFire,
		GuardLightning: value.GuardLightning, GuardHoly: value.GuardHoly,
		GuardBoost: value.GuardBoost, StatReqStr: value.StatReqStr,
		StatReqDex: value.StatReqDex, StatReqInt: value.StatReqInt,
		StatReqFai: value.StatReqFai, StatReqArc: value.StatReqArc,
		Critical: value.Critical, ScalingStrRaw: value.ScalingStrRaw,
		ScalingDexRaw: value.ScalingDexRaw, ScalingIntRaw: value.ScalingIntRaw,
		ScalingFaiRaw: value.ScalingFaiRaw, ScalingArcRaw: value.ScalingArcRaw,
		PassiveEffects: effects, DefaultAoWID: value.DefaultAoWID,
		IsInfusable: value.IsInfusable, IsSomber: value.IsSomber,
		MaxUpgrade: value.MaxUpgrade, SourceRowID: value.SourceRowID,
		Warnings: cloneStrings(value.Warnings),
	}
}
