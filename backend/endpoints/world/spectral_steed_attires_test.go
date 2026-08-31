package world

import (
	"encoding/binary"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	spectralSteedTreeGameID   = uint32(0x401EAA00)
	spectralSteedSilverGameID = uint32(0x401EAA0A)
	spectralSteedFuneralID    = uint32(0x401EAA14)
	spectralSteedUnrelatedID  = uint32(0x4000230B)
)

// spectralSteedInventory names the three physical representations the fixture
// can place: the editor handle in the common section, the raw game ID a
// game-placed key item carries, and an unrelated record Lock All must preserve.
type spectralSteedInventory struct {
	commonHandles []uint32
	keyGameIDs    []uint32
}

// writeSpectralSteedFixture builds a synthetic PC save whose slot carries the
// requested appearance flags and Inventory records.
func writeSpectralSteedFixture(
	t *testing.T, flags []uint32, inventory spectralSteedInventory, active bool,
) string {
	t.Helper()

	path := writeGetCookbooksFixture(t, nil, active)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchorBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize + getCookbooksAnchorAt

	for _, id := range flags {
		index := int64(id % 1000)
		offset := 6*getCookbooksBlockSize + index/8
		data[anchorBase+getCookbooksSectionAt+offset] |= 1 << uint8(7-index%8)
	}
	putRecord := func(at int64, handle uint32) {
		binary.LittleEndian.PutUint32(data[anchorBase+at:], handle)
		binary.LittleEndian.PutUint32(data[anchorBase+at+4:], 1)
	}
	for index, handle := range inventory.commonHandles {
		putRecord(getWhetbladesInventoryCommonAt+
			int64(index)*getWhetbladesInventoryRecord, handle)
	}
	binary.LittleEndian.PutUint32(
		data[anchorBase+getWhetbladesInventoryCommonAt-4:],
		uint32(len(inventory.commonHandles)))
	for index, gameID := range inventory.keyGameIDs {
		putRecord(getWhetbladesInventoryKeyAt+
			int64(index)*getWhetbladesInventoryRecord, gameID)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadSpectralSteedSession(
	t *testing.T, flags []uint32, inventory spectralSteedInventory, active bool,
) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(
		writeSpectralSteedFixture(t, flags, inventory, active), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

// spectralSteedGoodsHandle restates the confirmed goods handle encoding, so the
// fixture does not borrow it from the implementation.
func spectralSteedGoodsHandle(gameID uint32) uint32 {
	return 0xB0000000 | (gameID & 0x0FFFFFFF)
}

func spectralSteedOwnership(result GetSpectralSteedAttiresResult) map[string]bool {
	owned := make(map[string]bool, len(result.Attires))
	for _, attire := range result.Attires {
		owned[attire.AttireKey] = attire.Owned
	}
	return owned
}

func TestGetSpectralSteedAttiresReportsTheResolvedAppearance(t *testing.T) {
	for key, flagID := range map[string]uint32{
		SpectralSteedAttireKeyDefault:       6700,
		SpectralSteedAttireKeyTreeSentinel:  6701,
		SpectralSteedAttireKeySilverCaria:   6702,
		SpectralSteedAttireKeyFunerealNight: 6703,
	} {
		t.Run(key, func(t *testing.T) {
			engine, sessionID := loadSpectralSteedSession(
				t, []uint32{flagID}, spectralSteedInventory{}, true)
			result, err := GetSpectralSteedAttires(
				engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
			if err != nil {
				t.Fatalf("GetSpectralSteedAttires: %v", err)
			}
			if result.Status != SpectralSteedAttireStatusResolved {
				t.Errorf("status = %q, want %q", result.Status, SpectralSteedAttireStatusResolved)
			}
			if result.ActiveAttireKey != key {
				t.Errorf("activeAttireKey = %q, want %q", result.ActiveAttireKey, key)
			}
		})
	}
}

func TestGetSpectralSteedAttiresListsTheCatalogTable(t *testing.T) {
	engine, sessionID := loadSpectralSteedSession(
		t, []uint32{6701}, spectralSteedInventory{}, true)
	result, err := GetSpectralSteedAttires(
		engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSpectralSteedAttires: %v", err)
	}
	if !result.Active || result.SaveSessionID != sessionID ||
		result.CharacterID != getCookbooksSlot {
		t.Fatalf("result identity = %+v", result)
	}

	want := []SpectralSteedAttireEntry{
		{AttireKey: SpectralSteedAttireKeyDefault, Name: "Default Appearance", Owned: true},
		{AttireKey: SpectralSteedAttireKeyTreeSentinel,
			Name: "Tree Sentinel Spectral Steed Attire", RequiredResourceKind: schema.ResourceKindItem,
			RequiredResourceKey: "401EAA00",
			IconPath:            "assets/icons/items/key_items/tree_sentinel_spectral_steed_attire.png"},
		{AttireKey: SpectralSteedAttireKeySilverCaria,
			Name: "Silver of Caria Spectral Steed Attire", RequiredResourceKind: schema.ResourceKindItem,
			RequiredResourceKey: "401EAA0A",
			IconPath:            "assets/icons/items/key_items/silver_of_caria_spectral_steed_attire.png"},
		{AttireKey: SpectralSteedAttireKeyFunerealNight,
			Name: "Funereal Night Spectral Steed Attire", RequiredResourceKind: schema.ResourceKindItem,
			RequiredResourceKey: "401EAA14",
			IconPath:            "assets/icons/items/key_items/funereal_night_spectral_steed_attire.png"},
	}
	if !reflect.DeepEqual(result.Attires, want) {
		t.Errorf("attires = %+v, want %+v", result.Attires, want)
	}
}

func TestGetSpectralSteedAttiresReportsLegacyAndConflict(t *testing.T) {
	for name, flags := range map[string][]uint32{
		SpectralSteedAttireStatusLegacy:   nil,
		SpectralSteedAttireStatusConflict: {6701, 6703},
	} {
		t.Run(name, func(t *testing.T) {
			engine, sessionID := loadSpectralSteedSession(
				t, flags, spectralSteedInventory{}, true)
			result, err := GetSpectralSteedAttires(
				engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
			if err != nil {
				t.Fatalf("GetSpectralSteedAttires: %v", err)
			}
			if result.Status != name {
				t.Errorf("status = %q, want %q", result.Status, name)
			}
			if result.ActiveAttireKey != "" {
				t.Errorf("activeAttireKey = %q, want an empty key", result.ActiveAttireKey)
			}
			if len(result.Attires) != 4 {
				t.Errorf("attires = %d entries, want 4", len(result.Attires))
			}
		})
	}
}

func TestGetSpectralSteedAttiresReadsOwnershipFromInventoryOnly(t *testing.T) {
	// Tree Sentinel is worn but its item was dropped, Silver of Caria sits in the
	// common section as an editor handle and Funereal Night as the raw key-item
	// game ID the game itself writes.
	engine, sessionID := loadSpectralSteedSession(t, []uint32{6701}, spectralSteedInventory{
		commonHandles: []uint32{spectralSteedGoodsHandle(spectralSteedSilverGameID)},
		keyGameIDs:    []uint32{spectralSteedFuneralID},
	}, true)

	result, err := GetSpectralSteedAttires(
		engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSpectralSteedAttires: %v", err)
	}
	want := map[string]bool{
		SpectralSteedAttireKeyDefault:       true,
		SpectralSteedAttireKeyTreeSentinel:  false,
		SpectralSteedAttireKeySilverCaria:   true,
		SpectralSteedAttireKeyFunerealNight: true,
	}
	if got := spectralSteedOwnership(result); !reflect.DeepEqual(got, want) {
		t.Errorf("ownership = %v, want %v", got, want)
	}
	if result.ActiveAttireKey != SpectralSteedAttireKeyTreeSentinel {
		t.Errorf("activeAttireKey = %q, want the worn appearance", result.ActiveAttireKey)
	}
}

func TestGetSpectralSteedAttiresChangesNothing(t *testing.T) {
	engine, sessionID := loadSpectralSteedSession(t, []uint32{6702}, spectralSteedInventory{
		commonHandles: []uint32{spectralSteedGoodsHandle(spectralSteedSilverGameID)},
	}, true)
	gameCatalog := newCookbooksCatalog(t)

	first, err := GetSpectralSteedAttires(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSpectralSteedAttires: %v", err)
	}
	second, err := GetSpectralSteedAttires(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("repeated GetSpectralSteedAttires: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Error("a repeated read reported a different state")
	}
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Error("the getter marked the session dirty")
	}
	// The revision a mutation would have to quote must still be the loaded one.
	if _, err := SetSpectralSteedAttire(engine, gameCatalog, sessionID, getCookbooksSlot,
		SpectralSteedAttireKeyDefault, "0"); err != nil {
		t.Errorf("the getter advanced the saveRevision: %v", err)
	}
}

func TestGetSpectralSteedAttiresDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadSpectralSteedSession(t, []uint32{6703}, spectralSteedInventory{
		commonHandles: []uint32{spectralSteedGoodsHandle(spectralSteedFuneralID)},
	}, false)

	result, err := GetSpectralSteedAttires(
		engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSpectralSteedAttires: %v", err)
	}
	if result.Active || result.ActiveAttireKey != "" ||
		result.Status != SpectralSteedAttireStatusLegacy {
		t.Fatalf("residual slot reported %+v", result)
	}
	for _, attire := range result.Attires {
		if attire.AttireKey != SpectralSteedAttireKeyDefault && attire.Owned {
			t.Errorf("residual slot reports %q as owned", attire.AttireKey)
		}
	}
}

func TestSetSpectralSteedAttireCommitsEveryAppearance(t *testing.T) {
	engine, sessionID := loadSpectralSteedSession(t, nil, spectralSteedInventory{
		commonHandles: []uint32{
			spectralSteedGoodsHandle(spectralSteedTreeGameID),
			spectralSteedGoodsHandle(spectralSteedSilverGameID),
		},
		keyGameIDs: []uint32{spectralSteedFuneralID},
	}, true)
	gameCatalog := newCookbooksCatalog(t)

	revision := "0"
	for _, key := range []string{
		SpectralSteedAttireKeyTreeSentinel,
		SpectralSteedAttireKeySilverCaria,
		SpectralSteedAttireKeyFunerealNight,
		SpectralSteedAttireKeyDefault,
		// Selecting the appearance that is already active is a no-op on the save
		// and still commits one revision, like every other mutation.
		SpectralSteedAttireKeyDefault,
	} {
		result, err := SetSpectralSteedAttire(
			engine, gameCatalog, sessionID, getCookbooksSlot, key, revision)
		if err != nil {
			t.Fatalf("SetSpectralSteedAttire(%q): %v", key, err)
		}
		if result.AttireKey != key || result.CharacterID != getCookbooksSlot {
			t.Errorf("result = %+v, want appearance %q", result, key)
		}
		revision = result.SaveRevision

		state, err := GetSpectralSteedAttires(
			engine, gameCatalog, sessionID, getCookbooksSlot)
		if err != nil {
			t.Fatalf("GetSpectralSteedAttires: %v", err)
		}
		if state.Status != SpectralSteedAttireStatusResolved || state.ActiveAttireKey != key {
			t.Errorf("state after %q = %s/%q, want resolved/%q",
				key, state.Status, state.ActiveAttireKey, key)
		}
	}
}

func TestSetSpectralSteedAttireResolvesLegacyAndConflict(t *testing.T) {
	for name, flags := range map[string][]uint32{
		SpectralSteedAttireStatusLegacy:   nil,
		SpectralSteedAttireStatusConflict: {6700, 6702, 6703},
	} {
		t.Run(name, func(t *testing.T) {
			engine, sessionID := loadSpectralSteedSession(t, flags, spectralSteedInventory{
				commonHandles: []uint32{spectralSteedGoodsHandle(spectralSteedTreeGameID)},
			}, true)
			gameCatalog := newCookbooksCatalog(t)

			if _, err := SetSpectralSteedAttire(engine, gameCatalog, sessionID,
				getCookbooksSlot, SpectralSteedAttireKeyTreeSentinel, "0"); err != nil {
				t.Fatalf("SetSpectralSteedAttire: %v", err)
			}
			state, err := GetSpectralSteedAttires(
				engine, gameCatalog, sessionID, getCookbooksSlot)
			if err != nil {
				t.Fatalf("GetSpectralSteedAttires: %v", err)
			}
			if state.Status != SpectralSteedAttireStatusResolved ||
				state.ActiveAttireKey != SpectralSteedAttireKeyTreeSentinel {
				t.Errorf("state = %s/%q, want the resolved Tree Sentinel appearance",
					state.Status, state.ActiveAttireKey)
			}
		})
	}
}

func TestSetSpectralSteedAttireRejectsInvalidInput(t *testing.T) {
	engine, sessionID := loadSpectralSteedSession(
		t, []uint32{6700}, spectralSteedInventory{}, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := SetSpectralSteedAttire(
		nil, gameCatalog, sessionID, getCookbooksSlot,
		SpectralSteedAttireKeyDefault, "0"); err == nil {
		t.Fatal("nil SaveEngine was accepted")
	}
	if _, err := SetSpectralSteedAttire(
		engine, nil, sessionID, getCookbooksSlot,
		SpectralSteedAttireKeyDefault, "0"); err == nil {
		t.Fatal("nil GameCatalog was accepted")
	}
	if _, err := SetSpectralSteedAttire(
		engine, gameCatalog, sessionID, getCookbooksSlot, "tree-sentinel", "0",
	); err == nil || !strings.Contains(err.Error(), "attireKey must be one of") {
		t.Fatalf("unknown attireKey error = %v", err)
	}
	if _, err := SetSpectralSteedAttire(
		engine, gameCatalog, sessionID, getCookbooksSlot,
		SpectralSteedAttireKeyTreeSentinel, "0",
	); err == nil || !strings.Contains(err.Error(), "cannot be worn") {
		t.Fatalf("missing item error = %v", err)
	}
	if _, err := SetSpectralSteedAttire(
		engine, gameCatalog, sessionID, getCookbooksSlot,
		SpectralSteedAttireKeyDefault, "1",
	); err == nil || !strings.Contains(err.Error(), "does not match the current saveRevision") {
		t.Fatalf("stale revision error = %v", err)
	}

	state, err := GetSpectralSteedAttires(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSpectralSteedAttires: %v", err)
	}
	if state.ActiveAttireKey != SpectralSteedAttireKeyDefault {
		t.Errorf("a rejected call changed the active appearance to %q", state.ActiveAttireKey)
	}
}

func TestLockAllSpectralSteedAttiresRemovesTheThreeItems(t *testing.T) {
	engine, sessionID := loadSpectralSteedSession(t, []uint32{6702}, spectralSteedInventory{
		commonHandles: []uint32{
			spectralSteedGoodsHandle(spectralSteedTreeGameID),
			spectralSteedUnrelatedID,
			spectralSteedGoodsHandle(spectralSteedSilverGameID),
		},
		keyGameIDs: []uint32{spectralSteedFuneralID},
	}, true)
	gameCatalog := newCookbooksCatalog(t)

	result, err := LockAllSpectralSteedAttires(
		engine, gameCatalog, sessionID, getCookbooksSlot, "0")
	if err != nil {
		t.Fatalf("LockAllSpectralSteedAttires: %v", err)
	}
	if result.SaveRevision != "1" || result.AttireKey != SpectralSteedAttireKeyDefault {
		t.Errorf("result = %+v, want revision 1 and the default appearance", result)
	}

	state, err := GetSpectralSteedAttires(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSpectralSteedAttires: %v", err)
	}
	if state.Status != SpectralSteedAttireStatusResolved ||
		state.ActiveAttireKey != SpectralSteedAttireKeyDefault {
		t.Errorf("state = %s/%q, want the resolved default appearance",
			state.Status, state.ActiveAttireKey)
	}
	want := map[string]bool{
		SpectralSteedAttireKeyDefault:       true,
		SpectralSteedAttireKeyTreeSentinel:  false,
		SpectralSteedAttireKeySilverCaria:   false,
		SpectralSteedAttireKeyFunerealNight: false,
	}
	if got := spectralSteedOwnership(state); !reflect.DeepEqual(got, want) {
		t.Errorf("ownership after Lock All = %v, want %v", got, want)
	}

	present, err := engine.GetInventoryGoodsPresence(
		sessionID, getCookbooksSlot, []uint32{spectralSteedUnrelatedID})
	if err != nil {
		t.Fatalf("GetInventoryGoodsPresence: %v", err)
	}
	if !present[spectralSteedUnrelatedID] {
		t.Error("Lock All removed an unrelated Inventory record")
	}
}

func TestLockAllSpectralSteedAttiresRejectsInvalidInput(t *testing.T) {
	engine, sessionID := loadSpectralSteedSession(
		t, []uint32{6701}, spectralSteedInventory{}, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := LockAllSpectralSteedAttires(
		nil, gameCatalog, sessionID, getCookbooksSlot, "0"); err == nil {
		t.Fatal("nil SaveEngine was accepted")
	}
	if _, err := LockAllSpectralSteedAttires(
		engine, nil, sessionID, getCookbooksSlot, "0"); err == nil {
		t.Fatal("nil GameCatalog was accepted")
	}
	if _, err := LockAllSpectralSteedAttires(
		engine, gameCatalog, sessionID, getCookbooksSlot, "4",
	); err == nil || !strings.Contains(err.Error(), "does not match the current saveRevision") {
		t.Fatalf("stale revision error = %v", err)
	}

	state, err := GetSpectralSteedAttires(engine, gameCatalog, sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetSpectralSteedAttires: %v", err)
	}
	if state.ActiveAttireKey != SpectralSteedAttireKeyTreeSentinel {
		t.Errorf("a rejected Lock All changed the active appearance to %q", state.ActiveAttireKey)
	}
}
