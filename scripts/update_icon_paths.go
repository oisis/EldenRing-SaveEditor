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

	// Match entries like: 0x000F4240: {Name: "Dagger", MaxInventory: 1, MaxStorage: 1, MaxUpgrade: 25},
	reEntry := regexp.MustCompile(`\t(0x[0-9A-Fa-f]+): \{Name: "(.*)", MaxInventory: (\d+), MaxStorage: (\d+), MaxUpgrade: (\d+)\},`)

	category := filepath.Base(path)
	catDir := ""
	switch category {
	case "weapons.go":
		catDir = "weapons"
	case "armors.go":
		catDir = "armor"
	case "items.go":
		catDir = "goods"
	case "talismans.go":
		catDir = "talismans"
	case "aows.go":
		catDir = "ashes"
	}

	for scanner.Scan() {
		line := scanner.Text()

		if match := reEntry.FindStringSubmatch(line); match != nil {
			idStr := match[1]
			name := match[2]
			maxInv := match[3]
			maxStorage := match[4]
			maxUpgrade := match[5]

			iconPath := getItemIconPath(name, catDir)

			// Escape double quotes in name if any
			escapedName := strings.ReplaceAll(name, "\"", "\\\"")

			newLine := fmt.Sprintf("\t%s: {Name: \"%s\", MaxInventory: %s, MaxStorage: %s, MaxUpgrade: %s, IconPath: \"%s\"},", idStr, escapedName, maxInv, maxStorage, maxUpgrade, iconPath)
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

func getItemIconPath(name string, catDir string) string {
	cleanName := strings.ToLower(name)

	// 1. Final character normalization (only letters, numbers, and underscores)
	// Remove apostrophes
	cleanName = strings.ReplaceAll(cleanName, "'", "")
	// Replace spaces and hyphens with underscores
	cleanName = strings.ReplaceAll(cleanName, " ", "_")
	cleanName = strings.ReplaceAll(cleanName, "-", "_")

	// Remove everything except letters, numbers, and underscores
	reg := regexp.MustCompile(`[^\w]`)
	cleanName = reg.ReplaceAllString(cleanName, "")

	// Collapse multiple underscores
	reg2 := regexp.MustCompile(`_+`)
	cleanName = reg2.ReplaceAllString(cleanName, "_")

	// Trim underscores from ends
	cleanName = strings.Trim(cleanName, "_")

	// 2. Special cases
	if cleanName == "golden_vow" && catDir == "ashes" {
		cleanName = "ashes_of_war_golden_vow"
	}

	return fmt.Sprintf("items/%s/%s.png", catDir, cleanName)
}
