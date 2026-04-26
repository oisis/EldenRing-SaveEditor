package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/oisis/EldenRing-SaveEditor/backend/core"
)

// TestApplyMirrorFavoriteToCharacter verifies that applying Mirror Favorites slot 0
// to character slot 4 of ER0000-before.sl2 reproduces ER0000-after.sl2's FaceData.
//
// The reference saves capture an in-game M→F apply: a male character (slot 4, Roundtable
// default) with Mirror Favorites preset 0 (female) applied via the in-game mirror.
// Source: tmp/re-character/findings.md
func TestApplyMirrorFavoriteToCharacter(t *testing.T) {
	const (
		beforePath = "tmp/re-character/ER0000-before.sl2"
		afterPath  = "tmp/re-character/ER0000-after.sl2"
		charIdx    = 4
		mirrorIdx  = 0
	)

	if _, err := os.Stat(beforePath); os.IsNotExist(err) {
		t.Skipf("Reference save missing: %s", beforePath)
	}
	if _, err := os.Stat(afterPath); os.IsNotExist(err) {
		t.Skipf("Reference save missing: %s", afterPath)
	}

	before, err := core.LoadSave(beforePath)
	if err != nil {
		t.Fatalf("load before: %v", err)
	}
	ref, err := core.LoadSave(afterPath)
	if err != nil {
		t.Fatalf("load after: %v", err)
	}

	// Sanity: pre-conditions match the recorded scenario.
	if got := before.Slots[charIdx].Player.Gender; got != 1 {
		t.Fatalf("before slot %d Gender=%d, want 1 (male)", charIdx, got)
	}
	if got := ref.Slots[charIdx].Player.Gender; got != 0 {
		t.Fatalf("after slot %d Gender=%d, want 0 (female)", charIdx, got)
	}

	// Snapshot before's unk0x6c so we can verify it's preserved by apply.
	beforeFD := before.Slots[charIdx].FaceDataStart()
	preservedUnk := append([]byte(nil),
		before.Slots[charIdx].Data[beforeFD+core.FDOffUnknownBlock:beforeFD+core.FDOffUnknownBlock+64]...)

	app := &App{save: before}
	if err := app.ApplyMirrorFavoriteToCharacter(charIdx, mirrorIdx); err != nil {
		t.Fatalf("ApplyMirrorFavoriteToCharacter: %v", err)
	}

	got := &before.Slots[charIdx]
	want := &ref.Slots[charIdx]
	gotFD := got.FaceDataStart()
	wantFD := want.FaceDataStart()

	// Gender must flip to female.
	if got.Player.Gender != 0 {
		t.Errorf("Gender after apply: got %d, want 0 (female)", got.Player.Gender)
	}

	// FaceData segments that the game replaces verbatim.
	segments := []struct {
		name   string
		offset int
		size   int
	}{
		{"ModelIDs", core.FDOffFaceModel, 32},
		{"FaceShape", core.FDOffFaceShape, 64},
		{"Body", core.FDOffHead, 7},
		{"Skin", core.FDOffSkinR, 91},
	}
	for _, s := range segments {
		gb := got.Data[gotFD+s.offset : gotFD+s.offset+s.size]
		wb := want.Data[wantFD+s.offset : wantFD+s.offset+s.size]
		if !bytes.Equal(gb, wb) {
			t.Errorf("%s @ 0x%X: bytes differ\n  got=%X\n want=%X", s.name, s.offset, gb, wb)
		}
	}

	// unk0x6c must be preserved (game does NOT overwrite it on apply).
	gotUnk := got.Data[gotFD+core.FDOffUnknownBlock : gotFD+core.FDOffUnknownBlock+64]
	if !bytes.Equal(gotUnk, preservedUnk) {
		t.Errorf("unk0x6c was modified — should be preserved unchanged")
	}

	// Trailing flags: 0x124 stays 0x01, 0x125 and 0x126 zero out (per facedata_dump.txt).
	if got.Data[gotFD+0x124] != 0x01 {
		t.Errorf("flag@0x124: got 0x%02X, want 0x01", got.Data[gotFD+0x124])
	}
	if got.Data[gotFD+0x125] != 0x00 {
		t.Errorf("flag@0x125: got 0x%02X, want 0x00", got.Data[gotFD+0x125])
	}
	if got.Data[gotFD+0x126] != 0x00 {
		t.Errorf("flag@0x126: got 0x%02X, want 0x00", got.Data[gotFD+0x126])
	}
}

func TestApplyMirrorFavoriteToCharacter_Errors(t *testing.T) {
	app := &App{}
	if err := app.ApplyMirrorFavoriteToCharacter(0, 0); err == nil {
		t.Error("no save loaded: expected error")
	}

	const beforePath = "tmp/re-character/ER0000-before.sl2"
	if _, err := os.Stat(beforePath); os.IsNotExist(err) {
		t.Skipf("Reference save missing: %s", beforePath)
	}
	save, err := core.LoadSave(beforePath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	app.save = save

	for _, ci := range []int{-1, 10, 99} {
		if err := app.ApplyMirrorFavoriteToCharacter(ci, 0); err == nil {
			t.Errorf("charIndex=%d: expected error", ci)
		}
	}
	for _, mi := range []int{-1, 15, 99} {
		if err := app.ApplyMirrorFavoriteToCharacter(4, mi); err == nil {
			t.Errorf("mirrorIndex=%d: expected error", mi)
		}
	}
}
