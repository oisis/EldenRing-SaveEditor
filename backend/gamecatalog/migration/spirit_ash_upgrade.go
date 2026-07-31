package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type spiritAshUpgradeAudit struct {
	rules        schema.UpgradeRules
	goodsRows    []ParameterRow
	materialRows []ParameterRow
}

func (context *generationContext) spiritAshUpgrade(
	root ParameterRow,
) (spiritAshUpgradeAudit, error) {
	goodsTable, exists := context.regulation.Table(RegulationTableGoods)
	if !exists {
		return spiritAshUpgradeAudit{}, fmt.Errorf(
			"regulation table %q is not loaded",
			RegulationTableGoods,
		)
	}
	materialTable, exists := context.regulation.Table(RegulationTableMaterialSet)
	if !exists {
		return spiritAshUpgradeAudit{}, fmt.Errorf(
			"regulation table %q is not loaded",
			RegulationTableMaterialSet,
		)
	}

	audit := spiritAshUpgradeAudit{goodsRows: []ParameterRow{root}}
	visited := map[uint32]struct{}{root.RowID: {}}
	var model schema.UpgradeModel
	current := root
	for {
		nextID, err := regulationInt32(current, "reinforceGoodsId")
		if err != nil {
			return spiritAshUpgradeAudit{}, err
		}
		materialSetID, err := regulationInt32(current, "reinforceMaterialId")
		if err != nil {
			return spiritAshUpgradeAudit{}, err
		}
		if nextID < 0 {
			if materialSetID >= 0 {
				return spiritAshUpgradeAudit{}, fmt.Errorf(
					"Spirit Ash row %d terminates with material set %d",
					current.RowID,
					materialSetID,
				)
			}
			break
		}
		if materialSetID < 0 {
			return spiritAshUpgradeAudit{}, fmt.Errorf(
				"Spirit Ash row %d advances without a material set",
				current.RowID,
			)
		}
		materialRow, exists := materialTable.Row(uint32(materialSetID))
		if !exists {
			return spiritAshUpgradeAudit{}, fmt.Errorf(
				"Spirit Ash row %d references missing material set %d",
				current.RowID,
				materialSetID,
			)
		}
		materialGoodsID, err := regulationInt32(materialRow, "materialId01")
		if err != nil {
			return spiritAshUpgradeAudit{}, err
		}
		stepModel, ok := spiritAshUpgradeModel(materialGoodsID)
		if !ok {
			return spiritAshUpgradeAudit{}, fmt.Errorf(
				"material set %d references unsupported Spirit Ash material %d",
				materialSetID,
				materialGoodsID,
			)
		}
		if model != "" && model != stepModel {
			return spiritAshUpgradeAudit{}, fmt.Errorf(
				"Spirit Ash chain %d changes upgrade model from %q to %q",
				root.RowID,
				model,
				stepModel,
			)
		}
		model = stepModel
		nextRow, exists := goodsTable.Row(uint32(nextID))
		if !exists {
			return spiritAshUpgradeAudit{}, fmt.Errorf(
				"Spirit Ash row %d references missing upgrade row %d",
				current.RowID,
				nextID,
			)
		}
		if _, duplicate := visited[nextRow.RowID]; duplicate {
			return spiritAshUpgradeAudit{}, fmt.Errorf(
				"Spirit Ash chain %d contains a cycle at row %d",
				root.RowID,
				nextRow.RowID,
			)
		}
		visited[nextRow.RowID] = struct{}{}
		audit.materialRows = append(audit.materialRows, materialRow)
		audit.goodsRows = append(audit.goodsRows, nextRow)
		current = nextRow
	}
	if len(audit.goodsRows) != 11 || len(audit.materialRows) != 10 {
		return spiritAshUpgradeAudit{}, fmt.Errorf(
			"Spirit Ash chain %d has %d levels and %d material steps, want 11 and 10",
			root.RowID,
			len(audit.goodsRows),
			len(audit.materialRows),
		)
	}
	audit.rules = schema.UpgradeRules{
		Model:    model,
		MaxLevel: uint8(len(audit.goodsRows) - 1),
	}
	return audit, nil
}

func spiritAshUpgradeModel(materialGoodsID int32) (schema.UpgradeModel, bool) {
	switch {
	case materialGoodsID >= 10900 && materialGoodsID <= 10909:
		return schema.UpgradeModelGraveGlovewort, true
	case materialGoodsID >= 10910 && materialGoodsID <= 10919:
		return schema.UpgradeModelGhostGlovewort, true
	default:
		return "", false
	}
}

func spiritAshUpgradeEvidence(audit spiritAshUpgradeAudit) []schema.Provenance {
	evidence := make(
		[]schema.Provenance,
		0,
		len(audit.goodsRows)+len(audit.materialRows),
	)
	for _, row := range audit.goodsRows {
		evidence = append(evidence, schema.Provenance{
			Source: sourceIDByRegulationTable[RegulationTableGoods],
			Method: "used to validate the complete Spirit Ash upgrade chain",
			Table:  string(RegulationTableGoods),
			Row:    decimalRowID(row.RowID),
			Field:  "reinforceMaterialId,reinforceGoodsId",
		})
	}
	for _, row := range audit.materialRows {
		evidence = append(evidence, schema.Provenance{
			Source: sourceIDByRegulationTable[RegulationTableMaterialSet],
			Method: "used to resolve the Spirit Ash upgrade material model",
			Table:  string(RegulationTableMaterialSet),
			Row:    decimalRowID(row.RowID),
			Field:  "materialId01",
		})
	}
	return evidence
}
