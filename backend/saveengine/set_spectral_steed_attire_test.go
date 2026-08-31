package saveengine

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	spectralSteedDefaultFlag  = uint32(6700)
	spectralSteedTreeFlag     = uint32(6701)
	spectralSteedSilverFlag   = uint32(6702)
	spectralSteedFuneralFlag  = uint32(6703)
	spectralSteedTreeGameID   = uint32(0x401EAA00)
	spectralSteedSilverGameID = uint32(0x401EAA0A)
	spectralSteedFuneralID    = uint32(0x401EAA14)
	spectralSteedUnrelatedID  = uint32(0x400006A4)
)

var spectralSteedTestAttires = []SpectralSteedAttire{
	{EventFlagID: spectralSteedDefaultFlag},
	{EventFlagID: spectralSteedTreeFlag, GameID: spectralSteedTreeGameID},
	{EventFlagID: spectralSteedSilverFlag, GameID: spectralSteedSilverGameID},
	{EventFlagID: spectralSteedFuneralFlag, GameID: spectralSteedFuneralID},
}

// spectralSteedFixture builds a synthetic save whose slot carries both a
// complete event-flag chain and a usable InventoryHeld section, so one fixture
// serves the flag half and the item half of these mutations. set names the
// appearance flags that are already on.
func spectralSteedFixture(t *testing.T, platform Platform, set []uint32) (string, eventFlagTestFixture) {
	t.Helper()

	content := eventFlagTestContent(platform)
	content.anchorAt = setWhetbladeFullAnchorAt
	content.set = set
	path := writeEventFlagFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	slotBase := eventFlagTestPCSlotDataBase + int64(content.slot)*eventFlagTestPCSlotStride
	if platform == PlatformPS4 {
		slotBase = eventFlagTestPS4SlotDataBase + int64(content.slot)*eventFlagTestPS4SlotStride
	}
	binary.LittleEndian.PutUint32(data[slotBase:], 0x6E)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path, content
}

