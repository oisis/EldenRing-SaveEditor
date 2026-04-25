package core

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSetUnlockedRegionsInMemory verifies the in-memory mutation path:
// SetUnlockedRegions → state visible on the same slot pointer.
func TestSetUnlockedRegionsInMemory(t *testing.T) {
	path := "../../tmp/save/oisis_pl-org.txt"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", path)
	}
	save, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	slot := &save.Slots[0]
	if slot.Version == 0 {
		t.Skip("slot 0 empty")
	}

	originalLevel := slot.Player.Level
	want := []uint32{0x1000001, 0x1000002, 0x1000003, 0x1000001 /* dup */, 0x0FFFFFF}
	if err := SetUnlockedRegions(slot, want); err != nil {
		t.Fatalf("SetUnlockedRegions: %v", err)
	}
	// 5 input → 4 unique, sorted ascending.
	expectedSorted := []uint32{0x0FFFFFF, 0x1000001, 0x1000002, 0x1000003}
	if len(slot.UnlockedRegions) != len(expectedSorted) {
		t.Fatalf("len=%d, want %d", len(slot.UnlockedRegions), len(expectedSorted))
	}
	for i, v := range expectedSorted {
		if slot.UnlockedRegions[i] != v {
			t.Errorf("[%d]=%x, want %x", i, slot.UnlockedRegions[i], v)
		}
	}
	if slot.Player.Level != originalLevel {
		t.Errorf("Player.Level changed: %d → %d", originalLevel, slot.Player.Level)
	}
}

// TestSetUnlockedRegionsRoundTripPS4 verifies the full Set→Save→Load→Get path:
// after persisting via SaveFile and re-parsing the file, the slot's
// UnlockedRegions matches what we wrote.
func TestSetUnlockedRegionsRoundTripPS4(t *testing.T) {
	src := "../../tmp/save/oisis_pl-org.txt"
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", src)
	}
	save, err := LoadSave(src)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	slot := &save.Slots[0]
	if slot.Version == 0 {
		t.Skip("slot 0 empty")
	}
	originalLevel := slot.Player.Level

	// Add 20 synthetic regions at the start of the existing list.
	new := make([]uint32, 0, len(slot.UnlockedRegions)+20)
	for k := uint32(0); k < 20; k++ {
		new = append(new, 0x10000+k)
	}
	new = append(new, slot.UnlockedRegions...)
	if err := SetUnlockedRegions(slot, new); err != nil {
		t.Fatalf("SetUnlockedRegions: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "out.sl2")
	if err := save.SaveFile(tmp); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	reloaded, err := LoadSave(tmp)
	if err != nil {
		t.Fatalf("LoadSave reloaded: %v", err)
	}
	got := reloaded.Slots[0].UnlockedRegions
	if len(got) != len(slot.UnlockedRegions) {
		t.Fatalf("reloaded len=%d, want %d", len(got), len(slot.UnlockedRegions))
	}
	for i := range got {
		if got[i] != slot.UnlockedRegions[i] {
			t.Errorf("region[%d]=%x, want %x", i, got[i], slot.UnlockedRegions[i])
			break
		}
	}
	if reloaded.Slots[0].Player.Level != originalLevel {
		t.Errorf("Player.Level after round-trip changed: %d → %d",
			originalLevel, reloaded.Slots[0].Player.Level)
	}
}

// TestSetUnlockedRegionsAfterAddItem reproduces a real-world bug: the user
// added a weapon to inventory, then used Map → Reveal All (which adds DLC
// fragments via AddItemsToSlot), then unlocked all regions — the resulting
// save had a corrupted regCount because SetUnlockedRegions used stale
// UnlockedRegionsOffset (other writers don't refresh it).
//
// This test guards against regression: even when slot.Data has been mutated
// by another writer between Read and SetUnlockedRegions, the rebuild must
// still reload cleanly.
func TestSetUnlockedRegionsAfterAddItem(t *testing.T) {
	src := "../../tmp/save/ER0000.sl2"
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", src)
	}
	save, err := LoadSave(src)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	slot := &save.Slots[4]
	if slot.Version == 0 {
		t.Skip("slot 4 empty")
	}
	originalLevel := slot.Player.Level

	// Step 1: add a weapon (21B GaItem record). Slot.Data shifts by ~13 bytes,
	// but slot.UnlockedRegionsOffset is NOT refreshed.
	if err := AddItemsToSlot(slot, []uint32{0x80130880}, 1, 0, false); err != nil {
		t.Fatalf("AddItemsToSlot: %v", err)
	}

	// Step 2: SetUnlockedRegions — must internally refresh offsets before
	// rebuilding, or the resulting save is corrupted.
	merged := append([]uint32(nil), slot.UnlockedRegions...)
	for k := uint32(0); k < 75; k++ {
		merged = append(merged, 0x10000+k)
	}
	if err := SetUnlockedRegions(slot, merged); err != nil {
		t.Fatalf("SetUnlockedRegions: %v", err)
	}

	// Step 3: persist and reload — no warnings, level preserved, regions match.
	tmp := filepath.Join(t.TempDir(), "out.sl2")
	if err := save.SaveFile(tmp); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	reloaded, err := LoadSave(tmp)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	rs := &reloaded.Slots[4]
	if len(rs.Warnings) > 0 {
		t.Errorf("reloaded slot has warnings: %v", rs.Warnings)
	}
	if rs.Player.Level != originalLevel {
		t.Errorf("Player.Level changed: %d -> %d", originalLevel, rs.Player.Level)
	}
	if len(rs.UnlockedRegions) != len(slot.UnlockedRegions) {
		t.Errorf("regions count: got %d, want %d", len(rs.UnlockedRegions), len(slot.UnlockedRegions))
	}
}

func TestSetUnlockedRegionsRoundTripPC(t *testing.T) {
	src := "../../tmp/save/ER0000.sl2"
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", src)
	}
	save, err := LoadSave(src)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	// Find first active slot.
	slotIdx := -1
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
	originalLevel := slot.Player.Level
	originalSouls := slot.Player.Souls

	// Add 80 regions (well above any "extra_fits" measured in spec/30).
	new := make([]uint32, 0, len(slot.UnlockedRegions)+80)
	new = append(new, slot.UnlockedRegions...)
	for k := uint32(0); k < 80; k++ {
		new = append(new, 0x6E00000+k) // synthetic IDs that don't clash
	}
	if err := SetUnlockedRegions(slot, new); err != nil {
		t.Fatalf("SetUnlockedRegions: %v", err)
	}

	tmp := filepath.Join(t.TempDir(), "out.sl2")
	if err := save.SaveFile(tmp); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	reloaded, err := LoadSave(tmp)
	if err != nil {
		t.Fatalf("LoadSave reloaded: %v", err)
	}
	got := reloaded.Slots[slotIdx].UnlockedRegions
	if len(got) != len(slot.UnlockedRegions) {
		t.Fatalf("reloaded len=%d, want %d", len(got), len(slot.UnlockedRegions))
	}
	if reloaded.Slots[slotIdx].Player.Level != originalLevel {
		t.Errorf("Player.Level after round-trip: %d → %d", originalLevel, reloaded.Slots[slotIdx].Player.Level)
	}
	if reloaded.Slots[slotIdx].Player.Souls != originalSouls {
		t.Errorf("Player.Souls after round-trip: %d → %d", originalSouls, reloaded.Slots[slotIdx].Player.Souls)
	}
}
