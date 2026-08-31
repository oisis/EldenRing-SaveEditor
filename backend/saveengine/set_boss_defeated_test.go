package saveengine

import (
	"bytes"
	"testing"
)

// 9100 and 9101 are neighbouring synchronized defeat flags of the curated list
// and live in the same byte of block 9, so a mutation that touches its
// neighbour or writes a shifted byte fails here.
const (
	bossTestFlag          = 9100
	bossTestNeighbourFlag = 9101
)

func TestSetBossDefeatedMutatesOneBitOnBothPlatforms(t *testing.T) {
	// The two containers carry identical slot content, so only the platform base
	// differs and an engine that mixes the two bases cannot pass both cases.
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)
			content.set = []uint32{bossTestNeighbourFlag}

			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

			result, err := engine.SetBossDefeated(
				loaded.SaveSessionID, content.slot, bossTestFlag, true, "0")
			if err != nil {
				t.Fatalf("SetBossDefeated: %v", err)
			}
			want := SetBossDefeatedResult{
				SaveSessionID: loaded.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   content.slot,
				Defeated:      true,
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
			_, bit := eventFlagTestPosition(t, bossTestFlag)
			if delta := before[changed[0]] ^ after[changed[0]]; delta != 1<<bit {
				t.Errorf("changed bit mask = 0x%02X, want 0x%02X", delta, byte(1<<bit))
			}

			flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot,
				[]uint32{bossTestFlag, bossTestNeighbourFlag})
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}
			if !flags.Flags[bossTestFlag] {
				t.Errorf("flag %d is false, want true", bossTestFlag)
			}
			if !flags.Flags[bossTestNeighbourFlag] {
				t.Errorf("flag %d is false, want the neighbouring bit preserved",
					bossTestNeighbourFlag)
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

func TestSetBossDefeatedClearsOneBitAndKeepsItsNeighbour(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	content.set = []uint32{bossTestFlag, bossTestNeighbourFlag}

	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.SetBossDefeated(
		loaded.SaveSessionID, content.slot, bossTestFlag, false, "0")
	if err != nil {
		t.Fatalf("SetBossDefeated: %v", err)
	}
	if result.SaveRevision != "1" || result.Defeated {
		t.Errorf("result = %+v, want revision 1 and defeated false", result)
	}

	flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot,
		[]uint32{bossTestFlag, bossTestNeighbourFlag})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if flags.Flags[bossTestFlag] {
		t.Errorf("flag %d is true, want false", bossTestFlag)
	}
	if !flags.Flags[bossTestNeighbourFlag] {
		t.Errorf("flag %d is false, want the neighbouring bit in the same byte preserved",
			bossTestNeighbourFlag)
	}
}

func TestSetBossDefeatedRejectsWithoutMutating(t *testing.T) {
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
		flag        uint32
		revision    string
		want        string
	}{
		"stale revision": {
			activeLoaded.SaveSessionID, active.slot, bossTestFlag, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`,
		},
		"non-canonical revision": {
			activeLoaded.SaveSessionID, active.slot, bossTestFlag, "01",
			`expectedRevision must be a canonical decimal saveRevision; got "01"`,
		},
		"inactive slot": {
			inactiveLoaded.SaveSessionID, inactive.slot, bossTestFlag, "0",
			"character 3 is not active",
		},
		// A flag of another supported block must not be reachable through the
		// boss entry point: block 670 carries summoning pool activations.
		"flag outside block 9": {
			activeLoaded.SaveSessionID, active.slot, 670130, "0",
			"event flag 670130 lies in block 670, which is not the confirmed boss block 9",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			session, known := engine.sessions[testCase.sessionID]
			var before []byte
			if known {
				before = bytes.Clone(session.snapshot.data)
			}

			_, err := engine.SetBossDefeated(testCase.sessionID,
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
