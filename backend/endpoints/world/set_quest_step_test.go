package world

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSetQuestStepAppliesTheStepPlan(t *testing.T) {
	engine, sessionID := loadBossesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	questKey := "brother_corhyn"
	stepKey := "legacy_000"

	result, err := SetQuestStep(
		engine,
		gameCatalog,
		sessionID,
		getCookbooksSlot,
		"quest",
		questKey,
		"quest_step",
		stepKey,
		"0",
	)
	if err != nil {
		t.Fatalf("SetQuestStep: %v", err)
	}

	want := SetQuestStepResult{
		SaveSessionID: sessionID,
		SaveRevision:  "1",
		CharacterID:   getCookbooksSlot,
		QuestKind:     schema.ResourceKindQuest,
		QuestKey:      questKey,
		StepKind:      "quest_step",
		StepKey:       stepKey,
	}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	// Verify flags were written
	read, err := engine.GetEventFlags(sessionID, getCookbooksSlot, []uint32{60841, 11009456, 11102812})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if read.Flags[60841] != true {
		t.Errorf("flag 60841 = false, want true")
	}
	if read.Flags[11009456] != false {
		t.Errorf("flag 11009456 = true, want false")
	}
	if read.Flags[11102812] != true {
		t.Errorf("flag 11102812 = false, want true")
	}
}

func TestSetQuestStepRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadBossesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	questKey := "brother_corhyn"
	stepKey := "legacy_000"

	if _, err := SetQuestStep(nil, gameCatalog, sessionID, getCookbooksSlot,
		"quest", questKey, "quest_step", stepKey, "0"); err == nil ||
		err.Error() != "save engine is not available" {
		t.Errorf("nil SaveEngine error = %v, want \"save engine is not available\"", err)
	}
	if _, err := SetQuestStep(engine, nil, sessionID, getCookbooksSlot,
		"quest", questKey, "quest_step", stepKey, "0"); err == nil ||
		err.Error() != "game catalog is not available" {
		t.Errorf("nil GameCatalog error = %v, want \"game catalog is not available\"", err)
	}

	cases := map[string]struct {
		questKind string
		questKey  string
		stepKind  string
		stepKey   string
		revision  string
		want      string
	}{
		"wrong_quest_kind": {
			questKind: "item",
			questKey:  questKey,
			stepKind:  "quest_step",
			stepKey:   stepKey,
			revision:  "0",
			want:      `resource kind "item" is not "quest"`,
		},
		"wrong_step_kind": {
			questKind: "quest",
			questKey:  questKey,
			stepKind:  "step",
			stepKey:   stepKey,
			revision:  "0",
			want:      `step kind "step" is not "quest_step"`,
		},
		"unknown_quest_key": {
			questKind: "quest",
			questKey:  "unknown_npc",
			stepKind:  "quest_step",
			stepKey:   stepKey,
			revision:  "0",
			want:      `unknown resource key "unknown_npc" in kind "quest"`,
		},
		"unknown_step_key": {
			questKind: "quest",
			questKey:  questKey,
			stepKind:  "quest_step",
			stepKey:   "legacy_999",
			revision:  "0",
			want:      `unknown step key "legacy_999" in quest "brother_corhyn"`,
		},
		"stale_revision": {
			questKind: "quest",
			questKey:  questKey,
			stepKind:  "quest_step",
			stepKey:   stepKey,
			revision:  "999",
			want:      `expectedRevision "999" does not match the current saveRevision "0"`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := SetQuestStep(
				engine,
				gameCatalog,
				sessionID,
				getCookbooksSlot,
				tc.questKind,
				tc.questKey,
				tc.stepKind,
				tc.stepKey,
				tc.revision,
			)
			if err == nil {
				t.Fatalf("SetQuestStep accepted invalid request %q", name)
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err, tc.want)
			}
		})
	}

	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejections = %+v, want clean", info)
	}
}
