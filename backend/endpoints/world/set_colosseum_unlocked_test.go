package world

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSetColosseumUnlockedSetsAndClearsTheState(t *testing.T) {
	engine, sessionID := loadColosseumsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	// The fixture leaves limgrave_colosseum locked and both other arenas
	// unlocked, so unlocking the one and locking another proves the endpoint
	// resolves the exact requested key.
	result, err := SetColosseumUnlocked(engine, gameCatalog, sessionID,
		getCookbooksSlot, "colosseum", "limgrave_colosseum", true, "0")
	if err != nil {
		t.Fatalf("SetColosseumUnlocked: %v", err)
	}
	want := SetColosseumUnlockedResult{
		SaveSessionID: sessionID,
		SaveRevision:  "1",
		CharacterID:   getCookbooksSlot,
		ColosseumKind: schema.ResourceKindColosseum,
		ColosseumKey:  "limgrave_colosseum",
		Unlocked:      true,
	}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	if _, err := SetColosseumUnlocked(engine, gameCatalog, sessionID,
		getCookbooksSlot, "colosseum", "caelid_colosseum", false, "1"); err != nil {
		t.Fatalf("SetColosseumUnlocked clear: %v", err)
	}

	state, err := GetColosseums(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetColosseums: %v", err)
	}
	unlocked := make(map[string]bool, len(state.Colosseums))
	for _, entry := range state.Colosseums {
		unlocked[entry.Key] = entry.Unlocked
	}
	for key, expected := range map[string]bool{
		"limgrave_colosseum": true,
		"caelid_colosseum":   false,
		"royal_colosseum":    true,
	} {
		if unlocked[key] != expected {
			t.Errorf("colosseum %q unlocked = %t, want %t", key, unlocked[key], expected)
		}
	}
}

func TestSetColosseumUnlockedRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadColosseumsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := SetColosseumUnlocked(nil, gameCatalog, sessionID, getCookbooksSlot,
		"colosseum", "royal_colosseum", true, "0"); err == nil ||
		err.Error() != "save engine is not available" {
		t.Errorf("nil SaveEngine error = %v, want \"save engine is not available\"", err)
	}
	if _, err := SetColosseumUnlocked(engine, nil, sessionID, getCookbooksSlot,
		"colosseum", "royal_colosseum", true, "0"); err == nil ||
		err.Error() != "game catalog is not available" {
		t.Errorf("nil GameCatalog error = %v, want \"game catalog is not available\"", err)
	}

	for name, testCase := range map[string]struct {
		kind string
		key  string
		want string
	}{
		// A valid colosseum key under another kind must fail, which proves the kind
		// is checked before the key is looked up.
		"wrong kind": {
			"item", "royal_colosseum",
			`resource kind "item" is not "colosseum"`,
		},
		"unknown key": {
			"colosseum", "not_a_colosseum",
			`unknown resource key "not_a_colosseum" in kind "colosseum"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := SetColosseumUnlocked(engine, gameCatalog, sessionID,
				getCookbooksSlot, testCase.kind, testCase.key, true, "0")
			if err == nil {
				t.Fatalf("accepted %s", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
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
