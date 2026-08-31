package saveengine

import (
	"path/filepath"
	"strings"
	"testing"
)

const (
	// Limgrave, West is the confirmed pair of a block 62 visibility flag and its
	// Map Fragment; the Stormveil dungeon map is a region of the same table that
	// has no fragment at all.
	mapRegionWestFlag     = uint32(62010)
	mapRegionWestFragment = uint32(0x40002198)
	mapRegionDungeonFlag  = uint32(62100)
	// Block 63 is at BST position 13 in the confirmed table. It stays independent
	// of the production resolver, which deliberately does not accept this block.
	mapRegionAcquiredBlockPosition  = int64(13)
	mapRegionTestEventFlagBlockSize = int64(125)
	mapRegionTestFlagsPerBlock      = uint32(1000)
	// The acquired flag of Limgrave, West. It is a transient pickup trigger the
	// engine deliberately cannot even resolve, so it can only be named as a
	// rejected request.
	mapRegionWestAcquiredFlag = uint32(63010)
	mapRegionWestAcquiredMask = byte(1 << 5)
)

func TestSetMapRegionRevealedWritesFlagAndFragmentOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			path, content := setWhetbladeFixture(t, platform)
			engine := New()
			loaded, err := engine.LoadSave(path, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			acquiredAt, acquiredBefore := mapRegionAcquiredFixtureByte(
				t, engine, loaded.SaveSessionID, content.slot)
			acquiredWant := acquiredBefore | mapRegionWestAcquiredMask
			if err := engine.sessions[loaded.SaveSessionID].snapshot.writeAt(
				acquiredAt, []byte{acquiredWant}); err != nil {
				t.Fatalf("write acquired fixture flag: %v", err)
			}

			result, err := engine.SetMapRegionRevealed(loaded.SaveSessionID, content.slot,
				mapRegionWestFlag, mapRegionWestFragment, true, "0")
			if err != nil {
				t.Fatalf("SetMapRegionRevealed: %v", err)
			}
			want := SetMapRegionRevealedResult{
				SaveSessionID: loaded.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   content.slot,
				Revealed:      true,
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}
			assertMapRegionState(t, engine, loaded.SaveSessionID, content.slot, true)
			if _, got := mapRegionAcquiredFixtureByte(
				t, engine, loaded.SaveSessionID, content.slot); got != acquiredWant {
				t.Errorf("acquired fixture byte = 0x%02X, want 0x%02X", got, acquiredWant)
			}

			// The state has to survive the codec of the platform, not only the
			// snapshot it was written into.
			target := filepath.Join(t.TempDir(), "written.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			assertMapRegionState(t, reloadedEngine, reloaded.SaveSessionID, content.slot, true)
			if _, got := mapRegionAcquiredFixtureByte(
				t, reloadedEngine, reloaded.SaveSessionID, content.slot); got != acquiredWant {
				t.Errorf("reloaded acquired byte = 0x%02X, want 0x%02X", got, acquiredWant)
			}

			if _, err := engine.SetMapRegionRevealed(loaded.SaveSessionID, content.slot,
				mapRegionWestFlag, mapRegionWestFragment, false, "2"); err != nil {
				t.Fatalf("SetMapRegionRevealed hide: %v", err)
			}
			assertMapRegionState(t, engine, loaded.SaveSessionID, content.slot, false)
			if _, got := mapRegionAcquiredFixtureByte(
				t, engine, loaded.SaveSessionID, content.slot); got != acquiredWant {
				t.Errorf("acquired byte after hide = 0x%02X, want 0x%02X", got, acquiredWant)
			}
		})
	}
}

// A region without a Map Fragment writes exactly one bit and touches no other
// region, so the two independent parts of the contract cannot be conflated.
func TestSetMapRegionRevealedWithoutFragmentWritesOneBit(t *testing.T) {
	path, content := setWhetbladeFixture(t, PlatformPC)
	engine := New()
	loaded, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)

	if _, err := engine.SetMapRegionRevealed(loaded.SaveSessionID, content.slot,
		mapRegionDungeonFlag, 0, true, "0"); err != nil {
		t.Fatalf("SetMapRegionRevealed: %v", err)
	}

	changed := changedSnapshotBytes(before, engine.sessions[loaded.SaveSessionID].snapshot.data)
	if len(changed) != 1 {
		t.Fatalf("changed snapshot bytes = %v, want exactly the one flag byte", changed)
	}
	flags, err := engine.GetEventFlags(loaded.SaveSessionID, content.slot,
		[]uint32{mapRegionDungeonFlag, mapRegionWestFlag})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if !flags.Flags[mapRegionDungeonFlag] {
		t.Error("dungeon map flag is false, want true")
	}
	if flags.Flags[mapRegionWestFlag] {
		t.Error("another map region was written")
	}
}

