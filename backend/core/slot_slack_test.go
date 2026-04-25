package core

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestSlotSlackAnalysis is a diagnostic test (not a correctness test).
// It prints, for every active slot in the test saves:
//   - the size of the post_unlocked_regions blob,
//   - the count of trailing zero bytes inside that blob,
//   - and the resulting "extra regions that fit without losing data"
//     (trailing_zeros / 4 — each region ID is 4 bytes).
//
// Findings drive Step 3 of the slot rebuild plan; results are summarised
// in spec/30-slot-rebuild-research.md.
func TestSlotSlackAnalysis(t *testing.T) {
	saves := []struct {
		path     string
		platform Platform
	}{
		{"../../tmp/save/oisis_pl-org.txt", PlatformPS},
		{"../../tmp/save/oisisk_ps4.txt", PlatformPS},
		{"../../tmp/save/ER0000.sl2", PlatformPC},
	}

	t.Logf("Slack analysis (DlcSectionOffset = 0x%X, HashOffset = 0x%X, SlotSize = 0x%X):",
		DlcSectionOffset, HashOffset, SlotSize)
	t.Logf("%-26s %-4s %-7s %-9s %-12s %-14s %-12s %-10s %-12s %-12s %-12s",
		"save", "slot", "version", "regions", "regs_end", "post_blob_sz",
		"trail_zero", "extra_fits", "last_nz_pre", "last_nz_dlc", "last_nz_all")
	t.Logf("%s", strings.Repeat("-", 160))

	for _, sv := range saves {
		if _, err := os.Stat(sv.path); os.IsNotExist(err) {
			t.Logf("SKIP %s (not present)", sv.path)
			continue
		}
		save, err := LoadSave(sv.path)
		if err != nil {
			t.Errorf("LoadSave(%s): %v", sv.path, err)
			continue
		}
		for i := 0; i < 10; i++ {
			slot := &save.Slots[i]
			if slot.Version == 0 {
				continue
			}
			regsStart := slot.UnlockedRegionsOffset
			regsCount := len(slot.UnlockedRegions)
			regsEnd := regsStart + 4 + 4*regsCount
			postSize := DlcSectionOffset - regsEnd

			trailingZeros := 0
			for j := DlcSectionOffset - 1; j >= regsEnd; j-- {
				if slot.Data[j] != 0 {
					break
				}
				trailingZeros++
			}
			extraFits := trailingZeros / 4

			lastNZPreDLC := lastNonZero(slot.Data[:DlcSectionOffset])
			lastNZPreHash := lastNonZero(slot.Data[:HashOffset])
			lastNZAll := lastNonZero(slot.Data[:SlotSize])

			tag := shortName(sv.path)
			t.Logf("%-26s %-4d %-7d %-9d 0x%-10X %-14d %-12d %-10d 0x%-10X 0x%-10X 0x%-10X",
				tag, i, slot.Version, regsCount, regsEnd, postSize,
				trailingZeros, extraFits, lastNZPreDLC, lastNZPreHash, lastNZAll)
		}
	}
}

func lastNonZero(data []byte) int {
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] != 0 {
			return i
		}
	}
	return -1
}

func shortName(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// fmt is imported to avoid an unused import if logging is removed.
var _ = fmt.Sprintf
