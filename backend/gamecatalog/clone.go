package gamecatalog

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func cloneManifest(manifest schema.Manifest) schema.Manifest {
	cloned := manifest
	cloned.Sources = append([]schema.DataSource(nil), manifest.Sources...)
	return cloned
}

func cloneResource(resource schema.Resource) schema.Resource {
	cloned := resource
	if resource.Colosseum != nil {
		colosseum := *resource.Colosseum
		cloned.Colosseum = &colosseum
	}
	if resource.Region != nil {
		region := *resource.Region
		cloned.Region = &region
	}
	if resource.Item == nil {
		return cloned
	}

	item := *resource.Item
	item.Storage = cloneStorage(resource.Item.Storage)
	item.Acquisition = cloneAcquisition(resource.Item.Acquisition)
	item.Modifiers = cloneModifiers(resource.Item.Modifiers)
	item.Links = cloneItemLinks(resource.Item.Links)
	item.Variants = cloneVariants(resource.Item.Variants)
	item.Aliases = cloneAliases(resource.Item.Aliases)
	item.Unlocks = append([]schema.ItemUnlock(nil), resource.Item.Unlocks...)
	item.RelatedTechnicalRecords = cloneRelatedTechnicalRecords(resource.Item.RelatedTechnicalRecords)
	item.SourceRecords = cloneParameterRecords(resource.Item.SourceRecords)
	item.Capabilities = cloneCapabilities(resource.Item.Capabilities)
	if resource.Item.Weapon != nil {
		weapon := cloneWeaponData(*resource.Item.Weapon)
		item.Weapon = &weapon
	}
	if resource.Item.Armor != nil {
		armor := cloneArmorData(*resource.Item.Armor)
		item.Armor = &armor
	}
	if resource.Item.Talisman != nil {
		talisman := cloneTalismanData(*resource.Item.Talisman)
		item.Talisman = &talisman
	}
	if resource.Item.AshOfWar != nil {
		ash := cloneAshOfWarData(*resource.Item.AshOfWar)
		item.AshOfWar = &ash
	}
	if resource.Item.Spell != nil {
		spell := cloneSpellData(*resource.Item.Spell)
		item.Spell = &spell
	}
	if resource.Item.SpiritAsh != nil {
		spiritAsh := cloneSpiritAshData(*resource.Item.SpiritAsh)
		item.SpiritAsh = &spiritAsh
	}
	if resource.Item.Goods != nil {
		goods := cloneGoodsData(*resource.Item.Goods)
		item.Goods = &goods
	}
	if resource.Item.Gesture != nil {
		gesture := *resource.Item.Gesture
		gesture.Slots = cloneGestureSlots(resource.Item.Gesture.Slots)
		item.Gesture = &gesture
	}
	cloned.Item = &item
	return cloned
}

func cloneCapability[T any](capability schema.Capability[T], cloneRules func(T) T) schema.Capability[T] {
	cloned := capability
	if capability.Rules == nil {
		return cloned
	}
	rules := *capability.Rules
	if cloneRules != nil {
		rules = cloneRules(rules)
	}
	cloned.Rules = &rules
	cloned.RulesEvidence = append([]schema.Provenance(nil), capability.RulesEvidence...)
	return cloned
}

func cloneRelations(relations []schema.Relation) []schema.Relation {
	return append([]schema.Relation(nil), relations...)
}

func cloneVariants(variants []schema.ItemVariant) []schema.ItemVariant {
	cloned := append([]schema.ItemVariant(nil), variants...)
	for index := range cloned {
		cloned[index].Data = cloneVariantDocumentData(variants[index].Data)
		cloned[index].SourceRecords = cloneParameterRecords(variants[index].SourceRecords)
	}
	return cloned
}

func cloneVariantDocumentData(data schema.VariantDocumentData) schema.VariantDocumentData {
	cloned := data
	cloned.Storage = cloneStorage(data.Storage)
	cloned.Acquisition = cloneAcquisition(data.Acquisition)
	cloned.Modifiers = cloneModifiers(data.Modifiers)
	cloned.Links = cloneItemLinks(data.Links)
	cloned.Unlocks = append([]schema.ItemUnlock(nil), data.Unlocks...)
	cloned.RelatedTechnicalRecords = cloneRelatedTechnicalRecords(data.RelatedTechnicalRecords)
	cloned.Capabilities = cloneCapabilities(data.Capabilities)
	if data.Weapon != nil {
		weapon := cloneWeaponData(*data.Weapon)
		cloned.Weapon = &weapon
	}
	if data.SpiritAsh != nil {
		spiritAsh := cloneSpiritAshData(*data.SpiritAsh)
		cloned.SpiritAsh = &spiritAsh
	}
	return cloned
}

