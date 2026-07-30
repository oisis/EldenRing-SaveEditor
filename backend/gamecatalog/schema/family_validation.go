package schema

import "fmt"

func validateFamilyDocument(item ItemDocument, sources map[SourceID]struct{}) error {
	switch item.Family.Value {
	case ItemFamilyWeapon:
		if item.Weapon == nil || item.AshOfWar != nil {
			return fmt.Errorf("weapon item must have only weapon data")
		}
		return validateWeaponData(*item.Weapon, sources)
	case ItemFamilyAshOfWar:
		if item.AshOfWar == nil || item.Weapon != nil {
			return fmt.Errorf("Ash of War item must have only Ash of War data")
		}
		return validateAshOfWarData(*item.AshOfWar, sources)
	default:
		return fmt.Errorf("unsupported item family %q", item.Family.Value)
	}
}

func validateWeaponData(data WeaponData, sources map[SourceID]struct{}) error {
	if err := validateFact("item.weapon.sourceRowID", data.SourceRowID, sources); err != nil {
		return err
	}
	if data.SourceRowID.Known && data.SourceRowID.Value == 0 {
		return fmt.Errorf("item.weapon.sourceRowID must be greater than zero when known")
	}
	if err := validateFact("item.weapon.weaponTypeID", data.WeaponTypeID, sources); err != nil {
		return err
	}
	if err := validateFact("item.weapon.weight", data.Weight, sources); err != nil {
		return err
	}
	if err := validateFact("item.weapon.attackPhysical", data.AttackPhysical, sources); err != nil {
		return err
	}
	if err := validateFact("item.weapon.requiredStrength", data.RequiredStrength, sources); err != nil {
		return err
	}
	if err := validateFact("item.weapon.requiredDexterity", data.RequiredDexterity, sources); err != nil {
		return err
	}
	return validateFact("item.weapon.critical", data.Critical, sources)
}

func validateAshOfWarData(data AshOfWarData, sources map[SourceID]struct{}) error {
	if err := validateFact("item.ashOfWar.sourceRowID", data.SourceRowID, sources); err != nil {
		return err
	}
	if data.SourceRowID.Known && data.SourceRowID.Value == 0 {
		return fmt.Errorf("item.ashOfWar.sourceRowID must be greater than zero when known")
	}
	return validateFact("item.ashOfWar.compatibilityMask", data.CompatibilityMask, sources)
}