func loadSpectralSteedSession(
	t *testing.T, platform Platform, set []uint32,
) (*Engine, string, int) {
	t.Helper()

	path, content := spectralSteedFixture(t, platform, set)
	engine := New()
	loaded, err := engine.LoadSave(path, string(platform), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID, content.slot
}

// assertSpectralSteedFlags states the complete four-flag picture, so a mutation
// that sets the right flag but leaves an old one behind cannot pass.
func assertSpectralSteedFlags(
	t *testing.T, engine *Engine, saveSessionID string, characterID int, want map[uint32]bool,
) {
	t.Helper()

	ids := []uint32{
		spectralSteedDefaultFlag, spectralSteedTreeFlag,
		spectralSteedSilverFlag, spectralSteedFuneralFlag,
	}
	flags, err := engine.GetEventFlags(saveSessionID, characterID, ids)
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	for _, id := range ids {
		if flags.Flags[id] != want[id] {
			t.Errorf("event flag %d = %t, want %t", id, flags.Flags[id], want[id])
		}
	}
}

func spectralSteedOnly(active uint32) map[uint32]bool {
	return map[uint32]bool{
		spectralSteedDefaultFlag: active == spectralSteedDefaultFlag,
		spectralSteedTreeFlag:    active == spectralSteedTreeFlag,
		spectralSteedSilverFlag:  active == spectralSteedSilverFlag,
		spectralSteedFuneralFlag: active == spectralSteedFuneralFlag,
	}
}

func addSpectralSteedItem(
	t *testing.T, engine *Engine, saveSessionID string, characterID int,
	gameID uint32, expectedRevision string,
) string {
	t.Helper()

	result, err := engine.AddItemToInventory(
		saveSessionID, characterID, gameID, 1, expectedRevision, false, 1, 1)
	if err != nil {
		t.Fatalf("AddItemToInventory(0x%08X): %v", gameID, err)
	}
	return result.SaveRevision
}

// assertSpectralSteedRevision pins the session revision a refused mutation must
// have left untouched.
func assertSpectralSteedRevision(
	t *testing.T, engine *Engine, saveSessionID string, characterID int, want string,
) {
	t.Helper()

	undo, err := engine.GetUndoState(saveSessionID, characterID)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undo.SaveRevision != want {
		t.Errorf("saveRevision = %q, want %q", undo.SaveRevision, want)
	}
}

// spectralSteedCommonCount reads the declared common InventoryHeld count, which
// is the one field three removals from the same section share. A plan that wrote
// it once per removal instead of once per section would leave it too high here.
func spectralSteedCommonCount(
	t *testing.T, engine *Engine, saveSessionID string, characterID int,
) uint32 {
	t.Helper()

	loaded := engine.sessions[saveSessionID]
	sectionAt, err := inventoryHeldSectionAt(loaded, characterID)
	if err != nil {
		t.Fatalf("inventoryHeldSectionAt: %v", err)
	}
	count, err := loaded.snapshot.uint32At(sectionAt - 4)
	if err != nil {
		t.Fatalf("read common count: %v", err)
	}
	return count
}

// assertSpectralSteedClean proves a refused mutation did not mark the session
// dirty, so nothing it planned reached the private snapshot.
func assertSpectralSteedClean(t *testing.T, engine *Engine, saveSessionID string) {
	t.Helper()

	info, err := engine.GetSessionInfo(saveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session reports unsaved changes after a refused mutation: %+v", info)
	}
}

func spectralSteedItemPresence(
	t *testing.T, engine *Engine, saveSessionID string, characterID int,
) map[uint32]bool {
	t.Helper()

	present, err := engine.GetInventoryGoodsPresence(saveSessionID, characterID, []uint32{
		spectralSteedTreeGameID, spectralSteedSilverGameID,
		spectralSteedFuneralID, spectralSteedUnrelatedID,
	})
	if err != nil {
		t.Fatalf("GetInventoryGoodsPresence: %v", err)
	}
	return present
}

func TestSetSpectralSteedAttirePersistsOnBothPlatforms(t *testing.T) {
	// Both containers carry identical slot content, so a mutation that mixes the
	// two platform bases cannot pass both cases.
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, sessionID, slot := loadSpectralSteedSession(t, platform, nil)
			revision := addSpectralSteedItem(
				t, engine, sessionID, slot, spectralSteedSilverGameID, "0")

			result, err := engine.SetSpectralSteedAttire(
				sessionID, slot, spectralSteedTestAttires, spectralSteedSilverFlag, revision)
			if err != nil {
				t.Fatalf("SetSpectralSteedAttire: %v", err)
			}
			if result.SaveRevision != "2" || result.CharacterID != slot {
				t.Errorf("result = %+v, want revision 2 for character %d", result, slot)
			}
			assertSpectralSteedFlags(
				t, engine, sessionID, slot, spectralSteedOnly(spectralSteedSilverFlag))

			target := filepath.Join(t.TempDir(), "written.sl2")
			if _, err := engine.WriteSave(sessionID, "2", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			assertSpectralSteedFlags(t, reloadedEngine, reloaded.SaveSessionID, slot,
				spectralSteedOnly(spectralSteedSilverFlag))
			if !spectralSteedItemPresence(
				t, reloadedEngine, reloaded.SaveSessionID, slot)[spectralSteedSilverGameID] {
				t.Error("the reloaded save no longer holds the worn attire item")
			}
		})
	}
}

func TestSetSpectralSteedAttireSelectsEveryAppearance(t *testing.T) {
	engine, sessionID, slot := loadSpectralSteedSession(t, PlatformPC, nil)
	revision := "0"
	for _, gameID := range []uint32{
		spectralSteedTreeGameID, spectralSteedSilverGameID, spectralSteedFuneralID,
	} {
		revision = addSpectralSteedItem(t, engine, sessionID, slot, gameID, revision)
	}

	for _, flagID := range []uint32{
		spectralSteedTreeFlag, spectralSteedSilverFlag,
		spectralSteedFuneralFlag, spectralSteedDefaultFlag,
	} {
		result, err := engine.SetSpectralSteedAttire(
			sessionID, slot, spectralSteedTestAttires, flagID, revision)
		if err != nil {
			t.Fatalf("SetSpectralSteedAttire(%d): %v", flagID, err)
		}
		revision = result.SaveRevision
		assertSpectralSteedFlags(t, engine, sessionID, slot, spectralSteedOnly(flagID))
	}

	// Selecting the appearance that is already active moves no byte but still
	// commits one revision, exactly like every other mutation of this engine.
	result, err := engine.SetSpectralSteedAttire(
		sessionID, slot, spectralSteedTestAttires, spectralSteedDefaultFlag, revision)
	if err != nil {
		t.Fatalf("repeated SetSpectralSteedAttire: %v", err)
	}
	if result.SaveRevision == revision {
		t.Errorf("saveRevision stayed %q, want the next revision", result.SaveRevision)
	}
	assertSpectralSteedFlags(
		t, engine, sessionID, slot, spectralSteedOnly(spectralSteedDefaultFlag))
	undo, err := engine.GetUndoState(sessionID, slot)
	if err != nil {
		t.Fatalf("GetUndoState: %v", err)
	}
	if undo.Available {
		t.Error("a mutation that changed no byte recorded an undo point")
	}
}

