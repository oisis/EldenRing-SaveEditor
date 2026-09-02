package world

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSetBossDefeatedSetsAndClearsTheFlag(t *testing.T) {
	engine, sessionID := loadBossesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	// The fixture sets getBossesSetFlag (9100) and leaves its neighbour
	// getBossesClearKey (9101) clear. Both share one byte of block 9, so
	// defeating one and reviving the other proves the mutation addresses a
	// single bit.
	result, err := SetBossDefeated(engine, gameCatalog, sessionID,
		getCookbooksSlot, "boss", getBossesClearKey, true, "0")
	if err != nil {
		t.Fatalf("SetBossDefeated: %v", err)
	}
	want := SetBossDefeatedResult{
		MutationReceipt: wantWorldReceipt(
			t, result.MutationReceipt, SetBossDefeatedEndpointID, sessionID, "1"),
		CharacterID: getCookbooksSlot,
		BossKind:    schema.ResourceKindBoss,
		BossKey:     getBossesClearKey,
		Defeated:    true,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	if _, err := SetBossDefeated(engine, gameCatalog, sessionID,
		getCookbooksSlot, "boss", getBossesSetKey, false, "1"); err != nil {
		t.Fatalf("SetBossDefeated clear: %v", err)
	}

	state, err := GetBosses(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetBosses: %v", err)
	}
	defeatedByKey := make(map[string]bool, len(state.Bosses))
	for _, entry := range state.Bosses {
		defeatedByKey[entry.Key] = entry.Defeated
	}
	if !defeatedByKey[getBossesClearKey] {
		t.Errorf("boss %q is undefeated, want defeated", getBossesClearKey)
	}
	if defeatedByKey[getBossesSetKey] {
		t.Errorf("boss %q is defeated, want undefeated", getBossesSetKey)
	}
}

func TestSetBossDefeatedRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadBossesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := SetBossDefeated(nil, gameCatalog, sessionID, getCookbooksSlot,
		"boss", getBossesSetKey, true, "0"); err == nil ||
		err.Error() != "save engine is not available" {
		t.Errorf("nil SaveEngine error = %v, want \"save engine is not available\"", err)
	}
	if _, err := SetBossDefeated(engine, nil, sessionID, getCookbooksSlot,
		"boss", getBossesSetKey, true, "0"); err == nil ||
		err.Error() != "game catalog is not available" {
		t.Errorf("nil GameCatalog error = %v, want \"game catalog is not available\"", err)
	}

	for name, testCase := range map[string]struct {
		kind string
		key  string
		want string
	}{
		// A valid boss key under another kind must fail, which proves the kind is
		// checked before the key is looked up.
		"wrong kind": {
			"item", getBossesSetKey,
			`resource kind "item" is not "boss"`,
		},
		"unknown key": {
			"boss", "not_a_boss",
			`unknown resource key "not_a_boss" in kind "boss"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := SetBossDefeated(engine, gameCatalog, sessionID,
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
