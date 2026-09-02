package world

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSetSummoningPoolActivatedActivatesAndDeactivates(t *testing.T) {
	engine, sessionID := loadSummoningPoolsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	// The fixture sets getSummoningPoolsSetFlag (670130) and leaves its
	// neighbour getSummoningPoolsClearFlag (670131) clear. Both share one byte
	// of block 670, so activating one and deactivating the other proves the
	// mutation addresses a single bit.
	result, err := SetSummoningPoolActivated(engine, gameCatalog, sessionID,
		getCookbooksSlot, "summoning_pool", getSummoningPoolsClearKey, true, "0")
	if err != nil {
		t.Fatalf("SetSummoningPoolActivated: %v", err)
	}
	want := SetSummoningPoolActivatedResult{
		MutationReceipt: wantWorldReceipt(
			t, result.MutationReceipt, SetSummoningPoolActivatedEndpointID, sessionID, "1"),
		CharacterID:       getCookbooksSlot,
		SummoningPoolKind: schema.ResourceKindSummoningPool,
		SummoningPoolKey:  getSummoningPoolsClearKey,
		Activated:         true,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	if _, err := SetSummoningPoolActivated(engine, gameCatalog, sessionID,
		getCookbooksSlot, "summoning_pool", getSummoningPoolsSetKey, false, "1"); err != nil {
		t.Fatalf("SetSummoningPoolActivated deactivate: %v", err)
	}

	state, err := GetSummoningPools(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSummoningPools: %v", err)
	}
	activated := make(map[string]bool, len(state.SummoningPools))
	for _, entry := range state.SummoningPools {
		activated[entry.Key] = entry.Activated
	}
	if !activated[getSummoningPoolsClearKey] {
		t.Errorf("pool %q is deactivated, want activated", getSummoningPoolsClearKey)
	}
	if activated[getSummoningPoolsSetKey] {
		t.Errorf("pool %q is activated, want deactivated", getSummoningPoolsSetKey)
	}
}

func TestSetSummoningPoolActivatedRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadSummoningPoolsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := SetSummoningPoolActivated(nil, gameCatalog, sessionID, getCookbooksSlot,
		"summoning_pool", getSummoningPoolsSetKey, true, "0"); err == nil ||
		err.Error() != "save engine is not available" {
		t.Errorf("nil SaveEngine error = %v, want \"save engine is not available\"", err)
	}
	if _, err := SetSummoningPoolActivated(engine, nil, sessionID, getCookbooksSlot,
		"summoning_pool", getSummoningPoolsSetKey, true, "0"); err == nil ||
		err.Error() != "game catalog is not available" {
		t.Errorf("nil GameCatalog error = %v, want \"game catalog is not available\"", err)
	}

	for name, testCase := range map[string]struct {
		kind string
		key  string
		want string
	}{
		// A valid pool key under another kind must fail, which proves the kind is
		// checked before the key is looked up.
		"wrong kind": {
			"item", getSummoningPoolsSetKey,
			`resource kind "item" is not "summoning_pool"`,
		},
		"unknown key": {
			"summoning_pool", "not_a_pool",
			`unknown resource key "not_a_pool" in kind "summoning_pool"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := SetSummoningPoolActivated(engine, gameCatalog, sessionID,
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
