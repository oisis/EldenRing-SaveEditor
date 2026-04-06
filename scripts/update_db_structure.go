package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	files := []string{
		"backend/db/data/aows.go",
		"backend/db/data/armors.go",
		"backend/db/data/consumables.go",
		"backend/db/data/standard_ashes.go",
		"backend/db/data/renowned_ashes.go",
		"backend/db/data/legendary_ashes.go",
		"backend/db/data/puppets.go",
		"backend/db/data/special_ashes.go",
		"backend/db/data/gestures.go",
		"backend/db/data/talismans.go",
		"backend/db/data/weapons.go",
		"backend/db/data/bows.go",
		"backend/db/data/shields.go",
		"backend/db/data/staffs.go",
		"backend/db/data/seals.go",
		"backend/db/data/ammo.go",
		"backend/db/data/helms.go",
		"backend/db/data/chest.go",
		"backend/db/data/gauntlets.go",
		"backend/db/data/leggings.go",
		"backend/db/data/sorceries.go",
		"backend/db/data/incantations.go",
		"backend/db/data/base_materials.go",
		"backend/db/data/dlc_materials.go",
		"backend/db/data/smithing_stones.go",
		"backend/db/data/gloveworts.go",
		"backend/db/data/sacred_flasks.go",
		"backend/db/data/throwing_pots.go",
		"backend/db/data/perfume_arts.go",
		"backend/db/data/throwables.go",
		"backend/db/data/grease.go",
		"backend/db/data/misc_tools.go",
		"backend/db/data/quest_tools.go",
		"backend/db/data/golden_runes.go",
		"backend/db/data/remembrances.go",
		"backend/db/data/multiplayer.go",
		"backend/db/data/keyitems.go",
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}
		if err := processFile(file); err != nil {
			fmt.Printf("Error processing %s: %v\n", file, err)
		}
	}
}

func processFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	reMapStart := regexp.MustCompile(`var (\w+) = map\[uint32\]ItemData\{`)
	reEntry := regexp.MustCompile(`\t(0x[0-9A-Fa-f]+): \{Name: "([^"]+)", Category: "([^"]*)", MaxInventory: (\d+), MaxStorage: (\d+), MaxUpgrade: (\d+), IconPath: "([^"]*)"\},`)

	filename := filepath.Base(path)

	for scanner.Scan() {
		line := scanner.Text()

		if match := reMapStart.FindStringSubmatch(line); match != nil {
			lines = append(lines, line)
			continue
		}

		if match := reEntry.FindStringSubmatch(line); match != nil {
			idStr := match[1]
			name := match[2]
			
			maxInv, maxStorage := getLimits(name, filename)
			category := determineCategory(name, filename)
			
			maxUpgrade := match[6]
			iconPath := match[7]

			escapedName := strings.ReplaceAll(name, "\"", "\\\"")
			newLine := fmt.Sprintf("\t%s: {Name: \"%s\", Category: \"%s\", MaxInventory: %d, MaxStorage: %d, MaxUpgrade: %s, IconPath: \"%s\"},", 
				idStr, escapedName, category, maxInv, maxStorage, maxUpgrade, iconPath)
			lines = append(lines, newLine)
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func determineCategory(name, filename string) string {
	cat := strings.TrimSuffix(filename, ".go")
	if cat == "spirit_ashes" {
		return "spiritashes"
	}
	return cat
}

func getLimits(name, filename string) (uint32, uint32) {
	nameLower := strings.ToLower(name)

	// Equipment
	if filename == "weapons.go" || filename == "bows.go" || filename == "shields.go" || 
	   filename == "staffs.go" || filename == "seals.go" || filename == "talismans.go" || 
	   filename == "aows.go" || filename == "helms.go" || filename == "chest.go" || 
	   filename == "gauntlets.go" || filename == "leggings.go" {
		return 1, 1
	}

	// Ashes
	if strings.HasSuffix(filename, "_ashes.go") || filename == "puppets.go" {
		return 1, 1
	}

	// Tools & Items
	switch filename {
	case "sacred_flasks.go":
		return 1, 0
	case "throwing_pots.go", "perfume_arts.go", "grease.go", "throwables.go", "consumables.go":
		return 99, 600
	case "base_materials.go", "dlc_materials.go", "smithing_stones.go", "gloveworts.go":
		return 999, 999
	case "ammo.go":
		return 99, 600
	case "golden_runes.go":
		return 99, 600
	case "remembrances.go", "multiplayer.go", "misc_tools.go", "quest_tools.go", "keyitems.go":
		return 1, 0
	}

	return 1, 1
}
