package main

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ERDB data directory — extract 1.10.0.zip here first
const erdbDir = "tmp/erdb/1.10.0"
const outputFile = "backend/db/data/descriptions.go"

// FMG XML structures
type FMGFile struct {
	Entries []FMGEntry `xml:"entries>text"`
}

type FMGEntry struct {
	ID   uint32 `xml:"id,attr"`
	Text string `xml:",chardata"`
}

// Category prefix mapping: ERDB uses raw param IDs, our DB adds a prefix byte.
// weapon param IDs → 0x00000000, protector → 0x10000000, accessory → 0x20000000, goods → 0x40000000
var categoryPrefixes = map[string]uint32{
	"weapon":    0x00000000,
	"protector": 0x10000000,
	"accessory": 0x20000000,
	"goods":     0x40000000,
}

// descEntry holds parsed data before code generation
type descEntry struct {
	ID          uint32
	Description string
	Weight      float64
	Weapon      *weaponStats
	Armor       *armorStats
	Spell       *spellStats
}

type weaponStats struct {
	Weight     float64
	PhysDamage uint32
	MagDamage  uint32
	FireDamage uint32
	LitDamage  uint32
	HolyDamage uint32
	ScaleStr   uint32
	ScaleDex   uint32
	ScaleInt   uint32
	ScaleFai   uint32
	ReqStr     uint32
	ReqDex     uint32
	ReqInt     uint32
	ReqFai     uint32
	ReqArc     uint32
}

type armorStats struct {
	Weight     float64
	Physical   float64
	Strike     float64
	Slash      float64
	Pierce     float64
	Magic      float64
	Fire       float64
	Lightning  float64
	Holy       float64
	Immunity   uint32
	Robustness uint32
	Focus      uint32
	Vitality   uint32
	Poise      float64
}

type spellStats struct {
	FPCost uint32
	Slots  uint32
	ReqInt uint32
	ReqFai uint32
	ReqArc uint32
}

func main() {
	fmt.Println("Importing ERDB data from", erdbDir)

	entries := make(map[uint32]*descEntry)

	// 1. Parse descriptions from FMG XML files
	fmgMapping := map[string]uint32{
		"WeaponCaption.fmg.xml":    categoryPrefixes["weapon"],
		"ProtectorCaption.fmg.xml": categoryPrefixes["protector"],
		"AccessoryCaption.fmg.xml": categoryPrefixes["accessory"],
		"GoodsCaption.fmg.xml":     categoryPrefixes["goods"],
	}

	for file, prefix := range fmgMapping {
		parseFMGDescriptions(filepath.Join(erdbDir, file), prefix, entries)
	}
	fmt.Printf("  Descriptions parsed: %d entries\n", len(entries))

	// 2. Parse weapon stats from CSV
	parseWeaponCSV(filepath.Join(erdbDir, "EquipParamWeapon.csv"), entries)

	// 3. Parse armor stats from CSV
	parseArmorCSV(filepath.Join(erdbDir, "EquipParamProtector.csv"), entries)

	// 4. Parse spell stats from CSV
	parseSpellCSV(filepath.Join(erdbDir, "Magic.csv"), entries)

	// 5. Parse accessory weight from CSV
	parseAccessoryCSV(filepath.Join(erdbDir, "EquipParamAccessory.csv"), entries)

	// 6. Generate Go source file
	generateGoFile(entries)

	fmt.Println("Done!")
}

func parseFMGDescriptions(path string, prefix uint32, entries map[uint32]*descEntry) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("  WARNING: cannot read %s: %v\n", path, err)
		return
	}

	var fmg FMGFile
	if err := xml.Unmarshal(data, &fmg); err != nil {
		fmt.Printf("  WARNING: cannot parse %s: %v\n", path, err)
		return
	}

	count := 0
	for _, e := range fmg.Entries {
		text := strings.TrimSpace(e.Text)
		if text == "" || text == "%null%" {
			continue
		}
		id := e.ID | prefix
		if _, exists := entries[id]; !exists {
			entries[id] = &descEntry{ID: id}
		}
		entries[id].Description = text
		count++
	}
	fmt.Printf("  %s: %d descriptions\n", filepath.Base(path), count)
}

