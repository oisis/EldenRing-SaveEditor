package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func buildArmorData(item seed, row ParameterRow) (*schema.ArmorData, error) {
	iconMale, err := regulationUint32(row, "iconIdM")
	if err != nil {
		return nil, err
	}
	iconFemale, err := regulationUint32(row, "iconIdF")
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
	equipment, err := readArmorEquipmentFlags(row)
	if err != nil {
		return nil, err
	}
	stats, err := readRegulationArmorStats(row)
	if err != nil {
		return nil, err
	}
	data := &schema.ArmorData{
		SourceRowID:   knownRegulationFact(row.RowID, RegulationTableProtector, "Row ID", row.RowID),
		IconIDMale:    knownRegulationFact(iconMale, RegulationTableProtector, "iconIdM", row.RowID),
		IconIDFemale:  knownRegulationFact(iconFemale, RegulationTableProtector, "iconIdF", row.RowID),
		SortID:        knownRegulationFact(sortID, RegulationTableProtector, "sortId", row.RowID),
		SortGroupID:   knownRegulationFact(sortGroupID, RegulationTableProtector, "sortGroupId", row.RowID),
		Weight:        knownRegulationFact(stats.weight, RegulationTableProtector, "weight converted to one-decimal display value", row.RowID),
		Physical:      knownRegulationFact(stats.physical, RegulationTableProtector, "neutralDamageCutRate converted to display negation", row.RowID),
		Strike:        knownRegulationFact(stats.strike, RegulationTableProtector, "blowDamageCutRate converted to display negation", row.RowID),
		Slash:         knownRegulationFact(stats.slash, RegulationTableProtector, "slashDamageCutRate converted to display negation", row.RowID),
		Pierce:        knownRegulationFact(stats.pierce, RegulationTableProtector, "thrustDamageCutRate converted to display negation", row.RowID),
		Magic:         knownRegulationFact(stats.magic, RegulationTableProtector, "magicDamageCutRate converted to display negation", row.RowID),
		Fire:          knownRegulationFact(stats.fire, RegulationTableProtector, "fireDamageCutRate converted to display negation", row.RowID),
		Lightning:     knownRegulationFact(stats.lightning, RegulationTableProtector, "thunderDamageCutRate converted to display negation", row.RowID),
		Holy:          knownRegulationFact(stats.holy, RegulationTableProtector, "darkDamageCutRate converted to display negation", row.RowID),
		Immunity:      knownRegulationFact(stats.immunity, RegulationTableProtector, "resistPoison", row.RowID),
		Robustness:    knownRegulationFact(stats.robustness, RegulationTableProtector, "resistBlood", row.RowID),
		Focus:         knownRegulationFact(stats.focus, RegulationTableProtector, "resistSleep", row.RowID),
		Vitality:      knownRegulationFact(stats.vitality, RegulationTableProtector, "resistCurse", row.RowID),
		Poise:         knownRegulationFact(stats.poise, RegulationTableProtector, "toughnessCorrectRate converted to display poise", row.RowID),
		HeadEquipable: knownRegulationFact(equipment.head, RegulationTableProtector, "headEquip", row.RowID),
		BodyEquipable: knownRegulationFact(equipment.body, RegulationTableProtector, "bodyEquip", row.RowID),
		ArmEquipable:  knownRegulationFact(equipment.arms, RegulationTableProtector, "armEquip", row.RowID),
		LegEquipable:  knownRegulationFact(equipment.legs, RegulationTableProtector, "legEquip", row.RowID),
	}
	if err := attachArmorSaveForgeValues(data, item, stats, sortID, sortGroupID); err != nil {
		return nil, err
	}
	return data, nil
}
