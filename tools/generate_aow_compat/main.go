// Command generate_aow_compat regenerates the Ash of War ↔ weapon compatibility
// data from a regulation.bin CSV dump.
//
// It emits three files from a single source model (no hand-maintained duplicate):
//
//	backend/db/data/aow_compat.go       — AoWCompatMasks, WepTypeToCanMountBit,
//	                                       AoWHeuristicWepTypes, CanMountWepNames
//	backend/db/data/weapon_gem_mount.go — WeaponGemMounts (wepType + gemMountType)
//	frontend/src/data/aowCompat.generated.ts — the Go maps mirrored for the UI
//
// Compatibility model:
//
//	Layer 1 (direct, from regulation.bin):
//	  mask bits 0..35  = EquipParamGem.canMountWep_Dagger .. canMountWep_Torch
//	  mask bits 36..39 = EquipParamGem.reserved_canMountWep bits 0..3
//	    bit 36 (reserved 0) → wepType 88 (Hand-to-Hand / Dryleaf Arts)
//	    bit 37 (reserved 1) → wepType 89 (Perfume Bottles)
//	    bit 38 (reserved 2) → wepType 90 (Dueling / Thrusting Shields)
//	    bit 39 (reserved 3) → wepType 91 (Throwing / Smithscript Blades)
//	  reserved2_canMountWep is expected to be zero on every row; a non-zero value
//	  aborts generation rather than silently dropping data.
//
//	Layer 2 (heuristic, NOT a regulation field): a handful of DLC arts carry no
//	canMountWep / reserved bit at all (Backhand Blades, Light Greatswords, and the
//	native arts of Great Katanas / Beast Claws). For those the compatible wepType
//	is inferred from mountWepTextId grouping + swordArtsParamId (an art that is a
//	weapon's built-in skill is compatible with that weapon type). This is emitted
//	as AoWHeuristicWepTypes, kept separate from the direct mask, and only consulted
//	when the direct mask yields nothing.
//
// Usage:
//
//	go run ./tools/generate_aow_compat \
//	    -gem tmp/regulation-bin-dump/csv/EquipParamGem.csv \
//	    -weapon tmp/regulation-bin-dump/csv/EquipParamWeapon.csv
//
// The tool never reads a hard-coded tmp/ path; both inputs are explicit flags.
package main

