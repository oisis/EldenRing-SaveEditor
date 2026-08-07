package migration

import "github.com/oisis/EldenRing-SaveForge/backend/db/data"

func enrichLegacySeed(result *seed) {
	id := result.ID
	if value, ok := data.ItemTexts[id]; ok {
		result.Text = copyLegacyText(value)
	}
	if value, ok := data.Descriptions[id]; ok {
		result.Description = copyLegacyDescription(value)
	}
	if value, ok := data.GameLimitsByItemID[id]; ok {
		result.GameLimits = &gameLimitsSeed{
			MaxInventory:   value.MaxInventory,
			MaxStorage:     value.MaxStorage,
			InventoryKnown: value.InventoryKnown,
			StorageKnown:   value.StorageKnown,
		}
	}
	if value, ok := data.WeaponStatsV1ByID[id]; ok {
		result.WeaponStats = copyWeaponStats(value)
	}
	if value, ok := data.ItemWeights[id]; ok {
		result.Weight = &value
	}
	if value, ok := data.WeaponGemMounts[id]; ok {
		result.WeaponEdit = &weaponEditSeed{
			WepType:           value.WepType,
			GemMountType:      value.GemMountType,
			CanChangeAffinity: value.CanChangeAffinity,
		}
		if bit, exists := data.WepTypeToCanMountBit[value.WepType]; exists {
			result.WeaponEdit.CompatibilityBit = &bit
		}
	}
	if value, ok := data.AoWCompatMasks[id]; ok {
		result.AoWCompatMask = &value
		result.AoWCompatibleClasses = legacyAoWCompatibleClasses(value)
	}
	if value, ok := data.SpellMemorySlots[id]; ok {
		result.SpellMemory = &value
	}
	if value, ok := data.ItemSortKeys[id]; ok {
		result.SortKey = &sortKeySeed{
			SortID:      value.SortId,
			SortGroupID: value.SortGroupId,
		}
	}
	result.Acquisition = collectLegacyAcquisition(id)
	if value, ok := data.EquipLoadModifiers[id]; ok {
		result.EquipLoad = &equipLoadSeed{
			EnduranceBonus: int32(value.EnduranceBonus),
			EquipLoadRate:  value.EquipLoadRate,
		}
	}
}

func collectLegacyAcquisition(itemID uint32) acquisitionSeed {
	result := acquisitionSeed{
		IsContainer:           data.IsContainerItem(itemID),
		ContainerPickupFlags:  cloneUint32s(data.ContainerPickupFlags[itemID]),
		ContainerVendorFlags:  cloneUint32s(data.ContainerVendorPurchaseFlags[itemID]),
		BolsteringPickupFlags: cloneUint32s(data.BolsteringPickupFlags[itemID]),
		CompanionEventFlagIDs: cloneUint32s(data.CompanionEventFlagsForItem(itemID)),
	}
	if value, ok := data.RequiredContainer[itemID]; ok {
		result.RequiredContainerID = &value
	}
	if value, ok := data.WorldPickupFlagID[itemID]; ok {
		result.WorldPickupFlagID = &value
	}
	return result
}

func cloneUint32s(values []uint32) []uint32 {
	return append([]uint32(nil), values...)
}

func legacyAoWCompatibleClasses(mask uint64) []string {
	result := make([]string, 0, len(data.CanMountWepNames))
	for bit, name := range data.CanMountWepNames {
		if mask&(uint64(1)<<uint(bit)) != 0 {
			result = append(result, name)
		}
	}
	return result
}

func copyLegacyText(value data.ItemTextData) *textSeed {
	return &textSeed{
		Caption:           value.Caption,
		Description:       value.Description,
		Location:          value.Location,
		CaptionSource:     string(value.CaptionSource),
		DescriptionSource: string(value.DescriptionSource),
		LocationSource:    string(value.LocationSource),
		DLCSource:         value.DLCSource,
		Notes:             value.Notes,
	}
}
