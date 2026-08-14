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
	loaded, err := engine.LoadSave(writeSetWeaponUpgradeEndpointFixture(t, gameID), "pc")
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

			other, otherSessionID, otherToken := setWeaponUpgradeEndpointTarget(t, testCase.gameID)
			if _, err := SetWeaponUpgradeLevel(
				other, catalog, otherSessionID, 0, otherToken, testCase.invalid, "0"); err == nil {
				t.Fatalf("upgrade level %d succeeded", testCase.invalid)
			}
			info, err := other.GetSessionInfo(otherSessionID)
			if err != nil || info.UnsavedChanges {
				t.Fatalf("rejected mutation changed session: %+v, %v", info, err)
			}
		})
	}
}
