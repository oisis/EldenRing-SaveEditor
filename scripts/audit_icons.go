package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"github.com/oisis/EldenRing-SaveEditor/backend/db/data"
)

func cleanName(name string, category string) string {
	clean := strings.ToLower(name)
	clean = strings.TrimPrefix(clean, "ash of war: ")
	clean = strings.TrimPrefix(clean, "sorcery: ")
	clean = strings.TrimPrefix(clean, "incantation: ")

	if category == "weapons" {
		affixes := []string{
			"heavy ", "keen ", "quality ", "fire ", "flame art ",
			"lightning ", "sacred ", "magic ", "cold ", "poison ",
			"blood ", "occult ", "bloody ",
		}
		for _, affix := range affixes {
			if strings.HasPrefix(clean, affix) {
				clean = clean[len(affix):]
				break
			}
		}
	}

	// Remove upgrade levels
	if idx := strings.LastIndex(clean, " +"); idx != -1 {
		clean = clean[:idx]
	}

	// Normalization
	clean = strings.ReplaceAll(clean, "'", "")
	clean = strings.ReplaceAll(clean, " ", "_")
	clean = strings.ReplaceAll(clean, "(", "")
	clean = strings.ReplaceAll(clean, ")", "")
	clean = strings.ReplaceAll(clean, "+", "")
	clean = strings.ReplaceAll(clean, ",", "")
	clean = strings.ReplaceAll(clean, "[", "")
	clean = strings.ReplaceAll(clean, "]", "")
	clean = strings.ReplaceAll(clean, ":", "")
	clean = strings.ReplaceAll(clean, "!", "")
	clean = strings.ReplaceAll(clean, "-", "_") // Consistent with your instruction to use _

	if category == "armor" && strings.Contains(strings.ToLower(name), " (altered)") {
		clean = strings.ReplaceAll(clean, "_altered", "") + "_altered"
	}

	if clean == "golden_vow" && category == "ashes" {
		clean = "ashes_of_war_golden_vow"
	}

	return clean
}

func main() {
	categories := map[string]map[uint32]string{
		"weapons":   data.Weapons,
		"armor":     data.Armors,
		"goods":     data.Items,
		"talismans": data.Talismans,
		"ashes":     data.Aows,
	}

	missingCount := 0
	totalCount := 0

	for cat, items := range categories {
		fmt.Printf("Auditing category: %s\n", cat)
		for id, name := range items {
			// Skip duplicates/upgrades for weapons to avoid noise
			if cat == "weapons" && id%100 != 0 {
				continue
			}
			if name == "" || strings.HasPrefix(name, "Unknown") || strings.HasPrefix(name, "?") {
				continue
			}

			totalCount++
			fileName := cleanName(name, cat) + ".png"
			path := filepath.Join("frontend", "public", "items", cat, fileName)

			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Printf("MISSING: [%s] %s -> %s\n", cat, name, fileName)
				missingCount++
			}
		}
	}

	fmt.Printf("\nAudit finished.\nTotal items checked: %d\nMissing icons: %d\n", totalCount, missingCount)
}
