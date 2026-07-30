package dbviewer

import (
	"net/http"
	"sort"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type catalogPage struct {
	Meta     pageMeta
	Query    string
	Family   string
	Families []string
	Items    []catalogItemRow
}

type catalogItemRow struct {
	Name         string
	GameID       string
	GameIDPath   string
	Family       string
	Subcategory  string
	DocumentPath string
	UnknownCount int
}

func (server *Server) catalogHandler(response http.ResponseWriter, request *http.Request) {
	query := strings.TrimSpace(request.URL.Query().Get("q"))
	family := strings.TrimSpace(request.URL.Query().Get("family"))
	page := catalogPage{
		Meta:     server.pageMeta("Catalog"),
		Query:    query,
		Family:   family,
		Families: server.families(),
		Items:    server.catalogRows(query, family),
	}
	server.render(response, "catalog", page)
}

func (server *Server) catalogRows(query string, family string) []catalogItemRow {
	query = strings.ToLower(query)
	rows := make([]catalogItemRow, 0, len(server.data.Documents))
	for _, document := range server.data.Documents {
		resource := document.Resource
		item := resource.Item
		if family != "" && string(item.Family.Value) != family {
			continue
		}
		gameID := formatGameID(item.GameID.Value)
		searchable := strings.ToLower(strings.Join([]string{
			resource.Label.Value,
			resource.Key,
			gameID,
			string(item.Family.Value),
			item.Subcategory.Value,
		}, " "))
		if query != "" && !strings.Contains(searchable, query) {
			continue
		}
		subcategory := item.Subcategory.Value
		if !item.Subcategory.Known {
			subcategory = "Unknown"
		}
		rows = append(rows, catalogItemRow{
			Name:         resource.Label.Value,
			GameID:       gameID,
			GameIDPath:   strings.TrimPrefix(gameID, "0x"),
			Family:       string(item.Family.Value),
			Subcategory:  subcategory,
			DocumentPath: document.Path,
			UnknownCount: countUnknownFacts(resource),
		})
	}
	sort.Slice(rows, func(left int, right int) bool {
		return rows[left].Name < rows[right].Name
	})
	return rows
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
