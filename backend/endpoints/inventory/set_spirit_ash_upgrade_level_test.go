package inventory

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	setSpiritAshEndpointCurrent  = uint32(0x40038A44)
	setSpiritAshEndpointTargetID = uint32(0x40038A4A)
)

func setSpiritAshEndpointTarget(t *testing.T) (*saveengine.Engine, string, string) {
	t.Helper()
	return setSpiritAshEndpointTargetWithMatchmaking(t, 0)
}

func setSpiritAshEndpointTargetWithMatchmaking(t *testing.T, initialMatchmaking uint8) (*saveengine.Engine, string, string) {
	t.Helper()
	path := writeSetWeaponUpgradeEndpointFixture(t, setWeaponEndpointDaggerID)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	binary.LittleEndian.PutUint32(
		data[setWeaponEndpointInventoryAt+12:],
		0xB0038A44)
	matchmakingAt := int64(setWeaponEndpointSlotBase + setWeaponEndpointAnchorAt - 0xD5)
	data[matchmakingAt] = initialMatchmaking
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	inventory, err := engine.GetInventory(
		loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(inventory.Records) < 2 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(inventory.Records))
	}
	return engine, loaded.SaveSessionID, inventory.Records[1].OwnedItemID
}

func TestSetSpiritAshUpgradeLevelDefinitionAndCatalogLimit(t *testing.T) {
	want := []string{"saveSessionID", "characterID", "ownedItemID", "upgradeLevel", "expectedRevision"}
	if !reflect.DeepEqual(SetSpiritAshUpgradeLevelDefinition.SupportedResourceVariables, want) {
		t.Fatalf("variables = %#v, want %#v",
			SetSpiritAshUpgradeLevelDefinition.SupportedResourceVariables, want)
	}
	catalog := inventoryCatalog(t)
	engine, sessionID, token := setSpiritAshEndpointTarget(t)
	result, err := SetSpiritAshUpgradeLevel(
		engine, catalog, sessionID, 0, token, 10, "0")
	if err != nil {
		t.Fatalf("SetSpiritAshUpgradeLevel: %v", err)
	}
	if result.SaveRevision != "1" || result.PreviousGameID != setSpiritAshEndpointCurrent ||
		result.GameID != setSpiritAshEndpointTargetID || result.UpgradeLevel != 10 {
		t.Fatalf("result = %+v", result)
	}

	rejected, rejectedSessionID, rejectedToken := setSpiritAshEndpointTarget(t)
	if _, err := SetSpiritAshUpgradeLevel(
		rejected, catalog, rejectedSessionID, 0, rejectedToken, 11, "0"); err == nil {
		t.Fatal("upgrade level 11 succeeded")
	}
	info, err := rejected.GetSessionInfo(rejectedSessionID)
	if err != nil || info.UnsavedChanges {
		t.Fatalf("rejected mutation changed session: %+v, %v", info, err)
	}
}

func TestSetSpiritAshUpgradeLevelPreservesMatchmakingLevel(t *testing.T) {
	catalog := inventoryCatalog(t)
	const initialMatchmaking = uint8(4)
	engine, sessionID, token := setSpiritAshEndpointTargetWithMatchmaking(t, initialMatchmaking)

	result, err := SetSpiritAshUpgradeLevel(
		engine, catalog, sessionID, 0, token, 10, "0")
	if err != nil {
		t.Fatalf("SetSpiritAshUpgradeLevel +10: %v", err)
	}
	if result.UpgradeLevel != 10 || result.GameID != setSpiritAshEndpointTargetID {
		t.Fatalf("result = %+v", result)
	}

	matchmakingAt := int64(setWeaponEndpointSlotBase + setWeaponEndpointAnchorAt - 0xD5)
	targetFile := filepath.Join(t.TempDir(), "spirit-ash-upgraded.sl2")
	if _, err := engine.WriteSave(sessionID, result.SaveRevision, targetFile); err != nil {
		t.Fatalf("WriteSave: %v", err)
	}
	savedData, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got := savedData[matchmakingAt]; got != initialMatchmaking {
		t.Errorf("matchmaking level after Spirit Ash +10 = %d, want preserved %d", got, initialMatchmaking)
	}
}
