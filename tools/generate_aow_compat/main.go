// Command generate_aow_compat regenerates the Ash of War ↔ weapon compatibility
// data from a regulation.bin CSV dump.
//
// It emits three files from a single source model (no hand-maintained duplicate):
//
//	backend/db/data/aow_compat.go       — AoWCompatMasks, WepTypeToCanMountBit,
//	                                       AoWHeuristicWepTypes, CanMountWepNames
//	backend/db/data/weapon_gem_mount.go — WeaponGemMounts (wepType + AoW/affinity gates)
//	frontend/src/data/aowCompat.generated.ts — the Go maps mirrored for the UI
//
// Compatibility model:
//
//	Layer 1 (direct, from regulation.bin):
//	  mask bits 0..35  = EquipParamGem.canMountWep_Dagger .. canMountWep_Torch
//	  mask bits 36..43 = the eight DLC canMountWep fields packed after canMountWep_Torch
//	    bit 36 → wepType 88 (Hand-to-Hand / Dryleaf Arts)
//	    bit 37 → wepType 89 (Perfume Bottles)
//	    bit 38 → wepType 90 (Dueling / Thrusting Shields)
//	    bit 39 → wepType 91 (Throwing / Smithscript Blades)
//	    bit 40 → wepType 92 (Backhand Blades)
//	    bit 41 → wepType 93 (Light Greatswords)
//	    bit 42 → wepType 94 (Great Katanas)
//	    bit 43 → wepType 95 (Beast Claws)
//	  The checked-in CSV dump used an older PARAMDEF and exposes only bits 36..39.
//	  The raw EquipParamGem.param supplies the full 64-bit field. Its low 40 bits
//	  are cross-checked against the CSV before bits 40..43 are accepted.
//
//	Layer 2 (legacy fallback): if a future/older input genuinely has a zero direct
//	mask, compatibility may still be inferred from mountWepTextId grouping plus
//	swordArtsParamId. Current regulation data resolves all DLC arts directly.
//
// Usage:
//
//	go run ./tools/generate_aow_compat \
//	    -gem tmp/regulation-bin-dump/csv/EquipParamGem.csv \
//	    -gem-param tmp/regulation-bin-dump/params/EquipParamGem.param \
//	    -weapon tmp/regulation-bin-dump/csv/EquipParamWeapon.csv
//
// The tool never reads a hard-coded tmp/ path; both inputs are explicit flags.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"flag"
	"fmt"
	"go/format"
	"io"
	"math"
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

// reservedBitNames labels mask bits 36..43. The first four are exposed by the
// legacy CSV dump as reserved_canMountWep bits 0..3; the final four exist only
// in the raw PARAM because that dump used a pre-DLC PARAMDEF.
var reservedBitNames = []string{
	"canMountWep_HandToHand",       // bit 36 → wepType 88
	"canMountWep_PerfumeBottle",    // bit 37 → wepType 89
	"canMountWep_ThrustingShield",  // bit 38 → wepType 90
	"canMountWep_ThrowingWeapon",   // bit 39 → wepType 91
	"canMountWep_ReverseHandSword", // bit 40 → wepType 92
	"canMountWep_LightGreatsword",  // bit 41 → wepType 93
	"canMountWep_GreatKatana",      // bit 42 → wepType 94
	"canMountWep_BeastClaw",        // bit 43 → wepType 95
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
	9:  4,  // Curved Sword
	11: 5,  // Curved Greatsword
	13: 6,  // Katana
	14: 7,  // Twinblade
	15: 8,  // Thrusting Sword
	16: 9,  // Heavy Thrusting Sword
	17: 10, // Axe
	19: 11, // Greataxe
	21: 12, // Hammer
	23: 13, // Great Hammer
	24: 14, // Flail
	25: 15, // Spear
	28: 17, // Great Spear (engine checks canMountWep_SpearHeavy; canMountWep_SpearLarge is unset on every gem row)
	32: 17, // Heavy Spear engine category
	29: 18, // Halberd
	33: 18, // Legacy halberd engine category (no mountable app weapon)
	31: 19, // Reaper
	35: 20, // Fist
	37: 21, // Claw
	39: 22, // Whip
	41: 23, // Colossal Weapon
	50: 24, // Light Bow
	51: 25, // Bow
	53: 26, // Greatbow
	55: 27, // Crossbow
	56: 28, // Ballista
	57: 29, // Glintstone Staff
	61: 30, // Sacred Seal
	65: 32, // Small Shield
	67: 33, // Medium Shield
	69: 34, // Greatshield
	87: 35, // Torch (real torch wepType; the game checks canMountWep_Torch)
	88: 36, // Hand-to-Hand Arts (reserved_canMountWep bit 0)
	89: 37, // Perfume Bottles (reserved_canMountWep bit 1)
	90: 38, // Dueling / Thrusting Shields (reserved_canMountWep bit 2)
	91: 39, // Throwing / Smithscript Blades (reserved_canMountWep bit 3)
	92: 40, // Backhand Blades
	93: 41, // Light Greatswords
	94: 42, // Great Katanas
	95: 43, // Beast Claws
	// wepType 68 is intentionally absent: no weapon row uses it (real torches are
	// wepType 87), so mapping it would describe a non-existent weapon class.
}

