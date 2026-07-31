package migration

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) sourceRecordsForItem(
	item seed,
) ([]schema.ParameterRecord, error) {
	identity, err := primaryRegulationForLegacyItem(item)
	if err != nil {
		return nil, err
	}
	primary, exists, err := context.regulation.LookupFamilyRow(
		identity.Family,
		RegulationTableRolePrimary,
		identity.RowID,
	)
	if err != nil {
		return nil, err
	}
	if !exists {
		if item.ID == 0x40002341 || item.ID == 0x4000234E || item.ID == 0x40002354 {
			return context.gestureParameterRecords(item.ID), nil
		}
		return nil, fmt.Errorf(
			"required primary regulation row %s(%d) is missing",
			identity.Family,
			identity.RowID,
		)
	}

	records := []schema.ParameterRecord{parameterRecord(primary)}
	switch identity.Family {
	case RegulationFamilyWeapon, RegulationFamilyProtector:
		if identity.Family == RegulationFamilyWeapon {
			swordArts, appendErr := context.swordArtsParameterRecord(primary.Row)
			if appendErr != nil {
				return nil, appendErr
			}
			if swordArts != nil {
				records = append(records, *swordArts)
			}
			effects := item.WeaponStats.PassiveEffects
			if item.RegulationOnlyVariant {
				effects, appendErr = context.regulationWeaponPassiveEffects(primary.Row)
				if appendErr != nil {
					return nil, appendErr
				}
			}
			passiveEffects, appendErr := context.passiveEffectParameterRecords(effects)
			if appendErr != nil {
				return nil, appendErr
			}
			records = append(records, passiveEffects...)
		}
		rawReinforceID, ok := primary.Row.Field("reinforceTypeId")
		if !ok {
			return nil, fmt.Errorf("%s row %d has no reinforceTypeId", primary.Table, primary.RawRowID)
		}
		reinforceID, parseErr := strconv.ParseInt(rawReinforceID, 10, 32)
		if parseErr != nil {
			return nil, fmt.Errorf("%s row %d reinforceTypeId: %w", primary.Table, primary.RawRowID, parseErr)
		}
		if reinforceID >= 0 {
			supporting, found, lookupErr := context.regulation.LookupFamilyRow(
				identity.Family,
				RegulationTableRoleSupporting,
				uint32(reinforceID),
			)
			if lookupErr != nil {
				return nil, lookupErr
			}
			if !found {
				return nil, fmt.Errorf(
					"%s row %d references missing reinforce row %d",
					primary.Table,
					primary.RawRowID,
					reinforceID,
				)
			}
			records = append(records, parameterRecord(supporting))
		}
	case RegulationFamilySpell:
		goods, found, lookupErr := context.regulation.LookupFamilyRow(
			RegulationFamilyGoods,
			RegulationTableRolePrimary,
			identity.RowID,
		)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if !found {
			return nil, fmt.Errorf(
				"Magic row %d has no corresponding EquipParamGoods row",
				identity.RowID,
			)
		}
		records = append(records, parameterRecord(goods))
	case RegulationFamilyGoods:
		if item.Category == "gestures" {
			records = append(records, context.gestureParameterRecords(item.ID)...)
		}
		if item.Links.AboutTutorialID != nil {
			tutorial, lookupErr := context.requiredParameterRecord(
				RegulationTableTutorial,
				*item.Links.AboutTutorialID,
				"AboutTutorialID",
			)
			if lookupErr != nil {
				return nil, lookupErr
			}
			records = append(records, tutorial)
		}
	case RegulationFamilyAshOfWar:
		swordArts, appendErr := context.swordArtsParameterRecord(primary.Row)
		if appendErr != nil {
			return nil, appendErr
		}
		if swordArts != nil {
			records = append(records, *swordArts)
		}
		records = append(records, schema.ParameterRecord{
			Table: string(RegulationTableGem),
			RowID: int64(primary.RawRowID),
			Fields: []schema.ParameterField{{
				Name: "canMountWep[0:44]",
			}},
			Provenance: schema.Provenance{
				Source: sourceRegulationEquipParamGemRaw,
				Method: "referenced full-width Regulation parameter field",
				Table:  string(RegulationTableGem),
				Row:    decimalRowID(primary.RawRowID),
				Field:  "canMountWep[0:44]",
			},
		})
	}
	return records, nil
}

