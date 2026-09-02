package world

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSetTutorialUnlockedCommitsBothDirections(t *testing.T) {
	engine, sessionID := loadTutorialsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	unlock, err := SetTutorialUnlocked(engine, gameCatalog, sessionID, getCookbooksSlot,
		string(schema.ResourceKindTutorial), getTutorialsLockedKey, true, "0")
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
	want := SetTutorialUnlockedResult{
		MutationReceipt: wantWorldReceipt(
			t, unlock.MutationReceipt, SetTutorialUnlockedEndpointID, sessionID, "1"),
		CharacterID:  getCookbooksSlot,
		TutorialKind: schema.ResourceKindTutorial,
		TutorialKey:  getTutorialsLockedKey,
		Unlocked:     true,
	}
	if !reflect.DeepEqual(unlock, want) {
		t.Fatalf("unlock result = %+v, want %+v", unlock, want)
	}

	if _, err := SetTutorialUnlocked(engine, gameCatalog, sessionID, getCookbooksSlot,
		string(schema.ResourceKindTutorial), getTutorialsUnlockedKey, false, "1"); err != nil {
		t.Fatalf("lock: %v", err)
	}

	state, err := GetTutorials(engine, gameCatalog, sessionID, getCookbooksSlot, TutorialAvailabilityUnlocked)
	if err != nil {
		t.Fatalf("GetTutorials: %v", err)
	}
	if len(state.Tutorials) != 1 || state.Tutorials[0].Key != getTutorialsLockedKey {
		t.Fatalf("unlocked tutorials = %+v", state.Tutorials)
	}
}

func TestSetTutorialUnlockedRejectsInvalidDependenciesAndResources(t *testing.T) {
	engine, sessionID := loadTutorialsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := SetTutorialUnlocked(nil, gameCatalog, sessionID, getCookbooksSlot,
		string(schema.ResourceKindTutorial), getTutorialsUnlockedKey, true, "0"); err == nil {
		t.Fatal("nil SaveEngine was accepted")
	}
	if _, err := SetTutorialUnlocked(engine, nil, sessionID, getCookbooksSlot,
		string(schema.ResourceKindTutorial), getTutorialsUnlockedKey, true, "0"); err == nil {
		t.Fatal("nil GameCatalog was accepted")
	}

	for name, testCase := range map[string]struct {
		kind string
		key  string
		want string
	}{
		"wrong kind": {kind: "item", key: getTutorialsUnlockedKey, want: "is not \"tutorial\""},
		"unknown key": {
			kind: string(schema.ResourceKindTutorial), key: "not_a_tutorial",
			want: "unknown resource key",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := SetTutorialUnlocked(engine, gameCatalog, sessionID, getCookbooksSlot,
				testCase.kind, testCase.key, true, "0")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want one containing %q", err, testCase.want)
			}
		})
	}
}