const maskBits = 44

// EquipParamWeapon materializes only the +0 row of each affinity: a family base
// row and its 12 affinity anchors are spaced materializedRowStep apart, and the
// last one (Occult) sits maxAffinityRowOffset above the base. Upgrade levels
// 1..25 are derived through ReinforceParamWeapon and have no rows of their own.
const (
	materializedRowStep  = 100
	maxAffinityRowOffset = 1200
)

const (
	paramRowCountOffset = 0x0A
	paramRowTableOffset = 0x40
	paramRowEntrySize   = 0x18
	paramCompatOffset   = 0x38
	paramCompatSize     = 8
)

type gem struct {
	rid   uint32
	item  uint32
	sap   int
	mwtid int
	mask  uint64
}

type weapon struct {
	rid               uint32
	wepType           uint16
	gm                uint8
	canChangeAffinity bool
	sap               int
	originRaw         int    // EquipParamWeapon.originEquipWep verbatim (signed; -1 = none)
	origin            uint32 // originRaw once confirmed against the materialized rows; 0 = no confirmed relation
}

type result struct {
	gems         []gem
	weapons      []weapon
	masks        map[uint32]uint64   // AoW item ID → mask (only non-zero masks)
	heuristic    map[uint32][]uint16 // AoW item ID → wepTypes (mask==0 DLC arts)
	gemHash      string
	gemParamHash string
	weaponHash   string
	aowGo        string
	weaponGo     string
	tsSource     string
}

func main() {
	gemPath := flag.String("gem", "", "path to EquipParamGem.csv")
	gemParamPath := flag.String("gem-param", "", "path to raw EquipParamGem.param")
	weaponPath := flag.String("weapon", "", "path to EquipParamWeapon.csv")
	aowOut := flag.String("aow-out", "backend/db/data/aow_compat.go", "output path for aow_compat.go")
	weaponOut := flag.String("weapon-out", "backend/db/data/weapon_gem_mount.go", "output path for weapon_gem_mount.go")
	tsOut := flag.String("ts-out", "frontend/src/data/aowCompat.generated.ts", "output path for the TypeScript mirror")
	flag.Parse()

	if *gemPath == "" || *gemParamPath == "" || *weaponPath == "" {
		fatalf("-gem, -gem-param, and -weapon are required")
	}

	res, err := generate(*gemPath, *gemParamPath, *weaponPath)
	if err != nil {
		fatalf("%v", err)
	}

	write(*aowOut, res.aowGo)
	write(*weaponOut, res.weaponGo)
	write(*tsOut, res.tsSource)

	fmt.Printf("aow_compat: %d masks, %d heuristic entries\n", len(res.masks), len(res.heuristic))
	fmt.Printf("weapon_gem_mount: %d weapon metadata rows\n", countWeaponMetadata(res.weapons))
}

