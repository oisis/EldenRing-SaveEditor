package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/db/data"
)

const (
	gemCSVFixture = "testdata/EquipParamGem.csv"
	weaponFixture = "testdata/EquipParamWeapon.csv"
)

func mustGenerate(t *testing.T) *result {
	t.Helper()
	res, err := generate(gemCSVFixture, writeGemParamFixture(t, map[uint32]uint64{
		10:  0x3,
		100: 0xFF000000000,
		200: uint64(1) << 40,
		300: uint64(1) << 38,
	}), weaponFixture)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return res
}

func writeGemParamFixture(t *testing.T, masks map[uint32]uint64) string {
	t.Helper()
	rowIDs := make([]uint32, 0, len(masks))
	for rowID := range masks {
		rowIDs = append(rowIDs, rowID)
	}
	sort.Slice(rowIDs, func(i, j int) bool { return rowIDs[i] < rowIDs[j] })

	tableEnd := paramRowTableOffset + len(rowIDs)*paramRowEntrySize
	data := make([]byte, tableEnd+len(rowIDs)*0x60)
	binary.LittleEndian.PutUint16(data[paramRowCountOffset:paramRowCountOffset+2], uint16(len(rowIDs)))
	for i, rowID := range rowIDs {
		entry := paramRowTableOffset + i*paramRowEntrySize
		dataOffset := tableEnd + i*0x60
		binary.LittleEndian.PutUint64(data[entry:entry+8], uint64(rowID))
		binary.LittleEndian.PutUint64(data[entry+8:entry+16], uint64(dataOffset))
		binary.LittleEndian.PutUint64(
			data[dataOffset+paramCompatOffset:dataOffset+paramCompatOffset+paramCompatSize],
			masks[rowID],
		)
	}
	path := filepath.Join(t.TempDir(), "EquipParamGem.param")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write PARAM fixture: %v", err)
	}
	return path
}

func TestGenerate_MaskBits0to35(t *testing.T) {
	res := mustGenerate(t)
	// Row 10 sets canMountWep_Dagger (bit 0) + canMountWep_SwordNormal (bit 1).
	if got := res.masks[0x8000000A]; got != 0x3 {
		t.Errorf("row 10 mask = 0x%X, want 0x3", got)
	}
}

func TestGenerate_DLCBitsMapTo36_43(t *testing.T) {
	res := mustGenerate(t)
	// The CSV carries bits 36..39; the raw PARAM adds bits 40..43.
	if got := res.masks[0x80000064]; got != 0xFF000000000 {
		t.Errorf("row 100 mask = 0x%X, want 0xFF000000000 (bits 36-43)", got)
	}
	// Row 300 has reserved bit 2 → mask bit 38 (Dueling/Thrusting Shields).
	if got := res.masks[0x8000012C]; got != 0x4000000000 {
		t.Errorf("row 300 mask = 0x%X, want 0x4000000000 (bit 38)", got)
	}
}

func TestGenerate_RawParamResolvesBackhandBladeBit(t *testing.T) {
	res := mustGenerate(t)
	if got := res.masks[0x800000C8]; got != uint64(1)<<40 {
		t.Errorf("row 200 mask = 0x%X, want Backhand Blade bit 40", got)
	}
	if _, ok := res.heuristic[0x800000C8]; ok {
		t.Error("row 200 has a direct PARAM bit and must not use the heuristic")
	}
}

func TestGenerate_RawParamMustMatchCSVLowBits(t *testing.T) {
	paramPath := writeGemParamFixture(t, map[uint32]uint64{
		10: 0x2, 100: 0xF000000000, 200: 0, 300: uint64(1) << 38,
	})
	_, err := generate(gemCSVFixture, paramPath, weaponFixture)
	if err == nil {
		t.Fatal("expected CSV/PARAM mask mismatch")
	}
	for _, want := range []string{"Row ID 10", "compatibility mismatch", "0x3", "0x2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want substring %q", err, want)
		}
	}
}

func TestReadGemParamMasks_RejectsUnknownFutureBits(t *testing.T) {
	paramPath := writeGemParamFixture(t, map[uint32]uint64{10: uint64(1) << 44})
	_, _, err := readGemParamMasks(paramPath)
	if err == nil {
		t.Fatal("expected unsupported compatibility bit error")
	}
	if !strings.Contains(err.Error(), "unsupported compatibility bits above 43") {
		t.Errorf("error = %q, want unsupported-bit detail", err)
	}
}

