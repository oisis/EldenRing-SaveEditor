package saveengine

import (
	"encoding/binary"
	"os"
	"reflect"
	"testing"
)

const (
	regionsTestSectionSize = 0x100
	regionsTestRecordSize  = 4
	regionsTestMaxCount    = 20000
)

func regionsTestCountAt(content gestureTestFixture) int64 {
	return content.anchorAt + gestureTestProjectileCountAt + 4 +
		int64(content.projectiles)*gestureTestProjectileStride +
		gestureTestBlocksBefore + gestureTestStorageBoxSize + regionsTestSectionSize
}

func writeRegionsFixture(t *testing.T, content gestureTestFixture, ids []uint32, count uint32) string {
	t.Helper()
	path := writeGestureFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var slotBase int64
	if content.platform == PlatformPS4 {
		slotBase = gestureTestPS4SlotDataBase + int64(content.slot)*gestureTestPS4SlotStride
	} else {
		slotBase = gestureTestPCSlotDataBase + int64(content.slot)*gestureTestPCSlotStride
	}
	countAt := regionsTestCountAt(content)
	if countAt+4 <= gestureTestSlotDataSize {
		binary.LittleEndian.PutUint32(data[slotBase+countAt:], count)
	}
	for index, id := range ids {
		at := countAt + 4 + int64(index)*regionsTestRecordSize
		if at+regionsTestRecordSize <= gestureTestSlotDataSize {
			binary.LittleEndian.PutUint32(data[slotBase+at:], id)
		}
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetRegionsReadsBothPlatformsWithoutNormalising(t *testing.T) {
	wantIDs := []uint32{6100000, 9999999, 6100000, 0}
	cases := []gestureTestFixture{
		{platform: PlatformPC, slot: 0, flag: 1, anchorAt: 0x01A7},
		{platform: PlatformPS4, slot: 7, flag: 1, anchorAt: 0x1F4C2, projectiles: 37},
	}
	for _, content := range cases {
		t.Run(string(content.platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeRegionsFixture(t, content, wantIDs, uint32(len(wantIDs))),
				string(content.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			got, err := engine.GetRegions(loaded.SaveSessionID, content.slot)
			if err != nil {
				t.Fatalf("GetRegions: %v", err)
			}
			want := CharacterRegions{
				SaveSessionID: loaded.SaveSessionID,
				CharacterID:   content.slot,
				Active:        true,
				RegionIDs:     wantIDs,
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("result = %+v, want %+v", got, want)
			}
		})
	}
}

func TestGetRegionsDoesNotReadResidualSlot(t *testing.T) {
	content := gestureTestFixture{platform: PlatformPC, slot: 4, noAnchor: true}
	engine := New()
	loaded, err := engine.LoadSave(
		writeRegionsFixture(t, content, []uint32{6100000}, 1), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	got, err := engine.GetRegions(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetRegions: %v", err)
	}
	want := CharacterRegions{
		SaveSessionID: loaded.SaveSessionID,
		CharacterID:   content.slot,
		RegionIDs:     []uint32{},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result = %+v, want %+v", got, want)
	}
}

func TestGetRegionsRejectsInvalidRequestsAndLayout(t *testing.T) {
	engine := New()
	load := func(content gestureTestFixture, count uint32) string {
		t.Helper()
		loaded, err := engine.LoadSave(writeRegionsFixture(t, content, nil, count), "", "local")
		if err != nil {
			t.Fatalf("LoadSave: %v", err)
		}
		return loaded.SaveSessionID
	}
	present := load(gestureTestFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x640,
	}, 0)
	missingAnchor := load(gestureTestFixture{
		platform: PlatformPC, slot: 2, flag: 1, noAnchor: true,
	}, 0)
	corruptCount := load(gestureTestFixture{
		platform: PlatformPC, slot: 2, flag: 1, anchorAt: 0x640,
	}, regionsTestMaxCount+1)
	distance := regionsTestCountAt(gestureTestFixture{})
	outOfSlot := load(gestureTestFixture{
		platform: PlatformPC, slot: 2, flag: 1,
		anchorAt: gestureTestSlotDataSize - distance - 4,
	}, 1)

	cases := []struct {
		name        string
		sessionID   string
		characterID int
		wantError   string
	}{
		{"empty session", "", 0, "saveSessionID is required"},
		{"unknown session", "missing", 0, `unknown save session "missing"`},
		{"negative character", present, -1, "characterID -1 is outside the range 0..9"},
		{"large character", present, 10, "characterID 10 is outside the range 0..9"},
		{"missing anchor", missingAnchor, 2, "character 2 carries no gesture anchor"},
		{"corrupt count", corruptCount, 2,
			"character 2 declares 20001 unlocked regions, want at most 20000"},
		{"list outside slot", outOfSlot, 2,
			"unlocked regions of character 2 do not fit into their slot"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := engine.GetRegions(testCase.sessionID, testCase.characterID)
			if err == nil || err.Error() != testCase.wantError {
				t.Fatalf("error = %v, want %q", err, testCase.wantError)
			}
			if !reflect.DeepEqual(got, CharacterRegions{}) {
				t.Fatalf("result = %+v, want zero value", got)
			}
		})
	}
}
