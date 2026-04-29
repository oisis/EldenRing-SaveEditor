package data

// info_subcat.go — sub-category assignment for the Info tab.
//
// Sub-groups (in-game order):
//   1. Letters / Meeting Place Maps / Paintings (base game)
//   2. Letters / Meeting Place Maps / Paintings (DLC)
//   3. Mechanics / Locations Info — tutorials + notes (base + DLC combined)
//
// Classification rules:
//   - Name starts with "About "                      → Mechanics
//   - Name starts with "Note:" or contains " Note"    → Mechanics
//   - Otherwise (Letters / Maps / Paintings / Messages):
//        has "dlc" flag → DLC group
//        else           → base group
//
// Region/World Maps live in key_items.go (sub: World Maps), not info.

import "strings"

func classifyInfoItem(item ItemData) string {
	name := item.Name
	if strings.HasPrefix(name, "About ") {
		return SubcatInfoMechanicsLocations
	}
	if strings.HasPrefix(name, "Note:") || strings.Contains(name, " Note") {
		return SubcatInfoMechanicsLocations
	}
	for _, f := range item.Flags {
		if f == "dlc" {
			return SubcatInfoDLCLettersMaps
		}
	}
	return SubcatInfoLettersMaps
}

func init() {
	for id, item := range Information {
		if item.SubCategory != "" {
			continue
		}
		item.SubCategory = classifyInfoItem(item)
		Information[id] = item
	}
}
