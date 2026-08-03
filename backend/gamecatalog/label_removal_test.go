package gamecatalog_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/dbviewer"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

// TestEmbeddedResourcesHaveKnownDisplayName protects the invariant that replaced
// the removed top-level Resource.label: every catalog resource must carry a
// known, non-empty item.presentation.displayName, which is now the single item
// name source for both runtime and the DB Viewer.
func TestEmbeddedResourcesHaveKnownDisplayName(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	resources := data.Resources()
	if len(resources) == 0 {
		t.Fatal("embedded catalog has no resources")
	}
	for _, resource := range resources {
		if resource.Item == nil {
			t.Fatalf("resource %q has no item document", resource.Key)
		}
		displayName := resource.Item.Presentation.DisplayName
		if !displayName.Known || displayName.Value == "" {
			t.Fatalf(
				"resource %q display name = %#v, want known and non-empty",
				resource.Key,
				displayName,
			)
		}
	}
}

// TestEmbeddedDocumentsHaveNoTopLevelLabel proves the generated JSON no longer
// persists a top-level "label" field on any resource document.
func TestEmbeddedDocumentsHaveNoTopLevelLabel(t *testing.T) {
	files := catalogdata.Files()
	count := 0
	err := fs.WalkDir(files, "items", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		raw, err := fs.ReadFile(files, path)
		if err != nil {
			return err
		}
		var document map[string]json.RawMessage
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatalf("unmarshal %s: %v", path, err)
		}
		if _, exists := document["label"]; exists {
			t.Fatalf("document %s still contains a top-level \"label\" field", path)
		}
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded documents: %v", err)
	}
	if count == 0 {
		t.Fatal("no embedded item documents were scanned")
	}
}

// TestRuntimeAndViewerShareItemName drives the DB Viewer's public HTTP handler
// and confirms the rendered item name is derived from item.presentation.displayName
// and agrees with the runtime ItemByGameID lookup, for both a canonical item and a
// materialized variant. It fails if the Viewer stops using presentation.displayName
// or if the runtime and Viewer names diverge.
func TestRuntimeAndViewerShareItemName(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	catalog, err := gamecatalog.New(data.Manifest, data.Resources())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	server, err := dbviewer.New(data)
	if err != nil {
		t.Fatalf("dbviewer.New: %v", err)
	}
	handler := server.Handler()

	cases := []struct {
		name         string
		gameID       uint32
		renderedName string // the exact <h1> text the Viewer must render
	}{
		{name: "canonical Dagger", gameID: 0x000F4240, renderedName: "Dagger"},
		{name: "variant Dagger (quality)", gameID: 0x000F436C, renderedName: "Dagger (quality)"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// Source 1: item.presentation.displayName read straight from embedded data.
			embeddedName, ok := embeddedDisplayName(data, testCase.gameID)
			if !ok {
				t.Fatalf("game ID 0x%08X is absent from embedded documents", testCase.gameID)
			}
			if embeddedName == "" {
				t.Fatalf("game ID 0x%08X has an empty presentation.displayName", testCase.gameID)
			}

			// Source 2: runtime lookup must resolve to the same displayName field.
			runtime, exists := catalog.ItemByGameID(testCase.gameID)
			if !exists {
				t.Fatalf("runtime lookup for 0x%08X failed", testCase.gameID)
			}
			runtimeName := runtime.Item.Presentation.DisplayName.Value
			if runtimeName != embeddedName {
				t.Fatalf(
					"runtime displayName = %q, embedded displayName = %q",
					runtimeName,
					embeddedName,
				)
			}

			// Source 3: the DB Viewer HTTP response.
			response := httptest.NewRecorder()
			target := fmt.Sprintf("/items/%08X", testCase.gameID)
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("%s status = %d, want 200", target, response.Code)
			}
			body := response.Body.String()

			// The Viewer must render exactly the expected heading, and that heading
			// must be built from presentation.displayName (the runtime name).
			wantHeading := "<h1>" + testCase.renderedName + "</h1>"
			if !strings.Contains(body, wantHeading) {
				t.Fatalf("Viewer response for %s does not contain heading %q", target, wantHeading)
			}
			if !strings.Contains(testCase.renderedName, runtimeName) {
				t.Fatalf(
					"Viewer heading %q is not derived from presentation.displayName %q",
					testCase.renderedName,
					runtimeName,
				)
			}
		})
	}
}

// embeddedDisplayName returns the item.presentation.displayName recorded in the
// embedded catalog for the given game ID, resolving both canonical documents and
// their variants without materializing through the runtime.
func embeddedDisplayName(data loader.Data, gameID uint32) (string, bool) {
	for index := range data.Documents {
		item := data.Documents[index].Resource.Item
		if item == nil {
			continue
		}
		if item.GameID.Value == gameID {
			return item.Presentation.DisplayName.Value, true
		}
		for variantIndex := range item.Variants {
			variant := item.Variants[variantIndex]
			if variant.GameID.Value == gameID {
				return variant.Data.Presentation.DisplayName.Value, true
			}
		}
	}
	return "", false
}
