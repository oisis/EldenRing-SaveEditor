package migration

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

const (
	legacyRelatedEventWhetblade = "whetblade_related"
	legacyRelatedEventAoWMenu   = "aow_menu_unlock"
	legacyRelatedItemBundled    = "bundled_acquisition"
)

func collectLegacyLinks(itemID uint32) linksSeed {
	result := linksSeed{}
	if tutorialID, exists := data.AboutTutorialID[itemID]; exists {
		result.AboutTutorialID = &tutorialID
	}
	if flagID, exists := data.WhetbladeItemToFlagID[itemID]; exists {
		result = legacyWhetbladeLinks(flagID)
		if tutorialID, tutorialExists := data.AboutTutorialID[itemID]; tutorialExists {
			result.AboutTutorialID = &tutorialID
		}
	}
	if visibleFlagID, exists := data.MapFragmentItemToFlagID[itemID]; exists {
		visible, visibleExists := data.MapVisible[visibleFlagID]
		acquiredFlagID := visibleFlagID + 1000
		acquired, acquiredExists := data.MapAcquired[acquiredFlagID]
		if visibleExists &&
			acquiredExists &&
			visible.Name == acquired.Name &&
			visible.Area == acquired.Area {
			result.MapFragment = &mapFragmentSeed{
				Name:           visible.Name,
				Area:           visible.Area,
				AcquiredFlagID: acquiredFlagID,
			}
		}
	}
	return result
}

func legacyWhetbladeLinks(flagID uint32) linksSeed {
	result := linksSeed{}
	for _, relatedFlagID := range data.WhetbladeRelatedFlags[flagID] {
		result.RelatedEventFlags = append(
			result.RelatedEventFlags,
			relatedEventFlagSeed{
				Kind:   legacyRelatedEventWhetblade,
				FlagID: relatedFlagID,
			},
		)
	}
	result.RelatedEventFlags = append(
		result.RelatedEventFlags,
		relatedEventFlagSeed{
			Kind:   legacyRelatedEventAoWMenu,
			FlagID: data.AoWMenuUnlockedFlag,
		},
	)
	if flagID == data.WhetstoneKnifeFlag {
		result.RelatedItems = []relatedItemSeed{{
			Kind:   legacyRelatedItemBundled,
			ItemID: data.StormStompItemID,
		}}
	}
	sort.Slice(result.RelatedEventFlags, func(i, j int) bool {
		if result.RelatedEventFlags[i].Kind != result.RelatedEventFlags[j].Kind {
			return result.RelatedEventFlags[i].Kind < result.RelatedEventFlags[j].Kind
		}
		return result.RelatedEventFlags[i].FlagID < result.RelatedEventFlags[j].FlagID
	})
	return result
}
