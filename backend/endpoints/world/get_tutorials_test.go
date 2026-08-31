package world

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	getTutorialsCount       = 72
	getTutorialsUnlockedID  = 2010
	getTutorialsUnlockedKey = "2010"
	getTutorialsLockedKey   = "2020"
)

func loadTutorialsSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()

	path := writeGetCookbooksFixture(t, nil, active)
	if active {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		anchorBase := int64(getCookbooksHeaderSize) + 0x10 +
			getCookbooksSlot*getCookbooksSlotBlockSize + getCookbooksAnchorAt
		tutorialAt := anchorBase + getCookbooksSectionAt - getCookbooksScalarsSize -
			getCookbooksTutorialSize - getCookbooksDynamicHeader
		binary.LittleEndian.PutUint32(data[tutorialAt+8:], 1)
		binary.LittleEndian.PutUint32(data[tutorialAt+12:], getTutorialsUnlockedID)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func TestGetTutorialsReturnsCatalogStateAndFilters(t *testing.T) {
	engine, sessionID := loadTutorialsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	all, err := GetTutorials(engine, gameCatalog, sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetTutorials: %v", err)
	}
	if !all.Active || len(all.Tutorials) != getTutorialsCount {
		t.Fatalf("active/count = %t/%d, want true/%d",
			all.Active, len(all.Tutorials), getTutorialsCount)
	}
	found := map[string]bool{}
	for index, entry := range all.Tutorials {
		if entry.Kind != schema.ResourceKindTutorial || entry.Title == "" {
			t.Fatalf("tutorial entry = %+v", entry)
		}
		if index > 0 && all.Tutorials[index-1].Title > entry.Title {
			t.Fatalf("tutorials are not ordered by title at %q", entry.Title)
		}
		if entry.Key == getTutorialsUnlockedKey || entry.Key == getTutorialsLockedKey {
			found[entry.Key] = entry.Unlocked
		}
	}
	if !found[getTutorialsUnlockedKey] || found[getTutorialsLockedKey] {
		t.Fatalf("selected states = %v", found)
	}

	unlocked, err := GetTutorials(
		engine, gameCatalog, sessionID, getCookbooksSlot, TutorialAvailabilityUnlocked)
	if err != nil || len(unlocked.Tutorials) != 1 ||
		unlocked.Tutorials[0].Key != getTutorialsUnlockedKey {
		t.Fatalf("unlocked filter = %+v, error %v", unlocked, err)
	}
	locked, err := GetTutorials(
		engine, gameCatalog, sessionID, getCookbooksSlot, TutorialAvailabilityLocked)
	if err != nil || len(locked.Tutorials) != getTutorialsCount-1 {
		t.Fatalf("locked filter count = %d, error %v", len(locked.Tutorials), err)
	}
}

func TestGetTutorialsRejectsInvalidDependenciesAndFilter(t *testing.T) {
	engine, sessionID := loadTutorialsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)
	if _, err := GetTutorials(
		nil, gameCatalog, sessionID, getCookbooksSlot, ""); err == nil {
		t.Fatal("nil SaveEngine was accepted")
	}
	if _, err := GetTutorials(
		engine, nil, sessionID, getCookbooksSlot, ""); err == nil {
		t.Fatal("nil GameCatalog was accepted")
	}
	if _, err := GetTutorials(
		engine, gameCatalog, sessionID, getCookbooksSlot, "Unlocked"); err == nil || !strings.Contains(err.Error(), "availabilityFilter") {
		t.Fatalf("invalid filter error = %v", err)
	}
}

func TestGetTutorialsFiltersInactiveSlotWithoutReadingResidualData(t *testing.T) {
	engine, sessionID := loadTutorialsSession(t, false)
	result, err := GetTutorials(
		engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot,
		TutorialAvailabilityUnlocked)
	if err != nil {
		t.Fatalf("GetTutorials: %v", err)
	}
	if result.Active || result.Tutorials == nil || len(result.Tutorials) != 0 {
		t.Fatalf("inactive unlocked result = %+v", result)
	}
}
