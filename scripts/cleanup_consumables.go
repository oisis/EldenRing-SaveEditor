package main

import (
	"fmt"
	"os"
	"strings"
	"github.com/oisis/EldenRing-SaveEditor/backend/db/data"
)

func main() {
	// 1. Collect all IDs from other maps
	otherIDs := make(map[uint32]string)
	
	// List of all maps EXCEPT Consumables
	maps := map[string]map[uint32]data.ItemData{
		"ammo": data.Ammo,
		"aows": data.Aows,
		"base_materials": data.BaseMaterials,
		"bows": data.Bows,
		"chest": data.Chest,
		"dlc_materials": data.DlcMaterials,
		"gauntlets": data.Gauntlets,
		"gestures": data.Gestures,
		"gloveworts": data.Gloveworts,
		"golden_runes": data.GoldenRunes,
		"grease": data.Grease,
		"helms": data.Helms,
		"incantations": data.Incantations,
		"keyitems": data.Keyitems,
		"legendary_ashes": data.LegendaryAshes,
		"leggings": data.Leggings,
		"misc_tools": data.MiscTools,
		"multiplayer": data.Multiplayer,
		"perfume_arts": data.PerfumeArts,
		"puppets": data.Puppets,
		"quest_tools": data.QuestTools,
		"remembrances": data.Remembrances,
		"renowned_ashes": data.RenownedAshes,
		"sacred_flasks": data.SacredFlasks,
		"seals": data.Seals,
		"shields": data.Shields,
		"smithing_stones": data.SmithingStones,
		"sorceries": data.Sorceries,
		"staffs": data.Staffs,
		"standard_ashes": data.StandardAshes,
		"talismans": data.Talismans,
		"throwables": data.Throwables,
		"throwing_pots": data.ThrowingPots,
		"weapons": data.Weapons,
	}

	for name, m := range maps {
		for id := range m {
			otherIDs[id] = name
		}
	}

	// 2. Filter Consumables
	cleanConsumables := make(map[uint32]data.ItemData)
	removedCount := 0
	
	for id, item := range data.Consumables {
		if origin, exists := otherIDs[id]; exists {
			fmt.Printf("Removing duplicate from Consumables: %s (ID: 0x%X, exists in %s)\n", item.Name, id, origin)
			removedCount++
			continue
		}
		
		// Also remove items that clearly don't belong (e.g. Spirit Ashes, Spells)
		// based on their name or ID prefix if they weren't caught by other maps
		if strings.Contains(item.Name, "Ashes") || strings.Contains(item.Name, "Sorcery") || strings.Contains(item.Name, "Incantation") {
			fmt.Printf("Removing non-consumable by name: %s (ID: 0x%X)\n", item.Name, id)
			removedCount++
			continue
		}

		cleanConsumables[id] = item
	}

	fmt.Printf("Total removed: %d. Remaining: %d\n", removedCount, len(cleanConsumables))

	// 3. Write clean file
	f, err := os.Create("backend/db/data/consumables.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.WriteString("package data\n\nvar Consumables = map[uint32]ItemData{\n")
	for id, item := range cleanConsumables {
		f.WriteString(fmt.Sprintf("\t0x%X: {Name: %q, Category: \"consumables\", MaxInventory: %d, MaxStorage: %d, MaxUpgrade: %d, IconPath: %q},\n", 
			id, item.Name, item.MaxInventory, item.MaxStorage, item.MaxUpgrade, item.IconPath))
	}
	f.WriteString("}\n")
}
