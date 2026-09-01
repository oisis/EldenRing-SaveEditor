package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

const (
	setNameTestSlot             = 3
	setNameTestPlayerOffset     = -0x11B
	setNameTestSummaryOffset    = 0x195E
	setNameTestSummaryStride    = 0x24C
	setNameTestFieldSize        = 32
	setNameTestPCSlotBase       = 0x310
	setNameTestPCSlotStride     = 0x280010
	setNameTestPS4SlotBase      = 0x70
	setNameTestPS4SlotStride    = 0x280000
	setNameTestPCUserDataBase   = 0x19003B0
	setNameTestPS4UserDataBase  = 0x1900070
	setNameTestOrdinaryAnchorAt = 0x0640
	setNameTestFullAnchorAt     = 0x20 + 5120*8 + 0x1B0
)

func setNameTestOffsets(platform Platform, anchorAt int64) (int64, int64) {
	var slotBase, userDataBase int64
	if platform == PlatformPS4 {
		slotBase = setNameTestPS4SlotBase + setNameTestSlot*setNameTestPS4SlotStride
		userDataBase = setNameTestPS4UserDataBase
	} else {
		slotBase = setNameTestPCSlotBase + setNameTestSlot*setNameTestPCSlotStride
		userDataBase = setNameTestPCUserDataBase
	}
	return slotBase + anchorAt + setNameTestPlayerOffset,
		userDataBase + setNameTestSummaryOffset + setNameTestSlot*setNameTestSummaryStride
}

func setNameTestEncode(value string) []byte {
	encoded := make([]byte, setNameTestFieldSize)
	for index, unit := range utf16.Encode([]rune(value)) {
		binary.LittleEndian.PutUint16(encoded[index*2:], unit)
	}
	return encoded
}

func writeCharacterNameFixture(
	t *testing.T,
	platform Platform,
	active bool,
	withAnchor bool,
	playerName string,
	summaryName string,
) string {
	t.Helper()

	content := statsFixture{
		platform: platform,
		slot:     setNameTestSlot,
		anchorAt: setNameTestOrdinaryAnchorAt,
		noAnchor: !withAnchor,
	}
	if active {
		content.flag = userData10ActiveFlagValue
	}
	path := writeStatsFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	playerAt, summaryAt := setNameTestOffsets(platform, setNameTestOrdinaryAnchorAt)
	copy(data[playerAt:], setNameTestEncode(playerName))
	copy(data[summaryAt:], setNameTestEncode(summaryName))
	// These two units sit immediately behind the name fields. They prove the
	// mutation does not widen its writes into adjacent bytes.
	binary.LittleEndian.PutUint16(data[playerAt+setNameTestFieldSize:], 0xA1B2)
	binary.LittleEndian.PutUint16(data[summaryAt+setNameTestFieldSize:], 0xC3D4)

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	return path
}

func assertSetNameRejectedUnchanged(
	t *testing.T,
	engine *Engine,
	sessionID string,
	before []byte,
) {
	t.Helper()
	if !bytes.Equal(before, engine.sessions[sessionID].snapshot.data) {
		t.Error("rejected name mutation changed the private snapshot")
	}
	if revision := engine.sessions[sessionID].session.revisionString(); revision != "0" {
		t.Errorf("revision after rejection = %q, want 0", revision)
	}
	if engine.sessions[sessionID].session.dirty {
		t.Error("rejected name mutation marked the session dirty")
	}
}

