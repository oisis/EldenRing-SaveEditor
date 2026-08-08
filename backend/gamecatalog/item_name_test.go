package gamecatalog_test

import (
	"encoding/json"
	"fmt"
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

var wantLegacyNameFallbacks = map[uint32]string{
	0x400000A6: "Vision of Grace",
	0x4000D17E: "?GoodsName? Holy Water Pot",
	0x40002354: "?GoodsName?",
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
		if want, legacyFallback := wantLegacyNameFallbacks[item.GameID.Value]; legacyFallback {
			assertLegacyNameFallback(t, resource.Key, item.Presentation.Name, want)
		} else {
			assertNameProvenance(t, resource.Key, family, wantSources, item.Presentation.Name)
		}
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

func assertLegacyNameFallback(t *testing.T, label string, name schema.Fact[string], want string) {
	t.Helper()
	if !name.Known || name.Value != want {
		t.Fatalf("%s fallback name = %#v, want known %q", label, name, want)
	}
	if name.Provenance.Source != schema.SourceSaveForgeLegacy ||
		!strings.Contains(name.Provenance.Method, "no usable official FMG name entry exists") {
		t.Fatalf("%s fallback name provenance = %#v", label, name.Provenance)
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

func TestEmbeddedLegacyNameFallbacksKeepSafetyMarkers(t *testing.T) {
	remaining := make(map[uint32]string, len(wantLegacyNameFallbacks))
	for gameID, name := range wantLegacyNameFallbacks {
		remaining[gameID] = name
	}
	for _, resource := range loadEmbeddedResources(t) {
		item := resource.Item
		want, expected := remaining[item.GameID.Value]
		if !expected {
			continue
		}
		assertLegacyNameFallback(t, resource.Key, item.Presentation.Name, want)
		if item.GameID.Value != 0x400000A6 && !item.Safety.CutContent.Value {
			t.Fatalf("resource %q lost its cut-content marker", resource.Key)
		}
		delete(remaining, item.GameID.Value)
	}
	if len(remaining) != 0 {
		t.Fatalf("legacy name fallbacks are absent: %v", remaining)
	}
}

// TestEmbeddedVariantsCarryTheirOwnOfficialName proves a playable variant
// stores its own official name rather than inheriting the base item's name.
func TestEmbeddedVariantsCarryTheirOwnOfficialName(t *testing.T) {
	byGameID := indexEmbeddedNames(t)

	ordinary, exists := byGameID[0x000F42A4] // Heavy Dagger
	if !exists {
		t.Fatal("Heavy Dagger variant is absent from the embedded catalog")
	}
	if !ordinary.Known || ordinary.Value != "Heavy Dagger" {
		t.Errorf("Heavy Dagger variant name = %+v, want a known %q", ordinary, "Heavy Dagger")
	}

	base, exists := byGameID[0x008A8CC0] // Serpentbone Blade
	if !exists {
		t.Fatal("Serpentbone Blade base item is absent from the embedded catalog")
	}
	if !base.Known || base.Value != "Serpentbone Blade" {
		t.Errorf("Serpentbone Blade base name = %+v, want a known name", base)
	}
}

func TestEmbeddedCutContentRetainsOfficialFMGName(t *testing.T) {
	byGameID := indexEmbeddedNames(t)
	name, exists := byGameID[0x100AAE60] // "[ERROR]Brave's Cord Circlet"
	if !exists {
		t.Fatal("Brave's Cord Circlet is absent from the embedded catalog")
	}
	if !name.Known || name.Value != "Brave's Cord Circlet" {
		t.Fatalf("Brave's Cord Circlet name = %+v", name)
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

// TestViewerRendersEstablishedLegacyNameFallbacks proves that exceptional
// records missing usable FMG text retain their established names, and unsafe
// ones remain visibly marked as cut content.
func TestViewerRendersEstablishedLegacyNameFallbacks(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	server, err := dbviewer.New(data)
	if err != nil {
		t.Fatalf("dbviewer.New: %v", err)
	}
	handler := server.Handler()

	for _, test := range []struct {
		gameID     uint32
		name       string
		cutContent bool
	}{
		{gameID: 0x400000A6, name: "Vision of Grace"},
		{gameID: 0x4000D17E, name: "?GoodsName? Holy Water Pot", cutContent: true},
		{gameID: 0x40002354, name: "?GoodsName?", cutContent: true},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/items/%08X", test.gameID),
			nil,
		))
		if response.Code != http.StatusOK {
			t.Fatalf("0x%08X status = %d, want 200", test.gameID, response.Code)
		}
		body := response.Body.String()
		heading := "<h1>" + test.name + "</h1>"
		if test.cutContent {
			heading = `<h1 class="cut-content">` + test.name + "</h1>"
		}
		if !strings.Contains(body, heading) {
			t.Fatalf("0x%08X does not render %q", test.gameID, heading)
		}
		if test.cutContent && !strings.Contains(body, `<p class="cut-content-label">Cut content</p>`) {
			t.Fatalf("0x%08X lacks the cut-content marker", test.gameID)
		}
	}

	// A playable variant renders its own official name rather than its base
	// weapon's name.
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/items/000F42A4", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("variant status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, "<h1>Heavy Dagger</h1>") {
		t.Fatal("Viewer does not render the official variant name")
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
