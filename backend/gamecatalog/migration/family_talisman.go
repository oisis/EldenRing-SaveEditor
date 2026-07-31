package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func buildTalismanData(row ParameterRow) (*schema.TalismanData, error) {
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
	weight, err := regulationFloat64(row, "weight")
	if err != nil {
		return nil, err
	}
	return &schema.TalismanData{
		SourceRowID: knownRegulationFact(row.RowID, RegulationTableAccessory, "Row ID", row.RowID),
		IconID:      knownRegulationFact(iconID, RegulationTableAccessory, "iconId", row.RowID),
		SortID:      knownRegulationFact(sortID, RegulationTableAccessory, "sortId", row.RowID),
		SortGroupID: knownRegulationFact(sortGroupID, RegulationTableAccessory, "sortGroupId", row.RowID),
		Weight:      knownRegulationFact(weight, RegulationTableAccessory, "weight", row.RowID),
	}, nil
}
