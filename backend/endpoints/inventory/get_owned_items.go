/*
Endpoint: GetOwnedItems
EndpointID: get_owned_items
Purpose: Returns one authoritative page of the Inventory or the Storage Box of one character, resolved against GameCatalog and the active Safety Profile, with the mutations each record actually allows.
How it works: The runtime handler reads the complete addressed container through the existing GetInventory or GetStorage endpoint, resolves every record's ItemDocument through the already loaded GameCatalog, applies the shared safety-profile limits, then filters, sorts and pages over the complete container rather than over one served page. It opens no file, parses no save data of its own and writes nothing.
Supported resource types: ItemDocument.
Input variables: safetyProfile, saveSessionID, characterID, container, containerSection, search, category, favoritesOnly, favorites, sort, page, pageSize.
GameCatalog variables read: item.gameID, item.family, item.category, item.subcategory, item.presentation.name, item.presentation.iconPath, item.storage.recordMode, item.storage.maxInventory, item.storage.safeModeMaxInventory, item.storage.maxStorage, item.storage.safeModeMaxStorage, item.capabilities.stack, item.goods.isDepositable and the six item.safety flags.
Save variables read: the UserData10 activity flag of the requested slot and, for an active slot, the physical InventoryHeld or Storage Box records and the GaItem table needed to resolve them; the getter is non-mutating.
Implementation status: implemented
*/
package inventory

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetOwnedItemsEndpointID is the stable backend identifier of GetOwnedItems.
const GetOwnedItemsEndpointID = "get_owned_items"

// The two containers of one character slot. They are the exact wire values and
// are never trimmed, recased or aliased.
const (
	OwnedItemsContainerInventory = "inventory"
	OwnedItemsContainerStorage   = "storage"
)

// The closed vocabulary of authoritative sort orders. The empty value is the
// container's own order, which is the physical order the getter below it
// reports; every other value orders the complete container and falls back to
// that same order for rows that compare equal.
const (
	OwnedItemsSortContainer = ""
	OwnedItemsSortName      = "name"
	OwnedItemsSortCategory  = "category"
	OwnedItemsSortQuantity  = "quantity"
)

// GetOwnedItemsDefaultPageSize is the page size used when the caller passes 0.
// It is one Grid card of 5 x 6 physical fields.
const GetOwnedItemsDefaultPageSize = 30

// ownedItemsContainerCapacity is the number of physical common rows the reader
// below has to be asked for so the whole container is returned in one page.
// Filtering and sorting are container-wide by contract, so a partial read would
// silently answer a different question.
const ownedItemsContainerCapacity = 0xA80

// GetOwnedItemsDefinition describes the public getter contract.
var GetOwnedItemsDefinition = contract.MustDefine(contract.Definition{
	Name:                   "GetOwnedItems",
	ID:                     GetOwnedItemsEndpointID,
	Kind:                   contract.Getter,
	SupportedResourceTypes: "ItemDocument",
	SupportedResourceVariables: []string{
		"safetyProfile", "saveSessionID", "characterID", "container", "containerSection",
		"search", "category", "favoritesOnly", "favorites", "sort", "page", "pageSize",
	},
	Description: "Returns one authoritative page of one container of a character, resolved against GameCatalog and the active Safety Profile.",
})

// OwnedItemActions states which mutations the backend accepts for one record.
// It exists so the interface renders exactly the actions the backend supports
// instead of deriving them from a family, a category or a guess. Every flag is
// a necessary condition the backend checked, never a promise: the mutation
// itself still validates the complete plan and may reject it.
type OwnedItemActions struct {
	MoveToStorage   bool `json:"moveToStorage"`
	MoveToInventory bool `json:"moveToInventory"`
	Remove          bool `json:"remove"`
	SetQuantity     bool `json:"setQuantity"`
	Reorder         bool `json:"reorder"`
}