func TestSetMapRegionRevealedRejectsInvalidRequestsWithoutMutating(t *testing.T) {
	path, content := setWhetbladeFixture(t, PlatformPC)
	engine := New()
	loaded, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)

	for name, testCase := range map[string]struct {
		slot     int
		flagID   uint32
		gameID   uint32
		revision string
		want     string
	}{
		"stale revision": {
			content.slot, mapRegionWestFlag, mapRegionWestFragment, "7",
			"does not match the current saveRevision",
		},
		"malformed revision": {
			content.slot, mapRegionWestFlag, mapRegionWestFragment, "01",
			"canonical decimal saveRevision",
		},
		// The acquired flag of block 63 is the one a caller must never reach.
		"acquired flag": {
			content.slot, mapRegionWestAcquiredFlag, 0, "0",
			"lies in block 63, want block 62",
		},
		"non-goods fragment": {
			content.slot, mapRegionWestFlag, 0x80000000, "0",
			"is not a goods ID",
		},
		"inactive slot": {
			content.slot + 1, mapRegionWestFlag, mapRegionWestFragment, "0",
			"is not active",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := engine.SetMapRegionRevealed(loaded.SaveSessionID, testCase.slot,
				testCase.flagID, testCase.gameID, true, testCase.revision)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want one containing %q", err, testCase.want)
			}
		})
	}

	if changed := changedSnapshotBytes(
		before, engine.sessions[loaded.SaveSessionID].snapshot.data); len(changed) != 0 {
		t.Fatalf("changed snapshot bytes = %v, want none after rejections", changed)
	}
	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Errorf("session after rejections = %+v, want clean", info)
	}
}

func assertMapRegionState(
	t *testing.T,
	engine *Engine,
	sessionID string,
	characterID int,
	want bool,
) {
	t.Helper()
	flags, err := engine.GetEventFlags(sessionID, characterID,
		[]uint32{mapRegionWestFlag, mapRegionDungeonFlag})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	if flags.Flags[mapRegionWestFlag] != want {
		t.Errorf("visibility flag = %t, want %t", flags.Flags[mapRegionWestFlag], want)
	}
	if flags.Flags[mapRegionDungeonFlag] {
		t.Error("an unrelated map region flag was written")
	}
	present, err := engine.GetInventoryGoodsPresence(
		sessionID, characterID, []uint32{mapRegionWestFragment})
	if err != nil {
		t.Fatalf("GetInventoryGoodsPresence: %v", err)
	}
	if present[mapRegionWestFragment] != want {
		t.Errorf("Map Fragment present = %t, want %t", present[mapRegionWestFragment], want)
	}
	if want {
		handle, err := gaItemHandleForGameID(mapRegionWestFragment)
		if err != nil {
			t.Fatalf("gaItemHandleForGameID: %v", err)
		}
		common, err := engine.GetInventory(
			sessionID, characterID, InventorySectionCommon, 1, inventoryHeldCommonRecords)
		if err != nil {
			t.Fatalf("GetInventory common: %v", err)
		}
		key, err := engine.GetInventory(
			sessionID, characterID, InventorySectionKey, 1, inventoryHeldKeyRecords)
		if err != nil {
			t.Fatalf("GetInventory key: %v", err)
		}
		foundCommon := false
		for _, record := range common.Records {
			if record.GaItemHandle == handle && record.Quantity == 1 {
				foundCommon = true
			}
		}
		for _, record := range key.Records {
			if record.GaItemHandle == mapRegionWestFragment || record.GaItemHandle == handle {
				t.Fatalf("Map Fragment was written to key Inventory: %+v", record)
			}
		}
		if !foundCommon {
			t.Fatal("Map Fragment was not written to common Inventory with quantity 1")
		}
	}
}

func mapRegionAcquiredFixtureByte(
	t *testing.T,
	engine *Engine,
	sessionID string,
	characterID int,
) (int64, byte) {
	t.Helper()
	loaded := engine.sessions[sessionID]
	sectionAt, err := eventFlagSectionStart(loaded, characterID)
	if err != nil {
		t.Fatalf("eventFlagSectionStart: %v", err)
	}
	index := int64(mapRegionWestAcquiredFlag % mapRegionTestFlagsPerBlock)
	at := sectionAt + mapRegionAcquiredBlockPosition*mapRegionTestEventFlagBlockSize + index/8
	raw, err := loaded.snapshot.readAt(at, 1)
	if err != nil {
		t.Fatalf("read acquired fixture flag: %v", err)
	}
	return at, raw[0]
}
