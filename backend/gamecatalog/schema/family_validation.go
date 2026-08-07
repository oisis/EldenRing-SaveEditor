package schema

import "fmt"

func validateFamilyDocument(item ItemDocument, sources map[SourceID]struct{}) error {
	payloadCount := 0
	for _, present := range []bool{
		item.Weapon != nil,
		item.Armor != nil,
		item.Talisman != nil,
		item.AshOfWar != nil,
		item.Spell != nil,
		item.SpiritAsh != nil,
		item.Goods != nil,
		item.Gesture != nil,
	} {
		if present {
			payloadCount++
		}
	}
	if payloadCount != 1 {
		return fmt.Errorf("item must have exactly one family data section, got %d", payloadCount)
	}

	extended := !isOmittedFact(item.Category)
	switch item.Family.Value {
	case ItemFamilyWeapon:
		if item.Weapon == nil {
			return fmt.Errorf("weapon item must have weapon data")
		}
		return validateWeaponData(*item.Weapon, sources, extended)
	case ItemFamilyArmor:
		if item.Armor == nil {
			return fmt.Errorf("armor item must have armor data")
		}
		return validateArmorData(*item.Armor, sources)
	case ItemFamilyTalisman:
		if item.Talisman == nil {
			return fmt.Errorf("talisman item must have talisman data")
		}
		return validateTalismanData(*item.Talisman, sources)
	case ItemFamilyAshOfWar:
		if item.AshOfWar == nil {
			return fmt.Errorf("Ash of War item must have Ash of War data")
		}
		return validateAshOfWarData(*item.AshOfWar, sources, extended)
	case ItemFamilySpell:
		if item.Spell == nil {
			return fmt.Errorf("spell item must have spell data")
		}
		return validateSpellData(*item.Spell, sources)
	case ItemFamilySpiritAsh:
		if item.SpiritAsh == nil {
			return fmt.Errorf("spirit ash item must have spirit ash data")
		}
		return validateSpiritAshData(*item.SpiritAsh, sources)
	case ItemFamilyGoods:
		if item.Goods == nil {
			return fmt.Errorf("goods item must have goods data")
		}
		return validateGoodsData(*item.Goods, sources)
	case ItemFamilyGesture:
		if item.Gesture == nil {
			return fmt.Errorf("gesture item must have gesture data")
		}
		return validateGestureData(*item.Gesture, sources)
	default:
		return fmt.Errorf("unsupported item family %q", item.Family.Value)
	}
}

func validateWeaponData(data WeaponData, sources map[SourceID]struct{}, extended bool) error {
	required := []struct {
		name string
		fact anyFactValidator
	}{
		{"item.weapon.sourceRowID", factValidator(data.SourceRowID)},
		{"item.weapon.weaponTypeID", factValidator(data.WeaponTypeID)},
		{"item.weapon.weight", factValidator(data.Weight)},
		{"item.weapon.attackPhysical", factValidator(data.AttackPhysical)},
		{"item.weapon.requiredStrength", factValidator(data.RequiredStrength)},
		{"item.weapon.requiredDexterity", factValidator(data.RequiredDexterity)},
		{"item.weapon.critical", factValidator(data.Critical)},
	}
	if err := validateFactValidators(required, sources); err != nil {
		return err
	}
	if data.SourceRowID.Known && data.SourceRowID.Value == 0 {
		return fmt.Errorf("item.weapon.sourceRowID must be greater than zero when known")
	}

	optional := []struct {
		name string
		fact anyOptionalFactValidator
	}{
		{"item.weapon.iconID", optionalFactValidator(data.IconID)},
		{"item.weapon.sortID", optionalFactValidator(data.SortID)},
		{"item.weapon.sortGroupID", optionalFactValidator(data.SortGroupID)},
		{"item.weapon.reinforceTypeID", optionalFactValidator(data.ReinforceTypeID)},
		{"item.weapon.gemMountType", optionalFactValidator(data.GemMountType)},
		{"item.weapon.attackMagic", optionalFactValidator(data.AttackMagic)},
		{"item.weapon.attackFire", optionalFactValidator(data.AttackFire)},
		{"item.weapon.attackLightning", optionalFactValidator(data.AttackLightning)},
		{"item.weapon.attackHoly", optionalFactValidator(data.AttackHoly)},
		{"item.weapon.attackStamina", optionalFactValidator(data.AttackStamina)},
		{"item.weapon.guardPhysical", optionalFactValidator(data.GuardPhysical)},
		{"item.weapon.guardMagic", optionalFactValidator(data.GuardMagic)},
		{"item.weapon.guardFire", optionalFactValidator(data.GuardFire)},
		{"item.weapon.guardLightning", optionalFactValidator(data.GuardLightning)},
		{"item.weapon.guardHoly", optionalFactValidator(data.GuardHoly)},
		{"item.weapon.guardBoost", optionalFactValidator(data.GuardBoost)},
		{"item.weapon.requiredIntelligence", optionalFactValidator(data.RequiredIntelligence)},
		{"item.weapon.requiredFaith", optionalFactValidator(data.RequiredFaith)},
		{"item.weapon.requiredArcane", optionalFactValidator(data.RequiredArcane)},
		{"item.weapon.scalingStrengthRaw", optionalFactValidator(data.ScalingStrengthRaw)},
		{"item.weapon.scalingDexterityRaw", optionalFactValidator(data.ScalingDexterityRaw)},
		{"item.weapon.scalingIntelligenceRaw", optionalFactValidator(data.ScalingIntelligenceRaw)},
		{"item.weapon.scalingFaithRaw", optionalFactValidator(data.ScalingFaithRaw)},
		{"item.weapon.scalingArcaneRaw", optionalFactValidator(data.ScalingArcaneRaw)},
		{"item.weapon.defaultAshOfWarID", optionalFactValidator(data.DefaultAshOfWarID)},
		{"item.weapon.swordArtsName", optionalFactValidator(data.SwordArtsName)},
		{"item.weapon.isInfusable", optionalFactValidator(data.IsInfusable)},
		{"item.weapon.isSomber", optionalFactValidator(data.IsSomber)},
		{"item.weapon.maxUpgrade", optionalFactValidator(data.MaxUpgrade)},
		{"item.weapon.warnings", optionalFactValidator(data.Warnings)},
		{"item.weapon.rightHandEquipable", optionalFactValidator(data.RightHandEquipable)},
		{"item.weapon.leftHandEquipable", optionalFactValidator(data.LeftHandEquipable)},
		{"item.weapon.bothHandEquipable", optionalFactValidator(data.BothHandEquipable)},
		{"item.weapon.arrowSlotEquipable", optionalFactValidator(data.ArrowSlotEquipable)},
		{"item.weapon.boltSlotEquipable", optionalFactValidator(data.BoltSlotEquipable)},
	}
	if err := validateOptionalFactValidators(optional, sources, extended); err != nil {
		return err
	}
	for index, effect := range data.PassiveEffects {
		if err := validateWeaponPassiveEffect(index, effect, sources); err != nil {
			return err
		}
	}
	return nil
}

