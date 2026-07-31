package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	sourceLegacyData    schema.SourceID = schema.SourceSaveForgeLegacy
	sourceLegacyIcons   schema.SourceID = "legacy_item_icons"
	sourceLegacyUnknown schema.SourceID = "legacy_unknown"
)

var sourceIDByRegulationTable = map[RegulationTableName]schema.SourceID{
	RegulationTableWeapon:             "regulation_equip_param_weapon",
	RegulationTableProtector:          "regulation_equip_param_protector",
	RegulationTableAccessory:          "regulation_equip_param_accessory",
	RegulationTableGoods:              "regulation_equip_param_goods",
	RegulationTableGem:                "regulation_equip_param_gem",
	RegulationTableMagic:              "regulation_magic",
	RegulationTableGesture:            "regulation_gesture_param",
	RegulationTableSwordArts:          "regulation_sword_arts_param",
	RegulationTableSpEffect:           "regulation_sp_effect_param",
	RegulationTableTutorial:           "regulation_tutorial_param",
	RegulationTableMaterialSet:        "regulation_equip_mtrl_set_param",
	RegulationTableReinforceWeapon:    "regulation_reinforce_param_weapon",
	RegulationTableReinforceProtector: "regulation_reinforce_param_protector",
}

func buildManifest(
	options GenerateOptions,
	versions migrationSourceVersions,
) schema.Manifest {
	gameTextSources := options.GameText.manifestSources()
	sources := make(
		[]schema.DataSource,
		0,
		len(regulationTableSpecs)+len(gameTextSources)+4,
	)
	for _, spec := range regulationTableSpecs {
		table, _ := options.Regulation.Table(spec.name)
		source := table.Source()
		sources = append(sources, schema.DataSource{
			ID:       sourceIDByRegulationTable[spec.name],
			Kind:     "regulation_parameter_csv",
			Location: source.Location,
			Version:  source.Version,
			Evidence: schema.EvidenceRegulation,
			Reviewed: true,
		})
	}
	sources = append(sources, options.RegulationParams.manifestSource())
	sources = append(sources, gameTextSources...)
	sources = append(sources,
		schema.DataSource{
			ID:       sourceLegacyData,
			Kind:     "legacy_item_catalog",
			Location: "backend/db/data",
			Version:  versions.legacyData,
			Evidence: schema.EvidenceCurated,
			Reviewed: true,
		},
		schema.DataSource{
			ID:       sourceLegacyIcons,
			Kind:     "legacy_item_assets",
			Location: "frontend/public/items",
			Version:  versions.legacyIcons,
			Evidence: schema.EvidenceCurated,
			Reviewed: true,
		},
		schema.DataSource{
			ID:       sourceLegacyUnknown,
			Kind:     "legacy_unresolved_item",
			Location: "backend/db/data/gestures.go",
			Version:  versions.legacyData,
			Evidence: schema.EvidenceUnknown,
			Reviewed: false,
		},
	)
	return schema.Manifest{
		SchemaVersion: schema.CurrentSchemaVersion,
		GameVersion:   options.GameVersion,
		Sources:       sources,
	}
}

func regulationSourceID(table RegulationTableName) (schema.SourceID, error) {
	source, exists := sourceIDByRegulationTable[table]
	if !exists {
		return "", fmt.Errorf("regulation table %q has no source ID", table)
	}
	return source, nil
}
