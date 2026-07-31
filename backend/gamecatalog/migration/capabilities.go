package migration

import (
	"fmt"
	"strconv"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) buildCapabilities(
	item seed,
	family schema.ItemFamily,
	primaryRow ParameterRow,
	hasPrimaryRow bool,
) (schema.ItemCapabilities, error) {
	if !item.HasLegacyItem && !item.RegulationOnlyVariant {
		capabilities := unknownCapabilities(
			"slot-only gesture has no ItemData capability contract",
		)
		equipment, err := buildEquipmentCapability(
			family,
			primaryRow,
			hasPrimaryRow,
		)
		if err != nil {
			return schema.ItemCapabilities{}, err
		}
		capabilities.Equipment = equipment
		return capabilities, nil
	}
	capabilities := schema.ItemCapabilities{
		Upgrade:       disabledCapability[schema.UpgradeRules]("legacy ItemData.MaxUpgrade is zero"),
		Infusion:      disabledCapability[schema.InfusionRules]("legacy weapon edit metadata disallows affinity changes"),
		AshOfWarMount: disabledCapability[schema.AshOfWarMountRules]("legacy weapon edit metadata disallows custom Ash of War mounting"),
	}
	stack, err := context.buildStackCapability(item, family, primaryRow, hasPrimaryRow)
	if err != nil {
		return schema.ItemCapabilities{}, err
	}
	capabilities.Stack = stack
	if family == schema.ItemFamilyWeapon {
		if !hasPrimaryRow {
			return schema.ItemCapabilities{}, fmt.Errorf(
				"weapon 0x%08X has no primary EquipParamWeapon row",
				item.ID,
			)
		}
		reinforceTypeID, err := regulationInt32(primaryRow, "reinforceTypeId")
		if err != nil {
			return schema.ItemCapabilities{}, err
		}
		bandSize, err := context.weaponReinforcementBandSize(reinforceTypeID)
		if err != nil {
			return schema.ItemCapabilities{}, err
		}
		if bandSize > 1 {
			model := schema.UpgradeModelStandard
			if bandSize == 11 {
				model = schema.UpgradeModelSomber
			}
			capabilities.Upgrade = enabledRegulationCapability(
				schema.UpgradeRules{
					Model:    model,
					MaxLevel: uint8(bandSize - 1),
				},
				RegulationTableReinforceWeapon,
				uint32(reinforceTypeID),
				"Row ID",
				"derived from the explicit ReinforceParamWeapon band",
			)
		} else {
			capabilities.Upgrade = disabledRegulationCapability[schema.UpgradeRules](
				RegulationTableReinforceWeapon,
				uint32(reinforceTypeID),
				"Row ID",
				"ReinforceParamWeapon band contains no upgrade levels",
			)
		}
	}
	if family == schema.ItemFamilySpiritAsh {
		if !hasPrimaryRow {
			return schema.ItemCapabilities{}, fmt.Errorf(
				"spirit ash 0x%08X has no primary EquipParamGoods row",
				item.ID,
			)
		}
		upgrade, err := context.spiritAshUpgrade(primaryRow)
		if err != nil {
			return schema.ItemCapabilities{}, err
		}
		capabilities.Upgrade = enabledRegulationCapability(
			upgrade.rules,
			RegulationTableGoods,
			primaryRow.RowID,
			"reinforceMaterialId,reinforceGoodsId",
			"derived from the complete EquipParamGoods chain and EquipMtrlSetParam material records",
		)
		capabilities.Upgrade.RulesEvidence = spiritAshUpgradeEvidence(upgrade)
	}
	if family == schema.ItemFamilyWeapon {
		gemMountType, err := regulationUint8(primaryRow, "gemMountType")
		if err != nil {
			return schema.ItemCapabilities{}, err
		}
		disableGemAffinity, err := regulationUint8(primaryRow, "disableGemAttr")
		if err != nil {
			return schema.ItemCapabilities{}, err
		}
		if disableGemAffinity > 1 {
			return schema.ItemCapabilities{}, fmt.Errorf(
				"weapon 0x%08X disableGemAttr=%d is not boolean",
				item.ID,
				disableGemAffinity,
			)
		}
		canChangeAffinity := gemMountType == 2 && disableGemAffinity == 0
		affinities, err := regulationAllowedAffinities(primaryRow)
		if err != nil {
			return schema.ItemCapabilities{}, err
		}
		if canChangeAffinity {
			if len(affinities) == 0 {
				return schema.ItemCapabilities{}, fmt.Errorf(
					"infusable weapon 0x%08X has no enabled origin affinity rows",
					item.ID,
				)
			}
			capabilities.Infusion = enabledRegulationCapability(
				schema.InfusionRules{AllowedAffinities: affinities},
				RegulationTableWeapon,
				primaryRow.RowID,
				"gemMountType,disableGemAttr,originEquipWep,originEquipWep1,originEquipWep2,originEquipWep3,originEquipWep4,originEquipWep5,originEquipWep6,originEquipWep7,originEquipWep8,originEquipWep9,originEquipWep10,originEquipWep11,originEquipWep12",
				"enabled by gemMountType == 2 and disableGemAttr == 0; allowed affinities copied from explicit origin rows",
			)
		} else {
			capabilities.Infusion = disabledRegulationCapability[schema.InfusionRules](
				RegulationTableWeapon,
				primaryRow.RowID,
				"gemMountType,disableGemAttr",
				"disabled by EquipParamWeapon gemMountType or disableGemAttr",
			)
		}
	}
	if family == schema.ItemFamilyWeapon {
		gemMountType, err := regulationUint8(primaryRow, "gemMountType")
		if err != nil {
			return schema.ItemCapabilities{}, err
		}
		if gemMountType == 2 {
			if item.WeaponEdit == nil || item.WeaponEdit.CompatibilityBit == nil {
				return schema.ItemCapabilities{}, fmt.Errorf(
					"weapon 0x%08X allows custom Ashes of War but has no verified compatibility mapping",
					item.ID,
				)
			}
			weaponType := item.Subcategory
			if weaponType == "" {
				weaponType = strconv.FormatUint(uint64(item.WeaponEdit.WepType), 10)
			}
			capabilities.AshOfWarMount = enabledRegulationCapability(
				schema.AshOfWarMountRules{
					Mode:             schema.AshOfWarMountModeCustom,
					WeaponType:       weaponType,
					CompatibilityBit: *item.WeaponEdit.CompatibilityBit,
				},
				RegulationTableWeapon,
				primaryRow.RowID,
				"gemMountType,wepType",
				"enabled by EquipParamWeapon.gemMountType; compatibility bit retained from the verified weapon-type mapping",
			)
			capabilities.AshOfWarMount.RulesEvidence = []schema.Provenance{{
				Source: sourceLegacyData,
				Method: "copied from the reviewed WepTypeToCanMountBit game-mechanics mapping",
			}}
		} else {
			capabilities.AshOfWarMount = disabledRegulationCapability[schema.AshOfWarMountRules](
				RegulationTableWeapon,
				primaryRow.RowID,
				"gemMountType",
				"disabled by EquipParamWeapon.gemMountType",
			)
		}
	}
	equipment, err := buildEquipmentCapability(family, primaryRow, hasPrimaryRow)
	if err != nil {
		return schema.ItemCapabilities{}, err
	}
	capabilities.Equipment = equipment
	return capabilities, nil
}

