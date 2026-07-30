package prototype

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func daggerVariants(source schema.Provenance) []schema.ItemVariant {
	variants := []struct {
		offset   uint32
		affinity schema.Affinity
	}{
		{0, schema.AffinityStandard},
		{100, schema.AffinityHeavy},
		{200, schema.AffinityKeen},
		{300, schema.AffinityQuality},
		{400, schema.AffinityFire},
		{500, schema.AffinityFlameArt},
		{600, schema.AffinityLightning},
		{700, schema.AffinitySacred},
		{800, schema.AffinityMagic},
		{900, schema.AffinityCold},
		{1000, schema.AffinityPoison},
		{1100, schema.AffinityBlood},
		{1200, schema.AffinityOccult},
	}
	result := make([]schema.ItemVariant, 0, len(variants))
	for _, variant := range variants {
		rowID := uint32(1000000) + variant.offset
		result = append(result, schema.ItemVariant{
			GameID:      known(DaggerGameID+variant.offset, source),
			Affinity:    known(variant.affinity, source),
			SourceRowID: known(rowID, source),
		})
	}
	return result
}
