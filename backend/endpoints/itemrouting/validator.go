package itemrouting

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// Capability contains only the routing fields needed by endpoint preflight.
// GameCatalog remains the owner of the complete capability model.
type Capability struct {
	Enabled     bool
	EndpointID  string
	Description string
}

// ValidateItemRouting provides the single routing guard used by item-resource
// mutation endpoints before any SaveEngine mutation.
func ValidateItemRouting(
	resourceID schema.ResourceID,
	capabilityName string,
	capability Capability,
	expectedEndpointID string,
) error {
	if !capability.Enabled {
		return fmt.Errorf(
			"resource %d capability %s is disabled: %s",
			resourceID,
			capabilityName,
			capability.Description,
		)
	}
	if capability.EndpointID != expectedEndpointID {
		return fmt.Errorf(
			"resource %d capability %s endpointId mismatch: expected %q, got %q",
			resourceID,
			capabilityName,
			expectedEndpointID,
			capability.EndpointID,
		)
	}
	return nil
}
