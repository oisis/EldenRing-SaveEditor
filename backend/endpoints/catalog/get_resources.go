/*
Endpoint: GetResources
EndpointID: get_resources
Purpose: Returns a paginated resource list filtered by type, family, capability, endpoint, and other criteria, for building pickers without separate getters for every category.
How it works: The runtime handler reads the already loaded GameCatalog through Catalog.ResourceSummaries, applies the declared filters in catalog order (kind, then key), counts the matches and returns one page of a light projection without loading, reloading or modifying the catalog.
Supported resource types: GameResource.
Input variables: resourceType, family, capability, endpointId, search, page, pageSize.
GameCatalog variables read: Resource.Kind, Resource.Key, Item.Family, Item.Presentation.Name, Item.Capabilities (Known and Enabled only), Colosseum.Name, Region.Name, SummoningPool.Name, Grace.Name, Boss.Name, MapRegion.Name and Tutorial.Title. The full resource document is never projected; it stays the responsibility of GetResource.
Save variables read: none; the endpoint never opens or reads a save.
Implementation status: implemented; GetResources is the runtime handler of this contract.
*/
package catalog

import (
	"errors"
	"fmt"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// GetResourcesEndpointID is the stable backend identifier of GetResources.
const GetResourcesEndpointID = "get_resources"

// GetResourcesDefinition describes the public getter contract.
var GetResourcesDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetResources",
	ID:                         GetResourcesEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource",
	SupportedResourceVariables: []string{"resourceType", "family", "capability", "endpointId", "search", "page", "pageSize"},
	Description:                "Returns a paginated resource list filtered by type, family, capability, endpoint, and other criteria, for building pickers without separate getters for every category.",
})

// GetResourcesDefaultPageSize is the page size used when the caller passes 0.
// There is deliberately no maximum page size: the caller decides how much of the
// catalog it wants and the projection is small enough to carry in one response.
const GetResourcesDefaultPageSize = 50

// Capability filter names accepted by GetResources. They mirror the fields of
// schema.ItemCapabilities one by one; there is no alias and no normalisation.
const (
	GetResourcesCapabilityUpgrade       = "upgrade"
	GetResourcesCapabilityInfusion      = "infusion"
	GetResourcesCapabilityAshOfWarMount = "ashOfWarMount"
	GetResourcesCapabilityStack         = "stack"
	GetResourcesCapabilityEquipment     = "equipment"
)

// GetResourcesEntry is one row of the GetResources result. It is a picker-sized
// projection, not a resource document: relations, variants, provenance and
// capabilities are deliberately absent and stay the responsibility of
// GetResource, GetResourceRelations and GetItemVariants.
type GetResourcesEntry struct {
	Kind   schema.ResourceKind `json:"kind"`
	Key    string              `json:"key"`
	Family schema.ItemFamily   `json:"family"`
	Name   string              `json:"name"`
}

// GetResourcesResult is the typed result of GetResources. Total counts every
// resource that passed the filters, before paging, so a caller can size a picker
// without walking every page.
type GetResourcesResult struct {
	Resources []GetResourcesEntry `json:"resources"`
	Total     int                 `json:"total"`
	Page      int                 `json:"page"`
	PageSize  int                 `json:"pageSize"`
}