func regulationAllowedAffinities(row ParameterRow) ([]schema.Affinity, error) {
	fields := []struct {
		name     string
		affinity schema.Affinity
	}{
		{name: "originEquipWep", affinity: schema.AffinityStandard},
		{name: "originEquipWep1", affinity: schema.AffinityHeavy},
		{name: "originEquipWep2", affinity: schema.AffinityKeen},
		{name: "originEquipWep3", affinity: schema.AffinityQuality},
		{name: "originEquipWep4", affinity: schema.AffinityFire},
		{name: "originEquipWep5", affinity: schema.AffinityFlameArt},
		{name: "originEquipWep6", affinity: schema.AffinityLightning},
		{name: "originEquipWep7", affinity: schema.AffinitySacred},
		{name: "originEquipWep8", affinity: schema.AffinityMagic},
		{name: "originEquipWep9", affinity: schema.AffinityCold},
		{name: "originEquipWep10", affinity: schema.AffinityPoison},
		{name: "originEquipWep11", affinity: schema.AffinityBlood},
		{name: "originEquipWep12", affinity: schema.AffinityOccult},
	}
	result := make([]schema.Affinity, 0, len(fields))
	for _, field := range fields {
		value, err := regulationInt32(row, field.name)
		if err != nil {
			return nil, err
		}
		if value > 0 {
			result = append(result, field.affinity)
		}
	}
	return result, nil
}

func unknownCapabilities(method string) schema.ItemCapabilities {
	return schema.ItemCapabilities{
		Upgrade:       unknownCapability[schema.UpgradeRules](method),
		Infusion:      unknownCapability[schema.InfusionRules](method),
		AshOfWarMount: unknownCapability[schema.AshOfWarMountRules](method),
		Stack:         unknownCapability[schema.StackRules](method),
		Equipment:     unknownCapability[schema.EquipmentRules](method),
	}
}

