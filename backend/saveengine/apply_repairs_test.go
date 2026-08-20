package saveengine

import (
	"bytes"
	"testing"
)

func TestApplyRepairPlan_PreflightFailureLeavesTheWholeSnapshotUntouched(t *testing.T) {
	fixture := writeApplyTemplateFixture(t, PlatformPC, false)
	engine := New()
	session, err := engine.LoadSave(fixture, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[session.SaveSessionID].snapshot.data)
	attributes := CharacterAttributes{Vigor: 16, Mind: 10, Endurance: 11, Strength: 14, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7}

	_, err = engine.ApplyRepairPlan(session.SaveSessionID, applyTplTestSlot, []RepairAction{
		{Operation: RepairOperationSetCharacterStats, Attributes: &attributes},
		{Operation: "unsupported"},
	}, "0")
	if err == nil {
		t.Fatal("ApplyRepairPlan accepted an unsupported later action")
	}
	if !bytes.Equal(before, engine.sessions[session.SaveSessionID].snapshot.data) {
		t.Error("a preflight failure changed snapshot bytes before the transaction")
	}
	info, err := engine.GetSessionInfo(session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Error("a preflight failure marked the session dirty")
	}
}
