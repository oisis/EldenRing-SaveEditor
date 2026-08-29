package desktop_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/character"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/savesession"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
	"github.com/oisis/EldenRing-SaveForge/internal/desktop"
)

type endpointCall func() (any, error)

func newTestBridge(version string) *desktop.Bridge {
	return desktop.NewBridge(version, saveengine.New())
}

func assertCallsMatch(t *testing.T, bridged endpointCall, direct endpointCall) {
	t.Helper()

	bridgedResult, bridgedErr := bridged()
	directResult, directErr := direct()
	if !reflect.DeepEqual(bridgedResult, directResult) {
		t.Fatalf("bridge result = %#v, want endpoint result %#v", bridgedResult, directResult)
	}
	if (bridgedErr == nil) != (directErr == nil) {
		t.Fatalf("bridge error = %v, endpoint error = %v", bridgedErr, directErr)
	}
	if bridgedErr == nil {
		return
	}
	if reflect.TypeOf(bridgedErr) != reflect.TypeOf(directErr) {
		t.Fatalf("bridge error type = %T, want endpoint error type %T", bridgedErr, directErr)
	}
	if bridgedErr.Error() != directErr.Error() {
		t.Fatalf("bridge error = %q, want endpoint error %q", bridgedErr, directErr)
	}
}

func TestGetApplicationInfoPassesTheWiredVersionToTheEndpoint(t *testing.T) {
	// A value no fallback would produce proves the bridge forwards exactly what
	// the composition root wired, without trimming or substituting a default.
	const version = "  2.0.0-rc.1+local  "

	result, err := newTestBridge(version).GetApplicationInfo()
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

	bridged, err := newTestBridge(version).GetApplicationInfo()
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
	result, err := newTestBridge("dev").GetApplicationInfo()
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
	result, err := newTestBridge("").GetApplicationInfo()
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

func TestReadOnlySaveMethodsReturnEndpointResultsAndErrorsUnchanged(t *testing.T) {
	engine := saveengine.New()
	bridge := desktop.NewBridge("dev", engine)
	missingSource := filepath.Join(t.TempDir(), "missing.sl2")
	const unknownSessionID = "unknown-session"

	tests := []struct {
		name    string
		bridged endpointCall
		direct  endpointCall
	}{
		{
			name: "LoadSave",
			bridged: func() (any, error) {
				return bridge.LoadSave(missingSource, "pc")
			},
			direct: func() (any, error) {
				return savesession.LoadSave(engine, missingSource, "pc")
			},
		},
		{
			name: "GetLoadedSave",
			bridged: func() (any, error) {
				return bridge.GetLoadedSave(unknownSessionID)
			},
			direct: func() (any, error) {
				return savesession.GetLoadedSave(engine, unknownSessionID)
			},
		},
		{
			name: "CloseSave",
			bridged: func() (any, error) {
				return nil, bridge.CloseSave(unknownSessionID)
			},
			direct: func() (any, error) {
				return nil, savesession.CloseSave(engine, unknownSessionID)
			},
		},
		{
			name: "GetSaveCharacters",
			bridged: func() (any, error) {
				return bridge.GetSaveCharacters(unknownSessionID)
			},
			direct: func() (any, error) {
				return character.GetSaveCharacters(engine, unknownSessionID)
			},
		},
		{
			name: "GetCharacterProfile",
			bridged: func() (any, error) {
				return bridge.GetCharacterProfile(unknownSessionID, 0)
			},
			direct: func() (any, error) {
				return character.GetCharacterProfile(engine, unknownSessionID, 0)
			},
		},
		{
			name: "GetCharacterStats",
			bridged: func() (any, error) {
				return bridge.GetCharacterStats(unknownSessionID, 0)
			},
			direct: func() (any, error) {
				return character.GetCharacterStats(engine, unknownSessionID, 0)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertCallsMatch(t, test.bridged, test.direct)
		})
	}
}

func TestReadOnlySaveMethodsPropagateNilEngineErrorsWithoutFallback(t *testing.T) {
	bridge := desktop.NewBridge("dev", nil)

	tests := []struct {
		name    string
		bridged endpointCall
		direct  endpointCall
	}{
		{
			name: "LoadSave",
			bridged: func() (any, error) {
				return bridge.LoadSave("source.sl2", "")
			},
			direct: func() (any, error) {
				return savesession.LoadSave(nil, "source.sl2", "")
			},
		},
		{
			name: "GetLoadedSave",
			bridged: func() (any, error) {
				return bridge.GetLoadedSave("session")
			},
			direct: func() (any, error) {
				return savesession.GetLoadedSave(nil, "session")
			},
		},
		{
			name: "CloseSave",
			bridged: func() (any, error) {
				return nil, bridge.CloseSave("session")
			},
			direct: func() (any, error) {
				return nil, savesession.CloseSave(nil, "session")
			},
		},
		{
			name: "GetSaveCharacters",
			bridged: func() (any, error) {
				return bridge.GetSaveCharacters("session")
			},
			direct: func() (any, error) {
				return character.GetSaveCharacters(nil, "session")
			},
		},
		{
			name: "GetCharacterProfile",
			bridged: func() (any, error) {
				return bridge.GetCharacterProfile("session", 0)
			},
			direct: func() (any, error) {
				return character.GetCharacterProfile(nil, "session", 0)
			},
		},
		{
			name: "GetCharacterStats",
			bridged: func() (any, error) {
				return bridge.GetCharacterStats("session", 0)
			},
			direct: func() (any, error) {
				return character.GetCharacterStats(nil, "session", 0)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertCallsMatch(t, test.bridged, test.direct)
		})
	}
}
