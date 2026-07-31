package migration

import (
	"strings"
	"testing"
)

func TestLookupFamilyRowUsesExplicitFamilyRoleAndRawRowID(t *testing.T) {
	data, err := readRegulationFS(newRegulationMapFS())
	if err != nil {
		t.Fatalf("readRegulationFS: %v", err)
	}

	primary, exists, err := data.LookupFamilyRow(
		RegulationFamilyWeapon,
		RegulationTableRolePrimary,
		1000,
	)
	if err != nil || !exists {
		t.Fatalf("primary lookup = %+v, %t, %v", primary, exists, err)
	}
	if primary.Table != RegulationTableWeapon || primary.Row.RowID != 1000 {
		t.Fatalf("primary lookup = %+v", primary)
	}
	if primary.Source.Location != "regulation.bin/csv/EquipParamWeapon.csv" {
		t.Fatalf("primary source = %q", primary.Source.Location)
	}

	supporting, exists, err := data.LookupFamilyRow(
		RegulationFamilyWeapon,
		RegulationTableRoleSupporting,
		0,
	)
	if err != nil || !exists {
		t.Fatalf("supporting lookup = %+v, %t, %v", supporting, exists, err)
	}
	if supporting.Table != RegulationTableReinforceWeapon || supporting.Row.RowID != 0 {
		t.Fatalf("supporting lookup = %+v", supporting)
	}
	if supporting.Source.Location != "regulation.bin/csv/ReinforceParamWeapon.csv" {
		t.Fatalf("supporting source = %q", supporting.Source.Location)
	}
}

func TestLookupFamilyRowMapsEveryPrimaryFamily(t *testing.T) {
	data, err := readRegulationFS(newRegulationMapFS())
	if err != nil {
		t.Fatalf("readRegulationFS: %v", err)
	}

	tests := []struct {
		family RegulationFamily
		table  RegulationTableName
	}{
		{RegulationFamilyWeapon, RegulationTableWeapon},
		{RegulationFamilyProtector, RegulationTableProtector},
		{RegulationFamilyAccessory, RegulationTableAccessory},
		{RegulationFamilyGoods, RegulationTableGoods},
		{RegulationFamilyAshOfWar, RegulationTableGem},
		{RegulationFamilySpell, RegulationTableMagic},
		{RegulationFamilyGesture, RegulationTableGesture},
	}
	for _, test := range tests {
		t.Run(string(test.family), func(t *testing.T) {
			lookup, exists, lookupErr := data.LookupFamilyRow(
				test.family,
				RegulationTableRolePrimary,
				1,
			)
			if lookupErr != nil || !exists {
				t.Fatalf("lookup = %+v, %t, %v", lookup, exists, lookupErr)
			}
			if lookup.Table != test.table {
				t.Fatalf("table = %q, want %q", lookup.Table, test.table)
			}
		})
	}
}

func TestLookupFamilyRowFailsClosed(t *testing.T) {
	data, err := readRegulationFS(newRegulationMapFS())
	if err != nil {
		t.Fatalf("readRegulationFS: %v", err)
	}

	if lookup, exists, lookupErr := data.LookupFamilyRow(
		RegulationFamilyWeapon,
		RegulationTableRolePrimary,
		999,
	); lookupErr != nil || exists {
		t.Fatalf("missing lookup = %+v, %t, %v", lookup, exists, lookupErr)
	}

	_, _, err = data.LookupFamilyRow(
		RegulationFamilyGoods,
		RegulationTableRoleSupporting,
		1,
	)
	if err == nil || !strings.Contains(err.Error(), "has no supporting table") {
		t.Fatalf("unsupported supporting lookup error = %v", err)
	}

	_, _, err = data.LookupFamilyRow(
		RegulationFamily("unknown"),
		RegulationTableRolePrimary,
		1,
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported regulation family") {
		t.Fatalf("unknown family error = %v", err)
	}

	_, _, err = data.LookupFamilyRow(
		RegulationFamilyWeapon,
		RegulationTableRole("unknown"),
		1,
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported regulation table role") {
		t.Fatalf("unknown role error = %v", err)
	}
}
