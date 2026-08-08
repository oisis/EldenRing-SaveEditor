package gamecatalog

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (catalog *Catalog) deriveRelations(sources map[schema.SourceID]struct{}) error {
	if err := catalog.deriveRequiredContainerRelations(sources); err != nil {
		return err
	}
	weapons, ashes := catalog.itemsForCompatibility()
	for _, weapon := range weapons {
		mount := weapon.Item.Capabilities.AshOfWarMount
		if !mount.Known || !mount.Enabled {
			continue
		}
		bit := mount.Rules.CompatibilityBit
		if bit >= 64 {
			return fmt.Errorf("resource %q: Ash of War compatibility bit %d is out of range", weapon.Key, bit)
		}
		for _, ash := range ashes {
			mask := ash.Item.AshOfWar.CompatibilityMask
			if !mask.Known || mask.Value&(uint64(1)<<bit) == 0 {
				continue
			}
			relation := schema.Relation{
				From:       weapon.Ref(),
				To:         ash.Ref(),
				Kind:       schema.RelationCompatibleWithAshOfWar,
				Provenance: mask.Provenance,
			}
			if err := schema.ValidateRelation(relation, sources); err != nil {
				return fmt.Errorf("derived relation %q -> %q: %w", weapon.Key, ash.Key, err)
			}
			catalog.outgoing[weapon.Ref()] = append(catalog.outgoing[weapon.Ref()], relation)
			catalog.incoming[ash.Ref()] = append(catalog.incoming[ash.Ref()], relation)
		}
	}
	return nil
}

func (catalog *Catalog) deriveRequiredContainerRelations(sources map[schema.SourceID]struct{}) error {
	for _, resource := range catalog.sortedResources() {
		required := resource.Item.Acquisition.RequiredContainerID
		if !required.Known {
			continue
		}
		target, exists := catalog.byItemGameID[required.Value]
		if !exists {
			return fmt.Errorf(
				"resource %q: required container item 0x%08X is missing",
				resource.Key,
				required.Value,
			)
		}
		relation := schema.Relation{
			From:       resource.Ref(),
			To:         target,
			Kind:       schema.RelationRequiresContainer,
			Provenance: required.Provenance,
		}
		if err := schema.ValidateRelation(relation, sources); err != nil {
			return fmt.Errorf("derived required-container relation for %q: %w", resource.Key, err)
		}
		catalog.outgoing[resource.Ref()] = append(catalog.outgoing[resource.Ref()], relation)
		catalog.incoming[target] = append(catalog.incoming[target], relation)
	}
	return nil
}

func (catalog *Catalog) itemsForCompatibility() ([]schema.Resource, []schema.Resource) {
	weapons := make([]schema.Resource, 0)
	ashes := make([]schema.Resource, 0)
	for _, resource := range catalog.sortedResources() {
		switch resource.Item.Family.Value {
		case schema.ItemFamilyWeapon:
			weapons = append(weapons, resource)
		case schema.ItemFamilyAshOfWar:
			ashes = append(ashes, resource)
		}
	}
	return weapons, ashes
}

// sortedResources returns every resource ordered by kind and only then by key,
// so relation derivation never depends on Go map iteration order.
func (catalog *Catalog) sortedResources() []schema.Resource {
	refs := make([]schema.ResourceRef, 0, catalog.ResourceCount())
	for kind, byKey := range catalog.byKind {
		for key := range byKey {
			refs = append(refs, schema.ResourceRef{Kind: kind, Key: key})
		}
	}
	sortResourceRefs(refs)

	resources := make([]schema.Resource, 0, len(refs))
	for _, ref := range refs {
		resources = append(resources, catalog.byKind[ref.Kind][ref.Key])
	}
	return resources
}
