package network

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
)

// presetSources is the approved mapping between a public preset ID and the
// backend/core function that owns its values, in the contractual order. The
// implementation of those functions belongs to backend/core; only the mapping
// is tested here.
var presetSources = []struct {
	id     string
	source func() core.NetworkParamValues
}{
	{"vanilla", core.NetworkParamDefaults},
	{"faster-reds", core.NetworkParamFasterReds},
	{"aggressive-reds", core.NetworkParamAggressiveReds},
	{"faster-summons", core.NetworkParamFasterSummons},
	{"aggressive-summons", core.NetworkParamAggressiveSummons},
	{"faster-blue", core.NetworkParamFasterBlue},
	{"aggressive-blue", core.NetworkParamAggressiveBlue},
	{"faster-summon-host", core.NetworkParamFasterSummonHost},
	{"aggressive-summon-host", core.NetworkParamAggressiveSummonHost},
	{"faster-summon-guest", core.NetworkParamFasterSummonGuest},
	{"aggressive-summon-guest", core.NetworkParamAggressiveSummonGuest},
	{"faster-hunter", core.NetworkParamFasterHunter},
	{"aggressive-hunter", core.NetworkParamAggressiveHunter},
}

func TestGetNetworkPresetsReturnsEveryPresetInOrder(t *testing.T) {
	result, err := GetNetworkPresets("")
	if err != nil {
		t.Fatalf("GetNetworkPresets(\"\") = %v, want nil", err)
	}
	if result.Presets == nil {
		t.Fatal("Presets is nil, want a non-nil slice")
	}
	if len(result.Presets) != len(presetSources) {
		t.Fatalf("len(Presets) = %d, want %d", len(result.Presets), len(presetSources))
	}
	for index, want := range presetSources {
		if got := result.Presets[index].ID; got != want.id {
			t.Fatalf("Presets[%d].ID = %q, want %q", index, got, want.id)
		}
	}
}

// The endpoint owns the mapping only: each preset must carry exactly the values
// of its backend/core function.
func TestGetNetworkPresetsMapsEveryIDToItsCoreFunction(t *testing.T) {
	for _, testCase := range presetSources {
		t.Run(testCase.id, func(t *testing.T) {
			result, err := GetNetworkPresets(testCase.id)
			if err != nil {
				t.Fatalf("GetNetworkPresets(%q) = %v, want nil", testCase.id, err)
			}
			if len(result.Presets) != 1 {
				t.Fatalf("len(Presets) = %d, want 1", len(result.Presets))
			}
			if result.Presets[0].ID != testCase.id {
				t.Fatalf("Presets[0].ID = %q, want %q", result.Presets[0].ID, testCase.id)
			}
			if want := testCase.source(); !reflect.DeepEqual(result.Presets[0].Parameters, want) {
				t.Fatalf("Presets[0].Parameters = %#v, want %#v", result.Presets[0].Parameters, want)
			}
		})
	}
}

func TestGetNetworkPresetsRejectsAnUnknownPresetID(t *testing.T) {
	// The legacy backend/core presets are deliberately not exposed, so their
	// names must be rejected like any other unknown identifier.
	for _, presetID := range []string{
		"unknown",
		"fast-invasions",
		"light-invasions",
		"fast-summons",
		"fast-blue",
		"aggressive-host",
		"defaults",
	} {
		result, err := GetNetworkPresets(presetID)
		if err == nil {
			t.Fatalf("GetNetworkPresets(%q) = nil error, want a rejection", presetID)
		}
		if want := "unknown network preset \"" + presetID + "\""; err.Error() != want {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
		if len(result.Presets) != 0 {
			t.Fatalf("Presets = %#v, want an empty result", result.Presets)
		}
	}
}

// The ID is matched exactly: neither a different case nor surrounding spaces
// may resolve to a preset.
func TestGetNetworkPresetsMatchesTheIDExactly(t *testing.T) {
	for _, presetID := range []string{"Vanilla", "FASTER-REDS", " vanilla", "vanilla ", "faster_reds"} {
		if _, err := GetNetworkPresets(presetID); err == nil {
			t.Fatalf("GetNetworkPresets(%q) = nil error, want a rejection", presetID)
		}
	}
}

// Every call builds its own result, so mutating one must not affect the next.
func TestGetNetworkPresetsBuildsAnIndependentResultPerCall(t *testing.T) {
	first, err := GetNetworkPresets("")
	if err != nil {
		t.Fatalf("GetNetworkPresets(\"\") = %v, want nil", err)
	}
	first.Presets[0].ID = "mutated"
	first.Presets[0].Parameters.MaxBreakInTargetListCount = -1

	second, err := GetNetworkPresets("")
	if err != nil {
		t.Fatalf("GetNetworkPresets(\"\") = %v, want nil", err)
	}
	if second.Presets[0].ID != "vanilla" {
		t.Fatalf("Presets[0].ID = %q, want vanilla", second.Presets[0].ID)
	}
	if want := core.NetworkParamDefaults(); !reflect.DeepEqual(second.Presets[0].Parameters, want) {
		t.Fatalf("Presets[0].Parameters = %#v, want %#v", second.Presets[0].Parameters, want)
	}
}
