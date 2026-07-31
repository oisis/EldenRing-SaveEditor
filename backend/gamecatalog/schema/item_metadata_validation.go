package schema

import "fmt"

func validateTextMetadata(metadata ItemTextMetadata, sources map[SourceID]struct{}) error {
	facts := []struct {
		name string
		fact Fact[string]
	}{
		{"item.presentation.textMetadata.displayNameSource", metadata.DisplayNameSource},
		{"item.presentation.textMetadata.canonicalSource", metadata.CanonicalSource},
		{"item.presentation.textMetadata.captionSource", metadata.CaptionSource},
		{"item.presentation.textMetadata.descriptionSource", metadata.DescriptionSource},
		{"item.presentation.textMetadata.locationSource", metadata.LocationSource},
		{"item.presentation.textMetadata.dlcSource", metadata.DLCSource},
		{"item.presentation.textMetadata.notes", metadata.Notes},
	}
	for _, entry := range facts {
		if err := validateOptionalFact(entry.name, entry.fact, sources); err != nil {
			return err
		}
	}
	return nil
}

func validateAcquisition(acquisition ItemAcquisition, sources map[SourceID]struct{}) error {
	if err := validateOptionalFact("item.acquisition.requiredContainerID", acquisition.RequiredContainerID, sources); err != nil {
		return err
	}
	if acquisition.RequiredContainerID.Known && acquisition.RequiredContainerID.Value == 0 {
		return fmt.Errorf("item.acquisition.requiredContainerID must be greater than zero when known")
	}
	if err := validateOptionalFact("item.acquisition.isContainer", acquisition.IsContainer, sources); err != nil {
		return err
	}
	lists := []struct {
		name string
		fact Fact[[]uint32]
	}{
		{"item.acquisition.containerPickupFlagIDs", acquisition.ContainerPickupFlagIDs},
		{"item.acquisition.containerVendorFlagIDs", acquisition.ContainerVendorFlagIDs},
		{"item.acquisition.bolsteringPickupFlagIDs", acquisition.BolsteringPickupFlagIDs},
		{"item.acquisition.companionEventFlagIDs", acquisition.CompanionEventFlagIDs},
	}
	for _, entry := range lists {
		if err := validateOptionalIDList(entry.name, entry.fact, sources); err != nil {
			return err
		}
	}
	if err := validateOptionalFact("item.acquisition.worldPickupFlagID", acquisition.WorldPickupFlagID, sources); err != nil {
		return err
	}
	if acquisition.WorldPickupFlagID.Known && acquisition.WorldPickupFlagID.Value == 0 {
		return fmt.Errorf("item.acquisition.worldPickupFlagID must be greater than zero when known")
	}
	return nil
}

func validateModifiers(modifiers ItemModifiers, sources map[SourceID]struct{}) error {
	if modifiers.EquipLoad == nil {
		return nil
	}
	if err := validateFact(
		"item.modifiers.equipLoad.enduranceBonus",
		modifiers.EquipLoad.EnduranceBonus,
		sources,
	); err != nil {
		return err
	}
	if !modifiers.EquipLoad.EnduranceBonus.Known {
		return fmt.Errorf("item.modifiers.equipLoad.enduranceBonus must be known")
	}
	if err := validateFact(
		"item.modifiers.equipLoad.equipLoadRate",
		modifiers.EquipLoad.EquipLoadRate,
		sources,
	); err != nil {
		return err
	}
	if !modifiers.EquipLoad.EquipLoadRate.Known {
		return fmt.Errorf("item.modifiers.equipLoad.equipLoadRate must be known")
	}
	return nil
}

func validateDescriptionRecord(record ItemDescriptionRecord, sources map[SourceID]struct{}) error {
	if err := validateOptionalFact("technicalRecord.description.description", record.Description, sources); err != nil {
		return err
	}
	if err := validateOptionalFact("technicalRecord.description.location", record.Location, sources); err != nil {
		return err
	}
	if err := validateOptionalFact("technicalRecord.description.weight", record.Weight, sources); err != nil {
		return err
	}
	count := 0
	for _, present := range []bool{record.Weapon != nil, record.Armor != nil, record.Spell != nil} {
		if present {
			count++
		}
	}
	if count > 1 {
		return fmt.Errorf("technicalRecord.description must have at most one stats section")
	}
	if record.Weapon != nil {
		return validateCuratedWeaponStats(*record.Weapon, sources)
	}
	if record.Armor != nil {
		return validateCuratedArmorStats(*record.Armor, sources)
	}
	if record.Spell != nil {
		return validateCuratedSpellStats(*record.Spell, sources)
	}
	return nil
}

func validateCuratedWeaponStats(data CuratedWeaponStats, sources map[SourceID]struct{}) error {
	return validateFactValidators([]struct {
		name string
		fact anyFactValidator
	}{
		{"item.legacy.description.weapon.weight", factValidator(data.Weight)},
		{"item.legacy.description.weapon.attackPhysical", factValidator(data.AttackPhysical)},
		{"item.legacy.description.weapon.attackMagic", factValidator(data.AttackMagic)},
		{"item.legacy.description.weapon.attackFire", factValidator(data.AttackFire)},
		{"item.legacy.description.weapon.attackLightning", factValidator(data.AttackLightning)},
		{"item.legacy.description.weapon.attackHoly", factValidator(data.AttackHoly)},
		{"item.legacy.description.weapon.scalingStrength", factValidator(data.ScalingStrength)},
		{"item.legacy.description.weapon.scalingDexterity", factValidator(data.ScalingDexterity)},
		{"item.legacy.description.weapon.scalingIntelligence", factValidator(data.ScalingIntelligence)},
		{"item.legacy.description.weapon.scalingFaith", factValidator(data.ScalingFaith)},
		{"item.legacy.description.weapon.requiredStrength", factValidator(data.RequiredStrength)},
		{"item.legacy.description.weapon.requiredDexterity", factValidator(data.RequiredDexterity)},
		{"item.legacy.description.weapon.requiredIntelligence", factValidator(data.RequiredIntelligence)},
		{"item.legacy.description.weapon.requiredFaith", factValidator(data.RequiredFaith)},
		{"item.legacy.description.weapon.requiredArcane", factValidator(data.RequiredArcane)},
	}, sources)
}