func parseWeaponCSV(path string, entries map[uint32]*descEntry) {
	records := readCSV(path)
	if records == nil {
		return
	}

	header := records[0]
	colIdx := buildColumnIndex(header)

	count := 0
	for _, row := range records[1:] {
		rowID := parseUint32(row[0])
		if rowID == 0 {
			continue
		}

		id := rowID | categoryPrefixes["weapon"]

		weight := parseFloat64(getCol(row, colIdx, "weight"))
		physDmg := parseUint32(getCol(row, colIdx, "attackBasePhysics"))
		magDmg := parseUint32(getCol(row, colIdx, "attackBaseMagic"))
		fireDmg := parseUint32(getCol(row, colIdx, "attackBaseFire"))
		litDmg := parseUint32(getCol(row, colIdx, "attackBaseThunder"))
		holyDmg := parseUint32(getCol(row, colIdx, "attackBaseDark"))
		scaleStr := parseUint32(getCol(row, colIdx, "correctStrength"))
		scaleDex := parseUint32(getCol(row, colIdx, "correctAgility"))
		scaleInt := parseUint32(getCol(row, colIdx, "correctMagic"))
		scaleFai := parseUint32(getCol(row, colIdx, "correctFaith"))
		reqStr := parseUint32(getCol(row, colIdx, "properStrength"))
		reqDex := parseUint32(getCol(row, colIdx, "properAgility"))
		reqInt := parseUint32(getCol(row, colIdx, "properMagic"))
		reqFai := parseUint32(getCol(row, colIdx, "properFaith"))
		reqArc := parseUint32(getCol(row, colIdx, "properLuck"))

		// Skip dummy/disabled entries (no damage, no scaling, no weight)
		if physDmg == 0 && magDmg == 0 && fireDmg == 0 && litDmg == 0 && holyDmg == 0 && weight == 0 {
			continue
		}

		if _, exists := entries[id]; !exists {
			entries[id] = &descEntry{ID: id}
		}
		entries[id].Weapon = &weaponStats{
			Weight: weight, PhysDamage: physDmg, MagDamage: magDmg,
			FireDamage: fireDmg, LitDamage: litDmg, HolyDamage: holyDmg,
			ScaleStr: scaleStr, ScaleDex: scaleDex, ScaleInt: scaleInt, ScaleFai: scaleFai,
			ReqStr: reqStr, ReqDex: reqDex, ReqInt: reqInt, ReqFai: reqFai, ReqArc: reqArc,
		}
		count++
	}
	fmt.Printf("  EquipParamWeapon.csv: %d weapon stats\n", count)
}

