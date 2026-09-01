package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	setWeaponAoWFirstHandle  = uint32(0xC0000200)
	setWeaponAoWSecondHandle = uint32(0xC0000201)
	setWeaponAoWFirstGameID  = uint32(0x8000EA60)
	setWeaponAoWSecondGameID = uint32(0x8000EA61)
)

func writeSetWeaponAshOfWarFixture(
	t *testing.T,
	platform Platform,
	currentAshOfWarHandle uint32,
) string {
	t.Helper()
	path := writeSetEquippedArmamentsFixture(t, platform)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	slotBase := int64(inventoryTestPCSlotDataBase + setArmamentsSlot*inventoryTestPCSlotStride)
	if platform == PlatformPS4 {
		slotBase = int64(inventoryTestPS4SlotDataBase + setArmamentsSlot*inventoryTestPS4SlotStride)
	}
	aoWAt := slotBase + 0x20 + setArmamentsRecordCount*gaItemWeaponRecordSize
	for index, record := range []struct {
		handle uint32
		gameID uint32
	}{
		{handle: setWeaponAoWFirstHandle, gameID: setWeaponAoWFirstGameID},
		{handle: setWeaponAoWSecondHandle, gameID: setWeaponAoWSecondGameID},
	} {
		at := aoWAt + int64(index*gaItemRecordSize)
		binary.LittleEndian.PutUint32(data[at:], record.handle)
		binary.LittleEndian.PutUint32(data[at+4:], record.gameID)
	}
	weaponAt := slotBase + 0x20 + gaItemWeaponRecordSize
	binary.LittleEndian.PutUint32(data[weaponAt+16:], currentAshOfWarHandle)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func setWeaponAshOfWarTarget(
	t *testing.T,
	platform Platform,
	currentAshOfWarHandle uint32,
) (*Engine, SessionInfo, string) {
	t.Helper()
	engine := New()
	loaded, err := engine.LoadSave(
		writeSetWeaponAshOfWarFixture(t, platform, currentAshOfWarHandle), string(platform), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
	if err != nil || len(inventory.Records) < 2 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}
	return engine, loaded, inventory.Records[1].OwnedItemID
}

func TestSetWeaponAshOfWarAttachesChangesRemovesAndReloadsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, loaded, token := setWeaponAshOfWarTarget(
				t, platform, legacyNoCustomAshOfWarHandle)
			before := addItemTestSlotData(
				t, engine, loaded.SaveSessionID, platform, setArmamentsSlot)

			attached, err := engine.SetWeaponAshOfWar(
				loaded.SaveSessionID, setArmamentsSlot, token, "0",
				setArmamentsWeaponGameID, setWeaponAoWFirstGameID)
			if err != nil {
				t.Fatalf("attach SetWeaponAshOfWar: %v", err)
			}
			if attached.SaveRevision != "1" || attached.Container != ownedContainerInventory ||
				attached.PreviousAshOfWarGameID != 0 ||
				attached.AshOfWarGameID != setWeaponAoWFirstGameID {
				t.Fatalf("attached result = %+v", attached)
			}

			afterAttach := addItemTestSlotData(
				t, engine, loaded.SaveSessionID, platform, setArmamentsSlot)
			weaponReferenceAt := int64(0x20 + gaItemWeaponRecordSize + 16)
			for offset := range before {
				if before[offset] == afterAttach[offset] {
					continue
				}
				if int64(offset) < weaponReferenceAt || int64(offset) >= weaponReferenceAt+4 {
					t.Fatalf("attach changed unexpected slot byte 0x%X", offset)
				}
			}
			if got := binary.LittleEndian.Uint32(afterAttach[weaponReferenceAt:]); got != setWeaponAoWFirstHandle {
				t.Fatalf("attached handle = 0x%08X", got)
			}
			inventory, err := engine.GetInventory(
				loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
			if err != nil {
				t.Fatalf("GetInventory after attach: %v", err)
			}
			token = inventory.Records[1].OwnedItemID

			changed, err := engine.SetWeaponAshOfWar(
				loaded.SaveSessionID, setArmamentsSlot, token, "1",
				setArmamentsWeaponGameID, setWeaponAoWSecondGameID)
			if err != nil {
				t.Fatalf("change SetWeaponAshOfWar: %v", err)
			}
			if changed.PreviousAshOfWarGameID != setWeaponAoWFirstGameID ||
				changed.AshOfWarGameID != setWeaponAoWSecondGameID {
				t.Fatalf("changed result = %+v", changed)
			}
			inventory, err = engine.GetInventory(
				loaded.SaveSessionID, setArmamentsSlot, InventorySectionCommon, 1, 50)
			if err != nil {
				t.Fatalf("GetInventory after change: %v", err)
			}
			token = inventory.Records[1].OwnedItemID

			removed, err := engine.SetWeaponAshOfWar(
				loaded.SaveSessionID, setArmamentsSlot, token, "2",
				setArmamentsWeaponGameID, 0)
			if err != nil {
				t.Fatalf("remove SetWeaponAshOfWar: %v", err)
			}
			if removed.SaveRevision != "3" ||
				removed.PreviousAshOfWarGameID != setWeaponAoWSecondGameID ||
				removed.AshOfWarGameID != 0 {
				t.Fatalf("removed result = %+v", removed)
			}

			target := filepath.Join(t.TempDir(), "weapon-aow-reloaded.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "3", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			reloadedSlot := addItemTestSlotData(
				t, reloadedEngine, reloaded.SaveSessionID, platform, setArmamentsSlot)
			if got := binary.LittleEndian.Uint32(reloadedSlot[weaponReferenceAt:]); got != noCustomAshOfWarHandle {
				t.Fatalf("reloaded no-custom handle = 0x%08X", got)
			}
		})
	}
}

