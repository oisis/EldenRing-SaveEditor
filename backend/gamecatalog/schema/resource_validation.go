package schema

import "fmt"

func ValidateResource(resource Resource, sources map[SourceID]struct{}) error {
	if resource.ID == 0 {
		return fmt.Errorf("resource ID must be greater than zero")
	}
	if resource.Key == "" {
		return fmt.Errorf("resource %d: key is required", resource.ID)
	}
	if resource.Kind != ResourceKindItem {
		return fmt.Errorf("resource %q: unsupported kind %q", resource.Key, resource.Kind)
	}
	if resource.Item == nil {
		return fmt.Errorf("resource %q: item document is required", resource.Key)
	}
	if err := validateItemDocument(*resource.Item, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	displayName := resource.Item.Presentation.DisplayName
	if !displayName.Known || displayName.Value == "" {
		return fmt.Errorf(
			"resource %q: item.presentation.displayName must be known and non-empty",
			resource.Key,
		)
	}
	return nil
}
