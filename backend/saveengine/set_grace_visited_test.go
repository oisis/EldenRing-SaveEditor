package saveengine

import (
	"bytes"
	"reflect"
	"testing"
)

// The grace flags this test writes. graceTestFlag and graceTestNeighbourFlag are
// neighbouring visit flags of block 76 sharing one byte, graceTestDoorFlag is the
// confirmed door flag of the grace 73000, and the four companion flags are the
// confirmed Gatefront set SaveEngine applies on its own — 4680 and 4681
// deliberately share one byte.
const (
	graceTestFlag          = 76100
	graceTestNeighbourFlag = 76101
	graceTestDungeonFlag   = 73000
	graceTestDoorFlag      = 1043338600
	graceTestGatefrontFlag = 76111
)

var graceTestCompanionFlags = []uint32{60100, 4680, 710520, 4681}

// changedSnapshotBytes reports the snapshot indices one mutation changed.
func changedSnapshotBytes(before, after []byte) []int {
	changed := make([]int, 0, 4)
	for index := range before {
		if before[index] != after[index] {
			changed = append(changed, index)
		}
	}
	return changed
}

func TestSetGraceVisitedMutatesOneBitOnBothPlatforms(t *testing.T) {
	// The two containers carry identical slot content, so only the platform base
	// differs and an engine that mixes the two bases cannot pass both cases.
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)
			content.set = []uint32{graceTestNeighbourFlag}

			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

			result, err := engine.SetGraceVisited(loaded.SaveSessionID, content.slot,
				graceTestFlag, 0, true, "0")
			if err != nil {
				t.Fatalf("SetGraceVisited: %v", err)
			}
			want := SetGraceVisitedResult{
				MutationReceipt: wantCommitReceipt(
					t, result.MutationReceipt, kindSetGraceVisited, loaded.SaveSessionID, "1"),
				CharacterID: content.slot,
				Visited:     true,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			// A grace without a door and without companions changes exactly one
			// byte, and inside it exactly the requested bit.
			after := engine.sessions[loaded.SaveSessionID].snapshot.data
			changed := changedSnapshotBytes(before, after)
			if len(changed) != 1 {
				t.Fatalf("changed snapshot bytes = %v, want exactly one", changed)
			}
			_, bit := eventFlagTestPosition(t, graceTestFlag)
			if delta := before[changed[0]] ^ after[changed[0]]; delta != 1<<bit {
				t.Errorf("changed bit mask = 0x%02X, want 0x%02X", delta, byte(1<<bit))
			}

			flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot,
				[]uint32{graceTestFlag, graceTestNeighbourFlag})
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}
			if !flags.Flags[graceTestFlag] || !flags.Flags[graceTestNeighbourFlag] {
				t.Errorf("flags = %v, want both the visited grace and its neighbour set", flags.Flags)
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

func TestSetGraceVisitedWritesTheDoorFlagSymmetrically(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	content.set = nil

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if _, err := engine.SetGraceVisited(loaded.SaveSessionID, content.slot,
		graceTestDungeonFlag, graceTestDoorFlag, true, "0"); err != nil {
		t.Fatalf("SetGraceVisited true: %v", err)
	}
	flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot,
		[]uint32{graceTestDungeonFlag, graceTestDoorFlag})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if !flags.Flags[graceTestDungeonFlag] || !flags.Flags[graceTestDoorFlag] {
		t.Fatalf("flags after activation = %v, want visit and door set", flags.Flags)
	}

	if _, err := engine.SetGraceVisited(loaded.SaveSessionID, content.slot,
		graceTestDungeonFlag, graceTestDoorFlag, false, "1"); err != nil {
		t.Fatalf("SetGraceVisited false: %v", err)
	}
	flags, err = engine.GetEventFlags(loaded.SaveSessionID, content.slot,
		[]uint32{graceTestDungeonFlag, graceTestDoorFlag})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if flags.Flags[graceTestDungeonFlag] || flags.Flags[graceTestDoorFlag] {
		t.Fatalf("flags after deactivation = %v, want visit and door cleared", flags.Flags)
	}
}

