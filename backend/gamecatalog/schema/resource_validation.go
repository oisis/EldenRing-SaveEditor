package schema

import "fmt"

func ValidateResource(resource Resource, sources map[SourceID]struct{}) error {
	if resource.Key == "" {
		return fmt.Errorf("resource key is required")
	}
	switch resource.Kind {
	case ResourceKindItem:
		return validateItemResource(resource, sources)
	case ResourceKindColosseum:
		return validateColosseumResource(resource, sources)
	case ResourceKindRegion:
		return validateRegionResource(resource, sources)
	case ResourceKindSummoningPool:
		return validateSummoningPoolResource(resource, sources)
	case ResourceKindGrace:
		return validateGraceResource(resource, sources)
	default:
		return fmt.Errorf("resource %q: unsupported kind %q", resource.Key, resource.Kind)
	}
}

// validateSoleDocument rejects a resource that carries a document of any kind
// other than its own, so the union can never be served with a second document
// set. It replaces the pairwise checks the kinds used to repeat, which grew
// quadratically with every new kind while proving exactly this one rule. Each
// known kind calls it at the position its own pairwise checks used to hold, so
// an unknown kind still fails as an unsupported kind rather than as a foreign
// document.
func validateSoleDocument(resource Resource) error {
	present := []struct {
		kind    ResourceKind
		carried bool
	}{
		{ResourceKindItem, resource.Item != nil},
		{ResourceKindColosseum, resource.Colosseum != nil},
		{ResourceKindRegion, resource.Region != nil},
		{ResourceKindSummoningPool, resource.SummoningPool != nil},
		{ResourceKindGrace, resource.Grace != nil},
	}
	for _, document := range present {
		if document.carried && document.kind != resource.Kind {
			return fmt.Errorf("resource %q: %s resource must not carry a %s document",
				resource.Key, resource.Kind, document.kind)
		}
	}
	return nil
}

// validateSlugKey is the key rule every non-item kind shares: lowercase letters,
// digits and underscores only. An item key is hexadecimal and has its own rule.
func validateSlugKey(kind ResourceKind, key string) error {
	for _, character := range key {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') &&
			character != '_' {
			return fmt.Errorf(
				"resource %q: %s key must use lowercase letters, digits and underscores",
				key, kind,
			)
		}
	}
	return nil
}

