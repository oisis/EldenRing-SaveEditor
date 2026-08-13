package saveengine

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	setPhysickInventoryAt        = 505
	setPhysickInventoryRecords   = 0xA80
	setPhysickInventoryRowSize   = 12
	setPhysickInventoryKeyAt     = setPhysickInventoryRecords*setPhysickInventoryRowSize + 4
	setPhysickSlotVersion        = 82
	setPhysickAnchorAt           = 0x10020
	setPhysickTrailingSentinel   = uint32(0xDEADBEEF)
	setPhysickGreenspillTearID   = uint32(0x40002AF9)
	setPhysickCrimsonFlaskTearID = uint32(0x40002AFA)
)

func writeSetPhysickMixtureFixture(
	t *testing.T,
	platform Platform,
	slot int,
	active bool,
	hasFlask bool,
	ownedTears []uint32,
	mixture [2]uint32,
) (string, int64) {
	t.Helper()

	content := physickFixture{
		platform:        platform,
		slot:            slot,
		flag:            1,
		anchorAt:        setPhysickAnchorAt,
		projectileCount: 17,
		tears:           mixture,
		decoyTears:      [2]uint32{0x40002B01, 0x40002B02},
	}
	if !active {
		content.flag = 0
	}
	path := writePhysickFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var slotBase int64
	if platform == PlatformPS4 {
		slotBase = physickPS4SlotDataBase + int64(slot)*physickPS4SlotStride
	} else {
		slotBase = physickPCSlotDataBase + int64(slot)*physickPCSlotStride
	}
	binary.LittleEndian.PutUint32(data[slotBase:], setPhysickSlotVersion)

	inventoryAt := slotBase + setPhysickAnchorAt + setPhysickInventoryAt
	commonCount := uint32(0)
	if hasFlask {
		commonCount = 1
		writeSetPhysickInventoryRow(t, data, inventoryAt, physickFilledFlaskID, 1)
	}
	binary.LittleEndian.PutUint32(data[inventoryAt-4:], commonCount)

	keyAt := inventoryAt + setPhysickInventoryKeyAt
	binary.LittleEndian.PutUint32(data[keyAt-4:], uint32(len(ownedTears)))
	for index, gameID := range ownedTears {
		writeSetPhysickInventoryRow(
			t, data, keyAt+int64(index*setPhysickInventoryRowSize), gameID, uint32(index+2))
	}

	blockAt := slotBase + setPhysickAnchorAt + physickCountAt + 4 +
		int64(content.projectileCount)*8 + physickArmamentsAt
	binary.LittleEndian.PutUint32(data[blockAt+8:], setPhysickTrailingSentinel)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}
	return path, blockAt
}

func writeSetPhysickInventoryRow(
	t *testing.T,
	data []byte,
	at int64,
	gameID uint32,
	acquisitionIndex uint32,
) {
	t.Helper()
	handle, err := gaItemHandleForGameID(gameID)
	if err != nil {
		t.Fatalf("gaItemHandleForGameID(0x%08X): %v", gameID, err)
	}
	binary.LittleEndian.PutUint32(data[at:], handle)
	binary.LittleEndian.PutUint32(data[at+4:], 1)
	binary.LittleEndian.PutUint32(data[at+8:], acquisitionIndex)
}

