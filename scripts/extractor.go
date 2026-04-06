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

const PY_PROJECT_PATH = "tmp/repos/Elden-Ring-Save-Editor/src/Resources/Json/"
const RUST_PROJECT_PATH = "tmp/repos/ER-Save-Editor/src/db/"

type ItemData struct {
	Name         string
	MaxInventory uint32
	MaxStorage   uint32
	MaxUpgrade   uint32
	IconPath     string
}

func main() {
	fmt.Println("🚀 Starting data extraction with strict limits...")

	// Ensure target directory exists
	os.MkdirAll("backend/db/data", 0755)

	rebuildFromJSON("weapons.json", "Weapons")
	rebuildFromJSON("armor.json", "Armors")
	rebuildFromJSON("goods.json", "Items")
	rebuildFromJSON("talisman.json", "Talismans")
	rebuildFromJSON("aow.json", "Aows")
	extractGracesFromJSON("graces.json", "Graces")

	extractStats("stats.rs")
	extractClasses("classes.rs")
	extractEventFlags("event_flags.rs")

	fmt.Println("✅ Data extraction complete!")
}

func rebuildFromJSON(filename, varName string) {
	inputPath := PY_PROJECT_PATH + filename
	file, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("⚠️ Error opening %s: %v\n", inputPath, err)
		return
	}
	defer file.Close()

	var rawData map[string]string
	if err := json.NewDecoder(file).Decode(&rawData); err != nil {
		fmt.Printf("⚠️ Error decoding JSON from %s: %v\n", filename, err)
		return
	}

	data := make(map[uint32]ItemData)
	for name, idHexStr := range rawData {
		byteStr := strings.ReplaceAll(idHexStr, " ", "")
		byteSlice, err := hex.DecodeString(byteStr)
		if err != nil {
			continue
		}

		var rawID uint32
		if len(byteSlice) == 4 {
			rawID = binary.LittleEndian.Uint32(byteSlice)
		} else {
			padded := make([]byte, 4)
			copy(padded, byteSlice)
			rawID = binary.LittleEndian.Uint32(padded)
		}

		baseID := rawID & 0x0FFFFFFF

		// Determine limits
		inv, storage, upgrade := getLimits(name, varName, baseID)

		item := ItemData{
			Name:         name,
			MaxInventory: inv,
			MaxStorage:   storage,
			MaxUpgrade:   upgrade,
			IconPath:     "",
		}

		// Map based on category
		switch varName {
		case "Weapons":
			data[baseID] = item
			data[baseID|0x80000000] = item
		case "Armors":
			data[baseID|0x10000000] = item
			data[baseID|0x90000000] = item
		case "Talismans":
			data[baseID|0x20000000] = item
			data[baseID|0xA0000000] = item
		case "Items":
			data[baseID|0x40000000] = item
			data[baseID|0xB0000000] = item
		case "Aows":
			data[baseID|0x80000000] = item
			data[baseID|0xC0000000] = item
		}
	}

	addHardcoded(varName, data)
	writeGoFile(varName, data)
}

func getLimits(name, category string, id uint32) (inv, storage, upgrade uint32) {
	inv, storage, upgrade = 1, 1, 0

	switch category {
	case "Weapons":
		if isAmmo(name) {
			inv, storage = 99, 600
			if isGreatAmmo(name) {
				inv = 30
			}
		} else {
			upgrade = 25
			// Heuristic for unique weapons could be added here
		}
	case "Armors", "Talismans":
		inv, storage = 1, 1
	case "Aows":
		if name == "Lost Ashes of War" {
			inv, storage = 99, 600
		} else {
			inv, storage = 1, 1
		}
	case "Items":
		inv, storage = 99, 600
		if isMaterial(name) {
			inv, storage = 999, 999
		} else if isKeyItem(name) || isUniqueTool(name) {
			inv, storage = 1, 0
		} else if isSpiritAsh(name) {
			inv, storage, upgrade = 1, 1, 10
		}

		switch {
		case strings.Contains(name, "Starlight Shards"):
			inv = 10
		case strings.Contains(name, "Raw Meat Dumpling"):
			inv = 3
		case strings.Contains(name, "Kukri"):
			inv = 30
		case strings.Contains(name, "Fan Daggers") || strings.Contains(name, "Throwing Dagger") || strings.Contains(name, "Dart"):
			inv = 40
		case strings.Contains(name, "Ritual Pot") || strings.Contains(name, "Hefty") || strings.Contains(name, "Perfume"):
			inv = 10
		case strings.Contains(name, "Pot"):
			inv = 20
		case name == "Memory of Grace" || name == "Vision of Grace":
			inv, storage = 1, 0
		}
	}
	return
}

func isAmmo(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "arrow") || strings.Contains(n, "bolt") || strings.Contains(n, "harpoon")
}

func isGreatAmmo(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "great arrow") || strings.Contains(n, "ballista bolt") || strings.Contains(n, "greatbolt") || strings.Contains(n, "harpoon")
}

func isMaterial(name string) bool {
	n := strings.ToLower(name)
	if strings.Contains(n, "meat dumpling") {
		return false
	}
	materials := []string{"flower", "leaf", "root", "moss", "mushroom", "butterfly", "beetle", "fragment", "stone", "string", "meat", "egg", "feather", "bone", "liver", "fruit", "resin"}
	for _, m := range materials {
		if strings.Contains(n, m) {
			if strings.Contains(n, "pickled") || strings.Contains(n, "pot") || strings.Contains(n, "grease") {
				return false
			}
			return true
		}
	}
	return false
}

