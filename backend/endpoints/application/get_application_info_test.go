package application_test

import (
	"runtime"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGetApplicationInfoReturnsTheSuppliedVersion(t *testing.T) {
	// A version that no fallback would produce proves the value is returned
	// verbatim, without trimming or normalisation.
	const version = "  2.0.0-rc.1+local  "

	result, err := application.GetApplicationInfo(version)
	if err != nil {
		t.Fatalf("GetApplicationInfo: %v", err)
	}
	if result.ApplicationVersion != version {
		t.Fatalf("applicationVersion = %q, want %q", result.ApplicationVersion, version)
	}
	if result.Build == "" || result.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		t.Fatalf("build/platform = %q/%q", result.Build, result.Platform)
	}
}

func TestGetApplicationInfoReportsTheGameCatalogSchema(t *testing.T) {
	result, err := application.GetApplicationInfo("dev")
	if err != nil {
		t.Fatalf("GetApplicationInfo: %v", err)
	}
	if len(result.SupportedSchemas) != 1 {
		t.Fatalf("supportedSchemas = %#v, want exactly one entry", result.SupportedSchemas)
	}

	supported := result.SupportedSchemas[0]
	if supported.Name != "game_catalog" {
		t.Fatalf("schema name = %q, want game_catalog", supported.Name)
	}
	if supported.MinimumVersion != schema.MinimumSchemaVersion {
		t.Fatalf("minimumVersion = %d, want schema.MinimumSchemaVersion %d",
			supported.MinimumVersion, schema.MinimumSchemaVersion)
	}
	if supported.CurrentVersion != schema.CurrentSchemaVersion {
		t.Fatalf("currentVersion = %d, want schema.CurrentSchemaVersion %d",
			supported.CurrentVersion, schema.CurrentSchemaVersion)
	}
}

func TestGetApplicationInfoDeclaresOnlyCatalogRead(t *testing.T) {
	result, err := application.GetApplicationInfo("dev")
	if err != nil {
		t.Fatalf("GetApplicationInfo: %v", err)
	}
	if len(result.Capabilities) != 1 || result.Capabilities[0] != "catalog_read" {
		t.Fatalf("capabilities = %#v, want exactly [catalog_read]", result.Capabilities)
	}
}

func TestGetApplicationInfoRejectsAnEmptyVersion(t *testing.T) {
	result, err := application.GetApplicationInfo("")
	if err == nil {
		t.Fatal("GetApplicationInfo(\"\") = nil error, want a rejection")
	}
	if err.Error() != "application version is required" {
		t.Fatalf("error = %q, want %q", err.Error(), "application version is required")
	}
	if result.ApplicationVersion != "" || result.SupportedSchemas != nil || result.Capabilities != nil {
		t.Fatalf("result = %#v, want the empty result", result)
	}
}

// A shared backing array would let one caller corrupt another caller's result.
func TestGetApplicationInfoReturnsIndependentSlices(t *testing.T) {
	first, err := application.GetApplicationInfo("dev")
	if err != nil {
		t.Fatalf("GetApplicationInfo: %v", err)
	}
	if first.SupportedSchemas == nil || first.Capabilities == nil {
		t.Fatalf("result = %#v, want non-nil slices", first)
	}

	first.SupportedSchemas[0].Name = "mutated"
	first.Capabilities[0] = "mutated"

	second, err := application.GetApplicationInfo("dev")
	if err != nil {
		t.Fatalf("GetApplicationInfo: %v", err)
	}
	if second.SupportedSchemas[0].Name != "game_catalog" {
		t.Fatalf("schema name = %q after mutating an earlier result, want game_catalog",
			second.SupportedSchemas[0].Name)
	}
	if second.Capabilities[0] != "catalog_read" {
		t.Fatalf("capability = %q after mutating an earlier result, want catalog_read",
			second.Capabilities[0])
	}
}