// GetResources returns one page of catalog resources reduced to the fields a
// list or a picker needs. Every filter is matched exactly and case-sensitively
// except search, which is case-insensitive on Resource.Key and on the resource
// name. The accepted resource types are item, colosseum, region, summoning_pool,
// grace, boss, map_region and tutorial; family and capability describe items
// only, so a non-empty one of them never matches a non-item resource, whose
// family stays empty. An empty filter never filters. The order is the catalog
// order, kind first and only then key, so paging is stable across calls. Values
// are read from a value-only catalog snapshot, so the result can never reach the
// catalog.
func GetResources(
	gameCatalog *gamecatalog.Catalog,
	resourceType string,
	family string,
	capability string,
	endpointID string,
	search string,
	page int,
	pageSize int,
) (GetResourcesResult, error) {
	if gameCatalog == nil {
		return GetResourcesResult{}, errors.New("game catalog is not loaded")
	}
	switch schema.ResourceKind(resourceType) {
	case "", schema.ResourceKindItem, schema.ResourceKindColosseum, schema.ResourceKindRegion,
		schema.ResourceKindSummoningPool, schema.ResourceKindGrace, schema.ResourceKindBoss,
		schema.ResourceKindMapRegion, schema.ResourceKindTutorial:
	default:
		return GetResourcesResult{}, fmt.Errorf("unsupported resource type %q", resourceType)
	}
	if err := validateGetResourcesFamily(family); err != nil {
		return GetResourcesResult{}, err
	}
	if err := validateGetResourcesCapability(capability); err != nil {
		return GetResourcesResult{}, err
	}
	// GameCatalog stores no resource-to-endpoint relation, so the filter cannot
	// be answered from data. Guessing a mapping here would invent a contract the
	// catalog never declared.
	if endpointID != "" {
		return GetResourcesResult{}, fmt.Errorf(
			"the endpointId filter is not supported because GameCatalog does not declare endpoint relations yet; got %q",
			endpointID,
		)
	}
	if page < 0 {
		return GetResourcesResult{}, fmt.Errorf("page must not be negative; got %d", page)
	}
	if pageSize < 0 {
		return GetResourcesResult{}, fmt.Errorf("pageSize must not be negative; got %d", pageSize)
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = GetResourcesDefaultPageSize
	}

	lowercaseSearch := strings.ToLower(search)
	matches := make([]GetResourcesEntry, 0)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if resourceType != "" && summary.Kind != schema.ResourceKind(resourceType) {
			continue
		}
		if family != "" && !getResourcesHasFamily(summary, schema.ItemFamily(family)) {
			continue
		}
		if capability != "" && !getResourcesHasEnabledCapability(summary, capability) {
			continue
		}
		entry := getResourcesEntryOf(summary)
		if search != "" && !getResourcesMatchesSearch(entry, lowercaseSearch) {
			continue
		}
		matches = append(matches, entry)
	}

	total := len(matches)
	// The first index is derived by division instead of multiplication so a large
	// page never overflows before it is compared with the match count.
	if total == 0 || page-1 > (total-1)/pageSize {
		return GetResourcesResult{
			Resources: []GetResourcesEntry{},
			Total:     total,
			Page:      page,
			PageSize:  pageSize,
		}, nil
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return GetResourcesResult{
		Resources: matches[start:end],
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

// getResourcesEntryOf projects one summary onto the picker row. An unknown name
// stays the empty string: there is deliberately no fallback to the key, to a
// category or to a placeholder, because a synthesised name would be
// indistinguishable from a real one.
func getResourcesEntryOf(summary gamecatalog.ResourceSummary) GetResourcesEntry {
	entry := GetResourcesEntry{Kind: summary.Kind, Key: summary.Key}
	if summary.FamilyKnown {
		entry.Family = summary.Family
	}
	if summary.NameKnown {
		entry.Name = summary.Name
	}
	return entry
}

func getResourcesMatchesSearch(entry GetResourcesEntry, lowercaseSearch string) bool {
	return strings.Contains(strings.ToLower(entry.Key), lowercaseSearch) ||
		strings.Contains(strings.ToLower(entry.Name), lowercaseSearch)
}

// getResourcesHasFamily requires a known family, so a resource whose family was
// never established is never reported as a member of the requested family.
func getResourcesHasFamily(summary gamecatalog.ResourceSummary, family schema.ItemFamily) bool {
	return summary.FamilyKnown && summary.Family == family
}

// getResourcesHasEnabledCapability matches only a capability the catalog both
// knows and enables. An unknown capability is not an enabled one, so undecided
// data never widens a picker. The catalog only reports the two flags; deciding
// what they mean stays here.
func getResourcesHasEnabledCapability(summary gamecatalog.ResourceSummary, capability string) bool {
	switch capability {
	case GetResourcesCapabilityUpgrade:
		return summary.Upgrade.Known && summary.Upgrade.Enabled
	case GetResourcesCapabilityInfusion:
		return summary.Infusion.Known && summary.Infusion.Enabled
	case GetResourcesCapabilityAshOfWarMount:
		return summary.AshOfWarMount.Known && summary.AshOfWarMount.Enabled
	case GetResourcesCapabilityStack:
		return summary.Stack.Known && summary.Stack.Enabled
	case GetResourcesCapabilityEquipment:
		return summary.Equipment.Known && summary.Equipment.Enabled
	}
	return false
}

func validateGetResourcesFamily(family string) error {
	if family == "" {
		return nil
	}
	switch schema.ItemFamily(family) {
	case schema.ItemFamilyWeapon,
		schema.ItemFamilyArmor,
		schema.ItemFamilyTalisman,
		schema.ItemFamilyAshOfWar,
		schema.ItemFamilySpell,
		schema.ItemFamilySpiritAsh,
		schema.ItemFamilyGoods,
		schema.ItemFamilyGesture:
		return nil
	}
	return fmt.Errorf("unknown item family %q", family)
}

func validateGetResourcesCapability(capability string) error {
	switch capability {
	case "",
		GetResourcesCapabilityUpgrade,
		GetResourcesCapabilityInfusion,
		GetResourcesCapabilityAshOfWarMount,
		GetResourcesCapabilityStack,
		GetResourcesCapabilityEquipment:
		return nil
	}
	return fmt.Errorf("unknown capability %q", capability)
}