func isKeyItem(name string) bool {
	n := strings.ToLower(name)
	keys := []string{"key", "map", "letter", "scroll", "prayerbook", "cookbook", "bell bearing", "crystal tear", "great rune", "relic", "remembrance"}
	for _, k := range keys {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

func isUniqueTool(name string) bool {
	unique := []string{"Whistle", "Finger", "Effigy", "Severer", "Ring", "Physick", "Telescope", "Lantern", "Veil", "Shackle"}
	for _, u := range unique {
		if strings.Contains(name, u) {
			return true
		}
	}
	return false
}

func isSpiritAsh(name string) bool {
	return strings.Contains(name, "Ashes") && !strings.Contains(name, "War")
}

func addHardcoded(varName string, data map[uint32]ItemData) {
	switch varName {
	case "Armors":
		data[0x10002710] = ItemData{Name: "Naked (Head)", MaxInventory: 1, MaxStorage: 1}
		data[0x10002774] = ItemData{Name: "Naked (Body)", MaxInventory: 1, MaxStorage: 1}
		data[0x100027D8] = ItemData{Name: "Naked (Arms)", MaxInventory: 1, MaxStorage: 1}
		data[0x1000283C] = ItemData{Name: "Naked (Legs)", MaxInventory: 1, MaxStorage: 1}
	case "Items":
		data[0x400000A6] = ItemData{Name: "Vision of Grace", MaxInventory: 1, MaxStorage: 0}
	}
}

func writeGoFile(varName string, data map[uint32]ItemData) {
	outputPath := "backend/db/data/" + strings.ToLower(varName) + ".go"
	out, _ := os.Create(outputPath)
	defer out.Close()

	fmt.Fprintf(out, "package data\n\nvar %s = map[uint32]ItemData{\n", varName)

	keys := make([]uint32, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	for _, id := range keys {
		item := data[id]
		name := strings.ReplaceAll(item.Name, "\"", "\\\"")
		fmt.Fprintf(out, "\t0x%08X: {Name: \"%s\", MaxInventory: %d, MaxStorage: %d, MaxUpgrade: %d, IconPath: \"%s\"},\n",
			id, name, item.MaxInventory, item.MaxStorage, item.MaxUpgrade, item.IconPath)
	}
	fmt.Fprintln(out, "}")
}

type GraceJSON struct {
	GraceName string `json:"grace_name"`
	MapID     uint32 `json:"map_id"`
}

func extractGracesFromJSON(filename, varName string) {
	inputPath := PY_PROJECT_PATH + filename
	file, err := os.Open(inputPath)
	if err != nil {
		return
	}
	defer file.Close()

	var data []GraceJSON
	json.NewDecoder(file).Decode(&data)

	graces := make(map[uint32]string)
	for _, g := range data {
		graces[g.MapID] = g.GraceName
	}

	outputPath := "backend/db/data/graces.go"
	out, _ := os.Create(outputPath)
	defer out.Close()

	fmt.Fprintf(out, "package data\n\nvar Graces = map[uint32]string{\n")
	keys := make([]uint32, 0, len(graces))
	for k := range graces {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, id := range keys {
		fmt.Fprintf(out, "\t0x%08X: \"%s\",\n", id, graces[id])
	}
	fmt.Fprintln(out, "}")
}

func extractStats(filename string) {
	inputPath := RUST_PROJECT_PATH + filename
	content, _ := os.ReadFile(inputPath)
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
	inputPath := RUST_PROJECT_PATH + filename
	content, _ := os.ReadFile(inputPath)
	outputPath := "backend/db/data/classes.go"
	out, _ := os.Create(outputPath)
	defer out.Close()
	fmt.Fprintf(out, "package data\n\ntype StarterStats struct {\n\tLevel, Vigor, Mind, Endurance, Strength, Dexterity, Intelligence, Faith, Arcane uint32\n}\n\n")
	fmt.Fprintf(out, "var StarterClasses = map[uint8]StarterStats{\n")
	re := regexp.MustCompile(`ArcheType::(\w+),\s*Stats\{([\s\S]*?)\}`)
	matches := re.FindAllStringSubmatch(string(content), -1)
	archeTypeIDs := map[string]uint8{"Vagabond": 0, "Warrior": 1, "Hero": 2, "Bandit": 3, "Astrologer": 4, "Prophet": 5, "Confessor": 6, "Samurai": 7, "Prisoner": 8, "Wretch": 9}
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
				stats[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
		fmt.Fprintf(out, "\t%d: {Level: %s, Vigor: %s, Mind: %s, Endurance: %s, Strength: %s, Dexterity: %s, Intelligence: %s, Faith: %s, Arcane: %s}, // %s\n", id, stats["level"], stats["vigor"], stats["mind"], stats["endurance"], stats["strength"], stats["dexterity"], stats["intelligence"], stats["faith"], stats["arcane"], name)
	}
	fmt.Fprintln(out, "}")
}

func extractEventFlags(filename string) {
	inputPath := RUST_PROJECT_PATH + filename
	content, _ := os.ReadFile(inputPath)
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
}