func parseArmorCSV(path string, entries map[uint32]*descEntry) {
	records := readCSV(path)
	if records == nil {
		return
	}

	header := records[0]
	colIdx := buildColumnIndex(header)

	count := 0
	for _, row := range records[1:] {
		rowID := parseUint32(row[0])
		if rowID == 0 {
			continue
		}

		id := rowID | categoryPrefixes["protector"]

		weight := parseFloat64(getCol(row, colIdx, "weight"))
		physical := parseFloat64(getCol(row, colIdx, "neutralDamageCutRate"))
		slash := parseFloat64(getCol(row, colIdx, "slashDamageCutRate"))
		strike := parseFloat64(getCol(row, colIdx, "blowDamageCutRate"))
		pierce := parseFloat64(getCol(row, colIdx, "thrustDamageCutRate"))
		magic := parseFloat64(getCol(row, colIdx, "magicDamageCutRate"))
		fire := parseFloat64(getCol(row, colIdx, "fireDamageCutRate"))
		lightning := parseFloat64(getCol(row, colIdx, "thunderDamageCutRate"))
		holy := parseFloat64(getCol(row, colIdx, "darkDamageCutRate"))
		immunity := parseUint32(getCol(row, colIdx, "resistPoison"))
		robustness := parseUint32(getCol(row, colIdx, "resistBlood"))
		focus := parseUint32(getCol(row, colIdx, "resistMadness"))
		vitality := parseUint32(getCol(row, colIdx, "resistCurse"))
		poise := parseFloat64(getCol(row, colIdx, "toughnessDamageCutRate"))

		// Skip dummy entries
		if weight == 0 && physical == 0 && immunity == 0 {
			continue
		}

		// Convert DamageCutRate to damage negation percentage:
		// Game stores as multiplier (e.g. 0.956 = 4.4% negation)
		// Display as negation% = (1 - cutRate) * 100
		physical = roundTo2((1 - physical) * 100)
		slash = roundTo2((1 - slash) * 100)
		strike = roundTo2((1 - strike) * 100)
		pierce = roundTo2((1 - pierce) * 100)
		magic = roundTo2((1 - magic) * 100)
		fire = roundTo2((1 - fire) * 100)
		lightning = roundTo2((1 - lightning) * 100)
		holy = roundTo2((1 - holy) * 100)
		poise = roundTo2((1 - poise) * 100)

		if _, exists := entries[id]; !exists {
			entries[id] = &descEntry{ID: id}
		}
		entries[id].Armor = &armorStats{
			Weight: weight, Physical: physical, Strike: strike, Slash: slash, Pierce: pierce,
			Magic: magic, Fire: fire, Lightning: lightning, Holy: holy,
			Immunity: immunity, Robustness: robustness, Focus: focus, Vitality: vitality, Poise: poise,
		}
		count++
	}
	fmt.Printf("  EquipParamProtector.csv: %d armor stats\n", count)
}

func parseSpellCSV(path string, entries map[uint32]*descEntry) {
	records := readCSV(path)
	if records == nil {
		return
	}

	header := records[0]
	colIdx := buildColumnIndex(header)

	count := 0
	for _, row := range records[1:] {
		rowID := parseUint32(row[0])
		if rowID == 0 {
			continue
		}

		// Spells use goods prefix (0x40) — same as items in save file
		id := rowID | categoryPrefixes["goods"]

		fpCost := parseUint32(getCol(row, colIdx, "mp"))
		slots := parseUint32(getCol(row, colIdx, "slotLength"))
		reqInt := parseUint32(getCol(row, colIdx, "requirementIntellect"))
		reqFai := parseUint32(getCol(row, colIdx, "requirementFaith"))
		reqArc := parseUint32(getCol(row, colIdx, "requirementLuck"))

		if fpCost == 0 && slots == 0 {
			continue
		}

		if _, exists := entries[id]; !exists {
			entries[id] = &descEntry{ID: id}
		}
		entries[id].Spell = &spellStats{
			FPCost: fpCost, Slots: slots, ReqInt: reqInt, ReqFai: reqFai, ReqArc: reqArc,
		}
		count++
	}
	fmt.Printf("  Magic.csv: %d spell stats\n", count)
}

func parseAccessoryCSV(path string, entries map[uint32]*descEntry) {
	records := readCSV(path)
	if records == nil {
		return
	}

	header := records[0]
	colIdx := buildColumnIndex(header)

	count := 0
	for _, row := range records[1:] {
		rowID := parseUint32(row[0])
		if rowID == 0 {
			continue
		}

		id := rowID | categoryPrefixes["accessory"]
		weight := parseFloat64(getCol(row, colIdx, "weight"))

		if weight == 0 {
			continue
		}

		if _, exists := entries[id]; !exists {
			entries[id] = &descEntry{ID: id}
		}
		entries[id].Weight = weight
		count++
	}
	fmt.Printf("  EquipParamAccessory.csv: %d accessory weights\n", count)
}

