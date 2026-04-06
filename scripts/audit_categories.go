package main

import (
	"fmt"
	"strings"
	"github.com/oisis/EldenRing-SaveEditor/backend/db/data"
)

func main() {
	// Audit Weapons category
	fmt.Println("Items in data.Weapons that are NOT categorized as 'weapons':")
	for id, item := range data.Weapons {
		if id%100 != 0 {
			continue
		}
		subCat := GetItemSubCategory(id, item.Name, "Weapon")
		if subCat != "weapons" {
			fmt.Printf("ID: 0x%08X, Name: %-30s, SubCat: %s\n", id, item.Name, subCat)
		}
	}
}

func GetItemSubCategory(id uint32, name string, broadCategory string) string {
	if broadCategory == "Weapon" {
		if itemMatchesCategory(id, name, "ammo") {
			return "ammo"
		}
		if itemMatchesCategory(id, name, "bows") {
			return "bows"
		}
		if itemMatchesCategory(id, name, "seals") {
			return "seals"
		}
		if itemMatchesCategory(id, name, "staffs") {
			return "staffs"
		}
		if itemMatchesCategory(id, name, "shields") {
			return "shields"
		}
		return "weapons"
	}
	return "unknown"
}

func itemMatchesCategory(id uint32, name string, category string) bool {
	nameLower := strings.ToLower(name)
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
