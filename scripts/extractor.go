package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// PY_PROJECT_PATH is the path to the newer Python project, prioritized by the user.
const PY_PROJECT_PATH = "tmp/repos/Elden-Ring-Save-Editor/src/Resources/Json/"

// RUST_PROJECT_PATH is the path to the original Rust project, used for data not in the Python JSONs.
const RUST_PROJECT_PATH = "tmp/repos/ER-Save-Editor/src/db/"

func main() {
	fmt.Println("🚀 Starting data extraction...")

	// Ensure target directory exists
	os.MkdirAll("backend/db/data", 0755)

	// --- Data from Python Project (Primary) ---
	fmt.Println("🔍 Extracting from Python project JSON files...")
	extractNamesFromJSON("weapons.json", "Weapons")
	extractNamesFromJSON("armor.json", "Armors")
	extractNamesFromJSON("goods.json", "Items")
	extractNamesFromJSON("talisman.json", "Talismans")
	extractNamesFromJSON("aow.json", "Aows")
	extractGracesFromJSON("graces.json", "Graces")

	// --- Data from Rust Project (Secondary/Fallback) ---
	fmt.Println("🔍 Extracting from Rust project .rs files...")
	extractStats("stats.rs")
	extractClasses("classes.rs")
	extractEventFlags("event_flags.rs")

	fmt.Println("✅ Data extraction complete!")
}

// extractNamesFromJSON parses the key-value JSON files from the Python project.
func extractNamesFromJSON(filename, varName string) {
	inputPath := PY_PROJECT_PATH + filename
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("⚠️ Error opening %s: %v\n", inputPath, err)
		return
	}
	defer file.Close()

	var data map[string]string
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		fmt.Printf("⚠️ Error decoding JSON from %s: %v\n", filename, err)
		return
	}

	items := make(map[uint32]string)
	for name, idHexStr := range data {
		// Convert "80 97 FA 01" to a byte slice
		byteStr := strings.ReplaceAll(idHexStr, " ", "")
		byteSlice, err := hex.DecodeString(byteStr)
		if err != nil {
			fmt.Printf("⚠️ Skipping invalid hex '%s' for item '%s'\n", idHexStr, name)
			continue
		}

		// Interpret bytes as a little-endian uint32
		if len(byteSlice) != 4 {
			// Some items like 'Unarmed' might have different lengths, skip them.
			continue
		}
		id := binary.LittleEndian.Uint32(byteSlice)
		items[id] = name
	}

	generateGoMap(varName, items)
	fmt.Printf("   ✓ Extracted %d entries for %s\n", len(items), varName)
}

// GraceJSON represents the structure in Python project's graces.json
type GraceJSON struct {
	GraceName string `json:"grace_name"`
	MapID     uint32 `json:"map_id"`
	Offset    string `json:"offset"`
	Index     int    `json:"index"`
}

func extractGracesFromJSON(filename, varName string) {
	inputPath := PY_PROJECT_PATH + filename
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("⚠️ Error opening %s: %v\n", inputPath, err)
		return
	}
	defer file.Close()

	var data []GraceJSON
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		fmt.Printf("⚠️ Error decoding JSON from %s: %v\n", filename, err)
		return
	}

	graces := make(map[uint32]string)
	for _, g := range data {
		graces[g.MapID] = g.GraceName
	}

	generateGoMap(varName, graces)
	fmt.Printf("   ✓ Extracted %d entries for %s\n", len(graces), varName)
}

// generateGoMap writes the extracted map to a .go file.
func generateGoMap(varName string, data map[uint32]string) {
	outputPath := "backend/db/data/" + strings.ToLower(varName) + ".go"
	out, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("⚠️ Error creating output file %s: %v\n", outputPath, err)
		return
	}
	defer out.Close()

	fmt.Fprintf(out, "package data\n\n// %s contains the map of item IDs to names.\nvar %s = map[uint32]string{\n", varName, varName)

	// Sort keys for consistent output
	keys := make([]uint32, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	for _, id := range keys {
		// Escape double quotes in names
		name := strings.ReplaceAll(data[id], "\"", "\\\"")
		fmt.Fprintf(out, "\t0x%08X: \"%s\",\n", id, name)
	}
	fmt.Fprintln(out, "}")
}

func extractStats(filename string) {
	inputPath := RUST_PROJECT_PATH + filename
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
	fmt.Printf("   ✓ Extracted stats tables (HP, FP, SP)\n")
}

func extractClasses(filename string) {
	inputPath := RUST_PROJECT_PATH + filename
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
		id, ok := archeTypeIDs[name]
		if !ok {
			continue
		}

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
	fmt.Printf("   ✓ Extracted starter class stats\n")
}

func extractEventFlags(filename string) {
	inputPath := RUST_PROJECT_PATH + filename
	content, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("⚠️ Error opening %s: %v\n", filename, err)
		return
	}

	outputPath := "backend/db/data/event_flags.go"
	out, _ := os.Create(outputPath)
	defer out.Close()

	fmt.Fprintf(out, "package data\n\ntype EventFlagInfo struct {\n\tByte uint32\n\tBit  uint8\n}\n\n")
	fmt.Fprintf(out, "var EventFlags = map[uint32]EventFlagInfo{\n")

	re := regexp.MustCompile(`\((\d+),\s*\((0x[0-9A-Fa-f]+|\d+),\s*(\d+)\)\)`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	type eventFlagInfoHolder struct {
		Byte uint32
		Bit  uint8
	}
	uniqueFlags := make(map[uint32]eventFlagInfoHolder)
	for _, m := range matches {
		var id uint32
		var byteIdx uint32
		var bitIdx uint8

		fmt.Sscanf(m[1], "%d", &id)
		fmt.Sscanf(m[2], "%v", &byteIdx)
		fmt.Sscanf(m[3], "%d", &bitIdx)

		uniqueFlags[id] = eventFlagInfoHolder{Byte: byteIdx, Bit: bitIdx}
	}

	ids := make([]uint32, 0, len(uniqueFlags))
	for id := range uniqueFlags {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		info := uniqueFlags[id]
		fmt.Fprintf(out, "\t%d: {Byte: 0x%X, Bit: %d},\n", id, info.Byte, info.Bit)
	}
	fmt.Fprintln(out, "}")
	fmt.Printf("   ✓ Extracted %d event flags\n", len(uniqueFlags))
}
