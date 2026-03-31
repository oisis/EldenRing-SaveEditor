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
		byteStr := strings.ReplaceAll(idHexStr, " ", "")
		byteSlice, err := hex.DecodeString(byteStr)
		if err != nil {
			continue
		}

		var rawID uint32
		if len(byteSlice) == 4 {
			rawID = binary.LittleEndian.Uint32(byteSlice)
		} else if len(byteSlice) < 4 {
			padded := make([]byte, 4)
			copy(padded, byteSlice)
			rawID = binary.LittleEndian.Uint32(padded)
		} else {
			rawID = binary.LittleEndian.Uint32(byteSlice[:4])
		}

		// Clean ID from any existing prefix
		baseID := rawID & 0x0FFFFFFF
		
		// Generate variants based on category
		switch varName {
		case "Weapons":
			items[baseID] = name             // 0x0...
			items[baseID|0x80000000] = name  // 0x8... (Handle)
		case "Armors":
			items[baseID|0x10000000] = name  // 0x1...
			items[baseID|0x90000000] = name  // 0x9... (Handle)
		case "Talismans":
			items[baseID|0x20000000] = name  // 0x2...
			items[baseID|0xA0000000] = name  // 0xA... (Handle)
		case "Items":
			items[baseID|0x40000000] = name  // 0x4...
			items[baseID|0xB0000000] = name  // 0xB... (Handle)
		case "Aows":
			items[baseID|0x80000000] = name  // 0x8...
			items[baseID|0xC0000000] = name  // 0xC... (Handle)
		default:
			items[rawID] = name
		}
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

func getHardcodedItems(varName string) map[uint32]string {
	switch varName {
	case "Armors":
		return map[uint32]string{
			0x10002710: "Naked (Head)",
			0x10002774: "Naked (Body)",
			0x100027D8: "Naked (Arms)",
			0x1000283C: "Naked (Legs)",
		}
	case "Items":
		return map[uint32]string{
			0x40000064: "Tarnished's Furled Finger (Base)",
			0x40000065: "Small Golden Effigy (Base)",
			0x40000066: "Small Golden Effigy (Base Alt)",
			0x40000067: "Finger Severer (Base)",
			0x40000068: "Blue Cipher Ring (Base)",
			0x40000069: "White Cipher Ring (Base)",
			0x4000006C: "Duelist's Furled Finger (Base Alt)",
			0x4000006D: "Duelist's Furled Finger (Base)",
			0x4000006E: "Small Red Effigy (Base)",
			0x4000006F: "Small Red Effigy (Base Alt)",
			0x40000070: "Finger Severer (Base Alt)",
			0x40000082: "Duelist's Furled Finger (Alt)",
			0x40000096: "Tarnished's Furled Finger (Alt)",
			0x400000A6: "Vision of Grace",
			0x400000FA: "Flask of Wondrous Physick (Base)",
			0x40001FBF: "About Leveling Up",
			0x40002020: "About Sites of Grace",
			0x40002021: "About Map",
			0x400021FC: "About Fast Travel",
			0x400021FD: "About Map Gaps (Alt 3)",
			0x400021FE: "About Summoning Pools",
			0x400021FF: "About Stakes of Marika",
			0x40002200: "About Compass",
			0x40002201: "About Map Gaps (Alt 4)",
			0x40002203: "About Markers",
			0x40002204: "About Beacons",
			0x40002205: "About Map Symbols",
			0x40002206: "About Map Gaps (Alt)",
			0x40002207: "About Bird's-Eye Telescopes",
			0x40002208: "About Map Gaps",
			0x40002209: "About Map Gaps (Alt 2)",
			0x4000220B: "About Flask of Wondrous Physick",
			0x4000220D: "About Great Runes",
			0x400023B2: "About Dodging",
			0x400023B4: "About Wielding Armaments",
			0x400023B5: "About Pouch",
			0x400023B6: "About Adding to Pouch",
			0x400023BE: "About Spirit Calling Bell",
			0x400023BF: "About Summoning Spirits",
			0x400023C0: "About Multiplayer",
			0x400023C1: "About Cooperative Play",
			0x401EA3DF: "About Item Crafting",
		}
	case "Weapons":
		return map[uint32]string{
			0x02FB1790: "Weathered Straight Sword",
			0x030AA7F0: "Weathered Straight Sword (Alt)",
			0x03199C10: "Weathered Straight Sword (Alt 2)",
			0x0328DE50: "Weathered Straight Sword (Alt 3)",
			0x000138E4: "Torch",
			0x800138E4: "Torch (Handle)",
		}
	}
	return nil
}

// generateGoMap writes the extracted map to a .go file.
func generateGoMap(varName string, data map[uint32]string) {
	// Add hardcoded items
	hardcoded := getHardcodedItems(varName)
	for id, name := range hardcoded {
		data[id] = name
	}

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
