/*
Endpoint: GetItemDatabase
EndpointID: get_item_database
Purpose: Returns one authoritative page of the general Item Database: the item resources the active Safety Profile allows the interface to present, filtered, sorted and paged by the backend.
How it works: The runtime handler reads the already loaded GameCatalog through Catalog.ResourceSummaries, applies the visibility rule of the shared safety-profile policy, then the declared filters, then one of the closed sort orders, and finally slices one page out of the complete match set. It loads, reloads or modifies no catalog, opens no save and calls no other endpoint.
Supported resource types: ItemDocument.
Input variables: safetyProfile, family, category, search, favoritesOnly, favorites, sort, page, pageSize.
GameCatalog variables read: Resource.Kind, Resource.Key, Item.GameID, Item.Family, Item.Category, Item.Subcategory, Item.Presentation.Name, Item.Presentation.IconPath and the six Item.Safety flags. The full resource document is never projected; it stays the responsibility of GetResource.
Save variables read: none; the endpoint never opens or reads a save.
Implementation status: implemented
*/
package catalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
)

// GetItemDatabaseEndpointID is the stable backend identifier of GetItemDatabase.
const GetItemDatabaseEndpointID = "get_item_database"

// GetItemDatabaseDefaultPageSize is the page size used when the caller passes 0.
const GetItemDatabaseDefaultPageSize = 50

// The closed vocabulary of authoritative sort orders. The empty value is the
// catalog order — kind first and only then key — which is the same stable order
// every other catalog list uses. There is deliberately no free-form sort
// expression: a client selects one of these, and the comparison itself stays
// here so two screens can never order the same catalog differently.
const (
	GetItemDatabaseSortCatalog  = ""
	GetItemDatabaseSortName     = "name"
	GetItemDatabaseSortCategory = "category"
	GetItemDatabaseSortGameID   = "game_id"
)

// GetItemDatabaseDefinition describes the public getter contract.
var GetItemDatabaseDefinition = contract.MustDefine(contract.Definition{
	Name:                   "GetItemDatabase",
	ID:                     GetItemDatabaseEndpointID,
	Kind:                   contract.Getter,
	SupportedResourceTypes: "ItemDocument",
	SupportedResourceVariables: []string{
		"safetyProfile", "family", "category", "search", "favoritesOnly", "favorites",
		"sort", "page", "pageSize",
	},
	Description: "Returns one authoritative page of the general Item Database under the active Safety Profile.",
})

// ItemDatabaseEntry is one row of the Item Database. It is a list-sized
// projection: limits, statistics, capabilities, provenance and variants are
// deliberately absent and stay the responsibility of GetResource and
// GetItemVariants.
//
// The three presentation safety flags are carried so a row can be badged
// without a second call. They are the catalog's own known values; the row never
// states a risk level, a severity or an ordering derived from them.
type ItemDatabaseEntry struct {
	Kind        schema.ResourceKind `json:"kind"`
	Key         string              `json:"key"`
	GameID      uint32              `json:"gameID"`
	GameIDKnown bool                `json:"gameIDKnown"`
	Family      schema.ItemFamily   `json:"family"`
	Category    string              `json:"category"`
	Subcategory string              `json:"subcategory"`
	Name        string              `json:"name"`
	IconPath    string              `json:"iconPath"`
	BanRisk     bool                `json:"banRisk"`
	CutContent  bool                `json:"cutContent"`
	DLC         bool                `json:"dlc"`
	PreOrder    bool                `json:"preOrder"`
}

