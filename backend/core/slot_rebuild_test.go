package core

import (
	"bytes"
	"os"
	"testing"
)

// rebuildSlotIdentity asserts that RebuildSlot on an unmodified slot
// produces a byte-for-byte copy of slot.Data. This guards against
// unintended drift as the implementation is fleshed out.
func rebuildSlotIdentity(t *testing.T, savePath string, expectedPlatform Platform) {
	t.Helper()
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", savePath)
	}

	save, err := LoadSave(savePath)
	if err != nil {
		t.Fatalf("LoadSave(%q): %v", savePath, err)
	}
	if save.Platform != expectedPlatform {
		t.Fatalf("expected platform %s, got %s", expectedPlatform, save.Platform)
	}

	checked := 0
	for i := 0; i < 10; i++ {
		slot := &save.Slots[i]
		if slot.Version == 0 {
			continue
		}
		rebuilt, err := RebuildSlot(slot)
		if err != nil {
			t.Errorf("slot %d: RebuildSlot: %v", i, err)
			continue
		}
		if len(rebuilt) != SlotSize {
			t.Errorf("slot %d: rebuilt size %d, want %d", i, len(rebuilt), SlotSize)
			continue
		}
		if !bytes.Equal(rebuilt, slot.Data) {
			diff := firstDiff(rebuilt, slot.Data)
			t.Errorf("slot %d: rebuilt != slot.Data; first diff at offset 0x%X", i, diff)
			continue
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("no active slots checked in %s", savePath)
	}
	t.Logf("identity round-trip OK for %d active slots in %s", checked, savePath)
}

func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

func TestRebuildSlotIdentityPS4(t *testing.T) {
	rebuildSlotIdentity(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestRebuildSlotIdentityPC(t *testing.T) {
	rebuildSlotIdentity(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}

func sectionMapInvariants(t *testing.T, savePath string, expectedPlatform Platform) {
	t.Helper()
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", savePath)
	}
	save, err := LoadSave(savePath)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if save.Platform != expectedPlatform {
		t.Fatalf("expected platform %s, got %s", expectedPlatform, save.Platform)
	}

	for i := 0; i < 10; i++ {
		slot := &save.Slots[i]
		if len(slot.SectionMap) == 0 {
			t.Errorf("slot %d (version %d): empty SectionMap", i, slot.Version)
			continue
		}
		// Validation already runs inside buildSectionMap, but re-check here so
		// regressions in the invariants surface as test failures.
		if err := validateSectionMap(slot.SectionMap); err != nil {
			t.Errorf("slot %d: %v; map=%+v", i, err, slot.SectionMap)
			continue
		}

		// Empty slots collapse to a single covering section.
		if slot.Version == 0 {
			if len(slot.SectionMap) != 1 || slot.SectionMap[0].Name != SectionEmptySlot {
				t.Errorf("slot %d (empty): expected single empty_slot section, got %+v", i, slot.SectionMap)
			}
			continue
		}

		// Active slots must produce 5 named sections in canonical order.
		want := []string{
			SectionPreUnlockedRegs,
			SectionUnlockedRegs,
			SectionPostUnlockedRegs,
			SectionDLC,
			SectionHash,
		}
		if len(slot.SectionMap) != len(want) {
			t.Errorf("slot %d: got %d sections, want %d", i, len(slot.SectionMap), len(want))
			continue
		}
		for j, name := range want {
			if slot.SectionMap[j].Name != name {
				t.Errorf("slot %d section[%d]: got %q, want %q", i, j, slot.SectionMap[j].Name, name)
			}
		}

		// unlocked_regions size must match parsed UnlockedRegions length.
		regions := slot.SectionMap[1]
		wantSize := 4 + 4*len(slot.UnlockedRegions)
		if regions.Size() != wantSize {
			t.Errorf("slot %d unlocked_regions size %d, want %d (count=%d)",
				i, regions.Size(), wantSize, len(slot.UnlockedRegions))
		}
	}
}

func TestSectionMapPS4(t *testing.T) {
	sectionMapInvariants(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestSectionMapPC(t *testing.T) {
	sectionMapInvariants(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}

func TestRebuildSlotNilGuard(t *testing.T) {
	if _, err := RebuildSlot(nil); err == nil {
		t.Fatal("RebuildSlot(nil) should error")
	}
	bad := &SaveSlot{Data: make([]byte, 10)}
	if _, err := RebuildSlot(bad); err == nil {
		t.Fatal("RebuildSlot with wrong-size Data should error")
	}
}
