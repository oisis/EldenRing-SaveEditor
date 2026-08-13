package world

import (
	"bytes"
	"os"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetCookbookUnlockedUnlocksALockedCookbook(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	// getCookbooksSecondKey ("40002455") is clear in fixture (flag 67010)
	result, err := SetCookbookUnlocked(
		engine, gameCatalog, sessionID, getCookbooksSlot, "item", getCookbooksSecondKey, true, "0")
	if err != nil {
		t.Fatalf("SetCookbookUnlocked: %v", err)
	}

	want := SetCookbookUnlockedResult{
		SaveSessionID: sessionID,
		SaveRevision:  "1",
		CharacterID:   getCookbooksSlot,
		CookbookKind:  schema.ResourceKindItem,
		CookbookKey:   getCookbooksSecondKey,
		Unlocked:      true,
	}
	if result != want {
		t.Errorf("result = %+v, want %+v", result, want)
	}

	// Verify through GetCookbooks getter
	getRes, err := GetCookbooks(engine, gameCatalog, sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetCookbooks: %v", err)
	}
	var entry *CookbookEntry
	for i := range getRes.Cookbooks {
		if getRes.Cookbooks[i].Key == getCookbooksSecondKey {
			entry = &getRes.Cookbooks[i]
			break
		}
	}
	if entry == nil {
		t.Fatalf("cookbook %q missing from GetCookbooks result", getCookbooksSecondKey)
	}
	if !entry.Unlocked {
		t.Errorf("cookbook %q is locked in GetCookbooks, want unlocked", getCookbooksSecondKey)
	}
}

func TestSetCookbookUnlockedClearsAnUnlockedCookbook(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	// getCookbooksFirstKey ("40002454") is set in fixture (flag 67000)
	result, err := SetCookbookUnlocked(
		engine, gameCatalog, sessionID, getCookbooksSlot, "item", getCookbooksFirstKey, false, "0")
	if err != nil {
		t.Fatalf("SetCookbookUnlocked: %v", err)
	}

	if result.SaveRevision != "1" || result.Unlocked != false {
		t.Errorf("result = %+v, want revision 1 and unlocked false", result)
	}

	getRes, err := GetCookbooks(engine, gameCatalog, sessionID, getCookbooksSlot, "")
	if err != nil {
		t.Fatalf("GetCookbooks: %v", err)
	}
	var entry *CookbookEntry
	for i := range getRes.Cookbooks {
		if getRes.Cookbooks[i].Key == getCookbooksFirstKey {
			entry = &getRes.Cookbooks[i]
			break
		}
	}
	if entry == nil {
		t.Fatalf("cookbook %q missing from GetCookbooks result", getCookbooksFirstKey)
	}
	if entry.Unlocked {
		t.Errorf("cookbook %q is unlocked in GetCookbooks, want locked", getCookbooksFirstKey)
	}
}

func TestSetCookbookUnlockedRejectsMissingBackends(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	cases := map[string]struct {
		engine      *saveengine.Engine
		gameCatalog *gamecatalog.Catalog
		want        string
	}{
		"nil engine":  {nil, gameCatalog, "save engine is not available"},
		"nil catalog": {engine, nil, "game catalog is not available"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := SetCookbookUnlocked(
				tc.engine, tc.gameCatalog, sessionID, getCookbooksSlot, "item", getCookbooksFirstKey, true, "0")
			if err == nil {
				t.Fatalf("accepted %s", name)
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err, tc.want)
			}
		})
	}
}

func TestSetCookbookUnlockedRejectsInactiveSlot(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, false) // Inactive slot
	gameCatalog := newCookbooksCatalog(t)

	_, err := SetCookbookUnlocked(
		engine, gameCatalog, sessionID, getCookbooksSlot, "item", getCookbooksFirstKey, true, "0")
	if err == nil {
		t.Fatal("SetCookbookUnlocked accepted an inactive slot")
	}
	want := "character 3 is not active"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestSetCookbookUnlockedRejectsInvalidSessionCharacterAndRevision(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		expectedRev   string
		want          string
	}{
		"empty session": {
			"", getCookbooksSlot, "0",
			"saveSessionID is required",
		},
		"unknown session": {
			"unknown-session", getCookbooksSlot, "0",
			`unknown save session "unknown-session"`,
		},
		"characterID -1": {
			sessionID, -1, "0",
			"characterID -1 is outside the range 0..9",
		},
		"characterID 10": {
			sessionID, 10, "0",
			"characterID 10 is outside the range 0..9",
		},
		"non-canonical revision": {
			sessionID, getCookbooksSlot, "01",
			`expectedRevision must be a canonical decimal saveRevision; got "01"`,
		},
		"mismatched revision": {
			sessionID, getCookbooksSlot, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := SetCookbookUnlocked(
				engine, gameCatalog, tc.saveSessionID, tc.characterID, "item", getCookbooksFirstKey, true, tc.expectedRev)
			if err == nil {
				t.Fatalf("accepted %s", name)
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err, tc.want)
			}
		})
	}
}

