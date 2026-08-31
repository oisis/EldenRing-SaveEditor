package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// Independent test constants for Mirror Favorites and FaceData slot layout.
// None of the production layout constants or offset helpers are used to construct
// or assert the binary contract.
const (
	testFavSlotCount      = 15
	testFavSlotSize       = 0x130 // 304 bytes
	testFavBaseOffset     = 0x154
	testActiveFlagsOffset = 0x1954
	testActiveFlagValue   = 1

	testPCSlotDataBase   = 0x300 + 0x10
	testPCSlotStride     = 0x280010
	testPCUserData10Base = int64(0x300) + 10*0x280010 + 0x10

	testPS4SlotDataBase   = 0x70
	testPS4SlotStride     = 0x280000
	testPS4UserData10Base = int64(0x70) + 10*0x280000

	testFaceDataAlignment = 4
	testFaceDataInnerSize = 0x120
)

var (
	testFaceDataHeader = []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'}
	testPlayerAnchor   = []byte{
		0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
)

type testPlatformLayout struct {
	userData10Base int64
	slotDataBase   int64
	slotStride     int64
}

func testLayoutForPlatform(platform Platform) testPlatformLayout {
	if platform == PlatformPS4 {
		return testPlatformLayout{
			userData10Base: testPS4UserData10Base,
			slotDataBase:   testPS4SlotDataBase,
			slotStride:     testPS4SlotStride,
		}
	}
	return testPlatformLayout{
		userData10Base: testPCUserData10Base,
		slotDataBase:   testPCSlotDataBase,
		slotStride:     testPCSlotStride,
	}
}

func testFavTargetOffset(userData10Base int64, slot int) int64 {
	return userData10Base + testFavBaseOffset + int64(slot)*testFavSlotSize
}

// buildExpectedFavoriteSlotBuffer independently constructs the 0x130-byte expected
// preset buffer for a test character appearance.
func buildExpectedFavoriteSlotBuffer(favBodyType byte, modelIDs []uint32, faceShape []byte, unkBlock []byte, body []byte, skin []byte) []byte {
	buf := make([]byte, testFavSlotSize)
	binary.LittleEndian.PutUint16(buf[0x00:], 0xFACE)
	binary.LittleEndian.PutUint32(buf[0x04:], 0x11D0)
	buf[0x08] = 1
	buf[0x09] = favBodyType
	// 0x0A..0x14 zero pad
	// 0x14..0x18 active marker = 0
	copy(buf[0x18:], "FACE")
	binary.LittleEndian.PutUint32(buf[0x1C:], 4)
	binary.LittleEndian.PutUint32(buf[0x20:], 0x120)
	for m := 0; m < 8 && m < len(modelIDs); m++ {
		binary.LittleEndian.PutUint32(buf[0x24+m*4:], modelIDs[m])
	}
	copy(buf[0x44:], faceShape)
	copy(buf[0x84:], unkBlock)
	copy(buf[0xC4:], body)
	copy(buf[0xCB:], skin)
	// 0x126..0x130 zero pad
	return buf
}

func writeSetFavoritesFixture(t *testing.T, platform Platform, charSlot int, gender byte, activePresets map[int]bool) string {
	t.Helper()

	var data []byte
	layout := testLayoutForPlatform(platform)
	switch platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
	default:
		t.Fatalf("unknown platform %q", platform)
	}

	slotBase := layout.slotDataBase + int64(charSlot)*layout.slotStride

	// Activate character slot
	data[layout.userData10Base+testActiveFlagsOffset+int64(charSlot)] = testActiveFlagValue

	// Write player anchor, gender and voice type
	anchor := slotBase + 0x1000
	copy(data[anchor:], testPlayerAnchor)
	data[anchor-249] = gender // gender offset relative to anchor
	data[anchor-245] = 2      // voice type

	// Write face data block
	faceAt := slotBase + 0x2000
	copy(data[faceAt:], testFaceDataHeader)
	binary.LittleEndian.PutUint32(data[faceAt+0x08:], testFaceDataAlignment)
	binary.LittleEndian.PutUint32(data[faceAt+0x0C:], testFaceDataInnerSize)

	// Distinctive models (8 x uint32 @ 0x10)
	for m := 0; m < 8; m++ {
		binary.LittleEndian.PutUint32(data[faceAt+0x10+int64(m*4):], uint32(100+m))
	}
	// Face shape (64 bytes @ 0x30)
	for b := 0; b < 64; b++ {
		data[faceAt+0x30+int64(b)] = byte(0x10 + b)
	}
	// Unknown block (64 bytes @ 0x70)
	for b := 0; b < 64; b++ {
		data[faceAt+0x70+int64(b)] = byte(0x50 + b)
	}
	// Body (7 bytes @ 0xB0)
	for b := 0; b < 7; b++ {
		data[faceAt+0xB0+int64(b)] = byte(0x90 + b)
	}
	// Skin (91 bytes @ 0xB7)
	for b := 0; b < 91; b++ {
		data[faceAt+0xB7+int64(b)] = byte(0xA0 + (b % 30))
	}

	// Populate preset slots in UserData10 if specified
	for slot := 0; slot < testFavSlotCount; slot++ {
		slotAt := testFavTargetOffset(layout.userData10Base, slot)
		if activePresets != nil && activePresets[slot] {
			for b := 0; b < testFavSlotSize; b++ {
				data[slotAt+int64(b)] = byte(0xEE)
			}
			binary.LittleEndian.PutUint16(data[slotAt:], 0xFACE)
			binary.LittleEndian.PutUint32(data[slotAt+4:], 0x11D0)
			copy(data[slotAt+0x18:], "FACE")
		}
	}

	path := filepath.Join(t.TempDir(), "set_favorites.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestSetFavoritePresetTablePCAndPS4(t *testing.T) {
	tests := []struct {
		name        string
		platform    Platform
		charSlot    int
		charGender  byte
		favBodyType byte
		targetSlot  int
	}{
		{
			name:        "PC Male Character to Favorites Slot 0",
			platform:    PlatformPC,
			charSlot:    2,
			charGender:  1, // Male -> fav body type 0
			favBodyType: 0,
			targetSlot:  0,
		},
		{
			name:        "PS4 Female Character to Favorites Slot 14",
			platform:    PlatformPS4,
			charSlot:    5,
			charGender:  0, // Female -> fav body type 1
			favBodyType: 1,
			targetSlot:  14,
		},
	}

	// Shared source appearance payload matching writeSetFavoritesFixture
	modelIDs := make([]uint32, 8)
	for m := range modelIDs {
		modelIDs[m] = uint32(100 + m)
	}
	faceShape := make([]byte, 64)
	for b := range faceShape {
		faceShape[b] = byte(0x10 + b)
	}
	unkBlock := make([]byte, 64)
	for b := range unkBlock {
		unkBlock[b] = byte(0x50 + b)
	}
	body := make([]byte, 7)
	for b := range body {
		body[b] = byte(0x90 + b)
	}
	skin := make([]byte, 91)
	for b := range skin {
		skin[b] = byte(0xA0 + (b % 30))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activePresets := map[int]bool{1: true, 13: true}
			path := writeSetFavoritesFixture(t, tt.platform, tt.charSlot, tt.charGender, activePresets)
			layout := testLayoutForPlatform(tt.platform)

			engine := New()
			session, err := engine.LoadSave(path, string(tt.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			engine.mutex.Lock()
			loaded := engine.sessions[session.SaveSessionID]
			snapshotBefore := append([]byte(nil), loaded.snapshot.data...)
			engine.mutex.Unlock()

			res, err := engine.SetFavoritePreset(
				session.SaveSessionID, tt.targetSlot, tt.charSlot, "0")
			if err != nil {
				t.Fatalf("SetFavoritePreset: %v", err)
			}

			if res.SaveSessionID != session.SaveSessionID {
				t.Errorf("SaveSessionID = %q, want %q", res.SaveSessionID, session.SaveSessionID)
			}
			if res.SaveRevision != "1" {
				t.Errorf("SaveRevision = %q, want \"1\"", res.SaveRevision)
			}
			if res.FavoriteSlotID != tt.targetSlot {
				t.Errorf("FavoriteSlotID = %d, want %d", res.FavoriteSlotID, tt.targetSlot)
			}
			if res.SourceCharacterID != tt.charSlot {
				t.Errorf("SourceCharacterID = %d, want %d", res.SourceCharacterID, tt.charSlot)
			}

			engine.mutex.Lock()
			loaded = engine.sessions[session.SaveSessionID]
			snapshotAfter := append([]byte(nil), loaded.snapshot.data...)
			dirtyAfter := loaded.session.dirty
			engine.mutex.Unlock()

			if !dirtyAfter {
				t.Error("expected dirty = true after SetFavoritePreset")
			}

			// Validate entire 0x130-byte target slot buffer against independently built expected state
			targetAt := testFavTargetOffset(layout.userData10Base, tt.targetSlot)
			slotBytes := snapshotAfter[targetAt : targetAt+testFavSlotSize]
			expectedBuf := buildExpectedFavoriteSlotBuffer(tt.favBodyType, modelIDs, faceShape, unkBlock, body, skin)

			if !bytes.Equal(slotBytes, expectedBuf) {
				t.Errorf("slotBytes mismatch (-got +want):\n%#v\nvs\n%#v", slotBytes, expectedBuf)
			}

			// Verify all bytes outside target slot are completely untouched
			if !bytes.Equal(snapshotBefore[:targetAt], snapshotAfter[:targetAt]) {
				t.Error("bytes before target slot were modified")
			}
			if !bytes.Equal(snapshotBefore[targetAt+testFavSlotSize:], snapshotAfter[targetAt+testFavSlotSize:]) {
				t.Error("bytes after target slot were modified")
			}
		})
	}
}

func TestSetFavoritePresetOverwriteEmptyAndNoOp(t *testing.T) {
	// Slot 3 pre-populated with distinctive 0xEE bytes; slot 5 empty
	activePresets := map[int]bool{3: true}
	path := writeSetFavoritesFixture(t, PlatformPC, 0, 1, activePresets)

	engine := New()
	session, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	modelIDs := make([]uint32, 8)
	for m := range modelIDs {
		modelIDs[m] = uint32(100 + m)
	}
	faceShape := make([]byte, 64)
	for b := range faceShape {
		faceShape[b] = byte(0x10 + b)
	}
	unkBlock := make([]byte, 64)
	for b := range unkBlock {
		unkBlock[b] = byte(0x50 + b)
	}
	body := make([]byte, 7)
	for b := range body {
		body[b] = byte(0x90 + b)
	}
	skin := make([]byte, 91)
	for b := range skin {
		skin[b] = byte(0xA0 + (b % 30))
	}
	expectedBuf := buildExpectedFavoriteSlotBuffer(0, modelIDs, faceShape, unkBlock, body, skin)

	// 1. Overwrite active slot 3 (confirm 0xEE bytes replaced by exact expected preset)
	res1, err := engine.SetFavoritePreset(session.SaveSessionID, 3, 0, "0")
	if err != nil {
		t.Fatalf("overwrite slot 3: %v", err)
	}
	if res1.SaveRevision != "1" {
		t.Errorf("revision = %q, want \"1\"", res1.SaveRevision)
	}
	slot3At := testFavTargetOffset(testPCUserData10Base, 3)
	engine.mutex.Lock()
	slot3Bytes := append([]byte(nil), engine.sessions[session.SaveSessionID].snapshot.data[slot3At:slot3At+testFavSlotSize]...)
	engine.mutex.Unlock()
	if !bytes.Equal(slot3Bytes, expectedBuf) {
		t.Errorf("overwritten slot 3 does not match expected preset buffer")
	}

	// 2. Write empty slot 5 (confirm exact preset written)
	res2, err := engine.SetFavoritePreset(session.SaveSessionID, 5, 0, "1")
	if err != nil {
		t.Fatalf("write empty slot 5: %v", err)
	}
	if res2.SaveRevision != "2" {
		t.Errorf("revision = %q, want \"2\"", res2.SaveRevision)
	}
	slot5At := testFavTargetOffset(testPCUserData10Base, 5)
	engine.mutex.Lock()
	slot5Bytes := append([]byte(nil), engine.sessions[session.SaveSessionID].snapshot.data[slot5At:slot5At+testFavSlotSize]...)
	engine.mutex.Unlock()
	if !bytes.Equal(slot5Bytes, expectedBuf) {
		t.Errorf("written slot 5 does not match expected preset buffer")
	}

	// 3. No-op write (identical bytes into slot 5 again)
	engine.mutex.Lock()
	snapshotBeforeNoOp := append([]byte(nil), engine.sessions[session.SaveSessionID].snapshot.data...)
	engine.mutex.Unlock()

	res3, err := engine.SetFavoritePreset(session.SaveSessionID, 5, 0, "2")
	if err != nil {
		t.Fatalf("noop write slot 5: %v", err)
	}
	if res3.SaveRevision != "3" {
		t.Errorf("revision = %q, want \"3\"", res3.SaveRevision)
	}

	engine.mutex.Lock()
	loaded := engine.sessions[session.SaveSessionID]
	snapshotAfterNoOp := append([]byte(nil), loaded.snapshot.data...)
	dirtyAfterNoOp := loaded.session.dirty
	engine.mutex.Unlock()

	if !dirtyAfterNoOp {
		t.Error("expected dirty = true after no-op mutation")
	}
	if !bytes.Equal(snapshotBeforeNoOp, snapshotAfterNoOp) {
		t.Error("snapshot bytes were modified during byte-identical no-op write")
	}
}

func TestSetFavoritePresetMalformedSourceLayout(t *testing.T) {
	path := writeSetFavoritesFixture(t, PlatformPC, 0, 1, nil)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	// Corrupt FaceData alignment from 4 to 8
	faceAt := testPCSlotDataBase + 0x2000
	binary.LittleEndian.PutUint32(data[faceAt+0x08:], 8)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	engine := New()
	session, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	snapshotBefore := append([]byte(nil), engine.sessions[session.SaveSessionID].snapshot.data...)
	engine.mutex.Unlock()

	_, err = engine.SetFavoritePreset(session.SaveSessionID, 0, 0, "0")
	if err == nil {
		t.Fatal("expected error for malformed FaceData alignment, got nil")
	}

	engine.mutex.Lock()
	loaded := engine.sessions[session.SaveSessionID]
	snapshotAfter := append([]byte(nil), loaded.snapshot.data...)
	revisionAfter := loaded.session.revisionString()
	dirtyAfter := loaded.session.dirty
	engine.mutex.Unlock()

	if revisionAfter != "0" {
		t.Errorf("revision changed to %q, want \"0\"", revisionAfter)
	}
	if dirtyAfter {
		t.Error("dirty changed to true after rejected call")
	}
	if !bytes.Equal(snapshotBefore, snapshotAfter) {
		t.Error("snapshot bytes were modified after malformed layout error")
	}
}

func TestSetFavoritePresetRejections(t *testing.T) {
	path := writeSetFavoritesFixture(t, PlatformPC, 1, 1, nil)
	engine := New()
	session, err := engine.LoadSave(path, string(PlatformPC), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	snapshotBefore := append([]byte(nil), engine.sessions[session.SaveSessionID].snapshot.data...)
	engine.mutex.Unlock()

	tests := []struct {
		name              string
		slotID            int
		sourceCharacterID int
		revision          string
	}{
		{"negative slotID", -1, 1, "0"},
		{"slotID out of bounds", 15, 1, "0"},
		{"negative characterID", 0, -1, "0"},
		{"characterID out of bounds", 0, 10, "0"},
		{"inactive character slot", 0, 2, "0"}, // slot 2 is inactive in fixture
		{"stale revision", 0, 1, "1"},
		{"invalid decimal revision", 0, 1, "invalid"},
		{"leading zero revision", 0, 1, "01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.SetFavoritePreset(session.SaveSessionID, tt.slotID, tt.sourceCharacterID, tt.revision)
			if err == nil {
				t.Error("expected error, got nil")
			}

			engine.mutex.Lock()
			snapshotAfter := append([]byte(nil), engine.sessions[session.SaveSessionID].snapshot.data...)
			revisionAfter := engine.sessions[session.SaveSessionID].session.revisionString()
			dirtyAfter := engine.sessions[session.SaveSessionID].session.dirty
			engine.mutex.Unlock()

			if revisionAfter != "0" {
				t.Errorf("revision changed to %q after failed call", revisionAfter)
			}
			if dirtyAfter {
				t.Error("dirty changed to true after rejected call")
			}
			if !bytes.Equal(snapshotBefore, snapshotAfter) {
				t.Error("snapshot bytes were modified after failed call")
			}
		})
	}
}
