package dbviewer

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const catalogPageSize = 100

const (
	catalogSortName        = "name"
	catalogSortGameID      = "gameID"
	catalogSortEntry       = "entry"
	catalogSortFamily      = "family"
	catalogSortSubcategory = "subcategory"
	catalogSortUnknown     = "unknown"
	catalogSortDocument    = "document"
)

type catalogPaginationLink struct {
	Number   int
	URL      string
	Current  bool
	Ellipsis bool
}

type catalogPage struct {
	Meta          pageMeta
	Query         string
	Family        string
	Families      []string
	Subcategory   string
	Subcategories []string
	Items         []catalogItemRow
	Total         int
	First         int
	Last          int
	Page          int
	Pages         int
	Previous      string
	Next          string
	Sort          string
	Descending    bool
	SortURLs      map[string]string
	PageLinks     []catalogPaginationLink
}

type catalogItemRow struct {
	Name         string
	IconURL      string
	GameID       string
	GameIDPath   string
	Family       string
	Subcategory  string
	EntryType    string
	DocumentPath string
	UnknownCount int
	CutContent   bool
}

func (server *Server) catalogHandler(response http.ResponseWriter, request *http.Request) {
	query := strings.TrimSpace(request.URL.Query().Get("q"))
	family := strings.TrimSpace(request.URL.Query().Get("family"))
	subcategory := strings.TrimSpace(request.URL.Query().Get("subcategory"))
	sortField := catalogSortField(request.URL.Query().Get("sort"))
	descending := request.URL.Query().Get("direction") == "desc"
	requestedPage, _ := strconv.Atoi(request.URL.Query().Get("page"))
	rows := server.catalogRowsFiltered(query, family, subcategory)
	sortCatalogRows(rows, sortField, descending)
	items, currentPage, pages, first, last := paginateCatalogRows(rows, requestedPage)
	page := catalogPage{
		Meta:          server.pageMeta("Catalog"),
		Query:         query,
		Family:        family,
		Families:      server.families(),
		Subcategory:   subcategory,
		Subcategories: server.subcategories(),
		Items:         items,
		Total:         len(rows),
		First:         first,
		Last:          last,
		Page:          currentPage,
		Pages:         pages,
		Sort:          sortField,
		Descending:    descending,
		SortURLs:      catalogSortURLs(query, family, subcategory, sortField, descending),
		PageLinks:     catalogPaginationLinks(query, family, subcategory, sortField, descending, currentPage, pages),
	}
	if currentPage > 1 {
		page.Previous = catalogPageURL(query, family, subcategory, sortField, descending, currentPage-1)
	}
	if currentPage < pages {
		page.Next = catalogPageURL(query, family, subcategory, sortField, descending, currentPage+1)
	}
	server.render(response, "catalog", page)
}

func (server *Server) catalogRows(query string, family string) []catalogItemRow {
	return server.catalogRowsFiltered(query, family, "")
}