// OwnedItemRow is one container record with everything one list cell needs. The
// physical fields are the ones the raw getters report and are carried verbatim;
// the catalog fields are resolved here and are empty when the catalog does not
// know them.
//
// MaxQuantity is the limit of the record's own container under the active
// profile. MaxQuantityKnown false means the catalog states no usable limit, and
// the interface then shows no maximum rather than a substituted one.
type OwnedItemRow struct {
	OwnedItemID      string              `json:"ownedItemID"`
	Kind             schema.ResourceKind `json:"kind"`
	Key              string              `json:"key"`
	GameID           uint32              `json:"gameID"`
	Container        string              `json:"container"`
	ContainerSection string              `json:"containerSection"`
	PhysicalIndex    int                 `json:"physicalIndex"`
	AcquisitionIndex uint32              `json:"acquisitionIndex"`
	// OrderPosition is the zero-based rank of this record inside the manual
	// Inventory order, counted over the whole container before any filter. It is
	// meaningless unless OrderPositionKnown, which is true exactly for the
	// records ReorderInventoryItems accepts, so an interface never has to derive
	// a position from a page, an acquisition index or a physical row.
	OrderPosition      int               `json:"orderPosition"`
	OrderPositionKnown bool              `json:"orderPositionKnown"`
	Quantity           uint32            `json:"quantity"`
	MaxQuantity        uint32            `json:"maxQuantity"`
	MaxQuantityKnown   bool              `json:"maxQuantityKnown"`
	Family             schema.ItemFamily `json:"family"`
	Category           string            `json:"category"`
	Subcategory        string            `json:"subcategory"`
	Name               string            `json:"name"`
	IconPath           string            `json:"iconPath"`
	RecordMode         string            `json:"recordMode"`
	BanRisk            bool              `json:"banRisk"`
	CutContent         bool              `json:"cutContent"`
	DLC                bool              `json:"dlc"`
	PreOrder           bool              `json:"preOrder"`
	Actions            OwnedItemActions  `json:"actions"`
}

// GetOwnedItemsResult is one resolved page of one container.
//
// Total counts every record that passed the filters, before paging, so a caller
// can size its card navigation without walking every page.
type GetOwnedItemsResult struct {
	SaveSessionID string              `json:"saveSessionID"`
	SaveRevision  string              `json:"saveRevision"`
	CharacterID   int                 `json:"characterID"`
	Active        bool                `json:"active"`
	SafetyProfile string              `json:"safetyProfile"`
	Container     string              `json:"container"`
	Records       []OwnedItemRow      `json:"records"`
	Categories    []OwnedItemCategory `json:"categories"`
	Total         int                 `json:"total"`
	Page          int                 `json:"page"`
	PageSize      int                 `json:"pageSize"`
}