func TestSetPhysickMixtureWritesBothPlatformsAndReloads(t *testing.T) {
	cases := []struct {
		platform Platform
		slot     int
		tears    [2]uint32
	}{
		{PlatformPC, 0, [2]uint32{PhysickEmptyTearID, setPhysickGreenspillTearID}},
		{PlatformPS4, 6, [2]uint32{setPhysickCrimsonFlaskTearID, setPhysickGreenspillTearID}},
	}

	for _, testCase := range cases {
		t.Run(string(testCase.platform), func(t *testing.T) {
			source, blockAt := writeSetPhysickMixtureFixture(
				t,
				testCase.platform,
				testCase.slot,
				true,
				true,
				[]uint32{setPhysickGreenspillTearID, setPhysickCrimsonFlaskTearID},
				[2]uint32{PhysickEmptyTearID, PhysickEmptyTearID},
			)
			before, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read source before mutation: %v", err)
			}

			engine := New()
			loaded, err := engine.LoadSave(source, string(testCase.platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			result, err := engine.SetPhysickMixture(
				loaded.SaveSessionID, testCase.slot, testCase.tears, "0")
			if err != nil {
				t.Fatalf("SetPhysickMixture: %v", err)
			}
			if result.SaveRevision != "1" || result.Tears != testCase.tears {
				t.Fatalf("result = %+v, want revision 1 and tears %08X/%08X",
					result, testCase.tears[0], testCase.tears[1])
			}

			got, err := engine.GetPhysickMixture(loaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetPhysickMixture after mutation: %v", err)
			}
			if got.Tears != testCase.tears {
				t.Errorf("read-back tears = %08X/%08X, want %08X/%08X",
					got.Tears[0], got.Tears[1], testCase.tears[0], testCase.tears[1])
			}

			afterSource, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read source after mutation: %v", err)
			}
			if !reflect.DeepEqual(afterSource, before) {
				t.Fatal("SetPhysickMixture modified the source file")
			}

			target := filepath.Join(t.TempDir(), "written.sl2")
			written, err := engine.WriteSave(loaded.SaveSessionID, "1", target)
			if err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			if written.SaveRevision != "2" {
				t.Fatalf("WriteSave revision = %q, want 2", written.SaveRevision)
			}

			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(testCase.platform))
			if err != nil {
				t.Fatalf("reload written save: %v", err)
			}
			reloadedMixture, err := reloadedEngine.GetPhysickMixture(
				reloaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetPhysickMixture after reload: %v", err)
			}
			if reloadedMixture.Tears != testCase.tears {
				t.Errorf("reloaded tears = %08X/%08X, want %08X/%08X",
					reloadedMixture.Tears[0], reloadedMixture.Tears[1],
					testCase.tears[0], testCase.tears[1])
			}

			writtenData, err := os.ReadFile(target)
			if err != nil {
				t.Fatalf("read written save: %v", err)
			}
			if trailing := binary.LittleEndian.Uint32(writtenData[blockAt+8:]); trailing != setPhysickTrailingSentinel {
				t.Errorf("trailing Physick field = 0x%08X, want 0x%08X",
					trailing, setPhysickTrailingSentinel)
			}
		})
	}
}

func TestSetPhysickMixtureRejectsInvalidPlansWithoutMutation(t *testing.T) {
	tests := []struct {
		name      string
		hasFlask  bool
		active    bool
		owned     []uint32
		tears     [2]uint32
		revision  string
		wantError string
	}{
		{
			name: "duplicate tear", hasFlask: true, active: true,
			owned:     []uint32{setPhysickGreenspillTearID},
			tears:     [2]uint32{setPhysickGreenspillTearID, setPhysickGreenspillTearID},
			wantError: "cannot occupy both positions", revision: "0",
		},
		{
			name: "unowned tear", hasFlask: true, active: true,
			tears:     [2]uint32{setPhysickGreenspillTearID, PhysickEmptyTearID},
			wantError: "is not owned", revision: "0",
		},
		{
			name: "missing flask", active: true,
			owned:     []uint32{setPhysickGreenspillTearID},
			tears:     [2]uint32{setPhysickGreenspillTearID, PhysickEmptyTearID},
			wantError: "does not own a Flask", revision: "0",
		},
		{
			name: "inactive character", hasFlask: true,
			owned:     []uint32{setPhysickGreenspillTearID},
			tears:     [2]uint32{setPhysickGreenspillTearID, PhysickEmptyTearID},
			wantError: "is not active", revision: "0",
		},
		{
			name: "stale revision", hasFlask: true, active: true,
			owned:     []uint32{setPhysickGreenspillTearID},
			tears:     [2]uint32{setPhysickGreenspillTearID, PhysickEmptyTearID},
			wantError: "does not match", revision: "1",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			source, _ := writeSetPhysickMixtureFixture(
				t, PlatformPC, 0, testCase.active, testCase.hasFlask, testCase.owned,
				[2]uint32{PhysickEmptyTearID, PhysickEmptyTearID})
			engine := New()
			loaded, err := engine.LoadSave(source, "pc")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.SetPhysickMixture(
				loaded.SaveSessionID, 0, testCase.tears, testCase.revision)
			if err == nil || !strings.Contains(err.Error(), testCase.wantError) {
				t.Fatalf("error = %v, want containing %q", err, testCase.wantError)
			}
			if !reflect.DeepEqual(result, SetPhysickMixtureResult{}) {
				t.Errorf("result = %+v, want zero value", result)
			}
			info, infoErr := engine.GetSessionInfo(loaded.SaveSessionID)
			if infoErr != nil {
				t.Fatalf("GetSessionInfo: %v", infoErr)
			}
			if info.UnsavedChanges {
				t.Error("rejected mutation marked the session dirty")
			}
		})
	}
}
