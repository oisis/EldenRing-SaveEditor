package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func buildSpellData(row ParameterRow) (*schema.SpellData, error) {
	iconID, err := regulationUint32(row, "iconId")
	if err != nil {
		return nil, err
	}
	sortID, err := regulationUint32(row, "sortId")
	if err != nil {
		return nil, err
	}
	fpCost, err := regulationUint32(row, "mp")
	if err != nil {
		return nil, err
	}
	staminaCost, err := regulationUint32(row, "stamina")
	if err != nil {
		return nil, err
	}
	memorySlots, err := regulationUint8(row, "slotLength")
	if err != nil {
		return nil, err
	}
	requiredIntelligence, err := regulationUint32(row, "requirementIntellect")
	if err != nil {
		return nil, err
	}
	requiredFaith, err := regulationUint32(row, "requirementFaith")
	if err != nil {
		return nil, err
	}
	requiredArcane, err := regulationUint32(row, "requirementLuck")
	if err != nil {
		return nil, err
	}
	return &schema.SpellData{
		SourceRowID:          knownRegulationFact(row.RowID, RegulationTableMagic, "Row ID", row.RowID),
		IconID:               knownRegulationFact(iconID, RegulationTableMagic, "iconId", row.RowID),
		SortID:               knownRegulationFact(sortID, RegulationTableMagic, "sortId", row.RowID),
		FPCost:               knownRegulationFact(fpCost, RegulationTableMagic, "mp", row.RowID),
		StaminaCost:          knownRegulationFact(staminaCost, RegulationTableMagic, "stamina", row.RowID),
		MemorySlots:          knownRegulationFact(memorySlots, RegulationTableMagic, "slotLength", row.RowID),
		RequiredIntelligence: knownRegulationFact(requiredIntelligence, RegulationTableMagic, "requirementIntellect", row.RowID),
		RequiredFaith:        knownRegulationFact(requiredFaith, RegulationTableMagic, "requirementFaith", row.RowID),
		RequiredArcane:       knownRegulationFact(requiredArcane, RegulationTableMagic, "requirementLuck", row.RowID),
	}, nil
}
