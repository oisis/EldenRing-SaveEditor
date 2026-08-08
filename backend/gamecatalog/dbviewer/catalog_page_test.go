package dbviewer

import (
	"fmt"
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

func TestCatalogCanonicalUnknownCountExcludesVariants(t *testing.T) {
	rows := testServer(t).catalogRows("Dagger", "weapon")
	for _, row := range rows {
		if row.EntryType == "Canonical" && row.GameID == "0x000F4240" {
			if row.UnknownCount != 4 {
				t.Fatalf("Dagger canonical unknown count = %d, want 4", row.UnknownCount)
			}
			return
		}
	}
	t.Fatal("Dagger canonical catalog row is missing")
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
				Key: fmt.Sprintf("%08X", 0x70000000+uint32(index)),
				Item: &schema.ItemDocument{
					GameID: schema.Fact[uint32]{Known: true, Value: 0x70000000 + uint32(index)},
					Family: schema.Fact[schema.ItemFamily]{Known: true, Value: schema.ItemFamilyGoods},
					Presentation: schema.ItemPresentation{
						Name: schema.Fact[string]{Known: true, Value: "page-item"},
					},
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

func TestCatalogPageFiltersBySubcategory(t *testing.T) {
	response := request(t, testServer(t).Handler(), http.MethodGet, "/?subcategory=Daggers")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	body := response.Body.String()
	for _, expected := range []string{
		`<option value="Daggers" selected>Daggers</option>`,
		`href="/items/000F4240">Dagger</a>`,
		`href="/items/000F436C">Quality Dagger</a>`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("subcategory-filtered catalog does not contain %q", expected)
		}
	}
	if strings.Contains(body, ">Determination</a>") {
		t.Error("subcategory-filtered catalog unexpectedly contains Determination")
	}
}

func TestCatalogSubcategoryFilterIsPreservedInPaginationURL(t *testing.T) {
	got := catalogPageURL("needle", "weapon", "Daggers", catalogSortName, false, 2)
	want := "/?family=weapon&page=2&q=needle&subcategory=Daggers"
	if got != want {
		t.Fatalf("catalog pagination URL = %q, want %q", got, want)
	}
}

func TestCatalogSortAndPaginationURLsPreserveFilters(t *testing.T) {
	urls := catalogSortURLs("needle", "weapon", "Daggers", catalogSortUnknown, true)
	if got, want := urls[catalogSortUnknown], "/?family=weapon&q=needle&sort=unknown&subcategory=Daggers"; got != want {
		t.Fatalf("active sort URL = %q, want %q", got, want)
	}
	if got, want := urls[catalogSortName], "/?family=weapon&q=needle&subcategory=Daggers"; got != want {
		t.Fatalf("name sort URL = %q, want %q", got, want)
	}

	links := catalogPaginationLinks("needle", "weapon", "Daggers", catalogSortUnknown, true, 5, 10)
	var numbers []int
	for _, link := range links {
		if link.Ellipsis {
			numbers = append(numbers, 0)
			continue
		}
		numbers = append(numbers, link.Number)
	}
	if got, want := fmt.Sprint(numbers), "[1 0 3 4 5 6 7 0 10]"; got != want {
		t.Fatalf("pagination links = %s, want %s", got, want)
	}
	for _, link := range links {
		if link.Number == 4 && link.URL != "/?direction=desc&family=weapon&page=4&q=needle&sort=unknown&subcategory=Daggers" {
			t.Fatalf("page 4 URL = %q", link.URL)
		}
	}
}

func TestCatalogRowsSortByRequestedColumn(t *testing.T) {
	rows := []catalogItemRow{
		{Name: "Zulu", GameID: "0x00000003", EntryType: "Variant", Family: "goods", Subcategory: "Beta", UnknownCount: 2, DocumentPath: "items/c.json"},
		{Name: "Alpha", GameID: "0x00000002", EntryType: "Canonical", Family: "weapon", Subcategory: "Gamma", UnknownCount: 3, DocumentPath: "items/a.json"},
		{Name: "Bravo", GameID: "0x00000001", EntryType: "Upgrade", Family: "armor", Subcategory: "Alpha", UnknownCount: 1, DocumentPath: "items/b.json"},
	}
	tests := []struct {
		sortField string
		want      string
	}{
		{catalogSortName, "0x00000002,0x00000001,0x00000003"},
		{catalogSortGameID, "0x00000001,0x00000002,0x00000003"},
		{catalogSortEntry, "0x00000002,0x00000001,0x00000003"},
		{catalogSortFamily, "0x00000001,0x00000003,0x00000002"},
		{catalogSortSubcategory, "0x00000001,0x00000003,0x00000002"},
		{catalogSortUnknown, "0x00000001,0x00000003,0x00000002"},
		{catalogSortDocument, "0x00000002,0x00000001,0x00000003"},
	}
	for _, test := range tests {
		t.Run(test.sortField, func(t *testing.T) {
			copyRows := append([]catalogItemRow(nil), rows...)
			sortCatalogRows(copyRows, test.sortField, false)
			ids := make([]string, len(copyRows))
			for index, row := range copyRows {
				ids[index] = row.GameID
			}
			if got := strings.Join(ids, ","); got != test.want {
				t.Fatalf("sorted IDs = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCatalogSubcategoryFilterOptionsAreUniqueAndSorted(t *testing.T) {
	server := testServer(t)
	server.data.Documents = append(server.data.Documents, loader.Document{
		Path: "items/weapon/another-dagger.json",
		Resource: schema.Resource{
			Item: &schema.ItemDocument{
				Subcategory: schema.Fact[string]{Known: true, Value: "Daggers"},
			},
		},
	})

	got := server.subcategories()
	if len(got) != 1 || got[0] != "Daggers" {
		t.Fatalf("subcategories = %v, want [Daggers]", got)
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
		"Quality Dagger",
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
				Key: fmt.Sprintf("%08X", uint32(index+1)),
				Item: &schema.ItemDocument{
					GameID: schema.Fact[uint32]{Known: true, Value: uint32(index + 1)},
					Family: schema.Fact[schema.ItemFamily]{Known: true, Value: family},
					Presentation: schema.ItemPresentation{
						Name: schema.Fact[string]{Known: true, Value: string(family)},
					},
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
