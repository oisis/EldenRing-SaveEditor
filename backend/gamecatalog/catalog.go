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
	// networkPresets is read-only after construction and stays empty for a
	// catalog built without network parameters.
	networkPresets []NetworkPreset
	// appearancePresets is read-only after construction and stays empty for a
	// catalog built without appearance presets. It is a data set of its own, not
	// a resource kind.
	appearancePresets []AppearancePreset
}

// CatalogData is everything a catalog may be built from: the resource manifest
// and resources, plus the optional data sets the runtimes serve. Each field may
// stay empty, so a new data set never forces a new constructor.
type CatalogData struct {
	Manifest          schema.Manifest
	Resources         []schema.Resource
	NetworkPresets    []NetworkPreset
	AppearancePresets []AppearancePreset
}

// New builds a catalog without network parameters, which every caller that only
// reads resources needs.
func New(manifest schema.Manifest, resources []schema.Resource) (*Catalog, error) {
	return NewWithData(CatalogData{Manifest: manifest, Resources: resources})
}

// NewWithNetworkParams builds a catalog that additionally carries the network
// presets of NetworkParamsPath, for the runtimes that serve them.
func NewWithNetworkParams(
	manifest schema.Manifest,
	resources []schema.Resource,
	networkPresets []NetworkPreset,
) (*Catalog, error) {
	return NewWithData(CatalogData{
		Manifest:       manifest,
		Resources:      resources,
		NetworkPresets: networkPresets,
	})
}

// NewWithData builds a catalog from the complete data set. It is the single
// constructor; New and NewWithNetworkParams delegate to it so existing callers
// keep their signature.
func NewWithData(data CatalogData) (*Catalog, error) {
	manifest := data.Manifest
	resources := data.Resources
	if len(data.NetworkPresets) > 0 {
		if err := validateNetworkPresets(data.NetworkPresets); err != nil {
			return nil, fmt.Errorf("network parameters: %w", err)
		}
	}
	if len(data.AppearancePresets) > 0 {
		if err := validateAppearancePresets(data.AppearancePresets); err != nil {
			return nil, fmt.Errorf("appearance presets: %w", err)
		}
	}
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
		// The stored copies keep the caller's slices out of the catalog.
		networkPresets:    append([]NetworkPreset(nil), data.NetworkPresets...),
		appearancePresets: cloneAppearancePresets(data.AppearancePresets),
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
		cloned := cloneResource(resource)
		byKey[ref.Key] = cloned
		// Only an item carries a game ID, so only an item enters that index.
		if cloned.Item == nil {
			continue
		}
		if existing, exists := catalog.byItemGameID[resource.Item.GameID.Value]; exists && existing != ref {
			return nil, fmt.Errorf("resource %d: duplicate item game ID 0x%08X", index, resource.Item.GameID.Value)
		}
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