// ItemDatabaseCategory is one category the current profile can reach, with the
// number of rows it holds under the other active filters removed. It exists so
// the category filter is built from what the backend actually offers instead of
// from a hardcoded frontend list.
type ItemDatabaseCategory struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// GetItemDatabaseResult is one page of the Item Database plus the profile it
// was resolved under and the categories that profile can reach.
type GetItemDatabaseResult struct {
	SafetyProfile string                 `json:"safetyProfile"`
	Resources     []ItemDatabaseEntry    `json:"resources"`
	Categories    []ItemDatabaseCategory `json:"categories"`
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"pageSize"`
}

// GetItemDatabase returns one page of the general Item Database.
//
// Visibility is the shared safety-profile policy and nothing else: an item the
// catalog marks noDatabase never appears under any profile, and an item marked
// banRisk or cutContent appears only under the Chaos profile. dlc and preOrder
// are presentation facts and never hide a row.
//
// family, category and search are matched exactly and case-sensitively, except
// search, which is case-insensitive on the resource key and on the resource
// name. An empty filter never filters. favoritesOnly restricts the result to
// the exact identities in favorites, which is the caller's presentational
// preference: the backend owns the matching so a favourite outside the current
// page is never lost, and it stores nothing.
//
// Sorting, filtering and paging all happen over the complete match set, so the
// answer never depends on which page was asked for.
func GetItemDatabase(
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	family string,
	category string,
	search string,
	favoritesOnly bool,
	favorites []schema.ResourceRef,
	sortOrder string,
	page int,
	pageSize int,
) (GetItemDatabaseResult, error) {
	if gameCatalog == nil {
		return GetItemDatabaseResult{}, errors.New("game catalog is not loaded")
	}
	profile, err := safetyprofile.Parse(safetyProfile)
	if err != nil {
		return GetItemDatabaseResult{}, err
	}
	if err := validateGetResourcesFamily(family); err != nil {
		return GetItemDatabaseResult{}, err
	}
	switch sortOrder {
	case GetItemDatabaseSortCatalog, GetItemDatabaseSortName,
		GetItemDatabaseSortCategory, GetItemDatabaseSortGameID:
	default:
		return GetItemDatabaseResult{}, fmt.Errorf("unknown item database sort order %q", sortOrder)
	}
	if page < 0 {
		return GetItemDatabaseResult{}, fmt.Errorf("page must not be negative; got %d", page)
	}
	if pageSize < 0 {
		return GetItemDatabaseResult{}, fmt.Errorf("pageSize must not be negative; got %d", pageSize)
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = GetItemDatabaseDefaultPageSize
	}

	selected := make(map[schema.ResourceRef]struct{}, len(favorites))
	for index, favorite := range favorites {
		if favorite.Kind == "" || favorite.Key == "" {
			return GetItemDatabaseResult{}, fmt.Errorf(
				"favorites[%d] must carry a kind and a key", index)
		}
		selected[favorite] = struct{}{}
	}

	lowercaseSearch := strings.ToLower(search)
	matches := make([]ItemDatabaseEntry, 0)
	// Categories are counted over everything the profile and the family filter
	// allow, before the category filter itself is applied: a filter must not
	// erase the options a user can switch to.
	categoryCounts := make(map[string]int)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindItem {
			continue
		}
		if safetyprofile.HiddenFromItemDatabaseFlags(
			profile,
			summary.NoDatabaseKnown && summary.NoDatabase,
			summary.BanRiskKnown && summary.BanRisk,
			summary.CutContentKnown && summary.CutContent,
		) {
			continue
		}
		if family != "" && !getResourcesHasFamily(summary, schema.ItemFamily(family)) {
			continue
		}
		entry := itemDatabaseEntryOf(summary)
		if favoritesOnly {
			if _, favorite := selected[schema.ResourceRef{Kind: entry.Kind, Key: entry.Key}]; !favorite {
				continue
			}
		}
		if search != "" && !itemDatabaseMatchesSearch(entry, lowercaseSearch) {
			continue
		}
		if entry.Category != "" {
			categoryCounts[entry.Category]++
		}
		if category != "" && entry.Category != category {
			continue
		}
		matches = append(matches, entry)
	}
	sortItemDatabaseEntries(matches, sortOrder)

	result := GetItemDatabaseResult{
		SafetyProfile: string(profile),
		Resources:     []ItemDatabaseEntry{},
		Categories:    itemDatabaseCategories(categoryCounts),
		Total:         len(matches),
		Page:          page,
		PageSize:      pageSize,
	}
	// The first index is derived by division instead of multiplication so a
	// large page never overflows before it is compared with the match count.
	if result.Total == 0 || page-1 > (result.Total-1)/pageSize {
		return result, nil
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > result.Total {
		end = result.Total
	}
	result.Resources = matches[start:end]
	return result, nil
}

// itemDatabaseEntryOf projects one summary onto the list row. An unknown fact
// stays empty or false: there is deliberately no fallback to the key, to a
// category or to a placeholder, because a synthesised value would be
// indistinguishable from a real one.
func itemDatabaseEntryOf(summary gamecatalog.ResourceSummary) ItemDatabaseEntry {
	entry := ItemDatabaseEntry{Kind: summary.Kind, Key: summary.Key}
	if summary.GameIDKnown {
		entry.GameID = summary.GameID
		entry.GameIDKnown = true
	}
	if summary.FamilyKnown {
		entry.Family = summary.Family
	}
	if summary.CategoryKnown {
		entry.Category = summary.Category
	}
	if summary.SubcategoryKnown {
		entry.Subcategory = summary.Subcategory
	}
	if summary.NameKnown {
		entry.Name = summary.Name
	}
	if summary.IconPathKnown {
		entry.IconPath = summary.IconPath
	}
	entry.BanRisk = summary.BanRiskKnown && summary.BanRisk
	entry.CutContent = summary.CutContentKnown && summary.CutContent
	entry.DLC = summary.DLCKnown && summary.DLC
	entry.PreOrder = summary.PreOrderKnown && summary.PreOrder
	return entry
}

func itemDatabaseMatchesSearch(entry ItemDatabaseEntry, lowercaseSearch string) bool {
	return strings.Contains(strings.ToLower(entry.Key), lowercaseSearch) ||
		strings.Contains(strings.ToLower(entry.Name), lowercaseSearch)
}

// sortItemDatabaseEntries orders the complete match set. Every order falls back
// to the catalog order — kind and only then key — so the result is total and
// two rows that compare equal never swap between two calls.
func sortItemDatabaseEntries(entries []ItemDatabaseEntry, sortOrder string) {
	catalogOrder := func(left, right ItemDatabaseEntry) bool {
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.Key < right.Key
	}
	sort.SliceStable(entries, func(first, second int) bool {
		left, right := entries[first], entries[second]
		switch sortOrder {
		case GetItemDatabaseSortName:
			if left.Name != right.Name {
				// A row whose name the catalog does not know sorts last rather
				// than first, so an undecided fact never leads the list.
				if left.Name == "" || right.Name == "" {
					return right.Name == ""
				}
				return strings.ToLower(left.Name) < strings.ToLower(right.Name)
			}
		case GetItemDatabaseSortCategory:
			if left.Category != right.Category {
				if left.Category == "" || right.Category == "" {
					return right.Category == ""
				}
				return left.Category < right.Category
			}
		case GetItemDatabaseSortGameID:
			if left.GameIDKnown != right.GameIDKnown {
				return left.GameIDKnown
			}
			if left.GameID != right.GameID {
				return left.GameID < right.GameID
			}
		}
		return catalogOrder(left, right)
	})
}

func itemDatabaseCategories(counts map[string]int) []ItemDatabaseCategory {
	categories := make([]ItemDatabaseCategory, 0, len(counts))
	for category, count := range counts {
		categories = append(categories, ItemDatabaseCategory{Category: category, Count: count})
	}
	sort.Slice(categories, func(first, second int) bool {
		return categories[first].Category < categories[second].Category
	})
	return categories
}
