package world

import (
	"slices"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	// The curated questlines the stored catalog declares, and the questline the
	// step assertions use.
	getQuestsCount     = 36
	getQuestsKey       = "brother_corhyn"
	getQuestsStepCount = 42

	// On the all-cleared bitfield of the fixture exactly these two Brother Corhyn
	// plans hold their declared target values, because every flag they declare is
	// a cleared one. They are neither adjacent nor the first step, so a match
	// derived from step order or index would fail here.
	getQuestsMatchedFirst  = "legacy_001"
	getQuestsMatchedSecond = "legacy_005"

	// The plan of this step sets flags, so it only matches after SetQuestStep
	// applied it.
	getQuestsAppliedStep = "legacy_000"

	// Fia is the confirmed case of several plans matching the same save state at
	// once: the getter reports all of them and names none of them current.
	getQuestsMultiMatchKey   = "fia"
	getQuestsMultiMatchCount = 5
)

func matchedStepKeys(t *testing.T, result GetQuestsResult, questKey string) []string {
	t.Helper()

	for _, quest := range result.Quests {
		if quest.Key != questKey {
			continue
		}
		matched := make([]string, 0, len(quest.Steps))
		for _, step := range quest.Steps {
			if step.Matched {
				matched = append(matched, step.StepKey)
			}
		}
		return matched
	}
	t.Fatalf("result carries no quest %q", questKey)
	return nil
}

func TestGetQuestsReportsEveryMatchingStepPlan(t *testing.T) {
	engine, sessionID := loadBossesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	result, err := GetQuests(engine, gameCatalog, sessionID, getCookbooksSlot, "quest", "")
	if err != nil {
		t.Fatalf("GetQuests: %v", err)
	}
	if !result.Active || result.SaveSessionID != sessionID ||
		result.CharacterID != getCookbooksSlot {
		t.Fatalf("result identity = %+v, want the active requested slot", result)
	}
	if len(result.Quests) != getQuestsCount {
		t.Fatalf("quest count = %d, want %d", len(result.Quests), getQuestsCount)
	}

	for index, quest := range result.Quests {
		if quest.Kind != schema.ResourceKindQuest || quest.Name == "" || len(quest.Steps) == 0 {
			t.Fatalf("quest %q = %+v, want a kind, a name and steps", quest.Key, quest)
		}
		if index > 0 {
			previous := result.Quests[index-1]
			if previous.Name > quest.Name ||
				(previous.Name == quest.Name && previous.Key > quest.Key) {
				t.Fatalf("quests are not ordered at %q then %q", previous.Key, quest.Key)
			}
		}
		for _, step := range quest.Steps {
			if step.StepKind != questStepKind || step.StepKey == "" || step.Description == "" {
				t.Fatalf("step %q of %q = %+v, want a kind, a key and a description",
					step.StepKey, quest.Key, step)
			}
		}
	}

	corhyn := matchedStepKeys(t, result, getQuestsKey)
	want := []string{getQuestsMatchedFirst, getQuestsMatchedSecond}
	if !slices.Equal(corhyn, want) {
		t.Errorf("matched %q steps = %v, want %v", getQuestsKey, corhyn, want)
	}
	// Several plans of one questline may hold at the same time. The result must
	// carry all of them instead of picking one as the current step.
	if multi := matchedStepKeys(t, result, getQuestsMultiMatchKey); len(multi) != getQuestsMultiMatchCount {
		t.Errorf("matched %q steps = %v, want %d of them",
			getQuestsMultiMatchKey, multi, getQuestsMultiMatchCount)
	}

	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Fatalf("session after the getter = %+v, want clean", info)
	}

	// The applied plan is the confirmed semantics: after SetQuestStep wrote one
	// step's canonical flags, exactly that step matches. Its expectedRevision "0"
	// also proves the getter left the revision untouched.
	if _, err := SetQuestStep(engine, gameCatalog, sessionID, getCookbooksSlot,
		"quest", getQuestsKey, questStepKind, getQuestsAppliedStep, "0"); err != nil {
		t.Fatalf("SetQuestStep: %v", err)
	}
	applied, err := GetQuests(engine, gameCatalog, sessionID, getCookbooksSlot, "quest", getQuestsKey)
	if err != nil {
		t.Fatalf("GetQuests after SetQuestStep: %v", err)
	}
	if len(applied.Quests) != 1 || applied.Quests[0].Key != getQuestsKey ||
		len(applied.Quests[0].Steps) != getQuestsStepCount {
		t.Fatalf("filtered result = %+v, want only %q with %d steps",
			applied.Quests, getQuestsKey, getQuestsStepCount)
	}
	matched := matchedStepKeys(t, applied, getQuestsKey)
	if !slices.Contains(matched, getQuestsAppliedStep) {
		t.Errorf("matched steps after applying %q = %v, want it among them",
			getQuestsAppliedStep, matched)
	}
}

func TestGetQuestsDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadBossesSession(t, false)
	result, err := GetQuests(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot, "quest", "")
	if err != nil {
		t.Fatalf("GetQuests: %v", err)
	}
	if result.Active || len(result.Quests) != getQuestsCount {
		t.Fatalf("active/count = %t/%d, want false/%d",
			result.Active, len(result.Quests), getQuestsCount)
	}
	// The cleared bitfield of the deleted character would match the plans above,
	// so an entirely unmatched result proves the slot data was never decoded.
	for _, quest := range result.Quests {
		for _, step := range quest.Steps {
			if step.Matched {
				t.Fatalf("residual slot reports %q of %q matched", step.StepKey, quest.Key)
			}
		}
	}
}

func TestGetQuestsRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadBossesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := GetQuests(nil, gameCatalog, sessionID, getCookbooksSlot, "quest", ""); err == nil ||
		err.Error() != "save engine is not available" {
		t.Errorf("nil SaveEngine error = %v", err)
	}
	if _, err := GetQuests(engine, nil, sessionID, getCookbooksSlot, "quest", ""); err == nil ||
		err.Error() != "game catalog is not available" {
		t.Errorf("nil GameCatalog error = %v", err)
	}

	cases := map[string]struct {
		questKind string
		questKey  string
		session   string
		want      string
	}{
		"wrong_quest_kind": {
			questKind: "item",
			session:   sessionID,
			want:      `resource kind "item" is not "quest"`,
		},
		"empty_quest_kind": {
			session: sessionID,
			want:    `resource kind "" is not "quest"`,
		},
		"unknown_quest_key": {
			questKind: "quest",
			questKey:  "unknown_npc",
			session:   sessionID,
			want:      `unknown resource key "unknown_npc" in kind "quest"`,
		},
		"unknown_session": {
			questKind: "quest",
			session:   "missing",
			want:      `unknown save session "missing"`,
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := GetQuests(engine, gameCatalog, testCase.session, getCookbooksSlot,
				testCase.questKind, testCase.questKey)
			if err == nil {
				t.Fatalf("GetQuests accepted invalid request %q", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
		})
	}
}
