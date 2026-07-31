package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) buildAliases(
	canonicalID uint32,
) ([]schema.ItemAlias, error) {
	values := context.aliasesByItem[canonicalID]
	result := make([]schema.ItemAlias, len(values))
	for index, value := range values {
		rowID := value.AliasID & 0x0FFFFFFF
		lookup, exists, err := context.regulation.LookupFamilyRow(
			RegulationFamilyGoods,
			RegulationTableRolePrimary,
			rowID,
		)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf(
				"technical alias 0x%08X has no EquipParamGoods row %d",
				value.AliasID,
				rowID,
			)
		}
		result[index] = schema.ItemAlias{
			GameID: knownRegulationFact(
				value.AliasID,
				RegulationTableGoods,
				"Row ID plus goods item prefix",
				rowID,
			),
			SourceRecords: []schema.ParameterRecord{parameterRecord(lookup)},
		}
	}
	return result, nil
}
