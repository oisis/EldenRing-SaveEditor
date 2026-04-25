package core

import (
	"bytes"
	"os"
	"testing"
)

func playerCoordsEnd(t *testing.T, slot *SaveSlot) int {
	t.Helper()
	start := worldGeomBlockEnd(t, slot)
	r := NewReader(slot.Data)
	if _, err := r.Seek(int64(start), 0); err != nil {
		t.Fatalf("seek: %v", err)
	}
	var pc PlayerCoordinates
	if err := pc.Read(r); err != nil {
		t.Fatalf("PlayerCoordinates.Read: %v", err)
	}
	var sp SpawnPointBlock
	if err := sp.Read(r, slot.Version); err != nil {
		t.Fatalf("SpawnPointBlock.Read: %v", err)
	}
	return r.Pos()
}

func roundTripNetMan(t *testing.T, savePath string, expectedPlatform Platform) {
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
		start := playerCoordsEnd(t, slot)

		r := NewReader(slot.Data)
		if _, err := r.Seek(int64(start), 0); err != nil {
			t.Fatalf("slot %d: seek: %v", i, err)
		}
		var nm NetMan
		if err := nm.Read(r); err != nil {
			t.Errorf("slot %d: NetMan.Read: %v", i, err)
			continue
		}
		end := r.Pos()
		if end-start != NetManSize {
			t.Errorf("slot %d: NetMan size %d, want %d", i, end-start, NetManSize)
			continue
		}

		original := slot.Data[start:end]
		w := NewSectionWriter(NetManSize)
		nm.Write(w)

		if !bytes.Equal(w.Bytes(), original) {
			diff := firstDiff(w.Bytes(), original)
			t.Errorf("slot %d: NetMan round-trip mismatch at offset %d", i, diff)
			continue
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("no slots verified for %s", savePath)
	}
	t.Logf("NetMan round-trip OK for %d active slots in %s (%d bytes each)", checked, savePath, NetManSize)
}

func TestNetManRoundTripPS4(t *testing.T) {
	roundTripNetMan(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestNetManRoundTripPC(t *testing.T) {
	roundTripNetMan(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}
