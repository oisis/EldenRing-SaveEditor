package migration

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

var passiveEffectFields = map[string]string{
	"Poison":       "poizonAttackPower",
	"Scarlet Rot":  "diseaseAttackPower",
	"Blood Loss":   "bloodAttackPower",
	"Death Blight": "curseAttackPower",
	"Frost":        "freezeAttackPower",
	"Sleep":        "sleepAttackPower",
	"Madness":      "madnessAttackPower",
}

var weaponPassiveEffectSlots = []struct {
	source string
	kind   string
}{
	{"spEffectBehaviorId0", "on_hit"},
	{"spEffectBehaviorId1", "on_hit"},
	{"spEffectBehaviorId2", "on_hit"},
	{"residentSpEffectId", "resident"},
	{"residentSpEffectId1", "resident"},
	{"residentSpEffectId2", "resident"},
}

var orderedPassiveEffectFields = []struct {
	label string
	field string
}{
	{"Poison", "poizonAttackPower"},
	{"Scarlet Rot", "diseaseAttackPower"},
	{"Blood Loss", "bloodAttackPower"},
	{"Death Blight", "curseAttackPower"},
	{"Frost", "freezeAttackPower"},
	{"Sleep", "sleepAttackPower"},
	{"Madness", "madnessAttackPower"},
}

func (context *generationContext) regulationWeaponPassiveEffects(
	weaponRow ParameterRow,
) ([]passiveEffectSeed, error) {
	table, exists := context.regulation.Table(RegulationTableSpEffect)
	if !exists {
		return nil, fmt.Errorf(
			"regulation table %q is not loaded",
			RegulationTableSpEffect,
		)
	}
	result := make([]passiveEffectSeed, 0, len(weaponPassiveEffectSlots))
	for _, slot := range weaponPassiveEffectSlots {
		id, err := regulationInt32(weaponRow, slot.source)
		if err != nil {
			return nil, err
		}
		if id <= 0 {
			continue
		}
		effect := passiveEffectSeed{
			Kind:       slot.kind,
			Source:     slot.source,
			SpEffectID: id,
			Label:      "Unknown " + strings.ReplaceAll(slot.kind, "_", "-") + " effect",
		}
		if slot.kind == "resident" {
			effect.Label = "Unknown resident effect"
		}
		row, found := table.Row(uint32(id))
		if found && slot.kind == "on_hit" {
			var matched bool
			for _, candidate := range orderedPassiveEffectFields {
				value, readErr := regulationInt32(row, candidate.field)
				if readErr != nil {
					return nil, readErr
				}
				if value <= 0 {
					continue
				}
				if matched {
					matched = false
					break
				}
				matched = true
				effect.Label = candidate.label
				effect.Value = value
			}
			effect.Known = matched
			if !matched {
				effect.Label = "Unknown on-hit effect"
				effect.Value = 0
			}
		}
		result = append(result, effect)
	}
	return result, nil
}

func (context *generationContext) passiveEffectParameterRecords(
	effects []passiveEffectSeed,
) ([]schema.ParameterRecord, error) {
	table, exists := context.regulation.Table(RegulationTableSpEffect)
	if !exists {
		return nil, fmt.Errorf(
			"regulation table %q is not loaded",
			RegulationTableSpEffect,
		)
	}

	rows := make(map[uint32]ParameterRow)
	for _, effect := range effects {
		if effect.SpEffectID <= 0 {
			return nil, fmt.Errorf("invalid passive SpEffect ID %d", effect.SpEffectID)
		}
		rowID := uint32(effect.SpEffectID)
		row, found := table.Row(rowID)
		if !found {
			if effect.Known {
				return nil, fmt.Errorf(
					"known passive effect %d is missing from SpEffectParam",
					effect.SpEffectID,
				)
			}
			continue
		}
		if err := verifyKnownPassiveEffect(effect, row); err != nil {
			return nil, err
		}
		rows[rowID] = row
	}

	rowIDs := make([]uint32, 0, len(rows))
	for rowID := range rows {
		rowIDs = append(rowIDs, rowID)
	}
	sort.Slice(rowIDs, func(i, j int) bool {
		return rowIDs[i] < rowIDs[j]
	})

	records := make([]schema.ParameterRecord, 0, len(rowIDs))
	for _, rowID := range rowIDs {
		records = append(records, parameterRecord(RegulationRowLookup{
			Table:    RegulationTableSpEffect,
			Source:   table.Source(),
			RawRowID: rowID,
			Row:      rows[rowID],
		}))
	}
	return records, nil
}

