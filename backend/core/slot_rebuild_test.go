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

func TestRebuildSlotNilGuard(t *testing.T) {
	if _, err := RebuildSlot(nil); err == nil {
		t.Fatal("RebuildSlot(nil) should error")
	}
	bad := &SaveSlot{Data: make([]byte, 10)}
	if _, err := RebuildSlot(bad); err == nil {
		t.Fatal("RebuildSlot with wrong-size Data should error")
	}
}
