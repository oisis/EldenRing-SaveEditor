package migration

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

func TestLegacyLinksExactCoverage(t *testing.T) {
	snapshot := collectLegacySnapshot()
	items := make(map[uint32]seed, len(snapshot.Items))
	for _, item := range snapshot.Items {
		items[item.ID] = item
	}

	tutorialCount := 0
	whetbladeCount := 0
	mapFragmentCount := 0
	for itemID, item := range items {
		if item.Links.AboutTutorialID != nil {
			tutorialCount++
			want, exists := data.AboutTutorialID[itemID]
			if !exists || *item.Links.AboutTutorialID != want {
				t.Fatalf(
					"item 0x%08X tutorial link = %#v, want %d, %t",
					itemID,
					item.Links.AboutTutorialID,
					want,
					exists,
				)
			}
		}
		if _, exists := data.WhetbladeItemToFlagID[itemID]; exists {
			whetbladeCount++
			assertLegacyWhetbladeLinks(t, item)
		}
		if item.Links.MapFragment != nil {
			mapFragmentCount++
			assertLegacyMapFragmentLink(t, item)
		}
	}
	if tutorialCount != len(data.AboutTutorialID) || tutorialCount != 1 {
		t.Fatalf("tutorial links = %d, want exact legacy count %d", tutorialCount, len(data.AboutTutorialID))
	}
	if whetbladeCount != len(data.WhetbladeItemToFlagID) || whetbladeCount != 6 {
		t.Fatalf("whetblade links = %d, want exact legacy count %d", whetbladeCount, len(data.WhetbladeItemToFlagID))
	}
	if mapFragmentCount != len(data.MapFragmentItemToFlagID) || mapFragmentCount != 24 {
		t.Fatalf("map-fragment links = %d, want exact legacy count %d", mapFragmentCount, len(data.MapFragmentItemToFlagID))
	}
}

func assertLegacyWhetbladeLinks(t *testing.T, item seed) {
	t.Helper()
	flagID, exists := data.WhetbladeItemToFlagID[item.ID]
	if !exists {
		t.Fatalf("item 0x%08X has fabricated whetblade links", item.ID)
	}
	wantFlags := make([]relatedEventFlagSeed, 0, len(data.WhetbladeRelatedFlags[flagID])+1)
	for _, relatedFlagID := range data.WhetbladeRelatedFlags[flagID] {
		wantFlags = append(wantFlags, relatedEventFlagSeed{
			Kind:   legacyRelatedEventWhetblade,
			FlagID: relatedFlagID,
		})
	}
	wantFlags = append(wantFlags, relatedEventFlagSeed{
		Kind:   legacyRelatedEventAoWMenu,
		FlagID: data.AoWMenuUnlockedFlag,
	})
	sortRelatedEventFlagSeeds(wantFlags)
	if !reflect.DeepEqual(item.Links.RelatedEventFlags, wantFlags) {
		t.Fatalf(
			"item 0x%08X related flags = %#v, want %#v",
			item.ID,
			item.Links.RelatedEventFlags,
			wantFlags,
		)
	}
	wantItems := []relatedItemSeed(nil)
	if flagID == data.WhetstoneKnifeFlag {
		wantItems = []relatedItemSeed{{
			Kind:   legacyRelatedItemBundled,
			ItemID: data.StormStompItemID,
		}}
	}
	if !reflect.DeepEqual(item.Links.RelatedItems, wantItems) {
		t.Fatalf(
			"item 0x%08X related items = %#v, want %#v",
			item.ID,
			item.Links.RelatedItems,
			wantItems,
		)
	}
	for _, unlock := range item.Unlocks {
		if unlock.Kind == "whetblade" && unlock.FlagID == flagID {
			metadata, metadataExists := data.Whetblades[flagID]
			if !metadataExists {
				t.Fatalf("item 0x%08X whetblade metadata is missing", item.ID)
			}
			if unlock.Name != metadata.Name {
				t.Fatalf(
					"item 0x%08X unlock name = %q, want %q",
					item.ID,
					unlock.Name,
					metadata.Name,
				)
			}
			return
		}
	}
	t.Fatalf("item 0x%08X has no whetblade unlock", item.ID)
}

func sortRelatedEventFlagSeeds(values []relatedEventFlagSeed) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0; j-- {
			previous := values[j-1]
			current := values[j]
			if previous.Kind < current.Kind ||
				(previous.Kind == current.Kind && previous.FlagID <= current.FlagID) {
				break
			}
			values[j-1], values[j] = values[j], values[j-1]
		}
	}
}

func assertLegacyMapFragmentLink(t *testing.T, item seed) {
	t.Helper()
	visibleFlagID, exists := data.MapFragmentItemToFlagID[item.ID]
	if !exists {
		t.Fatalf("item 0x%08X has fabricated map-fragment link", item.ID)
	}
	visible, visibleExists := data.MapVisible[visibleFlagID]
	acquiredFlagID := visibleFlagID + 1000
	acquired, acquiredExists := data.MapAcquired[acquiredFlagID]
	if !visibleExists ||
		!acquiredExists ||
		visible.Name != acquired.Name ||
		visible.Area != acquired.Area {
		t.Fatalf(
			"item 0x%08X legacy map metadata is inconsistent: visible=%#v/%t acquired=%#v/%t",
			item.ID,
			visible,
			visibleExists,
			acquired,
			acquiredExists,
		)
	}
	want := mapFragmentSeed{
		Name:           visible.Name,
		Area:           visible.Area,
		AcquiredFlagID: acquiredFlagID,
	}
	if *item.Links.MapFragment != want {
		t.Fatalf("item 0x%08X map metadata = %#v, want %#v", item.ID, *item.Links.MapFragment, want)
	}
}

func TestLegacySwordArtsAndCompatibilityNamesExactCoverage(t *testing.T) {
	snapshot := collectLegacySnapshot()
	if len(snapshot.SwordArtsNames) != len(data.SwordArtsNames) ||
		len(snapshot.SwordArtsNames) != 130 {
		t.Fatalf(
			"sword-art names = %d, want exact legacy count %d",
			len(snapshot.SwordArtsNames),
			len(data.SwordArtsNames),
		)
	}
	for _, value := range snapshot.SwordArtsNames {
		if got, exists := data.SwordArtsNames[value.ID]; !exists || got != value.Name {
			t.Fatalf(
				"sword-art %d = %q, want %q, %t",
				value.ID,
				value.Name,
				got,
				exists,
			)
		}
	}
	for _, item := range snapshot.Items {
		if item.AoWCompatMask == nil {
			if item.AoWCompatibleClasses != nil {
				t.Fatalf(
					"item 0x%08X has compatibility names without a mask",
					item.ID,
				)
			}
			continue
		}
		want := legacyAoWCompatibleClasses(*item.AoWCompatMask)
		if !reflect.DeepEqual(item.AoWCompatibleClasses, want) {
			t.Fatalf(
				"item 0x%08X compatibility names = %#v, want %#v",
				item.ID,
				item.AoWCompatibleClasses,
				want,
			)
		}
	}
}
