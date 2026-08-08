package gamecatalog_test

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/dbviewer"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// removedNameFields are the historical SaveForge name fields that the official
// FMG name contract replaced. None of them may survive in generated data.
var removedNameFields = []string{
	"displayName",
	"canonicalName",
	"displayNameSource",
	"canonicalSource",
}

// wantNameSourceByFamily pins every item family to the official FMG name
// catalog that owns its names. The source ID carries both the FMG file and
// whether the entry came from the base or the DLC extract.
var wantNameSourceByFamily = map[schema.ItemFamily][]schema.SourceID{
	schema.ItemFamilyWeapon:    {"game_text_weapon_name_base", "game_text_weapon_name_dlc"},
	schema.ItemFamilyArmor:     {"game_text_protector_name_base", "game_text_protector_name_dlc"},
	schema.ItemFamilyTalisman:  {"game_text_accessory_name_base", "game_text_accessory_name_dlc"},
	schema.ItemFamilyGoods:     {"game_text_goods_name_base", "game_text_goods_name_dlc"},
	schema.ItemFamilyGesture:   {"game_text_goods_name_base", "game_text_goods_name_dlc"},
	schema.ItemFamilySpell:     {"game_text_goods_name_base", "game_text_goods_name_dlc"},
	schema.ItemFamilySpiritAsh: {"game_text_goods_name_base", "game_text_goods_name_dlc"},
	schema.ItemFamilyAshOfWar:  {"game_text_gem_name_base", "game_text_gem_name_dlc"},
}

// TestEmbeddedNamesComeFromTheOfficialFMGPerFamily proves every family resolves
// its names from its own official FMG catalog, with provenance that names the
// exact FMG file and entry ID, and that every family is actually represented.
func TestEmbeddedNamesComeFromTheOfficialFMGPerFamily(t *testing.T) {
	resources := loadEmbeddedResources(t)
	seenFamilies := make(map[schema.ItemFamily]int, len(wantNameSourceByFamily))
	for _, resource := range resources {
		item := resource.Item
		family := item.Family.Value
		wantSources, supported := wantNameSourceByFamily[family]
		if !supported {
			t.Fatalf("resource %q has unsupported family %q", resource.Key, family)
		}
		assertNameProvenance(t, resource.Key, family, wantSources, item.Presentation.Name)
		seenFamilies[family]++
		for _, variant := range item.Variants {
			assertNameProvenance(
				t,
				resource.Key+" variant",
				family,
				wantSources,
				variant.Data.Presentation.Name,
			)
		}
	}
	for family := range wantNameSourceByFamily {
		if seenFamilies[family] == 0 {
			t.Errorf("family %q is not represented in the embedded catalog", family)
		}
	}
}