// OwnedItemCategory is the category facet of one container: a category the
// container actually holds, with the number of rows behind it once the other
// active filters are applied. It exists so the category control is built from
// what the backend reports instead of from a hardcoded frontend list.
type OwnedItemCategory struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// GetOwnedItems returns one page of one container of one character.
//
// The complete container is read first and only then filtered, sorted and
// sliced, so search, category, favourites and sort order always describe the
// whole container instead of one served page. The filters are matched exactly
// and case-sensitively, except search, which is case-insensitive on the
// resource key and on the resource name; an empty filter never filters.
//
// The limits come from the shared safety-profile policy, which is the single
// place that decides whether a Safe Mode limit or the base limit applies.
// Nothing is defaulted, widened or clamped here.
func GetOwnedItems(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	container string,
	containerSection string,
	search string,
	category string,
	favoritesOnly bool,
	favorites []schema.ResourceRef,
	sortOrder string,
	page int,
	pageSize int,
) (GetOwnedItemsResult, error) {
	if engine == nil {
		return GetOwnedItemsResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetOwnedItemsResult{}, errors.New("game catalog is not available")
	}
	profile, err := safetyprofile.Parse(safetyProfile)
	if err != nil {
		return GetOwnedItemsResult{}, err
	}
	switch container {
	case OwnedItemsContainerInventory, OwnedItemsContainerStorage:
	default:
		return GetOwnedItemsResult{}, fmt.Errorf(
			"container must be %q or %q; got %q",
			OwnedItemsContainerInventory, OwnedItemsContainerStorage, container)
	}
	switch sortOrder {
	case OwnedItemsSortContainer, OwnedItemsSortName,
		OwnedItemsSortCategory, OwnedItemsSortQuantity:
	default:
		return GetOwnedItemsResult{}, fmt.Errorf("unknown owned item sort order %q", sortOrder)
	}
	if page < 0 {
		return GetOwnedItemsResult{}, fmt.Errorf("page must not be negative; got %d", page)
	}
	if pageSize < 0 {
		return GetOwnedItemsResult{}, fmt.Errorf("pageSize must not be negative; got %d", pageSize)
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = GetOwnedItemsDefaultPageSize
	}
	selected := make(map[schema.ResourceRef]struct{}, len(favorites))
	for index, favorite := range favorites {
		if favorite.Kind == "" || favorite.Key == "" {
			return GetOwnedItemsResult{}, fmt.Errorf(
				"favorites[%d] must carry a kind and a key", index)
		}
		selected[favorite] = struct{}{}
	}

	result := GetOwnedItemsResult{
		SafetyProfile: string(profile),
		Container:     container,
		Records:       []OwnedItemRow{},
		Categories:    []OwnedItemCategory{},
		Page:          page,
		PageSize:      pageSize,
	}

	// The whole container is read through the existing raw getters, so record
	// decoding, activity, sections, identities and the quantity mask keep their
	// single owner and are not restated here.
	rows := make([]OwnedItemRow, 0)
	if container == OwnedItemsContainerInventory {
		stored, err := GetInventory(
			engine, gameCatalog, saveSessionID, characterID, containerSection,
			1, ownedItemsContainerCapacity)
		if err != nil {
			return GetOwnedItemsResult{}, err
		}
		result.SaveSessionID = stored.SaveSessionID
		result.SaveRevision = stored.SaveRevision
		result.CharacterID = stored.CharacterID
		result.Active = stored.Active
		for _, record := range stored.Records {
			row, err := ownedItemRow(gameCatalog, profile, container, record.OwnedItemID,
				record.Kind, record.Key, record.GameID, record.ContainerSection,
				record.PhysicalIndex, record.AcquisitionIndex, record.Quantity)
			if err != nil {
				return GetOwnedItemsResult{}, err
			}
			rows = append(rows, row)
		}
	} else {
		stored, err := GetStorage(
			engine, gameCatalog, saveSessionID, characterID, containerSection,
			1, ownedItemsContainerCapacity)
		if err != nil {
			return GetOwnedItemsResult{}, err
		}
		result.SaveSessionID = stored.SaveSessionID
		result.SaveRevision = stored.SaveRevision
		result.CharacterID = stored.CharacterID
		result.Active = stored.Active
		for _, record := range stored.Records {
			row, err := ownedItemRow(gameCatalog, profile, container, record.OwnedItemID,
				record.Kind, record.Key, record.GameID, record.ContainerSection,
				record.PhysicalIndex, record.AcquisitionIndex, record.Quantity)
			if err != nil {
				return GetOwnedItemsResult{}, err
			}
			rows = append(rows, row)
		}
	}

	assignInventoryOrderPositions(rows)

	lowercaseSearch := strings.ToLower(search)
	matches := make([]OwnedItemRow, 0, len(rows))
	// Facets are counted before the category filter is applied, so a chosen
	// category never erases the categories a user can switch to.
	categoryCounts := make(map[string]int)
	for _, row := range rows {
		if favoritesOnly {
			if _, favorite := selected[schema.ResourceRef{Kind: row.Kind, Key: row.Key}]; !favorite {
				continue
			}
		}
		if search != "" && !ownedItemMatchesSearch(row, lowercaseSearch) {
			continue
		}
		if row.Category != "" {
			categoryCounts[row.Category]++
		}
		if category != "" && row.Category != category {
			continue
		}
		matches = append(matches, row)
	}
	sortOwnedItemRows(matches, sortOrder)

	result.Categories = ownedItemCategories(categoryCounts)
	result.Total = len(matches)
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
	result.Records = matches[start:end]
	return result, nil
}

