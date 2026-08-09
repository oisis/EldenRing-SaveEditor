package network

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
)

// presetIDs is the contractual public order of the presets stored in
// regulation/network_params.json. The values belong to that file; only the
// exposed identifiers and their order are the endpoint contract.
var presetIDs = []string{
	"vanilla",
	"faster-reds",
	"aggressive-reds",
	"faster-summons",
	"aggressive-summons",
	"faster-blue",
	"aggressive-blue",
	"faster-summon-host",
	"aggressive-summon-host",
	"faster-summon-guest",
	"aggressive-summon-guest",
	"faster-hunter",
	"aggressive-hunter",
}

func newCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.NewPrototype()
	if err != nil {
		t.Fatalf("gamecatalog.NewPrototype: %v", err)
	}
	return gameCatalog
}

func TestGetNetworkPresetsReturnsEveryPresetInOrder(t *testing.T) {
	result, err := GetNetworkPresets(newCatalog(t), "")
	if err != nil {
		t.Fatalf("GetNetworkPresets(\"\") = %v, want nil", err)
	}
	if result.Presets == nil {
		t.Fatal("Presets is nil, want a non-nil slice")
	}
	if len(result.Presets) != len(presetIDs) {
		t.Fatalf("len(Presets) = %d, want %d", len(result.Presets), len(presetIDs))
	}
	for index, want := range presetIDs {
		if got := result.Presets[index].ID; got != want {
			t.Fatalf("Presets[%d].ID = %q, want %q", index, got, want)
		}
	}
}

// vanilla carries the default object of the catalog data file.
func TestGetNetworkPresetsReturnsTheDefaultValuesForVanilla(t *testing.T) {
	want := gamecatalog.NetworkParamValues{
		MaxBreakInTargetListCount:     5,
		BreakInRequestIntervalTimeSec: 30,
		BreakInRequestTimeOutSec:      20,
		BreakInRequestAreaCount:       5,
		SummonTimeoutTime:             45,
		ReloadSignIntervalTime2:       60,
		ReloadSignTotalCount:          20,
		ReloadSignCellCount:           10,
		UpdateSignIntervalTime:        30,
		SingGetMax:                    32,
		SignDownloadSpan:              30,
		SignUpdateSpan:                60,
		ReloadVisitListCoolTime:       20,
		MaxCoopBlueSummonCount:        2,
		MaxVisitListCount:             5,
		ReloadSearchCoopBlueMin:       30,
		ReloadSearchCoopBlueMax:       180,
		AllAreaSearchRateCoopBlue:     30,
		AllAreaSearchRateVsBlue:       30,
		VisitorListMax:                10,
		VisitorTimeOutTime:            60,
		VisitorDownloadSpan:           60,
	}

	result, err := GetNetworkPresets(newCatalog(t), "vanilla")
	if err != nil {
		t.Fatalf("GetNetworkPresets(\"vanilla\") = %v, want nil", err)
	}
	if len(result.Presets) != 1 {
		t.Fatalf("len(Presets) = %d, want 1", len(result.Presets))
	}
	if !reflect.DeepEqual(result.Presets[0].Parameters, want) {
		t.Fatalf("Presets[0].Parameters = %#v, want %#v", result.Presets[0].Parameters, want)
	}
}

