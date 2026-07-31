package migration

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGenerateVerifiesAndAttachesPassiveEffectRows(t *testing.T) {
	options := localGenerateOptions(t)
	catalog, err := Generate(options)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	table, exists := options.Regulation.Table(RegulationTableSpEffect)
	if !exists {
		t.Fatal("SpEffectParam is not loaded")
	}

	total := 0
	known := 0
	unknown := 0
	presentIDs := make(map[int32]struct{})
	missingIDs := make(map[int32]struct{})
	for _, resource := range catalog.Resources {
		if resource.Item == nil ||
			resource.Item.Family.Value != schema.ItemFamilyWeapon {
			continue
		}
		checkPassiveEffectRows(
			t,
			table,
			resource.Item.Weapon,
			resource.Item.SourceRecords,
			&total,
			&known,
			&unknown,
			presentIDs,
			missingIDs,
		)
		for _, variant := range resource.Item.Variants {
			if variant.Data.Weapon == nil {
				t.Fatal("weapon variant data is missing")
			}
			resolved := *variant.Data.Weapon
			checkPassiveEffectRows(
				t,
				table,
				&resolved,
				variant.SourceRecords,
				&total,
				&known,
				&unknown,
				presentIDs,
				missingIDs,
			)
		}
	}

	if total != 1770 || known != 1297 || unknown != 473 {
		t.Fatalf(
			"passive effects total/known/unknown = %d/%d/%d, want 1770/1297/473",
			total,
			known,
			unknown,
		)
	}
	if len(presentIDs) != 128 {
		t.Fatalf("SpEffect IDs present in Regulation = %d, want 128", len(presentIDs))
	}
	wantMissing := map[int32]struct{}{
		5180500: {},
		5220300: {},
		5245100: {},
	}
	if !reflect.DeepEqual(missingIDs, wantMissing) {
		t.Fatalf("missing SpEffect IDs = %#v, want %#v", missingIDs, wantMissing)
	}
}

func checkPassiveEffectRows(
	t *testing.T,
	table *RegulationTable,
	weapon *schema.WeaponData,
	records []schema.ParameterRecord,
	total *int,
	known *int,
	unknown *int,
	presentIDs map[int32]struct{},
	missingIDs map[int32]struct{},
) {
	t.Helper()
	if weapon == nil {
		t.Fatal("weapon family data is missing")
	}
	recordsByID := make(map[int64]schema.ParameterRecord)
	for _, record := range records {
		if record.Table == string(RegulationTableSpEffect) {
			if _, duplicate := recordsByID[record.RowID]; duplicate {
				t.Fatalf("duplicate SpEffectParam source row %d", record.RowID)
			}
			recordsByID[record.RowID] = record
		}
	}
	for _, effect := range weapon.PassiveEffects {
		*total++
		if effect.Known.Value {
			*known++
		} else {
			*unknown++
		}
		row, exists := table.Row(uint32(effect.SpEffectID.Value))
		if !exists {
			missingIDs[effect.SpEffectID.Value] = struct{}{}
			if effect.Known.Value {
				t.Fatalf(
					"known passive effect %d is absent from SpEffectParam",
					effect.SpEffectID.Value,
				)
			}
			if _, attached := recordsByID[int64(effect.SpEffectID.Value)]; attached {
				t.Fatalf(
					"missing SpEffectParam row %d was attached",
					effect.SpEffectID.Value,
				)
			}
			continue
		}
		presentIDs[effect.SpEffectID.Value] = struct{}{}
		record, attached := recordsByID[int64(effect.SpEffectID.Value)]
		if !attached {
			t.Fatalf(
				"SpEffectParam row %d is not attached",
				effect.SpEffectID.Value,
			)
		}
		expected := parameterRecord(RegulationRowLookup{
			Table:    RegulationTableSpEffect,
			Source:   table.Source(),
			RawRowID: row.RowID,
			Row:      row,
		})
		expectedRecords := enrichParameterRecordFields(
			[]schema.ParameterRecord{expected},
			effect,
		)
		expected = expectedRecords[0]
		if !reflect.DeepEqual(record, expected) {
			t.Fatalf(
				"SpEffectParam source row %d differs from Regulation",
				row.RowID,
			)
		}
	}
}