func (context *generationContext) swordArtsParameterRecord(
	owner ParameterRow,
) (*schema.ParameterRecord, error) {
	rawID, exists := owner.Field("swordArtsParamId")
	if !exists {
		return nil, fmt.Errorf("row %d has no swordArtsParamId", owner.RowID)
	}
	id, err := strconv.ParseInt(rawID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf(
			"row %d swordArtsParamId %q is invalid: %w",
			owner.RowID,
			rawID,
			err,
		)
	}
	if id < 0 {
		return nil, nil
	}
	record, err := context.requiredParameterRecord(
		RegulationTableSwordArts,
		uint32(id),
		"swordArtsParamId",
	)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (context *generationContext) requiredParameterRecord(
	tableName RegulationTableName,
	rowID uint32,
	relation string,
) (schema.ParameterRecord, error) {
	table, exists := context.regulation.Table(tableName)
	if !exists {
		return schema.ParameterRecord{}, fmt.Errorf(
			"regulation table %q is not loaded",
			tableName,
		)
	}
	row, exists := table.Row(rowID)
	if !exists {
		return schema.ParameterRecord{}, fmt.Errorf(
			"%s references missing %s row %d",
			relation,
			tableName,
			rowID,
		)
	}
	return parameterRecord(RegulationRowLookup{
		Table:    tableName,
		Source:   table.Source(),
		RawRowID: rowID,
		Row:      row,
	}), nil
}

func (context *generationContext) gestureParameterRecords(
	fullItemID uint32,
) []schema.ParameterRecord {
	rows := context.gestureRows[fullItemID&0x0FFFFFFF]
	records := make([]schema.ParameterRecord, len(rows))
	table, _ := context.regulation.Table(RegulationTableGesture)
	for index, row := range rows {
		records[index] = parameterRecord(RegulationRowLookup{
			Table:    RegulationTableGesture,
			Source:   table.Source(),
			RawRowID: row.RowID,
			Row:      row,
		})
	}
	return records
}

func parameterRecord(lookup RegulationRowLookup) schema.ParameterRecord {
	sourceID := sourceIDByRegulationTable[lookup.Table]
	provenance := schema.Provenance{
		Source: sourceID,
		Method: "referenced Regulation parameter row",
		Table:  string(lookup.Table),
		Row:    decimalRowID(lookup.RawRowID),
		Field:  "Row ID",
	}
	return schema.ParameterRecord{
		Table:      string(lookup.Table),
		RowID:      int64(lookup.RawRowID),
		Fields:     []schema.ParameterField{{Name: "Row ID"}},
		Provenance: provenance,
	}
}

func parameterRecordsForRows(
	tableName RegulationTableName,
	rows []ParameterRow,
	regulation *RegulationData,
	fields ...string,
) []schema.ParameterRecord {
	table, _ := regulation.Table(tableName)
	records := make([]schema.ParameterRecord, len(rows))
	for index, row := range rows {
		records[index] = parameterRecord(RegulationRowLookup{
			Table:    tableName,
			Source:   table.Source(),
			RawRowID: row.RowID,
			Row:      row,
		})
		for _, field := range fields {
			records[index].Fields = append(
				records[index].Fields,
				schema.ParameterField{Name: field},
			)
		}
	}
	return records
}

func enrichParameterRecordFields(
	records []schema.ParameterRecord,
	values ...any,
) []schema.ParameterRecord {
	fieldsByRecord := make(map[string]map[string]struct{})
	for _, value := range values {
		collectProvenanceFields(reflect.ValueOf(value), fieldsByRecord)
	}
	enriched := append([]schema.ParameterRecord(nil), records...)
	for index := range enriched {
		key := parameterRecordKey(
			enriched[index].Provenance.Source,
			enriched[index].Table,
			strconv.FormatInt(enriched[index].RowID, 10),
		)
		names := fieldsByRecord[key]
		if names == nil {
			names = make(map[string]struct{})
		}
		for _, field := range enriched[index].Fields {
			names[field.Name] = struct{}{}
		}
		names["Row ID"] = struct{}{}
		sorted := make([]string, 0, len(names))
		for name := range names {
			sorted = append(sorted, name)
		}
		sort.Strings(sorted)
		enriched[index].Fields = make([]schema.ParameterField, len(sorted))
		for fieldIndex, name := range sorted {
			enriched[index].Fields[fieldIndex] = schema.ParameterField{Name: name}
		}
	}
	return enriched
}

func collectProvenanceFields(
	value reflect.Value,
	fieldsByRecord map[string]map[string]struct{},
) {
	if !value.IsValid() {
		return
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if value.Type() == reflect.TypeOf(schema.Provenance{}) {
		provenance := value.Interface().(schema.Provenance)
		if provenance.Table == "" || provenance.Row == "" || provenance.Field == "" {
			return
		}
		key := parameterRecordKey(provenance.Source, provenance.Table, provenance.Row)
		if fieldsByRecord[key] == nil {
			fieldsByRecord[key] = make(map[string]struct{})
		}
		for _, field := range strings.Split(provenance.Field, ",") {
			field = strings.TrimSpace(field)
			if field != "" {
				fieldsByRecord[key][field] = struct{}{}
			}
		}
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			if value.Type().Field(index).PkgPath == "" {
				collectProvenanceFields(value.Field(index), fieldsByRecord)
			}
		}
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			collectProvenanceFields(value.Index(index), fieldsByRecord)
		}
	}
}

func parameterRecordKey(source schema.SourceID, table, row string) string {
	return string(source) + "\x00" + table + "\x00" + row
}

func decimalRowID(rowID uint32) string {
	return strconv.FormatUint(uint64(rowID), 10)
}
