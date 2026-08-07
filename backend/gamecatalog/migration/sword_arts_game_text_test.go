package migration

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestBuildSwordArtsNameFactsPrefersGameTextAndAuditsLegacyConflict(t *testing.T) {
	regulation := &RegulationData{tables: map[RegulationTableName]*RegulationTable{
		RegulationTableSwordArts: {
			name:   RegulationTableSwordArts,
			rowIDs: []uint32{10, 20, 30},
			rowsByID: map[uint32]ParameterRow{
				10: swordArtsRow(10, "100"),
				20: swordArtsRow(20, "200"),
				30: swordArtsRow(30, "300"),
			},
		},
	}}
	gameText := &GameTextData{names: map[gameTextCatalog]map[int32]gameTextName{
		gameTextCatalogArts: {
			200: {text: "Official Name", source: sourceGameTextArtsNameBase},
			300: {text: "Official Replacement", source: sourceGameTextArtsNameDLC},
		},
	}}
	legacy := []swordArtsNameSeed{
		{ID: 10, Name: "Legacy Fallback"},
		{ID: 20, Name: "Official Name"},
		{ID: 30, Name: "Outdated Legacy Name"},
	}

	facts, err := buildSwordArtsNameFacts(regulation, gameText, legacy)
	if err != nil {
		t.Fatalf("buildSwordArtsNameFacts: %v", err)
	}
	if got := facts[10]; !got.Known ||
		got.Value != "Legacy Fallback" ||
		got.Provenance.Source != sourceLegacyData ||
		!strings.Contains(got.Provenance.Method, "FMG has no nonblank text") {
		t.Fatalf("legacy fallback fact = %#v", got)
	}
	if got := facts[20]; !got.Known ||
		got.Value != "Official Name" ||
		got.Provenance.Source != sourceGameTextArtsNameBase ||
		strings.Contains(got.Provenance.Method, "conflicting legacy") {
		t.Fatalf("matching game-text fact = %#v", got)
	}
	if got := facts[30]; !got.Known ||
		got.Value != "Official Replacement" ||
		got.Provenance.Source != sourceGameTextArtsNameDLC ||
		!strings.Contains(got.Provenance.Method, `replaced conflicting legacy name "Outdated Legacy Name"`) {
		t.Fatalf("conflicting game-text fact = %#v", got)
	}
}

func TestGenerateSwordArtsNamesExactCoverage(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	namesByParamID := make(map[int32]schema.Fact[string])
	add := func(ownerID uint32, paramID schema.Fact[int32], name schema.Fact[string]) {
		t.Helper()
		if !paramID.Known {
			t.Fatalf("item 0x%08X has unknown swordArtsParamId", ownerID)
		}
		if previous, duplicate := namesByParamID[paramID.Value]; duplicate {
			if !reflect.DeepEqual(previous, name) {
				t.Fatalf(
					"swordArtsParamId %d has inconsistent facts %#v and %#v",
					paramID.Value,
					previous,
					name,
				)
			}
			return
		}
		namesByParamID[paramID.Value] = name
	}

	for _, resource := range catalog.Resources {
		if resource.Item == nil {
			continue
		}
		switch resource.Item.Family.Value {
		case schema.ItemFamilyWeapon:
			add(
				resource.Item.GameID.Value,
				resource.Item.Weapon.DefaultAshOfWarID,
				resource.Item.Weapon.SwordArtsName,
			)
			for _, variant := range resource.Item.Variants {
				resolved := resolvedVariantWeapon(t, resource.Item, variant)
				add(
					variant.GameID.Value,
					resolved.DefaultAshOfWarID,
					resolved.SwordArtsName,
				)
			}
		case schema.ItemFamilyAshOfWar:
			add(
				resource.Item.GameID.Value,
				resource.Item.AshOfWar.SwordArtsParamID,
				resource.Item.AshOfWar.SwordArtsName,
			)
		}
	}

	if len(namesByParamID) != 267 {
		t.Fatalf("used unique SwordArtsParam rows = %d, want 267", len(namesByParamID))
	}
	known := 0
	conflicts := 0
	unknownIDs := make([]int32, 0, 1)
	for paramID, name := range namesByParamID {
		if !name.Known {
			unknownIDs = append(unknownIDs, paramID)
			continue
		}
		known++
		switch name.Provenance.Source {
		case sourceGameTextArtsNameBase, sourceGameTextArtsNameDLC, sourceLegacyData:
		default:
			t.Fatalf(
				"swordArtsParamId %d has unsupported name source %q",
				paramID,
				name.Provenance.Source,
			)
		}
		if strings.Contains(name.Provenance.Method, "replaced conflicting legacy name") {
			conflicts++
		}
	}
	if known != 266 {
		t.Fatalf("known used SwordArts names = %d, want 266", known)
	}
	if len(unknownIDs) != 1 || unknownIDs[0] != 0 {
		t.Fatalf("unknown used SwordArts rows = %v, want [0]", unknownIDs)
	}
	if conflicts != 22 {
		t.Fatalf("audited FMG/legacy conflicts = %d, want 22", conflicts)
	}
}

func swordArtsRow(rowID uint32, textID string) ParameterRow {
	return ParameterRow{
		RowID: rowID,
		Fields: []ParameterField{
			{Name: "Row ID", RawValue: decimalRowID(rowID)},
			{Name: "textId", RawValue: textID},
		},
	}
}

func TestGenerateGameTextManifestHasLogicalProvenanceOnly(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	wantLocations := map[schema.SourceID]string{
		sourceGameTextArtsNameBase: "regulation.bin/msg/engus/item.msgbnd/ArtsName.fmg",
		sourceGameTextArtsNameDLC:  "regulation.bin/msg/engus/item_dlc02.msgbnd/ArtsName_dlc01.fmg",
	}
	found := make(map[schema.SourceID]bool, len(wantLocations))
	for _, source := range catalog.Manifest.Sources {
		if strings.Contains(source.Location, "tmp/") ||
			strings.Contains(source.Location, "regulation-bin-dump") {
			t.Fatalf("manifest source exposes local fixture path %q", source.Location)
		}
		want, relevant := wantLocations[source.ID]
		if !relevant {
			continue
		}
		if source.Location != want ||
			source.Version == "" ||
			source.Evidence != schema.EvidenceGameData {
			t.Fatalf("game-text source %q = %+v", source.ID, source)
		}
		found[source.ID] = true
	}
	for sourceID := range wantLocations {
		if !found[sourceID] {
			t.Fatalf("game-text source %q is missing", sourceID)
		}
	}
}
