package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func buildSpiritAshData(row ParameterRow) (*schema.SpiritAshData, error) {
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
	return &schema.SpiritAshData{
		SourceRowID:         knownRegulationFact(row.RowID, RegulationTableGoods, "Row ID", row.RowID),
		IconID:              knownRegulationFact(iconID, RegulationTableGoods, "iconId", row.RowID),
		ReinforceGoodsID:    knownRegulationFact(reinforceGoodsID, RegulationTableGoods, "reinforceGoodsId", row.RowID),
		ReinforceMaterialID: knownRegulationFact(reinforceMaterialID, RegulationTableGoods, "reinforceMaterialId", row.RowID),
		ReinforcePrice:      knownRegulationFact(reinforcePrice, RegulationTableGoods, "reinforcePrice", row.RowID),
	}, nil
}
