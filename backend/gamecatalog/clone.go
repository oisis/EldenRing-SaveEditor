package gamecatalog

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func cloneManifest(manifest schema.Manifest) schema.Manifest {
	cloned := manifest
	cloned.Sources = append([]schema.DataSource(nil), manifest.Sources...)
	return cloned
}

func cloneResource(resource schema.Resource) schema.Resource {
	cloned := resource
	if resource.Item == nil {
		return cloned
	}

	item := *resource.Item
	item.Flags.Value = cloneStrings(resource.Item.Flags.Value)
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
		armor := *resource.Item.Armor
		item.Armor = &armor
	}
	if resource.Item.Talisman != nil {
		talisman := *resource.Item.Talisman
		item.Talisman = &talisman
	}
	if resource.Item.AshOfWar != nil {
		ash := *resource.Item.AshOfWar
		ash.CompatibleClassNames.Value = cloneStrings(resource.Item.AshOfWar.CompatibleClassNames.Value)
		item.AshOfWar = &ash
	}
	if resource.Item.Spell != nil {
		spell := *resource.Item.Spell
		item.Spell = &spell
	}
	if resource.Item.SpiritAsh != nil {
		spiritAsh := *resource.Item.SpiritAsh
		item.SpiritAsh = &spiritAsh
	}
	if resource.Item.Goods != nil {
		goods := *resource.Item.Goods
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
	cloned.Flags.Value = cloneStrings(data.Flags.Value)
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
		spiritAsh := *data.SpiritAsh
		cloned.SpiritAsh = &spiritAsh
	}
	return cloned
}

func cloneCapabilities(capabilities schema.ItemCapabilities) schema.ItemCapabilities {
	cloned := capabilities
	cloned.Upgrade = cloneCapability(capabilities.Upgrade, nil)
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
	cloned.PassiveEffects = append([]schema.WeaponPassiveEffectData(nil), data.PassiveEffects...)
	cloned.Warnings.Value = cloneStrings(data.Warnings.Value)
	return cloned
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