func (context *generationContext) buildStackCapability(
	item seed,
	family schema.ItemFamily,
	primaryRow ParameterRow,
	hasPrimaryRow bool,
) (schema.Capability[schema.StackRules], error) {
	if family == schema.ItemFamilyWeapon && item.Category == "arrows_and_bolts" {
		if !hasPrimaryRow {
			return unknownCapability[schema.StackRules]("primary EquipParamWeapon row is missing"), nil
		}
		maxPerStack, err := regulationUint32(primaryRow, "maxArrowQuantity")
		if err != nil {
			return schema.Capability[schema.StackRules]{}, err
		}
		if maxPerStack > 1 {
			return enabledRegulationCapability(
				schema.StackRules{MaxPerStack: maxPerStack},
				RegulationTableWeapon,
				primaryRow.RowID,
				"maxArrowQuantity",
				"derived from EquipParamWeapon.maxArrowQuantity",
			), nil
		}
		return disabledRegulationCapability[schema.StackRules](
			RegulationTableWeapon,
			primaryRow.RowID,
			"maxArrowQuantity",
			"EquipParamWeapon.maxArrowQuantity does not allow a quantity stack",
		), nil
	}
	goodsRow, exists, err := context.goodsStorageRow(item, family, primaryRow, hasPrimaryRow)
	if err != nil {
		return schema.Capability[schema.StackRules]{}, err
	}
	if exists {
		maxPerStack, err := regulationUint32(goodsRow, "maxNum")
		if err != nil {
			return schema.Capability[schema.StackRules]{}, err
		}
		if maxPerStack > 1 {
			return enabledRegulationCapability(
				schema.StackRules{MaxPerStack: maxPerStack},
				RegulationTableGoods,
				goodsRow.RowID,
				"maxNum",
				"derived from EquipParamGoods.maxNum",
			), nil
		}
		return disabledRegulationCapability[schema.StackRules](
			RegulationTableGoods,
			goodsRow.RowID,
			"maxNum",
			"EquipParamGoods.maxNum does not allow a quantity stack",
		), nil
	}
	if recordMode(item) == schema.RecordModeSeparateInstances {
		return disabledCapability[schema.StackRules]("item record mode uses separate instances"), nil
	}
	return unknownCapability[schema.StackRules](
		"no authoritative Regulation stack limit is available",
	), nil
}