func validateCuratedArmorStats(data CuratedArmorStats, sources map[SourceID]struct{}) error {
	return validateFactValidators([]struct {
		name string
		fact anyFactValidator
	}{
		{"item.legacy.description.armor.weight", factValidator(data.Weight)},
		{"item.legacy.description.armor.physical", factValidator(data.Physical)},
		{"item.legacy.description.armor.strike", factValidator(data.Strike)},
		{"item.legacy.description.armor.slash", factValidator(data.Slash)},
		{"item.legacy.description.armor.pierce", factValidator(data.Pierce)},
		{"item.legacy.description.armor.magic", factValidator(data.Magic)},
		{"item.legacy.description.armor.fire", factValidator(data.Fire)},
		{"item.legacy.description.armor.lightning", factValidator(data.Lightning)},
		{"item.legacy.description.armor.holy", factValidator(data.Holy)},
		{"item.legacy.description.armor.immunity", factValidator(data.Immunity)},
		{"item.legacy.description.armor.robustness", factValidator(data.Robustness)},
		{"item.legacy.description.armor.focus", factValidator(data.Focus)},
		{"item.legacy.description.armor.vitality", factValidator(data.Vitality)},
		{"item.legacy.description.armor.poise", factValidator(data.Poise)},
	}, sources)
}

func validateCuratedSpellStats(data CuratedSpellStats, sources map[SourceID]struct{}) error {
	return validateFactValidators([]struct {
		name string
		fact anyFactValidator
	}{
		{"item.legacy.description.spell.fpCost", factValidator(data.FPCost)},
		{"item.legacy.description.spell.memorySlots", factValidator(data.MemorySlots)},
		{"item.legacy.description.spell.requiredIntelligence", factValidator(data.RequiredIntelligence)},
		{"item.legacy.description.spell.requiredFaith", factValidator(data.RequiredFaith)},
		{"item.legacy.description.spell.requiredArcane", factValidator(data.RequiredArcane)},
	}, sources)
}

func validateOptionalIDList(name string, fact Fact[[]uint32], sources map[SourceID]struct{}) error {
	if err := validateOptionalFact(name, fact, sources); err != nil {
		return err
	}
	if !fact.Known {
		return nil
	}
	seen := make(map[uint32]struct{}, len(fact.Value))
	for index, id := range fact.Value {
		if id == 0 {
			return fmt.Errorf("%s[%d] must be greater than zero", name, index)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("%s contains duplicate ID %d", name, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateAliases(aliases []ItemAlias, item ItemDocument, sources map[SourceID]struct{}) error {
	occupied := make(map[uint32]struct{}, len(item.Variants)+1)
	occupied[item.GameID.Value] = struct{}{}
	for _, variant := range item.Variants {
		occupied[variant.GameID.Value] = struct{}{}
	}
	for index, alias := range aliases {
		name := fmt.Sprintf("item.aliases[%d]", index)
		if err := validateFact(name+".gameID", alias.GameID, sources); err != nil {
			return err
		}
		if !alias.GameID.Known || alias.GameID.Value == 0 {
			return fmt.Errorf("%s.gameID must be known and greater than zero", name)
		}
		if _, exists := occupied[alias.GameID.Value]; exists {
			return fmt.Errorf("%s: duplicate game ID 0x%08X", name, alias.GameID.Value)
		}
		occupied[alias.GameID.Value] = struct{}{}
		if err := validateParameterRecords(name+".sourceRecords", alias.SourceRecords, sources); err != nil {
			return err
		}
		identity := alias.GameID
		if err := validateRegulationProvenanceCoverage(
			name,
			identity,
			alias.SourceRecords,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateUnlocks(unlocks []ItemUnlock, sources map[SourceID]struct{}) error {
	seen := make(map[string]struct{}, len(unlocks))
	for index, unlock := range unlocks {
		name := fmt.Sprintf("item.unlocks[%d]", index)
		if err := validateFact(name+".kind", unlock.Kind, sources); err != nil {
			return err
		}
		if !unlock.Kind.Known || !validUnlockKind(unlock.Kind.Value) {
			return fmt.Errorf("%s.kind must be known and supported", name)
		}
		if err := validateFact(name+".eventFlagID", unlock.EventFlagID, sources); err != nil {
			return err
		}
		if !unlock.EventFlagID.Known || unlock.EventFlagID.Value == 0 {
			return fmt.Errorf("%s.eventFlagID must be known and greater than zero", name)
		}
		if err := validateOptionalNonEmptyString(name+".name", unlock.Name, sources); err != nil {
			return err
		}
		if err := validateOptionalNonEmptyString(name+".category", unlock.Category, sources); err != nil {
			return err
		}
		key := fmt.Sprintf("%s:%d", unlock.Kind.Value, unlock.EventFlagID.Value)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%s: duplicate unlock %s", name, key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validUnlockKind(kind string) bool {
	switch kind {
	case "ash_of_war", "bell_bearing", "cookbook", "map", "whetblade":
		return true
	default:
		return false
	}
}
