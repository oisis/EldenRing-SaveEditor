package network

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestApplyNetworkPresetUsesTheCatalogPresetAndSharedWriter(t *testing.T) {
	gameCatalog := newCatalog(t)
	want, err := GetNetworkPresets(gameCatalog, "faster-reds")
	if err != nil {
		t.Fatalf("GetNetworkPresets: %v", err)
	}
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSettingsFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := ApplyNetworkPreset(
		engine, gameCatalog, loaded.SaveSessionID, "faster-reds", "0")
	if err != nil {
		t.Fatalf("ApplyNetworkPreset: %v", err)
	}
	parameters := want.Presets[0].Parameters
	if result.SaveSessionID != loaded.SaveSessionID || result.SaveRevision != "1" ||
		result.PresetID != "faster-reds" || result.NetworkSettings != parameters {
		t.Fatalf("result = %+v", result)
	}
	stored, err := engine.GetNetworkSettings(loaded.SaveSessionID)
	if err != nil || stored.SaveRevision != "1" || stored.Parameters != parameters {
		t.Fatalf("stored settings = %+v, err = %v", stored, err)
	}
}

func TestApplyNetworkPresetRejectsUnknownAndLegacyIDsWithoutMutation(t *testing.T) {
	gameCatalog := newCatalog(t)
	for _, test := range []struct {
		name     string
		presetID string
	}{
		{name: "missing ID"},
		{name: "legacy alias", presetID: "defaults"},
		{name: "case mismatch", presetID: "FASTER-REDS"},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine := saveengine.New()
			loaded, err := engine.LoadSave(writeSettingsFixture(t), "pc", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := ApplyNetworkPreset(
				engine, gameCatalog, loaded.SaveSessionID, test.presetID, "0")
			if err == nil {
				t.Fatalf("ApplyNetworkPreset succeeded: %+v", result)
			}
			if !reflect.DeepEqual(result, ApplyNetworkPresetResult{}) {
				t.Errorf("rejected result = %+v, want the complete zero result", result)
			}
			info, err := engine.GetSessionInfo(loaded.SaveSessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if info.UnsavedChanges {
				t.Fatalf("session info = %+v, want no changes", info)
			}
			accepted, err := ApplyNetworkPreset(
				engine, gameCatalog, loaded.SaveSessionID, "vanilla", "0")
			if err != nil || accepted.SaveRevision != "1" {
				t.Fatalf("valid call after rejection = %+v, err = %v", accepted, err)
			}
		})
	}
}

func TestApplyNetworkPresetDelegatesRevisionValidation(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSettingsFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	rejected, err := ApplyNetworkPreset(
		engine, newCatalog(t), loaded.SaveSessionID, "vanilla", "1")
	if err == nil {
		t.Fatalf("ApplyNetworkPreset accepted a stale revision: %+v", rejected)
	}
	if !reflect.DeepEqual(rejected, ApplyNetworkPresetResult{}) {
		t.Errorf("rejected result = %+v, want the complete zero result", rejected)
	}
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Fatalf("session info = %+v, want no changes", info)
	}
	accepted, err := ApplyNetworkPreset(
		engine, newCatalog(t), loaded.SaveSessionID, "vanilla", "0")
	if err != nil || accepted.SaveRevision != "1" {
		t.Fatalf("valid call after conflict = %+v, err = %v", accepted, err)
	}
}

func TestApplyNetworkPresetRequiresItsDependencies(t *testing.T) {
	if _, err := ApplyNetworkPreset(nil, newCatalog(t), "session", "vanilla", "0"); err == nil {
		t.Fatal("a missing engine was accepted")
	}
	if _, err := ApplyNetworkPreset(saveengine.New(), nil, "session", "vanilla", "0"); err == nil {
		t.Fatal("a missing catalog was accepted")
	}
}