func TestSetGraceVisitedAppliesTheGatefrontCompanionsSetOnly(t *testing.T) {
	requested := append([]uint32{graceTestGatefrontFlag}, graceTestCompanionFlags...)

	// SaveEngine owns the companion set: the call names the visit flag only, so a
	// caller can neither add a flag to nor drop one from the confirmed four.
	t.Run("activation sets every companion", func(t *testing.T) {
		content := eventFlagTestContent(PlatformPC)
		content.set = nil

		engine := New()
		loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

		if _, err := engine.SetGraceVisited(loaded.SaveSessionID, content.slot,
			graceTestGatefrontFlag, 0, true, "0"); err != nil {
			t.Fatalf("SetGraceVisited: %v", err)
		}

		flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, requested)
		if err != nil {
			t.Fatalf("GetEventFlags: %v", err)
		}
		for _, id := range requested {
			if !flags.Flags[id] {
				t.Errorf("flag %d is false, want true", id)
			}
		}

		// 4680 and 4681 share one byte. Both bits above are only set when the two
		// targets were merged into a single write; an unmerged plan would compute
		// the second byte from the original value and drop the first bit. The four
		// distinct target bytes prove no unrelated byte was touched.
		changed := changedSnapshotBytes(before, engine.sessions[loaded.SaveSessionID].snapshot.data)
		if len(changed) != 4 {
			t.Fatalf("changed snapshot bytes = %v, want the four grace target bytes", changed)
		}
	})

	t.Run("deactivation leaves every companion untouched", func(t *testing.T) {
		content := eventFlagTestContent(PlatformPC)
		content.set = append([]uint32{graceTestGatefrontFlag}, graceTestCompanionFlags...)

		engine := New()
		loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

		if _, err := engine.SetGraceVisited(loaded.SaveSessionID, content.slot,
			graceTestGatefrontFlag, 0, false, "0"); err != nil {
			t.Fatalf("SetGraceVisited: %v", err)
		}

		flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, requested)
		if err != nil {
			t.Fatalf("GetEventFlags: %v", err)
		}
		if flags.Flags[graceTestGatefrontFlag] {
			t.Error("the visit flag is still set after deactivation")
		}
		for _, id := range graceTestCompanionFlags {
			if !flags.Flags[id] {
				t.Errorf("companion flag %d was cleared, want it untouched", id)
			}
		}
		if changed := changedSnapshotBytes(
			before, engine.sessions[loaded.SaveSessionID].snapshot.data); len(changed) != 1 {
			t.Fatalf("changed snapshot bytes = %v, want only the visit byte", changed)
		}
	})

	// Gatefront is a closed exception keyed on its own visit flag, so no other
	// grace may drag the four progression flags into a save.
	t.Run("an ordinary grace sets no companion", func(t *testing.T) {
		content := eventFlagTestContent(PlatformPC)
		content.set = nil

		engine := New()
		loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

		if _, err := engine.SetGraceVisited(loaded.SaveSessionID, content.slot,
			graceTestFlag, 0, true, "0"); err != nil {
			t.Fatalf("SetGraceVisited: %v", err)
		}

		flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, graceTestCompanionFlags)
		if err != nil {
			t.Fatalf("GetEventFlags: %v", err)
		}
		for _, id := range graceTestCompanionFlags {
			if flags.Flags[id] {
				t.Errorf("companion flag %d is set after an ordinary grace visit", id)
			}
		}
		if changed := changedSnapshotBytes(
			before, engine.sessions[loaded.SaveSessionID].snapshot.data); len(changed) != 1 {
			t.Fatalf("changed snapshot bytes = %v, want only the visit byte", changed)
		}
	})
}

func TestSetGraceVisitedRejectsWithoutMutating(t *testing.T) {
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

	cases := map[string]struct {
		sessionID   string
		characterID int
		visitFlag   uint32
		doorFlag    uint32
		visited     bool
		revision    string
		want        string
	}{
		"stale revision": {
			activeLoaded.SaveSessionID, active.slot, graceTestFlag, 0, true, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`,
		},
		"non-canonical revision": {
			activeLoaded.SaveSessionID, active.slot, graceTestFlag, 0, true, "01",
			`expectedRevision must be a canonical decimal saveRevision; got "01"`,
		},
		"inactive slot": {
			inactiveLoaded.SaveSessionID, inactive.slot, graceTestFlag, 0, true, "0",
			"character 3 is not active",
		},
		// A flag of another supported block must not be reachable through the
		// grace entry point: block 670 carries summoning pool activations.
		"visit flag outside the grace blocks": {
			activeLoaded.SaveSessionID, active.slot, 670130, 0, true, "0",
			"event flag 670130 lies in block 670, which is not a confirmed grace block [71 72 73 74 76]",
		},
		// A door flag whose block carries no confirmed BST position must fail
		// closed instead of being written at a guessed offset.
		"door flag in an unsupported block": {
			activeLoaded.SaveSessionID, active.slot, graceTestFlag, 999000, true, "0",
			"event flag 999000 lies in block 999, which is not a confirmed grace door block",
		},
		// resolveEventFlag answers block 4, but it carries the Gatefront companion
		// flags rather than a door, so it must not be writable as a door either.
		"door flag in a supported non-door block": {
			activeLoaded.SaveSessionID, active.slot, graceTestFlag, 4680, true, "0",
			"event flag 4680 lies in block 4, which is not a confirmed grace door block",
		},
		// A visit flag repeated as the door flag lies outside the door blocks and is
		// rejected by that bound.
		"door flag equal to the visit flag": {
			activeLoaded.SaveSessionID, active.slot, graceTestFlag, graceTestFlag, true, "0",
			"event flag 76100 lies in block 76, which is not a confirmed grace door block",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			session := engine.sessions[testCase.sessionID]
			before := bytes.Clone(session.snapshot.data)

			_, err := engine.SetGraceVisited(testCase.sessionID, testCase.characterID,
				testCase.visitFlag, testCase.doorFlag, testCase.visited, testCase.revision)
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
