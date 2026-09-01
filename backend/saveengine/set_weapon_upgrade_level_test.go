package saveengine

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"
)

const (
	setWeaponUpgradeHandle  = uint32(0x80000101)
	setWeaponUpgradeCurrent = uint32(0x000F4240)
	setWeaponUpgradeTarget  = uint32(0x000F4245)
)

func setWeaponUpgradeUint32(slot []byte, at int64) uint32 {
	return binary.LittleEndian.Uint32(slot[at:])
}

func equipSetWeaponUpgradeFixture(
	t *testing.T,
	engine *Engine,
	saveSessionID string,
	platform Platform,
) {
	t.Helper()
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded := engine.sessions[saveSessionID]
	slotBase := addItemTestSlotBase(t, platform, setArmamentsSlot)
	anchor := slotBase + setArmamentsAnchorAt
	fields := []struct {
		at    int64
		value uint32
	}{
		{anchor + setArmamentsIndexesAt, setArmamentsInventoryBase + 1},
		{anchor + setArmamentsItemIDsAt, setWeaponUpgradeCurrent},
		{anchor + setArmamentsHandlesAt, setWeaponUpgradeHandle},
		{anchor + setArmamentsProjectileAt + 4, setWeaponUpgradeCurrent},
	}
	for _, field := range fields {
		if err := loaded.snapshot.writeAt(field.at, littleEndianUint32(field.value)); err != nil {
			t.Fatalf("write fixture field at 0x%X: %v", field.at, err)
		}
	}
}

func TestSetWeaponUpgradeLevelUpdatesGaItemEquipmentAndGaItemDataOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeSetEquippedArmamentsFixture(t, platform), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			equipSetWeaponUpgradeFixture(t, engine, loaded.SaveSessionID, platform)
			inventory, err := engine.GetInventory(
				loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}
			token := inventory.Records[1].OwnedItemID
			before := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, setArmamentsSlot)

			result, err := engine.SetWeaponUpgradeLevel(
				loaded.SaveSessionID, setArmamentsSlot, token, 5, "0",
				setWeaponUpgradeCurrent, setWeaponUpgradeTarget, 5)
			if err != nil {
				t.Fatalf("SetWeaponUpgradeLevel: %v", err)
			}
			if result.SaveRevision != "1" || result.Container != "inventory" ||
				result.PreviousGameID != setWeaponUpgradeCurrent ||
				result.GameID != setWeaponUpgradeTarget || result.UpgradeLevel != 5 {
				t.Fatalf("result = %+v", result)
			}

			after := addItemTestSlotData(t, engine, loaded.SaveSessionID, platform, setArmamentsSlot)
			weaponRecordAt := int64(0x20 + gaItemWeaponRecordSize + 4)
			if got := setWeaponUpgradeUint32(after, weaponRecordAt); got != setWeaponUpgradeTarget {
				t.Errorf("GaItem game ID = 0x%08X, want 0x%08X", got, setWeaponUpgradeTarget)
			}
			if got := setWeaponUpgradeUint32(after, setArmamentsAnchorAt+setArmamentsItemIDsAt); got != setWeaponUpgradeTarget {
				t.Errorf("equipped bare ID = 0x%08X", got)
			}
			if got := setWeaponUpgradeUint32(after,
				setArmamentsAnchorAt+setArmamentsProjectileAt+4); got != setWeaponUpgradeTarget {
				t.Errorf("dynamic equipped ID = 0x%08X", got)
			}
			if got := setWeaponUpgradeUint32(after, setArmamentsAnchorAt+addItemTestGaItemDataAt); got != 1 {
				t.Fatalf("GaItemData count = %d, want 1", got)
			}
			if got := setWeaponUpgradeUint32(after,
				setArmamentsAnchorAt+addItemTestGaItemDataArrayAt); got != setWeaponUpgradeTarget {
				t.Errorf("GaItemData ID = 0x%08X", got)
			}
			if got := after[setArmamentsAnchorAt+statsMatchmakingWeaponLevelOffset]; got != 5 {
				t.Errorf("matchmaking level = %d, want 5", got)
			}
			allowed := [][2]int64{
				{weaponRecordAt, weaponRecordAt + 4},
				{setArmamentsAnchorAt + setArmamentsItemIDsAt,
					setArmamentsAnchorAt + setArmamentsItemIDsAt + 4},
				{setArmamentsAnchorAt + setArmamentsProjectileAt + 4,
					setArmamentsAnchorAt + setArmamentsProjectileAt + 8},
				{setArmamentsAnchorAt + addItemTestGaItemDataAt,
					setArmamentsAnchorAt + addItemTestGaItemDataAt + 4},
				{setArmamentsAnchorAt + addItemTestGaItemDataArrayAt,
					setArmamentsAnchorAt + addItemTestGaItemDataArrayAt + 8},
				{setArmamentsAnchorAt + statsMatchmakingWeaponLevelOffset,
					setArmamentsAnchorAt + statsMatchmakingWeaponLevelOffset + 1},
			}
			for offset := range before {
				if before[offset] == after[offset] {
					continue
				}
				inside := false
				for _, span := range allowed {
					inside = inside || int64(offset) >= span[0] && int64(offset) < span[1]
				}
				if !inside {
					t.Fatalf("unexpected byte modified at slot offset 0x%X", offset)
				}
			}

			target := filepath.Join(t.TempDir(), "weapon-upgrade-reloaded.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			reloadedInventory, err := reloadedEngine.GetInventory(
				reloaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
			if err != nil {
				t.Fatalf("GetInventory after reload: %v", err)
			}
			gameIDs, err := reloadedEngine.ResolveGaItemIDs(
				reloaded.SaveSessionID, setArmamentsSlot,
				[]uint32{reloadedInventory.Records[1].GaItemHandle})
			if err != nil || gameIDs[0] != setWeaponUpgradeTarget {
				t.Fatalf("reloaded game ID = %v, err=%v", gameIDs, err)
			}
			reloadedEngine.mutex.Lock()
			reloadedAnchor, err := findStatsAnchor(
				reloadedEngine.sessions[reloaded.SaveSessionID].snapshot, platform, setArmamentsSlot)
			if err != nil {
				reloadedEngine.mutex.Unlock()
				t.Fatalf("findStatsAnchor reloaded (%s): %v", platform, err)
			}
			reloadedMatchmaking, err := reloadedEngine.sessions[reloaded.SaveSessionID].snapshot.readAt(
				reloadedAnchor+statsMatchmakingWeaponLevelOffset, 1)
			reloadedEngine.mutex.Unlock()
			if err != nil {
				t.Fatalf("read matchmaking level after reload (%s): %v", platform, err)
			}
			if reloadedMatchmaking[0] != 5 {
				t.Errorf("reloaded %s matchmaking level = %d, want 5", platform, reloadedMatchmaking[0])
			}
		})
	}
}

