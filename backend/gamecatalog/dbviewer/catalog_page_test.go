package dbviewer

import (
	"net/http"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestCatalogPageListsBothDocuments(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		"Dagger",
		"Determination",
		"items/weapon/000f4240.json",
		"items/ash_of_war/8000ea60.json",
		`src="/catalog-assets/icons/items/melee_armaments/dagger.png"`,
		`src="/catalog-assets/icons/items/ashes_of_war/determination.png"`,
		`loading="lazy"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("catalog page does not contain %q", expected)
		}
	}
}

func TestCatalogPaginationBoundaries(t *testing.T) {
	rows := make([]catalogItemRow, 205)
	for index := range rows {
		rows[index].Name = "row"
	}

	first, page, pages, from, to := paginateCatalogRows(rows, 0)
	if len(first) != 100 || page != 1 || pages != 3 || from != 1 || to != 100 {
		t.Fatalf("first page = len %d, page %d/%d, %d-%d", len(first), page, pages, from, to)
	}
	second, page, pages, from, to := paginateCatalogRows(rows, 2)
	if len(second) != 100 || page != 2 || pages != 3 || from != 101 || to != 200 {
		t.Fatalf("second page = len %d, page %d/%d, %d-%d", len(second), page, pages, from, to)
	}
	last, page, pages, from, to := paginateCatalogRows(rows, 99)
	if len(last) != 5 || page != 3 || pages != 3 || from != 201 || to != 205 {
		t.Fatalf("last page = len %d, page %d/%d, %d-%d", len(last), page, pages, from, to)
	}
	empty, page, pages, from, to := paginateCatalogRows(nil, 5)
	if len(empty) != 0 || page != 1 || pages != 1 || from != 0 || to != 0 {
		t.Fatalf("empty page = len %d, page %d/%d, %d-%d", len(empty), page, pages, from, to)
	}
}

func TestCatalogPaginationPreservesQueryAndFamily(t *testing.T) {
	server := testServer(t)
	for index := 0; index < 120; index++ {
		server.data.Documents = append(server.data.Documents, loader.Document{
			Path: "items/goods/page-item.json",
			Resource: schema.Resource{
				Key:   "page-item",
				Label: schema.Fact[string]{Known: true, Value: "page-item"},
				Item: &schema.ItemDocument{
					GameID:      schema.Fact[uint32]{Known: true, Value: 0x70000000 + uint32(index)},
					Family:      schema.Fact[schema.ItemFamily]{Known: true, Value: schema.ItemFamilyGoods},
					Subcategory: schema.Fact[string]{Known: true, Value: "page-test"},
				},
			},
		})
	}

	response := request(
		t,
		server.Handler(),
		http.MethodGet,
		"/?family=goods&q=page-item&page=2",
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		"101–120 of 120",
		"Page 2 of 2",
		`href="/?family=goods&amp;q=page-item"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("second page does not contain %q", expected)
		}
	}
	if strings.Contains(body, "Next</a>") {
		t.Error("last page unexpectedly contains Next link")
	}
}

func TestCatalogPageFiltersByFamily(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/?family=ash_of_war")
	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if !strings.Contains(body, "Determination") {
		t.Error("filtered catalog does not contain Determination")
	}
	if strings.Contains(body, ">Dagger</a>") {
		t.Error("filtered catalog unexpectedly contains Dagger row")
	}
}

func TestCatalogPageSearchesHexadecimalGameID(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/?q=0x000F4240")
	body := response.Body.String()
	if !strings.Contains(body, ">Dagger</a>") {
		t.Error("hexadecimal search did not find Dagger")
	}
	if strings.Contains(body, ">Determination</a>") {
		t.Error("hexadecimal search unexpectedly found Determination")
	}
}

func TestCatalogPageListsVariantsAsSearchableEntries(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/?q=0x000F436C")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		`href="/items/000F436C"`,
		"Dagger (quality)",
		"<code>0x000F436C</code>",
		">Variant</span>",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("variant search result does not contain %q", expected)
		}
	}
	if strings.Contains(body, `href="/items/000F4240">Dagger</a>`) {
		t.Error("variant-only search unexpectedly contains the canonical row")
	}
}

func TestCatalogRowsAndFiltersCoverAllEightItemFamilies(t *testing.T) {
	families := []schema.ItemFamily{
		schema.ItemFamilyWeapon,
		schema.ItemFamilyArmor,
		schema.ItemFamilyTalisman,
		schema.ItemFamilyAshOfWar,
		schema.ItemFamilySpell,
		schema.ItemFamilySpiritAsh,
		schema.ItemFamilyGoods,
		schema.ItemFamilyGesture,
	}
	server := &Server{}
	for index, family := range families {
		server.data.Documents = append(server.data.Documents, loader.Document{
			Path: "items/" + string(family) + "/item.json",
			Resource: schema.Resource{
				Key:   "item:" + string(family),
				Label: schema.Fact[string]{Known: true, Value: string(family)},
				Item: &schema.ItemDocument{
					GameID:      schema.Fact[uint32]{Known: true, Value: uint32(index + 1)},
					Family:      schema.Fact[schema.ItemFamily]{Known: true, Value: family},
					Subcategory: schema.Fact[string]{Known: true, Value: "test"},
				},
			},
		})
	}

	rows := server.catalogRows("", "")
	if len(rows) != len(families) {
		t.Fatalf("catalog row count = %d, want %d", len(rows), len(families))
	}
	filterValues := server.families()
	if len(filterValues) != len(families) {
		t.Fatalf("family filter count = %d, want %d", len(filterValues), len(families))
	}
	for _, family := range families {
		if rows := server.catalogRows("", string(family)); len(rows) != 1 || rows[0].Family != string(family) {
			t.Fatalf("family %q rows = %+v", family, rows)
		}
	}
}
