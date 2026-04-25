package core

import (
	"bytes"
	"os"
	"testing"
)

// eventFlagsBlockEnd advances a Reader through everything up to and including
// PreEventFlagsScalars + EventFlagsBlock and returns position of the next byte.
func eventFlagsBlockEnd(t *testing.T, slot *SaveSlot) int {
	t.Helper()
	start := menuBlockEnd(t, slot)
	r := NewReader(slot.Data)
	if _, err := r.Seek(int64(start), 0); err != nil {
		t.Fatalf("seek: %v", err)
	}
	var s PreEventFlagsScalars
	if err := s.Read(r); err != nil {
		t.Fatalf("PreEventFlagsScalars.Read: %v", err)
	}
	var ef EventFlagsBlock
	if err := ef.Read(r); err != nil {
		t.Fatalf("EventFlagsBlock.Read: %v", err)
	}
	return r.Pos()
}

func roundTripWorldGeomBlock(t *testing.T, savePath string, expectedPlatform Platform) {
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
		start := eventFlagsBlockEnd(t, slot)

		r := NewReader(slot.Data)
		if _, err := r.Seek(int64(start), 0); err != nil {
			t.Fatalf("slot %d: seek: %v", i, err)
		}
		var wgb WorldGeomBlock
		if err := wgb.Read(r); err != nil {
			t.Errorf("slot %d: WorldGeomBlock.Read: %v", i, err)
			continue
		}
		end := r.Pos()

		original := slot.Data[start:end]
		w := NewSectionWriter(end - start)
		wgb.Write(w)

		if w.Len() != end-start {
			t.Errorf("slot %d: serialized %d, original %d", i, w.Len(), end-start)
			continue
		}
		if !bytes.Equal(w.Bytes(), original) {
			diff := firstDiff(w.Bytes(), original)
			t.Errorf("slot %d: WorldGeomBlock round-trip mismatch at offset %d", i, diff)
			continue
		}
		t.Logf("slot %d: field=%d world=%d geom=%d geom2=%d rend=%d total=%d",
			i, wgb.FieldArea.ByteSize(), wgb.WorldArea.ByteSize(),
			wgb.WorldGeomMan.ByteSize(), wgb.WorldGeomMan2.ByteSize(),
			wgb.RendMan.ByteSize(), end-start)
		checked++
	}
	if checked == 0 {
		t.Fatalf("no slots verified for %s", savePath)
	}
}

func TestWorldGeomBlockPS4(t *testing.T) {
	roundTripWorldGeomBlock(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestWorldGeomBlockPC(t *testing.T) {
	roundTripWorldGeomBlock(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}
