package application

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/db"
)

// SaveInventoryIntegrityReport describes the outcome of a read-only scan of
// every populated character slot. Clean == true means no slot holds a conflict
// and the editor is safe to mutate.
type SaveInventoryIntegrityReport struct {
	Clean bool                           `json:"clean"`
	Slots []SlotInventoryIntegrityReport `json:"slots"`
}

// SlotInventoryIntegrityReport is emitted for every slot with at least one
// inventory integrity conflict.
type SlotInventoryIntegrityReport struct {
	SlotIndex             int                          `json:"slotIndex"`
	CharacterName         string                       `json:"characterName"`
	Active                bool                         `json:"active"`
	DuplicateEntryCount   int                          `json:"duplicateEntryCount"`
	ConflictingIndexCount int                          `json:"conflictingIndexCount"`
	Conflicts             []InventoryIntegrityConflict `json:"conflicts"`
}

// InventoryIntegrityConflict groups records participating in one integrity
// problem. Kind is "acquisition_bucket_collision" or "duplicate_physick".
// For an acquisition_bucket_collision, Index carries the shared Index>>1 value
// used by the existing UI label. Native low-range adjacent values are not
// conflicts; exact low duplicates still share this displayed bucket.
type InventoryIntegrityConflict struct {
	Kind  string                           `json:"kind,omitempty"`
	Index uint32                           `json:"index"`
	Items []InventoryIntegrityConflictItem `json:"items"`
}

