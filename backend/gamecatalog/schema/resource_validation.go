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
	case ResourceKindBoss:
		return validateBossResource(resource, sources)
	case ResourceKindMapRegion:
		return validateMapRegionResource(resource, sources)
	case ResourceKindTutorial:
		return validateTutorialResource(resource, sources)
	case ResourceKindQuest:
		return validateQuestResource(resource, sources)
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
		{ResourceKindBoss, resource.Boss != nil},
		{ResourceKindMapRegion, resource.MapRegion != nil},
		{ResourceKindTutorial, resource.Tutorial != nil},
		{ResourceKindQuest, resource.Quest != nil},
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

// validateBossResource fails closed: an unknown or empty name, an unknown or
// empty region label, an unknown or unsupported encounter type, an unknown
// remembrance fact, an unknown or zero defeat flag and a defeat flag outside the
// one block the curated table confirms are all rejected, so a boss can never be
// served without every fact it is presented and resolved by.
func validateBossResource(resource Resource, sources map[SourceID]struct{}) error {
	if err := validateSlugKey(ResourceKindBoss, resource.Key); err != nil {
		return err
	}
	if err := validateSoleDocument(resource); err != nil {
		return err
	}
	if resource.Boss == nil {
		return fmt.Errorf("resource %q: boss document is required", resource.Key)
	}
	name := resource.Boss.Name
	if err := validateFact("boss.name", name, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !name.Known || name.Value == "" {
		return fmt.Errorf("resource %q: boss.name must be known and non-empty", resource.Key)
	}
	regionLabel := resource.Boss.RegionLabel
	if err := validateFact("boss.regionLabel", regionLabel, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !regionLabel.Known || regionLabel.Value == "" {
		return fmt.Errorf(
			"resource %q: boss.regionLabel must be known and non-empty", resource.Key)
	}
	encounterType := resource.Boss.EncounterType
	if err := validateFact("boss.encounterType", encounterType, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !encounterType.Known {
		return fmt.Errorf("resource %q: boss.encounterType must be known", resource.Key)
	}
	switch encounterType.Value {
	case BossEncounterTypeMain, BossEncounterTypeField:
	default:
		return fmt.Errorf("resource %q: boss.encounterType %q is not a confirmed value",
			resource.Key, encounterType.Value)
	}
	remembrance := resource.Boss.Remembrance
	if err := validateFact("boss.remembrance", remembrance, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !remembrance.Known {
		return fmt.Errorf("resource %q: boss.remembrance must be known", resource.Key)
	}
	defeat := resource.Boss.DefeatEventFlagID
	if err := validateFact("boss.defeatEventFlagID", defeat, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !defeat.Known || defeat.Value == 0 {
		return fmt.Errorf(
			"resource %q: boss.defeatEventFlagID must be known and non-zero", resource.Key)
	}
	if !IsConfirmedBossFlag(defeat.Value) {
		return fmt.Errorf(
			"resource %q: boss.defeatEventFlagID %d lies outside block %d, the only block the curated Bosses table confirms",
			resource.Key, defeat.Value, bossFlagBlock)
	}
	return nil
}

// validateMapRegionResource fails closed: an unknown or empty name, an unknown
// or empty area label, an unknown or zero visibility flag and a visibility flag
// outside the one block the curated table confirms are all rejected, so a map
// region can never be served without every fact it is presented and resolved by.
func validateMapRegionResource(resource Resource, sources map[SourceID]struct{}) error {
	if err := validateSlugKey(ResourceKindMapRegion, resource.Key); err != nil {
		return err
	}
	if err := validateSoleDocument(resource); err != nil {
		return err
	}
	if resource.MapRegion == nil {
		return fmt.Errorf("resource %q: map region document is required", resource.Key)
	}
	name := resource.MapRegion.Name
	if err := validateFact("mapRegion.name", name, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !name.Known || name.Value == "" {
		return fmt.Errorf("resource %q: mapRegion.name must be known and non-empty", resource.Key)
	}
	areaLabel := resource.MapRegion.AreaLabel
	if err := validateFact("mapRegion.areaLabel", areaLabel, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !areaLabel.Known || areaLabel.Value == "" {
		return fmt.Errorf(
			"resource %q: mapRegion.areaLabel must be known and non-empty", resource.Key)
	}
	visible := resource.MapRegion.VisibleEventFlagID
	if err := validateFact("mapRegion.visibleEventFlagID", visible, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !visible.Known || visible.Value == 0 {
		return fmt.Errorf(
			"resource %q: mapRegion.visibleEventFlagID must be known and non-zero", resource.Key)
	}
	if !IsConfirmedMapRegionFlag(visible.Value) {
		return fmt.Errorf(
			"resource %q: mapRegion.visibleEventFlagID %d lies outside block %d, the only block the curated map visibility table confirms",
			resource.Key, visible.Value, mapRegionFlagBlock)
	}
	return nil
}

// validateTutorialResource requires the stable TutorialParam identity and the
// official non-empty title used to present the resource. Untitled parameter
// rows are not converted into guessed public resources by the generator.
func validateTutorialResource(resource Resource, sources map[SourceID]struct{}) error {
	if err := validateSlugKey(ResourceKindTutorial, resource.Key); err != nil {
		return err
	}
	if err := validateSoleDocument(resource); err != nil {
		return err
	}
	if resource.Tutorial == nil {
		return fmt.Errorf("resource %q: tutorial document is required", resource.Key)
	}
	tutorialID := resource.Tutorial.TutorialID
	if err := validateFact("tutorial.tutorialID", tutorialID, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !tutorialID.Known || tutorialID.Value == 0 {
		return fmt.Errorf("resource %q: tutorial.tutorialID must be known and non-zero", resource.Key)
	}
	if resource.Key != fmt.Sprintf("%d", tutorialID.Value) {
		return fmt.Errorf(
			"resource %q: tutorial key must equal tutorial.tutorialID %d",
			resource.Key, tutorialID.Value)
	}
	title := resource.Tutorial.Title
	if err := validateFact("tutorial.title", title, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !title.Known || title.Value == "" {
		return fmt.Errorf("resource %q: tutorial.title must be known and non-empty", resource.Key)
	}
	return nil
}

// validateQuestResource fails closed: an unknown or empty name, missing steps,
// invalid step keys, duplicate step keys, unknown step descriptions or empty
// flag lists are all rejected.
func validateQuestResource(resource Resource, sources map[SourceID]struct{}) error {
	if err := validateSlugKey(ResourceKindQuest, resource.Key); err != nil {
		return err
	}
	if err := validateSoleDocument(resource); err != nil {
		return err
	}
	if resource.Quest == nil {
		return fmt.Errorf("resource %q: quest document is required", resource.Key)
	}
	name := resource.Quest.Name
	if err := validateFact("quest.name", name, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	if !name.Known || name.Value == "" {
		return fmt.Errorf("resource %q: quest.name must be known and non-empty", resource.Key)
	}
	if len(resource.Quest.Steps) == 0 {
		return fmt.Errorf("resource %q: quest must contain at least one supported step", resource.Key)
	}

	seenStepKeys := make(map[string]struct{}, len(resource.Quest.Steps))
	for stepIdx, step := range resource.Quest.Steps {
		if err := validateSlugKey("quest_step", step.Key); err != nil {
			return fmt.Errorf("resource %q step %d: %w", resource.Key, stepIdx, err)
		}
		if _, duplicate := seenStepKeys[step.Key]; duplicate {
			return fmt.Errorf("resource %q: duplicate step key %q", resource.Key, step.Key)
		}
		seenStepKeys[step.Key] = struct{}{}

		if err := validateFact("quest.step.description", step.Description, sources); err != nil {
			return fmt.Errorf("resource %q step %q: %w", resource.Key, step.Key, err)
		}
		if !step.Description.Known || step.Description.Value == "" {
			return fmt.Errorf("resource %q step %q: quest.step.description must be known and non-empty",
				resource.Key, step.Key)
		}

		if err := validateFact("quest.step.location", step.Location, sources); err != nil {
			return fmt.Errorf("resource %q step %q: %w", resource.Key, step.Key, err)
		}

		if len(step.Flags) == 0 {
			return fmt.Errorf("resource %q step %q: flags list must not be empty",
				resource.Key, step.Key)
		}

		seenFlags := make(map[uint32]struct{}, len(step.Flags))
		for _, flag := range step.Flags {
			if flag.ID == 0 {
				return fmt.Errorf("resource %q step %q: flag ID must be non-zero",
					resource.Key, step.Key)
			}
			if _, duplicate := seenFlags[flag.ID]; duplicate {
				return fmt.Errorf("resource %q step %q: duplicate flag ID %d in canonical plan",
					resource.Key, step.Key, flag.ID)
			}
			seenFlags[flag.ID] = struct{}{}
		}
	}
	return nil
}
