package world

import (
	"reflect"
	"strings"
	"testing"
)

func TestSetFogOfWarRemovedCommitsTheGlobalField(t *testing.T) {
	engine, sessionID := loadRegionsSession(t, true)

	result, err := SetFogOfWarRemoved(engine, sessionID, getCookbooksSlot, true, "0")
	if err != nil {
		t.Fatalf("SetFogOfWarRemoved: %v", err)
	}
	want := SetFogOfWarRemovedResult{
		MutationReceipt: wantWorldReceipt(
			t, result.MutationReceipt, SetFogOfWarRemovedEndpointID, sessionID, "1"),
		CharacterID: getCookbooksSlot,
		Removed:     true,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	// The field is global and cosmetic, so the unlocked-region list it is
	// measured from must come back unchanged.
	regions, err := GetRegions(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetRegions: %v", err)
	}
	unlocked := 0
	for _, entry := range regions.Regions {
		if entry.Unlocked {
			unlocked++
		}
	}
	if unlocked != 2 {
		t.Errorf("unlocked regions = %d, want the 2 the fixture declares", unlocked)
	}
}

// removed=false has no confirmed contract, so it must be refused before the
// session is touched instead of being read as "fill with zeros".
func TestSetFogOfWarRemovedRejectsRestoreAndMissingEngine(t *testing.T) {
	engine, sessionID := loadRegionsSession(t, true)

	if _, err := SetFogOfWarRemoved(nil, sessionID, getCookbooksSlot, true, "0"); err == nil ||
		err.Error() != "save engine is not available" {
		t.Fatalf("missing engine error = %v, want the unavailable engine rejection", err)
	}

	_, err := SetFogOfWarRemoved(engine, sessionID, getCookbooksSlot, false, "0")
	if err == nil || !strings.Contains(err.Error(), "removed must be true") {
		t.Fatalf("error = %v, want the removed=false rejection", err)
	}
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("rejected request dirtied the session: %+v", info)
	}
}
