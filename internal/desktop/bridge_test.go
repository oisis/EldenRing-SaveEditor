package desktop_test

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/internal/desktop"
)

func TestGetApplicationInfoPassesTheWiredVersionToTheEndpoint(t *testing.T) {
	// A value no fallback would produce proves the bridge forwards exactly what
	// the composition root wired, without trimming or substituting a default.
	const version = "  2.0.0-rc.1+local  "

	result, err := desktop.NewBridge(version).GetApplicationInfo()
	if err != nil {
		t.Fatalf("GetApplicationInfo: %v", err)
	}
	if result.ApplicationVersion != version {
		t.Fatalf("applicationVersion = %q, want %q", result.ApplicationVersion, version)
	}
}

// The bridge must return the endpoint result unchanged. Comparing against a
// direct endpoint call proves the bridge adds, drops and reorders nothing.
func TestGetApplicationInfoReturnsTheEndpointResultUnchanged(t *testing.T) {
	const version = "2.0.0"

	bridged, err := desktop.NewBridge(version).GetApplicationInfo()
	if err != nil {
		t.Fatalf("bridge GetApplicationInfo: %v", err)
	}
	direct, err := application.GetApplicationInfo(version)
	if err != nil {
		t.Fatalf("endpoint GetApplicationInfo: %v", err)
	}

	if !reflect.DeepEqual(bridged, direct) {
		t.Fatalf("bridge result = %#v, want the endpoint result %#v", bridged, direct)
	}
}

// Capabilities and schema versions are backend contract. The bridge must not
// declare, filter or extend them, so it can only report what the endpoint does.
func TestGetApplicationInfoDeclaresNoCapabilitiesOrSchemasOfItsOwn(t *testing.T) {
	result, err := desktop.NewBridge("dev").GetApplicationInfo()
	if err != nil {
		t.Fatalf("GetApplicationInfo: %v", err)
	}

	direct, err := application.GetApplicationInfo("dev")
	if err != nil {
		t.Fatalf("endpoint GetApplicationInfo: %v", err)
	}
	if !reflect.DeepEqual(result.Capabilities, direct.Capabilities) {
		t.Fatalf("capabilities = %#v, want the endpoint capabilities %#v",
			result.Capabilities, direct.Capabilities)
	}
	if !reflect.DeepEqual(result.SupportedSchemas, direct.SupportedSchemas) {
		t.Fatalf("supportedSchemas = %#v, want the endpoint schemas %#v",
			result.SupportedSchemas, direct.SupportedSchemas)
	}
}

// An empty version is a wiring error owned by the endpoint. The bridge must
// propagate it instead of hiding it behind a fallback version.
func TestGetApplicationInfoPropagatesTheEmptyVersionWiringError(t *testing.T) {
	result, err := desktop.NewBridge("").GetApplicationInfo()
	if err == nil {
		t.Fatal("GetApplicationInfo with an empty wired version = nil error, want a rejection")
	}
	if err.Error() != "application version is required" {
		t.Fatalf("error = %q, want %q", err.Error(), "application version is required")
	}
	if !reflect.DeepEqual(result, application.GetApplicationInfoResult{}) {
		t.Fatalf("result = %#v, want the empty result", result)
	}
}
