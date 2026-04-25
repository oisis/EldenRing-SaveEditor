package core

import (
	"bytes"
	"os"
	"testing"
)

// worldHeadStart returns the byte offset within slot.Data where the
// post-region "horse / blood_stain / scalars" block begins.
// = unlocked_regions_end = UnlockedRegionsOffset + 4 + 4*len(UnlockedRegions)
func worldHeadStart(slot *SaveSlot) int {
	return slot.UnlockedRegionsOffset + 4 + 4*len(slot.UnlockedRegions)
}

func roundTripWorldHead(t *testing.T, savePath string, expectedPlatform Platform) {
	t.Helper()
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Skipf("Test save not found: %s", savePath)
	}
	save, err := LoadSave(savePath)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if save.Platform != expectedPlatform {
		t.Fatalf("expected %s, got %s", expectedPlatform, save.Platform)
	}

	checked := 0
	for i := 0; i < 10; i++ {
		slot := &save.Slots[i]
		if slot.Version == 0 || slot.UnlockedRegionsOffset == 0 {
			continue
		}
		start := worldHeadStart(slot)
		end := start + WorldHeadSize
		if end > len(slot.Data) {
			t.Errorf("slot %d: WorldHead end 0x%X exceeds slot.Data", i, end)
			continue
		}
		original := slot.Data[start:end]

		r := NewReader(slot.Data)
		if _, err := r.Seek(int64(start), 0); err != nil {
			t.Fatalf("slot %d: seek: %v", i, err)
		}
		var head WorldHead
		if err := head.Read(r); err != nil {
			t.Errorf("slot %d: WorldHead.Read: %v", i, err)
			continue
		}

		w := NewSectionWriter(WorldHeadSize)
		head.Write(w)
		got := w.Bytes()

		if len(got) != WorldHeadSize {
			t.Errorf("slot %d: serialized size %d, want %d", i, len(got), WorldHeadSize)
			continue
		}
		if !bytes.Equal(got, original) {
			diff := firstDiff(got, original)
			t.Errorf("slot %d: WorldHead round-trip mismatch at offset %d", i, diff)
			continue
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("no slots verified for %s", savePath)
	}
	t.Logf("WorldHead round-trip OK for %d active slots in %s", checked, savePath)
}

func TestWorldHeadRoundTripPS4(t *testing.T) {
	roundTripWorldHead(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestWorldHeadRoundTripPC(t *testing.T) {
	roundTripWorldHead(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}
