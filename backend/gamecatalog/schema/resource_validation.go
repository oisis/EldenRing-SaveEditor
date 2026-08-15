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
	default:
		return fmt.Errorf("resource %q: unsupported kind %q", resource.Key, resource.Kind)
	}
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
	if resource.Colosseum != nil {
		return fmt.Errorf("resource %q: item resource must not carry a colosseum document", resource.Key)
	}
	if resource.Region != nil {
		return fmt.Errorf("resource %q: item resource must not carry a region document", resource.Key)
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
	if resource.Item != nil {
		return fmt.Errorf("resource %q: colosseum resource must not carry an item document", resource.Key)
	}
	if resource.Region != nil {
		return fmt.Errorf("resource %q: colosseum resource must not carry a region document", resource.Key)
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
	if resource.Item != nil {
		return fmt.Errorf("resource %q: region resource must not carry an item document", resource.Key)
	}
	if resource.Colosseum != nil {
		return fmt.Errorf("resource %q: region resource must not carry a colosseum document", resource.Key)
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