func cloneCapabilities(capabilities schema.ItemCapabilities) schema.ItemCapabilities {
	cloned := capabilities
	cloned.Upgrade = cloneCapability(
		capabilities.Upgrade,
		func(rules schema.UpgradeRules) schema.UpgradeRules {
			rules.MaxLevelSFV = cloneFactPointer(rules.MaxLevelSFV)
			return rules
		},
	)
	cloned.Infusion = cloneCapability(
		capabilities.Infusion,
		func(rules schema.InfusionRules) schema.InfusionRules {
			rules.AllowedAffinities = append([]schema.Affinity(nil), rules.AllowedAffinities...)
			return rules
		},
	)
	cloned.AshOfWarMount = cloneCapability(capabilities.AshOfWarMount, nil)
	cloned.Stack = cloneCapability(capabilities.Stack, nil)
	cloned.Equipment = cloneCapability(
		capabilities.Equipment,
		func(rules schema.EquipmentRules) schema.EquipmentRules {
			rules.AllowedSlots = append([]schema.EquipmentSlot(nil), rules.AllowedSlots...)
			return rules
		},
	)
	return cloned
}

func cloneAcquisition(acquisition schema.ItemAcquisition) schema.ItemAcquisition {
	cloned := acquisition
	cloned.ContainerPickupFlagIDs.Value = cloneUint32s(acquisition.ContainerPickupFlagIDs.Value)
	cloned.ContainerVendorFlagIDs.Value = cloneUint32s(acquisition.ContainerVendorFlagIDs.Value)
	cloned.BolsteringPickupFlagIDs.Value = cloneUint32s(acquisition.BolsteringPickupFlagIDs.Value)
	cloned.CompanionEventFlagIDs.Value = cloneUint32s(acquisition.CompanionEventFlagIDs.Value)
	return cloned
}

func cloneModifiers(modifiers schema.ItemModifiers) schema.ItemModifiers {
	cloned := modifiers
	if modifiers.EquipLoad != nil {
		equipLoad := *modifiers.EquipLoad
		equipLoad.EnduranceBonusSFV = cloneFactPointer(modifiers.EquipLoad.EnduranceBonusSFV)
		equipLoad.EquipLoadRateSFV = cloneFactPointer(modifiers.EquipLoad.EquipLoadRateSFV)
		cloned.EquipLoad = &equipLoad
	}
	return cloned
}

func cloneDescriptionRecord(record schema.ItemDescriptionRecord) schema.ItemDescriptionRecord {
	cloned := record
	if record.Weapon != nil {
		weapon := *record.Weapon
		cloned.Weapon = &weapon
	}
	if record.Armor != nil {
		armor := *record.Armor
		cloned.Armor = &armor
	}
	if record.Spell != nil {
		spell := *record.Spell
		cloned.Spell = &spell
	}
	return cloned
}

func cloneItemLinks(links schema.ItemLinks) schema.ItemLinks {
	cloned := links
	cloned.RelatedEventFlags = append([]schema.RelatedEventFlag(nil), links.RelatedEventFlags...)
	cloned.RelatedItems = append([]schema.RelatedItem(nil), links.RelatedItems...)
	if links.MapFragment != nil {
		mapFragment := *links.MapFragment
		cloned.MapFragment = &mapFragment
	}
	return cloned
}

func cloneRelatedTechnicalRecords(records []schema.RelatedTechnicalRecord) []schema.RelatedTechnicalRecord {
	cloned := append([]schema.RelatedTechnicalRecord(nil), records...)
	for index := range cloned {
		cloned[index].Description = cloneDescriptionRecord(records[index].Description)
		cloned[index].SourceRecords = cloneParameterRecords(records[index].SourceRecords)
	}
	return cloned
}