// InventoryIntegrityConflictItem describes one inventory row participating in
// a conflict. Name and Category are populated when the item resolves through
// the DB; Unknown == true lets the frontend render a hex fallback.
type InventoryIntegrityConflictItem struct {
	Scope          string `json:"scope"`
	Row            int    `json:"row"`
	Handle         uint32 `json:"handle"`
	ItemID         uint32 `json:"itemId"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Quantity       int    `json:"quantity"`
	CurrentUpgrade int    `json:"currentUpgrade"`
	InfusionName   string `json:"infusionName"`
	Unknown        bool   `json:"unknown"`
}

const (
	integrityScopeCommon = "inventory_common"
	integrityScopeKey    = "inventory_key"
)

// GetSaveInventoryIntegrityReport scans populated slots for repairable
// inventory integrity conflicts. It is strictly read-only.
func (a *App) GetSaveInventoryIntegrityReport() (SaveInventoryIntegrityReport, error) {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()
	if a.save == nil {
		return SaveInventoryIntegrityReport{}, fmt.Errorf("no save loaded")
	}

	a.lockAllSlots()
	defer a.unlockAllSlots()

	report := SaveInventoryIntegrityReport{Clean: true}
	for i := range a.save.Slots {
		slot := &a.save.Slots[i]
		if slot.Version == 0 {
			continue
		}
		indexIssues := core.ScanDuplicateInventoryIndices(slot)
		physickIssues := core.ScanDuplicateWondrousPhysick(slot)
		if len(indexIssues) == 0 && len(physickIssues) == 0 {
			continue
		}
		report.Clean = false
		report.Slots = append(report.Slots, buildSlotIntegrityReport(a.save, i, slot, indexIssues, physickIssues, a.save.ActiveSlots[i]))
	}
	return report, nil
}

func buildSlotIntegrityReport(save *core.SaveFile, slotIdx int, slot *core.SaveSlot, indexIssues []core.DuplicateInventoryIndexIssue, physickIssues []core.WondrousPhysickOccurrence, active bool) SlotInventoryIntegrityReport {
	// Group with the same canonical key as the core scanner. Recomputing only
	// Index>>1 here previously reintroduced low-range false positives.
	conflictingKeys := make(map[core.InventoryIndexCollisionKey]struct{}, len(indexIssues))
	displayBuckets := make(map[core.InventoryIndexCollisionKey]uint32, len(indexIssues))
	for _, issue := range indexIssues {
		key := issue.CollisionKey()
		conflictingKeys[key] = struct{}{}
		displayBuckets[key] = issue.Bucket
	}

	conflictByKey := make(map[core.InventoryIndexCollisionKey]*InventoryIntegrityConflict, len(conflictingKeys))
	conflictOrder := make([]core.InventoryIndexCollisionKey, 0, len(conflictingKeys))
	seenRow := make(map[[2]int]bool, len(indexIssues)*2)

	appendRow := func(scope string, row int, item core.InventoryItem) {
		var indexKey core.InventoryIndexCollisionKey
		matched := false
		for key := range conflictingKeys {
			if key.Matches(item.Index) {
				indexKey = key
				matched = true
				break
			}
		}
		if !matched {
			return
		}
		scopeKey := 0
		if scope == integrityScopeKey {
			scopeKey = 1
		}
		key := [2]int{scopeKey, row}
		if seenRow[key] {
			return
		}
		seenRow[key] = true

		group, ok := conflictByKey[indexKey]
		if !ok {
			group = &InventoryIntegrityConflict{Kind: "acquisition_bucket_collision", Index: displayBuckets[indexKey]}
			conflictByKey[indexKey] = group
			conflictOrder = append(conflictOrder, indexKey)
		}
		group.Items = append(group.Items, resolveConflictItem(scope, row, item, slot))
	}

	for row, item := range slot.Inventory.CommonItems {
		if item.GaItemHandle == core.GaHandleEmpty || item.GaItemHandle == core.GaHandleInvalid {
			continue
		}
		appendRow(integrityScopeCommon, row, item)
	}
	for row, item := range slot.Inventory.KeyItems {
		if item.GaItemHandle == core.GaHandleEmpty || item.GaItemHandle == core.GaHandleInvalid {
			continue
		}
		appendRow(integrityScopeKey, row, item)
	}

	conflicts := make([]InventoryIntegrityConflict, 0, len(conflictOrder)+1)
	for _, key := range conflictOrder {
		conflicts = append(conflicts, *conflictByKey[key])
	}
	if len(physickIssues) > 1 {
		conflict := InventoryIntegrityConflict{Kind: "duplicate_physick"}
		for _, occ := range physickIssues {
			if occ.Row < 0 || occ.Row >= len(slot.Inventory.CommonItems) {
				continue
			}
			conflict.Items = append(conflict.Items, resolveConflictItem(occ.Scope, occ.Row, slot.Inventory.CommonItems[occ.Row], slot))
		}
		conflicts = append(conflicts, conflict)
	}

	return SlotInventoryIntegrityReport{
		SlotIndex:             slotIdx,
		CharacterName:         resolveCharacterName(save, slotIdx),
		Active:                active,
		DuplicateEntryCount:   len(indexIssues) + max(0, len(physickIssues)-1),
		ConflictingIndexCount: len(conflictingKeys),
		Conflicts:             conflicts,
	}
}

func resolveConflictItem(scope string, row int, item core.InventoryItem, slot *core.SaveSlot) InventoryIntegrityConflictItem {
	itemID, ok := slot.GaMap[item.GaItemHandle]
	if !ok {
		itemID = db.HandleToItemID(item.GaItemHandle)
	}
	displayID := db.WondrousPhysickDisplayID(itemID)
	itemData, baseID := db.GetItemDataFuzzy(displayID)

	result := InventoryIntegrityConflictItem{
		Scope:    scope,
		Row:      row,
		Handle:   item.GaItemHandle,
		ItemID:   itemID,
		Quantity: int(item.Quantity & 0x7FFFFFFF),
	}

	if itemData.Name == "" {
		result.Unknown = true
		return result
	}

	result.Name = itemData.Name
	result.Category = itemData.Category
	if isWeaponLikeCategory(itemData.Category) {
		result.CurrentUpgrade, result.InfusionName = decodeWeaponUpgradeInfusion(displayID, baseID)
	}
	return result
}

// resolveCharacterName mirrors GetCharacterNames: from the slot's player block
// first, fall back to profile summary, yield "" if both are blank so frontend
// can render its own placeholder.
func resolveCharacterName(save *core.SaveFile, slotIdx int) string {
	if name := core.UTF16ToString(save.Slots[slotIdx].Player.CharacterName[:]); name != "" {
		return name
	}
	return core.UTF16ToString(save.ProfileSummaries[slotIdx].CharacterName[:])
}

func isWeaponLikeCategory(category string) bool {
	switch category {
	case "melee_armaments", "ranged_and_catalysts", "shields":
		return true
	default:
		return false
	}
}
