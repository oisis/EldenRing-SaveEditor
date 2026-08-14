package inventory

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSetWeaponInfusionPreservesUpgradeLevel(t *testing.T) {
	wantVariables := []string{
		"saveSessionID", "characterID", "ownedItemID", "affinity", "expectedRevision",
	}
	if !reflect.DeepEqual(SetWeaponInfusionDefinition.SupportedResourceVariables, wantVariables) {
		t.Fatalf("variables = %#v, want %#v",
			SetWeaponInfusionDefinition.SupportedResourceVariables, wantVariables)
	}

	const currentGameID = setWeaponEndpointDaggerID + 5
	engine, sessionID, token := setWeaponUpgradeEndpointTarget(t, currentGameID)
	result, err := SetWeaponInfusion(
		engine, inventoryCatalog(t), sessionID, 0, token, schema.AffinityHeavy, "0")
	if err != nil {
		t.Fatalf("SetWeaponInfusion: %v", err)
	}
	const wantGameID = setWeaponEndpointDaggerID + 100 + 5
	if result.SaveRevision != "1" || result.PreviousGameID != currentGameID ||
		result.GameID != wantGameID || result.Affinity != schema.AffinityHeavy ||
		result.UpgradeLevel != 5 {
		t.Fatalf("result = %+v", result)
	}

	listed, err := GetInventory(engine, inventoryCatalog(t), sessionID, 0, "common", 1, 50)
	if err != nil || listed.Records[1].GameID != wantGameID {
		t.Fatalf("GetInventory after infusion: %v, records=%+v", err, listed.Records)
	}
}

func TestSetWeaponInfusionRejectsUnsupportedAffinityWithoutMutation(t *testing.T) {
	engine, sessionID, token := setWeaponUpgradeEndpointTarget(t, setWeaponEndpointBlackKnife)
	if _, err := SetWeaponInfusion(
		engine, inventoryCatalog(t), sessionID, 0, token, schema.AffinityHeavy, "0"); err == nil {
		t.Fatal("somber weapon infusion succeeded")
	}
	info, err := engine.GetSessionInfo(sessionID)
	if err != nil || info.UnsavedChanges {
		t.Fatalf("rejected mutation changed session: %+v, %v", info, err)
	}
}