func validateItemResource(resource Resource, sources map[SourceID]struct{}) error {
	// An item key is exactly eight uppercase hexadecimal characters and never
	// carries a kind prefix. The rule lives here so a catalog loaded from disk
	// is held to it too, not only the one the generator just produced.
	wellFormed := len(resource.Key) == 8
	for _, character := range resource.Key {
		if (character < '0' || character > '9') && (character < 'A' || character > 'F') {
			wellFormed = false
		}
	}
	if !wellFormed {
		return fmt.Errorf(
			"resource %q: item key must be exactly eight uppercase hexadecimal characters",
			resource.Key,
		)
	}
	if resource.Item == nil {
		return fmt.Errorf("resource %q: item document is required", resource.Key)
	}
	if err := validateSoleDocument(resource); err != nil {
		return err
	}
	if err := validateItemDocument(*resource.Item, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	return nil
}

// validateColosseumResource fails closed: an unknown name, an unknown or zero
// event flag, a missing document and a document of the wrong kind are all
// rejected, so a colosseum can never be served without both facts.
func validateColosseumResource(resource Resource, sources map[SourceID]struct{}) error {
	if err := validateSlugKey(ResourceKindColosseum, resource.Key); err != nil {
		return err
	}
	if err := validateSoleDocument(resource); err != nil {
		return err
	}
	if resource.Colosseum == nil {
		return fmt.Errorf("resource %q: colosseum document is required", resource.Key)
	}
	name := resource.Colosseum.Name
	if err := validateFact("colosseum.name", name, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !name.Known || name.Value == "" {
		return fmt.Errorf("resource %q: colosseum.name must be known and non-empty", resource.Key)
	}
	flag := resource.Colosseum.UnlockEventFlagID
	if err := validateFact("colosseum.unlockEventFlagID", flag, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !flag.Known || flag.Value == 0 {
		return fmt.Errorf(
			"resource %q: colosseum.unlockEventFlagID must be known and non-zero", resource.Key)
	}
	return nil
}

// validateRegionResource fails closed: an unknown or zero region ID, an unknown
// or empty name, an unknown or empty area and a missing document are all
// rejected, so a region can never be served without every fact it is matched and
// presented by.
func validateRegionResource(resource Resource, sources map[SourceID]struct{}) error {
	if err := validateSlugKey(ResourceKindRegion, resource.Key); err != nil {
		return err
	}
	if err := validateSoleDocument(resource); err != nil {
		return err
	}
	if resource.Region == nil {
		return fmt.Errorf("resource %q: region document is required", resource.Key)
	}
	regionID := resource.Region.RegionID
	if err := validateFact("region.regionID", regionID, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !regionID.Known || regionID.Value == 0 {
		return fmt.Errorf(
			"resource %q: region.regionID must be known and non-zero", resource.Key)
	}
	name := resource.Region.Name
	if err := validateFact("region.name", name, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !name.Known || name.Value == "" {
		return fmt.Errorf("resource %q: region.name must be known and non-empty", resource.Key)
	}
	area := resource.Region.Area
	if err := validateFact("region.area", area, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !area.Known || area.Value == "" {
		return fmt.Errorf("resource %q: region.area must be known and non-empty", resource.Key)
	}
	return nil
}

// validateSummoningPoolResource fails closed: an unknown or empty name, an
// unknown or empty region label, an unknown or zero activation flag, a flag
// outside the confirmed block 670 and a missing document are all rejected, so a
// pool can never be served without every fact it is presented and resolved by.
func validateSummoningPoolResource(resource Resource, sources map[SourceID]struct{}) error {
	if err := validateSlugKey(ResourceKindSummoningPool, resource.Key); err != nil {
		return err
	}
	if err := validateSoleDocument(resource); err != nil {
		return err
	}
	if resource.SummoningPool == nil {
		return fmt.Errorf("resource %q: summoning pool document is required", resource.Key)
	}
	name := resource.SummoningPool.Name
	if err := validateFact("summoningPool.name", name, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !name.Known || name.Value == "" {
		return fmt.Errorf("resource %q: summoningPool.name must be known and non-empty", resource.Key)
	}
	regionLabel := resource.SummoningPool.RegionLabel
	if err := validateFact("summoningPool.regionLabel", regionLabel, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !regionLabel.Known || regionLabel.Value == "" {
		return fmt.Errorf(
			"resource %q: summoningPool.regionLabel must be known and non-empty", resource.Key)
	}
	flag := resource.SummoningPool.ActivationEventFlagID
	if err := validateFact("summoningPool.activationEventFlagID", flag, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !flag.Known || flag.Value == 0 {
		return fmt.Errorf(
			"resource %q: summoningPool.activationEventFlagID must be known and non-zero",
			resource.Key)
	}
	if flag.Value < SummoningPoolFlagBlockFirst || flag.Value > SummoningPoolFlagBlockLast {
		return fmt.Errorf(
			"resource %q: summoningPool.activationEventFlagID %d lies outside the confirmed block %d..%d",
			resource.Key, flag.Value, SummoningPoolFlagBlockFirst, SummoningPoolFlagBlockLast)
	}
	return nil
}

// validateGraceResource fails closed: an unknown or empty name, an unknown or
// empty region label, an unknown or zero visit flag, a visit flag outside the
// blocks the curated table confirms, an unknown boss-arena fact, an unknown or
// unsupported dungeon type, an unknown door flag and a door flag on a grace that
// is behind no sealed dungeon entrance are all rejected, so a grace can never be
// served without every fact it is presented and resolved by.
func validateGraceResource(resource Resource, sources map[SourceID]struct{}) error {
	if err := validateSlugKey(ResourceKindGrace, resource.Key); err != nil {
		return err
	}
	if err := validateSoleDocument(resource); err != nil {
		return err
	}
	if resource.Grace == nil {
		return fmt.Errorf("resource %q: grace document is required", resource.Key)
	}
	name := resource.Grace.Name
	if err := validateFact("grace.name", name, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !name.Known || name.Value == "" {
		return fmt.Errorf("resource %q: grace.name must be known and non-empty", resource.Key)
	}
	regionLabel := resource.Grace.RegionLabel
	if err := validateFact("grace.regionLabel", regionLabel, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !regionLabel.Known || regionLabel.Value == "" {
		return fmt.Errorf(
			"resource %q: grace.regionLabel must be known and non-empty", resource.Key)
	}
	visit := resource.Grace.VisitEventFlagID
	if err := validateFact("grace.visitEventFlagID", visit, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !visit.Known || visit.Value == 0 {
		return fmt.Errorf(
			"resource %q: grace.visitEventFlagID must be known and non-zero", resource.Key)
	}
	if !IsConfirmedGraceFlag(visit.Value) {
		return fmt.Errorf(
			"resource %q: grace.visitEventFlagID %d lies outside the blocks the curated Graces table confirms",
			resource.Key, visit.Value)
	}
	bossArena := resource.Grace.BossArena
	if err := validateFact("grace.bossArena", bossArena, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !bossArena.Known {
		return fmt.Errorf("resource %q: grace.bossArena must be known", resource.Key)
	}
	dungeonType := resource.Grace.DungeonType
	if err := validateFact("grace.dungeonType", dungeonType, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !dungeonType.Known {
		return fmt.Errorf("resource %q: grace.dungeonType must be known", resource.Key)
	}
	switch dungeonType.Value {
	case GraceDungeonTypeNone, GraceDungeonTypeCatacomb, GraceDungeonTypeHeroGrave:
	default:
		return fmt.Errorf("resource %q: grace.dungeonType %q is not a confirmed value",
			resource.Key, dungeonType.Value)
	}
	door := resource.Grace.DoorEventFlagID
	if err := validateFact("grace.doorEventFlagID", door, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !door.Known {
		return fmt.Errorf("resource %q: grace.doorEventFlagID must be known", resource.Key)
	}
	// Zero is the confirmed value for "no dependent door", and only a grace of a
	// sealed-entrance dungeon family ever carries a non-zero one in the curated
	// table. A door flag on a regular grace would be data no record supports.
	if door.Value != 0 && dungeonType.Value == GraceDungeonTypeNone {
		return fmt.Errorf(
			"resource %q: grace.doorEventFlagID %d is set on a grace without a dungeon type",
			resource.Key, door.Value)
	}
	return nil
}
