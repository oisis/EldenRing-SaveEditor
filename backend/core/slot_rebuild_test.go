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

// TestRebuildSlotMutationPC verifies that mutating slot.UnlockedRegions and
// rebuilding produces a slot whose post-mutation parse round-trips the new
// region list while preserving everything else.
//
// PC saves have ~419KB of tail rest, so this test exercises the "grow"
// path with abundant slack. PS4 saves have zero rest and are exercised in
// a separate test below (with a shrink mutation that must succeed regardless
// of slack).
func TestRebuildSlotMutationPC(t *testing.T) {
	path := "../../tmp/save/ER0000.sl2"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", path)
	}
	save, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// Pick the first active slot.
	var slotIdx = -1
	for i := 0; i < 10; i++ {
		if save.Slots[i].Version != 0 && save.Slots[i].UnlockedRegionsOffset != 0 {
			slotIdx = i
			break
		}
	}
	if slotIdx < 0 {
		t.Skip("no active slot")
	}
	slot := &save.Slots[slotIdx]
	originalCount := len(slot.UnlockedRegions)
	originalCopy := append([]uint32(nil), slot.UnlockedRegions...)

	// Append 50 synthetic region IDs (well within PC slack).
	for k := uint32(0); k < 50; k++ {
		slot.UnlockedRegions = append(slot.UnlockedRegions, 0xDEAD0000+k)
	}

	rebuilt, err := RebuildSlot(slot)
	if err != nil {
		t.Fatalf("RebuildSlot: %v", err)
	}
	if len(rebuilt) != SlotSize {
		t.Fatalf("rebuilt size %d, want %d", len(rebuilt), SlotSize)
	}

	// Re-parse the rebuilt slot via a fresh SaveSlot.
	var verify SaveSlot
	verify.Data = rebuilt
	verify.Version = slot.Version
	r := NewReader(rebuilt)
	if _, err := r.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := verify.Read(NewReader(rebuilt), string(save.Platform)); err != nil {
		t.Fatalf("re-parse rebuilt: %v", err)
	}
	if len(verify.UnlockedRegions) != originalCount+50 {
		t.Errorf("re-parsed UnlockedRegions count %d, want %d",
			len(verify.UnlockedRegions), originalCount+50)
	}
	// First N must equal original list, last 50 must equal what we appended.
	for i := 0; i < originalCount; i++ {
		if verify.UnlockedRegions[i] != originalCopy[i] {
			t.Errorf("region[%d]: %x, want %x", i, verify.UnlockedRegions[i], originalCopy[i])
			break
		}
	}
	for k := uint32(0); k < 50; k++ {
		if got := verify.UnlockedRegions[originalCount+int(k)]; got != 0xDEAD0000+k {
			t.Errorf("appended region[%d]: %x, want %x", k, got, 0xDEAD0000+k)
			break
		}
	}

	// Sanity: a couple of unrelated fields must survive verbatim.
	if verify.Player.Level != slot.Player.Level {
		t.Errorf("Player.Level changed: %d → %d", slot.Player.Level, verify.Player.Level)
	}
	if verify.Player.Souls != slot.Player.Souls {
		t.Errorf("Player.Souls changed: %d → %d", slot.Player.Souls, verify.Player.Souls)
	}
}

// TestRebuildSlotShrinkPS4 verifies that REMOVING regions works on PS4 saves
// (which have zero tail slack) — the rebuilt slot must zero-pad the gap and
// still re-parse correctly.
func TestRebuildSlotShrinkPS4(t *testing.T) {
	path := "../../tmp/save/oisis_pl-org.txt"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", path)
	}
	save, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	// Pick the slot with the most regions to maximise the shrink delta.
	var slotIdx = -1
	maxRegs := 0
	for i := 0; i < 10; i++ {
		if save.Slots[i].Version != 0 && len(save.Slots[i].UnlockedRegions) > maxRegs {
			slotIdx = i
			maxRegs = len(save.Slots[i].UnlockedRegions)
		}
	}
	if slotIdx < 0 || maxRegs < 10 {
		t.Skipf("no slot with >=10 regions (max=%d)", maxRegs)
	}
	slot := &save.Slots[slotIdx]

	// Drop the last 5 regions.
	slot.UnlockedRegions = slot.UnlockedRegions[:len(slot.UnlockedRegions)-5]
	wantCount := len(slot.UnlockedRegions)

	rebuilt, err := RebuildSlot(slot)
	if err != nil {
		t.Fatalf("RebuildSlot: %v", err)
	}

	var verify SaveSlot
	if err := verify.Read(NewReader(rebuilt), string(save.Platform)); err != nil {
		t.Fatalf("re-parse rebuilt: %v", err)
	}
	if len(verify.UnlockedRegions) != wantCount {
		t.Errorf("re-parsed UnlockedRegions count %d, want %d",
			len(verify.UnlockedRegions), wantCount)
	}
	if verify.Player.Level != slot.Player.Level {
		t.Errorf("Player.Level changed after shrink")
	}
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