import (
	"crypto/sha256"
	"encoding/csv"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// canMountColumns lists the EquipParamGem canMountWep_* columns in bit order.
// Bit N of the mask (0..35) comes from canMountColumns[N].
var canMountColumns = []string{
	"Dagger", "SwordNormal", "SwordLarge", "SwordGigantic",
	"SaberNormal", "SaberLarge", "katana", "SwordDoubleEdge",
	"SwordPierce", "RapierHeavy", "AxeNormal", "AxeLarge",
	"HammerNormal", "HammerLarge", "Flail", "SpearNormal",
	"SpearLarge", "SpearHeavy", "SpearAxe", "Sickle",
	"Knuckle", "Claw", "Whip", "AxhammerLarge",
	"BowSmall", "BowNormal", "BowLarge", "ClossBow",
	"Ballista", "Staff", "Sorcery", "Talisman",
	"ShieldSmall", "ShieldNormal", "ShieldLarge", "Torch",
}

// reservedBitNames labels mask bits 36..39, sourced from reserved_canMountWep
// bits 0..3. These are synthetic names — reserved_canMountWep is a single packed
// column in the paramdef, not four named columns.
var reservedBitNames = []string{
	"reserved0_HandToHandArts",  // bit 36 → wepType 88
	"reserved1_PerfumeBottle",   // bit 37 → wepType 89
	"reserved2_ThrustingShield", // bit 38 → wepType 90
	"reserved3_ThrowingBlade",   // bit 39 → wepType 91
}

// wepTypeToBit maps EquipParamWeapon.wepType → the mask bit the game engine
// checks for that weapon class. This mapping is NOT stored in regulation.bin
// (it is a hard-coded engine enum), so it lives here as the single canonical
// source and is emitted verbatim to both Go and TypeScript.
//
// Note: FromSoft's canMountWep_* column names do not always match a weapon's
// in-game display category (e.g. wepType 25 = Spear checks the column named
// "HammerNormal"). The comment on each generated entry names the real column,
// not a guessed category.
var wepTypeToBit = map[uint16]uint8{
	1:  0,  // Dagger
	3:  1,  // Straight Sword
	5:  2,  // Greatsword
	7:  3,  // Colossal Sword
	9:  8,  // Curved Sword
	11: 9,  // Curved Greatsword
	13: 6,  // Katana
	14: 5,  // Curved Greatsword (alt)
	15: 4,  // Curved Sword (alt)
	16: 7,  // Twinblade
	17: 7,  // Twinblade (alt)
	19: 11, // Greataxe
	21: 13, // Great Hammer
	23: 10, // Axe
	24: 10, // Axe (alt)
	25: 12, // Spear
	28: 14, // Great Spear
	29: 14, // Halberd
	31: 15, // Reaper
	32: 17, // Great Spear (heavy)
	33: 18, // Halberd (alt)
	35: 20, // Fist
	37: 19, // Sickle / Claw
	39: 20, // Whip
	41: 21, // Hand-to-hand / Beast weapons
	43: 22, // Whip (alt)
	50: 23, // Light Bow
	51: 24, // Bow
	52: 25, // Bow (alt)
	53: 26, // Greatbow
	54: 27, // Crossbow (alt)
	55: 28, // Crossbow
	57: 29, // Glintstone Staff
	61: 30, // Sacred Seal
	65: 32, // Small Shield
	66: 33, // Medium Shield
	67: 34, // Greatshield
	69: 34, // Greatshield / Towershield
	87: 35, // Torch (real torch wepType; the game checks canMountWep_Torch)
	88: 36, // Hand-to-Hand Arts (reserved_canMountWep bit 0)
	89: 37, // Perfume Bottles (reserved_canMountWep bit 1)
	90: 38, // Dueling / Thrusting Shields (reserved_canMountWep bit 2)
	91: 39, // Throwing / Smithscript Blades (reserved_canMountWep bit 3)
	94: 6,  // Great Katana — reuses the base katana column (bit 6)
	95: 21, // Beast Claw — reuses the base claw column (bit 21)
	// wepType 92 (Backhand Blades) and 93 (Light Greatswords) have no dedicated
	// regulation bit; they are resolved via AoWHeuristicWepTypes instead.
	// wepType 68 is intentionally absent: no weapon row uses it (real torches are
	// wepType 87), so mapping it would describe a non-existent weapon class.
}

const maskBits = 40 // 0..35 canMountWep, 36..39 reserved_canMountWep

type gem struct {
	rid   uint32
	item  uint32
	sap   int
	mwtid int
	mask  uint64
}

type weapon struct {
	rid     uint32
	wepType uint16
	gm      uint8
	sap     int
}

type result struct {
	gems       []gem
	weapons    []weapon
	masks      map[uint32]uint64   // AoW item ID → mask (only non-zero masks)
	heuristic  map[uint32][]uint16 // AoW item ID → wepTypes (mask==0 DLC arts)
	gemHash    string
	weaponHash string
	aowGo      string
	weaponGo   string
	tsSource   string
}

func main() {
	gemPath := flag.String("gem", "", "path to EquipParamGem.csv")
	weaponPath := flag.String("weapon", "", "path to EquipParamWeapon.csv")
	aowOut := flag.String("aow-out", "backend/db/data/aow_compat.go", "output path for aow_compat.go")
	weaponOut := flag.String("weapon-out", "backend/db/data/weapon_gem_mount.go", "output path for weapon_gem_mount.go")
	tsOut := flag.String("ts-out", "frontend/src/data/aowCompat.generated.ts", "output path for the TypeScript mirror")
	flag.Parse()

	if *gemPath == "" || *weaponPath == "" {
		fatalf("both -gem and -weapon are required")
	}

	res, err := generate(*gemPath, *weaponPath)
	if err != nil {
		fatalf("%v", err)
	}

	write(*aowOut, res.aowGo)
	write(*weaponOut, res.weaponGo)
	write(*tsOut, res.tsSource)

	fmt.Printf("aow_compat: %d masks, %d heuristic entries\n", len(res.masks), len(res.heuristic))
	fmt.Printf("weapon_gem_mount: %d weapons\n", countMountable(res.weapons))
}

// generate is the pure core: it reads both CSVs and returns every generated
// artifact plus the intermediate model, so tests can assert on it directly.
func generate(gemPath, weaponPath string) (*result, error) {
	gems, gemHash, err := readGems(gemPath)
	if err != nil {
		return nil, err
	}
	weapons, weaponHash, err := readWeapons(weaponPath)
	if err != nil {
		return nil, err
	}

	masks := make(map[uint32]uint64)
	for _, g := range gems {
		if g.mask != 0 {
			masks[g.item] = g.mask
		}
	}
	heuristic := computeHeuristic(gems, weapons)

	res := &result{
		gems:       gems,
		weapons:    weapons,
		masks:      masks,
		heuristic:  heuristic,
		gemHash:    gemHash,
		weaponHash: weaponHash,
	}

	aowGo, err := renderAoWGo(res, gemPath, weaponPath)
	if err != nil {
		return nil, err
	}
	weaponGo, err := renderWeaponGo(res, weaponPath)
	if err != nil {
		return nil, err
	}
	res.aowGo = aowGo
	res.weaponGo = weaponGo
	res.tsSource = renderTS(res, gemPath, weaponPath)
	return res, nil
}

// computeHeuristic infers wepType compatibility for DLC arts that carry no
// direct mask bit. It groups mask==0 gems by mountWepTextId, unions their
// swordArtsParamIds, and maps that set to the wepTypes of standard-infusable
// (gemMountType==2) weapons whose default skill is one of those arts.
func computeHeuristic(gems []gem, weapons []weapon) map[uint32][]uint16 {
	sapToWepTypes := make(map[int]map[uint16]bool)
	for _, w := range weapons {
		if w.gm != 2 || w.sap <= 0 || w.rid%100 != 0 {
			continue
		}
		if sapToWepTypes[w.sap] == nil {
			sapToWepTypes[w.sap] = make(map[uint16]bool)
		}
		sapToWepTypes[w.sap][w.wepType] = true
	}

	// mountWepTextId → set of swordArtsParamIds across the whole group.
	groupSAPs := make(map[int]map[int]bool)
	for _, g := range gems {
		if g.mwtid < 0 || g.sap <= 0 {
			continue
		}
		if groupSAPs[g.mwtid] == nil {
			groupSAPs[g.mwtid] = make(map[int]bool)
		}
		groupSAPs[g.mwtid][g.sap] = true
	}

	out := make(map[uint32][]uint16)
	for _, g := range gems {
		if g.mask != 0 || g.mwtid < 0 {
			continue
		}
		wts := make(map[uint16]bool)
		for sap := range groupSAPs[g.mwtid] {
			for wt := range sapToWepTypes[sap] {
				wts[wt] = true
			}
		}
		if len(wts) == 0 {
			continue
		}
		list := make([]uint16, 0, len(wts))
		for wt := range wts {
			list = append(list, wt)
		}
		sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
		out[g.item] = list
	}
	return out
}

func readGems(path string) ([]gem, string, error) {
	header, rows, hash, err := readCSV(path)
	if err != nil {
		return nil, "", err
	}
	required := []string{"Row ID", "swordArtsParamId", "reserved_canMountWep", "reserved2_canMountWep", "mountWepTextId"}
	required = append(required, canMountCols()...)
	if err := requireColumns(path, header, required); err != nil {
		return nil, "", err
	}

	var gems []gem
	seen := make(map[uint32]int) // Row ID → CSV row it first appeared on
	for i, row := range rows {
		rr := rowReader{path: path, csvRow: i + 2, rid: rawRowID(row, header), row: row, header: header}
		if rr.blank() {
			continue // trailing empty record from the CSV dump
		}
		rid, err := rr.reqUint32("Row ID")
		if err != nil {
			return nil, "", err
		}
		if first, dup := seen[rid]; dup {
			return nil, "", rr.err("Row ID", strconv.FormatUint(uint64(rid), 10),
				fmt.Sprintf("duplicate Row ID (already defined on CSV row %d)", first))
		}
		seen[rid] = rr.csvRow

		var mask uint64
		for bit, name := range canMountColumns {
			on, err := rr.reqBit("canMountWep_" + name)
			if err != nil {
				return nil, "", err
			}
			if on {
				mask |= uint64(1) << uint(bit)
			}
		}
		reserved, err := rr.reqInt("reserved_canMountWep")
		if err != nil {
			return nil, "", err
		}
		if reserved < 0 || reserved > 0xF {
			return nil, "", rr.err("reserved_canMountWep", strconv.Itoa(reserved), "out of the supported 0..15 range")
		}
		mask |= uint64(reserved&0xF) << 36
		if err := rr.reqZero("reserved2_canMountWep"); err != nil {
			return nil, "", err
		}
		sap, err := rr.reqInt("swordArtsParamId") // -1 allowed
		if err != nil {
			return nil, "", err
		}
		mwtid, err := rr.reqInt("mountWepTextId") // -1 allowed
		if err != nil {
			return nil, "", err
		}
		gems = append(gems, gem{
			rid:   rid,
			item:  0x80000000 | rid,
			sap:   sap,
			mwtid: mwtid,
			mask:  mask,
		})
	}
	sort.Slice(gems, func(i, j int) bool { return gems[i].rid < gems[j].rid })
	return gems, hash, nil
}

func readWeapons(path string) ([]weapon, string, error) {
	header, rows, hash, err := readCSV(path)
	if err != nil {
		return nil, "", err
	}
	if err := requireColumns(path, header, []string{"Row ID", "wepType", "gemMountType", "swordArtsParamId"}); err != nil {
		return nil, "", err
	}
	var weapons []weapon
	seen := make(map[uint32]int)
	for i, row := range rows {
		rr := rowReader{path: path, csvRow: i + 2, rid: rawRowID(row, header), row: row, header: header}
		if rr.blank() {
			continue
		}
		rid, err := rr.reqUint32("Row ID")
		if err != nil {
			return nil, "", err
		}
		if first, dup := seen[rid]; dup {
			return nil, "", rr.err("Row ID", strconv.FormatUint(uint64(rid), 10),
				fmt.Sprintf("duplicate Row ID (already defined on CSV row %d)", first))
		}
		seen[rid] = rr.csvRow

		wepType, err := rr.reqInt("wepType")
		if err != nil {
			return nil, "", err
		}
		if wepType < 0 || wepType > 0xFFFF {
			return nil, "", rr.err("wepType", strconv.Itoa(wepType), "out of the uint16 range 0..65535")
		}
		gm, err := rr.reqInt("gemMountType")
		if err != nil {
			return nil, "", err
		}
		if gm != 0 && gm != 1 && gm != 2 {
			return nil, "", rr.err("gemMountType", strconv.Itoa(gm), "not a contract value (expected 0, 1 or 2)")
		}
		sap, err := rr.reqInt("swordArtsParamId") // -1 allowed
		if err != nil {
			return nil, "", err
		}
		weapons = append(weapons, weapon{
			rid:     rid,
			wepType: uint16(wepType),
			gm:      uint8(gm),
			sap:     sap,
		})
	}
	sort.Slice(weapons, func(i, j int) bool { return weapons[i].rid < weapons[j].rid })
	return weapons, hash, nil
}

func renderAoWGo(res *result, gemPath, weaponPath string) (string, error) {
	var b strings.Builder
	b.WriteString(header(gemPath, weaponPath, res.gemHash, res.weaponHash))
	b.WriteString("package data\n\n")

	// Masks.
	b.WriteString("// AoWCompatMasks maps an Ash of War item ID to its 40-bit weapon-compatibility\n")
	b.WriteString("// mask, sourced directly from regulation.bin:\n")
	b.WriteString("//   bits 0..35  = EquipParamGem.canMountWep_Dagger .. canMountWep_Torch\n")
	b.WriteString("//   bits 36..39 = EquipParamGem.reserved_canMountWep bits 0..3\n")
	b.WriteString("// AoW item ID = EquipParamGem.Row ID | 0x80000000. Masks equal to 0 are omitted;\n")
	b.WriteString("// such arts (if any) resolve through AoWHeuristicWepTypes instead.\n")
	b.WriteString("var AoWCompatMasks = map[uint32]uint64{\n")
	for _, g := range res.gems {
		if g.mask == 0 {
			continue
		}
		fmt.Fprintf(&b, "\t0x%08X: 0x%010X, // row %d\n", g.item, g.mask, g.rid)
	}
	b.WriteString("}\n\n")

	// Heuristic.
	b.WriteString("// AoWHeuristicWepTypes lists compatible wepTypes for DLC arts that carry NO direct\n")
	b.WriteString("// canMountWep / reserved_canMountWep bit (Backhand Blades, Light Greatswords, and the\n")
	b.WriteString("// native arts of Great Katanas / Beast Claws). This is a HEURISTIC, not a regulation\n")
	b.WriteString("// field: the wepType is inferred from mountWepTextId grouping + swordArtsParamId (the\n")
	b.WriteString("// art is a weapon's built-in skill). It is consulted only after the direct mask check\n")
	b.WriteString("// fails, and never grants a wepType outside this list (fail-closed).\n")
	b.WriteString("var AoWHeuristicWepTypes = map[uint32][]uint16{\n")
	for _, item := range sortedKeys(res.heuristic) {
		b.WriteString(fmt.Sprintf("\t0x%08X: {%s},\n", item, joinUint16(res.heuristic[item])))
	}
	b.WriteString("}\n\n")

	// wepType → bit.
	b.WriteString("// WepTypeToCanMountBit maps EquipParamWeapon.wepType to the AoWCompatMasks bit the\n")
	b.WriteString("// game engine checks for that weapon class. This is an engine enum (not stored in\n")
	b.WriteString("// regulation.bin); the comment on each entry names the real EquipParamGem column,\n")
	b.WriteString("// which does not always match the weapon's in-game display category.\n")
	b.WriteString("var WepTypeToCanMountBit = map[uint16]uint8{\n")
	for _, wt := range sortedWepTypes(wepTypeToBit) {
		bit := wepTypeToBit[wt]
		fmt.Fprintf(&b, "\t%d: %d, // %s\n", wt, bit, bitColumnComment(bit))
	}
	b.WriteString("}\n\n")

	b.WriteString("// CanMountBitsForWepType returns the mask bit(s) to test for a wepType. Every wepType\n")
	b.WriteString("// maps to exactly one direct bit today; the slice return keeps the call site stable.\n")
	b.WriteString("func CanMountBitsForWepType(wepType uint16) ([]uint8, bool) {\n")
	b.WriteString("\tbit, ok := WepTypeToCanMountBit[wepType]\n")
	b.WriteString("\tif !ok {\n\t\treturn nil, false\n\t}\n")
	b.WriteString("\treturn []uint8{bit}, true\n")
	b.WriteString("}\n\n")

	// Column names.
	b.WriteString("// CanMountWepNames names each mask bit (0..39) for diagnostics. Bits 0..35 are the\n")
	b.WriteString("// EquipParamGem canMountWep_* columns; bits 36..39 are reserved_canMountWep bits 0..3.\n")
	b.WriteString("var CanMountWepNames = []string{\n")
	for i := 0; i < maskBits; i++ {
		fmt.Fprintf(&b, "\t%q,\n", bitName(uint8(i)))
	}
	b.WriteString("}\n")

	return gofmt(b.String())
}

func renderWeaponGo(res *result, weaponPath string) (string, error) {
	var b strings.Builder
	b.WriteString(headerSingle(weaponPath, res.weaponHash))
	b.WriteString("package data\n\n")
	b.WriteString("// WeaponGemMount holds AoW-mount metadata for a weapon base item ID.\n")
	b.WriteString("type WeaponGemMount struct {\n")
	b.WriteString("\tWepType      uint16 // EquipParamWeapon.wepType (weapon category integer)\n")
	b.WriteString("\tGemMountType uint8  // 0=none, 1=special/somber, 2=standard infusable\n")
	b.WriteString("}\n\n")
	b.WriteString("// WeaponGemMounts maps a weapon base item ID (Row ID, upgrade 0) to its mount data.\n")
	b.WriteString("// Only weapons with gemMountType != 0 are included. Infused/upgraded item IDs resolve\n")
	b.WriteString("// to their base via db.GetItemDataFuzzy before lookup.\n")
	b.WriteString("var WeaponGemMounts = map[uint32]WeaponGemMount{\n")
	for _, w := range res.weapons {
		if w.rid%100 != 0 || w.gm == 0 {
			continue
		}
		fmt.Fprintf(&b, "\t0x%08X: {WepType: %d, GemMountType: %d},\n", w.rid, w.wepType, w.gm)
	}
	b.WriteString("}\n")
	return gofmt(b.String())
}

func renderTS(res *result, gemPath, weaponPath string) string {
	var b strings.Builder
	b.WriteString("// Code generated by tools/generate_aow_compat; DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// Sources: %s (sha256 %s), %s (sha256 %s).\n",
		gemPath, res.gemHash, weaponPath, res.weaponHash)
	b.WriteString("//\n")
	b.WriteString("// Single source of truth shared with backend/db/data/aow_compat.go. Bits 0..35 come\n")
	b.WriteString("// from EquipParamGem.canMountWep_*, bits 36..39 from reserved_canMountWep. Backhand\n")
	b.WriteString("// Blades / Light Greatswords and the native arts of Great Katanas / Beast Claws carry\n")
	b.WriteString("// no direct bit and are resolved via AOW_HEURISTIC_WEPTYPES (heuristic, fail-closed).\n\n")

	b.WriteString("// Maps EquipParamWeapon.wepType to AoWCompatBitmask bit positions.\n")
	b.WriteString("export const WEP_TYPE_TO_BITS: Record<number, number[]> = {\n")
	for _, wt := range sortedWepTypes(wepTypeToBit) {
		fmt.Fprintf(&b, "    %d: [%d],\n", wt, wepTypeToBit[wt])
	}
	b.WriteString("};\n\n")

	b.WriteString("// Maps an Ash of War item ID to compatible wepTypes when it has no direct mask bit.\n")
	b.WriteString("export const AOW_HEURISTIC_WEPTYPES: Record<number, number[]> = {\n")
	for _, item := range sortedKeys(res.heuristic) {
		fmt.Fprintf(&b, "    0x%08X: [%s],\n", item, joinUint16(res.heuristic[item]))
	}
	b.WriteString("};\n")
	return b.String()
}

// --- rendering helpers ---

func header(gemPath, weaponPath, gemHash, weaponHash string) string {
	return fmt.Sprintf(
		"// Code generated by tools/generate_aow_compat; DO NOT EDIT.\n"+
			"//\n"+
			"// Sources:\n"+
			"//   %s (sha256 %s)\n"+
			"//   %s (sha256 %s)\n"+
			"//\n"+
			"// Direct data: mask bits 0..35 (canMountWep_*) and 36..39 (reserved_canMountWep).\n"+
			"// Heuristic:   AoWHeuristicWepTypes (mountWepTextId + swordArtsParamId inference).\n\n",
		gemPath, gemHash, weaponPath, weaponHash)
}

func headerSingle(path, hash string) string {
	return fmt.Sprintf(
		"// Code generated by tools/generate_aow_compat; DO NOT EDIT.\n"+
			"//\n"+
			"// Source: %s (sha256 %s)\n\n",
		path, hash)
}

func bitName(bit uint8) string {
	if int(bit) < len(canMountColumns) {
		return canMountColumns[bit]
	}
	i := int(bit) - len(canMountColumns)
	if i < len(reservedBitNames) {
		return reservedBitNames[i]
	}
	return fmt.Sprintf("bit%d", bit)
}

func bitColumnComment(bit uint8) string {
	if int(bit) < len(canMountColumns) {
		return fmt.Sprintf("canMountWep_%s (bit %d)", canMountColumns[bit], bit)
	}
	i := int(bit) - len(canMountColumns)
	return fmt.Sprintf("reserved_canMountWep bit %d (mask bit %d)", i, bit)
}

func canMountCols() []string {
	cols := make([]string, len(canMountColumns))
	for i, n := range canMountColumns {
		cols[i] = "canMountWep_" + n
	}
	return cols
}

// --- CSV / parsing helpers ---

func readCSV(path string) (map[string]int, [][]string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, "", fmt.Errorf("open %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", sum)

	r := csv.NewReader(strings.NewReader(string(data)))
	r.Comma = ';'
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	headerRow, err := r.Read()
	if err != nil {
		return nil, nil, "", fmt.Errorf("read header %s: %w", path, err)
	}
	header := make(map[string]int, len(headerRow))
	for i, name := range headerRow {
		header[strings.TrimSpace(name)] = i
	}

	var rows [][]string
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, "", fmt.Errorf("read %s: %w", path, err)
		}
		rows = append(rows, row)
	}
	return header, rows, hash, nil
}