func TestGenerate_WeaponMounts(t *testing.T) {
	res := mustGenerate(t)
	// Keys are the decimal Row ID rendered as hex (1000 → 0x3E8, 4000 → 0xFA0).
	// Every weapon is retained, including gemMountType 0, because affinity
	// validation needs a known-negative record instead of missing metadata.
	if !strings.Contains(res.weaponGo, "0x000003E8: {WepType: 92, GemMountType: 2, CanChangeAffinity: true}") {
		t.Error("weapon row 1000 (gm2) missing from generated WeaponGemMounts")
	}
	if !strings.Contains(res.weaponGo, "0x00000FA0: {WepType: 3, GemMountType: 0}") {
		t.Error("weapon row 4000 (gm0) missing from WeaponGemMounts")
	}
	if !strings.Contains(res.weaponGo, "0x00000BB8: {WepType: 3, GemMountType: 1}") {
		t.Error("somber weapon row 3000 (gm1) missing from generated WeaponGemMounts")
	}
	if !strings.Contains(res.weaponGo, "0x000007D0: {WepType: 3, GemMountType: 2}") {
		t.Error("gm2 + disableGemAttr=1 weapon must mount AoW but block affinity")
	}
}

func TestGenerate_WepTypeMappingMatchesEngineCategories(t *testing.T) {
	expected := map[uint16]uint8{
		1: 0, 3: 1, 5: 2, 7: 3, 9: 4, 11: 5, 13: 6, 14: 7,
		15: 8, 16: 9, 17: 10, 19: 11, 21: 12, 23: 13, 24: 14,
		25: 15, 28: 17, 32: 17, 29: 18, 33: 18, 31: 19, 35: 20,
		37: 21, 39: 22, 41: 23, 50: 24, 51: 25, 53: 26, 55: 27,
		56: 28, 57: 29, 61: 30, 65: 32, 67: 33, 69: 34, 87: 35,
		88: 36, 89: 37, 90: 38, 91: 39, 92: 40, 93: 41, 94: 42, 95: 43,
	}
	if len(wepTypeToBit) != len(expected) {
		t.Fatalf("wepTypeToBit has %d entries, want %d", len(wepTypeToBit), len(expected))
	}
	for wepType, wantBit := range expected {
		if got, ok := wepTypeToBit[wepType]; !ok || got != wantBit {
			t.Errorf("wepType %d maps to bit %d (present=%v), want %d", wepType, got, ok, wantBit)
		}
	}
}

func TestGenerate_Reserved2GuardFails(t *testing.T) {
	_, err := generate("testdata/EquipParamGem_reserved2.csv", "not-read.param", weaponFixture)
	if err == nil {
		t.Fatal("expected error for non-zero reserved2_canMountWep")
	}
	if !strings.Contains(err.Error(), "reserved2_canMountWep") {
		t.Errorf("error = %q, want mention of reserved2_canMountWep", err)
	}
}

func TestGenerate_MissingColumnFails(t *testing.T) {
	_, err := generate("testdata/EquipParamGem_missingcol.csv", "not-read.param", weaponFixture)
	if err == nil {
		t.Fatal("expected error for missing required column")
	}
	if !strings.Contains(err.Error(), "canMountWep_Torch") {
		t.Errorf("error = %q, want mention of the missing column", err)
	}
}

