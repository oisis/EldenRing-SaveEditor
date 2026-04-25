package core

import (
	"bytes"
	"os"
	"testing"
)

// menuBlockEnd advances a fresh Reader from worldHeadStart through
// WorldHead + Menu/Trophy/Gaitem/Tutorial and returns the position of the
// first byte of PreEventFlagsScalars.
func menuBlockEnd(t *testing.T, slot *SaveSlot) int {
	t.Helper()
	r := NewReader(slot.Data)
	if _, err := r.Seek(int64(worldHeadStart(slot)), 0); err != nil {
		t.Fatalf("seek: %v", err)
	}
	var head WorldHead
	if err := head.Read(r); err != nil {
		t.Fatalf("WorldHead.Read: %v", err)
	}
	var menu MenuSaveLoad
	if err := menu.Read(r); err != nil {
		t.Fatalf("MenuSaveLoad.Read: %v", err)
	}
	var trophy TrophyEquipData
	if err := trophy.Read(r); err != nil {
		t.Fatalf("TrophyEquipData.Read: %v", err)
	}
	var gaitem GaitemGameData
	if err := gaitem.Read(r); err != nil {
		t.Fatalf("GaitemGameData.Read: %v", err)
	}
	var tut TutorialData
	if err := tut.Read(r); err != nil {
		t.Fatalf("TutorialData.Read: %v", err)
	}
	return r.Pos()
}

func roundTripEventFlagsBlock(t *testing.T, savePath string, expectedPlatform Platform) {
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
		start := menuBlockEnd(t, slot)

		r := NewReader(slot.Data)
		if _, err := r.Seek(int64(start), 0); err != nil {
			t.Fatalf("slot %d: seek: %v", i, err)
		}
		var scalars PreEventFlagsScalars
		if err := scalars.Read(r); err != nil {
			t.Errorf("slot %d: scalars.Read: %v", i, err)
			continue
		}
		var ef EventFlagsBlock
		if err := ef.Read(r); err != nil {
			t.Errorf("slot %d: event_flags.Read: %v", i, err)
			continue
		}
		end := r.Pos()
		expectedSize := PreEventFlagsScalarsSize + EventFlagsBlockSize
		if end-start != expectedSize {
			t.Errorf("slot %d: block size %d, want %d", i, end-start, expectedSize)
		}

		original := slot.Data[start:end]
		w := NewSectionWriter(end - start)
		scalars.Write(w)
		ef.Write(w)

		if !bytes.Equal(w.Bytes(), original) {
			diff := firstDiff(w.Bytes(), original)
			t.Errorf("slot %d: scalars+event_flags round-trip mismatch at offset %d", i, diff)
			continue
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("no slots verified for %s", savePath)
	}
	t.Logf("scalars+event_flags round-trip OK for %d active slots in %s", checked, savePath)
}

func TestEventFlagsBlockPS4(t *testing.T) {
	roundTripEventFlagsBlock(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestEventFlagsBlockPC(t *testing.T) {
	roundTripEventFlagsBlock(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}
