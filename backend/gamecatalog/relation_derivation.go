package gamecatalog

import (
	"fmt"
	"sort"

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
				From:       weapon.ID,
				To:         ash.ID,
				Kind:       schema.RelationCompatibleWithAshOfWar,
				Provenance: mask.Provenance,
			}
			if err := schema.ValidateRelation(relation, sources); err != nil {
				return fmt.Errorf("derived relation %q -> %q: %w", weapon.Key, ash.Key, err)
			}
			catalog.outgoing[weapon.ID] = append(catalog.outgoing[weapon.ID], relation)
			catalog.incoming[ash.ID] = append(catalog.incoming[ash.ID], relation)
		}
	}
	return nil
}

func (catalog *Catalog) deriveRequiredContainerRelations(sources map[schema.SourceID]struct{}) error {
	resources := make([]schema.Resource, 0, len(catalog.byID))
	for _, resource := range catalog.byID {
		resources = append(resources, resource)
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ID < resources[j].ID
	})
	for _, resource := range resources {
		required := resource.Item.Acquisition.RequiredContainerID
		if !required.Known {
			continue
		}
		targetID, exists := catalog.byItemGameID[required.Value]
		if !exists {
			return fmt.Errorf(
				"resource %q: required container item 0x%08X is missing",
				resource.Key,
				required.Value,
			)
		}
		relation := schema.Relation{
			From:       resource.ID,
			To:         targetID,
			Kind:       schema.RelationRequiresContainer,
			Provenance: required.Provenance,
		}
		if err := schema.ValidateRelation(relation, sources); err != nil {
			return fmt.Errorf("derived required-container relation for %q: %w", resource.Key, err)
		}
		catalog.outgoing[resource.ID] = append(catalog.outgoing[resource.ID], relation)
		catalog.incoming[targetID] = append(catalog.incoming[targetID], relation)
	}
	return nil
}

func (catalog *Catalog) itemsForCompatibility() ([]schema.Resource, []schema.Resource) {
	weapons := make([]schema.Resource, 0)
	ashes := make([]schema.Resource, 0)
	for _, resource := range catalog.byID {
		switch resource.Item.Family.Value {
		case schema.ItemFamilyWeapon:
			weapons = append(weapons, resource)
		case schema.ItemFamilyAshOfWar:
			ashes = append(ashes, resource)
		}
	}
	sort.Slice(weapons, func(i, j int) bool { return weapons[i].ID < weapons[j].ID })
	sort.Slice(ashes, func(i, j int) bool { return ashes[i].ID < ashes[j].ID })
	return weapons, ashes
}
