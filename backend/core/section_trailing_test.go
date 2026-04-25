package core

import (
	"bytes"
	"os"
	"testing"
)

func netManEnd(t *testing.T, slot *SaveSlot) int {
	t.Helper()
	start := playerCoordsEnd(t, slot)
	r := NewReader(slot.Data)
	if _, err := r.Seek(int64(start), 0); err != nil {
		t.Fatalf("seek: %v", err)
	}
	var nm NetMan
	if err := nm.Read(r); err != nil {
		t.Fatalf("NetMan.Read: %v", err)
	}
	return r.Pos()
}

func roundTripTrailing(t *testing.T, savePath string, expectedPlatform Platform) {
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
		start := netManEnd(t, slot)

		r := NewReader(slot.Data)
		if _, err := r.Seek(int64(start), 0); err != nil {
			t.Fatalf("slot %d: seek: %v", i, err)
		}
		var tb TrailingFixedBlock
		if err := tb.Read(r); err != nil {
			t.Errorf("slot %d: TrailingFixedBlock.Read: %v", i, err)
			continue
		}
		end := r.Pos()
		if end-start != TrailingFixedBlockSize {
			t.Errorf("slot %d: trailing size %d, want %d", i, end-start, TrailingFixedBlockSize)
			continue
		}

		original := slot.Data[start:end]
		w := NewSectionWriter(TrailingFixedBlockSize)
		tb.Write(w)
		if !bytes.Equal(w.Bytes(), original) {
			diff := firstDiff(w.Bytes(), original)
			t.Errorf("slot %d: trailing round-trip mismatch at offset %d", i, diff)
			continue
		}
		t.Logf("slot %d: weather/time/base/steam/ps5/dlc total=%d", i, end-start)
		checked++
	}
	if checked == 0 {
		t.Fatalf("no slots verified for %s", savePath)
	}
}

func TestTrailingRoundTripPS4(t *testing.T) {
	roundTripTrailing(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestTrailingRoundTripPC(t *testing.T) {
	roundTripTrailing(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}
