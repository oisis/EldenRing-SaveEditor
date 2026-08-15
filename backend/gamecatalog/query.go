package gamecatalog

import (
	"errors"
	"fmt"
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type ItemView struct {
	Resource          schema.Resource
	OutgoingRelations []schema.Relation
	IncomingRelations []schema.Relation
	RelatedResources  []schema.Resource
}

// ResourceByKindAndKey resolves one resource by the exact (kind, key) pair. The
// kind is selected first and the key is matched only inside that kind; neither
// value is trimmed, normalised, or retried under another kind. The returned
// resource is an independent deep copy and carries no relations.
func (catalog *Catalog) ResourceByKindAndKey(
	kind schema.ResourceKind,
	key string,
) (schema.Resource, error) {
	resource, err := catalog.requireResource(kind, key)
	if err != nil {
		return schema.Resource{}, err
	}
	return cloneResource(resource), nil
}

// ResourceByKindKeyAndVariant resolves one item resource by the exact
// (kind, key) pair and selects either its base document or exactly one of the
// variants stored under that same pair. It is the single place that turns the
// shared (kind, key, variantID) input of the item endpoints into one document,
// so every caller selects a variant by the same rule.
//
// variantID is optional: a nil variantID selects the base ItemDocument, and a
// non-nil variantID selects only the stored variant of this resource whose
// ItemVariant.GameID is known and exactly equal to it. ItemVariant.GameID is
// the only variant identifier; nothing is derived from affinity, upgrade level
// or ID arithmetic, and no variant is ever synthesised. The base Item.GameID,
// an alias game ID and a variant that belongs to another resource are all
// unknown variants here, because the lookup never leaves the resolved resource
// and never consults the global game-ID index. Neither kind nor key is
// trimmed, normalised or retried under another kind, exactly as
// ResourceByKindAndKey resolves them.
//
// The result keeps the public identity (kind, key) of the resolved resource
// while Item.GameID and the document data are those of the selected variant.
// It is an independent deep copy, so mutating it never reaches the catalog.
func (catalog *Catalog) ResourceByKindKeyAndVariant(
	kind schema.ResourceKind,
	key string,
	variantID *uint32,
) (schema.Resource, error) {
	if catalog == nil {
		return schema.Resource{}, errors.New("game catalog is not loaded")
	}
	if kind == "" {
		return schema.Resource{}, errors.New("resource kind is required")
	}
	if kind != schema.ResourceKindItem {
		return schema.Resource{}, fmt.Errorf(
			"resource kind %q carries no item document; only kind %q is supported",
			kind,
			schema.ResourceKindItem,
		)
	}
	if key == "" {
		return schema.Resource{}, errors.New("resource key is required")
	}
	stored, err := catalog.requireResource(kind, key)
	if err != nil {
		return schema.Resource{}, err
	}
	if stored.Item == nil {
		return schema.Resource{}, fmt.Errorf(
			"resource kind %q key %q has no item document",
			kind,
			key,
		)
	}
	resource := cloneResource(stored)
	if variantID == nil {
		return resource, nil
	}
	for _, variant := range resource.Item.Variants {
		if !variant.GameID.Known || variant.GameID.Value != *variantID {
			continue
		}
		materialized := schema.MaterializeVariant(*resource.Item, variant)
		resource.Item = &materialized
		return resource, nil
	}
	return schema.Resource{}, fmt.Errorf(
		"unknown variant ID 0x%08X for resource kind %q key %q",
		*variantID,
		kind,
		key,
	)
}

// CapabilitySummary carries the two flags a list needs to decide whether a
// capability applies, without the rules and the provenance behind them.
type CapabilitySummary struct {
	Known   bool
	Enabled bool
}

// ResourceSummary is the value-only view of one resource: the fields a list or a
// picker reads, and nothing else. It holds no pointer, map or slice, so it copies
// no mutable catalog state and cannot be used to reach a stored document.
type ResourceSummary struct {
	Kind          schema.ResourceKind
	Key           string
	FamilyKnown   bool
	Family        schema.ItemFamily
	NameKnown     bool
	Name          string
	Upgrade       CapabilitySummary
	Infusion      CapabilitySummary
	AshOfWarMount CapabilitySummary
	Stack         CapabilitySummary
	Equipment     CapabilitySummary
}

// ResourceSummaries returns one summary per stored resource, ordered by kind and
// only then by key, which is the same deterministic order relation derivation
// uses. It copies scalars only, so a caller that lists resources never pays for
// deep-copying variants, provenance and capability rules it would discard. It
// deliberately offers no filtering, search or paging: selecting and slicing
// resources belongs to the endpoint that asks for them.
func (catalog *Catalog) ResourceSummaries() []ResourceSummary {
	stored := catalog.sortedResources()
	summaries := make([]ResourceSummary, 0, len(stored))
	for _, resource := range stored {
		summary := ResourceSummary{Kind: resource.Kind, Key: resource.Key}
		switch {
		case resource.Item != nil:
			summary.FamilyKnown = resource.Item.Family.Known
			summary.Family = resource.Item.Family.Value
			summary.NameKnown = resource.Item.Presentation.Name.Known
			summary.Name = resource.Item.Presentation.Name.Value

			capabilities := resource.Item.Capabilities
			summary.Upgrade = summariseCapability(capabilities.Upgrade)
			summary.Infusion = summariseCapability(capabilities.Infusion)
			summary.AshOfWarMount = summariseCapability(capabilities.AshOfWarMount)
			summary.Stack = summariseCapability(capabilities.Stack)
			summary.Equipment = summariseCapability(capabilities.Equipment)
		case resource.Colosseum != nil:
			// A colosseum has a name but no family and no capability, so those
			// stay zero and no filter built on them can ever match it.
			summary.NameKnown = resource.Colosseum.Name.Known
			summary.Name = resource.Colosseum.Name.Value
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

// summariseCapability drops the rules pointer and the provenance and keeps the
// two flags. Deciding what a known-and-enabled pair means belongs to the caller.
func summariseCapability[T any](capability schema.Capability[T]) CapabilitySummary {
	return CapabilitySummary{Known: capability.Known, Enabled: capability.Enabled}
}

// RelationsByKindAndKey returns the relations of one resource resolved by the
// exact (kind, key) pair. The kind is resolved first and the key is matched only
// inside that kind, so an unknown kind and an unknown key stay the two distinct
// errors ResourceByKindAndKey reports. Both slices are independent copies in
// catalog order, so the internal relation maps are never exposed, and a known
// resource without relations returns two empty, non-nil slices. The documents of
// the related resources are not part of the result.
func (catalog *Catalog) RelationsByKindAndKey(
	kind schema.ResourceKind,
	key string,
) (outgoing []schema.Relation, incoming []schema.Relation, err error) {
	resource, err := catalog.requireResource(kind, key)
	if err != nil {
		return nil, nil, err
	}
	ref := resource.Ref()
	return append([]schema.Relation{}, catalog.outgoing[ref]...),
		append([]schema.Relation{}, catalog.incoming[ref]...),
		nil
}

// requireResource is the single place that turns a (kind, key) pair into the
// stored resource, so every getter built on that pair reports the same two
// failures with the same wording. The returned resource is the stored value and
// must never leave the package without a clone.
func (catalog *Catalog) requireResource(
	kind schema.ResourceKind,
	key string,
) (schema.Resource, error) {
	byKey, exists := catalog.byKind[kind]
	if !exists {
		return schema.Resource{}, fmt.Errorf("unknown resource kind %q", kind)
	}
	resource, exists := byKey[key]
	if !exists {
		return schema.Resource{}, fmt.Errorf("unknown resource key %q in kind %q", key, kind)
	}
	return resource, nil
}

func (catalog *Catalog) resourceByRef(ref schema.ResourceRef) (schema.Resource, bool) {
	byKey, exists := catalog.byKind[ref.Kind]
	if !exists {
		return schema.Resource{}, false
	}
	resource, exists := byKey[ref.Key]
	if !exists {
		return schema.Resource{}, false
	}
	return cloneResource(resource), true
}

func (catalog *Catalog) ItemByGameID(gameID uint32) (schema.Resource, bool) {
	ref, exists := catalog.byItemGameID[gameID]
	if !exists {
		resource, _, matches := catalog.weaponUpgradeAnchor(gameID)
		if matches != 1 {
			return schema.Resource{}, false
		}
		resource.Item.GameID.Value = gameID
		return resource, true
	}
	resource, exists := catalog.resourceByRef(ref)
	if !exists || resource.Item == nil || resource.Item.GameID.Value == gameID {
		return resource, exists
	}
	for _, variant := range resource.Item.Variants {
		if !variant.GameID.Known || variant.GameID.Value != gameID {
			continue
		}
		materialized := schema.MaterializeVariant(*resource.Item, variant)
		resource.Item = &materialized
		return resource, true
	}
	for _, alias := range resource.Item.Aliases {
		if alias.GameID.Known && alias.GameID.Value == gameID {
			return resource, true
		}
	}
	return schema.Resource{}, false
}

// WeaponUpgradeTarget resolves one exact save-side weapon ID to the canonical
// catalog resource and derives the ID of the requested level without changing
// its affinity anchor. The catalog owns this arithmetic because upgrade rules
// and stored affinity variants are catalog semantics, not save layout.
func (catalog *Catalog) WeaponUpgradeTarget(
	currentGameID uint32,
	upgradeLevel uint8,
) (uint32, error) {
	if catalog == nil {
		return 0, errors.New("game catalog is not loaded")
	}
	resource, currentLevel, matches := catalog.weaponUpgradeAnchor(currentGameID)
	if matches == 0 {
		return 0, fmt.Errorf(
			"game ID 0x%08X is not a catalog weapon upgrade", currentGameID)
	}
	if matches != 1 {
		return 0, fmt.Errorf(
			"game ID 0x%08X matches more than one weapon upgrade range", currentGameID)
	}

	upgrade := resource.Item.Capabilities.Upgrade
	if !upgrade.Known {
		return 0, fmt.Errorf(
			"weapon 0x%08X has an unknown upgrade capability", currentGameID)
	}
	if !upgrade.Enabled {
		return 0, fmt.Errorf("weapon 0x%08X cannot be upgraded", currentGameID)
	}
	if upgrade.Rules == nil {
		return 0, fmt.Errorf("weapon 0x%08X carries no upgrade rules", currentGameID)
	}
	if upgrade.Rules.Model != schema.UpgradeModelStandard &&
		upgrade.Rules.Model != schema.UpgradeModelSomber {
		return 0, fmt.Errorf(
			"weapon 0x%08X uses unsupported upgrade model %q", currentGameID, upgrade.Rules.Model)
	}
	if upgradeLevel > upgrade.Rules.MaxLevel {
		return 0, fmt.Errorf(
			"upgradeLevel %d exceeds the catalog limit of %d for weapon 0x%08X",
			upgradeLevel, upgrade.Rules.MaxLevel, currentGameID)
	}

	anchor := currentGameID - uint32(currentLevel)
	return anchor + uint32(upgradeLevel), nil
}

// SpiritAshUpgradeTarget resolves one exact save-side Spirit Ash ID and returns
// the stored catalog variant for the requested level. Unlike weapon upgrades,
// Spirit Ash chains are explicit variants and are never derived arithmetically.
func (catalog *Catalog) SpiritAshUpgradeTarget(
	currentGameID uint32,
	upgradeLevel uint8,
) (uint32, error) {
	if catalog == nil {
		return 0, errors.New("game catalog is not loaded")
	}
	ref, exists := catalog.byItemGameID[currentGameID]
	if !exists {
		return 0, fmt.Errorf("game ID 0x%08X is not a catalog Spirit Ash", currentGameID)
	}
	resource, err := catalog.requireResource(ref.Kind, ref.Key)
	if err != nil {
		return 0, err
	}
	item := resource.Item
	if item == nil || !item.Family.Known || item.Family.Value != schema.ItemFamilySpiritAsh {
		return 0, fmt.Errorf("game ID 0x%08X is not a catalog Spirit Ash", currentGameID)
	}

	upgrade := item.Capabilities.Upgrade
	if !upgrade.Known {
		return 0, fmt.Errorf("Spirit Ash 0x%08X has an unknown upgrade capability", currentGameID)
	}
	if !upgrade.Enabled {
		return 0, fmt.Errorf("Spirit Ash 0x%08X cannot be upgraded", currentGameID)
	}
	if upgrade.Rules == nil {
		return 0, fmt.Errorf("Spirit Ash 0x%08X carries no upgrade rules", currentGameID)
	}
	if upgrade.Rules.Model != schema.UpgradeModelGraveGlovewort &&
		upgrade.Rules.Model != schema.UpgradeModelGhostGlovewort {
		return 0, fmt.Errorf(
			"Spirit Ash 0x%08X uses unsupported upgrade model %q",
			currentGameID, upgrade.Rules.Model)
	}
	if upgradeLevel > upgrade.Rules.MaxLevel {
		return 0, fmt.Errorf(
			"upgradeLevel %d exceeds the catalog limit of %d for Spirit Ash 0x%08X",
			upgradeLevel, upgrade.Rules.MaxLevel, currentGameID)
	}

	currentMatches := 0
	if item.GameID.Known && item.GameID.Value == currentGameID {
		currentMatches++
	}
	for _, variant := range item.Variants {
		if variant.GameID.Known && variant.GameID.Value == currentGameID &&
			variant.Kind.Known && variant.Kind.Value == schema.ItemVariantUpgrade &&
			variant.UpgradeLevel.Known {
			currentMatches++
		}
	}
	if currentMatches != 1 {
		return 0, fmt.Errorf(
			"Spirit Ash game ID 0x%08X has %d matching upgrade identities, want exactly 1",
			currentGameID, currentMatches)
	}

	if upgradeLevel == 0 {
		return item.GameID.Value, nil
	}
	target, matches := uint32(0), 0
	for _, variant := range item.Variants {
		if variant.Kind.Known && variant.Kind.Value == schema.ItemVariantUpgrade &&
			variant.UpgradeLevel.Known && variant.UpgradeLevel.Value == upgradeLevel &&
			variant.GameID.Known {
			target, matches = variant.GameID.Value, matches+1
		}
	}
	if matches != 1 {
		return 0, fmt.Errorf(
			"Spirit Ash 0x%08X has %d stored variants for upgradeLevel %d, want exactly 1",
			currentGameID, matches, upgradeLevel)
	}
	return target, nil
}

// WeaponInfusionTarget resolves one exact save-side weapon ID and derives the
// ID of the requested affinity while preserving its current upgrade level.
func (catalog *Catalog) WeaponInfusionTarget(
	currentGameID uint32,
	affinity schema.Affinity,
) (uint32, uint8, error) {
	if catalog == nil {
		return 0, 0, errors.New("game catalog is not loaded")
	}
	resource, currentLevel, matches := catalog.weaponUpgradeAnchor(currentGameID)
	if matches == 0 {
		return 0, 0, fmt.Errorf(
			"game ID 0x%08X is not a catalog weapon infusion", currentGameID)
	}
	if matches != 1 {
		return 0, 0, fmt.Errorf(
			"game ID 0x%08X matches more than one weapon infusion range", currentGameID)
	}

	stored, err := catalog.requireResource(resource.Kind, resource.Key)
	if err != nil {
		return 0, 0, err
	}
	infusion := stored.Item.Capabilities.Infusion
	if !infusion.Known {
		return 0, 0, fmt.Errorf(
			"weapon 0x%08X has an unknown infusion capability", currentGameID)
	}
	if !infusion.Enabled {
		return 0, 0, fmt.Errorf("weapon 0x%08X cannot be infused", currentGameID)
	}
	if infusion.Rules == nil {
		return 0, 0, fmt.Errorf("weapon 0x%08X carries no infusion rules", currentGameID)
	}
	allowed := false
	for _, candidate := range infusion.Rules.AllowedAffinities {
		if candidate == affinity {
			allowed = true
			break
		}
	}
	if !allowed {
		return 0, 0, fmt.Errorf(
			"affinity %q is not allowed for weapon 0x%08X", affinity, currentGameID)
	}

	currentAnchor := currentGameID - uint32(currentLevel)
	if currentAnchor != stored.Item.GameID.Value {
		currentMatches := 0
		for _, variant := range stored.Item.Variants {
			if variant.GameID.Known && variant.GameID.Value == currentAnchor &&
				variant.Affinity.Known && variant.UpgradeLevel.Known &&
				variant.UpgradeLevel.Value == 0 {
				currentMatches++
			}
		}
		if currentMatches != 1 {
			return 0, 0, fmt.Errorf(
				"weapon 0x%08X has %d matching current affinity anchors, want exactly 1",
				currentGameID, currentMatches)
		}
	}

	targetAnchor := stored.Item.GameID.Value
	targetUpgrade := stored.Item.Capabilities.Upgrade
	if affinity != schema.AffinityStandard {
		targetMatches := 0
		for _, variant := range stored.Item.Variants {
			if variant.GameID.Known && variant.Affinity.Known &&
				variant.Affinity.Value == affinity && variant.UpgradeLevel.Known &&
				variant.UpgradeLevel.Value == 0 {
				targetAnchor = variant.GameID.Value
				targetUpgrade = variant.Data.Capabilities.Upgrade
				targetMatches++
			}
		}
		if targetMatches != 1 {
			return 0, 0, fmt.Errorf(
				"weapon 0x%08X has %d catalog anchors for affinity %q, want exactly 1",
				currentGameID, targetMatches, affinity)
		}
	}
	if !targetUpgrade.Known || !targetUpgrade.Enabled || targetUpgrade.Rules == nil ||
		currentLevel > targetUpgrade.Rules.MaxLevel {
		return 0, 0, fmt.Errorf(
			"affinity %q cannot preserve upgrade level %d for weapon 0x%08X",
			affinity, currentLevel, currentGameID)
	}
	return targetAnchor + uint32(currentLevel), currentLevel, nil
}

// weaponUpgradeAnchor finds the one base or stored affinity anchor whose
// catalog range contains gameID. It deliberately scans only after the exact
// game-ID index misses, so normal lookups remain constant-time and the catalog
// does not carry a second, expanded index of every possible upgrade level.
func (catalog *Catalog) weaponUpgradeAnchor(gameID uint32) (schema.Resource, uint8, int) {
	if catalog == nil {
		return schema.Resource{}, 0, 0
	}
	var match schema.Resource
	var level uint8
	matches := 0
	for _, stored := range catalog.byKind[schema.ResourceKindItem] {
		if stored.Item == nil || !stored.Item.Family.Known ||
			stored.Item.Family.Value != schema.ItemFamilyWeapon {
			continue
		}
		matchCandidate := func(anchor uint32, upgrade schema.Capability[schema.UpgradeRules]) bool {
			maxLevel := uint8(0)
			if upgrade.Known && upgrade.Enabled && upgrade.Rules != nil {
				maxLevel = upgrade.Rules.MaxLevel
			}
			return gameID >= anchor && gameID-anchor <= uint32(maxLevel)
		}
		if matchCandidate(stored.Item.GameID.Value, stored.Item.Capabilities.Upgrade) {
			match = cloneResource(stored)
			level = uint8(gameID - stored.Item.GameID.Value)
			matches++
		}
		for _, variant := range stored.Item.Variants {
			if !variant.GameID.Known || !variant.UpgradeLevel.Known || variant.UpgradeLevel.Value != 0 {
				continue
			}
			if !matchCandidate(variant.GameID.Value, variant.Data.Capabilities.Upgrade) {
				continue
			}
			match = cloneResource(stored)
			materialized := schema.MaterializeVariant(*match.Item, variant)
			match.Item = &materialized
			level = uint8(gameID - variant.GameID.Value)
			matches++
		}
	}
	return match, level, matches
}

func (catalog *Catalog) ItemViewByGameID(gameID uint32) (ItemView, bool) {
	resource, exists := catalog.ItemByGameID(gameID)
	if !exists {
		return ItemView{}, false
	}

	ref := resource.Ref()
	outgoing := cloneRelations(catalog.outgoing[ref])
	incoming := cloneRelations(catalog.incoming[ref])
	relatedRefs := make(map[schema.ResourceRef]struct{}, len(outgoing)+len(incoming))
	for _, relation := range outgoing {
		relatedRefs[relation.To] = struct{}{}
	}
	for _, relation := range incoming {
		relatedRefs[relation.From] = struct{}{}
	}
	refs := make([]schema.ResourceRef, 0, len(relatedRefs))
	for relatedRef := range relatedRefs {
		refs = append(refs, relatedRef)
	}
	sortResourceRefs(refs)

	related := make([]schema.Resource, 0, len(refs))
	for _, relatedRef := range refs {
		relatedResource, found := catalog.resourceByRef(relatedRef)
		if !found {
			continue
		}
		related = append(related, relatedResource)
	}
	return ItemView{
		Resource:          resource,
		OutgoingRelations: outgoing,
		IncomingRelations: incoming,
		RelatedResources:  related,
	}, true
}

// sortResourceRefs orders references by kind and only then by key, which is the
// deterministic order every catalog-derived list uses.
func sortResourceRefs(refs []schema.ResourceRef) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Kind != refs[j].Kind {
			return refs[i].Kind < refs[j].Kind
		}
		return refs[i].Key < refs[j].Key
	})
}
