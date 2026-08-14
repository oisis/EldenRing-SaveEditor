package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	cloneTestSourceSlot      = 2
	cloneTestTargetSlot      = 5
	cloneTestCollisionSlot   = 7
	cloneTestSlotSize        = 0x280000
	cloneTestPCSlotBase      = 0x310
	cloneTestPCSlotStride    = 0x280010
	cloneTestPS4SlotBase     = 0x70
	cloneTestPS4SlotStride   = 0x280000
	cloneTestPCUserDataBase  = 0x19003B0
	cloneTestPS4UserDataBase = 0x1900070
	cloneTestFlagsOffset     = 0x1954
	cloneTestSummaryOffset   = 0x195E
	cloneTestSummaryStride   = 0x24C
	cloneTestAnchorAt        = 0x20 + 5120*8 + 0x1B0
	cloneTestPlayerNameAt    = cloneTestAnchorAt - 0x11B
)

func cloneTestSlotAt(platform Platform, slot int) int64 {
	if platform == PlatformPS4 {
		return cloneTestPS4SlotBase + int64(slot)*cloneTestPS4SlotStride
	}
	return cloneTestPCSlotBase + int64(slot)*cloneTestPCSlotStride
}

func cloneTestUserDataBase(platform Platform) int64 {
	if platform == PlatformPS4 {
		return cloneTestPS4UserDataBase
	}
	return cloneTestPCUserDataBase
}

func cloneTestFlagAt(platform Platform, slot int) int64 {
	return cloneTestUserDataBase(platform) + cloneTestFlagsOffset + int64(slot)
}

func cloneTestSummaryAt(platform Platform, slot int) int64 {
	return cloneTestUserDataBase(platform) + cloneTestSummaryOffset +
		int64(slot)*cloneTestSummaryStride
}

func writeCloneCharacterFixture(t *testing.T, platform Platform, sourceName string) string {
	t.Helper()

	content := gestureTestActiveFixture(
		platform, cloneTestSourceSlot, cloneTestAnchorAt, 0)
	content.records = setGestureTestRecords()
	path := writeGestureFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sourceSlotAt := cloneTestSlotAt(platform, cloneTestSourceSlot)
	binary.LittleEndian.PutUint32(data[sourceSlotAt:], 0x6E)
	copy(data[sourceSlotAt+cloneTestPlayerNameAt:], setNameTestEncode(sourceName))
	sourceSummaryAt := cloneTestSummaryAt(platform, cloneTestSourceSlot)
	copy(data[sourceSummaryAt:], setNameTestEncode(sourceName))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	return path
}

func addCloneTestResidualName(
	t *testing.T,
	path string,
	platform Platform,
	name string,
) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sourceSlotAt := cloneTestSlotAt(platform, cloneTestSourceSlot)
	collisionSlotAt := cloneTestSlotAt(platform, cloneTestCollisionSlot)
	copy(data[collisionSlotAt:collisionSlotAt+cloneTestSlotSize],
		data[sourceSlotAt:sourceSlotAt+cloneTestSlotSize])
	copy(data[collisionSlotAt+cloneTestPlayerNameAt:], setNameTestEncode(name))
	sourceSummaryAt := cloneTestSummaryAt(platform, cloneTestSourceSlot)
	collisionSummaryAt := cloneTestSummaryAt(platform, cloneTestCollisionSlot)
	copy(data[collisionSummaryAt:collisionSummaryAt+cloneTestSummaryStride],
		data[sourceSummaryAt:sourceSummaryAt+cloneTestSummaryStride])
	copy(data[collisionSummaryAt:], setNameTestEncode(name))
	data[cloneTestFlagAt(platform, cloneTestCollisionSlot)] = 0
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
}

