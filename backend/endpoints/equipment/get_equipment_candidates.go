/*
Endpoint: GetEquipmentCandidates
EndpointID: get_equipment_candidates
Purpose: Returns one paged, backend-filtered list of the resources one Equipment slot type currently accepts.
How it works: The runtime handler validates the slot type against the closed backend dictionary, resolves the compatibility rule of the owning setter through the same GameCatalog facts that setter reads, applies the shared safety-profile visibility policy, then searches, sorts and pages over the complete candidate set. Owned slot types are resolved from Inventory common of the addressed character, except Physick, which is resolved from Inventory common and Inventory key because its setter accepts a Crystal Tear from either section; spell memory is resolved from the catalog. Storage is never read. It opens no file and mutates nothing.
Supported resource types: ItemDocument of the family the requested slot type accepts.
Input variables: safetyProfile, saveSessionID, characterID, slotType, search, page, pageSize.
GameCatalog variables read: item.family, item.gameID, item.capabilities.equipment, item.spell.memorySlots, item.presentation.name, item.presentation.iconPath and the item.safety flags.
Save variables read: the UserData10 activity flag of the requested slot and, for an owned slot type, the Inventory records of that slot — common only, or common and key for Physick; the getter is non-mutating.
Implementation status: implemented
*/
package equipment

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/inventory"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetEquipmentCandidatesEndpointID is the stable backend identifier of GetEquipmentCandidates.
const GetEquipmentCandidatesEndpointID = "get_equipment_candidates"

// GetEquipmentCandidatesDefaultPageSize is the page size used when the caller
// passes 0. It is one picker card of 5 x 6 fields, the same window the Items
// container pages use.
const GetEquipmentCandidatesDefaultPageSize = 30

// GetEquipmentCandidatesDefinition describes the public getter contract.
var GetEquipmentCandidatesDefinition = contract.MustDefine(contract.Definition{
	Name:                   "GetEquipmentCandidates",
	ID:                     GetEquipmentCandidatesEndpointID,
	Kind:                   contract.Getter,
	SupportedResourceTypes: "ItemDocument of the family the requested slot type accepts",
	SupportedResourceVariables: []string{
		"safetyProfile", "saveSessionID", "characterID", "slotType", "search", "page", "pageSize",
	},
	Description: "Returns one paged list of the resources one Equipment slot type currently accepts.",
})

// equipmentCandidateSlots is the closed dictionary of slot types this getter
// answers for, mapped onto the item family its owning setter requires.
//
// The two ammunition slot types are deliberately absent: no confirmed writer
// addresses arrows or bolts, so offering candidates for them would advertise a
// mutation that does not exist.
var equipmentCandidateSlots = map[schema.EquipmentSlot]schema.ItemFamily{
	schema.EquipmentSlotLeftHand:    schema.ItemFamilyWeapon,
	schema.EquipmentSlotRightHand:   schema.ItemFamilyWeapon,
	schema.EquipmentSlotHead:        schema.ItemFamilyArmor,
	schema.EquipmentSlotChest:       schema.ItemFamilyArmor,
	schema.EquipmentSlotArms:        schema.ItemFamilyArmor,
	schema.EquipmentSlotLegs:        schema.ItemFamilyArmor,
	schema.EquipmentSlotTalisman:    schema.ItemFamilyTalisman,
	schema.EquipmentSlotQuickItem:   schema.ItemFamilyGoods,
	schema.EquipmentSlotPouch:       schema.ItemFamilyGoods,
	schema.EquipmentSlotPhysick:     schema.ItemFamilyGoods,
	schema.EquipmentSlotSpellMemory: schema.ItemFamilySpell,
}

