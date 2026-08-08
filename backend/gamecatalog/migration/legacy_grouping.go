package migration

import (
	"fmt"
	"sort"
	"strconv"
)

type legacyVariantKind string

const (
	legacyVariantAffinity legacyVariantKind = "affinity"
	legacyVariantUpgrade  legacyVariantKind = "upgrade"
)

type legacyVariantSeed struct {
	Item         seed
	Kind         legacyVariantKind
	CanonicalID  uint32
	Affinity     string
	UpgradeLevel uint8
}

type legacyItemGroup struct {
	Canonical seed
	Variants  []legacyVariantSeed
}

var affinityByRegulationOffset = map[uint32]string{
	1:  "heavy",
	2:  "keen",
	3:  "quality",
	4:  "fire",
	5:  "flame_art",
	6:  "lightning",
	7:  "sacred",
	8:  "magic",
	9:  "cold",
	10: "poison",
	11: "blood",
	12: "occult",
}

func groupLegacyItems(items []seed, regulation *RegulationData) ([]legacyItemGroup, error) {
	if regulation == nil {
		return nil, fmt.Errorf("regulation data is required")
	}
	itemsByID := make(map[uint32]seed, len(items))
	for _, item := range items {
		if _, duplicate := itemsByID[item.ID]; duplicate {
			return nil, fmt.Errorf("duplicate legacy item 0x%08X", item.ID)
		}
		itemsByID[item.ID] = item
	}

	canonicalByVariant := make(map[uint32]legacyVariantSeed, 3576)
	suppressedAffinityRows := make(map[uint32]struct{})
	if err := collectWeaponAffinityVariants(items, itemsByID, regulation, canonicalByVariant, suppressedAffinityRows); err != nil {
		return nil, err
	}
	if err := collectMissingWeaponAffinityVariants(itemsByID, regulation, canonicalByVariant); err != nil {
		return nil, err
	}
	if err := collectSpiritAshUpgradeVariants(items, itemsByID, regulation, canonicalByVariant); err != nil {
		return nil, err
	}

	groupsByCanonical := make(map[uint32]*legacyItemGroup, len(items)-len(canonicalByVariant))
	for _, item := range items {
		if _, suppressed := suppressedAffinityRows[item.ID]; suppressed {
			continue
		}
		if _, isVariant := canonicalByVariant[item.ID]; isVariant {
			continue
		}
		itemCopy := item
		groupsByCanonical[item.ID] = &legacyItemGroup{Canonical: itemCopy}
	}
	for variantID, variant := range canonicalByVariant {
		group, exists := groupsByCanonical[variant.CanonicalID]
		if !exists {
			return nil, fmt.Errorf(
				"variant 0x%08X references missing canonical item 0x%08X",
				variantID,
				variant.CanonicalID,
			)
		}
		group.Variants = append(group.Variants, variant)
	}

	groups := make([]legacyItemGroup, 0, len(groupsByCanonical))
	for _, group := range groupsByCanonical {
		sort.Slice(group.Variants, func(i, j int) bool {
			return group.Variants[i].Item.ID < group.Variants[j].Item.ID
		})
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Canonical.ID < groups[j].Canonical.ID
	})
	return groups, nil
}

