package schema

import "fmt"

func ValidateResource(resource Resource, sources map[SourceID]struct{}) error {
	if resource.Key == "" {
		return fmt.Errorf("resource key is required")
	}
	if resource.Kind != ResourceKindItem {
		return fmt.Errorf("resource %q: unsupported kind %q", resource.Key, resource.Kind)
	}
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
	if err := validateItemDocument(*resource.Item, sources); err != nil {
		return fmt.Errorf("resource %q: %w", resource.Key, err)
	}
	return nil
}
