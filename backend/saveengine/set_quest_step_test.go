package saveengine

import (
	"reflect"
	"testing"
)

func TestSetQuestStepMutatesBitsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := eventFlagTestContent(platform)
			engine := New()
			loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			// Brother Corhyn step legacy_000 plan
			plan := []QuestFlagTarget{
				{ID: 60841, Value: true},
				{ID: 11009456, Value: false},
				{ID: 11009458, Value: false},
				{ID: 11102812, Value: true},
				{ID: 11102814, Value: true},
			}

			result, err := engine.SetQuestStep(
				loaded.SaveSessionID,
				content.slot,
				plan,
				"0",
			)
			if err != nil {
				t.Fatalf("SetQuestStep: %v", err)
			}
			if result.SaveRevision != "1" {
				t.Errorf("saveRevision = %q, want 1", result.SaveRevision)
			}
			if result.CharacterID != content.slot {
				t.Errorf("characterID = %d, want %d", result.CharacterID, content.slot)
			}

			// Verify with GetEventFlags
			req := []uint32{60841, 11009456, 11009458, 11102812, 11102814}
			read, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, req)
			if err != nil {
				t.Fatalf("GetEventFlags: %v", err)
			}

			want := map[uint32]bool{
				60841:    true,
				11009456: false,
				11009458: false,
				11102812: true,
				11102814: true,
			}
			if !reflect.DeepEqual(read.Flags, want) {
				t.Errorf("flags = %+v, want %+v", read.Flags, want)
			}
		})
	}
}

func TestSetQuestStepMergesMultipleFlagsInSameByte(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	// 60840, 60841 and 60842 share one byte in block 60. Only the first two are
	// planned, so the pre-set 60842 proves the merged single-byte write keeps
	// every bit it does not target.
	content.set = append(content.set, 60842)
	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	plan := []QuestFlagTarget{
		{ID: 60840, Value: false},
		{ID: 60841, Value: true},
	}

	_, err = engine.SetQuestStep(
		loaded.SaveSessionID,
		content.slot,
		plan,
		"0",
	)
	if err != nil {
		t.Fatalf("SetQuestStep: %v", err)
	}

	read, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot, []uint32{60840, 60841, 60842})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}

	if read.Flags[60840] != false {
		t.Errorf("flag 60840 = true, want false")
	}
	if read.Flags[60841] != true {
		t.Errorf("flag 60841 = false, want true")
	}
	if read.Flags[60842] != true {
		t.Errorf("untargeted flag 60842 = false, want the pre-set true")
	}
}

func TestSetQuestStepRejectsWithoutMutating(t *testing.T) {
	content := eventFlagTestContent(PlatformPC)
	engine := New()
	loaded, err := engine.LoadSave(writeEventFlagFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	validPlan := []QuestFlagTarget{
		{ID: 60841, Value: true},
	}

	cases := map[string]struct {
		slot     int
		plan     []QuestFlagTarget
		revision string
	}{
		"stale_revision": {
			slot:     content.slot,
			plan:     validPlan,
			revision: "999",
		},
		"non-canonical_revision": {
			slot:     content.slot,
			plan:     validPlan,
			revision: "01",
		},
		"inactive_slot": {
			slot:     content.slot + 1,
			plan:     validPlan,
			revision: "0",
		},
		"empty_flags": {
			slot:     content.slot,
			plan:     nil,
			revision: "0",
		},
		"unsupported_block_flag": {
			slot:     content.slot,
			plan:     []QuestFlagTarget{{ID: 5007, Value: true}},
			revision: "0",
		},
		"duplicate_flag_id": {
			slot: content.slot,
			plan: []QuestFlagTarget{
				{ID: 60841, Value: true},
				{ID: 60841, Value: false},
			},
			revision: "0",
		},
		"zero_flag_id": {
			slot:     content.slot,
			plan:     []QuestFlagTarget{{ID: 0, Value: true}},
			revision: "0",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := engine.SetQuestStep(
				loaded.SaveSessionID,
				tc.slot,
				tc.plan,
				tc.revision,
			)
			if err == nil {
				t.Fatalf("SetQuestStep accepted invalid request %q", name)
			}
		})
	}

	// Every rejection above must leave the session exactly as it was loaded:
	// no committed byte, no dirty flag and no revision advance.
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejections = %+v, want clean", info)
	}
	if _, err := engine.SetQuestStep(
		loaded.SaveSessionID, content.slot, validPlan, "0"); err != nil {
		t.Errorf("saveRevision moved despite only rejected requests: %v", err)
	}
}