func (server *Server) catalogRowsFiltered(query string, family string, subcategory string) []catalogItemRow {
	query = strings.ToLower(query)
	rows := make([]catalogItemRow, 0, len(server.data.Documents))
	for _, document := range server.data.Documents {
		resource := document.Resource
		item := resource.Item
		if item == nil {
			continue
		}
		if family != "" && string(item.Family.Value) != family {
			continue
		}
		if subcategory != "" &&
			(!item.Subcategory.Known || item.Subcategory.Value != subcategory) {
			continue
		}

		canonicalID := formatGameID(item.GameID.Value)
		canonicalSearch := []string{
			itemName(resource),
			resource.Key,
			canonicalID,
			string(item.Family.Value),
			item.Category.Value,
			item.Subcategory.Value,
		}
		for _, alias := range item.Aliases {
			if alias.GameID.Known {
				canonicalSearch = append(canonicalSearch, formatGameID(alias.GameID.Value))
			}
		}
		canonical := catalogItemRow{
			Name:         itemName(resource),
			IconURL:      itemIconURL(item),
			GameID:       canonicalID,
			GameIDPath:   strings.TrimPrefix(canonicalID, "0x"),
			Family:       string(item.Family.Value),
			Subcategory:  knownText(item.Subcategory.Known, item.Subcategory.Value),
			EntryType:    "Canonical",
			DocumentPath: document.Path,
			UnknownCount: countUnknownFactsForCanonicalItem(item),
			CutContent:   item.Safety.CutContent.Known && item.Safety.CutContent.Value,
		}
		if matchesCatalogQuery(query, canonicalSearch...) {
			rows = append(rows, canonical)
		}

		for _, variant := range item.Variants {
			if !variant.GameID.Known || variant.GameID.Value == item.GameID.Value {
				continue
			}
			variantID := formatGameID(variant.GameID.Value)
			name := variantName(resource, variant)
			variantSearch := []string{
				name,
				itemName(resource),
				resource.Key,
				variantID,
				string(item.Family.Value),
				string(variant.Kind.Value),
				item.Category.Value,
				item.Subcategory.Value,
				string(variant.Affinity.Value),
				fmt.Sprint(variant.UpgradeLevel.Value),
			}
			if !matchesCatalogQuery(query, variantSearch...) {
				continue
			}
			rows = append(rows, catalogItemRow{
				Name:         name,
				IconURL:      variantIconURL(item, variant),
				GameID:       variantID,
				GameIDPath:   strings.TrimPrefix(variantID, "0x"),
				Family:       string(item.Family.Value),
				Subcategory:  knownText(item.Subcategory.Known, item.Subcategory.Value),
				EntryType:    "Variant",
				DocumentPath: document.Path,
				UnknownCount: countUnknownFactsForFamily(variant, item.Family.Value),
				CutContent:   variant.Data.Safety.CutContent.Known && variant.Data.Safety.CutContent.Value,
			})
		}
	}
	sortCatalogRows(rows, catalogSortName, false)
	return rows
}

func catalogSortField(value string) string {
	switch value {
	case catalogSortGameID, catalogSortEntry, catalogSortFamily,
		catalogSortSubcategory, catalogSortUnknown, catalogSortDocument:
		return value
	default:
		return catalogSortName
	}
}

func sortCatalogRows(rows []catalogItemRow, sortField string, descending bool) {
	sort.Slice(rows, func(left, right int) bool {
		compare := compareCatalogRows(rows[left], rows[right], sortField)
		if compare == 0 {
			compare = strings.Compare(rows[left].Name, rows[right].Name)
		}
		if compare == 0 {
			compare = strings.Compare(rows[left].GameID, rows[right].GameID)
		}
		if descending {
			return compare > 0
		}
		return compare < 0
	})
}

func compareCatalogRows(left, right catalogItemRow, sortField string) int {
	switch sortField {
	case catalogSortGameID:
		return strings.Compare(left.GameID, right.GameID)
	case catalogSortEntry:
		return strings.Compare(left.EntryType, right.EntryType)
	case catalogSortFamily:
		return strings.Compare(left.Family, right.Family)
	case catalogSortSubcategory:
		return strings.Compare(left.Subcategory, right.Subcategory)
	case catalogSortUnknown:
		return left.UnknownCount - right.UnknownCount
	case catalogSortDocument:
		return strings.Compare(left.DocumentPath, right.DocumentPath)
	default:
		return strings.Compare(left.Name, right.Name)
	}
}

func matchesCatalogQuery(query string, values ...string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(strings.Join(values, " ")), query)
}

func knownText(known bool, value string) string {
	if !known {
		return "Unknown"
	}
	return value
}

// itemName is the Viewer's single item-name source: item.presentation.name,
// falling back to the technical resource key when that name is unknown.
func itemName(resource schema.Resource) string {
	return presentationName(resource.Item.Presentation.Name, resource.Key)
}

func presentationName(name schema.Fact[string], key string) string {
	if name.Known && name.Value != "" {
		return name.Value
	}
	return key
}

