package world

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// The curated graces this test drives: the fixture leaves getGracesClearKey
// unvisited, tombsward is the catacomb grace that declares a door flag, and
// gatefront is the single grace with confirmed companion flags.
const (
	setGraceDungeonKey   = "weeping_peninsula_tombsward_catacombs"
	setGraceGatefrontKey = "limgrave_west_gatefront"
)

func TestSetGraceVisitedSetsAndClearsTheFlag(t *testing.T) {
	engine, sessionID := loadGracesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	// The fixture sets getGracesSetFlag (71000) and leaves its neighbour
	// getGracesClearKey (71001) clear. Both share one byte of block 71, so
	// visiting one and clearing the other proves the mutation addresses a
	// single bit.
	result, err := SetGraceVisited(engine, gameCatalog, sessionID,
		getCookbooksSlot, "grace", getGracesClearKey, true, "0")
	if err != nil {
		t.Fatalf("SetGraceVisited: %v", err)
	}
	want := SetGraceVisitedResult{
		MutationReceipt: wantWorldReceipt(
			t, result.MutationReceipt, SetGraceVisitedEndpointID, sessionID, "1"),
		CharacterID: getCookbooksSlot,
		GraceKind:   schema.ResourceKindGrace,
		GraceKey:    getGracesClearKey,
		Visited:     true,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	if _, err := SetGraceVisited(engine, gameCatalog, sessionID,
		getCookbooksSlot, "grace", getGracesSetKey, false, "1"); err != nil {
		t.Fatalf("SetGraceVisited clear: %v", err)
	}

	// A catacomb grace also writes its private door flag, and Gatefront also
	// writes its four confirmed companion flags. Neither is public, so both are
	// driven here and their bit-level effect is proven in the SaveEngine tests.
	if _, err := SetGraceVisited(engine, gameCatalog, sessionID,
		getCookbooksSlot, "grace", setGraceDungeonKey, true, "2"); err != nil {
		t.Fatalf("SetGraceVisited dungeon: %v", err)
	}
	if _, err := SetGraceVisited(engine, gameCatalog, sessionID,
		getCookbooksSlot, "grace", setGraceGatefrontKey, true, "3"); err != nil {
		t.Fatalf("SetGraceVisited gatefront: %v", err)
	}

	state, err := GetGraces(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetGraces: %v", err)
	}
	visited := make(map[string]bool, len(state.Graces))
	for _, entry := range state.Graces {
		visited[entry.Key] = entry.Visited
	}
	for key, expected := range map[string]bool{
		getGracesClearKey:    true,
		getGracesSetKey:      false,
		setGraceDungeonKey:   true,
		setGraceGatefrontKey: true,
	} {
		if visited[key] != expected {
			t.Errorf("grace %q visited = %t, want %t", key, visited[key], expected)
		}
	}

	// Clearing Gatefront must clear its own flag only; the companion flags stay
	// set, which the SaveEngine test proves at bit level.
	if _, err := SetGraceVisited(engine, gameCatalog, sessionID,
		getCookbooksSlot, "grace", setGraceGatefrontKey, false, "4"); err != nil {
		t.Fatalf("SetGraceVisited gatefront clear: %v", err)
	}
	state, err = GetGraces(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetGraces after clear: %v", err)
	}
	for _, entry := range state.Graces {
		if entry.Key == setGraceGatefrontKey && entry.Visited {
			t.Error("Gatefront is still visited after being cleared")
		}
	}
}

func TestSetGraceVisitedRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadGracesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := SetGraceVisited(nil, gameCatalog, sessionID, getCookbooksSlot,
		"grace", getGracesSetKey, true, "0"); err == nil ||
		err.Error() != "save engine is not available" {
		t.Errorf("nil SaveEngine error = %v, want \"save engine is not available\"", err)
	}
	if _, err := SetGraceVisited(engine, nil, sessionID, getCookbooksSlot,
		"grace", getGracesSetKey, true, "0"); err == nil ||
		err.Error() != "game catalog is not available" {
		t.Errorf("nil GameCatalog error = %v, want \"game catalog is not available\"", err)
	}

	for name, testCase := range map[string]struct {
		kind string
		key  string
		want string
	}{
		// A valid grace key under another kind must fail, which proves the kind is
		// checked before the key is looked up.
		"wrong kind": {
			"item", getGracesSetKey,
			`resource kind "item" is not "grace"`,
		},
		"unknown key": {
			"grace", "not_a_grace",
			`unknown resource key "not_a_grace" in kind "grace"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := SetGraceVisited(engine, gameCatalog, sessionID,
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