func requireColumns(path string, header map[string]int, names []string) error {
	var missing []string
	for _, n := range names {
		if _, ok := header[n]; !ok {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s: missing required column(s): %s", path, strings.Join(missing, ", "))
	}
	return nil
}

// rowReader parses one CSV record with fail-closed, strictly-typed field access.
// Every required field must be present and well-formed; a malformed or truncated
// record aborts generation with an error naming file, CSV row, Row ID, field and
// value, rather than silently substituting a default (which would emit a
// syntactically valid but incomplete SSOT).
type rowReader struct {
	path   string
	csvRow int    // 1-based CSV line (header is row 1, first data row is row 2)
	rid    string // raw Row ID text, for diagnostics before it is parsed
	row    []string
	header map[string]int
}

func rawRowID(row []string, header map[string]int) string {
	idx, ok := header["Row ID"]
	if !ok || idx >= len(row) {
		return "?"
	}
	return strings.TrimSpace(row[idx])
}

// blank reports whether every cell in the record is empty — a trailing empty
// line from the dump, which may be skipped.
func (rr rowReader) blank() bool {
	for _, c := range rr.row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func (rr rowReader) err(field, value, reason string) error {
	return fmt.Errorf("%s: CSV row %d (Row ID %s): field %q value %q: %s",
		rr.path, rr.csvRow, rr.rid, field, value, reason)
}

// raw returns a required column's trimmed value, or an error if the record is
// too short to contain it. The column is guaranteed present in the header by
// requireColumns; a missing index therefore means a truncated record.
func (rr rowReader) raw(field string) (string, error) {
	idx := rr.header[field]
	if idx >= len(rr.row) {
		return "", rr.err(field, "", fmt.Sprintf("truncated record: only %d columns, required column absent", len(rr.row)))
	}
	return strings.TrimSpace(rr.row[idx]), nil
}

func (rr rowReader) reqUint32(field string) (uint32, error) {
	s, err := rr.raw(field)
	if err != nil {
		return 0, err
	}
	v, e := strconv.ParseUint(s, 10, 32)
	if e != nil {
		return 0, rr.err(field, s, "not a valid uint32")
	}
	return uint32(v), nil
}

func (rr rowReader) reqInt(field string) (int, error) {
	s, err := rr.raw(field)
	if err != nil {
		return 0, err
	}
	v, e := strconv.Atoi(s)
	if e != nil {
		return 0, rr.err(field, s, "not a valid integer")
	}
	return v, nil
}

// reqBit parses a canMountWep_* flag, which must be exactly "0" or "1".
func (rr rowReader) reqBit(field string) (bool, error) {
	s, err := rr.raw(field)
	if err != nil {
		return false, err
	}
	switch s {
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, rr.err(field, s, "must be exactly 0 or 1")
	}
}

// reqZero validates a field that must be zero. An empty cell is accepted as the
// unset reserved-padding convention used by the regulation dump: reserved2_canMountWep
// is blank on every row of the current EquipParamGem export, where an empty column is
// the canonical encoding of zero. Any present non-zero integer, or any non-integer
// text, aborts generation.
func (rr rowReader) reqZero(field string) error {
	s, err := rr.raw(field)
	if err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	v, e := strconv.Atoi(s)
	if e != nil {
		return rr.err(field, s, "not a valid integer")
	}
	if v != 0 {
		return rr.err(field, s, "is non-zero; this generator does not model it — add explicit handling before regenerating")
	}
	return nil
}

func sortedKeys(m map[uint32][]uint16) []uint32 {
	keys := make([]uint32, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func sortedWepTypes(m map[uint16]uint8) []uint16 {
	keys := make([]uint16, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func joinUint16(v []uint16) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = strconv.Itoa(int(x))
	}
	return strings.Join(parts, ", ")
}

func countMountable(weapons []weapon) int {
	n := 0
	for _, w := range weapons {
		if w.rid%100 == 0 && w.gm != 0 {
			n++
		}
	}
	return n
}

func gofmt(src string) (string, error) {
	out, err := format.Source([]byte(src))
	if err != nil {
		return "", fmt.Errorf("gofmt: %w\n%s", err, src)
	}
	return string(out), nil
}

func write(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fatalf("write %s: %v", path, err)
	}
}

func fatalf(f string, args ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", args...)
	os.Exit(1)
}