// EquipmentCandidate is one resource the requested slot type accepts, with only
// what a picker cell renders and what the owning setter needs to be called.
//
// OwnedItemID is present exactly for the slot types whose setter addresses one
// owned record — the hand, armor, talisman, Quick Item and Pouch slots — and is
// the revision-scoped identity of that record. The Physick and spell setters
// take a catalog reference instead, so those candidates carry none.
//
// Quantity is the stored quantity of the owned record and is meaningless for a
// candidate without an OwnedItemID. MemorySlots is the confirmed capacity cost
// of a spell and is meaningless for every other slot type.
type EquipmentCandidate struct {
	Resource    schema.ResourceRef `json:"resource"`
	OwnedItemID string             `json:"ownedItemID,omitempty"`
	Name        string             `json:"name"`
	IconPath    string             `json:"iconPath"`
	Quantity    uint32             `json:"quantity,omitempty"`
	MemorySlots int                `json:"memorySlots,omitempty"`
	BanRisk     bool               `json:"banRisk"`
	CutContent  bool               `json:"cutContent"`
}

// GetEquipmentCandidatesResult is one page of the candidates of one slot type.
//
// Total counts every candidate that passed the filters, before paging, so a
// picker can size its navigation without walking every page.
type GetEquipmentCandidatesResult struct {
	SaveSessionID string               `json:"saveSessionID"`
	SaveRevision  string               `json:"saveRevision"`
	CharacterID   int                  `json:"characterID"`
	Active        bool                 `json:"active"`
	SafetyProfile string               `json:"safetyProfile"`
	SlotType      schema.EquipmentSlot `json:"slotType"`
	Candidates    []EquipmentCandidate `json:"candidates"`
	Total         int                  `json:"total"`
	Page          int                  `json:"page"`
	PageSize      int                  `json:"pageSize"`
}

// GetEquipmentCandidates returns one page of what the requested slot type
// currently accepts.
//
// Compatibility is decided here and nowhere above: the family and the
// capability check are the ones the owning setter performs, because both call
// the same validator, so a candidate that is offered is one the setter
// recognises. Being offered is still a necessary condition and never a promise
// — the setter validates the complete plan and may reject it, for instance when
// the same record is assigned twice.
//
// Visibility is the shared safety-profile policy: a noDatabase resource is
// never a candidate, and banRisk or cutContent resources are candidates only
// under the Chaos profile. The profile is read by the transport from the host
// setting and is never supplied by the interface.
func GetEquipmentCandidates(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	safetyProfile string,
	saveSessionID string,
	characterID int,
	slotType string,
	search string,
	page int,
	pageSize int,
) (GetEquipmentCandidatesResult, error) {
	if engine == nil {
		return GetEquipmentCandidatesResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetEquipmentCandidatesResult{}, errors.New("game catalog is not available")
	}
	profile, err := safetyprofile.Parse(safetyProfile)
	if err != nil {
		return GetEquipmentCandidatesResult{}, err
	}
	slot := schema.EquipmentSlot(slotType)
	family, supported := equipmentCandidateSlots[slot]
	if !supported {
		return GetEquipmentCandidatesResult{}, fmt.Errorf(
			"slotType %q has no equipment candidate contract", slotType)
	}
	if page < 0 {
		return GetEquipmentCandidatesResult{}, fmt.Errorf("page must not be negative; got %d", page)
	}
	if pageSize < 0 {
		return GetEquipmentCandidatesResult{}, fmt.Errorf(
			"pageSize must not be negative; got %d", pageSize)
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = GetEquipmentCandidatesDefaultPageSize
	}

	result := GetEquipmentCandidatesResult{
		SafetyProfile: string(profile),
		SlotType:      slot,
		Candidates:    []EquipmentCandidate{},
		Page:          page,
		PageSize:      pageSize,
	}

	var matches []EquipmentCandidate
	if slot == schema.EquipmentSlotSpellMemory {
		// A spell is equipped by catalog reference and needs no owned record, so
		// only the session identity of the addressed slot is read here.
		stored, err := GetEquipment(engine, saveSessionID, characterID)
		if err != nil {
			return GetEquipmentCandidatesResult{}, err
		}
		result.SaveSessionID = stored.SaveSessionID
		result.SaveRevision = stored.SaveRevision
		result.CharacterID = stored.CharacterID
		result.Active = stored.Active
		if stored.Active {
			matches = spellCandidates(gameCatalog, profile)
		}
	} else {
		// Physick is the one owned slot type whose setter also accepts a Crystal
		// Tear stored in Inventory key, and Crystal Tears are key items, so it is
		// read through the both-sections contract of GetInventory. Every other
		// owned slot type is restricted to Inventory common by its own setter.
		// Storage takes part in neither: no equipment setter addresses it.
		//
		// Filtering, sorting and paging are container-wide by contract, so the
		// whole container is asked for in one page; the physical capacity behind
		// that number stays SaveEngine's.
		section := saveengine.InventorySectionCommon
		if slot == schema.EquipmentSlotPhysick {
			section = ""
		}
		stored, err := inventory.GetInventory(
			engine, gameCatalog, saveSessionID, characterID,
			section, 1, saveengine.InventoryHeldMaxRecords)
		if err != nil {
			return GetEquipmentCandidatesResult{}, err
		}
		result.SaveSessionID = stored.SaveSessionID
		result.SaveRevision = stored.SaveRevision
		result.CharacterID = stored.CharacterID
		result.Active = stored.Active
		matches, err = ownedCandidates(gameCatalog, profile, slot, family, stored.Records)
		if err != nil {
			return GetEquipmentCandidatesResult{}, err
		}
	}

	if search != "" {
		lowercase := strings.ToLower(search)
		filtered := matches[:0]
		for _, candidate := range matches {
			if strings.Contains(strings.ToLower(candidate.Name), lowercase) ||
				strings.Contains(strings.ToLower(candidate.Resource.Key), lowercase) {
				filtered = append(filtered, candidate)
			}
		}
		matches = filtered
	}
	sortEquipmentCandidates(matches)

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
	result.Candidates = matches[start:end]
	return result, nil
}

