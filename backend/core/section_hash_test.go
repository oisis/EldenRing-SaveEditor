package core

import (
	"bytes"
	"os"
	"testing"
)

// trailingBlockEnd is the offset right after TrailingFixedBlock (= start of hash).
func trailingBlockEnd(t *testing.T, slot *SaveSlot) int {
	t.Helper()
	start := netManEnd(t, slot)
	r := NewReader(slot.Data)
	if _, err := r.Seek(int64(start), 0); err != nil {
		t.Fatalf("seek: %v", err)
	}
	var tb TrailingFixedBlock
	if err := tb.Read(r); err != nil {
		t.Fatalf("TrailingFixedBlock.Read: %v", err)
	}
	return r.Pos()
}

func roundTripHashAndRest(t *testing.T, savePath string, expectedPlatform Platform) {
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
		hashStart := trailingBlockEnd(t, slot)
		// Spec/22 says hash size is "slot_end - position"; in practice we observe
		// hashStart == HashOffset (= SlotSize - 0x80) and tail rest = 0 bytes.
		// Capture both invariants in this test.
		if hashStart != HashOffset {
			t.Logf("slot %d: hash starts at 0x%X (HashOffset 0x%X); rest=%d",
				i, hashStart, HashOffset, SlotSize-hashStart-PlayerGameDataHashSize)
		}

		r := NewReader(slot.Data)
		if _, err := r.Seek(int64(hashStart), 0); err != nil {
			t.Fatalf("slot %d: seek: %v", i, err)
		}
		var h PlayerGameDataHash
		if err := h.Read(r); err != nil {
			t.Errorf("slot %d: PlayerGameDataHash.Read: %v", i, err)
			continue
		}
		hashEnd := r.Pos()
		restSize := SlotSize - hashEnd
		var rest []byte
		if restSize > 0 {
			b, err := r.ReadBytes(restSize)
			if err != nil {
				t.Errorf("slot %d: tail rest: %v", i, err)
				continue
			}
			rest = append([]byte(nil), b...)
		}

		original := slot.Data[hashStart:SlotSize]
		w := NewSectionWriter(SlotSize - hashStart)
		h.Write(w)
		if len(rest) > 0 {
			w.WriteBytes(rest)
		}
		if !bytes.Equal(w.Bytes(), original) {
			diff := firstDiff(w.Bytes(), original)
			t.Errorf("slot %d: hash+rest round-trip mismatch at offset %d", i, diff)
			continue
		}
		t.Logf("slot %d: hash=128B rest=%dB total=%dB", i, restSize, len(w.Bytes()))
		checked++
	}
	if checked == 0 {
		t.Fatalf("no slots verified for %s", savePath)
	}
}

func TestHashAndRestRoundTripPS4(t *testing.T) {
	roundTripHashAndRest(t, "../../tmp/save/oisis_pl-org.txt", PlatformPS)
}

func TestHashAndRestRoundTripPC(t *testing.T) {
	roundTripHashAndRest(t, "../../tmp/save/ER0000.sl2", PlatformPC)
}