func TestSetCharacterNameSynchronizesBothCopiesOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			path := writeCharacterNameFixture(
				t, platform, true, true, "Old slot", "Old menu")
			engine := New()
			loaded, err := engine.LoadSave(path, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)
			name := "Ranni 🌙"
			result, err := engine.SetCharacterName(
				loaded.SaveSessionID, setNameTestSlot, name, "0")
			if err != nil {
				t.Fatalf("SetCharacterName: %v", err)
			}
			assertCommittedReceipt(t, result.MutationReceipt, loaded.SaveSessionID,
				kindSetCharacterName, "1")
			// The receipt is pinned from the result because operationID names one
			// execution and cannot be predicted; every other member is asserted above.
			want := SetCharacterNameResult{
				MutationReceipt: result.MutationReceipt,
				CharacterID:     setNameTestSlot,
				Name:            name,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			playerAt, summaryAt := setNameTestOffsets(platform, setNameTestOrdinaryAnchorAt)
			expected := bytes.Clone(before)
			encoded := setNameTestEncode(name)
			copy(expected[playerAt:], encoded)
			copy(expected[summaryAt:], encoded)
			after := engine.sessions[loaded.SaveSessionID].snapshot.data
			if !bytes.Equal(after, expected) {
				t.Error("mutation changed bytes outside the two 32-byte name fields")
			}
			if got := binary.LittleEndian.Uint16(after[playerAt+setNameTestFieldSize:]); got != 0xA1B2 {
				t.Errorf("PlayerGameData trailing unit = 0x%04X, want 0xA1B2", got)
			}
			if got := binary.LittleEndian.Uint16(after[summaryAt+setNameTestFieldSize:]); got != 0xC3D4 {
				t.Errorf("ProfileSummary trailing unit = 0x%04X, want 0xC3D4", got)
			}

			profile, err := engine.GetCharacterProfile(loaded.SaveSessionID, setNameTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile: %v", err)
			}
			if profile.Name != name {
				t.Errorf("profile name = %q, want %q", profile.Name, name)
			}
			if !engine.sessions[loaded.SaveSessionID].session.dirty {
				t.Error("successful name mutation did not mark the session dirty")
			}
		})
	}
}