func TestSetWeaponUpgradeLevelRejectsAmbiguousGaItemWithoutMutation(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeSetEquippedArmamentsFixture(t, PlatformPC), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	token := inventory.Records[1].OwnedItemID
	engine.mutex.Lock()
	duplicateAt := addItemTestSlotBase(t, PlatformPC, setArmamentsSlot) + 0x20 + 2*gaItemWeaponRecordSize
	binary.LittleEndian.PutUint32(engine.sessions[loaded.SaveSessionID].snapshot.data[duplicateAt:],
		setWeaponUpgradeHandle)
	binary.LittleEndian.PutUint32(engine.sessions[loaded.SaveSessionID].snapshot.data[duplicateAt+4:],
		setWeaponUpgradeCurrent)
	before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)
	engine.mutex.Unlock()

	_, err = engine.SetWeaponUpgradeLevel(
		loaded.SaveSessionID, setArmamentsSlot, token, 5, "0",
		setWeaponUpgradeCurrent, setWeaponUpgradeTarget, 5)
	if err == nil || !strings.Contains(err.Error(), "want exactly 1") {
		t.Fatalf("duplicate GaItem error = %v", err)
	}
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) ||
		engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" {
		t.Fatal("rejected mutation changed snapshot or revision")
	}
}

// moveArmamentWeaponToStorageCommon rewrites the shared armament fixture of an
// already loaded session so the weapon carrying setWeaponUpgradeHandle sits in
// Storage common instead of Inventory common, and returns its OwnedItemID.
//
// Every weapon writer resolves its target through the same opaque OwnedItemID,
// so this one piece of fixture surgery is what proves that each of them accepts
// a Storage record. That is the evidence behind the storage scope those kinds
// report in TestRepresentativeMutationsReportTheirExactChangedScopes.
func moveArmamentWeaponToStorageCommon(t *testing.T, engine *Engine, saveSessionID string) string {
	t.Helper()

	engine.mutex.Lock()
	snapshot := engine.sessions[saveSessionID].snapshot
	slotBase := addItemTestSlotBase(t, PlatformPC, setArmamentsSlot)
	anchor := slotBase + setArmamentsAnchorAt
	inventoryRow := slotBase + setArmamentsInventoryAt
	if err := snapshot.writeAt(inventoryRow+setArmamentsInventoryStride,
		make([]byte, setArmamentsInventoryStride)); err != nil {
		engine.mutex.Unlock()
		t.Fatalf("clear inventory row: %v", err)
	}
	storageRow := anchor + addItemTestStorageAt + addItemTestStorageCommonAt
	for _, field := range []struct {
		at    int64
		value uint32
	}{{storageRow, setWeaponUpgradeHandle}, {storageRow + 4, 1}, {storageRow + 8, 7}} {
		if err := snapshot.writeAt(field.at, littleEndianUint32(field.value)); err != nil {
			engine.mutex.Unlock()
			t.Fatalf("write storage field: %v", err)
		}
	}
	engine.mutex.Unlock()

	storage, err := engine.GetStorage(
		saveSessionID, setArmamentsSlot, StorageSectionCommon, 1, 50)
	if err != nil || len(storage.Records) != 1 {
		t.Fatalf("GetStorage: %v, len=%d", err, len(storage.Records))
	}
	return storage.Records[0].OwnedItemID
}

