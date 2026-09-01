package saveengine

import (
	"bytes"
	"encoding/binary"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Independent test constants for ApplyFavoritePreset layout and offsets.
const (
	applyFavSlotSize        = 0x130 // 304 bytes
	applyFavBaseOffset      = 0x154
	applyFavPCUserData10    = int64(0x300) + 10*0x280010 + 0x10
	applyFavPS4UserData10   = int64(0x70) + 10*0x280000
	applyFavFaceOpaqueStart = 0x70
	applyFavFaceOpaqueEnd   = 0xB0
	applyFavSexFlagsOffset  = 0x125
	applyFavSexFlagsSize    = 2
)

type byteRange struct {
	start int64
	end   int64
}

func applyFavUserData10Base(platform Platform) int64 {
	if platform == PlatformPS4 {
		return applyFavPS4UserData10
	}
	return applyFavPCUserData10
}

func applyFavSlotOffset(platform Platform, slot int) int64 {
	return applyFavUserData10Base(platform) + applyFavBaseOffset + int64(slot)*applyFavSlotSize
}

func inAllowedRanges(offset int64, ranges []byteRange) bool {
	for _, r := range ranges {
		if offset >= r.start && offset < r.end {
			return true
		}
	}
	return false
}

func writeApplyFavoriteFixture(
	t *testing.T,
	platform Platform,
	charSlot int,
	initialAppearance CharacterAppearanceValues,
	favSlot int,
	favBodyType byte,
	favModelIDs [8]uint32,
	favFaceShape [64]byte,
	favUnkBlock [64]byte,
	favBody [7]byte,
	favSkin [91]byte,
	favActive bool,
) (string, int64, int64, int64, int64) {
	t.Helper()

	path, genderAt, voiceAt, faceAt := writeSetAppearanceFixture(
		t, platform, initialAppearance)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	slotAt := applyFavSlotOffset(platform, favSlot)

	favBuf := make([]byte, applyFavSlotSize)
	if favActive {
		binary.LittleEndian.PutUint16(favBuf[0x00:], 0xFACE)
		binary.LittleEndian.PutUint32(favBuf[0x04:], 0x11D0)
		favBuf[0x08] = 1
		favBuf[0x09] = favBodyType
		copy(favBuf[0x18:], "FACE")
		binary.LittleEndian.PutUint32(favBuf[0x1C:], 4)
		binary.LittleEndian.PutUint32(favBuf[0x20:], 0x120)
		for m := 0; m < 8; m++ {
			binary.LittleEndian.PutUint32(favBuf[0x24+m*4:], favModelIDs[m])
		}
		copy(favBuf[0x44:], favFaceShape[:])
		copy(favBuf[0x84:], favUnkBlock[:])
		copy(favBuf[0xC4:], favBody[:])
		copy(favBuf[0xCB:], favSkin[:])
	}
	copy(data[slotAt:slotAt+applyFavSlotSize], favBuf)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}

	return path, genderAt, voiceAt, faceAt, slotAt
}