func assertNameProvenance(
	t *testing.T,
	label string,
	family schema.ItemFamily,
	wantSources []schema.SourceID,
	name schema.Fact[string],
) {
	t.Helper()
	if !name.Known {
		return
	}
	matched := false
	for _, want := range wantSources {
		if name.Provenance.Source == want {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf(
			"%s (family %q) name source = %q, want one of %v",
			label,
			family,
			name.Provenance.Source,
			wantSources,
		)
	}
	if !strings.Contains(name.Provenance.Method, "official English") ||
		!strings.Contains(name.Provenance.Method, ".fmg entry ") {
		t.Fatalf("%s name method = %q, want the FMG file and entry ID", label, name.Provenance.Method)
	}
}

// TestEmbeddedOrdinaryItemsUseTheOfficialFMGName spot-checks one ordinary item
// per family, including the cases where the official name replaces the old
// SaveForge text.
func TestEmbeddedOrdinaryItemsUseTheOfficialFMGName(t *testing.T) {
	cases := []struct {
		gameID uint32
		want   string
	}{
		{gameID: 0x000F4240, want: "Dagger"},                    // weapon
		{gameID: 0x1000C350, want: "Kaiden Helm"},               // armor
		{gameID: 0x200003E8, want: "Crimson Amber Medallion"},   // talisman
		{gameID: 0x400003E9, want: "Flask of Crimson Tears"},    // goods
		{gameID: 0x8000EA60, want: "Ash of War: Determination"}, // ash of war
	}
	byGameID := indexEmbeddedNames(t)
	for _, testCase := range cases {
		got, exists := byGameID[testCase.gameID]
		if !exists {
			t.Fatalf("game ID 0x%08X is absent from the embedded catalog", testCase.gameID)
		}
		if !got.Known || got.Value != testCase.want {
			t.Errorf("0x%08X name = %+v, want a known %q", testCase.gameID, got, testCase.want)
		}
	}
}

// TestEmbeddedVisionOfGraceHasUnknownName pins the single base item without any
// official FMG entry. It must stay in the catalog with an unknown name and must
// never fall back to the old SaveForge text.
func TestEmbeddedVisionOfGraceHasUnknownName(t *testing.T) {
	for _, resource := range loadEmbeddedResources(t) {
		if resource.Item.GameID.Value != visionOfGraceGameID {
			continue
		}
		name := resource.Item.Presentation.Name
		if name.Known || name.Value != "" {
			t.Fatalf("Vision of Grace name = %+v, want unknown and empty", name)
		}
		return
	}
	t.Fatalf("game ID 0x%08X is absent from the embedded catalog", visionOfGraceGameID)
}

// TestEmbeddedUnsafeItemsWithoutOfficialNameAreUnknown proves that cut-content
// and ban-risk items whose FMG entry is missing or an "[ERROR]" placeholder keep
// an unknown name instead of an old SaveForge fallback, and that they are still
// present with their safety flags intact.
func TestEmbeddedUnsafeItemsWithoutOfficialNameAreUnknown(t *testing.T) {
	unknownUnsafe := 0
	for _, resource := range loadEmbeddedResources(t) {
		item := resource.Item
		if item.Presentation.Name.Known {
			continue
		}
		if item.GameID.Value == visionOfGraceGameID {
			continue
		}
		if !item.Safety.CutContent.Value && !item.Safety.BanRisk.Value {
			t.Fatalf("resource %q has an unknown name without a safety flag", resource.Key)
		}
		if item.Presentation.Name.Value != "" {
			t.Fatalf("resource %q keeps a value on an unknown name", resource.Key)
		}
		unknownUnsafe++
	}
	if unknownUnsafe == 0 {
		t.Fatal("no cut-content or ban-risk item carries an unknown name")
	}
}

// TestEmbeddedVariantsCarryTheirOwnOfficialName proves a variant stores the
// official name of the variant itself, not the base item's name, and that an
// "[ERROR]" variant is unknown rather than inheriting the base name.
func TestEmbeddedVariantsCarryTheirOwnOfficialName(t *testing.T) {
	byGameID := indexEmbeddedNames(t)

	ordinary, exists := byGameID[0x000F42A4] // Heavy Dagger
	if !exists {
		t.Fatal("Heavy Dagger variant is absent from the embedded catalog")
	}
	if !ordinary.Known || ordinary.Value != "Heavy Dagger" {
		t.Errorf("Heavy Dagger variant name = %+v, want a known %q", ordinary, "Heavy Dagger")
	}

	placeholder, exists := byGameID[0x008A8D24] // "[ERROR]Heavy Serpentbone Blade"
	if !exists {
		t.Fatal("Heavy Serpentbone Blade variant is absent from the embedded catalog")
	}
	if placeholder.Known || placeholder.Value != "" {
		t.Errorf("[ERROR] variant name = %+v, want unknown and empty", placeholder)
	}

	base, exists := byGameID[0x008A8CC0] // Serpentbone Blade
	if !exists {
		t.Fatal("Serpentbone Blade base item is absent from the embedded catalog")
	}
	if !base.Known || base.Value != "Serpentbone Blade" {
		t.Errorf("Serpentbone Blade base name = %+v, want a known name", base)
	}
}

// TestEmbeddedDocumentsHaveNoRemovedNameFields proves the generated JSON no
// longer persists the historical SaveForge name fields anywhere.
func TestEmbeddedDocumentsHaveNoRemovedNameFields(t *testing.T) {
	files := catalogdata.Files()
	scanned := 0
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
		var document any
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatalf("unmarshal %s: %v", path, err)
		}
		assertNoRemovedNameKeys(t, path, document)
		scanned++
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded documents: %v", err)
	}
	if scanned == 0 {
		t.Fatal("no embedded item documents were scanned")
	}
}

func assertNoRemovedNameKeys(t *testing.T, path string, value any) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			for _, removed := range removedNameFields {
				if key == removed {
					t.Fatalf("document %s still contains %q", path, removed)
				}
			}
			assertNoRemovedNameKeys(t, path, nested)
		}
	case []any:
		for _, nested := range typed {
			assertNoRemovedNameKeys(t, path, nested)
		}
	}
}

// TestViewerFallsBackToResourceKeyForUnknownName proves the Viewer renders the
// technical resource key — never an old SaveForge name or a generated label —
// when item.presentation.name is unknown.
func TestViewerFallsBackToResourceKeyForUnknownName(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	server, err := dbviewer.New(data)
	if err != nil {
		t.Fatalf("dbviewer.New: %v", err)
	}
	handler := server.Handler()

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/items/400000A6", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "<h1>400000A6</h1>") {
		t.Fatal("Viewer does not render the technical resource key for an unknown name")
	}
	if strings.Contains(body, "Vision of Grace") || strings.Contains(body, "Memory of Grace") {
		t.Fatal("Viewer rendered a legacy SaveForge name for an unknown name")
	}

	// A variant whose own official name is an "[ERROR]" placeholder must fall
	// back to the same technical resource key, never to the base item's name and
	// never to a label rebuilt from the affinity.
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/items/008A8D24", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("variant status = %d, want 200", response.Code)
	}
	body = response.Body.String()
	if !strings.Contains(body, "<h1>008A8CC0</h1>") {
		t.Fatal("Viewer does not render the technical resource key for an unknown variant name")
	}
	if strings.Contains(body, "<h1>Serpentbone Blade") {
		t.Fatal("Viewer rendered the base item name for an unknown variant name")
	}
}

func loadEmbeddedResources(t *testing.T) []schema.Resource {
	t.Helper()
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	resources := data.Resources()
	if len(resources) == 0 {
		t.Fatal("embedded catalog has no resources")
	}
	return resources
}

func indexEmbeddedNames(t *testing.T) map[uint32]schema.Fact[string] {
	t.Helper()
	resources := loadEmbeddedResources(t)
	names := make(map[uint32]schema.Fact[string], len(resources))
	for _, resource := range resources {
		names[resource.Item.GameID.Value] = resource.Item.Presentation.Name
		for _, variant := range resource.Item.Variants {
			names[variant.GameID.Value] = variant.Data.Presentation.Name
		}
	}
	return names
}
