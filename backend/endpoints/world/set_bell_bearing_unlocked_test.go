package world

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSetBellBearingUnlockedCommitsCatalogResource(t *testing.T) {
	engine, sessionID := loadBellBearingsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	const key = "400022CF" // flag 11109711 is clear in the fixture.

	result, err := SetBellBearingUnlocked(
		engine, gameCatalog, sessionID, getCookbooksSlot,
		"item", key, true, "0")
	if err != nil {
		t.Fatalf("SetBellBearingUnlocked: %v", err)
	}
	want := SetBellBearingUnlockedResult{
		SaveSessionID: sessionID, SaveRevision: "1", CharacterID: getCookbooksSlot,
		BellBearingKind: schema.ResourceKindItem, BellBearingKey: key, Unlocked: true,
	}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	got, err := GetBellBearings(
		engine, gameCatalog, sessionID, getCookbooksSlot, BellBearingAvailabilityUnlocked)
	if err != nil {
		t.Fatalf("GetBellBearings: %v", err)
	}
	found := false
	for _, entry := range got.BellBearings {
		if entry.Key == key {
			found = entry.Unlocked
		}
	}
	if !found {
		t.Errorf("GetBellBearings does not report %s unlocked", key)
	}
}

func TestSetBellBearingUnlockedRejectsMissingDependencies(t *testing.T) {
	engine, sessionID := loadBellBearingsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	const key = "400022CE"

	if _, err := SetBellBearingUnlocked(
		nil, gameCatalog, sessionID, getCookbooksSlot,
		"item", key, true, "0"); err == nil {
		t.Fatal("nil SaveEngine was accepted")
	}
	if _, err := SetBellBearingUnlocked(
		engine, nil, sessionID, getCookbooksSlot,
		"item", key, true, "0"); err == nil {
		t.Fatal("nil GameCatalog was accepted")
	}
	if _, err := SetBellBearingUnlocked(
		engine, gameCatalog, sessionID, getCookbooksSlot,
		"item", key, true, "1"); err == nil ||
		!strings.Contains(err.Error(), "does not match the current saveRevision") {
		t.Fatalf("stale revision error = %v", err)
	}
	if _, err := SetBellBearingUnlocked(
		engine, gameCatalog, sessionID, getCookbooksSlot,
		"item", "40002454", true, "0"); err == nil ||
		!strings.Contains(err.Error(), "declares no bell_bearing unlock") {
		t.Fatalf("non-Bell-Bearing error = %v", err)
	}

	resources := patchCookbookDocument(t, storedCookbookResources(t), "400022CF",
		func(document *schema.ItemDocument) {
			for index := range document.Unlocks {
				if document.Unlocks[index].Kind.Known &&
					document.Unlocks[index].Kind.Value == bellBearingUnlockKind {
					document.Unlocks[index].EventFlagID.Value = 11109710
				}
			}
		})
	if _, err := SetBellBearingUnlocked(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot,
		"item", key, true, "0"); err == nil ||
		!strings.Contains(err.Error(), "both declare event flag 11109710") {
		t.Fatalf("duplicate flag error = %v", err)
	}
}
