package saveengine

import "testing"

func TestSetWeaponInfusionUsesTheSharedWeaponIDMutation(t *testing.T) {
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
	const targetGameID = setWeaponUpgradeCurrent + 105
	result, err := engine.SetWeaponInfusion(
		loaded.SaveSessionID, setArmamentsSlot, inventory.Records[1].OwnedItemID, "0",
		setWeaponUpgradeCurrent, targetGameID)
	if err != nil {
		t.Fatalf("SetWeaponInfusion: %v", err)
	}
	if result.SaveRevision != "1" || result.Container != "inventory" ||
		result.PreviousGameID != setWeaponUpgradeCurrent || result.GameID != targetGameID {
		t.Fatalf("result = %+v", result)
	}

	gameIDs, err := engine.ResolveGaItemIDs(
		loaded.SaveSessionID, setArmamentsSlot, []uint32{setWeaponUpgradeHandle})
	if err != nil || gameIDs[0] != targetGameID {
		t.Fatalf("resolved game ID = %v, err=%v", gameIDs, err)
	}
}
