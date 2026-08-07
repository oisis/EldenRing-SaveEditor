package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) buildVariants(
	values []legacyVariantSeed,
	canonical builtDocumentData,
) ([]schema.ItemVariant, error) {
	result := make([]schema.ItemVariant, len(values))
	for index, value := range values {
		records, err := context.sourceRecordsForItem(value.Item)
		if err != nil {
			return nil, err
		}
		identity, err := primaryRegulationForLegacyItem(value.Item)
		if err != nil {
			return nil, err
		}
		lookup, exists, err := context.regulation.LookupFamilyRow(
			identity.Family,
			RegulationTableRolePrimary,
			identity.RowID,
		)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf(
				"variant item 0x%08X has no primary regulation row %d",
				value.Item.ID,
				identity.RowID,
			)
		}
		family, table, err := itemFamily(value.Item)
		if err != nil {
			return nil, err
		}
		var capabilitiesOverride *schema.ItemCapabilities
		if family == schema.ItemFamilySpiritAsh {
			capabilitiesOverride = &canonical.Capabilities
			rootRowID := canonical.SpiritAsh.SourceRowID.Value
			rootTable, tableExists := context.regulation.Table(RegulationTableGoods)
			if !tableExists {
				return nil, fmt.Errorf(
					"regulation table %q is not loaded",
					RegulationTableGoods,
				)
			}
			rootRow, rowExists := rootTable.Row(rootRowID)
			if !rowExists {
				return nil, fmt.Errorf(
					"canonical Spirit Ash row %d is missing",
					rootRowID,
				)
			}
			upgradeRecords, recordsErr := context.spiritAshUpgradeSourceRecords(rootRow)
			if recordsErr != nil {
				return nil, recordsErr
			}
			records, recordsErr = mergeParameterRecords(records, upgradeRecords)
			if recordsErr != nil {
				return nil, recordsErr
			}
		}
		data, err := context.buildDocumentDataWithCapabilities(
			value.Item,
			family,
			lookup.Row,
			true,
			capabilitiesOverride,
			true,
		)
		if err != nil {
			return nil, err
		}
		variantData, err := fullVariantData(family, data)
		if err != nil {
			return nil, fmt.Errorf("variant item 0x%08X: %w", value.Item.ID, err)
		}
		variant := schema.ItemVariant{
			GameID:        knownRegulationFact(value.Item.ID, table, "Row ID plus item-family prefix", identity.RowID),
			Kind:          knownRegulationDerivedFact(schema.ItemVariantKind(value.Kind), table, "derived from explicit regulation relationship", identity.RowID, variantRelationshipField(value.Kind)),
			SourceRowID:   knownRegulationFact(identity.RowID, table, "Row ID", identity.RowID),
			Data:          variantData,
			SourceRecords: records,
		}
		switch value.Kind {
		case legacyVariantAffinity:
			variant.Affinity = knownRegulationDerivedFact(
				schema.Affinity(value.Affinity),
				table,
				"derived from EquipParamWeapon originEquipWep offset",
				identity.RowID,
				"originEquipWep",
			)
		case legacyVariantUpgrade:
			if family == schema.ItemFamilySpiritAsh {
				variant.Affinity = notApplicableCatalogFact[schema.Affinity](
					"spirit ash upgrade variants do not have an affinity",
				)
			}
			variant.UpgradeLevel = knownRegulationDerivedFact(
				value.UpgradeLevel,
				table,
				"derived from the explicit EquipParamGoods reinforceGoodsId chain",
				identity.RowID,
				"reinforceGoodsId",
			)
		default:
			return nil, fmt.Errorf("unsupported legacy variant kind %q", value.Kind)
		}
		variant.SourceRecords = enrichParameterRecordFields(variant.SourceRecords, variant)
		result[index] = variant
	}
	return result, nil
}

func fullVariantData(
	family schema.ItemFamily,
	variant builtDocumentData,
) (schema.VariantDocumentData, error) {
	result := schema.VariantDocumentData{
		Family:                  itemFamilyFact(family, primaryTableForFamily(family), variantPrimaryRowID(variant)),
		Category:                variant.Category,
		Subcategory:             variant.Subcategory,
		Presentation:            variant.Presentation,
		Storage:                 variant.Storage,
		Capabilities:            variant.Capabilities,
		Safety:                  variant.Safety,
		Acquisition:             variant.Acquisition,
		Modifiers:               variant.Modifiers,
		Links:                   variant.Links,
		Unlocks:                 variant.Unlocks,
		RelatedTechnicalRecords: variant.RelatedTechnicalRecords,
	}
	switch family {
	case schema.ItemFamilyWeapon:
		if variant.Weapon == nil {
			return schema.VariantDocumentData{}, fmt.Errorf("weapon family data is missing")
		}
		result.Weapon = variant.Weapon
		return result, nil
	case schema.ItemFamilySpiritAsh:
		if variant.SpiritAsh == nil {
			return schema.VariantDocumentData{}, fmt.Errorf("spirit ash family data is missing")
		}
		result.SpiritAsh = variant.SpiritAsh
		return result, nil
	default:
		return schema.VariantDocumentData{}, fmt.Errorf(
			"family %q does not support variants",
			family,
		)
	}
}

func variantPrimaryRowID(variant builtDocumentData) uint32 {
	if variant.Weapon != nil && variant.Weapon.SourceRowID.Known {
		return variant.Weapon.SourceRowID.Value
	}
	if variant.SpiritAsh != nil && variant.SpiritAsh.SourceRowID.Known {
		return variant.SpiritAsh.SourceRowID.Value
	}
	return 0
}

func primaryTableForFamily(family schema.ItemFamily) RegulationTableName {
	switch family {
	case schema.ItemFamilyWeapon:
		return RegulationTableWeapon
	case schema.ItemFamilySpiritAsh:
		return RegulationTableGoods
	default:
		return ""
	}
}

func variantRelationshipField(kind legacyVariantKind) string {
	if kind == legacyVariantUpgrade {
		return "reinforceGoodsId"
	}
	return "originEquipWep"
}
