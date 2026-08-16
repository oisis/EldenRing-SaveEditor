package saveengine

import (
	"encoding/binary"
	"os"
	"reflect"
	"strings"
	"testing"
)

func writeTutorialDataFixture(
	t *testing.T, platform Platform, active bool, size uint32, ids []uint32,
) string {
	t.Helper()

	content := eventFlagTestFixture{
		platform: platform,
		slot:     3,
		flag:     1,
		anchorAt: 0x1A7,
		menuSize: 0,
		regions:  0,
		set:      nil,
	}
	if !active {
		content.flag = 0
		content.noAnchor = true
	}
	content.tutorialSize = size
	path := writeEventFlagFixture(t, content)
	if !active {
		return path
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var slotBase int64
	switch platform {
	case PlatformPC:
		slotBase = eventFlagTestPCSlotDataBase + int64(content.slot)*eventFlagTestPCSlotStride
	case PlatformPS4:
		slotBase = eventFlagTestPS4SlotDataBase + int64(content.slot)*eventFlagTestPS4SlotStride
	}
	tutorialAt := slotBase + content.anchorAt + eventFlagTestTutorialHeaderAt
	binary.LittleEndian.PutUint32(data[tutorialAt+8:], uint32(len(ids)))
	for index, id := range ids {
		binary.LittleEndian.PutUint32(data[tutorialAt+12+int64(index*4):], id)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetTutorialIDsReadsBothPlatformsAndUsesDeclaredSize(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeTutorialDataFixture(t, platform, true, 0x24, []uint32{2010, 1590}),
				string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			result, err := engine.GetTutorialIDs(loaded.SaveSessionID, 3)
			if err != nil {
				t.Fatalf("GetTutorialIDs: %v", err)
			}
			if !result.Active || !reflect.DeepEqual(result.IDs, []uint32{2010, 1590}) {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestGetTutorialIDsDoesNotReadResidualSlot(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeTutorialDataFixture(t, PlatformPC, false, 0, nil), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	result, err := engine.GetTutorialIDs(loaded.SaveSessionID, 3)
	if err != nil {
		t.Fatalf("GetTutorialIDs: %v", err)
	}
	if result.Active || result.IDs == nil || len(result.IDs) != 0 {
		t.Fatalf("residual result = %+v, want inactive with an empty list", result)
	}
}

func TestGetTutorialIDsRejectsCountOutsideTheDeclaredPayload(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeTutorialDataFixture(t, PlatformPC, true, 8, []uint32{2010, 2020}), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = engine.GetTutorialIDs(loaded.SaveSessionID, 3)
	if err == nil || !strings.Contains(err.Error(), "tutorial count") {
		t.Fatalf("malformed count error = %v", err)
	}
}

func TestGetTutorialIDsRejectsAPayloadTooSmallForTheCount(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeTutorialDataFixture(t, PlatformPC, true, tutorialDataCountSize-1, nil), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	_, err = engine.GetTutorialIDs(loaded.SaveSessionID, 3)
	if err == nil ||
		!strings.Contains(err.Error(), "does not hold the 4-byte tutorial count field") {
		t.Fatalf("undersized payload error = %v", err)
	}
}
