package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func buildLinks(value linksSeed) schema.ItemLinks {
	result := emptyItemLinks()
	if value.AboutTutorialID != nil {
		result.AboutTutorialID = knownLegacyFact(
			*value.AboutTutorialID,
			"copied from legacy AboutTutorialID",
		)
	}
	result.RelatedEventFlags = make(
		[]schema.RelatedEventFlag,
		len(value.RelatedEventFlags),
	)
	for index, related := range value.RelatedEventFlags {
		result.RelatedEventFlags[index] = schema.RelatedEventFlag{
			Kind: knownLegacyFact(
				schema.RelatedEventFlagKind(related.Kind),
				"copied from legacy whetblade mutation metadata",
			),
			EventFlagID: knownLegacyFact(
				related.FlagID,
				"copied from legacy whetblade mutation metadata",
			),
		}
	}
	result.RelatedItems = make([]schema.RelatedItem, len(value.RelatedItems))
	for index, related := range value.RelatedItems {
		result.RelatedItems[index] = schema.RelatedItem{
			Kind: knownLegacyFact(
				schema.RelatedItemKind(related.Kind),
				"copied from legacy whetblade bundled acquisition",
			),
			GameID: knownLegacyFact(
				related.ItemID,
				"copied from legacy whetblade bundled acquisition",
			),
		}
	}
	if value.WhetbladeName != "" {
		result.WhetbladeName = knownLegacyFact(
			value.WhetbladeName,
			"copied from legacy Whetblades.Name",
		)
	}
	if value.MapFragment != nil {
		result.MapFragment = &schema.MapFragmentMetadata{
			Name: knownLegacyFact(
				value.MapFragment.Name,
				"copied from legacy MapVisible.Name",
			),
			Area: knownLegacyFact(
				value.MapFragment.Area,
				"copied from legacy MapVisible.Area",
			),
			AcquiredFlagID: knownLegacyFact(
				value.MapFragment.AcquiredFlagID,
				"matched to legacy MapAcquired by region identity",
			),
		}
	}
	return result
}

func emptyItemLinks() schema.ItemLinks {
	return schema.ItemLinks{
		AboutTutorialID: unknownCatalogFact[uint32](
			"legacy AboutTutorialID has no entry for this item",
		),
		WhetbladeName: unknownCatalogFact[string](
			"legacy Whetblades has no entry for this item",
		),
	}
}
