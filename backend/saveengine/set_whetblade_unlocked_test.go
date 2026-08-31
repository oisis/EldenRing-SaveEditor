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
	setWhetbladeKnifeFlag    = uint32(60130)
	setWhetbladeKnifeGameID  = uint32(0x4000218E)
	setWhetbladeIronFlag     = uint32(65610)
	setWhetbladeIronGameID   = uint32(0x4000230A)
	setWhetbladeMenuFlag     = uint32(65800)
	setWhetbladeSystemFlag   = uint32(1042378601)
	setWhetbladeFullAnchorAt = int64(0x20 + 5120*8)
)

var setWhetbladeOthers = []WhetbladeState{
	{EventFlagID: 65610, GameID: 0x4000230A},
	{EventFlagID: 65640, GameID: 0x4000230B},
	{EventFlagID: 65660, GameID: 0x4000230C},
	{EventFlagID: 65680, GameID: 0x4000230D},
	{EventFlagID: 65700, GameID: 0x4000230E},
}

func setWhetbladeFixture(t *testing.T, platform Platform) (string, eventFlagTestFixture) {
	t.Helper()
	content := eventFlagTestContent(platform)
	content.anchorAt = setWhetbladeFullAnchorAt
	content.set = nil
	path := writeEventFlagFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	slotBase := eventFlagTestPCSlotDataBase + int64(content.slot)*eventFlagTestPCSlotStride
	if platform == PlatformPS4 {
		slotBase = eventFlagTestPS4SlotDataBase + int64(content.slot)*eventFlagTestPS4SlotStride
	}
	binary.LittleEndian.PutUint32(data[slotBase:], 0x6E)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path, content
}

func TestSetWhetbladeUnlockedPersistsCompleteStateOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			path, content := setWhetbladeFixture(t, platform)
			engine := New()
			loaded, err := engine.LoadSave(path, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.SetWhetbladeUnlocked(
				loaded.SaveSessionID,
				content.slot,
				WhetbladeState{EventFlagID: setWhetbladeKnifeFlag, GameID: setWhetbladeKnifeGameID},
				[]uint32{65600, setWhetbladeSystemFlag},
				setWhetbladeOthers,
				setWhetbladeMenuFlag,
				true,
				"0",
			)
			if err != nil {
				t.Fatalf("SetWhetbladeUnlocked: %v", err)
			}
			if result.SaveRevision != "1" || !result.Unlocked {
				t.Errorf("result = %+v, want revision 1 and unlocked", result)
			}
			assertWhetbladeState(t, engine, loaded.SaveSessionID, content.slot, true)

			target := filepath.Join(t.TempDir(), "written.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			assertWhetbladeState(t, reloadedEngine, reloaded.SaveSessionID, content.slot, true)
		})
	}
}

