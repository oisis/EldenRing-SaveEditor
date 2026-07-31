package itemrouting

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestValidateItemRouting(t *testing.T) {
	t.Parallel()

	const expectedEndpointID = "set_gesture_unlocked"
	resourceID := schema.ResourceID(42)

	t.Run("matching enabled capability is accepted", func(t *testing.T) {
		err := ValidateItemRouting(resourceID, "grant", Capability{
			Enabled:    true,
			EndpointID: expectedEndpointID,
		}, expectedEndpointID)
		if err != nil {
			t.Fatalf("ValidateItemRouting() error = %v", err)
		}
	})

	t.Run("disabled capability returns description without checking endpoint", func(t *testing.T) {
		err := ValidateItemRouting(resourceID, "grant", Capability{
			EndpointID:  "wrong_endpoint",
			Description: "gesture cannot be granted",
		}, expectedEndpointID)
		if err == nil {
			t.Fatal("ValidateItemRouting() error = nil")
		}
		if !strings.Contains(err.Error(), "gesture cannot be granted") {
			t.Fatalf("error %q does not contain capability description", err)
		}
		if strings.Contains(err.Error(), "endpointId mismatch") {
			t.Fatalf("disabled capability unexpectedly checked endpointId: %q", err)
		}
	})

	t.Run("mismatched endpoint identifies expected and actual endpointId", func(t *testing.T) {
		err := ValidateItemRouting(resourceID, "grant", Capability{
			Enabled:    true,
			EndpointID: "set_tutorial_unlocked",
		}, expectedEndpointID)
		if err == nil {
			t.Fatal("ValidateItemRouting() error = nil")
		}
		for _, expected := range []string{"endpointId mismatch", expectedEndpointID, "set_tutorial_unlocked"} {
			if !strings.Contains(err.Error(), expected) {
				t.Fatalf("error %q does not contain %q", err, expected)
			}
		}
	})
}