// TestGenerate_StrictValidationRejects covers the fail-closed parsing: every
// malformed required field aborts generation instead of being silently dropped
// or defaulted. Each case names the substrings the error must carry so the
// message stays actionable (field name + offending value).
func TestGenerate_StrictValidationRejects(t *testing.T) {
	cases := []struct {
		name    string
		gem     string
		weapon  string
		wantSub []string
	}{
		{"invalid Row ID", "testdata/EquipParamGem_badrowid.csv", weaponFixture, []string{"Row ID", "abc"}},
		{"duplicate Row ID", "testdata/EquipParamGem_duprowid.csv", weaponFixture, []string{"duplicate Row ID", "10"}},
		{"canMountWep = 2", "testdata/EquipParamGem_canmount2.csv", weaponFixture, []string{"canMountWep_Dagger", "2", "exactly 0 or 1"}},
		{"canMountWep = abc", "testdata/EquipParamGem_canmountabc.csv", weaponFixture, []string{"canMountWep_Dagger", "abc"}},
		{"reserved out of range", "testdata/EquipParamGem_badreserved.csv", weaponFixture, []string{"reserved_canMountWep", "16", "0..15"}},
		{"reserved2 non-integer text", "testdata/EquipParamGem_reserved2text.csv", weaponFixture, []string{"reserved2_canMountWep", "abc"}},
		{"reserved2 non-zero", "testdata/EquipParamGem_reserved2.csv", weaponFixture, []string{"reserved2_canMountWep", "non-zero"}},
		{"truncated CSV row", "testdata/EquipParamGem_short.csv", weaponFixture, []string{"truncated record"}},
		{"wepType overflows uint16", gemCSVFixture, "testdata/EquipParamWeapon_weptype_overflow.csv", []string{"wepType", "70000", "uint16"}},
		{"gemMountType out of 0..2", gemCSVFixture, "testdata/EquipParamWeapon_gemmount_bad.csv", []string{"gemMountType", "3"}},
		{"disableGemAttr out of 0..1", gemCSVFixture, "testdata/EquipParamWeapon_disablegem_bad.csv", []string{"disableGemAttr", "2", "exactly 0 or 1"}},
		{"weapon invalid Row ID", gemCSVFixture, "testdata/EquipParamWeapon_badrowid.csv", []string{"Row ID", "abc"}},
		// Duplicate must name file, current CSV row (3), the duplicate Row ID (1000)
		// and the CSV row it first appeared on (2).
		{"weapon duplicate Row ID", gemCSVFixture, "testdata/EquipParamWeapon_duprowid.csv", []string{"EquipParamWeapon_duprowid.csv", "CSV row 3", "duplicate Row ID", "1000", "CSV row 2"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			paramPath := "not-read.param"
			if c.gem == gemCSVFixture {
				paramPath = writeGemParamFixture(t, map[uint32]uint64{
					10: 0x3, 100: 0xF000000000, 200: 0, 300: uint64(1) << 38,
				})
			}
			_, err := generate(c.gem, paramPath, c.weapon)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			for _, sub := range c.wantSub {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("error = %q, want substring %q", err, sub)
				}
			}
		})
	}
}

// TestGenerate_NegativeOneAccepted verifies -1 is a valid value where allowed
// (swordArtsParamId, mountWepTextId) and does not abort generation.
func TestGenerate_NegativeOneAccepted(t *testing.T) {
	paramPath := writeGemParamFixture(t, map[uint32]uint64{10: 0})
	if _, err := generate("testdata/EquipParamGem_neg1.csv", paramPath, weaponFixture); err != nil {
		t.Fatalf("generate with -1 swordArtsParamId/mountWepTextId: %v", err)
	}
}

func TestGenerate_Deterministic(t *testing.T) {
	paramPath := writeGemParamFixture(t, map[uint32]uint64{
		10: 0x3, 100: 0xF000000000, 200: 0, 300: uint64(1) << 38,
	})
	a, err := generate(gemCSVFixture, paramPath, weaponFixture)
	if err != nil {
		t.Fatal(err)
	}
	b, err := generate(gemCSVFixture, paramPath, weaponFixture)
	if err != nil {
		t.Fatal(err)
	}
	if a.aowGo != b.aowGo || a.weaponGo != b.weaponGo || a.tsSource != b.tsSource {
		t.Error("generator output is not deterministic across runs")
	}
}

func TestGenerate_ProvenanceHeader(t *testing.T) {
	res := mustGenerate(t)
	if !strings.Contains(res.aowGo, "Code generated by tools/generate_aow_compat") {
		t.Error("aow_compat.go missing DO NOT EDIT provenance line")
	}
	// The header must carry the input hashes so the source dump is identifiable.
	if !strings.Contains(res.aowGo, res.gemHash) || !strings.Contains(res.aowGo, res.weaponHash) {
		t.Error("aow_compat.go header missing input CSV sha256 hashes")
	}
	if !strings.Contains(res.aowGo, filepath.ToSlash(gemCSVFixture)) {
		t.Error("aow_compat.go header missing gem source path")
	}
	if !strings.Contains(res.tsSource, res.gemHash) || !strings.Contains(res.tsSource, res.gemParamHash) {
		t.Error("TS mirror missing gem source hashes")
	}
}

