package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
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
		25: 15, 28: 16, 32: 17, 29: 18, 33: 18, 31: 19, 35: 20,
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
