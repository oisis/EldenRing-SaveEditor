package dbviewer

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestItemPageShowsHumanReadableDataAndRelations(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/items/000F4240")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := html.UnescapeString(response.Body.String())
	for _, expected := range []string{
		"Maximum level: +25",
		"Allowed: standard, heavy, keen",
		"Physical attack",
		"Determination",
		"regulation.bin/csv/EquipParamWeapon.csv",
		`src="/catalog-assets/icons/items/melee_armaments/dagger.png"`,
		`href="/items/000F4240/raw"`,
		">4 unknown fields</span>",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("item page does not contain %q", expected)
		}
	}
}

func TestItemPageResolvesVariantToCanonicalDocument(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/items/000F436C")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		"items/weapon/000f4240.json",
		"Quality Dagger",
		"<code>0x000F436C</code>",
		`href="/items/000F436C/raw"`,
		">Variant</span>",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("variant page does not contain %q", expected)
		}
	}
}

func TestItemPageDoesNotExposeAffinityRowsForFixedWeapons(t *testing.T) {
	server := testServer(t)
	for _, test := range []struct {
		name       string
		baseGameID uint32
	}{
		{name: "Serpentbone Blade", baseGameID: 0x008A8CC0},
		{name: "Treespear", baseGameID: 0x010477B0},
		{name: "Great Club", baseGameID: 0x015F41E0},
		{name: "Troll's Hammer", baseGameID: 0x0160EF90},
	} {
		t.Run(test.name, func(t *testing.T) {
			affinityGameID := test.baseGameID + 100
			if _, exists := server.catalog.ItemViewByGameID(affinityGameID); exists {
				t.Fatalf("0x%08X is exposed as a playable variant", affinityGameID)
			}
			response := request(t, server.Handler(), http.MethodGet, fmt.Sprintf("/items/%08X", affinityGameID))
			if response.Code != http.StatusNotFound {
				t.Fatalf("0x%08X status = %d, want 404", affinityGameID, response.Code)
			}
		})
	}
}

func TestItemPageMarksCutContentWithoutAColoredBackground(t *testing.T) {
	server := testServerWithCutContentDagger(t)

	response := request(t, server.Handler(), http.MethodGet, "/items/000F4240")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		`<h1 class="cut-content">Dagger</h1>`,
		`<p class="cut-content-label">Cut content</p>`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("item cut-content page does not contain %q", expected)
		}
	}
}

func TestItemPageReturnsNotFoundForInvalidOrUnknownID(t *testing.T) {
	handler := testServer(t).Handler()
	for _, target := range []string{"/items/not-a-number", "/items/DEADBEEF"} {
		response := request(t, handler, http.MethodGet, target)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", target, response.Code)
		}
	}
}

func TestVariantItemPageShowsOnlyExactVariantSourceRecords(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	var item *schema.ItemDocument
	for index := range data.Documents {
		candidate := data.Documents[index].Resource.Item
		if candidate != nil && candidate.GameID.Value == 0x000F4240 {
			item = candidate
			break
		}
	}
	if item == nil {
		t.Fatal("Dagger document is missing")
	}
	var variant *schema.ItemVariant
	for index := range item.Variants {
		if item.Variants[index].GameID.Value != 1000300 {
			continue
		}
		variant = &item.Variants[index]
		break
	}
	if variant == nil {
		t.Fatal("quality Dagger variant is missing")
	}
	server, err := New(data)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	view, exists := server.catalog.ItemViewByGameID(1000300)
	if !exists {
		t.Fatal("quality Dagger view is missing")
	}
	document := server.documentsByRef[view.Resource.Ref()]
	page := server.buildItemPage(view, document, 1000300)
	if len(page.SourceRecords) != len(variant.SourceRecords) {
		t.Fatalf("variant source records = %+v", page.SourceRecords)
	}
	expectedRecords := make(map[string]struct{}, len(variant.SourceRecords))
	for _, record := range variant.SourceRecords {
		expectedRecords[fmt.Sprintf("%s:%d", record.Table, record.RowID)] = struct{}{}
	}
	for _, record := range page.SourceRecords {
		key := fmt.Sprintf("%s:%d", record.Table, record.RowID)
		if _, exists := expectedRecords[key]; !exists {
			t.Fatalf("variant page contains canonical-only source record %s", key)
		}
	}
	if _, canonicalPrimaryShown := expectedRecords["EquipParamWeapon:1000000"]; canonicalPrimaryShown {
		t.Fatal("quality Dagger variant contains canonical primary row 1000000")
	}

	response := request(t, server.Handler(), http.MethodGet, "/items/000F436C")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{"row 1000300"} {
		if !strings.Contains(body, expected) {
			t.Errorf("variant item page does not contain %q", expected)
		}
	}
}

func TestVariantItemPageUsesFullVariantData(t *testing.T) {
	server := testServer(t)
	view, exists := server.catalog.ItemViewByGameID(1000300)
	if !exists {
		t.Fatal("quality Dagger variant is missing")
	}
	variantFound := false
	for index := range view.Resource.Item.Variants {
		variant := &view.Resource.Item.Variants[index]
		if variant.GameID.Value != 1000300 {
			continue
		}
		variant.Data.Storage.MaxInventory = schema.Fact[uint32]{
			Known:      true,
			Value:      2,
			Provenance: variant.Data.Storage.RecordMode.Provenance,
		}
		variant.Data.Weapon.AttackPhysical = schema.Fact[int32]{
			Known: true,
			Value: 999,
		}
		variantFound = true
		break
	}
	if !variantFound {
		t.Fatal("quality Dagger variant is missing from resource")
	}
	document := server.documentsByRef[view.Resource.Ref()]

	page := server.buildItemPage(view, document, 1000300)
	if page.Name != "Quality Dagger" {
		t.Fatalf("variant name = %q", page.Name)
	}
	if !containsFact(page.Storage, "Maximum inventory", "2") {
		t.Fatalf("variant storage facts = %+v", page.Storage)
	}
	if !containsFact(page.FamilyData, "Physical attack", "999") {
		t.Fatalf("variant family facts = %+v", page.FamilyData)
	}
}

func containsFact(facts []factView, label string, value string) bool {
	for _, fact := range facts {
		if fact.Label == label && fact.Value == value {
			return true
		}
	}
	return false
}
