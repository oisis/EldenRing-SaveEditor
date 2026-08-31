package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

const (
	setRunesTestSlot             = 3
	setRunesTestFieldOffset      = int64(-331)
	setRunesTestMaximum          = uint32(999_999_999)
	setRunesTestPCSlotBase       = 0x310
	setRunesTestPCSlotStride     = 0x280010
	setRunesTestPS4SlotBase      = 0x70
	setRunesTestPS4SlotStride    = 0x280000
	setRunesTestOrdinaryAnchorAt = 0x0640
	setRunesTestFullAnchorAt     = 0x20 + 5120*8 + 0x1B0
)

func setRunesTestFieldAt(platform Platform, anchorAt int64) int64 {
	base := int64(setRunesTestPCSlotBase) + setRunesTestSlot*setRunesTestPCSlotStride
	if platform == PlatformPS4 {
		base = setRunesTestPS4SlotBase + setRunesTestSlot*setRunesTestPS4SlotStride
	}
	return base + anchorAt + setRunesTestFieldOffset
}

func writeCharacterRunesFixture(
	t *testing.T,
	platform Platform,
	active bool,
	withAnchor bool,
	runes uint32,
) string {
	t.Helper()
	content := statsFixture{
		platform: platform,
		slot:     setRunesTestSlot,
		anchorAt: setRunesTestOrdinaryAnchorAt,
		noAnchor: !withAnchor,
	}
	if active {
		content.flag = userData10ActiveFlagValue
	}
	path := writeStatsFixture(t, content)
	if !withAnchor {
		return path
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	binary.LittleEndian.PutUint32(
		data[setRunesTestFieldAt(platform, setRunesTestOrdinaryAnchorAt):], runes)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	return path
}

func assertSetRunesRejectedUnchanged(
	t *testing.T,
	engine *Engine,
	sessionID string,
	before []byte,
) {
	t.Helper()
	if !bytes.Equal(before, engine.sessions[sessionID].snapshot.data) {
		t.Error("rejected runes mutation changed the private snapshot")
	}
	if revision := engine.sessions[sessionID].session.revisionString(); revision != "0" {
		t.Errorf("revision after rejection = %q, want 0", revision)
	}
	if engine.sessions[sessionID].session.dirty {
		t.Error("rejected runes mutation marked the session dirty")
	}
}

func TestSetCharacterRunesChangesOnlyTheConfirmedFieldOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeCharacterRunesFixture(
				t, platform, true, true, 123), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

			result, err := engine.SetCharacterRunes(
				loaded.SaveSessionID, setRunesTestSlot, 456_789, "0")
			if err != nil {
				t.Fatalf("SetCharacterRunes: %v", err)
			}
			want := SetCharacterRunesResult{
				SaveSessionID: loaded.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   setRunesTestSlot,
				Runes:         456_789,
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			expected := bytes.Clone(before)
			fieldAt := setRunesTestFieldAt(platform, setRunesTestOrdinaryAnchorAt)
			binary.LittleEndian.PutUint32(expected[fieldAt:], 456_789)
			if !bytes.Equal(engine.sessions[loaded.SaveSessionID].snapshot.data, expected) {
				t.Error("mutation changed bytes outside the four-byte held-runes field")
			}
			if !engine.sessions[loaded.SaveSessionID].session.dirty {
				t.Error("successful runes mutation did not mark the session dirty")
			}
		})
	}
}

func TestSetCharacterRunesAcceptsZeroAndTheLegalMaximum(t *testing.T) {
	for name, runes := range map[string]uint32{
		"zero":    0,
		"maximum": setRunesTestMaximum,
	} {
		t.Run(name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeCharacterRunesFixture(
				t, PlatformPC, true, true, 1), "", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			result, err := engine.SetCharacterRunes(
				loaded.SaveSessionID, setRunesTestSlot, runes, "0")
			if err != nil {
				t.Fatalf("SetCharacterRunes(%d): %v", runes, err)
			}
			if result.Runes != runes || result.SaveRevision != "1" {
				t.Errorf("result = %+v, want runes %d at revision 1", result, runes)
			}
		})
	}
}

func TestSetCharacterRunesRejectsAboveTheLegalMaximum(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeCharacterRunesFixture(t, PlatformPC, true, true, 123), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	_, err = engine.SetCharacterRunes(
		loaded.SaveSessionID, setRunesTestSlot, 1_000_000_000, "0")
	if err == nil || err.Error() != "runes 1000000000 exceeds the maximum 999999999" {
		t.Fatalf("error = %v, want legal-maximum error", err)
	}
	assertSetRunesRejectedUnchanged(t, engine, loaded.SaveSessionID, before)
}

