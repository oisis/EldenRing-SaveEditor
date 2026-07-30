package loader_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
)

func embeddedCatalogFS(t *testing.T) fstest.MapFS {
	t.Helper()
	source := catalogdata.Files()
	paths := []string{
		"catalog.json",
		"items/weapon/000f4240.json",
		"items/ash_of_war/8000ea60.json",
	}
	result := make(fstest.MapFS, len(paths))
	for _, path := range paths {
		content, err := fs.ReadFile(source, path)
		if err != nil {
			t.Fatalf("read embedded %s: %v", path, err)
		}
		result[path] = &fstest.MapFile{Data: append([]byte(nil), content...)}
	}
	return result
}