// generate is the pure core: it reads both CSVs and returns every generated
// artifact plus the intermediate model, so tests can assert on it directly.
func generate(gemPath, gemParamPath, weaponPath string) (*result, error) {
	gems, gemHash, err := readGems(gemPath)
	if err != nil {
		return nil, err
	}
	rawMasks, gemParamHash, err := readGemParamMasks(gemParamPath)
	if err != nil {
		return nil, err
	}
	if err := applyRawGemMasks(gems, rawMasks, gemPath, gemParamPath); err != nil {
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
		gems:         gems,
		weapons:      weapons,
		masks:        masks,
		heuristic:    heuristic,
		gemHash:      gemHash,
		gemParamHash: gemParamHash,
		weaponHash:   weaponHash,
	}

	aowGo, err := renderAoWGo(res, gemPath, gemParamPath, weaponPath)
	if err != nil {
		return nil, err
	}
	weaponGo, err := renderWeaponGo(res, weaponPath)
	if err != nil {
		return nil, err
	}
	res.aowGo = aowGo
	res.weaponGo = weaponGo
	res.tsSource = renderTS(res, gemPath, gemParamPath, weaponPath)
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

// readGemParamMasks reads the 64-bit weapon compatibility field directly from
// EquipParamGem.param. The CSV export in tmp/ was produced with a pre-DLC
// PARAMDEF: it exposes bits 0..39 but labels bits 40..43 as padding. Reading the
// raw field is therefore required to recover the four final DLC categories.
//
// These offsets are the stable Elden Ring PARAM layout for this table. The
// caller cross-checks every row ID and the low 40 mask bits against the CSV, so
// a schema/layout mismatch fails before any generated artifact is written.
func readGemParamMasks(path string) (map[uint32]uint64, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("open %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", sum)

	if len(data) < paramRowTableOffset {
		return nil, "", fmt.Errorf("%s: truncated PARAM header (%d bytes)", path, len(data))
	}
	rowCount := int(binary.LittleEndian.Uint16(data[paramRowCountOffset : paramRowCountOffset+2]))
	tableEnd := paramRowTableOffset + rowCount*paramRowEntrySize
	if rowCount == 0 || tableEnd > len(data) {
		return nil, "", fmt.Errorf("%s: invalid PARAM row table (rows=%d, end=0x%X, size=0x%X)",
			path, rowCount, tableEnd, len(data))
	}

	masks := make(map[uint32]uint64, rowCount)
	for i := 0; i < rowCount; i++ {
		entry := paramRowTableOffset + i*paramRowEntrySize
		rawRowID := int64(binary.LittleEndian.Uint64(data[entry : entry+8]))
		if rawRowID < 0 || rawRowID > math.MaxUint32 {
			return nil, "", fmt.Errorf("%s: PARAM row %d has Row ID %d outside uint32", path, i+1, rawRowID)
		}
		rowID := uint32(rawRowID)
		if _, duplicate := masks[rowID]; duplicate {
			return nil, "", fmt.Errorf("%s: duplicate PARAM Row ID %d", path, rowID)
		}

		dataOffset64 := binary.LittleEndian.Uint64(data[entry+8 : entry+16])
		if dataOffset64 > uint64(len(data)) {
			return nil, "", fmt.Errorf("%s: PARAM Row ID %d data offset 0x%X exceeds file size 0x%X",
				path, rowID, dataOffset64, len(data))
		}
		dataOffset := int(dataOffset64)
		maskEnd := dataOffset + paramCompatOffset + paramCompatSize
		if dataOffset < tableEnd || maskEnd > len(data) {
			return nil, "", fmt.Errorf("%s: PARAM Row ID %d has invalid data range 0x%X..0x%X",
				path, rowID, dataOffset, maskEnd)
		}
		mask := binary.LittleEndian.Uint64(data[dataOffset+paramCompatOffset : maskEnd])
		if mask>>maskBits != 0 {
			return nil, "", fmt.Errorf("%s: PARAM Row ID %d sets unsupported compatibility bits above %d (mask 0x%X)",
				path, rowID, maskBits-1, mask)
		}
		masks[rowID] = mask
	}
	return masks, hash, nil
}

func applyRawGemMasks(gems []gem, rawMasks map[uint32]uint64, csvPath, paramPath string) error {
	if len(rawMasks) != len(gems) {
		return fmt.Errorf("%s and %s row count mismatch: CSV=%d PARAM=%d",
			csvPath, paramPath, len(gems), len(rawMasks))
	}
	const csvMaskBits = 40
	const csvMask = (uint64(1) << csvMaskBits) - 1
	for i := range gems {
		rawMask, ok := rawMasks[gems[i].rid]
		if !ok {
			return fmt.Errorf("%s: PARAM missing EquipParamGem Row ID %d from %s",
				paramPath, gems[i].rid, csvPath)
		}
		if rawMask&csvMask != gems[i].mask {
			return fmt.Errorf("EquipParamGem Row ID %d compatibility mismatch: %s=0x%X, %s low %d bits=0x%X",
				gems[i].rid, csvPath, gems[i].mask, paramPath, csvMaskBits, rawMask&csvMask)
		}
		gems[i].mask = rawMask
	}
	return nil
}

func readWeapons(path string) ([]weapon, string, error) {
	header, rows, hash, err := readCSV(path)
	if err != nil {
		return nil, "", err
	}
	if err := requireColumns(path, header, []string{"Row ID", "wepType", "gemMountType", "disableGemAttr", "swordArtsParamId", "originEquipWep"}); err != nil {
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
		disableGemAttr, err := rr.reqBit("disableGemAttr")
		if err != nil {
			return nil, "", err
		}
		sap, err := rr.reqInt("swordArtsParamId") // -1 allowed
		if err != nil {
			return nil, "", err
		}
		// originEquipWep is read as a signed integer on purpose: rows without a
		// family relation carry -1, which must never wrap into a uint32 row ID.
		originRaw, err := rr.reqInt("originEquipWep") // -1 allowed
		if err != nil {
			return nil, "", err
		}
		weapons = append(weapons, weapon{
			rid:               rid,
			wepType:           uint16(wepType),
			gm:                uint8(gm),
			canChangeAffinity: gm == 2 && !disableGemAttr,
			sap:               sap,
			originRaw:         originRaw,
		})
	}
	sort.Slice(weapons, func(i, j int) bool { return weapons[i].rid < weapons[j].rid })
	resolveWeaponOrigins(weapons)
	return weapons, hash, nil
}

// resolveWeaponOrigins turns each row's raw originEquipWep back-pointer into a
// confirmed canonical family base row, or into 0 when the data does not prove
// the relation. It is deliberately fail-soft rather than fatal: regulation.bin
// legitimately contains rows with no family (originEquipWep == -1, e.g. the
// Unarmed placeholder row 110000) alongside rows pointing outside the
// materialized set. Emitting 0 records "no confirmed relation"; it never
// substitutes a hypothesis for missing evidence.
func resolveWeaponOrigins(weapons []weapon) {
	materialized := make(map[uint32]bool, len(weapons))
	for _, w := range weapons {
		if w.rid%materializedRowStep == 0 {
			materialized[w.rid] = true
		}
	}
	for i := range weapons {
		weapons[i].origin = confirmedOrigin(weapons[i].rid, weapons[i].originRaw, materialized)
	}
}

// confirmedOrigin accepts a family relation only when every condition holds:
// the pointer is non-negative and inside the uint32 row-ID range, it names a
// materialized row, the row it is attached to is not below it, and the distance
// between them is one of the 13 affinity offsets {0, 100, ..., 1200}. Anything
// else returns 0.
func confirmedOrigin(rid uint32, originRaw int, materialized map[uint32]bool) uint32 {
	if originRaw < 0 || int64(originRaw) > math.MaxUint32 {
		return 0
	}
	origin := uint32(originRaw)
	if !materialized[origin] || rid < origin {
		return 0
	}
	offset := rid - origin
	if offset > maxAffinityRowOffset || offset%materializedRowStep != 0 {
		return 0
	}
	return origin
}

func renderAoWGo(res *result, gemPath, gemParamPath, weaponPath string) (string, error) {
	var b strings.Builder
	b.WriteString(header(gemPath, gemParamPath, weaponPath, res.gemHash, res.gemParamHash, res.weaponHash))
	b.WriteString("package data\n\n")

	// Masks.
	b.WriteString("// AoWCompatMasks maps an Ash of War item ID to its 44-bit weapon-compatibility\n")
	b.WriteString("// mask, sourced directly from regulation.bin:\n")
	b.WriteString("//   bits 0..35  = EquipParamGem.canMountWep_Dagger .. canMountWep_Torch\n")
	b.WriteString("//   bits 36..43 = DLC canMountWep fields HandToHand .. BeastClaw\n")
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
	b.WriteString("// AoWHeuristicWepTypes lists compatible wepTypes for legacy inputs whose arts carry\n")
	b.WriteString("// no direct canMountWep bit. This is a HEURISTIC, not a regulation\n")
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
	b.WriteString("// CanMountWepNames names each mask bit (0..43) for diagnostics. Bits 0..35 are the\n")
	b.WriteString("// base-game EquipParamGem fields; bits 36..43 are the DLC canMountWep fields.\n")
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
	b.WriteString("// WeaponGemMount holds AoW and affinity metadata for a weapon item ID.\n")
	b.WriteString("type WeaponGemMount struct {\n")
	b.WriteString("\tWepType          uint16 // EquipParamWeapon.wepType (weapon category integer)\n")
	b.WriteString("\tGemMountType     uint8  // EquipParamWeapon.gemMountType: 2 permits custom AoW mounting\n")
	b.WriteString("\tCanChangeAffinity bool   // gemMountType == 2 && disableGemAttr == 0\n")
	b.WriteString("\tOriginEquipWep uint32 // EquipParamWeapon.originEquipWep: canonical family base row, confirmed against the materialized rows. 0 means no confirmed relation.\n")
	b.WriteString("}\n\n")
	b.WriteString("// WeaponGemMounts maps every EquipParamWeapon Row ID to its AoW/affinity metadata.\n")
	b.WriteString("// Rows with gemMountType == 0 are deliberately retained: they are required to\n")
	b.WriteString("// distinguish a known non-editable weapon from missing metadata.\n")
	b.WriteString("var WeaponGemMounts = map[uint32]WeaponGemMount{\n")
	for _, w := range res.weapons {
		if w.rid%100 != 0 {
			continue
		}
		fields := fmt.Sprintf("WepType: %d, GemMountType: %d", w.wepType, w.gm)
		if w.canChangeAffinity {
			fields += ", CanChangeAffinity: true"
		}
		if w.origin != 0 {
			fields += fmt.Sprintf(", OriginEquipWep: 0x%08X", w.origin)
		}
		fmt.Fprintf(&b, "\t0x%08X: {%s},\n", w.rid, fields)
	}
	b.WriteString("}\n")
	return gofmt(b.String())
}

func renderTS(res *result, gemPath, gemParamPath, weaponPath string) string {
	var b strings.Builder
	b.WriteString("// Code generated by tools/generate_aow_compat; DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// Sources: %s (sha256 %s), %s (sha256 %s), %s (sha256 %s).\n",
		gemPath, res.gemHash, gemParamPath, res.gemParamHash, weaponPath, res.weaponHash)
	b.WriteString("//\n")
	b.WriteString("// Single source of truth shared with backend/db/data/aow_compat.go. Bits 0..35 come\n")
	b.WriteString("// from the CSV canMountWep_* columns; bits 36..43 come from the raw PARAM field.\n\n")

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

func header(gemPath, gemParamPath, weaponPath, gemHash, gemParamHash, weaponHash string) string {
	return fmt.Sprintf(
		"// Code generated by tools/generate_aow_compat; DO NOT EDIT.\n"+
			"//\n"+
			"// Sources:\n"+
			"//   %s (sha256 %s)\n"+
			"//   %s (sha256 %s)\n"+
			"//   %s (sha256 %s)\n"+
			"//\n"+
			"// Direct data: mask bits 0..43 (base-game and DLC canMountWep fields).\n"+
			"// Legacy fallback: AoWHeuristicWepTypes (mountWepTextId + swordArtsParamId inference).\n\n",
		gemPath, gemHash, gemParamPath, gemParamHash, weaponPath, weaponHash)
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
	return fmt.Sprintf("%s (mask bit %d)", reservedBitNames[i], bit)
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

func countWeaponMetadata(weapons []weapon) int {
	n := 0
	for _, w := range weapons {
		if w.rid%100 == 0 {
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
