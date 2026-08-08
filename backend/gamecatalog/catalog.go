package gamecatalog

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type Catalog struct {
	manifest schema.Manifest
	// byKind resolves the resource kind first and only then the key inside that
	// kind, so the same key may later exist under a different kind.
	byKind       map[schema.ResourceKind]map[string]schema.Resource
	byItemGameID map[uint32]schema.ResourceRef
	outgoing     map[schema.ResourceRef][]schema.Relation
	incoming     map[schema.ResourceRef][]schema.Relation
}

func New(manifest schema.Manifest, resources []schema.Resource) (*Catalog, error) {
	sources, err := schema.ValidateManifest(manifest)
	if err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}

	catalog := &Catalog{
		manifest:     cloneManifest(manifest),
		byKind:       make(map[schema.ResourceKind]map[string]schema.Resource),
		byItemGameID: make(map[uint32]schema.ResourceRef, len(resources)),
		outgoing:     make(map[schema.ResourceRef][]schema.Relation),
		incoming:     make(map[schema.ResourceRef][]schema.Relation),
	}
	for index, resource := range resources {
		if err := schema.ValidateResource(resource, sources); err != nil {
			return nil, fmt.Errorf("resource %d: %w", index, err)
		}
		ref := resource.Ref()
		byKey, exists := catalog.byKind[ref.Kind]
		if !exists {
			byKey = make(map[string]schema.Resource)
			catalog.byKind[ref.Kind] = byKey
		}
		if _, duplicate := byKey[ref.Key]; duplicate {
			return nil, fmt.Errorf("resource %d: duplicate resource kind %q key %q", index, ref.Kind, ref.Key)
		}
		if existing, exists := catalog.byItemGameID[resource.Item.GameID.Value]; exists && existing != ref {
			return nil, fmt.Errorf("resource %d: duplicate item game ID 0x%08X", index, resource.Item.GameID.Value)
		}

		cloned := cloneResource(resource)
		byKey[ref.Key] = cloned
		catalog.byItemGameID[cloned.Item.GameID.Value] = ref
		for _, variant := range cloned.Item.Variants {
			if existing, exists := catalog.byItemGameID[variant.GameID.Value]; exists && existing != ref {
				return nil, fmt.Errorf(
					"resource %d: variant game ID 0x%08X belongs to resource kind %q key %q",
					index,
					variant.GameID.Value,
					existing.Kind,
					existing.Key,
				)
			}
			catalog.byItemGameID[variant.GameID.Value] = ref
		}
		for _, alias := range cloned.Item.Aliases {
			if existing, exists := catalog.byItemGameID[alias.GameID.Value]; exists {
				return nil, fmt.Errorf(
					"resource %d: alias game ID 0x%08X already belongs to resource kind %q key %q",
					index,
					alias.GameID.Value,
					existing.Kind,
					existing.Key,
				)
			}
			catalog.byItemGameID[alias.GameID.Value] = ref
		}
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
	count := 0
	for _, byKey := range catalog.byKind {
		count += len(byKey)
	}
	return count
}
