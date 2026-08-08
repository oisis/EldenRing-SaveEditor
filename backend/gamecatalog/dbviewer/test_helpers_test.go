package dbviewer

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	data, err := loader.LoadFS(embeddedCatalogMapFS(t))
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	server, err := New(data)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return server
}

func testServerWithCutContentDagger(t *testing.T) *Server {
	t.Helper()
	data, err := loader.LoadFS(embeddedCatalogMapFS(t))
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	for index := range data.Documents {
		item := data.Documents[index].Resource.Item
		if item == nil || item.GameID.Value != 0x000F4240 {
			continue
		}
		item.Safety.CutContent = schema.Fact[bool]{
			Known:      true,
			Value:      true,
			Provenance: item.Safety.CutContent.Provenance,
		}
		server, err := New(data)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		return server
	}
	t.Fatal("Dagger document is missing")
	return nil
}

func embeddedCatalogMapFS(t *testing.T) fstest.MapFS {
	t.Helper()
	source := catalogdata.Files()
	documentPaths := []string{
		"items/weapon/000f4240.json",
		"items/ash_of_war/8000ea60.json",
	}
	assetPaths := []string{
		"assets/icons/items/melee_armaments/dagger.png",
		"assets/icons/items/ashes_of_war/determination.png",
	}
	indexJSON, err := fs.ReadFile(source, "catalog.json")
	if err != nil {
		t.Fatalf("read embedded catalog.json: %v", err)
	}
	var index loader.CatalogFile
	if err := json.Unmarshal(indexJSON, &index); err != nil {
		t.Fatalf("decode embedded catalog.json: %v", err)
	}
	index.Documents = documentPaths
	indexJSON, err = json.Marshal(index)
	if err != nil {
		t.Fatalf("encode test catalog.json: %v", err)
	}

	paths := append(append([]string(nil), documentPaths...), assetPaths...)
	result := make(fstest.MapFS, len(paths))
	result["catalog.json"] = &fstest.MapFile{Data: indexJSON}
	for _, path := range paths {
		content, err := fs.ReadFile(source, path)
		if err != nil {
			t.Fatalf("read embedded %s: %v", path, err)
		}
		result[path] = &fstest.MapFile{Data: append([]byte(nil), content...)}
	}
	return result
}

func request(t *testing.T, handler http.Handler, method string, target string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, target, nil))
	return response
}