func TestSetSpectralSteedAttireResolvesLegacyAndConflict(t *testing.T) {
	for name, set := range map[string][]uint32{
		"legacy":   nil,
		"conflict": {spectralSteedTreeFlag, spectralSteedFuneralFlag},
	} {
		t.Run(name, func(t *testing.T) {
			engine, sessionID, slot := loadSpectralSteedSession(t, PlatformPC, set)
			result, err := engine.SetSpectralSteedAttire(
				sessionID, slot, spectralSteedTestAttires, spectralSteedDefaultFlag, "0")
			if err != nil {
				t.Fatalf("SetSpectralSteedAttire: %v", err)
			}
			if result.SaveRevision != "1" {
				t.Errorf("saveRevision = %q, want 1", result.SaveRevision)
			}
			assertSpectralSteedFlags(
				t, engine, sessionID, slot, spectralSteedOnly(spectralSteedDefaultFlag))
		})
	}
}

func TestSetSpectralSteedAttireRejectsAnAppearanceWithoutItsItem(t *testing.T) {
	engine, sessionID, slot := loadSpectralSteedSession(
		t, PlatformPC, []uint32{spectralSteedDefaultFlag})

	if _, err := engine.SetSpectralSteedAttire(
		sessionID, slot, spectralSteedTestAttires, spectralSteedTreeFlag, "0",
	); err == nil || !strings.Contains(err.Error(), "cannot be worn") {
		t.Fatalf("missing item error = %v", err)
	}
	assertSpectralSteedRevision(t, engine, sessionID, slot, "0")
	assertSpectralSteedClean(t, engine, sessionID)
	assertSpectralSteedFlags(
		t, engine, sessionID, slot, spectralSteedOnly(spectralSteedDefaultFlag))
}

func TestSetSpectralSteedAttireRejectsInvalidInput(t *testing.T) {
	engine, sessionID, slot := loadSpectralSteedSession(t, PlatformPC, nil)

	for name, call := range map[string]func() error{
		"unknown appearance": func() error {
			_, err := engine.SetSpectralSteedAttire(
				sessionID, slot, spectralSteedTestAttires, 6704, "0")
			return err
		},
		"malformed revision": func() error {
			_, err := engine.SetSpectralSteedAttire(
				sessionID, slot, spectralSteedTestAttires, spectralSteedDefaultFlag, "00")
			return err
		},
		"stale revision": func() error {
			_, err := engine.SetSpectralSteedAttire(
				sessionID, slot, spectralSteedTestAttires, spectralSteedDefaultFlag, "7")
			return err
		},
		"empty set": func() error {
			_, err := engine.SetSpectralSteedAttire(sessionID, slot, nil, spectralSteedDefaultFlag, "0")
			return err
		},
		"two default appearances": func() error {
			_, err := engine.SetSpectralSteedAttire(sessionID, slot, []SpectralSteedAttire{
				{EventFlagID: spectralSteedDefaultFlag}, {EventFlagID: spectralSteedTreeFlag},
			}, spectralSteedDefaultFlag, "0")
			return err
		},
		"unsupported event flag block": func() error {
			_, err := engine.SetSpectralSteedAttire(sessionID, slot, []SpectralSteedAttire{
				{EventFlagID: 5000}, {EventFlagID: spectralSteedTreeFlag, GameID: spectralSteedTreeGameID},
			}, 5000, "0")
			return err
		},
		"non-goods game ID": func() error {
			_, err := engine.SetSpectralSteedAttire(sessionID, slot, []SpectralSteedAttire{
				{EventFlagID: spectralSteedDefaultFlag},
				{EventFlagID: spectralSteedTreeFlag, GameID: 0x00000010},
			}, spectralSteedDefaultFlag, "0")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); err == nil {
				t.Fatal("invalid input was accepted")
			}
		})
	}

	assertSpectralSteedRevision(t, engine, sessionID, slot, "0")
	assertSpectralSteedClean(t, engine, sessionID)
}

