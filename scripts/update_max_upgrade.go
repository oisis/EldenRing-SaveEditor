package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	files := []string{
		"backend/db/data/weapons.go",
		"backend/db/data/items.go",
		"backend/db/data/spirit_ashes.go",
		"backend/db/data/gestures.go",
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

	// Match entries like: 0x000F4240: {Name: "Dagger", MaxInventory: 1, MaxStorage: 1},
	// or with existing MaxUpgrade: 0x000F4240: {Name: "Dagger", MaxInventory: 1, MaxStorage: 1, MaxUpgrade: 0},
	reEntry := regexp.MustCompile(`\t(0x[0-9A-Fa-f]+): \{Name: "(.*)", MaxInventory: (\d+), MaxStorage: (\d+)(, MaxUpgrade: \d+)?\},`)

	isWeapons := strings.Contains(path, "weapons.go")

	for scanner.Scan() {
		line := scanner.Text()

		if match := reEntry.FindStringSubmatch(line); match != nil {
			idStr := match[1]
			name := match[2]
			maxInv := match[3]
			maxStorage := match[4]

			maxUpgrade := uint32(0)
			if isWeapons {
				maxUpgrade = getWeaponMaxUpgrade(name)
			} else {
				maxUpgrade = getItemMaxUpgrade(name)
			}

			// Escape double quotes in name if any
			escapedName := strings.ReplaceAll(name, "\"", "\\\"")

			newLine := fmt.Sprintf("\t%s: {Name: \"%s\", MaxInventory: %s, MaxStorage: %s, MaxUpgrade: %d},", idStr, escapedName, maxInv, maxStorage, maxUpgrade)
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

func getWeaponMaxUpgrade(name string) uint32 {
	nameLower := strings.ToLower(name)

	// List of Somber weapons (+10)
	somberKeywords := []string{
		"moonveil", "rivers of blood", "blasphemous blade", "dark moon greatsword",
		"bloodhound's fang", "reduvia", "black knife", "coded sword",
		"sword of night and flame", "hand of malenia", "mohgwyn's sacred spear",
		"sacred relic sword", "winged scythe", "halo scythe", "godslayer's greatsword",
		"maliketh's black blade", "dragon king's cragblade", "bolt of gransax",
		"ruins greatsword", "starscourge greatsword", "grafted blade greatsword",
		"fallingstar beast jaw", "bastard's stars", "regalia of eochaid",
		"marais executioner's sword", "eleonora's poleblade", "vyke's war spear",
		"siluria's tree", "ordovis's greatsword", "devourer's scepter",
		"scepter of the all-knowing", "glintstone kris", "blade of calling",
		"ivory sickle", "crystal knife", "scorpion's stinger", "cinquedea",
		"miquellan knight's sword", "golden epitaph", "inseparable sword",
		"nox flowing sword", "lazuli glintstone sword", "carian knight's sword",
		"crystal sword", "rotten crystal sword", "sword of st. trina",
		"velvet sword of st. trina", "star-lined sword", "stone-sheathed sword",
		"sword of light", "sword of darkness", "dragonscale blade", "serpent-hunter",
		"giant's red braid", "bastard's stars", "winged greathorn", "axe of godrick",
		"grafted dragon", "remembrance", "regalia", "carian regal scepter",
		"lusat's glintstone staff", "azur's glintstone staff", "staff of the guilty",
		"frenzied flame seal", "dragon communion seal", "golden order seal",
	}

	for _, kw := range somberKeywords {
		if strings.Contains(nameLower, kw) {
			return 10
		}
	}

	// Default for weapons
	if nameLower == "unarmed" {
		return 0
	}

	return 25
}

func getItemMaxUpgrade(name string) uint32 {
	nameLower := strings.ToLower(name)

	// Spirit Ashes (+10)
	if strings.Contains(nameLower, "ashes") || strings.Contains(nameLower, "puppet") {
		// Exclude some items that might have "ashes" in name but aren't spirits
		if !strings.Contains(nameLower, "war") && !strings.Contains(nameLower, "mountain") {
			return 10
		}
	}

	return 0
}
