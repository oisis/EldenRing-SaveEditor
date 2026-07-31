package migration

import "testing"

func TestAuditLegacyVariantMetadataCoverage(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	groups, err := groupLegacyItems(collectLegacySnapshot().Items, regulation)
	if err != nil {
		t.Fatalf("groupLegacyItems: %v", err)
	}
	var text, description, limits, weaponStats, weight, weaponEdit, sortKey, acquisition, unlocks int
	for _, group := range groups {
		for _, variant := range group.Variants {
			item := variant.Item
			if item.Text != nil {
				text++
			}
			if item.Description != nil {
				description++
			}
			if item.GameLimits != nil {
				limits++
			}
			if item.WeaponStats != nil {
				weaponStats++
			}
			if item.Weight != nil {
				weight++
			}
			if item.WeaponEdit != nil {
				weaponEdit++
			}
			if item.SortKey != nil {
				sortKey++
			}
			if item.Acquisition.RequiredContainerID != nil ||
				item.Acquisition.WorldPickupFlagID != nil ||
				len(item.Acquisition.BolsteringPickupFlags) > 0 ||
				len(item.Acquisition.CompanionEventFlagIDs) > 0 {
				acquisition++
			}
			if len(item.Unlocks) > 0 {
				unlocks++
			}
		}
	}
	assertCount(t, "variant text records", text, 3624)
	assertCount(t, "variant description records", description, 3600)
	assertCount(t, "variant game-limit records", limits, 840)
	assertCount(t, "variant weapon-stat records", weaponStats, 2784)
	assertCount(t, "variant item-weight records", weight, 2784)
	assertCount(t, "variant weapon-edit records", weaponEdit, 2784)
	assertCount(t, "variant sort-key records", sortKey, 2784)
	assertCount(t, "variant acquisition records", acquisition, 0)
	assertCount(t, "variant unlock records", unlocks, 0)
}
