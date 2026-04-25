package core

import (
	"bytes"
	"os"
	"testing"
)

func worldGeomBlockEnd(t *testing.T, slot *SaveSlot) int {
	t.Helper()
	start := eventFlagsBlockEnd(t, slot)
	r := NewReader(slot.Data)
	if _, err := r.Seek(int64(start), 0); err != nil {
		t.Fatalf("seek: %v", err)
	}
	var wgb WorldGeomBlock
	if err := wgb.Read(r); err != nil {
		t.Fatalf("WorldGeomBlock.Read: %v", err)
	}
	return r.Pos()
}

func roundTripPlayerCoords(t *testing.T, savePath string, expectedPlatform Platform) {
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
		start := worldGeomBlockEnd(t, slot)

		r := NewReader(slot.Data)
		if _, err := r.Seek(int64(start), 0); err != nil {
			t.Fatalf("slot %d: seek: %v", i, err)
		}
		var pc PlayerCoordinates
		if err := pc.Read(r); err != nil {
			t.Errorf("slot %d: PlayerCoordinates.Read: %v", i, err)
			continue
		}
		var sp SpawnPointBlock
		if err := sp.Read(r, slot.Version); err != nil {
			t.Errorf("slot %d: SpawnPointBlock.Read: %v", i, err)
			continue
		}
		end := r.Pos()

		original := slot.Data[start:end]
		w := NewSectionWriter(end - start)
		pc.Write(w)
		sp.Write(w)

		if w.Len() != end-start {
			t.Errorf("slot %d: serialized %d, original %d", i, w.Len(), end-start)
			continue
		}
		if !bytes.Equal(w.Bytes(), original) {
			diff := firstDiff(w.Bytes(), original)
			t.Errorf("slot %d: player_coords+spawn round-trip mismatch at offset %d", i, diff)
			continue
		}
		t.Logf("slot %d (v=%d): coords=%d spawn=%d total=%d",
			i, slot.Version, PlayerCoordinatesSize, sp.ByteSize(), end-start)
		checked++
	}
	if checked == 0 {
		t.Fatalf("no slots verified for %s", savePath)
	}
}

func TestPlayerCoordsRoundTripPS4(t *testing.T) {
	roundTripPlayerCoords(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestPlayerCoordsRoundTripPC(t *testing.T) {
	roundTripPlayerCoords(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}