func TestCloneCharacterCopiesOnlyTheTargetAndPersistsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeCloneCharacterFixture(t, platform, "Ranni"), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			session := engine.sessions[loaded.SaveSessionID]
			before := bytes.Clone(session.snapshot.data)

			result, err := engine.CloneCharacter(
				loaded.SaveSessionID, cloneTestSourceSlot, cloneTestTargetSlot, "0")
			if err != nil {
				t.Fatalf("CloneCharacter: %v", err)
			}
			want := CloneCharacterResult{
				SaveSessionID:     loaded.SaveSessionID,
				SaveRevision:      "1",
				SourceCharacterID: cloneTestSourceSlot,
				TargetSlotID:      cloneTestTargetSlot,
				Name:              "Ranni 2",
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			expected := bytes.Clone(before)
			sourceSlotAt := cloneTestSlotAt(platform, cloneTestSourceSlot)
			targetSlotAt := cloneTestSlotAt(platform, cloneTestTargetSlot)
			copy(expected[targetSlotAt:targetSlotAt+cloneTestSlotSize],
				expected[sourceSlotAt:sourceSlotAt+cloneTestSlotSize])
			copy(expected[targetSlotAt+cloneTestPlayerNameAt:], setNameTestEncode("Ranni 2"))
			sourceSummaryAt := cloneTestSummaryAt(platform, cloneTestSourceSlot)
			targetSummaryAt := cloneTestSummaryAt(platform, cloneTestTargetSlot)
			copy(expected[targetSummaryAt:targetSummaryAt+cloneTestSummaryStride],
				expected[sourceSummaryAt:sourceSummaryAt+cloneTestSummaryStride])
			copy(expected[targetSummaryAt:], setNameTestEncode("Ranni 2"))
			expected[cloneTestFlagAt(platform, cloneTestTargetSlot)] = 1
			if !bytes.Equal(session.snapshot.data, expected) {
				t.Error("clone changed bytes outside the target slot, flag and summary")
			}

			target := filepath.Join(t.TempDir(), "cloned-save")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloaded := New()
			reloadedSession, err := reloaded.LoadSave(target, string(platform))
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			profile, err := reloaded.GetCharacterProfile(
				reloadedSession.SaveSessionID, cloneTestTargetSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile: %v", err)
			}
			if !profile.Active || profile.Name != "Ranni 2" {
				t.Errorf("reloaded profile = %+v, want active Ranni 2", profile)
			}
		})
	}
}

func TestCloneCharacterCountsResidualNamesWhenChoosingTheSuffix(t *testing.T) {
	path := writeCloneCharacterFixture(t, PlatformPC, "Ranni")
	addCloneTestResidualName(t, path, PlatformPC, "Ranni 2")
	engine := New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.CloneCharacter(
		loaded.SaveSessionID, cloneTestSourceSlot, cloneTestTargetSlot, "0")
	if err != nil {
		t.Fatalf("CloneCharacter: %v", err)
	}
	if result.Name != "Ranni 3" {
		t.Errorf("clone name = %q, want Ranni 3", result.Name)
	}
}

func TestUniqueClonedCharacterNameRespectsTheUTF16Boundary(t *testing.T) {
	base := strings.Repeat("😀", 8)
	got := uniqueClonedCharacterName(base, map[string]struct{}{base: {}})
	want := strings.Repeat("😀", 7) + " 2"
	if got != want {
		t.Errorf("clone name = %q, want %q", got, want)
	}
}

func TestCloneCharacterRejectsAnUnavailableTargetWithoutMutation(t *testing.T) {
	for name, configure := range map[string]func([]byte){
		"active": func(data []byte) {
			data[cloneTestFlagAt(PlatformPC, cloneTestTargetSlot)] = 1
		},
		"residual": func(data []byte) {
			data[cloneTestSlotAt(PlatformPC, cloneTestTargetSlot)+0x100] = 1
		},
		"residual summary": func(data []byte) {
			data[cloneTestSummaryAt(PlatformPC, cloneTestTargetSlot)] = 1
		},
		"unknown activity": func(data []byte) {
			data[cloneTestFlagAt(PlatformPC, cloneTestTargetSlot)] = 2
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := writeCloneCharacterFixture(t, PlatformPC, "Ranni")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			configure(data)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatalf("rewrite fixture: %v", err)
			}

			engine := New()
			loaded, err := engine.LoadSave(path, "")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			session := engine.sessions[loaded.SaveSessionID]
			before := bytes.Clone(session.snapshot.data)
			if _, err := engine.CloneCharacter(
				loaded.SaveSessionID, cloneTestSourceSlot, cloneTestTargetSlot, "0"); err == nil {
				t.Fatal("CloneCharacter accepted an unavailable target")
			}
			if !bytes.Equal(session.snapshot.data, before) ||
				session.session.revisionString() != "0" || session.session.dirty {
				t.Error("rejected clone changed snapshot, revision or dirty state")
			}
		})
	}
}
