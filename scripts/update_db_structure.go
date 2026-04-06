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
		"backend/db/data/items.go",
		"backend/db/data/spirit_ashes.go",
		"backend/db/data/gestures.go",
		"backend/db/data/talismans.go",
		"backend/db/data/weapons.go",
	}

	for _, file := range files {
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
	// Also handle the old format if it was already converted but we want to re-run with better rules
	reMapStartOld := regexp.MustCompile(`var (\w+) = map\[uint32\]string\{`)

	reEntry := regexp.MustCompile(`\t(0x[0-9A-Fa-f]+): \{(Name: )?"(.*)", (MaxInventory: \d+, MaxStorage: \d+)?\},`)
	reEntryOld := regexp.MustCompile(`\t(0x[0-9A-Fa-f]+): "(.*)",`)

	var mapName string
	category := filepath.Base(path)

	for scanner.Scan() {
		line := scanner.Text()

		if match := reMapStart.FindStringSubmatch(line); match != nil {
			mapName = match[1]
			lines = append(lines, fmt.Sprintf("var %s = map[uint32]ItemData{", mapName))
			continue
		}
		if match := reMapStartOld.FindStringSubmatch(line); match != nil {
			mapName = match[1]
			lines = append(lines, fmt.Sprintf("var %s = map[uint32]ItemData{", mapName))
			continue
		}

		if match := reEntry.FindStringSubmatch(line); match != nil {
			id := match[1]
			name := match[3]
			maxInv, maxStorage := getLimits(name, category)
			escapedName := strings.ReplaceAll(name, "\"", "\\\"")
			newLine := fmt.Sprintf("\t%s: {Name: \"%s\", MaxInventory: %d, MaxStorage: %d},", id, escapedName, maxInv, maxStorage)
			lines = append(lines, newLine)
			continue
		}
		if match := reEntryOld.FindStringSubmatch(line); match != nil {
			id := match[1]
			name := match[2]
			maxInv, maxStorage := getLimits(name, category)
			escapedName := strings.ReplaceAll(name, "\"", "\\\"")
			newLine := fmt.Sprintf("\t%s: {Name: \"%s\", MaxInventory: %d, MaxStorage: %d},", id, escapedName, maxInv, maxStorage)
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

func getLimits(name, filename string) (uint32, uint32) {
	nameLower := strings.ToLower(name)

	// Default for equipment
	if filename == "weapons.go" || filename == "armors.go" || filename == "talismans.go" || filename == "aows.go" {
		return 1, 1
	}

	// Items (goods)
	if filename == "items.go" {
		// Flasks
		if strings.Contains(nameLower, "flask of") {
			return 1, 0
		}

		// Pots (Consumable versions)
		if strings.Contains(nameLower, " pot") && !strings.Contains(nameLower, "cracked") && !strings.Contains(nameLower, "ritual") {
			return 20, 600
		}

		// Perfume Bottles (Consumable versions)
		if strings.Contains(nameLower, "perfume") && !strings.Contains(nameLower, "bottle") {
			return 10, 600
		}

		// Key Items / Tools
		keyItemKeywords := []string{
			"key", "map", "letter", "note", "painting", "bell bearing",
			"crystal tear", "great rune", "mending rune", "remembrance",
			"shackle", "whetblade", "cookbook", "scroll", "gesture",
			"cracked pot", "ritual pot", "perfume bottle", "memory stone",
			"talisman pouch", "withered finger", "furled finger", "severer",
			"effigy", "cipher ring", "bloody finger", "recusant finger",
			"whistle", "physick", "telescope", "lantern", "tonic",
		}
		for _, kw := range keyItemKeywords {
			if strings.Contains(nameLower, kw) {
				return 1, 0
			}
		}

		// Upgrade Materials (Smithing Stones, Gloveworts)
		if strings.Contains(nameLower, "smithing stone") || strings.Contains(nameLower, "glovewort") {
			return 999, 999
		}

		// Crafting Materials (Commonly found in the world)
		craftingKeywords := []string{
			"mushroom", "leaf", "flower", "fruit", "butterfly", "firefly",
			"root", "moss", "resin", "bone", "feather", "liver", "meat",
			"blood", "eye", "skin", "horn", "fang", "claw", "scale",
			"shell", "egg", "string", "crystal", "fragment", "shard",
			"arteria", "starlight", "dew", "nectar", "mold", "calculus",
		}
		for _, kw := range craftingKeywords {
			if strings.Contains(nameLower, kw) {
				return 999, 999
			}
		}

		// Ammo
		if strings.Contains(nameLower, "arrow") || strings.Contains(nameLower, "bolt") {
			return 99, 600
		}

		// Runes
		if strings.Contains(nameLower, "rune") && (strings.Contains(nameLower, "golden") || strings.Contains(nameLower, "hero's") || strings.Contains(nameLower, "lord's") || strings.Contains(nameLower, "numen's")) {
			return 99, 600
		}

		// Consumables / Materials (Default)
		return 99, 999
	}

	return 1, 1
}