func TestSetWeaponAshOfWarRejectsUnsafeReferencesWithoutMutation(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		setup func(*Engine, string, int64)
		want  string
	}{
		{
			name: "shared current handle",
			setup: func(engine *Engine, sessionID string, slotBase int64) {
				otherWeaponAt := slotBase + 0x20 + 2*gaItemWeaponRecordSize
				binary.LittleEndian.PutUint32(
					engine.sessions[sessionID].snapshot.data[otherWeaponAt+16:],
					setWeaponAoWFirstHandle)
			},
			want: "referenced by 2 weapon records",
		},
		{
			name: "no free target copy",
			setup: func(engine *Engine, sessionID string, slotBase int64) {
				otherWeaponAt := slotBase + 0x20 + 2*gaItemWeaponRecordSize
				binary.LittleEndian.PutUint32(
					engine.sessions[sessionID].snapshot.data[otherWeaponAt+16:],
					setWeaponAoWSecondHandle)
			},
			want: "no unique free Ash of War GaItem",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			current := setWeaponAoWFirstHandle
			target := uint32(0)
			if testCase.name == "no free target copy" {
				current = noCustomAshOfWarHandle
				target = setWeaponAoWSecondGameID
			}
			engine, loaded, token := setWeaponAshOfWarTarget(t, PlatformPC, current)
			slotBase := int64(
				inventoryTestPCSlotDataBase + setArmamentsSlot*inventoryTestPCSlotStride)
			engine.mutex.Lock()
			testCase.setup(engine, loaded.SaveSessionID, slotBase)
			before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)
			engine.mutex.Unlock()

			_, err := engine.SetWeaponAshOfWar(
				loaded.SaveSessionID, setArmamentsSlot, token, "0",
				setArmamentsWeaponGameID, target)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			engine.mutex.Lock()
			defer engine.mutex.Unlock()
			if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) ||
				engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" {
				t.Fatal("rejected mutation changed snapshot or revision")
			}
		})
	}
}

// The Ash of War writer addresses its target through the shared OwnedItemID
// registry too, so it accepts a Storage common weapon. This is the evidence for
// the storage scope kindSetWeaponAshOfWar reports.
func TestSetWeaponAshOfWarSupportsStorageCommon(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeSetWeaponAshOfWarFixture(t, PlatformPC, legacyNoCustomAshOfWarHandle), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	ownedItemID := moveArmamentWeaponToStorageCommon(t, engine, loaded.SaveSessionID)

	result, err := engine.SetWeaponAshOfWar(
		loaded.SaveSessionID, setArmamentsSlot, ownedItemID, "0",
		setArmamentsWeaponGameID, setWeaponAoWFirstGameID)
	if err != nil {
		t.Fatalf("SetWeaponAshOfWar: %v", err)
	}
	if result.Container != ownedContainerStorage ||
		result.AshOfWarGameID != setWeaponAoWFirstGameID {
		t.Fatalf("result = %+v", result)
	}
}
