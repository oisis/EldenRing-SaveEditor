package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) spiritAshUpgradeSourceRecords(
	root ParameterRow,
) ([]schema.ParameterRecord, error) {
	upgrade, err := context.spiritAshUpgrade(root)
	if err != nil {
		return nil, err
	}
	records := parameterRecordsForRows(
		RegulationTableGoods,
		upgrade.goodsRows,
		context.regulation,
		"reinforceMaterialId",
		"reinforceGoodsId",
	)
	records = append(
		records,
		parameterRecordsForRows(
			RegulationTableMaterialSet,
			upgrade.materialRows,
			context.regulation,
			"materialId01",
		)...,
	)
	return records, nil
}

func mergeParameterRecords(
	base []schema.ParameterRecord,
	additional []schema.ParameterRecord,
) ([]schema.ParameterRecord, error) {
	result := append([]schema.ParameterRecord(nil), base...)
	indexByKey := make(map[string]int, len(result))
	for index, record := range result {
		indexByKey[parameterRecordMergeKey(record)] = index
	}
	for _, record := range additional {
		key := parameterRecordMergeKey(record)
		if index, exists := indexByKey[key]; exists {
			if result[index].Provenance.Source != record.Provenance.Source {
				return nil, fmt.Errorf(
					"parameter record %s has conflicting sources %q and %q",
					key,
					result[index].Provenance.Source,
					record.Provenance.Source,
				)
			}
			result[index].Fields = append(result[index].Fields, record.Fields...)
			continue
		}
		indexByKey[key] = len(result)
		result = append(result, record)
	}
	return result, nil
}

func parameterRecordMergeKey(record schema.ParameterRecord) string {
	return fmt.Sprintf("%s:%d", record.Table, record.RowID)
}