func TestSetCharacterNameAcceptsUTF16BoundaryAndPreservesInput(t *testing.T) {
	// Whitespace, a decomposed accent, ten BMP units and one surrogate pair use
	// all sixteen save units. Exact equality proves there is no trimming or
	// Unicode normalisation on the accepted boundary.
	name := " e\u0301" + strings.Repeat("A", 10) + "😀 "
	engine := New()
	loaded, err := engine.LoadSave(
		writeCharacterNameFixture(t, PlatformPC, true, true, "Old", "Old"), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.SetCharacterName(
		loaded.SaveSessionID, setNameTestSlot, name, "0")
	if err != nil {
		t.Fatalf("SetCharacterName: %v", err)
	}
	if result.Name != name {
		t.Errorf("result name = %q, want %q", result.Name, name)
	}
	profile, err := engine.GetCharacterProfile(loaded.SaveSessionID, setNameTestSlot)
	if err != nil {
		t.Fatalf("GetCharacterProfile: %v", err)
	}
	if profile.Name != name {
		t.Errorf("profile name = %q, want %q", profile.Name, name)
	}
}

func TestSetCharacterNameRejectsInvalidNamesWithoutMutation(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeCharacterNameFixture(t, PlatformPC, true, true, "Original", "Original"), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	cases := map[string]struct {
		name string
		want string
	}{
		"empty":         {"", "name is required"},
		"invalid UTF-8": {string([]byte{0xFF}), "name must be valid UTF-8"},
		"NUL":           {"Bad\x00Name", "name must not contain NUL"},
		"seventeen units": {strings.Repeat("A", 17),
			"name uses 17 UTF-16 code units, maximum is 16"},
		"surrogate overflow": {strings.Repeat("A", 15) + "😀",
			"name uses 17 UTF-16 code units, maximum is 16"},
	}
	for testName, testCase := range cases {
		t.Run(testName, func(t *testing.T) {
			_, err := engine.SetCharacterName(
				loaded.SaveSessionID, setNameTestSlot, testCase.name, "0")
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
	assertSetNameRejectedUnchanged(t, engine, loaded.SaveSessionID, before)
}

func TestSetCharacterNameRejectsInvalidSessionSlotAndRevision(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeCharacterNameFixture(t, PlatformPC, true, true, "Original", "Original"), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	cases := map[string]struct {
		sessionID        string
		characterID      int
		expectedRevision string
		want             string
	}{
		"empty session": {"", setNameTestSlot, "0", "saveSessionID is required"},
		"unknown session": {"missing", setNameTestSlot, "0",
			`unknown save session "missing"`},
		"character below": {loaded.SaveSessionID, -1, "0",
			"characterID -1 is outside the range 0..9"},
		"character above": {loaded.SaveSessionID, 10, "0",
			"characterID 10 is outside the range 0..9"},
		"noncanonical revision": {loaded.SaveSessionID, setNameTestSlot, "00",
			`expectedRevision must be a canonical decimal saveRevision; got "00"`},
		"stale revision": {loaded.SaveSessionID, setNameTestSlot, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`},
	}
	for testName, testCase := range cases {
		t.Run(testName, func(t *testing.T) {
			_, err := engine.SetCharacterName(
				testCase.sessionID, testCase.characterID, "New", testCase.expectedRevision)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
	assertSetNameRejectedUnchanged(t, engine, loaded.SaveSessionID, before)
}

func TestSetCharacterNameRejectsInactiveAndMissingAnchor(t *testing.T) {
	cases := []struct {
		name       string
		platform   Platform
		active     bool
		withAnchor bool
		want       string
	}{
		{"inactive", PlatformPC, false, false, "character 3 is not active"},
		{"missing anchor", PlatformPS4, true, false, "character 3 carries no statistics anchor"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeCharacterNameFixture(
				t, testCase.platform, testCase.active, testCase.withAnchor, "Old", "Old"), string(testCase.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)
			_, err = engine.SetCharacterName(
				loaded.SaveSessionID, setNameTestSlot, "New", "0")
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			assertSetNameRejectedUnchanged(t, engine, loaded.SaveSessionID, before)
		})
	}
}

func TestSetCharacterNameIdempotentAssignmentAdvancesRevision(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeCharacterNameFixture(t, PlatformPC, true, true, "Same", "Same"), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	result, err := engine.SetCharacterName(
		loaded.SaveSessionID, setNameTestSlot, "Same", "0")
	if err != nil {
		t.Fatalf("SetCharacterName: %v", err)
	}
	if result.SaveRevision != "1" {
		t.Errorf("saveRevision = %q, want 1", result.SaveRevision)
	}
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) {
		t.Error("idempotent assignment changed the snapshot")
	}
}

func TestSetCharacterNamePersistsAndReloadsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := gestureTestActiveFixture(
				platform, setNameTestSlot, setNameTestFullAnchorAt, 0)
			content.records = setGestureTestRecords()
			path := writeGestureFixture(t, content)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read full fixture: %v", err)
			}
			slotBase := int64(setNameTestPCSlotBase) +
				setNameTestSlot*setNameTestPCSlotStride
			if platform == PlatformPS4 {
				slotBase = setNameTestPS4SlotBase +
					setNameTestSlot*setNameTestPS4SlotStride
			}
			binary.LittleEndian.PutUint32(data[slotBase:], 0x6E)
			playerAt, summaryAt := setNameTestOffsets(platform, setNameTestFullAnchorAt)
			copy(data[playerAt:], setNameTestEncode("Before"))
			copy(data[summaryAt:], setNameTestEncode("Before"))
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatalf("rewrite full fixture: %v", err)
			}
			sourceBefore := bytes.Clone(data)

			engine := New()
			loaded, err := engine.LoadSave(path, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			if _, err := engine.SetCharacterName(
				loaded.SaveSessionID, setNameTestSlot, "After 🌙", "0"); err != nil {
				t.Fatalf("SetCharacterName: %v", err)
			}
			sourceAfter, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read source after mutation: %v", err)
			}
			if !bytes.Equal(sourceBefore, sourceAfter) {
				t.Error("in-memory mutation changed the source file")
			}

			target := filepath.Join(t.TempDir(), "renamed.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload target: %v", err)
			}
			profile, err := reloadedEngine.GetCharacterProfile(
				reloaded.SaveSessionID, setNameTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile: %v", err)
			}
			if profile.Name != "After 🌙" {
				t.Errorf("reloaded profile name = %q, want %q", profile.Name, "After 🌙")
			}
			playerAt, _ = setNameTestOffsets(platform, setNameTestFullAnchorAt)
			playerRaw, err := reloadedEngine.sessions[reloaded.SaveSessionID].snapshot.readAt(
				playerAt, setNameTestFieldSize)
			if err != nil {
				t.Fatalf("read reloaded PlayerGameData name: %v", err)
			}
			if got := decodeCharacterName(playerRaw); got != "After 🌙" {
				t.Errorf("reloaded PlayerGameData name = %q, want %q", got, "After 🌙")
			}
		})
	}
}
