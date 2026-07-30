package schema

import "fmt"

func ValidateManifest(manifest Manifest) (map[SourceID]struct{}, error) {
	if manifest.SchemaVersion == 0 {
		return nil, fmt.Errorf("schema version must be greater than zero")
	}
	if manifest.DataVersion == "" {
		return nil, fmt.Errorf("data version is required")
	}
	if manifest.GameVersion == "" {
		return nil, fmt.Errorf("game version is required")
	}

	sources := make(map[SourceID]struct{}, len(manifest.Sources))
	for index, source := range manifest.Sources {
		if source.ID == "" {
			return nil, fmt.Errorf("source %d: ID is required", index)
		}
		if source.Kind == "" {
			return nil, fmt.Errorf("source %q: kind is required", source.ID)
		}
		if source.Location == "" {
			return nil, fmt.Errorf("source %q: location is required", source.ID)
		}
		if err := validateSourceLocation(source.Location); err != nil {
			return nil, fmt.Errorf("source %q: %w", source.ID, err)
		}
		if source.Version == "" {
			return nil, fmt.Errorf("source %q: version is required", source.ID)
		}
		if !validEvidenceLevel(source.Evidence) {
			return nil, fmt.Errorf("source %q: unsupported evidence level %q", source.ID, source.Evidence)
		}
		if _, exists := sources[source.ID]; exists {
			return nil, fmt.Errorf("duplicate source ID %q", source.ID)
		}
		sources[source.ID] = struct{}{}
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("at least one data source is required")
	}
	return sources, nil
}

func validEvidenceLevel(level EvidenceLevel) bool {
	switch level {
	case EvidenceRegulation, EvidenceVerifiedResearch, EvidenceCurated, EvidenceUnknown:
		return true
	default:
		return false
	}
}