// ownedCandidates turns the Inventory records of one character into the
// candidates the requested slot type accepts. The Physick slot is the one
// exception to the one-record-one-candidate rule: its setter takes a catalog
// reference, so two owned copies of the same Crystal Tear — in either section —
// are one candidate, deduplicated by canonical ResourceRef.
func ownedCandidates(
	gameCatalog *gamecatalog.Catalog,
	profile safetyprofile.Profile,
	slot schema.EquipmentSlot,
	family schema.ItemFamily,
	records []inventory.InventoryRecord,
) ([]EquipmentCandidate, error) {
	candidates := make([]EquipmentCandidate, 0, len(records))
	seen := make(map[schema.ResourceRef]struct{}, len(records))
	// The wording of the shared rule is irrelevant here — only whether it
	// accepts — so it is built once instead of once per record.
	slotPhrase := fmt.Sprintf("slot %q", slot)
	for _, record := range records {
		if record.Quantity == 0 {
			continue
		}
		resource, found := gameCatalog.ItemByGameID(record.GameID)
		if !found || resource.Item == nil {
			// The reader above already rejects an unknown game ID, so reaching
			// this point would mean the two disagree.
			return nil, fmt.Errorf(
				"owned item %q: game ID 0x%08X is not a known item",
				record.OwnedItemID, record.GameID)
		}
		item := resource.Item
		if hiddenFromEquipmentPicker(profile, item) {
			continue
		}
		if slot == schema.EquipmentSlotPhysick {
			// The Physick rule is exactly the one SetPhysickMixture enforces,
			// asked for through that endpoint's own validator.
			if _, err := physickTearGameID(resource); err != nil {
				continue
			}
		} else if validateEquipmentSlotCompatibility(
			resource, family, slot, string(slot), slotPhrase) != nil {
			continue
		}

		candidate := EquipmentCandidate{
			Resource:   resource.Ref(),
			BanRisk:    item.Safety.BanRisk.Known && item.Safety.BanRisk.Value,
			CutContent: item.Safety.CutContent.Known && item.Safety.CutContent.Value,
		}
		candidate.Name, candidate.IconPath = itemPresentation(item)
		if slot == schema.EquipmentSlotPhysick {
			if _, duplicate := seen[candidate.Resource]; duplicate {
				continue
			}
			seen[candidate.Resource] = struct{}{}
		} else {
			candidate.OwnedItemID = record.OwnedItemID
			candidate.Quantity = record.Quantity
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

// spellCandidates lists every catalog spell the spell setter accepts. Ownership
// is not part of that contract: SetEquippedSpells validates the resource, the
// confirmed capacity cost and the total capacity, and never an owned record.
func spellCandidates(
	gameCatalog *gamecatalog.Catalog,
	profile safetyprofile.Profile,
) []EquipmentCandidate {
	candidates := make([]EquipmentCandidate, 0)
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindItem {
			continue
		}
		if !summary.FamilyKnown || summary.Family != schema.ItemFamilySpell {
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
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			continue
		}
		// The one shared spell validator: a resource it rejects has no confirmed
		// raw MagicParam ID or no confirmed positive capacity cost, and can never
		// be offered as a candidate.
		_, memoryCost, err := gamecatalog.ValidateSpellResource(resource)
		if err != nil {
			continue
		}
		candidate := EquipmentCandidate{
			Resource:    resource.Ref(),
			MemorySlots: memoryCost,
			BanRisk:     summary.BanRiskKnown && summary.BanRisk,
			CutContent:  summary.CutContentKnown && summary.CutContent,
		}
		candidate.Name, candidate.IconPath = itemPresentation(resource.Item)
		candidates = append(candidates, candidate)
	}
	return candidates
}

// hiddenFromEquipmentPicker applies the shared visibility policy to one owned
// record's document, so a picker hides exactly what the Item Database hides.
func hiddenFromEquipmentPicker(profile safetyprofile.Profile, item *schema.ItemDocument) bool {
	return safetyprofile.HiddenFromItemDatabase(profile, item)
}

// sortEquipmentCandidates orders the complete match set by name and falls back
// to the canonical key, so the order is total and two candidates that compare
// equal never swap between two calls. A resource whose name the catalog does
// not know sorts last rather than first.
func sortEquipmentCandidates(candidates []EquipmentCandidate) {
	sort.SliceStable(candidates, func(first, second int) bool {
		left, right := candidates[first], candidates[second]
		if left.Name != right.Name {
			if left.Name == "" || right.Name == "" {
				return right.Name == ""
			}
			return strings.ToLower(left.Name) < strings.ToLower(right.Name)
		}
		if left.Resource.Key != right.Resource.Key {
			return left.Resource.Key < right.Resource.Key
		}
		return left.OwnedItemID < right.OwnedItemID
	})
}

// validateEquipmentSlotCompatibility is the single family-and-capability rule of
// this domain. The hand, armor, talisman, Quick Item and Pouch setters validate
// an assignment with it and GetEquipmentCandidates offers by it, so a candidate
// the picker shows is one the setter recognises and neither side can drift.
//
// Only the wording is per caller: capability names the group in the missing
// capability failure and slotPhrase names the position in the compatibility
// failure, so every setter keeps the exact message its contract documents.
//
// The Physick and spell rules are deliberately not expressed here. Physick adds
// confirmed game-ID conditions of its own and stays in physickTearGameID; a
// spell is not addressed by an equipment slot at all and stays in
// gamecatalog.ValidateSpellResource.
func validateEquipmentSlotCompatibility(
	resource schema.Resource,
	family schema.ItemFamily,
	slot schema.EquipmentSlot,
	capability string,
	slotPhrase string,
) error {
	item := resource.Item
	if item == nil {
		return fmt.Errorf("resource kind %q key %q has no item document",
			resource.Kind, resource.Key)
	}
	if !item.Family.Known || item.Family.Value != family {
		return fmt.Errorf("resource kind %q key %q has item family %q, want %q",
			resource.Kind, resource.Key, item.Family.Value, family)
	}
	equipment := item.Capabilities.Equipment
	if !equipment.Known || !equipment.Enabled || equipment.Rules == nil {
		return fmt.Errorf("resource kind %q key %q has no confirmed %s equipment capability",
			resource.Kind, resource.Key, capability)
	}
	for _, allowed := range equipment.Rules.AllowedSlots {
		if allowed == slot {
			return nil
		}
	}
	return fmt.Errorf("resource kind %q key %q cannot be equipped in %s",
		resource.Kind, resource.Key, slotPhrase)
}
