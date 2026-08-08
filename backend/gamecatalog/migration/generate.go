package migration

import (
	"fmt"
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type GenerateOptions struct {
	Regulation          *RegulationData
	RegulationParams    *RegulationParameterData
	GameText            *GameTextData
	LegacyIconDirectory string
	GameVersion         string
}

type GeneratedCatalog struct {
	Manifest    schema.Manifest
	Resources   []schema.Resource
	IconSources map[string]string
}

func Generate(options GenerateOptions) (GeneratedCatalog, error) {
	if options.Regulation == nil {
		return GeneratedCatalog{}, fmt.Errorf("regulation data is required")
	}
	if options.GameText == nil {
		return GeneratedCatalog{}, fmt.Errorf("game text data is required")
	}
	if options.RegulationParams == nil {
		return GeneratedCatalog{}, fmt.Errorf("regulation parameter data is required")
	}
	if options.GameVersion == "" {
		return GeneratedCatalog{}, fmt.Errorf("game version is required")
	}

	snapshot := collectLegacySnapshot()
	iconSources := collectIconSources(snapshot.Items)
	sourceVersions, err := hashMigrationSources(options, iconSources, snapshot)
	if err != nil {
		return GeneratedCatalog{}, err
	}
	groups, err := groupLegacyItems(snapshot.Items, options.Regulation)
	if err != nil {
		return GeneratedCatalog{}, err
	}
	groups = append(groups, buildSlotOnlyGestureGroups(snapshot)...)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Canonical.ID < groups[j].Canonical.ID
	})

	context, err := newGenerationContext(options, sourceVersions, snapshot)
	if err != nil {
		return GeneratedCatalog{}, err
	}
	resources := make([]schema.Resource, 0, len(groups))
	for _, group := range groups {
		resource, buildErr := context.buildResource(group)
		if buildErr != nil {
			return GeneratedCatalog{}, fmt.Errorf(
				"build item 0x%08X: %w",
				group.Canonical.ID,
				buildErr,
			)
		}
		resources = append(resources, resource)
	}

	manifest := context.manifest
	manifest.DataVersion, err = computeCatalogDataVersion(
		manifest,
		resources,
		iconSources,
	)
	if err != nil {
		return GeneratedCatalog{}, err
	}
	result := GeneratedCatalog{
		Manifest:    manifest,
		Resources:   resources,
		IconSources: iconSources,
	}
	if err := validateGeneratedCatalog(result); err != nil {
		return GeneratedCatalog{}, err
	}
	return result, nil
}
