package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ItemGroup struct {
	Category string   `json:"category"`
	Items    []Item   `json:"items"`
}

type Item struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}

func main() {
	dbPath := "org-src/src/db"
	outputPath := "backend/db/json"

	if err := os.MkdirAll(outputPath, 0755); err != nil {
		panic(err)
	}

	files := []string{"weapons.rs", "armors.rs", "items.rs", "talismans.rs"}

	for _, file := range files {
		extractGroups(filepath.Join(dbPath, file), filepath.Join(outputPath, strings.TrimSuffix(file, ".rs")+".json"))
	}

	extractSimpleMap(filepath.Join(dbPath, "graces.rs"), filepath.Join(outputPath, "graces.json"), `\(Grace::.+?,\s*\(MapName::.+?,\s*(\d+)\s*,\s*"(.+?)"\)\)`)
	extractSimpleMap(filepath.Join(dbPath, "bosses.rs"), filepath.Join(outputPath, "bosses.json"), `\(Boss::.+?,\s*\((\d+)\s*,\s*"(.+?)"\)\)`)
}

func extractSimpleMap(inputPath, outputPath, regexStr string) {
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("Error opening %s: %v\n", inputPath, err)
		return
	}
	defer file.Close()

	var items []Item
	re := regexp.MustCompile(regexStr)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		match := re.FindStringSubmatch(line)
		if match != nil {
			id, _ := strconv.ParseUint(match[1], 10, 32)
			items = append(items, Item{
				ID:   uint32(id),
				Name: match[2],
			})
		}
	}

	jsonData, _ := json.MarshalIndent(items, "", "  ")
	os.WriteFile(outputPath, jsonData, 0644)
	fmt.Printf("Extracted %d items to %s\n", len(items), outputPath)
}

func extractGroups(inputPath, outputPath string) {
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("Error opening %s: %v\n", inputPath, err)
		return
	}
	defer file.Close()

	var groups []ItemGroup
	var currentGroup *ItemGroup

	reCategory := regexp.MustCompile(`\.insert\("(.+?)"\.to_string\(\),\s*vec!\[`)
	reItem := regexp.MustCompile(`(0x[0-9A-Fa-f]+),\s*//\s*(.+)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		catMatch := reCategory.FindStringSubmatch(line)

		if catMatch != nil {
			if currentGroup != nil {
				groups = append(groups, *currentGroup)
			}
			currentGroup = &ItemGroup{
				Category: catMatch[1],
				Items:    []Item{},
			}
			continue
		}

		itemMatch := reItem.FindStringSubmatch(line)
		if itemMatch != nil && currentGroup != nil {
			idStr := itemMatch[1]
			name := strings.TrimSpace(itemMatch[2])
			id, _ := strconv.ParseUint(strings.TrimPrefix(idStr, "0x"), 16, 32)
			currentGroup.Items = append(currentGroup.Items, Item{
				ID:   uint32(id),
				Name: name,
			})
		}

		if strings.Contains(line, "]);") && currentGroup != nil {
			groups = append(groups, *currentGroup)
			currentGroup = nil
		}
	}

	jsonData, _ := json.MarshalIndent(groups, "", "  ")
	os.WriteFile(outputPath, jsonData, 0644)
	fmt.Printf("Extracted %d groups to %s\n", len(groups), outputPath)
}
