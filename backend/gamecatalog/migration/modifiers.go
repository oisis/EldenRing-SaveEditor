package migration

import (
	"fmt"
	"math"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type regulationEquipLoadModifier struct {
	row            ParameterRow
	enduranceBonus int32
	equipLoadRate  float64
	referenceField string
}

func (context *generationContext) buildModifiers(
	item seed,
	family schema.ItemFamily,
	primaryRow ParameterRow,
	hasPrimaryRow bool,
) (schema.ItemModifiers, error) {
	if item.EquipLoad == nil {
		return schema.ItemModifiers{}, nil
	}
	if !hasPrimaryRow {
		return schema.ItemModifiers{}, fmt.Errorf(
			"item 0x%08X has an equip-load modifier but no primary Regulation row",
			item.ID,
		)
	}
	modifier, err := context.readRegulationEquipLoadModifier(
		item.ID,
		family,
		primaryRow,
	)
	if err != nil {
		return schema.ItemModifiers{}, err
	}
	enduranceSFV, err := saveForgeConsensusValue(
		"enduranceBonus",
		modifier.enduranceBonus,
		legacyCandidate[int32]{
			available: true,
			value:     item.EquipLoad.EnduranceBonus,
			source:    "EquipLoadModifiers.EnduranceBonus",
		},
	)
	if err != nil {
		return schema.ItemModifiers{}, err
	}
	rateSFV, err := saveForgeConsensusValue(
		"equipLoadRate",
		modifier.equipLoadRate,
		legacyCandidate[float64]{
			available: true,
			value:     item.EquipLoad.EquipLoadRate,
			source:    "EquipLoadModifiers.EquipLoadRate",
		},
	)
	if err != nil {
		return schema.ItemModifiers{}, err
	}
	return schema.ItemModifiers{
		EquipLoad: &schema.EquipLoadModifier{
			EnduranceBonus: knownRegulationDerivedFact(
				modifier.enduranceBonus,
				RegulationTableSpEffect,
				"read the permanent equip-load endurance bonus from the SpEffect referenced by "+modifier.referenceField,
				modifier.row.RowID,
				"addEndureStatus",
			),
			EnduranceBonusSFV: enduranceSFV,
			EquipLoadRate: knownRegulationDerivedFact(
				modifier.equipLoadRate,
				RegulationTableSpEffect,
				"converted the permanent equip-load multiplier to an additive rate from the SpEffect referenced by "+modifier.referenceField,
				modifier.row.RowID,
				"equipWeightChangeRate",
			),
			EquipLoadRateSFV: rateSFV,
		},
	}, nil
}

func (context *generationContext) readRegulationEquipLoadModifier(
	itemID uint32,
	family schema.ItemFamily,
	primaryRow ParameterRow,
) (regulationEquipLoadModifier, error) {
	var referenceFields []string
	switch family {
	case schema.ItemFamilyTalisman:
		referenceFields = []string{
			"refId",
			"residentSpEffectId1",
			"residentSpEffectId2",
			"residentSpEffectId3",
			"residentSpEffectId4",
		}
	case schema.ItemFamilyArmor:
		referenceFields = []string{
			"residentSpEffectId",
			"residentSpEffectId2",
			"residentSpEffectId3",
		}
	default:
		return regulationEquipLoadModifier{}, fmt.Errorf(
			"item 0x%08X has an equip-load modifier in unsupported family %q",
			itemID,
			family,
		)
	}

	table, exists := context.regulation.Table(RegulationTableSpEffect)
	if !exists {
		return regulationEquipLoadModifier{}, fmt.Errorf(
			"regulation table %q is not loaded",
			RegulationTableSpEffect,
		)
	}
	candidates := make([]regulationEquipLoadModifier, 0, 1)
	for _, field := range referenceFields {
		effectID, err := regulationInt32(primaryRow, field)
		if err != nil {
			return regulationEquipLoadModifier{}, err
		}
		if effectID <= 0 {
			continue
		}
		effectRow, found := table.Row(uint32(effectID))
		if !found {
			return regulationEquipLoadModifier{}, fmt.Errorf(
				"item 0x%08X %s references missing SpEffectParam row %d",
				itemID,
				field,
				effectID,
			)
		}
		enduranceBonus, err := regulationInt32(effectRow, "addEndureStatus")
		if err != nil {
			return regulationEquipLoadModifier{}, err
		}
		multiplier, err := regulationFloat64(effectRow, "equipWeightChangeRate")
		if err != nil {
			return regulationEquipLoadModifier{}, err
		}
		if enduranceBonus == 0 && math.Abs(multiplier-1) < 0.0000001 {
			continue
		}
		candidates = append(candidates, regulationEquipLoadModifier{
			row:            effectRow,
			enduranceBonus: enduranceBonus,
			equipLoadRate:  math.Round((multiplier-1)*1000) / 1000,
			referenceField: field,
		})
	}
	if len(candidates) != 1 {
		return regulationEquipLoadModifier{}, fmt.Errorf(
			"item 0x%08X has %d permanent equip-load SpEffect candidates, want exactly one",
			itemID,
			len(candidates),
		)
	}
	return candidates[0], nil
}