// TestGenerate_WeaponOriginEquipWep covers the confirmed-family relation the
// canonical weapon-identity resolver depends on: the base row points at itself,
// affinity anchors point back at the base, and the +1200 (Occult) boundary is
// inclusive.
func TestGenerate_WeaponOriginEquipWep(t *testing.T) {
	paramPath := writeGemParamFixture(t, map[uint32]uint64{
		10: 0x3, 100: 0xF000000000, 200: 0, 300: uint64(1) << 38,
	})
	res, err := generate(gemCSVFixture, paramPath, "testdata/EquipParamWeapon_origin.csv")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Row 1000 → 0x3E8 (self-origin), 1100 → 0x44C (+100), 2200 → 0x898 (+1200).
	for _, want := range []string{
		"0x000003E8: {WepType: 92, GemMountType: 2, CanChangeAffinity: true, OriginEquipWep: 0x000003E8}",
		"0x0000044C: {WepType: 92, GemMountType: 2, CanChangeAffinity: true, OriginEquipWep: 0x000003E8}",
		"0x00000898: {WepType: 92, GemMountType: 2, CanChangeAffinity: true, OriginEquipWep: 0x000003E8}",
	} {
		if !strings.Contains(res.weaponGo, want) {
			t.Errorf("generated WeaponGemMounts missing %q", want)
		}
	}
}

// TestGenerate_WeaponOriginRequiresColumn proves originEquipWep is a required
// input: silently defaulting it would emit a WeaponGemMounts whose relations are
// all "unconfirmed", which the resolver cannot distinguish from real data.
func TestGenerate_WeaponOriginRequiresColumn(t *testing.T) {
	paramPath := writeGemParamFixture(t, map[uint32]uint64{
		10: 0x3, 100: 0xF000000000, 200: 0, 300: uint64(1) << 38,
	})
	_, err := generate(gemCSVFixture, paramPath, "testdata/EquipParamWeapon_origin_missingcol.csv")
	if err == nil {
		t.Fatal("expected error for missing originEquipWep column")
	}
	if !strings.Contains(err.Error(), "originEquipWep") {
		t.Errorf("error = %q, want mention of originEquipWep", err)
	}
}

// TestGenerate_WeaponOriginUnconfirmed is the fail-closed half of the contract:
// every relation the data does not prove must emit 0 ("no confirmed relation")
// and must never be converted into a hypothesis. -1 in particular must not wrap
// into a uint32 row ID.
func TestGenerate_WeaponOriginUnconfirmed(t *testing.T) {
	cases := []struct {
		name    string
		weapon  string
		absent  []string
		present []string
	}{
		{
			// The shared fixture carries originEquipWep = -1 on every row.
			name:    "negative one is not a row ID",
			weapon:  weaponFixture,
			absent:  []string{", OriginEquipWep:"},
			present: []string{"0x000003E8: {WepType: 92, GemMountType: 2, CanChangeAffinity: true}"},
		},
		{
			// Row 1100 points at 1050, which is not a materialized row.
			name:   "origin outside the materialized rows",
			weapon: "testdata/EquipParamWeapon_origin_unmaterialized.csv",
			absent: []string{", OriginEquipWep:"},
			present: []string{
				"0x0000044C: {WepType: 92, GemMountType: 2, CanChangeAffinity: true}",
			},
		},
		{
			// Row 900 sits below its origin; row 2300 is +1300 above it.
			name:   "offset outside 0..1200",
			weapon: "testdata/EquipParamWeapon_origin_offset.csv",
			absent: []string{", OriginEquipWep:"},
			present: []string{
				"0x00000384: {WepType: 92, GemMountType: 2, CanChangeAffinity: true}",
				"0x000008FC: {WepType: 92, GemMountType: 2, CanChangeAffinity: true}",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			paramPath := writeGemParamFixture(t, map[uint32]uint64{
				10: 0x3, 100: 0xF000000000, 200: 0, 300: uint64(1) << 38,
			})
			res, err := generate(gemCSVFixture, paramPath, c.weapon)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			for _, sub := range c.absent {
				if strings.Contains(res.weaponGo, sub) {
					t.Errorf("generated WeaponGemMounts must not emit %q for unconfirmed relations", sub)
				}
			}
			for _, sub := range c.present {
				if !strings.Contains(res.weaponGo, sub) {
					t.Errorf("generated WeaponGemMounts missing %q", sub)
				}
			}
		})
	}
}

