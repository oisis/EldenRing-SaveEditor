package saveengine

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

// charactersFixture describes the synthetic UserData10 content one test save is
// built from: which slots carry an activity flag and what raw name and level
// their profile summary holds. A residual slot is expressed as a zero flag with
// a name and a level still written into the summary.
type charactersFixture struct {
	flags [characterSlotCount]byte
	names [characterSlotCount]string
	level [characterSlotCount]uint32
}

// writeCharactersFixture builds a synthetic save of platform and returns its
// path inside t.TempDir(). Only the fields GetSaveCharacters reads are written;
// the rest of the container stays zeroed.
func writeCharactersFixture(t *testing.T, platform Platform, content charactersFixture) string {
	t.Helper()

	var data []byte
	var base int64
	switch platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		base = pcUserData10DataOffset
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		base = ps4UserData10DataOffset
	default:
		t.Fatalf("unknown platform %q", platform)
	}

	for slot := 0; slot < characterSlotCount; slot++ {
		data[base+userData10ActiveFlagsOffset+int64(slot)] = content.flags[slot]

		summary := base + userData10SummaryOffset + int64(slot)*userData10SummaryStride
		for index, unit := range utf16.Encode([]rune(content.names[slot])) {
			binary.LittleEndian.PutUint16(data[summary+summaryNameOffset+int64(index)*2:], unit)
		}
		binary.LittleEndian.PutUint32(data[summary+summaryLevelOffset:], content.level[slot])
	}

	path := filepath.Join(t.TempDir(), "characters.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetSaveCharactersReadsTheTenSlotsOfBothPlatforms(t *testing.T) {
	// Exactly 16 UTF-16 units, so the name field holds no NUL terminator.
	unterminated := strings.Repeat("A", summaryNameUnits)

	content := charactersFixture{}
	// Slot 0: an ordinary active character.
	content.flags[0], content.names[0], content.level[0] = 1, "Tarnished", 150
	// Slot 2: a residual slot — the flag is cleared but the deleted character's
	// name and level are still in the summary.
	content.flags[2], content.names[2], content.level[2] = 0, "Deleted", 99
	// Slot 5: an active name that fills the whole field.
	content.flags[5], content.names[5], content.level[5] = 1, unterminated, 713
	// Slot 9: an active flag must be exactly 1; any other value is not active.
	content.flags[9], content.names[9], content.level[9] = 2, "Ghost", 42

	want := make([]CharacterSummary, characterSlotCount)
	for slot := range want {
		want[slot] = CharacterSummary{CharacterID: slot}
	}
	want[0] = CharacterSummary{CharacterID: 0, Active: true, Name: "Tarnished", Level: 150}
	want[5] = CharacterSummary{CharacterID: 5, Active: true, Name: unterminated, Level: 713}

	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			path := writeCharactersFixture(t, platform, content)

			engine := New()
			info, err := engine.LoadSave(path, string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetSaveCharacters(info.SaveSessionID)
			if err != nil {
				t.Fatalf("GetSaveCharacters: %v", err)
			}
			if result.SaveSessionID != info.SaveSessionID {
				t.Errorf("saveSessionID = %q, want %q", result.SaveSessionID, info.SaveSessionID)
			}
			if result.Characters == nil {
				t.Fatal("characters slice is nil")
			}
			if !reflect.DeepEqual(result.Characters, want) {
				t.Errorf("characters =\n%+v\nwant\n%+v", result.Characters, want)
			}
		})
	}
}

func TestGetSaveCharactersRejectsUnusableIdentifiers(t *testing.T) {
	path := writeCharactersFixture(t, PlatformPC, charactersFixture{})
	engine := New()
	info, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if err := engine.CloseSession(info.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	cases := []struct {
		name          string
		saveSessionID string
		want          string
	}{
		{"empty", "", "saveSessionID is required"},
		{"unknown", "not-a-session", `unknown save session "not-a-session"`},
		{"closed", info.SaveSessionID, `unknown save session "` + info.SaveSessionID + `"`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result, err := engine.GetSaveCharacters(test.saveSessionID)
			if err == nil {
				t.Fatal("GetSaveCharacters accepted an unusable identifier")
			}
			if err.Error() != test.want {
				t.Errorf("error = %q, want %q", err, test.want)
			}
			if !reflect.DeepEqual(result, SaveCharacters{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}
