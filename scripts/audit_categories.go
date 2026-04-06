package main

import (
	"fmt"
	"github.com/oisis/EldenRing-SaveEditor/backend/db/data"
	"strings"
)

func main() {
	// Audit Weapons category
	fmt.Println("Items in data.Weapons that are NOT categorized as 'weapons':")
	for id, item := range data.Weapons {
		if id%100 != 0 {
			continue
		}
		subCat := GetItemSubCategory(id, item, "Weapon")
		if subCat != "weapons" {
			fmt.Printf("ID: 0x%08X, Name: %-30s, SubCat: %s\n", id, item.Name, subCat)
		}
	}
}

func GetItemSubCategory(id uint32, item data.ItemData, broadCategory string) string {
	if broadCategory == "Weapon" {
		if itemMatchesCategory(id, item, "ammo") {
			return "ammo"
		}
		if itemMatchesCategory(id, item, "bows") {
			return "bows"
		}
		if itemMatchesCategory(id, item, "seals") {
			return "seals"
		}
		if itemMatchesCategory(id, item, "staffs") {
			return "staffs"
		}
		if itemMatchesCategory(id, item, "shields") {
			return "shields"
		}
		return "weapons"
	}
	return "unknown"
}

func itemMatchesCategory(id uint32, item data.ItemData, category string) bool {
	nameLower := strings.ToLower(item.Name)
	switch category {
	case "bows":
		return strings.Contains(nameLower, "bow") || strings.Contains(nameLower, "ballista") || strings.Contains(nameLower, "crossbow")
	case "seals":
		return strings.Contains(nameLower, "seal")
	case "staffs":
		return strings.Contains(nameLower, "staff") || strings.Contains(nameLower, "scepter")
	case "shields":
		return strings.Contains(nameLower, "shield") || strings.Contains(nameLower, "buckler")
	case "ammo":
		if strings.Contains(nameLower, "bolt of gransax") {
			return false
		}
		return strings.Contains(nameLower, "arrow") || strings.Contains(nameLower, "bolt")
	}
	return false
}