func collectWeaponAffinityVariants(
	items []seed,
	itemsByID map[uint32]seed,
	regulation *RegulationData,
	variants map[uint32]legacyVariantSeed,
	suppressed map[uint32]struct{},
) error {
	for _, item := range items {
		if !legacyWeaponCategory(item.Category) {
			continue
		}
		identity, err := primaryRegulationForLegacyItem(item)
		if err != nil {
			return err
		}
		lookup, exists, err := regulation.LookupFamilyRow(
			identity.Family,
			RegulationTableRolePrimary,
			identity.RowID,
		)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("weapon item 0x%08X has no regulation row %d", item.ID, identity.RowID)
		}
		originRaw, ok := lookup.Row.Field("originEquipWep")
		if !ok {
			return fmt.Errorf("weapon row %d has no originEquipWep field", identity.RowID)
		}
		origin, err := parseRegulationUint32(originRaw)
		if err != nil {
			if originRaw == "-1" {
				continue
			}
			return fmt.Errorf("weapon row %d originEquipWep: %w", identity.RowID, err)
		}
		if origin >= identity.RowID {
			continue
		}
		delta := identity.RowID - origin
		if delta%100 != 0 {
			continue
		}
		affinityOffset := delta / 100
		affinity, allowed := affinityByRegulationOffset[affinityOffset]
		if !allowed {
			continue
		}
		canonical, exists := itemsByID[origin]
		if !exists || !legacyWeaponCategory(canonical.Category) {
			return fmt.Errorf(
				"weapon affinity row %d references missing legacy origin %d",
				identity.RowID,
				origin,
			)
		}
		canonicalLookup, canonicalExists, err := regulation.LookupFamilyRow(
			RegulationFamilyWeapon,
			RegulationTableRolePrimary,
			origin,
		)
		if err != nil {
			return err
		}
		if !canonicalExists {
			return fmt.Errorf("weapon affinity row %d references missing canonical regulation row %d", identity.RowID, origin)
		}
		permitted, err := canonicalAllowsAffinityVariants(canonicalLookup.Row)
		if err != nil {
			return err
		}
		if !permitted {
			suppressed[item.ID] = struct{}{}
			continue
		}
		variants[item.ID] = legacyVariantSeed{
			Item:        item,
			Kind:        legacyVariantAffinity,
			CanonicalID: canonical.ID,
			Affinity:    affinity,
		}
	}
	return nil
}

func collectMissingWeaponAffinityVariants(
	itemsByID map[uint32]seed,
	regulation *RegulationData,
	variants map[uint32]legacyVariantSeed,
) error {
	table, exists := regulation.Table(RegulationTableWeapon)
	if !exists {
		return fmt.Errorf("regulation table %q is not loaded", RegulationTableWeapon)
	}
	for _, row := range table.Rows() {
		if _, alreadyMigrated := itemsByID[row.RowID]; alreadyMigrated {
			continue
		}
		originRaw, ok := row.Field("originEquipWep")
		if !ok {
			return fmt.Errorf("weapon row %d has no originEquipWep field", row.RowID)
		}
		origin, err := parseRegulationUint32(originRaw)
		if err != nil {
			if originRaw == "-1" {
				continue
			}
			return fmt.Errorf("weapon row %d originEquipWep: %w", row.RowID, err)
		}
		if origin >= row.RowID {
			continue
		}
		delta := row.RowID - origin
		if delta%100 != 0 {
			continue
		}
		affinity, allowed := affinityByRegulationOffset[delta/100]
		if !allowed {
			continue
		}
		canonical, exists := itemsByID[origin]
		if !exists || !legacyWeaponCategory(canonical.Category) {
			continue
		}
		canonicalLookup, canonicalExists, err := regulation.LookupFamilyRow(
			RegulationFamilyWeapon,
			RegulationTableRolePrimary,
			origin,
		)
		if err != nil {
			return err
		}
		if !canonicalExists {
			return fmt.Errorf("weapon affinity row %d references missing canonical regulation row %d", row.RowID, origin)
		}
		permitted, err := canonicalAllowsAffinityVariants(canonicalLookup.Row)
		if err != nil {
			return err
		}
		if !permitted {
			continue
		}
		variant := canonical
		variant.ID = row.RowID
		variant.HasLegacyItem = false
		variant.RegulationOnlyVariant = true
		variant.Acquisition = acquisitionSeed{}
		variant.Unlocks = nil
		variant.Links = linksSeed{}
		enrichLegacySeed(&variant)
		variants[variant.ID] = legacyVariantSeed{
			Item:        variant,
			Kind:        legacyVariantAffinity,
			CanonicalID: canonical.ID,
			Affinity:    affinity,
		}
	}
	return nil
}

func canonicalAllowsAffinityVariants(canonical ParameterRow) (bool, error) {
	return weaponCanChangeAffinity(canonical)
}

