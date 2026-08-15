package saveengine

import (
	"bytes"
	"testing"
)

// 670130 and 670131 are neighbouring activation flags of the curated list and
// live in the same byte of block 670, so a mutation that touches its neighbour
// or writes a shifted byte fails here.
const (
	summoningPoolTestFlag          = 670130
	summoningPoolTestNeighbourFlag = 670131
)

func TestSetSummoningPoolActivatedMutatesOneBitOnBothPlatforms(t *testing.T) {
	// The two containers carry identical slot content, so only the platform base
	// differs and an engine that mixes the two bases cannot pass both cases.
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)
			content.set = []uint32{summoningPoolTestNeighbourFlag}

			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

			result, err := engine.SetSummoningPoolActivated(
				loaded.SaveSessionID, content.slot, summoningPoolTestFlag, true, "0")
			if err != nil {
				t.Fatalf("SetSummoningPoolActivated: %v", err)
			}
			want := SetSummoningPoolActivatedResult{
				SaveSessionID: loaded.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   content.slot,
				Activated:     true,
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			// Exactly one byte changed, and inside it exactly the requested bit.
			after := engine.sessions[loaded.SaveSessionID].snapshot.data
			changed := make([]int, 0, 1)
			for index := range before {
				if before[index] != after[index] {
					changed = append(changed, index)
				}
			}
			if len(changed) != 1 {
				t.Fatalf("changed snapshot bytes = %v, want exactly one", changed)
			}
			_, bit := eventFlagTestPosition(t, summoningPoolTestFlag)
			if delta := before[changed[0]] ^ after[changed[0]]; delta != 1<<bit {
				t.Errorf("changed bit mask = 0x%02X, want 0x%02X", delta, byte(1<<bit))
			}

			flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot,
				[]uint32{summoningPoolTestFlag, summoningPoolTestNeighbourFlag})
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}
			if !flags.Flags[summoningPoolTestFlag] {
				t.Errorf("flag %d is false, want true", summoningPoolTestFlag)
			}
			if !flags.Flags[summoningPoolTestNeighbourFlag] {
				t.Errorf("flag %d is false, want the neighbouring bit preserved",
					summoningPoolTestNeighbourFlag)
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

func TestSetSummoningPoolActivatedClearsOneBitAndKeepsItsNeighbour(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	content.set = []uint32{summoningPoolTestFlag, summoningPoolTestNeighbourFlag}

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.SetSummoningPoolActivated(
		loaded.SaveSessionID, content.slot, summoningPoolTestFlag, false, "0")
	if err != nil {
		t.Fatalf("SetSummoningPoolActivated: %v", err)
	}
	if result.SaveRevision != "1" || result.Activated {
		t.Errorf("result = %+v, want revision 1 and activated false", result)
	}

	flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot,
		[]uint32{summoningPoolTestFlag, summoningPoolTestNeighbourFlag})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if flags.Flags[summoningPoolTestFlag] {
		t.Errorf("flag %d is true, want false", summoningPoolTestFlag)
	}
	if !flags.Flags[summoningPoolTestNeighbourFlag] {
		t.Errorf("flag %d is false, want the neighbouring bit in the same byte preserved",
			summoningPoolTestNeighbourFlag)
	}
}

func TestSetSummoningPoolActivatedRejectsWithoutMutating(t *testing.T) {
	engine := New()
	active := eventFlagTestContent(PlatformPC)
	activeLoaded, err := engine.LoadSave(writeEventFlagFixture(t, active), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inactive := eventFlagTestContent(PlatformPC)
	inactive.flag = 0
	inactiveLoaded, err := engine.LoadSave(writeEventFlagFixture(t, inactive), "")
	if err != nil {
		t.Fatalf("LoadSave inactive: %v", err)
	}

	cases := map[string]struct {
		sessionID   string
		characterID int
		flag        uint32
		revision    string
		want        string
	}{
		"stale revision": {
			activeLoaded.SaveSessionID, active.slot, summoningPoolTestFlag, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`,
		},
		"non-canonical revision": {
			activeLoaded.SaveSessionID, active.slot, summoningPoolTestFlag, "01",
			`expectedRevision must be a canonical decimal saveRevision; got "01"`,
		},
		"inactive slot": {
			inactiveLoaded.SaveSessionID, inactive.slot, summoningPoolTestFlag, "0",
			"character 3 is not active",
		},
		// A flag of another supported block must not be reachable through the
		// summoning pool entry point: block 71 carries grace visits.
		"flag outside block 670": {
			activeLoaded.SaveSessionID, active.slot, 71000, "0",
			"event flag 71000 lies in block 71, which is not the confirmed summoning pool block 670",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			session, known := engine.sessions[testCase.sessionID]
			var before []byte
			if known {
				before = bytes.Clone(session.snapshot.data)
			}

			_, err := engine.SetSummoningPoolActivated(testCase.sessionID,
				testCase.characterID, testCase.flag, true, testCase.revision)
			if err == nil {
				t.Fatalf("accepted %s", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !known {
				return
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
