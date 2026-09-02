package saveengine

import (
	"bytes"
	"reflect"
	"testing"
)

// The confirmed flag sets this test drives: the Caelid activation flag with its
// four members, the Limgrave activation flag that must stay untouched, and the
// three global SET-only flags.
const colosseumTestCaelidFlag = 60350

var (
	colosseumTestCaelidSet   = []uint32{60350, 62720, 69450, 710850}
	colosseumTestGlobalFlags = []uint32{6080, 60100, 69480}
	colosseumTestLimgraveSet = []uint32{60360, 62730, 69460, 710860}
)

func TestSetColosseumUnlockedWritesTheCompleteSetOnBothPlatforms(t *testing.T) {
	// The two containers carry identical slot content, so only the platform base
	// differs and an engine that mixes the two bases cannot pass both cases.
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)
			content.set = nil

			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

			result, err := engine.SetColosseumUnlocked(loaded.SaveSessionID, content.slot,
				colosseumTestCaelidFlag, true, "0")
			if err != nil {
				t.Fatalf("SetColosseumUnlocked: %v", err)
			}
			want := SetColosseumUnlockedResult{
				MutationReceipt: wantCommitReceipt(
					t, result.MutationReceipt, kindSetColosseumUnlocked, loaded.SaveSessionID, "1"),
				CharacterID: content.slot,
				Unlocked:    true,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			// The four colosseum flags and the three globals lie in seven distinct
			// bytes, so the general invariant is checkable directly: exactly those
			// bytes changed, and inside each of them exactly the expected bit.
			expected := append(append([]uint32{}, colosseumTestCaelidSet...), colosseumTestGlobalFlags...)
			after := engine.sessions[loaded.SaveSessionID].snapshot.data
			changed := changedSnapshotBytes(before, after)
			if len(changed) != len(expected) {
				t.Fatalf("changed snapshot bytes = %v, want the %d colosseum target bytes",
					changed, len(expected))
			}
			flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot,
				append(expected, colosseumTestLimgraveSet...))
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}
			for _, id := range expected {
				if !flags.Flags[id] {
					t.Errorf("flag %d is false, want true", id)
				}
			}
			// A second arena must not ride along on the shared globals.
			for _, id := range colosseumTestLimgraveSet {
				if flags.Flags[id] {
					t.Errorf("flag %d of another colosseum was set", id)
				}
			}

			info, err := engine.GetSessionInfo(loaded.SaveSessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if !info.UnsavedChanges {
				t.Error("UnsavedChanges = false, want true after mutation")
			}
		})
	}
}

// Locking one colosseum may never clear the globals: another arena can still own
// them and 60100 is shared with the Torrent progression of the grace mutation.
func TestSetColosseumUnlockedClearsTheArenaFlagsOnly(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	content.set = append(append([]uint32{}, colosseumTestCaelidSet...), colosseumTestGlobalFlags...)

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	if _, err := engine.SetColosseumUnlocked(loaded.SaveSessionID, content.slot,
		colosseumTestCaelidFlag, false, "0"); err != nil {
		t.Fatalf("SetColosseumUnlocked: %v", err)
	}

	changed := changedSnapshotBytes(before, engine.sessions[loaded.SaveSessionID].snapshot.data)
	if len(changed) != len(colosseumTestCaelidSet) {
		t.Fatalf("changed snapshot bytes = %v, want only the four arena bytes", changed)
	}
	flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot,
		append(append([]uint32{}, colosseumTestCaelidSet...), colosseumTestGlobalFlags...))
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	for _, id := range colosseumTestCaelidSet {
		if flags.Flags[id] {
			t.Errorf("arena flag %d is still set after locking", id)
		}
	}
	for _, id := range colosseumTestGlobalFlags {
		if !flags.Flags[id] {
			t.Errorf("global flag %d was cleared, want it untouched", id)
		}
	}
}

func TestSetColosseumUnlockedRejectsWithoutMutating(t *testing.T) {
	engine := New()
	active := eventFlagTestContent(PlatformPC)
	activeLoaded, err := engine.LoadSave(writeEventFlagFixture(t, active), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inactive := eventFlagTestContent(PlatformPC)
	inactive.flag = 0
	inactiveLoaded, err := engine.LoadSave(writeEventFlagFixture(t, inactive), "", "local")
	if err != nil {
		t.Fatalf("LoadSave inactive: %v", err)
	}

	const unconfirmed = "is not a confirmed colosseum unlock flag [60350 60360 60370]"
	cases := map[string]struct {
		sessionID   string
		characterID int
		eventFlagID uint32
		revision    string
		want        string
	}{
		"stale revision": {
			activeLoaded.SaveSessionID, active.slot, colosseumTestCaelidFlag, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`,
		},
		"non-canonical revision": {
			activeLoaded.SaveSessionID, active.slot, colosseumTestCaelidFlag, "01",
			`expectedRevision must be a canonical decimal saveRevision; got "01"`,
		},
		"inactive slot": {
			inactiveLoaded.SaveSessionID, inactive.slot, colosseumTestCaelidFlag, "0",
			"character 3 is not active",
		},
		// Another flag of the supported block 60 must not be reachable: there is
		// no fallback that would treat an arbitrary block 60 flag as an arena.
		"unconfirmed flag of the supported block": {
			activeLoaded.SaveSessionID, active.slot, 60100, "0",
			"event flag 60100 " + unconfirmed,
		},
		// A derivative member of a confirmed set is not an entry point either.
		"derivative flag of a confirmed set": {
			activeLoaded.SaveSessionID, active.slot, 62720, "0",
			"event flag 62720 " + unconfirmed,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			session := engine.sessions[testCase.sessionID]
			before := bytes.Clone(session.snapshot.data)

			_, err := engine.SetColosseumUnlocked(testCase.sessionID, testCase.characterID,
				testCase.eventFlagID, true, testCase.revision)
			if err == nil {
				t.Fatalf("accepted %s", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !bytes.Equal(before, session.snapshot.data) {
				t.Error("rejected mutation changed the private snapshot")
			}
			if revision := session.session.revisionString(); revision != "0" {
				t.Errorf("revision after rejection = %q, want 0", revision)
			}
			info, err := engine.GetSessionInfo(testCase.sessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if info.UnsavedChanges {
				t.Errorf("session after rejection = %+v, want clean", info)
			}
		})
	}
}
