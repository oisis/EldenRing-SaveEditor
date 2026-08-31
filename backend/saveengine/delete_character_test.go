package saveengine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	deleteTestSlot            = 3
	deleteTestSlotSize        = 0x280000
	deleteTestPCSlotBase      = 0x310
	deleteTestPCSlotStride    = 0x280010
	deleteTestPS4SlotBase     = 0x70
	deleteTestPS4SlotStride   = 0x280000
	deleteTestFlagsOffset     = 0x1954
	deleteTestSummaryOffset   = 0x195E
	deleteTestSummaryStride   = 0x24C
	deleteTestPCUserDataBase  = 0x19003B0
	deleteTestPS4UserDataBase = 0x1900070
)

func deleteTestRanges(platform Platform) (int64, int64, int64) {
	if platform == PlatformPS4 {
		return deleteTestPS4SlotBase + deleteTestSlot*deleteTestPS4SlotStride,
			deleteTestPS4UserDataBase + deleteTestFlagsOffset + deleteTestSlot,
			deleteTestPS4UserDataBase + deleteTestSummaryOffset +
				deleteTestSlot*deleteTestSummaryStride
	}
	return deleteTestPCSlotBase + deleteTestSlot*deleteTestPCSlotStride,
		deleteTestPCUserDataBase + deleteTestFlagsOffset + deleteTestSlot,
		deleteTestPCUserDataBase + deleteTestSummaryOffset +
			deleteTestSlot*deleteTestSummaryStride
}

func deleteTestAllZero(data []byte) bool {
	return bytes.Count(data, []byte{0}) == len(data)
}

func TestDeleteCharacterClearsOnlyTheTargetSlotOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			path := writeCharacterNameFixture(
				t, platform, true, true, "Ranni", "Ranni")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			slotAt, flagAt, summaryAt := deleteTestRanges(platform)
			data[slotAt+deleteTestSlotSize-1] = 0xA5
			data[summaryAt+deleteTestSummaryStride-1] = 0x5A
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatalf("rewrite fixture: %v", err)
			}

			engine := New()
			loaded, err := engine.LoadSave(path, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

			result, err := engine.DeleteCharacter(
				loaded.SaveSessionID, deleteTestSlot, "0")
			if err != nil {
				t.Fatalf("DeleteCharacter: %v", err)
			}
			want := DeleteCharacterResult{
				SaveSessionID: loaded.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   deleteTestSlot,
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			expected := bytes.Clone(before)
			clear(expected[slotAt : slotAt+deleteTestSlotSize])
			expected[flagAt] = 0
			clear(expected[summaryAt : summaryAt+deleteTestSummaryStride])
			if after := engine.sessions[loaded.SaveSessionID].snapshot.data; !bytes.Equal(after, expected) {
				t.Error("deletion changed bytes outside the target slot, flag and summary")
			}

			profile, err := engine.GetCharacterProfile(
				loaded.SaveSessionID, deleteTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile: %v", err)
			}
			if profile.Active || profile.Name != "" {
				t.Errorf("deleted profile = %+v, want inactive zero values", profile)
			}
		})
	}
}

func TestDeleteCharacterClearsAnInactiveResidualSummary(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeCharacterNameFixture(
		t, PlatformPC, false, false, "", "Residual"), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	if _, err := engine.DeleteCharacter(
		loaded.SaveSessionID, deleteTestSlot, "0"); err != nil {
		t.Fatalf("DeleteCharacter: %v", err)
	}
	slotAt, flagAt, summaryAt := deleteTestRanges(PlatformPC)
	after := engine.sessions[loaded.SaveSessionID].snapshot.data
	if !deleteTestAllZero(after[slotAt:slotAt+deleteTestSlotSize]) || after[flagAt] != 0 ||
		!deleteTestAllZero(after[summaryAt:summaryAt+deleteTestSummaryStride]) {
		t.Error("residual deletion did not clear all three target ranges")
	}
}

func TestDeleteCharacterRejectsEmptyAndUnknownSlotsWithoutMutation(t *testing.T) {
	for name, testCase := range map[string]struct {
		configure func([]byte)
		want      string
	}{
		"empty": {
			func([]byte) {},
			"no active or residual character data",
		},
		"unknown activity": {
			func(data []byte) {
				_, flagAt, _ := deleteTestRanges(PlatformPC)
				data[flagAt] = 2
			},
			"unsupported activity flag 0x02",
		},
		"unknown inactive data": {
			func(data []byte) {
				slotAt, _, _ := deleteTestRanges(PlatformPC)
				data[slotAt+0x100] = 1
			},
			"cannot establish whether inactive character 3 contains residual data",
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := writeStatsFixture(t, statsFixture{
				platform: PlatformPC,
				slot:     deleteTestSlot,
				noAnchor: true,
			})
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			testCase.configure(data)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatalf("rewrite fixture: %v", err)
			}

			engine := New()
			loaded, err := engine.LoadSave(path, "", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)
			_, err = engine.DeleteCharacter(loaded.SaveSessionID, deleteTestSlot, "0")
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			session := engine.sessions[loaded.SaveSessionID]
			if !bytes.Equal(session.snapshot.data, before) ||
				session.session.revisionString() != "0" || session.session.dirty {
				t.Error("rejected deletion changed snapshot, revision or dirty state")
			}
		})
	}
}

func TestDeleteCharacterPersistsAndReloadsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeCharacterNameFixture(
				t, platform, true, true, "Ranni", "Ranni"), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			if _, err := engine.DeleteCharacter(
				loaded.SaveSessionID, deleteTestSlot, "0"); err != nil {
				t.Fatalf("DeleteCharacter: %v", err)
			}

			target := filepath.Join(t.TempDir(), "deleted-save")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloaded := New()
			session, err := reloaded.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			slotAt, flagAt, summaryAt := deleteTestRanges(platform)
			after := reloaded.sessions[session.SaveSessionID].snapshot.data
			if !deleteTestAllZero(after[slotAt:slotAt+deleteTestSlotSize]) || after[flagAt] != 0 ||
				!deleteTestAllZero(after[summaryAt:summaryAt+deleteTestSummaryStride]) {
				t.Error("reloaded save does not contain the cleared target ranges")
			}
		})
	}
}