func cloneWeaponData(data schema.WeaponData) schema.WeaponData {
	cloned := data
	cloned.WeaponTypeIDSFV = cloneFactPointer(data.WeaponTypeIDSFV)
	cloned.SortIDSFV = cloneFactPointer(data.SortIDSFV)
	cloned.SortGroupIDSFV = cloneFactPointer(data.SortGroupIDSFV)
	cloned.ReinforceTypeIDSFV = cloneFactPointer(data.ReinforceTypeIDSFV)
	cloned.GemMountTypeSFV = cloneFactPointer(data.GemMountTypeSFV)
	cloned.WeightSFV = cloneFactPointer(data.WeightSFV)
	cloned.AttackPhysicalSFV = cloneFactPointer(data.AttackPhysicalSFV)
	cloned.AttackMagicSFV = cloneFactPointer(data.AttackMagicSFV)
	cloned.AttackFireSFV = cloneFactPointer(data.AttackFireSFV)
	cloned.AttackLightningSFV = cloneFactPointer(data.AttackLightningSFV)
	cloned.AttackHolySFV = cloneFactPointer(data.AttackHolySFV)
	cloned.AttackStaminaSFV = cloneFactPointer(data.AttackStaminaSFV)
	cloned.GuardPhysicalSFV = cloneFactPointer(data.GuardPhysicalSFV)
	cloned.GuardMagicSFV = cloneFactPointer(data.GuardMagicSFV)
	cloned.GuardFireSFV = cloneFactPointer(data.GuardFireSFV)
	cloned.GuardLightningSFV = cloneFactPointer(data.GuardLightningSFV)
	cloned.GuardHolySFV = cloneFactPointer(data.GuardHolySFV)
	cloned.GuardBoostSFV = cloneFactPointer(data.GuardBoostSFV)
	cloned.RequiredStrengthSFV = cloneFactPointer(data.RequiredStrengthSFV)
	cloned.RequiredDexteritySFV = cloneFactPointer(data.RequiredDexteritySFV)
	cloned.RequiredIntelligenceSFV = cloneFactPointer(data.RequiredIntelligenceSFV)
	cloned.RequiredFaithSFV = cloneFactPointer(data.RequiredFaithSFV)
	cloned.RequiredArcaneSFV = cloneFactPointer(data.RequiredArcaneSFV)
	cloned.CriticalSFV = cloneFactPointer(data.CriticalSFV)
	cloned.DefaultAshOfWarIDSFV = cloneFactPointer(data.DefaultAshOfWarIDSFV)
	cloned.SwordArtsNameSFV = cloneFactPointer(data.SwordArtsNameSFV)
	cloned.IsInfusableSFV = cloneFactPointer(data.IsInfusableSFV)
	cloned.IsSomberSFV = cloneFactPointer(data.IsSomberSFV)
	cloned.MaxUpgradeSFV = cloneFactPointer(data.MaxUpgradeSFV)
	cloned.PassiveEffects = append([]schema.WeaponPassiveEffectData(nil), data.PassiveEffects...)
	cloned.Warnings.Value = cloneStrings(data.Warnings.Value)
	return cloned
}

func cloneStorage(storage schema.ItemStorage) schema.ItemStorage {
	cloned := storage
	cloned.SafeModeMaxInventory = cloneFactPointer(storage.SafeModeMaxInventory)
	cloned.MaxInventorySFV = cloneFactPointer(storage.MaxInventorySFV)
	cloned.SafeModeMaxStorage = cloneFactPointer(storage.SafeModeMaxStorage)
	cloned.MaxStorageSFV = cloneFactPointer(storage.MaxStorageSFV)
	return cloned
}

func cloneArmorData(data schema.ArmorData) schema.ArmorData {
	cloned := data
	cloned.SortIDSFV = cloneFactPointer(data.SortIDSFV)
	cloned.SortGroupIDSFV = cloneFactPointer(data.SortGroupIDSFV)
	cloned.WeightSFV = cloneFactPointer(data.WeightSFV)
	cloned.PhysicalSFV = cloneFactPointer(data.PhysicalSFV)
	cloned.StrikeSFV = cloneFactPointer(data.StrikeSFV)
	cloned.SlashSFV = cloneFactPointer(data.SlashSFV)
	cloned.PierceSFV = cloneFactPointer(data.PierceSFV)
	cloned.MagicSFV = cloneFactPointer(data.MagicSFV)
	cloned.FireSFV = cloneFactPointer(data.FireSFV)
	cloned.LightningSFV = cloneFactPointer(data.LightningSFV)
	cloned.HolySFV = cloneFactPointer(data.HolySFV)
	cloned.ImmunitySFV = cloneFactPointer(data.ImmunitySFV)
	cloned.RobustnessSFV = cloneFactPointer(data.RobustnessSFV)
	cloned.FocusSFV = cloneFactPointer(data.FocusSFV)
	cloned.VitalitySFV = cloneFactPointer(data.VitalitySFV)
	cloned.PoiseSFV = cloneFactPointer(data.PoiseSFV)
	return cloned
}

