package loader_test

import (
	"encoding/json"
	"io/fs"
	"testing"
	"testing/fstest"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

func embeddedCatalogFS(t *testing.T) fstest.MapFS {
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