func TestApplyFavoritePresetWritesExactFieldsAndPreservesVoiceAndOpaqueBlock(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			initial := setAppearanceTestValues(0x11)
			initial.Gender = 1
			initial.VoiceType = 4

			var favModelIDs [8]uint32
			for i := 0; i < 8; i++ {
				favModelIDs[i] = uint32(0x1000 + i*7)
			}
			var favFaceShape [64]byte
			for i := 0; i < 64; i++ {
				favFaceShape[i] = byte(0x30 + i)
			}
			var favUnkBlock [64]byte
			for i := 0; i < 64; i++ {
				favUnkBlock[i] = 0xEE // Preset's opaque block — MUST NOT be copied to character
			}
			var favBody [7]byte
			for i := 0; i < 7; i++ {
				favBody[i] = byte(0x80 + i)
			}
			var favSkin [91]byte
			for i := 0; i < 91; i++ {
				favSkin[i] = byte(0x10 + (i % 50))
			}

			// Preset bodyType = 1 (female) -> applied character Gender must be 0 (female)
			path, genderAt, voiceAt, faceAt, slotAt := writeApplyFavoriteFixture(
				t, platform, setAppearanceTestSlot, initial,
				3, 1, favModelIDs, favFaceShape, favUnkBlock, favBody, favSkin, true,
			)

			fileBefore, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read file before: %v", err)
			}

			engine := New()
			loadedSession, err := engine.LoadSave(path, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			loaded := engine.sessions[loadedSession.SaveSessionID]
			snapshotBefore := bytes.Clone(loaded.snapshot.data)

			result, err := engine.ApplyFavoritePreset(
				loadedSession.SaveSessionID, setAppearanceTestSlot, 3, "0")
			if err != nil {
				t.Fatalf("ApplyFavoritePreset: %v", err)
			}
			assertCommittedReceipt(t, result.MutationReceipt, loadedSession.SaveSessionID,
				kindApplyFavoritePreset, "1")
			// The receipt is pinned from the result because operationID names one
			// execution and cannot be predicted; every other member is asserted above.
			wantResult := ApplyFavoritePresetResult{
				MutationReceipt: result.MutationReceipt,
				CharacterID:     setAppearanceTestSlot,
				FavoriteSlotID:  3,
			}
			if !reflect.DeepEqual(result, wantResult) {
				t.Errorf("result = %+v, want %+v", result, wantResult)
			}

			snapshotAfter := bytes.Clone(loaded.snapshot.data)

			// Byte-by-byte diff check: prove no bytes changed outside confirmed appearance fields
			allowedRanges := []byteRange{
				{start: genderAt, end: genderAt + 1},
				{start: faceAt + 0x10, end: faceAt + 0x70},  // Model IDs (0x10..0x30) + Face Shape (0x30..0x70)
				{start: faceAt + 0xB0, end: faceAt + 0x112}, // Body (0xB0..0xB7) + Skin (0xB7..0x112)
				{start: faceAt + applyFavSexFlagsOffset, end: faceAt + applyFavSexFlagsOffset + applyFavSexFlagsSize}, // Trailing sex flags (0x125..0x127)
			}
			for i := int64(0); i < int64(len(snapshotBefore)); i++ {
				if snapshotBefore[i] != snapshotAfter[i] {
					if !inAllowedRanges(i, allowedRanges) {
						t.Fatalf("unexpected byte change at 0x%X: before=0x%02X, after=0x%02X", i, snapshotBefore[i], snapshotAfter[i])
					}
				}
			}

			// Verify via GetCharacterAppearance
			readBack, err := engine.GetCharacterAppearance(loadedSession.SaveSessionID, setAppearanceTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterAppearance: %v", err)
			}
			wantAppearance := CharacterAppearance{
				SaveSessionID: loadedSession.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   setAppearanceTestSlot,
				Active:        true,
				Gender:        0, // inverted from preset bodyType 1
				VoiceType:     4, // preserved
				ModelIDs:      favModelIDs,
				FaceShape:     favFaceShape,
				Body:          favBody,
				Skin:          favSkin,
			}
			if !reflect.DeepEqual(readBack, wantAppearance) {
				t.Errorf("readBack = %+v, want %+v", readBack, wantAppearance)
			}

			// Persistence and reload verification
			savePath := filepath.Join(t.TempDir(), "applied_reload.sl2")
			if _, err := engine.WriteSave(loadedSession.SaveSessionID, "1", savePath); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloaded, err := engine.LoadSave(savePath, string(platform), "local")
			if err != nil {
				t.Fatalf("reload LoadSave: %v", err)
			}
			reloadedAppearance, err := engine.GetCharacterAppearance(reloaded.SaveSessionID, setAppearanceTestSlot)
			if err != nil {
				t.Fatalf("reloaded GetCharacterAppearance: %v", err)
			}
			wantReloaded := wantAppearance
			wantReloaded.SaveSessionID = reloaded.SaveSessionID
			wantReloaded.SaveRevision = "0"
			if !reflect.DeepEqual(reloadedAppearance, wantReloaded) {
				t.Errorf("reloadedAppearance = %+v, want %+v", reloadedAppearance, wantReloaded)
			}

			// Check direct binary layout from disk
			savedBytes, err := os.ReadFile(savePath)
			if err != nil {
				t.Fatalf("read savedBytes: %v", err)
			}
			if savedBytes[genderAt] != 0 {
				t.Errorf("saved gender = %d, want 0", savedBytes[genderAt])
			}
			if savedBytes[voiceAt] != 4 {
				t.Errorf("saved voice = %d, want 4 (preserved)", savedBytes[voiceAt])
			}
			// Opaque block in character FaceData must be preserved (0x70..0xB0)
			for i := applyFavFaceOpaqueStart; i < applyFavFaceOpaqueEnd; i++ {
				wantByte := byte(i ^ 0xA5)
				if gotByte := savedBytes[faceAt+int64(i)]; gotByte != wantByte {
					t.Fatalf("opaque block byte at 0x%X = 0x%02X, want preserved 0x%02X", i, gotByte, wantByte)
				}
			}
			// Trailing sex flags must be zeroed (two bytes: 0x125 and 0x126)
			if savedBytes[faceAt+applyFavSexFlagsOffset] != 0 || savedBytes[faceAt+applyFavSexFlagsOffset+1] != 0 {
				t.Errorf("trailing sex flags = [%02X, %02X], want [00, 00]",
					savedBytes[faceAt+applyFavSexFlagsOffset], savedBytes[faceAt+applyFavSexFlagsOffset+1])
			}

			// Favorite slot in UserData10 must remain identical to fileBefore
			favBefore := fileBefore[slotAt : slotAt+applyFavSlotSize]
			favAfter := savedBytes[slotAt : slotAt+applyFavSlotSize]
			if !bytes.Equal(favBefore, favAfter) {
				t.Errorf("Favorite slot in UserData10 was unexpectedly modified")
			}
		})
	}
}