func cloneTalismanData(data schema.TalismanData) schema.TalismanData {
	cloned := data
	cloned.SortIDSFV = cloneFactPointer(data.SortIDSFV)
	cloned.SortGroupIDSFV = cloneFactPointer(data.SortGroupIDSFV)
	cloned.WeightSFV = cloneFactPointer(data.WeightSFV)
	return cloned
}

func cloneAshOfWarData(data schema.AshOfWarData) schema.AshOfWarData {
	cloned := data
	cloned.SortIDSFV = cloneFactPointer(data.SortIDSFV)
	cloned.SortGroupIDSFV = cloneFactPointer(data.SortGroupIDSFV)
	cloned.SwordArtsNameSFV = cloneFactPointer(data.SwordArtsNameSFV)
	cloned.CompatibilityMaskSFV = cloneFactPointer(data.CompatibilityMaskSFV)
	cloned.CompatibleClassNames.Value = cloneStrings(data.CompatibleClassNames.Value)
	return cloned
}

func cloneSpellData(data schema.SpellData) schema.SpellData {
	cloned := data
	cloned.SortIDSFV = cloneFactPointer(data.SortIDSFV)
	cloned.FPCostSFV = cloneFactPointer(data.FPCostSFV)
	cloned.MemorySlotsSFV = cloneFactPointer(data.MemorySlotsSFV)
	cloned.RequiredIntelligenceSFV = cloneFactPointer(data.RequiredIntelligenceSFV)
	cloned.RequiredFaithSFV = cloneFactPointer(data.RequiredFaithSFV)
	cloned.RequiredArcaneSFV = cloneFactPointer(data.RequiredArcaneSFV)
	return cloned
}

func cloneSpiritAshData(data schema.SpiritAshData) schema.SpiritAshData {
	cloned := data
	cloned.SortIDSFV = cloneFactPointer(data.SortIDSFV)
	cloned.SortGroupIDSFV = cloneFactPointer(data.SortGroupIDSFV)
	return cloned
}

func cloneGoodsData(data schema.GoodsData) schema.GoodsData {
	cloned := data
	cloned.SortIDSFV = cloneFactPointer(data.SortIDSFV)
	cloned.SortGroupIDSFV = cloneFactPointer(data.SortGroupIDSFV)
	cloned.WeightSFV = cloneFactPointer(data.WeightSFV)
	return cloned
}

func cloneFactPointer[T any](fact *schema.Fact[T]) *schema.Fact[T] {
	if fact == nil {
		return nil
	}
	cloned := *fact
	return &cloned
}

func cloneAliases(aliases []schema.ItemAlias) []schema.ItemAlias {
	cloned := append([]schema.ItemAlias(nil), aliases...)
	for index := range cloned {
		cloned[index].SourceRecords = cloneParameterRecords(aliases[index].SourceRecords)
	}
	return cloned
}

func cloneGestureSlots(slots []schema.GestureSlotRecord) []schema.GestureSlotRecord {
	cloned := append([]schema.GestureSlotRecord(nil), slots...)
	for index := range cloned {
		cloned[index].Flags.Value = cloneStrings(slots[index].Flags.Value)
		cloned[index].SourceRecords = cloneParameterRecords(slots[index].SourceRecords)
	}
	return cloned
}

func cloneParameterRecords(records []schema.ParameterRecord) []schema.ParameterRecord {
	cloned := append([]schema.ParameterRecord(nil), records...)
	for index := range cloned {
		cloned[index].Fields = append([]schema.ParameterField(nil), records[index].Fields...)
	}
	return cloned
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func cloneUint32s(values []uint32) []uint32 {
	return append([]uint32(nil), values...)
}
