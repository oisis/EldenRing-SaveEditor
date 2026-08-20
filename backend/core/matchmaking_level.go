package core

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/db"
)

// somberMatchmakingTable maps Somber weapon upgrade levels +0..+10 to the matchmaking weapon scale.
// Values: 0, 0, 5, 7, 10, 12, 15, 17, 20, 24, 25.
var somberMatchmakingTable = [...]uint8{0, 0, 5, 7, 10, 12, 15, 17, 20, 24, 25}

// isMatchmakingWeaponCategory returns true only for genuine weapons (melee, ranged/catalysts, shields).
// Spirit Ashes (category "ashes") and other non-weapon items are excluded.
func isMatchmakingWeaponCategory(category string) bool {
	switch category {
	case "melee_armaments", "ranged_and_catalysts", "shields":
		return true
	}
	return false
}

// ComputeWeaponMatchmakingLevel calculates the matchmaking level (0..25) for a given ItemID.
// Returns 0 for non-weapons, unknown items, or unupgraded base weapons.
func ComputeWeaponMatchmakingLevel(itemID uint32) uint8 {
	itemData, baseID := db.GetItemDataFuzzy(itemID)
	if itemData.Name == "" || baseID == 0 || !isMatchmakingWeaponCategory(itemData.Category) {
		return 0
	}
	if itemID < baseID {
		return 0
	}
	level := int((itemID - baseID) % 100)
	if itemData.MaxUpgrade == 10 {
		if level <= 0 {
			return 0
		}
		if level < len(somberMatchmakingTable) {
			return somberMatchmakingTable[level]
		}
		return somberMatchmakingTable[len(somberMatchmakingTable)-1]
	}
	if level <= 0 {
		return 0
	}
	if level > 25 {
		return 25
	}
	return uint8(level)
}

// MaxOwnedWeaponMatchmakingLevel scans Inventory and Storage in slot and returns
// the maximum matchmaking weapon level (0..25) across all currently held weapons.
func MaxOwnedWeaponMatchmakingLevel(slot *SaveSlot) uint8 {
	if slot == nil {
		return 0
	}
	maxLvl := uint8(0)
	scanItems := func(items []InventoryItem) {
		for _, it := range items {
			if it.GaItemHandle == 0 || it.GaItemHandle == GaHandleInvalid {
				continue
			}
			itemID := uint32(0)
			if id, ok := slot.GaMap[it.GaItemHandle]; ok && id != 0 {
				itemID = id
			} else {
				itemID = db.HandleToItemID(it.GaItemHandle)
			}
			lvl := ComputeWeaponMatchmakingLevel(itemID)
			if lvl > maxLvl {
				maxLvl = lvl
			}
		}
	}
	scanItems(slot.Inventory.CommonItems)
	scanItems(slot.Storage.CommonItems)
	return maxLvl
}

// SyncMatchmakingWeaponLevel updates the durable matchmaking weapon level byte
// at slot.MagicOffset - 0xD5 based on the highest weapon currently owned in
// Inventory or Storage. It is monotonic (only raises, never lowers) and fails closed.
func SyncMatchmakingWeaponLevel(slot *SaveSlot) error {
	if slot == nil || len(slot.Data) == 0 {
		return fmt.Errorf("sync matchmaking level: nil slot or empty data")
	}
	if slot.MagicOffset < MinMagicOffset {
		if len(slot.Data) >= SlotSize {
			return fmt.Errorf("sync matchmaking level: invalid magic offset %d (len %d)",
				slot.MagicOffset, len(slot.Data))
		}
		return nil
	}
	targetOffset := slot.MagicOffset + OffMatchmakingWeaponLevel
	if targetOffset < 0 || targetOffset >= len(slot.Data) {
		return fmt.Errorf("sync matchmaking level: target offset %d out of range (len %d)",
			targetOffset, len(slot.Data))
	}

	maxLvl := MaxOwnedWeaponMatchmakingLevel(slot)
	currentLvl := slot.Data[targetOffset]
	if maxLvl > currentLvl {
		slot.Data[targetOffset] = maxLvl
	}
	return nil
}
