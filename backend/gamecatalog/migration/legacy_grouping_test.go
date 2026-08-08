package migration

import "testing"

func TestGroupLegacyItemsUsesExplicitRegulationRelationships(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	items := collectLegacySnapshot().Items
	groups, err := groupLegacyItems(items, regulation)
	if err != nil {
		t.Fatalf("groupLegacyItems: %v", err)
	}

	if len(groups) != 2808 {
		t.Fatalf("canonical legacy documents = %d, want 2808", len(groups))
	}
	seen := make(map[uint32]struct{}, len(items))
	affinityCount := 0
	upgradeCount := 0
	for _, group := range groups {
		assertUniqueGroupedID(t, seen, group.Canonical.ID)
		for _, variant := range group.Variants {
			assertUniqueGroupedID(t, seen, variant.Item.ID)
			if variant.CanonicalID != group.Canonical.ID {
				t.Fatalf(
					"variant 0x%08X canonical = 0x%08X, group = 0x%08X",
					variant.Item.ID,
					variant.CanonicalID,
					group.Canonical.ID,
				)
			}
			switch variant.Kind {
			case legacyVariantAffinity:
				affinityCount++
			case legacyVariantUpgrade:
				upgradeCount++
			default:
				t.Fatalf("variant 0x%08X has unknown kind %q", variant.Item.ID, variant.Kind)
			}
		}
	}
	if len(seen) != 6384 {
		t.Fatalf("grouped catalog IDs = %d, want 6384", len(seen))
	}
	if affinityCount != 2736 {
		t.Fatalf("weapon affinity variants = %d, want 2736", affinityCount)
	}
	if upgradeCount != 840 {
		t.Fatalf("spirit-ash upgrade variants = %d, want 840", upgradeCount)
	}
}

func TestGroupLegacyItemsExcludesAffinityRowsForFixedCanonicalWeapons(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	items := collectLegacySnapshot().Items
	groups, err := groupLegacyItems(items, regulation)
	if err != nil {
		t.Fatalf("groupLegacyItems: %v", err)
	}
	grouped := make(map[uint32]struct{}, len(items))
	for _, group := range groups {
		grouped[group.Canonical.ID] = struct{}{}
		for _, variant := range group.Variants {
			grouped[variant.Item.ID] = struct{}{}
		}
	}

	for _, item := range items {
		if !legacyWeaponCategory(item.Category) {
			continue
		}
		identity, err := primaryRegulationForLegacyItem(item)
		if err != nil {
			t.Fatalf("item 0x%08X identity: %v", item.ID, err)
		}
		lookup, exists, err := regulation.LookupFamilyRow(
			RegulationFamilyWeapon,
			RegulationTableRolePrimary,
			identity.RowID,
		)
		if err != nil || !exists {
			t.Fatalf("item 0x%08X row lookup = %+v, %t, %v", item.ID, lookup, exists, err)
		}
		originRaw, exists := lookup.Row.Field("originEquipWep")
		if !exists {
			t.Fatalf("item 0x%08X has no originEquipWep field", item.ID)
		}
		origin, err := parseRegulationUint32(originRaw)
		if err != nil || origin >= identity.RowID {
			continue
		}
		delta := identity.RowID - origin
		if delta%100 != 0 {
			continue
		}
		if _, affinity := affinityByRegulationOffset[delta/100]; !affinity {
			continue
		}
		canonical, exists, err := regulation.LookupFamilyRow(
			RegulationFamilyWeapon,
			RegulationTableRolePrimary,
			origin,
		)
		if err != nil || !exists {
			t.Fatalf("item 0x%08X canonical row lookup = %+v, %t, %v", item.ID, canonical, exists, err)
		}
		allowed, err := canonicalAllowsAffinityVariants(canonical.Row)
		if err != nil {
			t.Fatalf("item 0x%08X canonical affinity capability: %v", item.ID, err)
		}
		if !allowed {
			if _, exists := grouped[item.ID]; exists {
				t.Fatalf("fixed weapon affinity row 0x%08X remains in the catalog", item.ID)
			}
		}
	}
}

