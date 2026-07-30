package prototype

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

const (
	DaggerGameID        uint32 = 0x000F4240
	DeterminationGameID uint32 = 0x8000EA60
)

const (
	sourceRegulationWeapon schema.SourceID = "regulation_weapon"
	sourceRegulationGem    schema.SourceID = "regulation_gem"
	sourceRegulationText   schema.SourceID = "regulation_text"
	sourceLegacyData       schema.SourceID = "legacy_db_data"
	sourceSaveResearch     schema.SourceID = "verified_save_research"
	sourceUnknown          schema.SourceID = "prototype_unknown"
)

func Data() (schema.Manifest, []schema.Resource) {
	return manifest(), []schema.Resource{
		daggerResource(),
		determinationResource(),
	}
}