func TestSetCharacterRunesRejectsInvalidSessionSlotAndRevision(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeCharacterRunesFixture(t, PlatformPC, true, true, 123), "", "local")
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
		"empty session": {"", setRunesTestSlot, "0", "saveSessionID is required"},
		"unknown session": {"missing", setRunesTestSlot, "0",
			`unknown save session "missing"`},
		"character below": {loaded.SaveSessionID, -1, "0",
			"characterID -1 is outside the range 0..9"},
		"character above": {loaded.SaveSessionID, 10, "0",
			"characterID 10 is outside the range 0..9"},
		"noncanonical revision": {loaded.SaveSessionID, setRunesTestSlot, "00",
			`expectedRevision must be a canonical decimal saveRevision; got "00"`},
		"stale revision": {loaded.SaveSessionID, setRunesTestSlot, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := engine.SetCharacterRunes(
				testCase.sessionID, testCase.characterID, 500, testCase.expectedRevision)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
	assertSetRunesRejectedUnchanged(t, engine, loaded.SaveSessionID, before)
}

func TestSetCharacterRunesRejectsInactiveAndMissingAnchor(t *testing.T) {
	cases := []struct {
		name       string
		platform   Platform
		active     bool
		withAnchor bool
		want       string
	}{
		{"inactive", PlatformPC, false, false, "character 3 is not active"},
		{"missing anchor", PlatformPS4, true, false,
			"character 3 carries no statistics anchor"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeCharacterRunesFixture(
				t, testCase.platform, testCase.active, testCase.withAnchor, 123), string(testCase.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)
			_, err = engine.SetCharacterRunes(
				loaded.SaveSessionID, setRunesTestSlot, 500, "0")
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			assertSetRunesRejectedUnchanged(t, engine, loaded.SaveSessionID, before)
		})
	}
}

func TestSetCharacterRunesIdempotentAssignmentAdvancesRevision(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeCharacterRunesFixture(t, PlatformPC, true, true, 500), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	result, err := engine.SetCharacterRunes(
		loaded.SaveSessionID, setRunesTestSlot, 500, "0")
	if err != nil {
		t.Fatalf("SetCharacterRunes: %v", err)
	}
	if result.SaveRevision != "1" {
		t.Errorf("saveRevision = %q, want 1", result.SaveRevision)
	}
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) {
		t.Error("idempotent assignment changed the snapshot")
	}
}

func TestSetCharacterRunesPersistsAndReloadsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := gestureTestActiveFixture(
				platform, setRunesTestSlot, setRunesTestFullAnchorAt, 0)
			content.records = setGestureTestRecords()
			source := writeGestureFixture(t, content)
			data, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read full fixture: %v", err)
			}
			slotBase := int64(setRunesTestPCSlotBase) +
				setRunesTestSlot*setRunesTestPCSlotStride
			if platform == PlatformPS4 {
				slotBase = setRunesTestPS4SlotBase +
					setRunesTestSlot*setRunesTestPS4SlotStride
			}
			binary.LittleEndian.PutUint32(data[slotBase:], 0x6E)
			fieldAt := setRunesTestFieldAt(platform, setRunesTestFullAnchorAt)
			binary.LittleEndian.PutUint32(data[fieldAt:], 123)
			if err := os.WriteFile(source, data, 0o600); err != nil {
				t.Fatalf("rewrite full fixture: %v", err)
			}
			sourceBefore := bytes.Clone(data)

			engine := New()
			loaded, err := engine.LoadSave(source, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			if _, err := engine.SetCharacterRunes(
				loaded.SaveSessionID, setRunesTestSlot, setRunesTestMaximum, "0"); err != nil {
				t.Fatalf("SetCharacterRunes: %v", err)
			}
			sourceAfter, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read source after mutation: %v", err)
			}
			if !bytes.Equal(sourceBefore, sourceAfter) {
				t.Error("in-memory mutation changed the source file")
			}

			target := filepath.Join(t.TempDir(), "runes.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload target: %v", err)
			}
			raw, err := reloadedEngine.sessions[reloaded.SaveSessionID].snapshot.readAt(fieldAt, 4)
			if err != nil {
				t.Fatalf("read reloaded runes: %v", err)
			}
			if got := binary.LittleEndian.Uint32(raw); got != setRunesTestMaximum {
				t.Errorf("reloaded runes = %d, want %d", got, setRunesTestMaximum)
			}
		})
	}
}