func verifyKnownPassiveEffect(
	effect passiveEffectSeed,
	row ParameterRow,
) error {
	if !effect.Known || effect.Kind != "on_hit" {
		return nil
	}
	field, exists := passiveEffectFields[effect.Label]
	if !exists {
		return fmt.Errorf(
			"known on-hit SpEffect %d has unsupported label %q",
			effect.SpEffectID,
			effect.Label,
		)
	}
	value, err := regulationInt32(row, field)
	if err != nil {
		return fmt.Errorf("SpEffectParam row %d: %w", row.RowID, err)
	}
	if value != effect.Value {
		return fmt.Errorf(
			"SpEffectParam row %d %s = %d, legacy passive effect value = %d",
			row.RowID,
			field,
			value,
			effect.Value,
		)
	}
	return nil
}

func (context *generationContext) buildWeaponPassiveEffects(
	weaponRow ParameterRow,
	effects []passiveEffectSeed,
) ([]schema.WeaponPassiveEffectData, error) {
	result := make([]schema.WeaponPassiveEffectData, len(effects))
	table, _ := context.regulation.Table(RegulationTableSpEffect)
	for index, effect := range effects {
		rawID, exists := weaponRow.Field(effect.Source)
		if !exists {
			return nil, fmt.Errorf(
				"EquipParamWeapon row %d has no %s field",
				weaponRow.RowID,
				effect.Source,
			)
		}
		sourceID, err := strconv.ParseInt(rawID, 10, 32)
		if err != nil || int32(sourceID) != effect.SpEffectID {
			return nil, fmt.Errorf(
				"EquipParamWeapon row %d %s=%q differs from passive SpEffect ID %d",
				weaponRow.RowID,
				effect.Source,
				rawID,
				effect.SpEffectID,
			)
		}
		data := schema.WeaponPassiveEffectData{
			Kind: knownRegulationDerivedFact(
				effect.Kind,
				RegulationTableWeapon,
				"classified from "+effect.Source,
				weaponRow.RowID,
				effect.Source,
			),
			Source: knownRegulationFact(
				effect.Source,
				RegulationTableWeapon,
				effect.Source+" field name",
				weaponRow.RowID,
			),
			SpEffectID: knownRegulationFact(
				effect.SpEffectID,
				RegulationTableWeapon,
				effect.Source,
				weaponRow.RowID,
			),
			Label: knownLegacyFact(
				effect.Label,
				"copied from the legacy passive-effect interpretation",
			),
			Known: knownLegacyFact(
				effect.Known,
				"copied from the legacy passive-effect resolution status",
			),
			Value: unknownCatalogFact[int32](
				"passive effect has no verified numeric buildup value",
			),
		}
		if effect.Known && effect.Kind == "on_hit" {
			field := passiveEffectFields[effect.Label]
			spEffectRow, exists := table.Row(uint32(effect.SpEffectID))
			if !exists {
				return nil, fmt.Errorf(
					"known passive effect %d is missing from SpEffectParam",
					effect.SpEffectID,
				)
			}
			data.Value = knownRegulationFact(
				effect.Value,
				RegulationTableSpEffect,
				field,
				spEffectRow.RowID,
			)
			data.Known = knownRegulationFact(
				true,
				RegulationTableSpEffect,
				"verified nonzero "+field,
				spEffectRow.RowID,
			)
		}
		result[index] = data
	}
	return result, nil
}