func generateGoFile(entries map[uint32]*descEntry) {
	// Sort IDs for deterministic output
	ids := make([]uint32, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var sb strings.Builder
	sb.WriteString("package data\n\n")
	sb.WriteString("// Code generated by scripts/import_erdb.go — DO NOT EDIT.\n\n")
	sb.WriteString("func init() {\n")
	sb.WriteString("\tDescriptions = map[uint32]ItemDescription{\n")

	written := 0
	for _, id := range ids {
		e := entries[id]
		// Skip entries with no useful data
		if e.Description == "" && e.Weapon == nil && e.Armor == nil && e.Spell == nil && e.Weight == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("\t\t0x%08X: {", id))

		parts := []string{}
		if e.Description != "" {
			parts = append(parts, fmt.Sprintf("Description: %s", goString(e.Description)))
		}
		if e.Weight != 0 {
			parts = append(parts, fmt.Sprintf("Weight: %s", formatFloat(e.Weight)))
		}
		if e.Weapon != nil {
			w := e.Weapon
			parts = append(parts, fmt.Sprintf("Weapon: &WeaponStats{Weight: %s, PhysDamage: %d, MagDamage: %d, FireDamage: %d, LitDamage: %d, HolyDamage: %d, ScaleStr: %d, ScaleDex: %d, ScaleInt: %d, ScaleFai: %d, ReqStr: %d, ReqDex: %d, ReqInt: %d, ReqFai: %d, ReqArc: %d}",
				formatFloat(w.Weight), w.PhysDamage, w.MagDamage, w.FireDamage, w.LitDamage, w.HolyDamage,
				w.ScaleStr, w.ScaleDex, w.ScaleInt, w.ScaleFai, w.ReqStr, w.ReqDex, w.ReqInt, w.ReqFai, w.ReqArc))
		}
		if e.Armor != nil {
			a := e.Armor
			parts = append(parts, fmt.Sprintf("Armor: &ArmorStats{Weight: %s, Physical: %s, Strike: %s, Slash: %s, Pierce: %s, Magic: %s, Fire: %s, Lightning: %s, Holy: %s, Immunity: %d, Robustness: %d, Focus: %d, Vitality: %d, Poise: %s}",
				formatFloat(a.Weight), formatFloat(a.Physical), formatFloat(a.Strike), formatFloat(a.Slash), formatFloat(a.Pierce),
				formatFloat(a.Magic), formatFloat(a.Fire), formatFloat(a.Lightning), formatFloat(a.Holy),
				a.Immunity, a.Robustness, a.Focus, a.Vitality, formatFloat(a.Poise)))
		}
		if e.Spell != nil {
			s := e.Spell
			parts = append(parts, fmt.Sprintf("Spell: &SpellStats{FPCost: %d, Slots: %d, ReqInt: %d, ReqFai: %d, ReqArc: %d}",
				s.FPCost, s.Slots, s.ReqInt, s.ReqFai, s.ReqArc))
		}

		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString("},\n")
		written++
	}

	sb.WriteString("\t}\n")
	sb.WriteString("}\n")

	if err := os.WriteFile(outputFile, []byte(sb.String()), 0644); err != nil {
		fmt.Printf("ERROR: cannot write %s: %v\n", outputFile, err)
		os.Exit(1)
	}
	fmt.Printf("  Generated %s: %d entries\n", outputFile, written)
}

// --- Helpers ---

func readCSV(path string) [][]string {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("  WARNING: cannot open %s: %v\n", path, err)
		return nil
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.LazyQuotes = true
	r.FieldsPerRecord = -1 // ERDB CSVs have inconsistent field counts

	records, err := r.ReadAll()
	if err != nil {
		fmt.Printf("  WARNING: cannot parse %s: %v\n", path, err)
		return nil
	}
	return records
}

func buildColumnIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, col := range header {
		// Strip BOM from first column
		col = strings.TrimPrefix(col, "\ufeff")
		idx[strings.TrimSpace(col)] = i
	}
	return idx
}

func getCol(row []string, colIdx map[string]int, name string) string {
	if i, ok := colIdx[name]; ok && i < len(row) {
		return row[i]
	}
	return ""
}

func parseUint32(s string) uint32 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-1" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	if v < 0 {
		return 0
	}
	return uint32(v)
}

func parseFloat64(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func roundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}

func formatFloat(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%.1f", v)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func goString(s string) string {
	// Escape for Go string literal
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", `\t`)
	return `"` + s + `"`
}