// ownedItemRow resolves one physical record into the row a list cell renders.
// A record whose catalog document cannot be resolved is a hard error: the raw
// getters already reject an unknown game ID, so reaching this point without a
// document would mean the two disagree.
func ownedItemRow(
	gameCatalog *gamecatalog.Catalog,
	profile safetyprofile.Profile,
	container string,
	ownedItemID string,
	kind schema.ResourceKind,
	key string,
	gameID uint32,
	containerSection string,
	physicalIndex int,
	acquisitionIndex uint32,
	quantity uint32,
) (OwnedItemRow, error) {
	resource, exists := gameCatalog.ItemByGameID(gameID)
	if !exists || resource.Item == nil {
		return OwnedItemRow{}, fmt.Errorf(
			"owned item %q: game ID 0x%08X is not a known item", ownedItemID, gameID)
	}
	item := resource.Item
	row := OwnedItemRow{
		OwnedItemID:      ownedItemID,
		Kind:             kind,
		Key:              key,
		GameID:           gameID,
		Container:        container,
		ContainerSection: containerSection,
		PhysicalIndex:    physicalIndex,
		AcquisitionIndex: acquisitionIndex,
		Quantity:         quantity,
	}
	if item.Family.Known {
		row.Family = item.Family.Value
	}
	if item.Category.Known {
		row.Category = item.Category.Value
	}
	if item.Subcategory.Known {
		row.Subcategory = item.Subcategory.Value
	}
	if item.Presentation.Name.Known {
		row.Name = item.Presentation.Name.Value
	}
	if item.Presentation.IconPath.Known {
		row.IconPath = item.Presentation.IconPath.Value
	}
	if item.Storage.RecordMode.Known {
		row.RecordMode = string(item.Storage.RecordMode.Value)
	}
	row.BanRisk = item.Safety.BanRisk.Known && item.Safety.BanRisk.Value
	row.CutContent = item.Safety.CutContent.Known && item.Safety.CutContent.Value
	row.DLC = item.Safety.DLC.Known && item.Safety.DLC.Value
	row.PreOrder = item.Safety.PreOrder.Known && item.Safety.PreOrder.Value

	if container == OwnedItemsContainerInventory {
		row.MaxQuantity, row.MaxQuantityKnown = safetyprofile.InventoryLimit(profile, item)
	} else {
		row.MaxQuantity, row.MaxQuantityKnown = safetyprofile.StorageLimit(profile, item)
	}
	row.Actions = ownedItemActions(gameCatalog, profile, item, gameID, container, containerSection)
	return row, nil
}

