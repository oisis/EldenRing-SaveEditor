package migration

// RegulationTableName identifies one supported regulation parameter table.
type RegulationTableName string

const (
	RegulationTableWeapon             RegulationTableName = "EquipParamWeapon"
	RegulationTableProtector          RegulationTableName = "EquipParamProtector"
	RegulationTableAccessory          RegulationTableName = "EquipParamAccessory"
	RegulationTableGoods              RegulationTableName = "EquipParamGoods"
	RegulationTableGem                RegulationTableName = "EquipParamGem"
	RegulationTableMagic              RegulationTableName = "Magic"
	RegulationTableGesture            RegulationTableName = "GestureParam"
	RegulationTableSwordArts          RegulationTableName = "SwordArtsParam"
	RegulationTableSpEffect           RegulationTableName = "SpEffectParam"
	RegulationTableTutorial           RegulationTableName = "TutorialParam"
	RegulationTableMaterialSet        RegulationTableName = "EquipMtrlSetParam"
	RegulationTableReinforceWeapon    RegulationTableName = "ReinforceParamWeapon"
	RegulationTableReinforceProtector RegulationTableName = "ReinforceParamProtector"
)

type regulationTableSpec struct {
	name     RegulationTableName
	filename string
}

var regulationTableSpecs = []regulationTableSpec{
	{name: RegulationTableWeapon, filename: "EquipParamWeapon.csv"},
	{name: RegulationTableProtector, filename: "EquipParamProtector.csv"},
	{name: RegulationTableAccessory, filename: "EquipParamAccessory.csv"},
	{name: RegulationTableGoods, filename: "EquipParamGoods.csv"},
	{name: RegulationTableGem, filename: "EquipParamGem.csv"},
	{name: RegulationTableMagic, filename: "Magic.csv"},
	{name: RegulationTableGesture, filename: "GestureParam.csv"},
	{name: RegulationTableSwordArts, filename: "SwordArtsParam.csv"},
	{name: RegulationTableSpEffect, filename: "SpEffectParam.csv"},
	{name: RegulationTableTutorial, filename: "TutorialParam.csv"},
	{name: RegulationTableMaterialSet, filename: "EquipMtrlSetParam.csv"},
	{name: RegulationTableReinforceWeapon, filename: "ReinforceParamWeapon.csv"},
	{name: RegulationTableReinforceProtector, filename: "ReinforceParamProtector.csv"},
}

// RegulationSource identifies the immutable input used to load a parameter table.
type RegulationSource struct {
	Location string
	Version  string
}

// ParameterField preserves one CSV column exactly as it appeared in the input row.
type ParameterField struct {
	Name     string
	RawValue string
}

// ParameterRow is one parameter record indexed by its explicit raw Row ID.
type ParameterRow struct {
	RowID  uint32
	Fields []ParameterField
}

// Field returns the raw value of a named parameter column.
func (row ParameterRow) Field(name string) (string, bool) {
	for _, field := range row.Fields {
		if field.Name == name {
			return field.RawValue, true
		}
	}
	return "", false
}

// RegulationTable is an immutable, indexed view of one parameter CSV.
type RegulationTable struct {
	name     RegulationTableName
	source   RegulationSource
	rowIDs   []uint32
	rowsByID map[uint32]ParameterRow
}

func (table *RegulationTable) Name() RegulationTableName {
	return table.name
}

func (table *RegulationTable) Source() RegulationSource {
	return table.source
}

func (table *RegulationTable) RowCount() int {
	return len(table.rowIDs)
}

// Row returns a detached copy so callers cannot mutate the loaded table.
func (table *RegulationTable) Row(rawRowID uint32) (ParameterRow, bool) {
	row, exists := table.rowsByID[rawRowID]
	if !exists {
		return ParameterRow{}, false
	}
	return cloneParameterRow(row), true
}

// Rows returns detached records in their deterministic source-file order.
func (table *RegulationTable) Rows() []ParameterRow {
	rows := make([]ParameterRow, 0, len(table.rowIDs))
	for _, rowID := range table.rowIDs {
		row, _ := table.Row(rowID)
		rows = append(rows, row)
	}
	return rows
}

func cloneParameterRow(row ParameterRow) ParameterRow {
	cloned := row
	cloned.Fields = append([]ParameterField(nil), row.Fields...)
	return cloned
}

// RegulationData contains all parameter tables required by the migration layer.
type RegulationData struct {
	tables map[RegulationTableName]*RegulationTable
}

func (data *RegulationData) Table(name RegulationTableName) (*RegulationTable, bool) {
	table, exists := data.tables[name]
	return table, exists
}
