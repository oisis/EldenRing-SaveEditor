package core

import (
	"bytes"
	"os"
	"testing"
)

// Round-trip the menu/trophy/gaitem/tutorial block on real save data.
// Start position: end of WorldHead (= regions_end + WorldHeadSize).
func roundTripMenuBlock(t *testing.T, savePath string, expectedPlatform Platform) {
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
		start := worldHeadStart(slot) + WorldHeadSize

		r := NewReader(slot.Data)
		if _, err := r.Seek(int64(start), 0); err != nil {
			t.Fatalf("slot %d: seek: %v", i, err)
		}

		var menu MenuSaveLoad
		if err := menu.Read(r); err != nil {
			t.Errorf("slot %d: MenuSaveLoad.Read: %v", i, err)
			continue
		}
		var trophy TrophyEquipData
		if err := trophy.Read(r); err != nil {
			t.Errorf("slot %d: TrophyEquipData.Read: %v", i, err)
			continue
		}
		var gaitem GaitemGameData
		if err := gaitem.Read(r); err != nil {
			t.Errorf("slot %d: GaitemGameData.Read: %v", i, err)
			continue
		}
		var tut TutorialData
		if err := tut.Read(r); err != nil {
			t.Errorf("slot %d: TutorialData.Read: %v", i, err)
			continue
		}
		end := r.Pos()

		original := slot.Data[start:end]

		w := NewSectionWriter(end - start)
		menu.Write(w)
		trophy.Write(w)
		gaitem.Write(w)
		tut.Write(w)

		if w.Len() != end-start {
			t.Errorf("slot %d: serialized %d bytes, original block %d bytes",
				i, w.Len(), end-start)
			continue
		}
		if !bytes.Equal(w.Bytes(), original) {
			diff := firstDiff(w.Bytes(), original)
			t.Errorf("slot %d: menu/trophy/gaitem/tutorial round-trip mismatch at offset %d (block start 0x%X)",
				i, diff, start)
			continue
		}
		t.Logf("slot %d: menu=%d trophy=52 gaitem=%d tut=%d total=%d",
			i, menu.ByteSize(), GaitemGameDataSize, tut.ByteSize(), end-start)
		checked++
	}
	if checked == 0 {
		t.Fatalf("no slots verified for %s", savePath)
	}
}

func TestMenuBlockRoundTripPS4(t *testing.T) {
	roundTripMenuBlock(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestMenuBlockRoundTripPC(t *testing.T) {
	roundTripMenuBlock(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}
