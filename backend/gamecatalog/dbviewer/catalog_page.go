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
}

func (server *Server) catalogHandler(response http.ResponseWriter, request *http.Request) {
	query := strings.TrimSpace(request.URL.Query().Get("q"))
	family := strings.TrimSpace(request.URL.Query().Get("family"))
	subcategory := strings.TrimSpace(request.URL.Query().Get("subcategory"))
	requestedPage, _ := strconv.Atoi(request.URL.Query().Get("page"))
	rows := server.catalogRowsFiltered(query, family, subcategory)
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
	}
	if currentPage > 1 {
		page.Previous = catalogPageURL(query, family, subcategory, currentPage-1)
	}
	if currentPage < pages {
		page.Next = catalogPageURL(query, family, subcategory, currentPage+1)
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
			resource.Item.Presentation.DisplayName.Value,
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
			Name:         resource.Item.Presentation.DisplayName.Value,
			IconURL:      itemIconURL(item),
			GameID:       canonicalID,
			GameIDPath:   strings.TrimPrefix(canonicalID, "0x"),
			Family:       string(item.Family.Value),
			Subcategory:  knownText(item.Subcategory.Known, item.Subcategory.Value),
			EntryType:    "Canonical",
			DocumentPath: document.Path,
			UnknownCount: countUnknownFactsForFamily(resource, item.Family.Value),
		}
		if matchesCatalogQuery(query, canonicalSearch...) {
			rows = append(rows, canonical)
		}

		for _, variant := range item.Variants {
			if !variant.GameID.Known || variant.GameID.Value == item.GameID.Value {
				continue
			}
			variantID := formatGameID(variant.GameID.Value)
			variantName := variantDisplayName(resource.Item.Presentation.DisplayName.Value, variant)
			variantSearch := []string{
				variantName,
				resource.Item.Presentation.DisplayName.Value,
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
				Name:         variantName,
				IconURL:      variantIconURL(item, variant),
				GameID:       variantID,
				GameIDPath:   strings.TrimPrefix(variantID, "0x"),
				Family:       string(item.Family.Value),
				Subcategory:  knownText(item.Subcategory.Known, item.Subcategory.Value),
				EntryType:    "Variant",
				DocumentPath: document.Path,
				UnknownCount: countUnknownFactsForFamily(variant, item.Family.Value),
			})
		}
	}
	sort.Slice(rows, func(left int, right int) bool {
		if rows[left].Name != rows[right].Name {
			return rows[left].Name < rows[right].Name
		}
		return rows[left].GameID < rows[right].GameID
	})
	return rows
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

func variantDisplayName(canonicalName string, variant schema.ItemVariant) string {
	if variant.Affinity.Known && variant.Affinity.Value != "" && variant.UpgradeLevel.Known {
		return fmt.Sprintf("%s (%s +%d)", canonicalName, variant.Affinity.Value, variant.UpgradeLevel.Value)
	}
	if variant.Affinity.Known && variant.Affinity.Value != "" {
		return fmt.Sprintf("%s (%s)", canonicalName, variant.Affinity.Value)
	}
	if variant.UpgradeLevel.Known {
		return fmt.Sprintf("%s (+%d)", canonicalName, variant.UpgradeLevel.Value)
	}
	return canonicalName + " (variant)"
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

func catalogPageURL(query string, family string, subcategory string, page int) string {
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
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/?" + encoded
	}
	return "/"
}