func TestLockAllSpectralSteedAttiresRemovesOnlyTheThreeAttireItems(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, sessionID, slot := loadSpectralSteedSession(t, platform, nil)
			revision := "0"
			for _, gameID := range []uint32{
				spectralSteedTreeGameID, spectralSteedSilverGameID,
				spectralSteedFuneralID, spectralSteedUnrelatedID,
			} {
				revision = addSpectralSteedItem(t, engine, sessionID, slot, gameID, revision)
			}
			selected, err := engine.SetSpectralSteedAttire(
				sessionID, slot, spectralSteedTestAttires, spectralSteedFuneralFlag, revision)
			if err != nil {
				t.Fatalf("SetSpectralSteedAttire: %v", err)
			}
			if count := spectralSteedCommonCount(t, engine, sessionID, slot); count != 4 {
				t.Fatalf("common Inventory count before Lock All = %d, want 4", count)
			}

			result, err := engine.LockAllSpectralSteedAttires(
				sessionID, slot, spectralSteedTestAttires, selected.SaveRevision)
			if err != nil {
				t.Fatalf("LockAllSpectralSteedAttires: %v", err)
			}
			if result.CharacterID != slot {
				t.Errorf("result = %+v, want character %d", result, slot)
			}

			assertLocked := func(engine *Engine, sessionID string) {
				t.Helper()
				assertSpectralSteedFlags(t, engine, sessionID, slot,
					spectralSteedOnly(spectralSteedDefaultFlag))
				present := spectralSteedItemPresence(t, engine, sessionID, slot)
				for _, gameID := range []uint32{
					spectralSteedTreeGameID, spectralSteedSilverGameID, spectralSteedFuneralID,
				} {
					if present[gameID] {
						t.Errorf("attire item 0x%08X survived Lock All", gameID)
					}
				}
				if !present[spectralSteedUnrelatedID] {
					t.Error("Lock All removed an unrelated Inventory record")
				}
			}
			assertLocked(engine, sessionID)
			if count := spectralSteedCommonCount(t, engine, sessionID, slot); count != 1 {
				t.Errorf("common Inventory count after Lock All = %d, want 1", count)
			}

			target := filepath.Join(t.TempDir(), "written.sl2")
			if _, err := engine.WriteSave(sessionID, result.SaveRevision, target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			assertLocked(reloadedEngine, reloaded.SaveSessionID)
		})
	}
}

func TestLockAllSpectralSteedAttiresIsIdempotentWithoutItems(t *testing.T) {
	engine, sessionID, slot := loadSpectralSteedSession(
		t, PlatformPC, []uint32{spectralSteedTreeFlag})

	if _, err := engine.LockAllSpectralSteedAttires(
		sessionID, slot, spectralSteedTestAttires, "0"); err != nil {
		t.Fatalf("LockAllSpectralSteedAttires: %v", err)
	}
	assertSpectralSteedFlags(
		t, engine, sessionID, slot, spectralSteedOnly(spectralSteedDefaultFlag))
}

func TestLockAllSpectralSteedAttiresLeavesNoPartialMutation(t *testing.T) {
	engine, sessionID, slot := loadSpectralSteedSession(t, PlatformPC, nil)
	revision := addSpectralSteedItem(
		t, engine, sessionID, slot, spectralSteedTreeGameID, "0")
	revision = addSpectralSteedItem(
		t, engine, sessionID, slot, spectralSteedSilverGameID, revision)
	// A second physical record of one attire is a state this mutation refuses to
	// resolve. The refusal has to arrive after the first attire was already
	// planned, so it proves the whole plan is discarded rather than half applied.
	duplicate, err := engine.AddItemToInventory(
		sessionID, slot, spectralSteedSilverGameID, 1, revision, true, 1, 2)
	if err != nil {
		t.Fatalf("AddItemToInventory duplicate: %v", err)
	}
	selected, err := engine.SetSpectralSteedAttire(
		sessionID, slot, spectralSteedTestAttires,
		spectralSteedTreeFlag, duplicate.SaveRevision)
	if err != nil {
		t.Fatalf("SetSpectralSteedAttire: %v", err)
	}

	if _, err := engine.LockAllSpectralSteedAttires(
		sessionID, slot, spectralSteedTestAttires, selected.SaveRevision,
	); err == nil || !strings.Contains(err.Error(), "Inventory records") {
		t.Fatalf("duplicate record error = %v", err)
	}

	assertSpectralSteedRevision(t, engine, sessionID, slot, selected.SaveRevision)
	assertSpectralSteedFlags(
		t, engine, sessionID, slot, spectralSteedOnly(spectralSteedTreeFlag))
	present := spectralSteedItemPresence(t, engine, sessionID, slot)
	if !present[spectralSteedTreeGameID] || !present[spectralSteedSilverGameID] {
		t.Error("the refused Lock All removed an Inventory record")
	}
}
