package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func buildGoodsData(row ParameterRow) (*schema.GoodsData, error) {
	iconID, err := regulationUint32(row, "iconId")
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
	goodsType, err := regulationUint16(row, "goodsType")
	if err != nil {
		return nil, err
	}
	weight, err := regulationFloat64(row, "weight")
	if err != nil {
		return nil, err
	}
	maxQuantity, err := regulationUint32(row, "maxNum")
	if err != nil {
		return nil, err
	}
	maxRepository, err := regulationUint32(row, "maxRepositoryNum")
	if err != nil {
		return nil, err
	}
	tutorialFlagID, err := regulationUint32(row, "itemGetTutorialFlagId")
	if err != nil {
		return nil, err
	}
	isEquipable, err := regulationBool(row, "isEquip")
	if err != nil {
		return nil, err
	}
	isConsumable, err := regulationBool(row, "isConsume")
	if err != nil {
		return nil, err
	}
	isDiscardable, err := regulationBool(row, "isDiscard")
	if err != nil {
		return nil, err
	}
	isDepositable, err := regulationBool(row, "isDeposit")
	if err != nil {
		return nil, err
	}
	isDroppable, err := regulationBool(row, "isDrop")
	if err != nil {
		return nil, err
	}
	return &schema.GoodsData{
		SourceRowID:    knownRegulationFact(row.RowID, RegulationTableGoods, "Row ID", row.RowID),
		IconID:         knownRegulationFact(iconID, RegulationTableGoods, "iconId", row.RowID),
		SortID:         knownRegulationFact(sortID, RegulationTableGoods, "sortId", row.RowID),
		SortGroupID:    knownRegulationFact(sortGroupID, RegulationTableGoods, "sortGroupId", row.RowID),
		GoodsType:      knownRegulationFact(goodsType, RegulationTableGoods, "goodsType", row.RowID),
		Weight:         knownRegulationFact(weight, RegulationTableGoods, "weight", row.RowID),
		MaxQuantity:    knownRegulationFact(maxQuantity, RegulationTableGoods, "maxNum", row.RowID),
		MaxRepository:  knownRegulationFact(maxRepository, RegulationTableGoods, "maxRepositoryNum", row.RowID),
		TutorialFlagID: knownRegulationFact(tutorialFlagID, RegulationTableGoods, "itemGetTutorialFlagId", row.RowID),
		IsEquipable:    knownRegulationFact(isEquipable, RegulationTableGoods, "isEquip", row.RowID),
		IsConsumable:   knownRegulationFact(isConsumable, RegulationTableGoods, "isConsume", row.RowID),
		IsDiscardable:  knownRegulationFact(isDiscardable, RegulationTableGoods, "isDiscard", row.RowID),
		IsDepositable:  knownRegulationFact(isDepositable, RegulationTableGoods, "isDeposit", row.RowID),
		IsDroppable:    knownRegulationFact(isDroppable, RegulationTableGoods, "isDrop", row.RowID),
	}, nil
}
