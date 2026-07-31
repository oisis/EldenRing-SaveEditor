package migration

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGenerateCrossChecksSwordArtsAndTutorialParameterRecords(t *testing.T) {
	options := localGenerateOptions(t)
	catalog, err := Generate(options)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	expectedNames, err := buildSwordArtsNameFacts(
		options.Regulation,
		options.GameText,
		collectLegacySnapshot().SwordArtsNames,
	)
	if err != nil {
		t.Fatalf("buildSwordArtsNameFacts: %v", err)
	}
	assertSupportingManifestSource(
		t,
		catalog.Manifest,
		"regulation_sword_arts_param",
		"regulation.bin/csv/SwordArtsParam.csv",
	)
	assertSupportingManifestSource(
		t,
		catalog.Manifest,
		"regulation_tutorial_param",
		"regulation.bin/csv/TutorialParam.csv",
	)

	swordArtsOwners := 0
	tutorialOwners := 0
	for _, resource := range catalog.Resources {
		if resource.Item == nil {
			continue
		}
		switch resource.Item.Family.Value {
		case schema.ItemFamilyWeapon:
			swordArtsOwners++
			assertSwordArtsParameterRecord(
				t,
				resource.Item.GameID.Value,
				resource.Item.Weapon.DefaultAshOfWarID,
				resource.Item.Weapon.SwordArtsName,
				resource.Item.SourceRecords,
				expectedNames,
			)
		case schema.ItemFamilyAshOfWar:
			swordArtsOwners++
			assertSwordArtsParameterRecord(
				t,
				resource.Item.GameID.Value,
				resource.Item.AshOfWar.SwordArtsParamID,
				resource.Item.AshOfWar.SwordArtsName,
				resource.Item.SourceRecords,
				expectedNames,
			)
		}
		if resource.Item.Links.AboutTutorialID.Known {
			tutorialOwners++
			assertSingleSupportingRecord(
				t,
				resource.Item.GameID.Value,
				resource.Item.SourceRecords,
				RegulationTableTutorial,
				int64(resource.Item.Links.AboutTutorialID.Value),
			)
		}
		for _, variant := range resource.Item.Variants {
			switch resource.Item.Family.Value {
			case schema.ItemFamilyWeapon:
				if variant.Data.Weapon == nil {
					t.Fatal("weapon variant data is missing")
				}
				resolved := *variant.Data.Weapon
				swordArtsOwners++
				assertSwordArtsParameterRecord(
					t,
					variant.GameID.Value,
					resolved.DefaultAshOfWarID,
					resolved.SwordArtsName,
					variant.SourceRecords,
					expectedNames,
				)
			}
		}
	}
	if swordArtsOwners != 3448 {
		t.Fatalf("weapon/Ash-of-War records = %d, want 3448", swordArtsOwners)
	}
	if tutorialOwners != len(data.AboutTutorialID) || tutorialOwners != 1 {
		t.Fatalf(
			"tutorial-linked records = %d, want exact legacy count %d",
			tutorialOwners,
			len(data.AboutTutorialID),
		)
	}
}

func assertSupportingManifestSource(
	t *testing.T,
	manifest schema.Manifest,
	sourceID schema.SourceID,
	location string,
) {
	t.Helper()
	for _, source := range manifest.Sources {
		if source.ID != sourceID {
			continue
		}
		if source.Location != location || source.Version == "" {
			t.Fatalf(
				"manifest source %q = location %q version %q",
				sourceID,
				source.Location,
				source.Version,
			)
		}
		return
	}
	t.Fatalf("manifest source %q is missing", sourceID)
}

func assertSwordArtsParameterRecord(
	t *testing.T,
	ownerID uint32,
	paramID schema.Fact[int32],
	name schema.Fact[string],
	records []schema.ParameterRecord,
	expectedNames map[int32]schema.Fact[string],
) {
	t.Helper()
	if !paramID.Known {
		t.Fatalf("item 0x%08X swordArtsParamId is unknown", ownerID)
	}
	if paramID.Value >= 0 {
		assertSingleSupportingRecord(
			t,
			ownerID,
			records,
			RegulationTableSwordArts,
			int64(paramID.Value),
		)
	} else {
		assertNoSupportingRecord(t, ownerID, records, RegulationTableSwordArts)
	}
	wantName, exists := expectedNames[paramID.Value]
	if !exists {
		t.Fatalf(
			"item 0x%08X has no expected sword-art fact for param %d",
			ownerID,
			paramID.Value,
		)
	}
	if !reflect.DeepEqual(name, wantName) {
		t.Fatalf(
			"item 0x%08X sword-art name = %#v, want %#v",
			ownerID,
			name,
			wantName,
		)
	}
}

func assertSingleSupportingRecord(
	t *testing.T,
	ownerID uint32,
	records []schema.ParameterRecord,
	table RegulationTableName,
	rowID int64,
) {
	t.Helper()
	count := 0
	for _, record := range records {
		if record.Table != string(table) {
			continue
		}
		count++
		if record.RowID != rowID {
			t.Fatalf(
				"item 0x%08X %s row = %d, want %d",
				ownerID,
				table,
				record.RowID,
				rowID,
			)
		}
	}
	if count != 1 {
		t.Fatalf(
			"item 0x%08X %s record count = %d, want 1",
			ownerID,
			table,
			count,
		)
	}
}

func assertNoSupportingRecord(
	t *testing.T,
	ownerID uint32,
	records []schema.ParameterRecord,
	table RegulationTableName,
) {
	t.Helper()
	for _, record := range records {
		if record.Table == string(table) {
			t.Fatalf(
				"item 0x%08X has unexpected %s record %d",
				ownerID,
				table,
				record.RowID,
			)
		}
	}
}