// ownedItemActions states the necessary conditions each mutation of one record
// has, using the same catalog facts the mutations themselves read.
//
// ponytail: these are the conditions that can be decided from the catalog and
// the record's own address. Whether an Equipment, Quick Item or Pouch slot
// references the record is decided by the shared removal planner inside the
// mutation, which fails closed; duplicating that scan in a list getter would be
// a second implementation of the same rule.
func ownedItemActions(
	gameCatalog *gamecatalog.Catalog,
	profile safetyprofile.Profile,
	item *schema.ItemDocument,
	gameID uint32,
	container string,
	containerSection string,
) OwnedItemActions {
	inventoryCommon := container == OwnedItemsContainerInventory &&
		containerSection == saveengine.InventorySectionCommon
	storageCommon := container == OwnedItemsContainerStorage &&
		containerSection == saveengine.StorageSectionCommon

	actions := OwnedItemActions{}
	// The Storage key section has no confirmed write contract, so nothing is
	// offered for it.
	actions.Remove = inventoryCommon ||
		(container == OwnedItemsContainerInventory &&
			containerSection == saveengine.InventorySectionKey)

	_, inventoryLimitKnown := safetyprofile.InventoryLimit(profile, item)
	_, storageLimitKnown := safetyprofile.StorageLimit(profile, item)
	depositable := item.Family.Known && item.Family.Value != schema.ItemFamilyGoods
	if item.Goods != nil && item.Goods.IsDepositable.Known {
		depositable = item.Goods.IsDepositable.Value
	}
	actions.MoveToStorage = inventoryCommon && storageLimitKnown && depositable
	actions.MoveToInventory = storageCommon && inventoryLimitKnown

	stack := item.Capabilities.Stack
	actions.SetQuantity = item.Storage.RecordMode.Known &&
		item.Storage.RecordMode.Value == schema.RecordModeQuantityStack &&
		stack.Known && stack.Enabled && stack.Rules != nil && stack.Rules.MaxPerStack > 0

	if inventoryCommon {
		// The Inventory order contract owns which categories can be reordered;
		// this getter asks it instead of restating the list.
		supported, err := supportsItemOrder(gameCatalog, gameID)
		actions.Reorder = err == nil && supported
	}
	return actions
}

// assignInventoryOrderPositions ranks the records the manual Inventory order
// accepts. The rank is taken over the complete container, before any filter or
// paging, because the position a reorder addresses is a property of the
// container and not of the page a user happens to be looking at.
//
// The order is the ascending acquisition index, which is the same logical order
// SetInventoryOrder and ReorderInventoryItems write.
func assignInventoryOrderPositions(rows []OwnedItemRow) {
	reorderable := make([]int, 0, len(rows))
	for index := range rows {
		if rows[index].Actions.Reorder {
			reorderable = append(reorderable, index)
		}
	}
	sort.SliceStable(reorderable, func(first, second int) bool {
		return rows[reorderable[first]].AcquisitionIndex < rows[reorderable[second]].AcquisitionIndex
	})
	for position, index := range reorderable {
		rows[index].OrderPosition = position
		rows[index].OrderPositionKnown = true
	}
}

func ownedItemMatchesSearch(row OwnedItemRow, lowercaseSearch string) bool {
	return strings.Contains(strings.ToLower(row.Key), lowercaseSearch) ||
		strings.Contains(strings.ToLower(row.Name), lowercaseSearch)
}

// sortOwnedItemRows orders the complete match set. Every order falls back to the
// container's own physical order, so the result is total and two rows that
// compare equal never swap between two calls.
func sortOwnedItemRows(rows []OwnedItemRow, sortOrder string) {
	sort.SliceStable(rows, func(first, second int) bool {
		left, right := rows[first], rows[second]
		switch sortOrder {
		case OwnedItemsSortName:
			if left.Name != right.Name {
				// A row whose name the catalog does not know sorts last rather
				// than first, so an undecided fact never leads the list.
				if left.Name == "" || right.Name == "" {
					return right.Name == ""
				}
				return strings.ToLower(left.Name) < strings.ToLower(right.Name)
			}
		case OwnedItemsSortCategory:
			if left.Category != right.Category {
				if left.Category == "" || right.Category == "" {
					return right.Category == ""
				}
				return left.Category < right.Category
			}
		case OwnedItemsSortQuantity:
			if left.Quantity != right.Quantity {
				return left.Quantity > right.Quantity
			}
		}
		if left.ContainerSection != right.ContainerSection {
			return left.ContainerSection < right.ContainerSection
		}
		return left.PhysicalIndex < right.PhysicalIndex
	})
}

func ownedItemCategories(counts map[string]int) []OwnedItemCategory {
	categories := make([]OwnedItemCategory, 0, len(counts))
	for category, count := range counts {
		categories = append(categories, OwnedItemCategory{Category: category, Count: count})
	}
	sort.Slice(categories, func(first, second int) bool {
		return categories[first].Category < categories[second].Category
	})
	return categories
}
