package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

func main() {
	fmt.Println("🚀 Starting data extraction from Rust source...")
	
	// Ensure directory exists
	os.MkdirAll("backend/db/data", 0755)
	
	// Phase 2.2: Constants
	extractItems("weapons.rs", "Weapons")
	extractItems("armors.rs", "Armors")
	extractItems("items.rs", "Items")
	extractItems("talismans.rs", "Talismans")
	extractGraces("graces.rs", "Graces")
	
	// Phase 2.3: Stats & Classes
	extractStats("stats.rs")
	extractClasses("classes.rs")
	
	fmt.Println("✅ Data extraction complete!")
}

func extractItems(filename, varName string) {
	inputPath := "tmp/org-src/src/db/" + filename
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("⚠️ Error opening %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	re := regexp.MustCompile(`(0x[0-9A-Fa-f]+),\s*//\s*(.*)`)
	items := make(map[string]string)
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) == 3 {
			items[matches[1]] = strings.TrimSpace(matches[2])
		}
	}

	generateGoMap(varName, items)
}

func extractGraces(filename, varName string) {
	inputPath := "tmp/org-src/src/db/" + filename
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("⚠️ Error opening %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	re := regexp.MustCompile(`\(Grace::.*,\s*\(MapName::(.*),\s*(\d+),\s*"(.*)"\)\)`)
	graces := make(map[string]string)
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) == 4 {
			id := matches[2]
			region := matches[1]
			name := matches[3]
			graces[id] = fmt.Sprintf("%s (%s)", name, region)
		}
	}

	generateGoMap(varName, graces)
}

func extractStats(filename string) {
	inputPath := "tmp/org-src/src/db/" + filename
	content, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("⚠️ Error opening %s: %v\n", filename, err)
		return
	}
	
	re := regexp.MustCompile(`pub const (HP|FP|SP): \[f32; \d+\] = (\[[\s\S]*?\]);`)
	matches := re.FindAllStringSubmatch(string(content), -1)
	
	outputPath := "backend/db/data/stats.go"
	out, _ := os.Create(outputPath)
	defer out.Close()

	fmt.Fprintf(out, "package data\n\n")
	for _, m := range matches {
		name := m[1]
		rawValues := m[2]
		values := "[]float32{" + rawValues[1:len(rawValues)-1] + "}"
		values = regexp.MustCompile(`(\d+)\.,`).ReplaceAllString(values, "$1.0,")
		fmt.Fprintf(out, "var %s = %s\n\n", name, values)
	}
}

func extractClasses(filename string) {
	inputPath := "tmp/org-src/src/db/" + filename
	content, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("⚠️ Error opening %s: %v\n", filename, err)
		return
	}
	
	outputPath := "backend/db/data/classes.go"
	out, _ := os.Create(outputPath)
	defer out.Close()

	fmt.Fprintf(out, "package data\n\ntype StarterStats struct {\n\tLevel, Vigor, Mind, Endurance, Strength, Dexterity, Intelligence, Faith, Arcane uint32\n}\n\n")
	fmt.Fprintf(out, "var StarterClasses = map[uint8]StarterStats{\n")

	re := regexp.MustCompile(`ArcheType::(\w+),\s*Stats\{([\s\S]*?)\}`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	archeTypeIDs := map[string]uint8{
		"Vagabond": 0, "Warrior": 1, "Hero": 2, "Bandit": 3, "Astrologer": 4,
		"Prophet": 5, "Confessor": 6, "Samurai": 7, "Prisoner": 8, "Wretch": 9,
	}

	for _, m := range matches {
		name := m[1]
		statsRaw := m[2]
		id := archeTypeIDs[name]

		stats := make(map[string]string)
		statLines := strings.Split(statsRaw, ",")
		for _, line := range statLines {
			kv := strings.Split(line, ":")
			if len(kv) == 2 {
				k := strings.TrimSpace(kv[0])
				v := strings.TrimSpace(kv[1])
				stats[k] = v
			}
		}

		fmt.Fprintf(out, "\t%d: {Level: %s, Vigor: %s, Mind: %s, Endurance: %s, Strength: %s, Dexterity: %s, Intelligence: %s, Faith: %s, Arcane: %s}, // %s\n",
			id, stats["level"], stats["vigor"], stats["mind"], stats["endurance"],
			stats["strength"], stats["dexterity"], stats["intelligence"], stats["faith"], stats["arcane"], name)
	}
	fmt.Fprintln(out, "}")
}

func generateGoMap(varName string, data map[string]string) {
	outputPath := "backend/db/data/" + strings.ToLower(varName) + ".go"
	out, _ := os.Create(outputPath)
	defer out.Close()

	fmt.Fprintf(out, "package data\n\nvar %s = map[uint32]string{\n", varName)
	
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, id := range keys {
		fmt.Fprintf(out, "\t%s: %q,\n", id, data[id])
	}
	fmt.Fprintln(out, "}")
}