// TestGenerate_WeaponOriginStableEmission pins the emitted field order so a
// regeneration cannot reorder WeaponGemMount literals into a churn-only diff.
func TestGenerate_WeaponOriginStableEmission(t *testing.T) {
	paramPath := writeGemParamFixture(t, map[uint32]uint64{
		10: 0x3, 100: 0xF000000000, 200: 0, 300: uint64(1) << 38,
	})
	a, err := generate(gemCSVFixture, paramPath, "testdata/EquipParamWeapon_origin.csv")
	if err != nil {
		t.Fatal(err)
	}
	b, err := generate(gemCSVFixture, paramPath, "testdata/EquipParamWeapon_origin.csv")
	if err != nil {
		t.Fatal(err)
	}
	if a.weaponGo != b.weaponGo {
		t.Error("WeaponGemMounts emission is not reproducible across runs")
	}
	if !regexp.MustCompile(`OriginEquipWep\s+uint32`).MatchString(a.weaponGo) {
		t.Error("WeaponGemMount struct is missing the generated OriginEquipWep field")
	}
	// Keys stay in ascending Row ID order.
	iBase := strings.Index(a.weaponGo, "0x000003E8:")
	iMid := strings.Index(a.weaponGo, "0x0000044C:")
	iLast := strings.Index(a.weaponGo, "0x00000898:")
	if iBase < 0 || iMid < iBase || iLast < iMid {
		t.Errorf("WeaponGemMounts keys are not in ascending Row ID order (%d, %d, %d)", iBase, iMid, iLast)
	}
}

// --- weapon_weptype_generated.go ---

func TestRenderWepTypeGo_AppIDsOnlySortedAndDense(t *testing.T) {
	weapons := []weapon{
		{rid: 0x00000BB8, wepType: 5},
		{rid: 0x000003E8, wepType: 1},
		{rid: 0x0000044C, wepType: 1},
		{rid: 0x000007D0, wepType: 90},
		{rid: 0x00000FA0, wepType: 67}, // present in the param but not in the app DB
		{rid: 0x00000898, wepType: 92},
		{rid: 0x000009C4, wepType: 94},
		{rid: 0x00000A28, wepType: 93},
	}
	appIDs := map[uint32]bool{
		0x000003E8: true, 0x0000044C: true, 0x000007D0: true,
		0x00000898: true, 0x000009C4: true, 0x00000A28: true, 0x00000BB8: true,
	}

	src, err := renderWepTypeGo(weapons, appIDs, weaponFixture)
	if err != nil {
		t.Fatalf("renderWepTypeGo: %v", err)
	}

	if !strings.Contains(src, "// Regenerated from "+weaponFixture+"; do not\n") {
		t.Errorf("header does not record the EquipParamWeapon path it was given (%s):\n%s", weaponFixture, src)
	}

	if !strings.Contains(src, "var weaponWepType = map[uint32]uint16{ // 7 entries") {
		t.Errorf("entry counter missing or wrong:\n%s", src)
	}
	if strings.Contains(src, "0x00000FA0") {
		t.Error("emitted a param row that the app DB does not expose")
	}
	want := "\t0x000003E8: 1, 0x0000044C: 1, 0x000007D0: 90, 0x00000898: 92, 0x000009C4: 94, 0x00000A28: 93,\n\t0x00000BB8: 5,\n}\n"
	if !strings.HasSuffix(src, want) {
		t.Errorf("dense sorted layout mismatch:\ngot:\n%s\nwant suffix:\n%s", src, want)
	}
	if !strings.HasPrefix(src, "package data\n\n// weapon_weptype_generated.go —") {
		t.Errorf("file header changed:\n%s", src)
	}
}

func TestRenderWepTypeGo_MissingParamRowIsFatal(t *testing.T) {
	_, err := renderWepTypeGo(
		[]weapon{{rid: 0x000003E8, wepType: 1}},
		map[uint32]bool{0x000003E8: true, 0x0000044C: true},
		weaponFixture,
	)
	if err == nil {
		t.Fatal("expected an error for an app item ID with no EquipParamWeapon row")
	}
	if !strings.Contains(err.Error(), "0x0000044C") {
		t.Errorf("error does not name the missing ID: %v", err)
	}
}

func TestAppWeaponIDs_CoversEveryArmamentMap(t *testing.T) {
	ids := appWeaponIDs()
	want := len(data.Weapons) + len(data.Shields) + len(data.RangedAndCatalysts)
	if len(ids) != want {
		t.Fatalf("appWeaponIDs()=%d, want %d (maps must not overlap)", len(ids), want)
	}
	for _, id := range []uint32{0x000F4240, 0x01C9C380, 0x02719C40} { // Dagger, Buckler, Longbow
		if !ids[id] {
			t.Errorf("0x%08X missing from appWeaponIDs()", id)
		}
	}
}
