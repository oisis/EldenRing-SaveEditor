package migration

import (
	"errors"
	"fmt"
)

// RegulationFamily selects a parameter family explicitly. It is never inferred
// from a Row ID or from the presence of a record in another parameter table.
type RegulationFamily string

const (
	RegulationFamilyWeapon    RegulationFamily = "weapon"
	RegulationFamilyProtector RegulationFamily = "protector"
	RegulationFamilyAccessory RegulationFamily = "accessory"
	RegulationFamilyGoods     RegulationFamily = "goods"
	RegulationFamilyAshOfWar  RegulationFamily = "ash_of_war"
	RegulationFamilySpell     RegulationFamily = "spell"
	RegulationFamilyGesture   RegulationFamily = "gesture"
)

type RegulationTableRole string

const (
	RegulationTableRolePrimary    RegulationTableRole = "primary"
	RegulationTableRoleSupporting RegulationTableRole = "supporting"
)

type regulationFamilyTableSet struct {
	primary    RegulationTableName
	supporting RegulationTableName
}

var regulationTablesByFamily = map[RegulationFamily]regulationFamilyTableSet{
	RegulationFamilyWeapon: {
		primary:    RegulationTableWeapon,
		supporting: RegulationTableReinforceWeapon,
	},
	RegulationFamilyProtector: {
		primary:    RegulationTableProtector,
		supporting: RegulationTableReinforceProtector,
	},
	RegulationFamilyAccessory: {primary: RegulationTableAccessory},
	RegulationFamilyGoods:     {primary: RegulationTableGoods},
	RegulationFamilyAshOfWar:  {primary: RegulationTableGem},
	RegulationFamilySpell:     {primary: RegulationTableMagic},
	RegulationFamilyGesture:   {primary: RegulationTableGesture},
}

// RegulationRowLookup records the explicit family, role and raw Row ID used for
// one lookup, together with the table provenance that supplied the result.
type RegulationRowLookup struct {
	Family   RegulationFamily
	Role     RegulationTableRole
	Table    RegulationTableName
	Source   RegulationSource
	RawRowID uint32
	Row      ParameterRow
}

// LookupFamilyRow resolves one explicitly supplied raw Row ID in the primary or
// supporting table owned by family. It does not derive a family or another Row ID.
func (data *RegulationData) LookupFamilyRow(
	family RegulationFamily,
	role RegulationTableRole,
	rawRowID uint32,
) (RegulationRowLookup, bool, error) {
	if data == nil {
		return RegulationRowLookup{}, false, errors.New("regulation data is required")
	}

	tableSet, exists := regulationTablesByFamily[family]
	if !exists {
		return RegulationRowLookup{}, false, fmt.Errorf("unsupported regulation family %q", family)
	}

	var tableName RegulationTableName
	switch role {
	case RegulationTableRolePrimary:
		tableName = tableSet.primary
	case RegulationTableRoleSupporting:
		if tableSet.supporting == "" {
			return RegulationRowLookup{}, false, fmt.Errorf(
				"regulation family %q has no supporting table",
				family,
			)
		}
		tableName = tableSet.supporting
	default:
		return RegulationRowLookup{}, false, fmt.Errorf("unsupported regulation table role %q", role)
	}

	table, exists := data.Table(tableName)
	if !exists {
		return RegulationRowLookup{}, false, fmt.Errorf("regulation table %q is not loaded", tableName)
	}
	row, exists := table.Row(rawRowID)
	if !exists {
		return RegulationRowLookup{}, false, nil
	}
	return RegulationRowLookup{
		Family:   family,
		Role:     role,
		Table:    tableName,
		Source:   table.Source(),
		RawRowID: rawRowID,
		Row:      row,
	}, true, nil
}
