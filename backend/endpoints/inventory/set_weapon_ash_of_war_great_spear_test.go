package inventory

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	// greatSpearLanceGameID and greatSpearMessmerGameID are EquipParamWeapon
	// wepType 28 weapons that accept a custom Ash of War.
	greatSpearLanceGameID   = uint32(0x010450A0)
	greatSpearMessmerGameID = uint32(0x010B2E70)
	// chillingMistGameID has canMountWep_SpearHeavy (bit 17) set and
	// canMountWep_SpearLarge (bit 16) clear, so it is only reachable through
	// the corrected Great Spear compatibility bit.
	chillingMistGameID = uint32(0x800058AC)
	chillingMistKey    = "800058AC"
	chillingMistHandle = uint32(0xC0000200)
)

// greatSpearAshOfWarTarget builds a save whose common weapon is the requested
// Great Spear and whose eighth GaItem slot holds a free Chilling Mist copy.
func greatSpearAshOfWarTarget(
	t *testing.T,
	weaponGameID uint32,
) (*saveengine.Engine, string, string) {
	t.Helper()
	path := writeSetWeaponUpgradeEndpointFixture(t, weaponGameID)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	ashAt := int64(setWeaponEndpointSlotBase + 0x20 + 7*21)
	binary.LittleEndian.PutUint32(data[ashAt:], chillingMistHandle)
	binary.LittleEndian.PutUint32(data[ashAt+4:], chillingMistGameID)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, "pc", "local")
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

// TestSetWeaponAshOfWarAcceptsChillingMistOnGreatSpears is the regression for
// the wepType 28 compatibility bit. With the wrong bit 16 the catalog derives no
// compatible_with_aow relation and the endpoint rejects the mount before it ever
// reaches SaveEngine.
func TestSetWeaponAshOfWarAcceptsChillingMistOnGreatSpears(t *testing.T) {
	catalog := inventoryCatalog(t)
	for name, weaponGameID := range map[string]uint32{
		"Lance":                   greatSpearLanceGameID,
		"Messmer Soldier's Spear": greatSpearMessmerGameID,
	} {
		t.Run(name, func(t *testing.T) {
			engine, sessionID, token := greatSpearAshOfWarTarget(t, weaponGameID)
			kind, key := "item", chillingMistKey

			attached, err := SetWeaponAshOfWar(
				engine, catalog, sessionID, 0, token, &kind, &key, "0")
			if err != nil {
				t.Fatalf("attach Chilling Mist to 0x%08X: %v", weaponGameID, err)
			}
			if attached.AshOfWarGameID != chillingMistGameID {
				t.Fatalf("attached Ash of War = 0x%08X, want 0x%08X",
					attached.AshOfWarGameID, chillingMistGameID)
			}

			listed, err := engine.GetInventory(
				sessionID, 0, saveengine.InventorySectionCommon, 1, 50)
			if err != nil {
				t.Fatalf("GetInventory after attach: %v", err)
			}
			removed, err := SetWeaponAshOfWar(
				engine, catalog, sessionID, 0,
				listed.Records[1].OwnedItemID, nil, nil, attached.SaveRevision)
			if err != nil {
				t.Fatalf("remove Ash of War from 0x%08X: %v", weaponGameID, err)
			}
			if removed.PreviousAshOfWarGameID != chillingMistGameID ||
				removed.AshOfWarGameID != 0 {
				t.Fatalf("removal receipt = %+v", removed)
			}
		})
	}
}
