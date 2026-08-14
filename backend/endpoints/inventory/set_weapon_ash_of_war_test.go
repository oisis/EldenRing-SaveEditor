package inventory

import (
	"encoding/binary"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	setWeaponAoWEndpointGameID = uint32(0x8000EA60)
	setWeaponAoWEndpointHandle = uint32(0xC0000200)
	setWeaponAoWEndpointKind   = "item"
	setWeaponAoWEndpointKey    = "8000EA60"
)

func writeSetWeaponAshOfWarEndpointFixture(t *testing.T) string {
	t.Helper()
	path := writeSetWeaponUpgradeEndpointFixture(t, setWeaponEndpointDaggerID)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	aoWAt := int64(setWeaponEndpointSlotBase + 0x20 + 7*21)
	binary.LittleEndian.PutUint32(data[aoWAt:], setWeaponAoWEndpointHandle)
	binary.LittleEndian.PutUint32(data[aoWAt+4:], setWeaponAoWEndpointGameID)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func setWeaponAshOfWarEndpointTarget(
	t *testing.T,
) (*saveengine.Engine, string, string) {
	t.Helper()
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSetWeaponAshOfWarEndpointFixture(t), "pc")
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

func TestSetWeaponAshOfWarDefinitionAttachAndRemove(t *testing.T) {
	wantVariables := []string{
		"saveSessionID", "characterID", "weaponOwnedItemID",
		"ashOfWarKind", "ashOfWarKey", "expectedRevision",
	}
	if !reflect.DeepEqual(SetWeaponAshOfWarDefinition.SupportedResourceVariables, wantVariables) {
		t.Fatalf("variables = %#v, want %#v",
			SetWeaponAshOfWarDefinition.SupportedResourceVariables, wantVariables)
	}

	engine, sessionID, token := setWeaponAshOfWarEndpointTarget(t)
	kind, key := setWeaponAoWEndpointKind, setWeaponAoWEndpointKey
	attached, err := SetWeaponAshOfWar(
		engine, inventoryCatalog(t), sessionID, 0, token, &kind, &key, "0")
	if err != nil {
		t.Fatalf("attach SetWeaponAshOfWar: %v", err)
	}
	if attached.SaveRevision != "1" || attached.WeaponGameID != setWeaponEndpointDaggerID ||
		attached.PreviousAshOfWarGameID != 0 ||
		attached.AshOfWarGameID != setWeaponAoWEndpointGameID {
		t.Fatalf("attached result = %+v", attached)
	}

	inventory, err := engine.GetInventory(
		sessionID, 0, saveengine.InventorySectionCommon, 1, 50)
	if err != nil {
		t.Fatalf("GetInventory after attach: %v", err)
	}
	removed, err := SetWeaponAshOfWar(
		engine, inventoryCatalog(t), sessionID, 0,
		inventory.Records[1].OwnedItemID, nil, nil, "1")
	if err != nil {
		t.Fatalf("remove SetWeaponAshOfWar: %v", err)
	}
	if removed.SaveRevision != "2" ||
		removed.PreviousAshOfWarGameID != setWeaponAoWEndpointGameID ||
		removed.AshOfWarGameID != 0 {
		t.Fatalf("removed result = %+v", removed)
	}
}

func TestSetWeaponAshOfWarRejectsInvalidResourceSelectionWithoutMutation(t *testing.T) {
	itemKind, daggerKey := "item", "000F4240"
	for _, testCase := range []struct {
		name string
		kind *string
		key  *string
		want string
	}{
		{
			name: "half of optional pair",
			kind: &itemKind,
			want: "must either both",
		},
		{
			name: "non Ash resource",
			kind: &itemKind,
			key:  &daggerKey,
			want: "not a confirmed Ash of War",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			engine, sessionID, token := setWeaponAshOfWarEndpointTarget(t)
			_, err := SetWeaponAshOfWar(
				engine, inventoryCatalog(t), sessionID, 0, token,
				testCase.kind, testCase.key, "0")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			info, infoErr := engine.GetSessionInfo(sessionID)
			if infoErr != nil || info.UnsavedChanges {
				t.Fatalf("rejected mutation changed session: %+v, %v", info, infoErr)
			}
		})
	}
}

func TestSetWeaponAshOfWarRemovalDoesNotRequireMountCapability(t *testing.T) {
	manifest, resources := prototype.Data()
	for index := range resources {
		item := resources[index].Item
		if item == nil || item.GameID.Value != setWeaponEndpointDaggerID {
			continue
		}
		item.Capabilities.AshOfWarMount.Enabled = false
		item.Capabilities.AshOfWarMount.Rules = nil
		item.Capabilities.AshOfWarMount.RulesEvidence = nil
	}
	catalog, err := gamecatalog.New(manifest, resources)
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	engine, sessionID, token := setWeaponAshOfWarEndpointTarget(t)
	result, err := SetWeaponAshOfWar(
		engine, catalog, sessionID, 0, token, nil, nil, "0")
	if err != nil {
		t.Fatalf("remove SetWeaponAshOfWar: %v", err)
	}
	if result.SaveRevision != "1" || result.AshOfWarGameID != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestSetWeaponAshOfWarRequiresDependencies(t *testing.T) {
	engine, sessionID, token := setWeaponAshOfWarEndpointTarget(t)
	kind, key := setWeaponAoWEndpointKind, setWeaponAoWEndpointKey
	if _, err := SetWeaponAshOfWar(
		nil, inventoryCatalog(t), sessionID, 0, token, &kind, &key, "0"); err == nil {
		t.Fatal("missing SaveEngine was accepted")
	}
	if _, err := SetWeaponAshOfWar(
		engine, nil, sessionID, 0, token, &kind, &key, "0"); err == nil {
		t.Fatal("missing GameCatalog was accepted")
	}
}