// variantName is the Viewer's single variant-name source: the variant's own
// official FMG name in variant.data.presentation.name, falling back to the
// technical resource key when that name is unknown. The name already carries
// the affinity and upgrade level, so the Viewer must not decorate it again.
func variantName(resource schema.Resource, variant schema.ItemVariant) string {
	return presentationName(variant.Data.Presentation.Name, resource.Key)
}

func (server *Server) families() []string {
	seen := make(map[schema.ItemFamily]struct{})
	for _, document := range server.data.Documents {
		seen[document.Resource.Item.Family.Value] = struct{}{}
	}
	families := make([]string, 0, len(seen))
	for family := range seen {
		families = append(families, string(family))
	}
	sort.Strings(families)
	return families
}

func (server *Server) subcategories() []string {
	seen := make(map[string]struct{})
	for _, document := range server.data.Documents {
		item := document.Resource.Item
		if item == nil || !item.Subcategory.Known || item.Subcategory.Value == "" {
			continue
		}
		seen[item.Subcategory.Value] = struct{}{}
	}
	subcategories := make([]string, 0, len(seen))
	for subcategory := range seen {
		subcategories = append(subcategories, subcategory)
	}
	sort.Strings(subcategories)
	return subcategories
}

func paginateCatalogRows(
	rows []catalogItemRow,
	requestedPage int,
) ([]catalogItemRow, int, int, int, int) {
	total := len(rows)
	pages := (total + catalogPageSize - 1) / catalogPageSize
	if pages == 0 {
		pages = 1
	}
	page := requestedPage
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * catalogPageSize
	end := min(start+catalogPageSize, total)
	if start == end {
		return nil, page, pages, 0, 0
	}
	return rows[start:end], page, pages, start + 1, end
}

func catalogSortURLs(
	query string,
	family string,
	subcategory string,
	currentSort string,
	descending bool,
) map[string]string {
	fields := []string{
		catalogSortName,
		catalogSortGameID,
		catalogSortEntry,
		catalogSortFamily,
		catalogSortSubcategory,
		catalogSortUnknown,
		catalogSortDocument,
	}
	urls := make(map[string]string, len(fields))
	for _, sortField := range fields {
		nextDescending := sortField == currentSort && !descending
		urls[sortField] = catalogPageURL(
			query,
			family,
			subcategory,
			sortField,
			nextDescending,
			1,
		)
	}
	return urls
}

func catalogPaginationLinks(
	query string,
	family string,
	subcategory string,
	sortField string,
	descending bool,
	currentPage int,
	pages int,
) []catalogPaginationLink {
	numbers := []int{1, currentPage - 2, currentPage - 1, currentPage, currentPage + 1, currentPage + 2, pages}
	seen := make(map[int]struct{}, len(numbers))
	valid := make([]int, 0, len(numbers))
	for _, number := range numbers {
		if number < 1 || number > pages {
			continue
		}
		if _, exists := seen[number]; exists {
			continue
		}
		seen[number] = struct{}{}
		valid = append(valid, number)
	}
	sort.Ints(valid)

	links := make([]catalogPaginationLink, 0, len(valid)+2)
	previous := 0
	for _, number := range valid {
		if previous != 0 && number > previous+1 {
			links = append(links, catalogPaginationLink{Ellipsis: true})
		}
		links = append(links, catalogPaginationLink{
			Number:  number,
			URL:     catalogPageURL(query, family, subcategory, sortField, descending, number),
			Current: number == currentPage,
		})
		previous = number
	}
	return links
}

func catalogPageURL(
	query string,
	family string,
	subcategory string,
	sortField string,
	descending bool,
	page int,
) string {
	values := make(url.Values)
	if query != "" {
		values.Set("q", query)
	}
	if family != "" {
		values.Set("family", family)
	}
	if subcategory != "" {
		values.Set("subcategory", subcategory)
	}
	if sortField != catalogSortName || descending {
		values.Set("sort", sortField)
	}
	if descending {
		values.Set("direction", "desc")
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/?" + encoded
	}
	return "/"
}