func buildEquipmentCapability(
	family schema.ItemFamily,
	primaryRow ParameterRow,
	hasPrimaryRow bool,
) (schema.Capability[schema.EquipmentRules], error) {
	var slots []schema.EquipmentSlot
	source := sourceLegacyData
	method := "normalized from the legacy item family"
	var regulationTable RegulationTableName
	var regulationFields string
	switch family {
	case schema.ItemFamilyWeapon:
		if !hasPrimaryRow {
			return unknownCapability[schema.EquipmentRules](
				"primary EquipParamWeapon row is missing",
			), nil
		}
		flags, err := readWeaponEquipmentFlags(primaryRow)
		if err != nil {
			return schema.Capability[schema.EquipmentRules]{}, err
		}
		slots = flags.slots()
		source = sourceIDByRegulationTable[RegulationTableWeapon]
		regulationTable = RegulationTableWeapon
		regulationFields = "rightHandEquipable,leftHandEquipable,bothHandEquipable,arrowSlotEquipable,boltSlotEquipable"
		method = "copied from explicit EquipParamWeapon equipment-slot flags"
	case schema.ItemFamilyArmor:
		if !hasPrimaryRow {
			return unknownCapability[schema.EquipmentRules](
				"primary EquipParamProtector row is missing",
			), nil
		}
		flags, err := readArmorEquipmentFlags(primaryRow)
		if err != nil {
			return schema.Capability[schema.EquipmentRules]{}, err
		}
		slots = flags.slots()
		source = sourceIDByRegulationTable[RegulationTableProtector]
		regulationTable = RegulationTableProtector
		regulationFields = "headEquip,bodyEquip,armEquip,legEquip"
		method = "copied from explicit EquipParamProtector equipment-slot flags"
	case schema.ItemFamilyTalisman:
		slots = []schema.EquipmentSlot{schema.EquipmentSlotTalisman}
	case schema.ItemFamilySpell:
		slots = []schema.EquipmentSlot{schema.EquipmentSlotSpellMemory}
	case schema.ItemFamilySpiritAsh, schema.ItemFamilyGoods:
		if !hasPrimaryRow {
			return unknownCapability[schema.EquipmentRules](
				"primary EquipParamGoods row is missing",
			), nil
		}
		goodsType, err := regulationUint16(primaryRow, "goodsType")
		if err != nil {
			return schema.Capability[schema.EquipmentRules]{}, err
		}
		isEquipable, err := regulationBool(primaryRow, "isEquip")
		if err != nil {
			return schema.Capability[schema.EquipmentRules]{}, err
		}
		source = sourceIDByRegulationTable[RegulationTableGoods]
		regulationTable = RegulationTableGoods
		regulationFields = "goodsType,isEquip"
		switch {
		case family == schema.ItemFamilyGoods && goodsType == 10:
			slots = []schema.EquipmentSlot{schema.EquipmentSlotPhysick}
			method = "classified as a Wondrous Physick tear by EquipParamGoods.goodsType"
		case isEquipable:
			slots = []schema.EquipmentSlot{
				schema.EquipmentSlotQuickItem,
				schema.EquipmentSlotPouch,
			}
			method = "enabled by EquipParamGoods.isEquip"
		default:
			return disabledRegulationCapability[schema.EquipmentRules](
				RegulationTableGoods,
				primaryRow.RowID,
				regulationFields,
				"EquipParamGoods does not allow a supported equipment slot",
			), nil
		}
	case schema.ItemFamilyGesture:
		return disabledCapability[schema.EquipmentRules](
			"gestures are invoked through gesture slots, not item equipment slots",
		), nil
	case schema.ItemFamilyAshOfWar:
		return disabledCapability[schema.EquipmentRules](
			"Ashes of War are mounted through the Ash of War operation, not item equipment slots",
		), nil
	default:
		return unknownCapability[schema.EquipmentRules](
			"legacy database does not define a complete equipment-slot policy for this family",
		), nil
	}
	if len(slots) == 0 {
		if regulationTable != "" {
			return disabledRegulationCapability[schema.EquipmentRules](
				regulationTable,
				primaryRow.RowID,
				regulationFields,
				"regulation row does not enable any supported equipment slot",
			), nil
		}
		return disabledCapabilityFromSource[schema.EquipmentRules](
			source,
			"regulation row does not enable any supported equipment slot",
		), nil
	}
	if regulationTable != "" {
		return enabledRegulationCapability(
			schema.EquipmentRules{AllowedSlots: slots},
			regulationTable,
			primaryRow.RowID,
			regulationFields,
			method,
		), nil
	}
	return enabledCapabilityFromSource(
		schema.EquipmentRules{AllowedSlots: slots},
		source,
		method,
	), nil
}

func enabledCapability[T any](rules T, method string) schema.Capability[T] {
	return enabledCapabilityFromSource(rules, sourceLegacyData, method)
}

func enabledCapabilityFromSource[T any](
	rules T,
	source schema.SourceID,
	method string,
) schema.Capability[T] {
	provenance := schema.Provenance{
		Source: source,
		Method: method,
	}
	return schema.Capability[T]{
		Known:         true,
		Enabled:       true,
		Rules:         &rules,
		Provenance:    provenance,
		RulesEvidence: []schema.Provenance{provenance},
	}
}

func enabledRegulationCapability[T any](
	rules T,
	table RegulationTableName,
	rowID uint32,
	fields string,
	method string,
) schema.Capability[T] {
	capability := enabledCapabilityFromSource(
		rules,
		sourceIDByRegulationTable[table],
		method,
	)
	capability.Provenance.Table = string(table)
	capability.Provenance.Row = decimalRowID(rowID)
	capability.Provenance.Field = fields
	capability.RulesEvidence = []schema.Provenance{capability.Provenance}
	return capability
}

func disabledCapability[T any](method string) schema.Capability[T] {
	return disabledCapabilityFromSource[T](sourceLegacyData, method)
}

func disabledCapabilityFromSource[T any](
	source schema.SourceID,
	method string,
) schema.Capability[T] {
	return schema.Capability[T]{
		Known: true,
		Provenance: schema.Provenance{
			Source: source,
			Method: method,
		},
	}
}

func disabledRegulationCapability[T any](
	table RegulationTableName,
	rowID uint32,
	fields string,
	method string,
) schema.Capability[T] {
	capability := disabledCapabilityFromSource[T](
		sourceIDByRegulationTable[table],
		method,
	)
	capability.Provenance.Table = string(table)
	capability.Provenance.Row = decimalRowID(rowID)
	capability.Provenance.Field = fields
	return capability
}

func unknownCapability[T any](method string) schema.Capability[T] {
	return schema.Capability[T]{
		Provenance: schema.Provenance{
			Source: sourceLegacyUnknown,
			Method: method,
		},
	}
}
