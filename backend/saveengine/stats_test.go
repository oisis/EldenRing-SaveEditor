package saveengine

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

// Synthetic container layout used only by this test. The offsets are restated
// literally instead of reused from the implementation, so a changed base, stride
// or field offset fails here.
const (
	statsPCSlotDataBase   = 0x300 + 0x10 // first PC slot data, behind its MD5 prefix
	statsPCSlotStride     = 0x280010
	statsPS4SlotDataBase  = 0x70 // first PS4 slot data, no MD5 prefix
	statsPS4SlotStride    = 0x280000
	statsFixtureSlotStats = 0x280000
)

// statsFixture describes the synthetic slot content one test save is built from:
// which slot carries which activity flag, where its statistics anchor sits
// relative to the start of the slot data, and which raw values precede it. A
// residual slot is expressed as a zero flag with the anchor and the values still
// written into the file.
type statsFixture struct {
	platform Platform
	slot     int
	flag     byte
	anchorAt int64
	values   CharacterStats
	noAnchor bool
}

// writeStatsFixture builds a synthetic save and returns its path inside
// t.TempDir(). Only the activity flag, the anchor and the confirmed statistics
// are written; the rest of the container stays zeroed.
func writeStatsFixture(t *testing.T, content statsFixture) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch content.platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = pcUserData10DataOffset
		slotBase = statsPCSlotDataBase + int64(content.slot)*statsPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = ps4UserData10DataOffset
		slotBase = statsPS4SlotDataBase + int64(content.slot)*statsPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", content.platform)
	}

	data[userData10Base+userData10ActiveFlagsOffset+int64(content.slot)] = content.flag

	if !content.noAnchor {
		anchor := slotBase + content.anchorAt
		if anchor+int64(len(statsAnchor)) > slotBase+statsFixtureSlotStats {
			t.Fatalf("anchor at 0x%X does not fit into the slot data", content.anchorAt)
		}
		copy(data[anchor:], statsAnchor)

		values := content.values
		for offset, value := range map[int64]uint32{
			-423: values.HP,
			-419: values.MaxHP,
			-415: values.BaseMaxHP,
			-411: values.FP,
			-407: values.MaxFP,
			-403: values.BaseMaxFP,
			-395: values.SP,
			-391: values.MaxSP,
			-387: values.BaseMaxSP,
			-379: values.Vigor,
			-375: values.Mind,
			-371: values.Endurance,
			-367: values.Strength,
			-363: values.Dexterity,
			-359: values.Intelligence,
			-355: values.Faith,
			-351: values.Arcane,
			-335: values.Level,
		} {
			binary.LittleEndian.PutUint32(data[anchor+offset:], value)
		}
	}

	path := filepath.Join(t.TempDir(), "stats.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestGetCharacterStatsReadsTheActiveSlotOfBothPlatforms(t *testing.T) {
	// The two fixtures put the anchor at a different position inside the slot, so
	// a fixed offset instead of a search cannot pass both cases.
	cases := []statsFixture{
		{
			platform: PlatformPC,
			slot:     0,
			flag:     1,
			anchorAt: 0x01A7,
			values: CharacterStats{
				HP: 1450, MaxHP: 1900, BaseMaxHP: 1800,
				FP: 210, MaxFP: 240, BaseMaxFP: 230,
				SP: 130, MaxSP: 135, BaseMaxSP: 132,
				Vigor: 40, Mind: 20, Endurance: 25, Strength: 50,
				Dexterity: 18, Intelligence: 12, Faith: 14, Arcane: 9,
				Level: 109,
			},
		},
		{
			platform: PlatformPS4,
			slot:     7,
			flag:     1,
			anchorAt: 0x1F4C2,
			values: CharacterStats{
				HP: 700, MaxHP: 700, BaseMaxHP: 700,
				FP: 95, MaxFP: 95, BaseMaxFP: 95,
				SP: 100, MaxSP: 100, BaseMaxSP: 100,
				Vigor: 15, Mind: 10, Endurance: 11, Strength: 14,
				Dexterity: 13, Intelligence: 9, Faith: 12, Arcane: 8,
				Level: 13,
			},
		},
	}

	for _, testCase := range cases {
		t.Run(string(testCase.platform), func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(writeStatsFixture(t, testCase), string(testCase.platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := engine.GetCharacterStats(loaded.SaveSessionID, testCase.slot)
			if err != nil {
				t.Fatalf("GetCharacterStats: %v", err)
			}

			want := testCase.values
			want.SaveSessionID = loaded.SaveSessionID
			want.SaveRevision = "0"
			want.CharacterID = testCase.slot
			want.Active = true
			if !reflect.DeepEqual(result, want) {
				t.Errorf("result = %+v, want %+v", result, want)
			}
		})
	}
}

func TestGetCharacterStatsReportsAResidualSlotAsInactive(t *testing.T) {
	content := statsFixture{
		platform: PlatformPC,
		slot:     4,
		flag:     0,
		anchorAt: 0x0800,
		values: CharacterStats{
			HP: 999, MaxHP: 999, BaseMaxHP: 999,
			FP: 99, MaxFP: 99, BaseMaxFP: 99,
			SP: 88, MaxSP: 88, BaseMaxSP: 88,
			Vigor: 60, Mind: 30, Endurance: 40, Strength: 70,
			Dexterity: 20, Intelligence: 15, Faith: 16, Arcane: 10,
			Level: 178,
		},
	}

	engine := New()
	loaded, err := engine.LoadSave(writeStatsFixture(t, content), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := engine.GetCharacterStats(loaded.SaveSessionID, content.slot)
	if err != nil {
		t.Fatalf("GetCharacterStats: %v", err)
	}

	want := CharacterStats{SaveSessionID: loaded.SaveSessionID, SaveRevision: "0", CharacterID: content.slot}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("result = %+v, want %+v", result, want)
	}
}

func TestGetCharacterStatsRejectsInvalidRequests(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(writeStatsFixture(t, statsFixture{
		platform: PlatformPC,
		slot:     2,
		flag:     1,
		noAnchor: true,
	}), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	cases := map[string]struct {
		saveSessionID string
		characterID   int
		want          string
	}{
		"empty session":         {"", 0, "saveSessionID is required"},
		"unknown session":       {"missing", 0, `unknown save session "missing"`},
		"characterID -1":        {loaded.SaveSessionID, -1, "characterID -1 is outside the range 0..9"},
		"characterID 10":        {loaded.SaveSessionID, 10, "characterID 10 is outside the range 0..9"},
		"characterID 11":        {loaded.SaveSessionID, 11, "characterID 11 is outside the range 0..9"},
		"active slot no anchor": {loaded.SaveSessionID, 2, "character 2 carries no statistics anchor"},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.GetCharacterStats(testCase.saveSessionID, testCase.characterID)
			if err == nil {
				t.Fatalf("GetCharacterStats accepted %s", strconv.Quote(name))
			}
			if err.Error() != testCase.want {
				t.Errorf("error = %q, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, CharacterStats{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
}