func TestGroupLegacyItemsIncludesAllDaggerAffinityVariants(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	groups, err := groupLegacyItems(collectLegacySnapshot().Items, regulation)
	if err != nil {
		t.Fatalf("groupLegacyItems: %v", err)
	}

	dagger := findLegacyGroup(t, groups, 0x000F4240)
	if len(dagger.Variants) != len(affinityByRegulationOffset) {
		t.Fatalf(
			"Dagger affinity variants = %d, want %d",
			len(dagger.Variants),
			len(affinityByRegulationOffset),
		)
	}
	for offset, affinity := range affinityByRegulationOffset {
		id := uint32(1_000_000) + offset*100
		variant := findLegacyVariant(t, dagger.Variants, id)
		if variant.Affinity != affinity {
			t.Fatalf(
				"Dagger variant %d affinity = %q, want %q",
				id,
				variant.Affinity,
				affinity,
			)
		}
	}
}

func TestGroupLegacyItemsKeepsFlaskLevelsCanonical(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	groups, err := groupLegacyItems(collectLegacySnapshot().Items, regulation)
	if err != nil {
		t.Fatalf("groupLegacyItems: %v", err)
	}

	flaskIDs := []uint32{
		0x400003E9, 0x400003EB, 0x400003ED, 0x400003EF, 0x400003F1, 0x400003F3,
		0x400003F5, 0x400003F7, 0x400003F9, 0x400003FB, 0x400003FD, 0x400003FF,
		0x40000401,
		0x4000041B, 0x4000041D, 0x4000041F, 0x40000421, 0x40000423, 0x40000425,
		0x40000427, 0x40000429, 0x4000042B, 0x4000042D, 0x4000042F, 0x40000431,
		0x40000433,
	}
	groupsByID := make(map[uint32]legacyItemGroup, len(groups))
	for _, group := range groups {
		groupsByID[group.Canonical.ID] = group
	}
	for _, id := range flaskIDs {
		group, exists := groupsByID[id]
		if !exists {
			t.Fatalf("flask 0x%08X was incorrectly grouped as a variant", id)
		}
		if len(group.Variants) != 0 {
			t.Fatalf("flask 0x%08X has variants: %#v", id, group.Variants)
		}
	}
}

func TestGroupLegacyItemsPreservesRegulationAffinityAndUpgradeLevel(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	groups, err := groupLegacyItems(collectLegacySnapshot().Items, regulation)
	if err != nil {
		t.Fatalf("groupLegacyItems: %v", err)
	}

	queelign := findLegacyGroup(t, groups, 0x00632EA0)
	heavy := findLegacyVariant(t, queelign.Variants, 0x00632F04)
	if heavy.Kind != legacyVariantAffinity || heavy.Affinity != "heavy" {
		t.Fatalf("Heavy Dagger variant = %#v", heavy)
	}

	fangedImp := findLegacyGroup(t, groups, 0x400318F8)
	plusTen := findLegacyVariant(t, fangedImp.Variants, 0x40031902)
	if plusTen.Kind != legacyVariantUpgrade || plusTen.UpgradeLevel != 10 {
		t.Fatalf("Fanged Imp Ashes +10 variant = %#v", plusTen)
	}
}

func assertUniqueGroupedID(t *testing.T, seen map[uint32]struct{}, id uint32) {
	t.Helper()
	if _, duplicate := seen[id]; duplicate {
		t.Fatalf("legacy ID 0x%08X appears in more than one group position", id)
	}
	seen[id] = struct{}{}
}

func findLegacyGroup(t *testing.T, groups []legacyItemGroup, id uint32) legacyItemGroup {
	t.Helper()
	for _, group := range groups {
		if group.Canonical.ID == id {
			return group
		}
	}
	t.Fatalf("legacy group 0x%08X not found", id)
	return legacyItemGroup{}
}

func findLegacyVariant(t *testing.T, variants []legacyVariantSeed, id uint32) legacyVariantSeed {
	t.Helper()
	for _, variant := range variants {
		if variant.Item.ID == id {
			return variant
		}
	}
	t.Fatalf("legacy variant 0x%08X not found", id)
	return legacyVariantSeed{}
}