func TestSetWeaponUpgradeLevelSupportsStorageCommon(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeSetEquippedArmamentsFixture(t, PlatformPC), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	ownedItemID := moveArmamentWeaponToStorageCommon(t, engine, loaded.SaveSessionID)

	result, err := engine.SetWeaponUpgradeLevel(
		loaded.SaveSessionID, setArmamentsSlot, ownedItemID, 5, "0",
		setWeaponUpgradeCurrent, setWeaponUpgradeTarget, 5)
	if err != nil {
		t.Fatalf("SetWeaponUpgradeLevel: %v", err)
	}
	if result.Container != "storage" || result.GameID != setWeaponUpgradeTarget {
		t.Fatalf("result = %+v", result)
	}
}

func TestSetWeaponUpgradeLevelMatchmakingMonotonicityAndFailClosed(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeSetEquippedArmamentsFixture(t, PlatformPC), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	token := inventory.Records[1].OwnedItemID
	slotBase := addItemTestSlotBase(t, PlatformPC, setArmamentsSlot)
	matchmakingAt := slotBase + setArmamentsAnchorAt + statsMatchmakingWeaponLevelOffset

	// 1. Initial upgrade to matchmaking level 25 raises the byte from 0 to 25
	res1, err := engine.SetWeaponUpgradeLevel(
		loaded.SaveSessionID, setArmamentsSlot, token, 25, "0",
		setWeaponUpgradeCurrent, setWeaponUpgradeCurrent+25, 25)
	if err != nil {
		t.Fatalf("SetWeaponUpgradeLevel +25: %v", err)
	}
	if got := engine.sessions[loaded.SaveSessionID].snapshot.data[matchmakingAt]; got != 25 {
		t.Fatalf("matchmaking level = %d, want 25", got)
	}

	// 2. Subsequent lower upgrade (e.g. level 5) mutates the weapon but NEVER lowers matchmaking level
	inventory, err = engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	token = inventory.Records[1].OwnedItemID
	_, err = engine.SetWeaponUpgradeLevel(
		loaded.SaveSessionID, setArmamentsSlot, token, 5, res1.SaveRevision,
		setWeaponUpgradeCurrent+25, setWeaponUpgradeCurrent+5, 5)
	if err != nil {
		t.Fatalf("SetWeaponUpgradeLevel +5: %v", err)
	}
	if got := engine.sessions[loaded.SaveSessionID].snapshot.data[matchmakingAt]; got != 25 {
		t.Fatalf("matchmaking level after downgrade = %d, want 25 (monotonic)", got)
	}

	// 3. Fail-closed: invalid matchmaking level > 25 fails and rolls back
	inventory, err = engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	token = inventory.Records[1].OwnedItemID
	currRev := engine.sessions[loaded.SaveSessionID].session.revisionString()
	beforeData := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)

	if _, err := engine.SetWeaponUpgradeLevel(
		loaded.SaveSessionID, setArmamentsSlot, token, 26, currRev,
		setWeaponUpgradeCurrent+5, setWeaponUpgradeCurrent+26, 26); err == nil {
		t.Fatal("SetWeaponUpgradeLevel with matchmaking level 26 succeeded, want error")
	}

	if !bytes.Equal(beforeData, engine.sessions[loaded.SaveSessionID].snapshot.data) {
		t.Fatal("failed mutation modified snapshot")
	}
}