func TestApplyFavoritePresetUndoAndIdempotence(t *testing.T) {
	initial := setAppearanceTestValues(0x22)
	initial.Gender = 0
	initial.VoiceType = 2

	var favModelIDs [8]uint32
	for i := 0; i < 8; i++ {
		favModelIDs[i] = uint32(0x2000 + i)
	}
	var favFaceShape [64]byte
	var favUnkBlock [64]byte
	var favBody [7]byte
	var favSkin [91]byte

	// Preset bodyType = 0 (male) -> character Gender = 1 (male)
	path, _, _, _, _ := writeApplyFavoriteFixture(
		t, PlatformPC, setAppearanceTestSlot, initial,
		0, 0, favModelIDs, favFaceShape, favUnkBlock, favBody, favSkin, true,
	)

	engine := New()
	loadedSession, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	initialAppearance, err := engine.GetCharacterAppearance(loadedSession.SaveSessionID, setAppearanceTestSlot)
	if err != nil {
		t.Fatalf("GetCharacterAppearance initial: %v", err)
	}

	// 1. First mutation
	res1, err := engine.ApplyFavoritePreset(loadedSession.SaveSessionID, setAppearanceTestSlot, 0, "0")
	if err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if res1.SaveRevision != "1" {
		t.Fatalf("res1 revision = %q, want \"1\"", res1.SaveRevision)
	}

	loaded := engine.sessions[loadedSession.SaveSessionID]
	snapAfter1 := bytes.Clone(loaded.snapshot.data)

	undoState1, err := engine.GetUndoState(loadedSession.SaveSessionID, setAppearanceTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if !undoState1.Available || undoState1.OperationKind != kindApplyFavoritePreset {
		t.Errorf("undoState = %+v, want Available=true, op=%s", undoState1, kindApplyFavoritePreset)
	}

	// 2. Idempotent second mutation with same values
	res2, err := engine.ApplyFavoritePreset(loadedSession.SaveSessionID, setAppearanceTestSlot, 0, "1")
	if err != nil {
		t.Fatalf("idempotent apply: %v", err)
	}
	if res2.SaveRevision != "2" {
		t.Fatalf("res2 revision = %q, want \"2\"", res2.SaveRevision)
	}

	snapAfter2 := bytes.Clone(loaded.snapshot.data)
	if !bytes.Equal(snapAfter1, snapAfter2) {
		t.Fatal("idempotent mutation changed snapshot bytes")
	}
	if !loaded.session.dirty {
		t.Fatal("idempotent mutation did not set dirty flag")
	}

	// Idempotent commit did not change any character bytes, so earlier undo point is dropped and no new point is recorded
	undoState2, err := engine.GetUndoState(loadedSession.SaveSessionID, setAppearanceTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState2: %v", err)
	}
	if undoState2.Available {
		t.Errorf("undoState2 after idempotent commit = %+v, want Available=false", undoState2)
	}

	// 3. Fresh session for testing complete undo restoration
	loadedSession2, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = engine.ApplyFavoritePreset(loadedSession2.SaveSessionID, setAppearanceTestSlot, 0, "0")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	undoToken, err := engine.GetUndoState(loadedSession2.SaveSessionID, setAppearanceTestSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undoToken.OperationKind != kindApplyFavoritePreset {
		t.Errorf("undoToken.OperationKind = %q, want %q", undoToken.OperationKind, kindApplyFavoritePreset)
	}

	undoRes, err := engine.UndoCharacterChanges(loadedSession2.SaveSessionID, setAppearanceTestSlot, undoToken.UndoToken, "1")
	if err != nil {
		t.Fatalf("UndoCharacterChanges: %v", err)
	}
	if undoRes.SaveRevision != "2" {
		t.Errorf("undo revision = %q, want \"2\"", undoRes.SaveRevision)
	}

	undoneAppearance, err := engine.GetCharacterAppearance(loadedSession2.SaveSessionID, setAppearanceTestSlot)
	if err != nil {
		t.Fatalf("undone GetCharacterAppearance: %v", err)
	}
	wantUndone := initialAppearance
	wantUndone.SaveSessionID = loadedSession2.SaveSessionID
	wantUndone.SaveRevision = undoRes.SaveRevision
	if !reflect.DeepEqual(undoneAppearance, wantUndone) {
		t.Errorf("undone appearance = %+v, want %+v", undoneAppearance, wantUndone)
	}
}

func TestApplyFavoritePresetRejectsInvalidRequests(t *testing.T) {
	initial := setAppearanceTestValues(0x33)
	var favModelIDs [8]uint32
	var favFaceShape [64]byte
	var favUnkBlock [64]byte
	var favBody [7]byte
	var favSkin [91]byte

	path, _, _, _, _ := writeApplyFavoriteFixture(
		t, PlatformPC, setAppearanceTestSlot, initial,
		1, 0, favModelIDs, favFaceShape, favUnkBlock, favBody, favSkin, true,
	)

	// Make slot 2 empty
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	slot2At := applyFavSlotOffset(PlatformPC, 2)
	clear(data[slot2At : slot2At+applyFavSlotSize])

	// Make slot 4 malformed (invalid bodyType = 2)
	slot4At := applyFavSlotOffset(PlatformPC, 4)
	copy(data[slot4At:], data[applyFavSlotOffset(PlatformPC, 1):applyFavSlotOffset(PlatformPC, 1)+applyFavSlotSize])
	data[slot4At+0x09] = 2

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	engine := New()
	loadedSession, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	cases := []struct {
		name             string
		charID           int
		favSlotID        int
		expectedRevision string
	}{
		{"empty preset slot", setAppearanceTestSlot, 2, "0"},
		{"malformed preset body type", setAppearanceTestSlot, 4, "0"},
		{"negative favoriteSlotID", setAppearanceTestSlot, -1, "0"},
		{"out of bounds favoriteSlotID", setAppearanceTestSlot, 15, "0"},
		{"negative characterID", -1, 1, "0"},
		{"out of bounds characterID", 10, 1, "0"},
		{"inactive character", 0, 1, "0"},
		{"stale expectedRevision", setAppearanceTestSlot, 1, "99"},
		{"non-canonical expectedRevision", setAppearanceTestSlot, 1, "01"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loaded := engine.sessions[loadedSession.SaveSessionID]
			snapshotBefore := bytes.Clone(loaded.snapshot.data)
			revBefore := loaded.session.revision
			dirtyBefore := loaded.session.dirty
			undoBefore := loaded.session.undo
			ownedSeqBefore := loaded.session.ownedSeq
			ownedByLocatorBefore := maps.Clone(loaded.session.ownedByLocator)
			ownedByIDBefore := maps.Clone(loaded.session.ownedByID)

			res, err := engine.ApplyFavoritePreset(
				loadedSession.SaveSessionID, tc.charID, tc.favSlotID, tc.expectedRevision)
			if err == nil {
				t.Fatalf("ApplyFavoritePreset(%s) accepted invalid request", tc.name)
			}
			if !reflect.DeepEqual(res, ApplyFavoritePresetResult{}) {
				t.Errorf("ApplyFavoritePreset(%s) returned non-zero result on error: %+v", tc.name, res)
			}
			if !bytes.Equal(loaded.snapshot.data, snapshotBefore) {
				t.Errorf("ApplyFavoritePreset(%s) mutated snapshot bytes on error", tc.name)
			}
			if loaded.session.revision != revBefore {
				t.Errorf("ApplyFavoritePreset(%s) advanced revision on error: got %d, want %d", tc.name, loaded.session.revision, revBefore)
			}
			if loaded.session.dirty != dirtyBefore {
				t.Errorf("ApplyFavoritePreset(%s) changed dirty flag on error: got %v, want %v", tc.name, loaded.session.dirty, dirtyBefore)
			}
			if loaded.session.undo != undoBefore {
				t.Errorf("ApplyFavoritePreset(%s) mutated undo state on error", tc.name)
			}
			if loaded.session.ownedSeq != ownedSeqBefore {
				t.Errorf("ApplyFavoritePreset(%s) changed ownedSeq on error: got %d, want %d", tc.name, loaded.session.ownedSeq, ownedSeqBefore)
			}
			if !reflect.DeepEqual(loaded.session.ownedByLocator, ownedByLocatorBefore) {
				t.Errorf("ApplyFavoritePreset(%s) changed ownedByLocator on error", tc.name)
			}
			if !reflect.DeepEqual(loaded.session.ownedByID, ownedByIDBefore) {
				t.Errorf("ApplyFavoritePreset(%s) changed ownedByID on error", tc.name)
			}
		})
	}
}
