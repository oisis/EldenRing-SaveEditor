package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func buildSpiritAshData(item seed, row ParameterRow) (*schema.SpiritAshData, error) {
	iconID, err := regulationUint32(row, "iconId")
	if err != nil {
		return nil, err
	}
	reinforceGoodsID, err := regulationInt32(row, "reinforceGoodsId")
	if err != nil {
		return nil, err
	}
	reinforceMaterialID, err := regulationInt32(row, "reinforceMaterialId")
	if err != nil {
		return nil, err
	}
	reinforcePrice, err := regulationUint32(row, "reinforcePrice")
	if err != nil {
		return nil, err
	}
	sortID, err := regulationUint32(row, "sortId")
	if err != nil {
		return nil, err
	}
	sortGroupID, err := regulationUint8(row, "sortGroupId")
	if err != nil {
		return nil, err
	}
	data := &schema.SpiritAshData{
		SourceRowID:         knownRegulationFact(row.RowID, RegulationTableGoods, "Row ID", row.RowID),
		IconID:              knownRegulationFact(iconID, RegulationTableGoods, "iconId", row.RowID),
		SortID:              knownRegulationFact(sortID, RegulationTableGoods, "sortId", row.RowID),
		SortGroupID:         knownRegulationFact(sortGroupID, RegulationTableGoods, "sortGroupId", row.RowID),
		ReinforceGoodsID:    knownRegulationFact(reinforceGoodsID, RegulationTableGoods, "reinforceGoodsId", row.RowID),
		ReinforceMaterialID: knownRegulationFact(reinforceMaterialID, RegulationTableGoods, "reinforceMaterialId", row.RowID),
		ReinforcePrice:      knownRegulationFact(reinforcePrice, RegulationTableGoods, "reinforcePrice", row.RowID),
	}
	if item.HasLegacyItem && item.SortKey != nil {
		data.SortIDSFV = saveForgeValue(
			true,
			item.SortKey.SortID,
			sortID,
			"preserved conflicting SaveForge value from ItemSortKeys.SortId",
		)
		data.SortGroupIDSFV = saveForgeValue(
			true,
			item.SortKey.SortGroupID,
			sortGroupID,
			"preserved conflicting SaveForge value from ItemSortKeys.SortGroupId",
		)
	}
	return data, nil
}
