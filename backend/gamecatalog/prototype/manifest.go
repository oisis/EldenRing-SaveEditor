package prototype

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func manifest() schema.Manifest {
	return schema.Manifest{
		SchemaVersion: 1,
		DataVersion:   "prototype-1",
		GameVersion:   "regulation-version-not-recorded",
		Sources: []schema.DataSource{
			{
				ID:       sourceRegulationWeapon,
				Kind:     "regulation",
				Location: "regulation.bin/csv/EquipParamWeapon.csv",
				Version:  "version-not-recorded",
				Evidence: schema.EvidenceRegulation,
			},
			{
				ID:       sourceRegulationGem,
				Kind:     "regulation",
				Location: "regulation.bin/csv/EquipParamGem.csv",
				Version:  "version-not-recorded",
				Evidence: schema.EvidenceRegulation,
			},
			{
				ID:       sourceRegulationText,
				Kind:     "regulation",
				Location: "regulation.bin/msg/item.msgbnd",
				Version:  "version-not-recorded",
				Evidence: schema.EvidenceRegulation,
			},
			{
				ID:       sourceLegacyData,
				Kind:     "curated_data",
				Location: "backend/db/data",
				Version:  "working-tree",
				Evidence: schema.EvidenceCurated,
			},
			{
				ID:       sourceSaveResearch,
				Kind:     "research",
				Location: "docs/sl2-binary-format-spec.md",
				Version:  "working-tree",
				Evidence: schema.EvidenceVerifiedResearch,
				Reviewed: true,
			},
			{
				ID:       sourceUnknown,
				Kind:     "unresolved",
				Location: "unresolved",
				Version:  "prototype-1",
				Evidence: schema.EvidenceUnknown,
			},
		},
	}
}