func TestSetCookbookUnlockedRejectsInvalidCatalogResources(t *testing.T) {
	engine, sessionID := loadCookbooksSession(t, true)

	t.Run("unknown resource", func(t *testing.T) {
		_, err := SetCookbookUnlocked(
			engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot, "item", "UNKNOWN", true, "0")
		if err == nil {
			t.Fatal("accepted unknown resource")
		}
		want := `unknown resource key "UNKNOWN" in kind "item"`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err, want)
		}
	})

	t.Run("non-item resource kind", func(t *testing.T) {
		_, err := SetCookbookUnlocked(
			engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot,
			"gesture", getCookbooksFirstKey, true, "0")
		if err == nil {
			t.Fatal("accepted a non-item resource kind")
		}
		want := `unknown resource kind "gesture"`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err, want)
		}
	})

	t.Run("resource without cookbook unlock", func(t *testing.T) {
		resources := patchCookbookDocument(t, storedCookbookResources(t), getCookbooksFirstKey,
			func(document *schema.ItemDocument) { document.Unlocks = nil })
		_, err := SetCookbookUnlocked(
			engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot,
			"item", getCookbooksFirstKey, true, "0")
		if err == nil {
			t.Fatal("accepted a resource without a cookbook unlock")
		}
		want := `resource kind "item" key "40002454" declares no cookbook unlock`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err, want)
		}
	})

	for name, testCase := range map[string]struct {
		change func(*schema.ItemUnlock)
		want   string
	}{
		"missing name": {
			func(unlock *schema.ItemUnlock) { unlock.Name = schema.Fact[string]{} },
			`cookbook "40002454" unlock 0 has no known name`,
		},
		"missing category": {
			func(unlock *schema.ItemUnlock) { unlock.Category = schema.Fact[string]{} },
			`cookbook "40002454" unlock 0 has no known category`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			resources := patchCookbookUnlock(
				t, storedCookbookResources(t), getCookbooksFirstKey, testCase.change)
			_, err := SetCookbookUnlocked(
				engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot,
				"item", getCookbooksFirstKey, true, "0")
			if err == nil {
				t.Fatalf("accepted %s", name)
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
		})
	}

	t.Run("unknown event flag", func(t *testing.T) {
		resources := patchCookbookUnlock(t, storedCookbookResources(t), getCookbooksFirstKey,
			func(unlock *schema.ItemUnlock) { unlock.EventFlagID.Known = false })
		for _, resource := range resources {
			if resource.Key != getCookbooksFirstKey {
				continue
			}
			_, _, err := declaredCookbookFromResource(resource)
			if err == nil {
				t.Fatal("shared cookbook resolver accepted an unknown event flag")
			}
			want := `cookbook "40002454" unlock 0 has no known event flag ID`
			if err.Error() != want {
				t.Errorf("error = %q, want %q", err, want)
			}
			return
		}
		t.Fatalf("resource %q not found", getCookbooksFirstKey)
	})

	t.Run("non-goods family", func(t *testing.T) {
		resources := storedCookbookResources(t)
		unlock := storedCookbookUnlock(t, resources, getCookbooksFirstKey)
		unlock.EventFlagID.Value = 67999
		resources = patchCookbookDocument(t, resources, getCookbooksWeaponKey,
			func(document *schema.ItemDocument) {
				document.Unlocks = append(document.Unlocks, unlock)
			})
		gameCatalog := cookbooksCatalogOf(t, resources)

		_, err := SetCookbookUnlocked(
			engine, gameCatalog, sessionID, getCookbooksSlot, "item", getCookbooksWeaponKey, true, "0")
		if err == nil {
			t.Fatal("accepted weapon resource as cookbook")
		}
		want := `cookbook "0001ADB0" has item family "weapon", want "goods"`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err, want)
		}
	})

	t.Run("resource with two cookbook unlocks", func(t *testing.T) {
		resources := storedCookbookResources(t)
		second := storedCookbookUnlock(t, resources, getCookbooksFirstKey)
		second.EventFlagID.Value = 67999
		second.Name.Value = "Second Cookbook"
		second.Category.Value = "Second Series"
		resources = patchCookbookDocument(t, resources, getCookbooksFirstKey,
			func(document *schema.ItemDocument) {
				document.Unlocks = append(document.Unlocks, second)
			})
		gameCatalog := cookbooksCatalogOf(t, resources)

		_, err := SetCookbookUnlocked(
			engine, gameCatalog, sessionID, getCookbooksSlot, "item", getCookbooksFirstKey, true, "0")
		if err == nil {
			t.Fatal("accepted resource with 2 cookbook unlocks")
		}
		want := `cookbook "40002454" declares 2 cookbook unlocks, want exactly one`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err, want)
		}
	})

	t.Run("duplicate event flag in catalog", func(t *testing.T) {
		resources := patchCookbookUnlock(t, storedCookbookResources(t), getCookbooksSecondKey,
			func(unlock *schema.ItemUnlock) { unlock.EventFlagID.Value = 67000 })
		gameCatalog := cookbooksCatalogOf(t, resources)

		_, err := SetCookbookUnlocked(
			engine, gameCatalog, sessionID, getCookbooksSlot, "item", getCookbooksFirstKey, true, "0")
		if err == nil {
			t.Fatal("accepted catalog with duplicate event flag")
		}
		want := `cookbooks "40002454" and "40002455" both declare event flag 67000`
		if err.Error() != want {
			t.Errorf("error = %q, want %q", err, want)
		}
	})

	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after catalog rejections = %+v, want clean", info)
	}
	result, err := SetCookbookUnlocked(
		engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot,
		"item", getCookbooksSecondKey, true, "0")
	if err != nil {
		t.Fatalf("valid mutation after catalog rejections: %v", err)
	}
	if result.SaveRevision != "1" {
		t.Errorf("revision after catalog rejections = %q, want first commit revision 1", result.SaveRevision)
	}
}

func TestSetCookbookUnlockedLeavesTheSaveFileUntouched(t *testing.T) {
	path := writeGetCookbooksFixture(t, getCookbooksSetFlags, true)

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	gameCatalog := newCookbooksCatalog(t)
	_, err = SetCookbookUnlocked(
		engine, gameCatalog, loaded.SaveSessionID, getCookbooksSlot, "item", getCookbooksSecondKey, true, "0")
	if err != nil {
		t.Fatalf("SetCookbookUnlocked: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reread fixture: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Error("the save file on disk changed, want it untouched until WriteSave")
	}
}
