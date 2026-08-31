package network

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestSetNetworkSettingsDelegatesTheCompleteAssignment(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeSettingsFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	settings := validSettingsForSet()

	result, err := SetNetworkSettings(engine, loaded.SaveSessionID, settings, "0")
	if err != nil {
		t.Fatalf("SetNetworkSettings: %v", err)
	}
	if result.SaveSessionID != loaded.SaveSessionID || result.SaveRevision != "1" ||
		result.NetworkSettings != settings {
		t.Fatalf("result = %+v", result)
	}
	stored, err := engine.GetNetworkSettings(loaded.SaveSessionID)
	if err != nil || stored != settings {
		t.Fatalf("stored settings = %+v, err = %v", stored, err)
	}
}

func TestSetNetworkSettingsRejectsAMissingEngine(t *testing.T) {
	if _, err := SetNetworkSettings(nil, "session", validSettingsForSet(), "0"); err == nil {
		t.Fatal("a missing engine was accepted")
	}
}

func validSettingsForSet() gamecatalog.NetworkParamValues {
	return gamecatalog.NetworkParamValues{
		MaxBreakInTargetListCount:     8,
		BreakInRequestIntervalTimeSec: 12,
		BreakInRequestTimeOutSec:      8,
		BreakInRequestAreaCount:       8,
		SummonTimeoutTime:             45,
		ReloadSignIntervalTime2:       20,
		ReloadSignTotalCount:          40,
		ReloadSignCellCount:           20,
		UpdateSignIntervalTime:        15,
		SingGetMax:                    64,
		SignDownloadSpan:              15,
		SignUpdateSpan:                20,
		ReloadVisitListCoolTime:       8,
		MaxCoopBlueSummonCount:        2,
		MaxVisitListCount:             10,
		ReloadSearchCoopBlueMin:       10,
		ReloadSearchCoopBlueMax:       40,
		AllAreaSearchRateCoopBlue:     60,
		AllAreaSearchRateVsBlue:       30,
		VisitorListMax:                10,
		VisitorTimeOutTime:            60,
		VisitorDownloadSpan:           60,
	}
}
