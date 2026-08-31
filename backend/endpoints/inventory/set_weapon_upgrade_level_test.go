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
	setWeaponEndpointFixtureSize = int64(0x300) + 10*0x280010 + 0x60010
	setWeaponEndpointSlotBase    = 0x310
	setWeaponEndpointAnchorAt    = 0xA07B
	setWeaponEndpointInventoryAt = setWeaponEndpointSlotBase + setWeaponEndpointAnchorAt + 505
	setWeaponEndpointDaggerID    = uint32(1000000)
	setWeaponEndpointBlackKnife  = uint32(1010000)
)

var setWeaponEndpointAnchor = []byte{
	0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

func writeSetWeaponUpgradeEndpointFixture(t *testing.T, gameID uint32) string {
	t.Helper()
	data := make([]byte, setWeaponEndpointFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[0x0C:], 12)
	data[int64(0x300)+10*0x280010+0x10+0x1954] = 1
	binary.LittleEndian.PutUint32(data[setWeaponEndpointSlotBase:], 83)
	for index := 0; index < 7; index++ {
		handle := uint32(0x80000100 + index)
		at := int64(setWeaponEndpointSlotBase + 0x20 + index*21)
		binary.LittleEndian.PutUint32(data[at:], handle)
		binary.LittleEndian.PutUint32(data[at+4:], gameID)
		rowAt := int64(setWeaponEndpointInventoryAt + index*12)
		binary.LittleEndian.PutUint32(data[rowAt:], handle)
		binary.LittleEndian.PutUint32(data[rowAt+4:], 1)
		binary.LittleEndian.PutUint32(data[rowAt+8:], uint32(index+1))
	}
	copy(data[setWeaponEndpointSlotBase+setWeaponEndpointAnchorAt:], setWeaponEndpointAnchor)

	path := filepath.Join(t.TempDir(), "set-weapon-upgrade.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func setWeaponUpgradeEndpointTarget(
	t *testing.T,
	gameID uint32,
) (*saveengine.Engine, string, string) {
	t.Helper()
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSetWeaponUpgradeEndpointFixture(t, gameID), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	listed, err := engine.GetInventory(
		loaded.SaveSessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil || len(listed.Records) < 2 {
		t.Fatalf("GetInventory: %v, len=%d", err, len(listed.Records))
	}
	return engine, loaded.SaveSessionID, listed.Records[1].OwnedItemID
}

func TestSetWeaponUpgradeLevelDefinitionAndCatalogLimits(t *testing.T) {
	want := []string{"saveSessionID", "characterID", "ownedItemID", "upgradeLevel", "expectedRevision"}
	if !reflect.DeepEqual(SetWeaponUpgradeLevelDefinition.SupportedResourceVariables, want) {
		t.Fatalf("variables = %#v, want %#v",
			SetWeaponUpgradeLevelDefinition.SupportedResourceVariables, want)
	}
	catalog := inventoryCatalog(t)
	for _, testCase := range []struct {
		name    string
		gameID  uint32
		level   uint8
		target  uint32
		invalid uint8
	}{
		{"standard", setWeaponEndpointDaggerID, 25, setWeaponEndpointDaggerID + 25, 26},
		{"somber", setWeaponEndpointBlackKnife, 10, setWeaponEndpointBlackKnife + 10, 11},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			engine, sessionID, token := setWeaponUpgradeEndpointTarget(t, testCase.gameID)
			result, err := SetWeaponUpgradeLevel(
				engine, catalog, sessionID, 0, token, testCase.level, "0")
			if err != nil {
				t.Fatalf("SetWeaponUpgradeLevel: %v", err)
			}
			if result.SaveRevision != "1" || result.GameID != testCase.target ||
				result.PreviousGameID != testCase.gameID || result.UpgradeLevel != testCase.level {
				t.Fatalf("result = %+v", result)
			}
			listed, err := GetInventory(engine, catalog, sessionID, 0, "common", 1, 50)
			if err != nil || listed.Records[1].GameID != testCase.target {
				t.Fatalf("GetInventory after upgrade: %v, records=%+v", err, listed.Records)
			}

			// Verify durable matchmaking weapon level byte is updated to 25
			matchmakingAt := int64(setWeaponEndpointSlotBase + setWeaponEndpointAnchorAt - 0xD5)
			targetFile := filepath.Join(t.TempDir(), "upgrade-persisted.sl2")
			if _, err := engine.WriteSave(sessionID, result.SaveRevision, targetFile); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := saveengine.New()
			reloadedSession, err := reloadedEngine.LoadSave(targetFile, "pc", "local")
			if err != nil {
				t.Fatalf("LoadSave reloaded: %v", err)
			}
			info, err := reloadedEngine.GetSessionInfo(reloadedSession.SaveSessionID)
			if err != nil {
				t.Fatalf("GetSessionInfo: %v", err)
			}
			if info.Platform != "pc" {
				t.Fatalf("platform = %q, want pc", info.Platform)
			}
			persistedData, err := os.ReadFile(targetFile)
			if err != nil {
				t.Fatalf("ReadFile target: %v", err)
			}
			if got := persistedData[matchmakingAt]; got != 25 {
				t.Errorf("%s upgrade: matchmaking level in persisted file = %d, want 25", testCase.name, got)
			}

			other, otherSessionID, otherToken := setWeaponUpgradeEndpointTarget(t, testCase.gameID)
			if _, err := SetWeaponUpgradeLevel(
				other, catalog, otherSessionID, 0, otherToken, testCase.invalid, "0"); err == nil {
				t.Fatalf("upgrade level %d succeeded", testCase.invalid)
			}
			otherInfo, err := other.GetSessionInfo(otherSessionID)
			if err != nil || otherInfo.UnsavedChanges {
				t.Fatalf("rejected mutation changed session: %+v, %v", otherInfo, err)
			}
		})
	}
}

func TestSetWeaponUpgradeLevelMatchmakingMonotonicityAndDowngrade(t *testing.T) {
	catalog := inventoryCatalog(t)
	engine, sessionID, token := setWeaponUpgradeEndpointTarget(t, setWeaponEndpointDaggerID)

	// Step 1: Upgrade to +25 -> matchmaking byte becomes 25
	res1, err := SetWeaponUpgradeLevel(
		engine, catalog, sessionID, 0, token, 25, "0")
	if err != nil {
		t.Fatalf("SetWeaponUpgradeLevel +25: %v", err)
	}

	matchmakingAt := int64(setWeaponEndpointSlotBase + setWeaponEndpointAnchorAt - 0xD5)
	targetFile1 := filepath.Join(t.TempDir(), "step1.sl2")
	writeRes1, err := engine.WriteSave(sessionID, res1.SaveRevision, targetFile1)
	if err != nil {
		t.Fatalf("WriteSave step 1: %v", err)
	}
	data1, err := os.ReadFile(targetFile1)
	if err != nil {
		t.Fatalf("ReadFile step 1: %v", err)
	}
	if got := data1[matchmakingAt]; got != 25 {
		t.Fatalf("matchmaking level after +25 = %d, want 25", got)
	}

	// Step 2: Downgrade to +0 -> weapon changes to +0, but matchmaking byte REMAINS 25 (monotonic)
	listed, err := GetInventory(engine, catalog, sessionID, 0, "common", 1, 50)
	if err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	token2 := listed.Records[1].OwnedItemID
	res2, err := SetWeaponUpgradeLevel(
		engine, catalog, sessionID, 0, token2, 0, writeRes1.SaveRevision)
	if err != nil {
		t.Fatalf("SetWeaponUpgradeLevel +0: %v", err)
	}
	if res2.GameID != setWeaponEndpointDaggerID || res2.UpgradeLevel != 0 {
		t.Fatalf("downgraded result = %+v", res2)
	}

	targetFile2 := filepath.Join(t.TempDir(), "step2.sl2")
	if _, err := engine.WriteSave(sessionID, res2.SaveRevision, targetFile2); err != nil {
		t.Fatalf("WriteSave step 2: %v", err)
	}
	data2, err := os.ReadFile(targetFile2)
	if err != nil {
		t.Fatalf("ReadFile step 2: %v", err)
	}
	if got := data2[matchmakingAt]; got != 25 {
		t.Fatalf("matchmaking level after downgrade to +0 = %d, want 25 (monotonic)", got)
	}
}

func TestSetWeaponUpgradeLevelRejectsSpiritAshInWeaponEndpoint(t *testing.T) {
	catalog := inventoryCatalog(t)
	const spiritAshGameID = uint32(0x40038A40)
	engine, sessionID, token := setWeaponUpgradeEndpointTarget(t, spiritAshGameID)

	if _, err := SetWeaponUpgradeLevel(
		engine, catalog, sessionID, 0, token, 10, "0"); err == nil {
		t.Fatal("SetWeaponUpgradeLevel for Spirit Ash succeeded, want error")
	}
}
