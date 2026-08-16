package dbviewer

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"testing"
	"testing/fstest"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

var nonItemResourcePaths = []string{
	"colosseums/royal_colosseum.json",
	"regions/limgrave_the_first_step.json",
	"summoning_pools/stormveil_castle_gateside_chamber.json",
	"graces/weeping_peninsula_tombsward_catacombs.json",
	"bosses/stormveil_castle_godrick_the_grafted.json",
	"map_regions/limgrave_limgrave_west.json",
	"tutorials/2010.json",
	"quests/brother_corhyn.json",
}

// catalogFSWithNonItems extends the two-item test catalog with stored resources
// that carry no item document at all.
func catalogFSWithNonItems(t *testing.T) fstest.MapFS {
	t.Helper()

	files := embeddedCatalogMapFS(t)
	for _, path := range nonItemResourcePaths {
		content, err := fs.ReadFile(catalogdata.Files(), path)
		if err != nil {
			t.Fatalf("read embedded %s: %v", path, err)
		}
		files[path] = &fstest.MapFile{Data: content}
	}

	var index loader.CatalogFile
	if err := json.Unmarshal(files["catalog.json"].Data, &index); err != nil {
		t.Fatalf("decode test catalog.json: %v", err)
	}
	index.Documents = append(index.Documents, nonItemResourcePaths...)
	indexJSON, err := json.Marshal(index)
	if err != nil {
		t.Fatalf("encode test catalog.json: %v", err)
	}
	files["catalog.json"] = &fstest.MapFile{Data: indexJSON}
	return files
}

// The Viewer is an item tool. A catalog that also stores resources of another
// kind must not panic it, and the family list must stay item-only.
func TestViewerIgnoresNonItemResources(t *testing.T) {
	data, err := loader.LoadFS(catalogFSWithNonItems(t))
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	server, err := New(data)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	families := server.families()
	want := []string{"ash_of_war", "weapon"}
	if len(families) != len(want) {
		t.Fatalf("families = %v, want %v", families, want)
	}
	for index := range want {
		if families[index] != want[index] {
			t.Fatalf("families = %v, want %v", families, want)
		}
	}

	if response := request(t, server.Handler(), http.MethodGet, "/"); response.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want %d", response.Code, http.StatusOK)
	}
}