// faster-reds is the representative preset: it must carry the stored values,
// which differ from the default only in the invader fields.
func TestGetNetworkPresetsReturnsTheStoredValuesOfAPreset(t *testing.T) {
	result, err := GetNetworkPresets(newCatalog(t), "faster-reds")
	if err != nil {
		t.Fatalf("GetNetworkPresets(\"faster-reds\") = %v, want nil", err)
	}
	if len(result.Presets) != 1 {
		t.Fatalf("len(Presets) = %d, want 1", len(result.Presets))
	}
	if result.Presets[0].ID != "faster-reds" {
		t.Fatalf("Presets[0].ID = %q, want faster-reds", result.Presets[0].ID)
	}
	want := gamecatalog.NetworkParamValues{
		MaxBreakInTargetListCount:     8,
		BreakInRequestIntervalTimeSec: 12,
		BreakInRequestTimeOutSec:      8,
		BreakInRequestAreaCount:       8,
		SummonTimeoutTime:             45,
		ReloadSignIntervalTime2:       60,
		ReloadSignTotalCount:          20,
		ReloadSignCellCount:           10,
		UpdateSignIntervalTime:        30,
		SingGetMax:                    32,
		SignDownloadSpan:              30,
		SignUpdateSpan:                60,
		ReloadVisitListCoolTime:       20,
		MaxCoopBlueSummonCount:        2,
		MaxVisitListCount:             5,
		ReloadSearchCoopBlueMin:       30,
		ReloadSearchCoopBlueMax:       180,
		AllAreaSearchRateCoopBlue:     30,
		AllAreaSearchRateVsBlue:       30,
		VisitorListMax:                10,
		VisitorTimeOutTime:            60,
		VisitorDownloadSpan:           60,
	}
	if !reflect.DeepEqual(result.Presets[0].Parameters, want) {
		t.Fatalf("Presets[0].Parameters = %#v, want %#v", result.Presets[0].Parameters, want)
	}
}

// A non-empty presetID selects exactly one preset, and the values it returns are
// the ones the full list carries for that identifier.
func TestGetNetworkPresetsFiltersByPresetID(t *testing.T) {
	gameCatalog := newCatalog(t)

	all, err := GetNetworkPresets(gameCatalog, "")
	if err != nil {
		t.Fatalf("GetNetworkPresets(\"\") = %v, want nil", err)
	}
	for _, preset := range all.Presets {
		t.Run(preset.ID, func(t *testing.T) {
			result, err := GetNetworkPresets(gameCatalog, preset.ID)
			if err != nil {
				t.Fatalf("GetNetworkPresets(%q) = %v, want nil", preset.ID, err)
			}
			if len(result.Presets) != 1 {
				t.Fatalf("len(Presets) = %d, want 1", len(result.Presets))
			}
			if !reflect.DeepEqual(result.Presets[0], preset) {
				t.Fatalf("Presets[0] = %#v, want %#v", result.Presets[0], preset)
			}
		})
	}
}

func TestGetNetworkPresetsRejectsAnUnknownPresetID(t *testing.T) {
	gameCatalog := newCatalog(t)

	// The legacy backend/core presets are not part of the catalog data, so their
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
		result, err := GetNetworkPresets(gameCatalog, presetID)
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
	gameCatalog := newCatalog(t)

	for _, presetID := range []string{"Vanilla", "FASTER-REDS", " vanilla", "vanilla ", "faster_reds"} {
		if _, err := GetNetworkPresets(gameCatalog, presetID); err == nil {
			t.Fatalf("GetNetworkPresets(%q) = nil error, want a rejection", presetID)
		}
	}
}

// Every call returns its own copy, so mutating one must not affect the next.
func TestGetNetworkPresetsBuildsAnIndependentResultPerCall(t *testing.T) {
	gameCatalog := newCatalog(t)

	first, err := GetNetworkPresets(gameCatalog, "")
	if err != nil {
		t.Fatalf("GetNetworkPresets(\"\") = %v, want nil", err)
	}
	want := first.Presets[0]
	first.Presets[0].ID = "mutated"
	first.Presets[0].Parameters.MaxBreakInTargetListCount = -1

	second, err := GetNetworkPresets(gameCatalog, "")
	if err != nil {
		t.Fatalf("GetNetworkPresets(\"\") = %v, want nil", err)
	}
	if !reflect.DeepEqual(second.Presets[0], want) {
		t.Fatalf("Presets[0] = %#v, want %#v", second.Presets[0], want)
	}
}

// Without a catalog, and without network parameters inside it, the getter has no
// data to report and must say so instead of returning an empty list.
func TestGetNetworkPresetsRequiresACatalogWithNetworkParameters(t *testing.T) {
	if _, err := GetNetworkPresets(nil, ""); err == nil {
		t.Fatal("GetNetworkPresets(nil, \"\") = nil error, want a rejection")
	}

	manifest, resources := prototype.Data()
	withoutParameters, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	if _, err := GetNetworkPresets(withoutParameters, ""); err == nil {
		t.Fatal("GetNetworkPresets without network parameters = nil error, want a rejection")
	}
}