func TestSetWhetbladeUnlockedKeepsMenuUntilTheLastWhetbladeIsLocked(t *testing.T) {
	path, content := setWhetbladeFixture(t, PlatformPC)
	engine := New()
	loaded, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	knife := WhetbladeState{EventFlagID: setWhetbladeKnifeFlag, GameID: setWhetbladeKnifeGameID}
	iron := WhetbladeState{EventFlagID: setWhetbladeIronFlag, GameID: setWhetbladeIronGameID}
	otherThanIron := append([]WhetbladeState{knife}, setWhetbladeOthers[1:]...)

	if _, err := engine.SetWhetbladeUnlocked(
		loaded.SaveSessionID, content.slot, knife,
		[]uint32{65600, setWhetbladeSystemFlag}, setWhetbladeOthers,
		setWhetbladeMenuFlag, true, "0"); err != nil {
		t.Fatalf("unlock knife: %v", err)
	}
	if _, err := engine.SetWhetbladeUnlocked(
		loaded.SaveSessionID, content.slot, iron,
		[]uint32{65620, 65630}, otherThanIron,
		setWhetbladeMenuFlag, true, "1"); err != nil {
		t.Fatalf("unlock iron: %v", err)
	}
	// Leave Iron represented only by its Inventory item. The shared menu rule
	// must use the same flag-or-item state as GetWhetblades, not only main flags.
	sectionAt, err := eventFlagSectionStart(engine.sessions[loaded.SaveSessionID], content.slot)
	if err != nil {
		t.Fatalf("eventFlagSectionStart: %v", err)
	}
	ironPosition, err := resolveEventFlag(setWhetbladeIronFlag)
	if err != nil {
		t.Fatalf("resolve iron flag: %v", err)
	}
	ironRaw, err := engine.sessions[loaded.SaveSessionID].snapshot.readAt(
		sectionAt+ironPosition.offset, 1)
	if err != nil {
		t.Fatalf("read iron flag: %v", err)
	}
	if err := engine.sessions[loaded.SaveSessionID].snapshot.writeAt(
		sectionAt+ironPosition.offset, []byte{ironRaw[0] &^ (1 << ironPosition.bit)}); err != nil {
		t.Fatalf("clear iron main flag in fixture: %v", err)
	}
	if _, err := engine.SetWhetbladeUnlocked(
		loaded.SaveSessionID, content.slot, knife,
		[]uint32{65600, setWhetbladeSystemFlag}, setWhetbladeOthers,
		setWhetbladeMenuFlag, false, "2"); err != nil {
		t.Fatalf("lock knife: %v", err)
	}
	flags, err := engine.GetEventFlags(
		loaded.SaveSessionID, content.slot,
		[]uint32{setWhetbladeKnifeFlag, 65600, setWhetbladeSystemFlag, setWhetbladeMenuFlag})
	if err != nil {
		t.Fatalf("GetEventFlags after knife lock: %v", err)
	}
	if flags.Flags[setWhetbladeKnifeFlag] || flags.Flags[65600] ||
		flags.Flags[setWhetbladeSystemFlag] || !flags.Flags[setWhetbladeMenuFlag] {
		t.Errorf("flags after knife lock = %+v", flags.Flags)
	}
	if present := whetbladePresence(t, engine, loaded.SaveSessionID, content.slot,
		setWhetbladeKnifeGameID); present {
		t.Error("Whetstone Knife item remains after locking")
	}

	if _, err := engine.SetWhetbladeUnlocked(
		loaded.SaveSessionID, content.slot, iron,
		[]uint32{65620, 65630}, otherThanIron,
		setWhetbladeMenuFlag, false, "3"); err != nil {
		t.Fatalf("lock iron: %v", err)
	}
	flags, err = engine.GetEventFlags(
		loaded.SaveSessionID, content.slot, []uint32{setWhetbladeIronFlag, 65620, 65630, setWhetbladeMenuFlag})
	if err != nil {
		t.Fatalf("GetEventFlags after final lock: %v", err)
	}
	for flagID, value := range flags.Flags {
		if value {
			t.Errorf("flag %d remains set after final lock", flagID)
		}
	}
}

func TestSetWhetbladeUnlockedRejectsInvalidStateWithoutMutation(t *testing.T) {
	path, content := setWhetbladeFixture(t, PlatformPC)
	engine := New()
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	_, err = engine.SetWhetbladeUnlocked(
		loaded.SaveSessionID, content.slot,
		WhetbladeState{EventFlagID: setWhetbladeKnifeFlag, GameID: setWhetbladeKnifeGameID},
		[]uint32{67000}, setWhetbladeOthers, setWhetbladeMenuFlag, true, "0")
	if err == nil || !strings.Contains(err.Error(), "block 67") {
		t.Fatalf("unsupported related flag error = %v", err)
	}
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) {
		t.Error("rejected mutation changed the snapshot")
	}
	info, _ := engine.GetSessionInfo(loaded.SaveSessionID)
	if info.UnsavedChanges || engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" {
		t.Errorf("rejected session = %+v, revision %q", info,
			engine.sessions[loaded.SaveSessionID].session.revisionString())
	}
}

func assertWhetbladeState(
	t *testing.T,
	engine *Engine,
	sessionID string,
	characterID int,
	want bool,
) {
	t.Helper()
	flags, err := engine.GetEventFlags(sessionID, characterID, []uint32{
		setWhetbladeKnifeFlag, 65600, setWhetbladeSystemFlag, setWhetbladeMenuFlag,
	})
	if err != nil {
		t.Fatalf("GetEventFlags: %v", err)
	}
	for flagID, value := range flags.Flags {
		if value != want {
			t.Errorf("flag %d = %t, want %t", flagID, value, want)
		}
	}
	if present := whetbladePresence(
		t, engine, sessionID, characterID, setWhetbladeKnifeGameID); present != want {
		t.Errorf("Whetstone Knife present = %t, want %t", present, want)
	}
}

func whetbladePresence(
	t *testing.T,
	engine *Engine,
	sessionID string,
	characterID int,
	gameID uint32,
) bool {
	t.Helper()
	present, err := engine.GetInventoryGoodsPresence(
		sessionID, characterID, []uint32{gameID})
	if err != nil {
		t.Fatalf("GetInventoryGoodsPresence: %v", err)
	}
	return present.Presence[gameID]
}
