package gamecatalog_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
)

func TestLoadNetworkParamsReadsTheCatalogFile(t *testing.T) {
	presets, err := gamecatalog.LoadNetworkParams(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadNetworkParams: %v", err)
	}
	if len(presets) != 13 {
		t.Fatalf("len(presets) = %d, want 13", len(presets))
	}
	// The default object is loaded first, so the catalog order starts with it.
	if presets[0].ID != "vanilla" {
		t.Fatalf("presets[0].ID = %q, want vanilla", presets[0].ID)
	}
	if presets[0].Parameters.MaxBreakInTargetListCount != 5 {
		t.Fatalf(
			"presets[0].Parameters.MaxBreakInTargetListCount = %d, want 5",
			presets[0].Parameters.MaxBreakInTargetListCount,
		)
	}
}

func TestLoadNetworkParamsRejectsInvalidData(t *testing.T) {
	for name, content := range map[string]string{
		"malformed JSON": `{"default": {"id": "vanilla",`,
		"duplicate ID": `{
			"default": {"id": "vanilla", "parameters": {}},
			"presets": [
				{"id": "faster-reds", "parameters": {}},
				{"id": "faster-reds", "parameters": {}}
			]
		}`,
		// The default preset is the vanilla preset; no other identifier may take
		// its place.
		"default ID other than vanilla": `{
			"default": {"id": "defaults", "parameters": {}},
			"presets": [{"id": "faster-reds", "parameters": {}}]
		}`,
		// A second document must not be ignored just because the first one parsed.
		"trailing second document": `{
			"default": {"id": "vanilla", "parameters": {}},
			"presets": [{"id": "faster-reds", "parameters": {}}]
		}
		{"default": {"id": "vanilla", "parameters": {}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			catalogFS := fstest.MapFS{
				gamecatalog.NetworkParamsPath: &fstest.MapFile{Data: []byte(content)},
			}
			if _, err := gamecatalog.LoadNetworkParams(catalogFS); err == nil {
				t.Fatalf("LoadNetworkParams(%s) = nil error, want a rejection", name)
			} else if !strings.Contains(err.Error(), gamecatalog.NetworkParamsPath) {
				t.Fatalf("error = %q, want it to name %s", err.Error(), gamecatalog.NetworkParamsPath)
			}
		})
	}
}