func validateWeaponPassiveEffect(index int, effect WeaponPassiveEffectData, sources map[SourceID]struct{}) error {
	name := fmt.Sprintf("item.weapon.passiveEffects[%d]", index)
	required := []struct {
		name string
		fact anyFactValidator
	}{
		{name + ".kind", factValidator(effect.Kind)},
		{name + ".source", factValidator(effect.Source)},
		{name + ".spEffectID", factValidator(effect.SpEffectID)},
		{name + ".label", factValidator(effect.Label)},
		{name + ".value", factValidator(effect.Value)},
		{name + ".known", factValidator(effect.Known)},
	}
	if err := validateFactValidators(required, sources); err != nil {
		return err
	}
	if !effect.Kind.Known || (effect.Kind.Value != "on_hit" && effect.Kind.Value != "resident") {
		return fmt.Errorf("%s.kind must be known and supported", name)
	}
	if !effect.Source.Known || effect.Source.Value == "" {
		return fmt.Errorf("%s.source must be known and non-empty", name)
	}
	return nil
}

func validateAshOfWarData(data AshOfWarData, sources map[SourceID]struct{}, extended bool) error {
	if err := validateFact("item.ashOfWar.sourceRowID", data.SourceRowID, sources); err != nil {
		return err
	}
	if data.SourceRowID.Known && data.SourceRowID.Value == 0 {
		return fmt.Errorf("item.ashOfWar.sourceRowID must be greater than zero when known")
	}
	if err := validateFact("item.ashOfWar.compatibilityMask", data.CompatibilityMask, sources); err != nil {
		return err
	}
	optional := []struct {
		name string
		fact anyOptionalFactValidator
	}{
		{"item.ashOfWar.iconID", optionalFactValidator(data.IconID)},
		{"item.ashOfWar.sortID", optionalFactValidator(data.SortID)},
		{"item.ashOfWar.sortGroupID", optionalFactValidator(data.SortGroupID)},
		{"item.ashOfWar.swordArtsParamID", optionalFactValidator(data.SwordArtsParamID)},
		{"item.ashOfWar.swordArtsName", optionalFactValidator(data.SwordArtsName)},
		{"item.ashOfWar.defaultAffinity", optionalFactValidator(data.DefaultAffinity)},
	}
	if err := validateOptionalFactValidators(optional, sources, extended); err != nil {
		return err
	}
	if extended {
		if err := validateFact(
			"item.ashOfWar.compatibleClassNames",
			data.CompatibleClassNames,
			sources,
		); err != nil {
			return err
		}
		if !data.CompatibleClassNames.Known {
			return fmt.Errorf("item.ashOfWar.compatibleClassNames must be known")
		}
	}
	return validateOptionalStringList(
		"item.ashOfWar.compatibleClassNames",
		data.CompatibleClassNames,
		sources,
	)
}

type anyFactValidator func(string, map[SourceID]struct{}) error
type anyOptionalFactValidator func(string, map[SourceID]struct{}, bool) error

func factValidator[T any](fact Fact[T]) anyFactValidator {
	return func(name string, sources map[SourceID]struct{}) error {
		return validateFact(name, fact, sources)
	}
}

func optionalFactValidator[T any](fact Fact[T]) anyOptionalFactValidator {
	return func(name string, sources map[SourceID]struct{}, required bool) error {
		if required {
			return validateFact(name, fact, sources)
		}
		return validateOptionalFact(name, fact, sources)
	}
}

func validateFactValidators(
	validators []struct {
		name string
		fact anyFactValidator
	},
	sources map[SourceID]struct{},
) error {
	for _, validator := range validators {
		if err := validator.fact(validator.name, sources); err != nil {
			return err
		}
	}
	return nil
}

func validateOptionalFactValidators(
	validators []struct {
		name string
		fact anyOptionalFactValidator
	},
	sources map[SourceID]struct{},
	required bool,
) error {
	for _, validator := range validators {
		if err := validator.fact(validator.name, sources, required); err != nil {
			return err
		}
	}
	return nil
}
