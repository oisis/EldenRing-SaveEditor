package world

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSetWhetbladeUnlockedCommitsCatalogResource(t *testing.T) {
	engine, sessionID := loadWhetbladesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	const key = "4000230C"

	result, err := SetWhetbladeUnlocked(
		engine, gameCatalog, sessionID, getCookbooksSlot,
		"item", key, true, "0")
	if err != nil {
		t.Fatalf("SetWhetbladeUnlocked: %v", err)
	}
	want := SetWhetbladeUnlockedResult{
		MutationReceipt: wantWorldReceipt(t, result.MutationReceipt, SetWhetbladeUnlockedEndpointID, sessionID, "1"),
		CharacterID:     getCookbooksSlot,
		WhetbladeKind:   schema.ResourceKindItem, WhetbladeKey: key, Unlocked: true,
	}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	got, err := GetWhetblades(
		engine, gameCatalog, sessionID, getCookbooksSlot, WhetbladeAvailabilityUnlocked)
	if err != nil {
		t.Fatalf("GetWhetblades: %v", err)
	}
	found := false
	for _, entry := range got.Whetblades {
		if entry.Key == key {
			found = entry.Unlocked
		}
	}
	if !found {
		t.Errorf("GetWhetblades does not report %s unlocked", key)
	}
}

func TestSetWhetbladeUnlockedRejectsInvalidInput(t *testing.T) {
	engine, sessionID := loadWhetbladesSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	const key = "4000230C"

	if _, err := SetWhetbladeUnlocked(
		nil, gameCatalog, sessionID, getCookbooksSlot,
		"item", key, true, "0"); err == nil {
		t.Fatal("nil SaveEngine was accepted")
	}
	if _, err := SetWhetbladeUnlocked(
		engine, nil, sessionID, getCookbooksSlot,
		"item", key, true, "0"); err == nil {
		t.Fatal("nil GameCatalog was accepted")
	}
	if _, err := SetWhetbladeUnlocked(
		engine, gameCatalog, sessionID, getCookbooksSlot,
		"item", key, true, "1"); err == nil ||
		!strings.Contains(err.Error(), "does not match the current saveRevision") {
		t.Fatalf("stale revision error = %v", err)
	}
	if _, err := SetWhetbladeUnlocked(
		engine, gameCatalog, sessionID, getCookbooksSlot,
		"item", "40002454", true, "0"); err == nil ||
		!strings.Contains(err.Error(), "declares no whetblade unlock") {
		t.Fatalf("non-Whetblade error = %v", err)
	}

	resources := patchCookbookDocument(t, storedCookbookResources(t), key,
		func(document *schema.ItemDocument) {
			document.Links.RelatedEventFlags = nil
		})
	if _, err := SetWhetbladeUnlocked(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot,
		"item", key, true, "0"); err == nil ||
		!strings.Contains(err.Error(), "declares no whetblade_related event flag") {
		t.Fatalf("incomplete related flags error = %v", err)
	}
}
