package saveengine

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	setActiveTestSlot            = 3
	setActiveTestFlagsOffset     = 0x1954
	setActiveTestPCUserDataBase  = 0x19003B0
	setActiveTestPS4UserDataBase = 0x1900070
)

func setActiveTestFlagAt(platform Platform) int64 {
	if platform == PlatformPS4 {
		return setActiveTestPS4UserDataBase + setActiveTestFlagsOffset + setActiveTestSlot
	}
	return setActiveTestPCUserDataBase + setActiveTestFlagsOffset + setActiveTestSlot
}

func assertSetActiveRejectedUnchanged(
	t *testing.T,
	engine *Engine,
	sessionID string,
	before []byte,
) {
	t.Helper()
	session := engine.sessions[sessionID]
	if !bytes.Equal(session.snapshot.data, before) {
		t.Error("rejected activity mutation changed the private snapshot")
	}
	if revision := session.session.revisionString(); revision != "0" {
		t.Errorf("revision after rejection = %q, want 0", revision)
	}
	if session.session.dirty {
		t.Error("rejected activity mutation marked the session dirty")
	}
}

func TestSetCharacterActiveChangesOnlyTheFlagOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeCharacterNameFixture(
				t, platform, true, true, "Ranni", "Ranni"), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)
			result, err := engine.SetCharacterActive(
				loaded.SaveSessionID, setActiveTestSlot, false, "0")
			if err != nil {
				t.Fatalf("SetCharacterActive(false): %v", err)
			}
			assertCommittedReceipt(t, result.MutationReceipt, loaded.SaveSessionID,
				kindSetCharacterActive, "1")
			// The receipt is pinned from the result because operationID names one
			// execution and cannot be predicted; every other member is asserted above.
			want := SetCharacterActiveResult{
				MutationReceipt: result.MutationReceipt,
				Changed:         true,
				CharacterID:     setActiveTestSlot,
				Active:          false,
			}
			if !reflect.DeepEqual(result, want) {
				t.Errorf("deactivation result = %+v, want %+v", result, want)
			}

			expected := bytes.Clone(before)
			expected[setActiveTestFlagAt(platform)] = 0
			if after := engine.sessions[loaded.SaveSessionID].snapshot.data; !bytes.Equal(after, expected) {
				t.Error("deactivation changed bytes outside the activity flag")
			}
			profile, err := engine.GetCharacterProfile(
				loaded.SaveSessionID, setActiveTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile after deactivation: %v", err)
			}
			if profile.Active || profile.Name != "" {
				t.Errorf("inactive profile = %+v, want hidden residual data", profile)
			}

			result, err = engine.SetCharacterActive(
				loaded.SaveSessionID, setActiveTestSlot, true, "1")
			if err != nil {
				t.Fatalf("SetCharacterActive(true): %v", err)
			}
			if result.SaveRevision != "2" || !result.Active {
				t.Errorf("reactivation result = %+v, want active revision 2", result)
			}
			if after := engine.sessions[loaded.SaveSessionID].snapshot.data; !bytes.Equal(after, before) {
				t.Error("reactivation did not restore the original snapshot bytes")
			}
		})
	}
}

func TestSetCharacterActiveIdempotentRequestDoesNotCommit(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeCharacterNameFixture(
		t, PlatformPC, true, true, "Ranni", "Ranni"), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	result, err := engine.SetCharacterActive(
		loaded.SaveSessionID, setActiveTestSlot, true, "0")
	if err != nil {
		t.Fatalf("SetCharacterActive: %v", err)
	}
	if result.SaveRevision != "0" || !result.Active {
		t.Errorf("result = %+v, want unchanged active revision 0", result)
	}
	assertSetActiveRejectedUnchanged(t, engine, loaded.SaveSessionID, before)
}

func TestSetCharacterActiveRejectsUnsafeReactivationWithoutMutation(t *testing.T) {
	for name, testCase := range map[string]struct {
		fixture string
		want    string
	}{
		"empty slot": {
			writeCharacterNameFixture(t, PlatformPC, false, true, "", ""),
			"character 3 has no residual character data to reactivate",
		},
		"missing slot anchor": {
			writeCharacterNameFixture(t, PlatformPC, false, false, "", "Residual summary"),
			"cannot reactivate character 3: character 3 carries no statistics anchor",
		},
	} {
		t.Run(name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(testCase.fixture, "", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

			_, err = engine.SetCharacterActive(
				loaded.SaveSessionID, setActiveTestSlot, true, "0")
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			assertSetActiveRejectedUnchanged(t, engine, loaded.SaveSessionID, before)
		})
	}

	engine := New()
	path := writeCharacterNameFixture(
		t, PlatformPC, false, true, "Ranni", "Ranni")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	data[setActiveTestFlagAt(PlatformPC)] = 2
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	loaded, err := engine.LoadSave(path, "", "local")
	if err != nil {
		t.Fatalf("LoadSave unknown flag: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)
	_, err = engine.SetCharacterActive(
		loaded.SaveSessionID, setActiveTestSlot, true, "0")
	if err == nil || !strings.Contains(err.Error(), "unsupported activity flag 0x02") {
		t.Fatalf("unknown flag error = %v", err)
	}
	assertSetActiveRejectedUnchanged(t, engine, loaded.SaveSessionID, before)
}

func TestSetCharacterActivePersistsInactiveStateOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeCharacterNameFixture(
				t, platform, true, true, "Ranni", "Ranni"), string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			if _, err := engine.SetCharacterActive(
				loaded.SaveSessionID, setActiveTestSlot, false, "0"); err != nil {
				t.Fatalf("SetCharacterActive: %v", err)
			}

			target := filepath.Join(t.TempDir(), "inactive-save")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloaded := New()
			session, err := reloaded.LoadSave(target, string(platform), "local")
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			profile, err := reloaded.GetCharacterProfile(
				session.SaveSessionID, setActiveTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile: %v", err)
			}
			if profile.Active {
				t.Errorf("reloaded profile = %+v, want inactive", profile)
			}
		})
	}
}