func collectSpiritAshUpgradeVariants(
	items []seed,
	itemsByID map[uint32]seed,
	regulation *RegulationData,
	variants map[uint32]legacyVariantSeed,
) error {
	successor := make(map[uint32]uint32, 840)
	predecessor := make(map[uint32]uint32, 840)
	for _, item := range items {
		if item.Category != "ashes" {
			continue
		}
		rowID := item.ID & 0x0FFFFFFF
		lookup, exists, err := regulation.LookupFamilyRow(
			RegulationFamilyGoods,
			RegulationTableRolePrimary,
			rowID,
		)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("spirit ash item 0x%08X has no regulation row %d", item.ID, rowID)
		}
		nextRaw, ok := lookup.Row.Field("reinforceGoodsId")
		if !ok {
			return fmt.Errorf("goods row %d has no reinforceGoodsId field", rowID)
		}
		if nextRaw == "-1" {
			continue
		}
		nextRowID, err := parseRegulationUint32(nextRaw)
		if err != nil {
			return fmt.Errorf("goods row %d reinforceGoodsId: %w", rowID, err)
		}
		nextItemID := uint32(0x40000000) | nextRowID
		nextItem, exists := itemsByID[nextItemID]
		if !exists || nextItem.Category != "ashes" {
			return fmt.Errorf(
				"spirit ash row %d reinforces to missing legacy ash %d",
				rowID,
				nextRowID,
			)
		}
		if previous, duplicate := predecessor[nextRowID]; duplicate {
			return fmt.Errorf(
				"spirit ash row %d has multiple predecessors %d and %d",
				nextRowID,
				previous,
				rowID,
			)
		}
		successor[rowID] = nextRowID
		predecessor[nextRowID] = rowID
	}

	for _, item := range items {
		if item.Category != "ashes" {
			continue
		}
		rowID := item.ID & 0x0FFFFFFF
		if _, hasPredecessor := predecessor[rowID]; !hasPredecessor {
			if _, hasSuccessor := successor[rowID]; !hasSuccessor {
				return fmt.Errorf("spirit ash row %d is not in a regulation upgrade chain", rowID)
			}
			if err := addSpiritAshChain(rowID, successor, itemsByID, variants); err != nil {
				return err
			}
		}
	}
	return nil
}

func addSpiritAshChain(
	rootRowID uint32,
	successor map[uint32]uint32,
	itemsByID map[uint32]seed,
	variants map[uint32]legacyVariantSeed,
) error {
	current := rootRowID
	seen := map[uint32]struct{}{current: {}}
	for level := uint8(1); level <= 10; level++ {
		next, exists := successor[current]
		if !exists {
			return fmt.Errorf("spirit ash chain %d stops before upgrade +%d", rootRowID, level)
		}
		if _, duplicate := seen[next]; duplicate {
			return fmt.Errorf("spirit ash chain %d contains cycle at row %d", rootRowID, next)
		}
		seen[next] = struct{}{}
		itemID := uint32(0x40000000) | next
		item, exists := itemsByID[itemID]
		if !exists || item.Category != "ashes" {
			return fmt.Errorf("spirit ash chain %d misses legacy row %d", rootRowID, next)
		}
		variants[itemID] = legacyVariantSeed{
			Item:         item,
			Kind:         legacyVariantUpgrade,
			CanonicalID:  uint32(0x40000000) | rootRowID,
			UpgradeLevel: level,
		}
		current = next
	}
	if next, exists := successor[current]; exists {
		return fmt.Errorf("spirit ash chain %d continues past +10 to %d", rootRowID, next)
	}
	return nil
}

func legacyWeaponCategory(category string) bool {
	switch category {
	case "melee_armaments", "ranged_and_catalysts", "shields", "arrows_and_bolts":
		return true
	default:
		return false
	}
}

func parseRegulationUint32(raw string) (uint32, error) {
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%q is not an unsigned decimal integer", raw)
	}
	return uint32(value), nil
}
