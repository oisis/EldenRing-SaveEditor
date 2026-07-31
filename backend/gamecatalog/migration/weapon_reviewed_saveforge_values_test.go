package migration

import "testing"

func TestReviewedWeaponSaveForgeValuesAreDiscarded(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	ammunitionCount := 0
	for _, resource := range catalog.Resources {
		item := resource.Item
		if item == nil || item.Category.Value != "arrows_and_bolts" {
			continue
		}
		ammunitionCount++
		weapon := item.Weapon
		if weapon.WeightSFV != nil ||
			weapon.AttackMagicSFV != nil ||
			weapon.AttackFireSFV != nil ||
			weapon.AttackLightningSFV != nil ||
			weapon.AttackHolySFV != nil {
			t.Fatalf(
				"ammunition 0x%08X retains reviewed SFV: %+v",
				item.GameID.Value,
				weapon,
			)
		}
	}
	if ammunitionCount == 0 {
		t.Fatal("generated catalog contains no ammunition")
	}

	weaponCount := 0
	for _, resource := range catalog.Resources {
		item := resource.Item
		if item == nil || item.Weapon == nil {
			continue
		}
		weaponCount++
		if item.Weapon.SwordArtsNameSFV != nil {
			t.Fatalf(
				"weapon 0x%08X retains reviewed swordArtsName-sfv: %+v",
				item.GameID.Value,
				item.Weapon.SwordArtsNameSFV,
			)
		}
		if item.Weapon.AttackPhysicalSFV != nil {
			t.Fatalf(
				"weapon 0x%08X retains reviewed attackPhysical-sfv: %+v",
				item.GameID.Value,
				item.Weapon.AttackPhysicalSFV,
			)
		}
	}
	if weaponCount == 0 {
		t.Fatal("generated catalog contains no weapons")
	}

	giantsRedBraid := findGeneratedItem(t, catalog, giantsRedBraidItemID).Weapon
	if giantsRedBraid.AttackFire.Value != 54 ||
		giantsRedBraid.AttackFireSFV != nil {
		t.Fatalf("Giant's Red Braid attack fire = %+v", giantsRedBraid.AttackFire)
	}

	magmaWhip := findGeneratedItem(t, catalog, magmaWhipCandlestickItemID).Weapon
	if magmaWhip.AttackFire.Value != 74 || magmaWhip.AttackFireSFV != nil {
		t.Fatalf("Magma Whip Candlestick attack fire = %+v", magmaWhip.AttackFire)
	}

	meteoriteStaff := findGeneratedItem(t, catalog, meteoriteStaffItemID).Weapon
	if meteoriteStaff.MaxUpgrade.Value != 0 || meteoriteStaff.MaxUpgradeSFV != nil {
		t.Fatalf("Meteorite Staff max upgrade = %+v", meteoriteStaff.MaxUpgrade)
	}

	velvetSword := findGeneratedItem(t, catalog, velvetSwordOfSaintTrinaItemID).Weapon
	if velvetSword.Weight.Value != 2.5 ||
		velvetSword.AttackPhysical.Value != 95 ||
		velvetSword.AttackMagic.Value != 61 ||
		velvetSword.RequiredStrength.Value != 10 ||
		velvetSword.RequiredDexterity.Value != 12 ||
		velvetSword.RequiredIntelligence.Value != 14 {
		t.Fatalf("Velvet Sword Regulation values = %+v", velvetSword)
	}
	if velvetSword.WeightSFV != nil ||
		velvetSword.AttackPhysicalSFV != nil ||
		velvetSword.AttackMagicSFV != nil ||
		velvetSword.RequiredStrengthSFV != nil ||
		velvetSword.RequiredDexteritySFV != nil ||
		velvetSword.RequiredIntelligenceSFV != nil {
		t.Fatalf("Velvet Sword retains reviewed SFV: %+v", velvetSword)
	}
}

func TestReviewedWeaponAttackPhysicalAlwaysUsesRegulation(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	boneArrow := findGeneratedItem(t, catalog, 0x02FB65B0).Weapon
	if boneArrow.AttackPhysical.Value != 42 || boneArrow.AttackPhysicalSFV != nil {
		t.Fatalf(
			"Bone Arrow (Fletched) attack physical = %+v, want Regulation value 42",
			boneArrow.AttackPhysical,
		)
	}

	magmaWhip := findGeneratedItem(t, catalog, magmaWhipCandlestickItemID).Weapon
	if magmaWhip.AttackPhysical.Value != 74 || magmaWhip.AttackPhysicalSFV != nil {
		t.Fatalf(
			"Magma Whip Candlestick attack physical = %+v, want Regulation value 74",
			magmaWhip.AttackPhysical,
		)
	}
}
