package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) buildRelatedTechnicalRecords(
	family schema.ItemFamily,
	row ParameterRow,
	hasPrimaryRow bool,
) ([]schema.RelatedTechnicalRecord, error) {
	if family != schema.ItemFamilyGoods || !hasPrimaryRow {
		return nil, nil
	}
	rawTarget, exists := row.Field("appearanceReplaceItemId")
	if !exists {
		return nil, nil
	}
	targetRowID, err := regulationInt32(row, "appearanceReplaceItemId")
	if err != nil {
		return nil, err
	}
	if targetRowID <= 0 {
		return nil, nil
	}
	targetID := legacyGoodsItemPrefix | uint32(targetRowID)
	legacy, exists := context.technicalItems[targetID]
	if !exists {
		return nil, fmt.Errorf(
			"EquipParamGoods row %d references appearance record %s (0x%08X) without complete legacy description and limits",
			row.RowID,
			rawTarget,
			targetID,
		)
	}
	target, exists, err := context.regulation.LookupFamilyRow(
		RegulationFamilyGoods,
		RegulationTableRolePrimary,
		uint32(targetRowID),
	)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf(
			"EquipParamGoods row %d references missing appearance row %d",
			row.RowID,
			targetRowID,
		)
	}
	owner, err := context.requiredParameterRecord(
		RegulationTableGoods,
		row.RowID,
		"appearanceReplaceItemId",
	)
	if err != nil {
		return nil, err
	}
	record := schema.RelatedTechnicalRecord{
		Kind: knownRegulationDerivedFact(
			schema.TechnicalRecordAppearanceState,
			RegulationTableGoods,
			"derived from positive appearanceReplaceItemId",
			row.RowID,
			"appearanceReplaceItemId",
		),
		GameID: knownRegulationFact(
			targetID,
			RegulationTableGoods,
			"appearanceReplaceItemId plus item-family prefix",
			row.RowID,
		),
		Description: buildDescriptionRecord(legacy.Description),
		GameMaxInventory: knownLegacyFact(
			legacy.GameLimits.MaxInventory,
			"copied from protected legacy GameLimitsByItemID.MaxInventory",
		),
		GameMaxStorage: knownLegacyFact(
			legacy.GameLimits.MaxStorage,
			"copied from protected legacy GameLimitsByItemID.MaxStorage",
		),
		SourceRecords: []schema.ParameterRecord{owner, parameterRecord(target)},
	}
	record.SourceRecords = enrichParameterRecordFields(record.SourceRecords, record)
	return []schema.RelatedTechnicalRecord{record}, nil
}
