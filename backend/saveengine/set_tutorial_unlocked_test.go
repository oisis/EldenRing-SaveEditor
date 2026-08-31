package saveengine

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

const setTutorialSlot = 3

// setTutorialPayloadSize holds eight IDs: (0x24 - 4) / 4.
const setTutorialPayloadSize = 0x24

func loadTutorialSession(
	t *testing.T, platform Platform, size uint32, ids []uint32,
) (*Engine, string) {
	t.Helper()

	engine := New()
	loaded, err := engine.LoadSave(
		writeTutorialDataFixture(t, platform, true, size, ids), string(platform), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func tutorialIDsOf(t *testing.T, engine *Engine, sessionID string) []uint32 {
	t.Helper()

	state, err := engine.GetTutorialIDs(sessionID, setTutorialSlot)
	if err != nil {
		t.Fatalf("GetTutorialIDs: %v", err)
	}
	return state.IDs
}

func TestSetTutorialUnlockedAddsAscendingAndRemovesOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, sessionID := loadTutorialSession(
				t, platform, setTutorialPayloadSize, []uint32{1000, 2010, 3000})

			if _, err := engine.SetTutorialUnlocked(
				sessionID, setTutorialSlot, 1590, true, "0"); err != nil {
				t.Fatalf("unlock: %v", err)
			}
			if got := tutorialIDsOf(t, engine, sessionID); !reflect.DeepEqual(
				got, []uint32{1000, 1590, 2010, 3000}) {
				t.Fatalf("after unlock = %v", got)
			}

			if _, err := engine.SetTutorialUnlocked(
				sessionID, setTutorialSlot, 2010, false, "1"); err != nil {
				t.Fatalf("lock: %v", err)
			}
			if got := tutorialIDsOf(t, engine, sessionID); !reflect.DeepEqual(
				got, []uint32{1000, 1590, 3000}) {
				t.Fatalf("after lock = %v", got)
			}
		})
	}
}

func TestSetTutorialUnlockedRoundTripRestoresThePayload(t *testing.T) {
	engine, sessionID := loadTutorialSession(
		t, PlatformPC, setTutorialPayloadSize, []uint32{1000, 3000})
	before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

	if _, err := engine.SetTutorialUnlocked(
		sessionID, setTutorialSlot, 1590, true, "0"); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if bytes.Equal(engine.sessions[sessionID].snapshot.data, before) {
		t.Fatal("unlock changed no byte")
	}
	if _, err := engine.SetTutorialUnlocked(
		sessionID, setTutorialSlot, 1590, false, "1"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if !bytes.Equal(engine.sessions[sessionID].snapshot.data, before) {
		t.Fatal("the round trip did not restore the original payload")
	}
}

func TestSetTutorialUnlockedIsIdempotentWithoutChangingAByte(t *testing.T) {
	engine, sessionID := loadTutorialSession(
		t, PlatformPC, setTutorialPayloadSize, []uint32{1590})
	before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

	if _, err := engine.SetTutorialUnlocked(
		sessionID, setTutorialSlot, 1590, true, "0"); err != nil {
		t.Fatalf("idempotent unlock: %v", err)
	}
	if _, err := engine.SetTutorialUnlocked(
		sessionID, setTutorialSlot, 2010, false, "1"); err != nil {
		t.Fatalf("idempotent lock: %v", err)
	}
	if !bytes.Equal(engine.sessions[sessionID].snapshot.data, before) {
		t.Fatal("an idempotent request changed a byte")
	}

	undo, err := engine.GetUndoState(sessionID, setTutorialSlot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undo.Available || undo.SaveRevision != "2" {
		t.Fatalf("undo state = %+v, want no point at revision 2", undo)
	}
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if !info.UnsavedChanges {
		t.Fatalf("session info = %+v, want dirty", info)
	}
}

func TestSetTutorialUnlockedAppendsToAnUnsortedLegacyListAndKeepsUnknownIDs(t *testing.T) {
	engine, sessionID := loadTutorialSession(
		t, PlatformPC, setTutorialPayloadSize, []uint32{3000, 1000, 999999})

	if _, err := engine.SetTutorialUnlocked(
		sessionID, setTutorialSlot, 1590, true, "0"); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if got := tutorialIDsOf(t, engine, sessionID); !reflect.DeepEqual(
		got, []uint32{3000, 1000, 999999, 1590}) {
		t.Fatalf("legacy list = %v, want the original order plus an appended ID", got)
	}
}

func TestSetTutorialUnlockedRemovesEveryDuplicateOfTheRequestedID(t *testing.T) {
	engine, sessionID := loadTutorialSession(
		t, PlatformPC, setTutorialPayloadSize, []uint32{1590, 2010, 1590, 999999})

	if _, err := engine.SetTutorialUnlocked(
		sessionID, setTutorialSlot, 1590, false, "0"); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if got := tutorialIDsOf(t, engine, sessionID); !reflect.DeepEqual(
		got, []uint32{2010, 999999}) {
		t.Fatalf("after removal = %v", got)
	}

	// The two freed entries must be zeroed, so re-adding and removing the same
	// ID returns to exactly this payload.
	after := bytes.Clone(engine.sessions[sessionID].snapshot.data)
	if _, err := engine.SetTutorialUnlocked(
		sessionID, setTutorialSlot, 1590, true, "1"); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if _, err := engine.SetTutorialUnlocked(
		sessionID, setTutorialSlot, 1590, false, "2"); err != nil {
		t.Fatalf("lock again: %v", err)
	}
	if !bytes.Equal(engine.sessions[sessionID].snapshot.data, after) {
		t.Fatal("freed tutorial entries were not zeroed")
	}
}

func TestSetTutorialUnlockedFailsClosed(t *testing.T) {
	full := []uint32{1, 2, 3, 4, 5, 6, 7, 8}
	cases := map[string]struct {
		size     uint32
		ids      []uint32
		active   bool
		revision string
		want     string
	}{
		"full list": {
			size: setTutorialPayloadSize, ids: full, active: true,
			revision: "0", want: "TutorialData is full",
		},
		"malformed count": {
			size: 8, ids: []uint32{2010, 2020}, active: true,
			revision: "0", want: "tutorial count",
		},
		"payload too small for the count": {
			size: tutorialDataCountSize - 1, ids: nil, active: true,
			revision: "0", want: "does not hold the 4-byte tutorial count field",
		},
		"stale revision": {
			size: setTutorialPayloadSize, ids: []uint32{2010}, active: true,
			revision: "7", want: "expectedRevision",
		},
		"inactive slot": {
			active: false, revision: "0", want: "is not active",
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeTutorialDataFixture(t, PlatformPC, testCase.active, testCase.size, testCase.ids),
				"pc", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

			_, err = engine.SetTutorialUnlocked(
				loaded.SaveSessionID, setTutorialSlot, 1590, true, testCase.revision)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want one containing %q", err, testCase.want)
			}
			if !bytes.Equal(engine.sessions[loaded.SaveSessionID].snapshot.data, before) {
				t.Fatal("a rejected request changed the snapshot")
			}

			undo, err := engine.GetUndoState(loaded.SaveSessionID, setTutorialSlot)
			if err != nil {
				t.Fatalf("GetUndoState: %v", err)
			}
			if undo.Available || undo.SaveRevision != "0" {
				t.Fatalf("undo state = %+v, want no point at revision 0", undo)
			}
			info, err := engine.GetSessionInfo(loaded.SaveSessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if info.UnsavedChanges {
				t.Fatalf("session info = %+v, want a clean session", info)
			}
		})
	}
}
