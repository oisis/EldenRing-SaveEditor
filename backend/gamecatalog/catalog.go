package gamecatalog

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type Catalog struct {
	manifest     schema.Manifest
	byID         map[schema.ResourceID]schema.Resource
	byItemGameID map[uint32]schema.ResourceID
	outgoing     map[schema.ResourceID][]schema.Relation
	incoming     map[schema.ResourceID][]schema.Relation
}

func New(manifest schema.Manifest, resources []schema.Resource) (*Catalog, error) {
	sources, err := schema.ValidateManifest(manifest)
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}

	catalog := &Catalog{
		manifest:     cloneManifest(manifest),
		byID:         make(map[schema.ResourceID]schema.Resource, len(resources)),
		byItemGameID: make(map[uint32]schema.ResourceID, len(resources)),
		outgoing:     make(map[schema.ResourceID][]schema.Relation),
		incoming:     make(map[schema.ResourceID][]schema.Relation),
	}
	keys := make(map[string]struct{}, len(resources))
	for index, resource := range resources {
		if err := schema.ValidateResource(resource, sources); err != nil {
			return nil, fmt.Errorf("resource %d: %w", index, err)
		}
		if _, exists := catalog.byID[resource.ID]; exists {
			return nil, fmt.Errorf("resource %d: duplicate resource ID %d", index, resource.ID)
		}
		if _, exists := keys[resource.Key]; exists {
			return nil, fmt.Errorf("resource %d: duplicate resource key %q", index, resource.Key)
		}
		if existing, exists := catalog.byItemGameID[resource.Item.GameID.Value]; exists && existing != resource.ID {
			return nil, fmt.Errorf("resource %d: duplicate item game ID 0x%08X", index, resource.Item.GameID.Value)
		}

		cloned := cloneResource(resource)
		catalog.byID[cloned.ID] = cloned
		catalog.byItemGameID[cloned.Item.GameID.Value] = cloned.ID
		for _, variant := range cloned.Item.Variants {
			if existing, exists := catalog.byItemGameID[variant.GameID.Value]; exists && existing != cloned.ID {
				return nil, fmt.Errorf("resource %d: variant game ID 0x%08X belongs to resource %d", index, variant.GameID.Value, existing)
			}
			catalog.byItemGameID[variant.GameID.Value] = cloned.ID
		}
		for _, alias := range cloned.Item.Aliases {
			if existing, exists := catalog.byItemGameID[alias.GameID.Value]; exists {
				return nil, fmt.Errorf(
					"resource %d: alias game ID 0x%08X already belongs to resource %d",
					index,
					alias.GameID.Value,
					existing,
				)
			}
			catalog.byItemGameID[alias.GameID.Value] = cloned.ID
		}
		keys[cloned.Key] = struct{}{}
	}

	if err := catalog.deriveRelations(sources); err != nil {
		return nil, err
	}
	return catalog, nil
}

func (catalog *Catalog) Manifest() schema.Manifest {
	return cloneManifest(catalog.manifest)
}

func (catalog *Catalog) ResourceCount() int {
	return len(catalog.byID)
}
