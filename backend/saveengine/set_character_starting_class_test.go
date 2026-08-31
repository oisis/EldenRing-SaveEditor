package saveengine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	setStartingClassTestSlot         = 3
	setStartingClassTestAnchorAt     = int64(0x0640)
	setStartingClassTestFullAnchorAt = int64(0x20 + 5120*8 + 0x1B0)

	setStartingClassTestPCSlotBase    = int64(0x310)
	setStartingClassTestPCSlotStride  = int64(0x280010)
	setStartingClassTestPS4SlotBase   = int64(0x70)
	setStartingClassTestPS4SlotStride = int64(0x280000)

	setStartingClassTestSummaryOffset = int64(0x195E)
	setStartingClassTestSummaryStride = int64(0x24C)
	setStartingClassTestSummaryLevel  = int64(0x22)
	setStartingClassTestSummaryClass  = int64(0x243)

	setStartingClassTestVigorOffset      = int64(-379)
	setStartingClassTestLevelOffset      = int64(-335)
	setStartingClassTestRunesOffset      = int64(-331)
	setStartingClassTestSoulMemoryOffset = int64(-327)
	setStartingClassTestClassOffset      = int64(-248)

	setStartingClassTestVagabond   = 0
	setStartingClassTestWarrior    = 1
	setStartingClassTestHero       = 2
	setStartingClassTestAstrologer = 4
	setStartingClassTestProphet    = 5
	setStartingClassTestWretch     = 9
)

type setStartingClassTestContent struct {
	platform       Platform
	active         bool
	withAnchor     bool
	anchorAt       int64
	classID        byte
	summaryClassID byte
	attributes     CharacterAttributes
	level          uint32
	runes          uint32
	soulMemory     uint32
}

func setStartingClassTestSlotBase(platform Platform) int64 {
	if platform == PlatformPS4 {
		return setStartingClassTestPS4SlotBase + setStartingClassTestSlot*setStartingClassTestPS4SlotStride
	}
	return setStartingClassTestPCSlotBase + setStartingClassTestSlot*setStartingClassTestPCSlotStride
}

func setStartingClassTestSummaryAt(platform Platform) int64 {
	base := pcUserData10DataOffset
	if platform == PlatformPS4 {
		base = ps4UserData10DataOffset
	}
	return base + setStartingClassTestSummaryOffset + setStartingClassTestSlot*setStartingClassTestSummaryStride
}

func writeSetStartingClassFixture(t *testing.T, content setStartingClassTestContent) string {
	t.Helper()

	fixture := statsFixture{
		platform: content.platform,
		slot:     setStartingClassTestSlot,
		anchorAt: content.anchorAt,
		noAnchor: !content.withAnchor,
		values: CharacterStats{
			HP: 1000, MaxHP: 1200, BaseMaxHP: 1100,
			FP: 100, MaxFP: 120, BaseMaxFP: 110,
			SP: 90, MaxSP: 95, BaseMaxSP: 92,
			Vigor:        content.attributes.Vigor,
			Mind:         content.attributes.Mind,
			Endurance:    content.attributes.Endurance,
			Strength:     content.attributes.Strength,
			Dexterity:    content.attributes.Dexterity,
			Intelligence: content.attributes.Intelligence,
			Faith:        content.attributes.Faith,
			Arcane:       content.attributes.Arcane,
			Level:        content.level,
		},
	}
	if content.active {
		fixture.flag = userData10ActiveFlagValue
	}

	path := writeStatsFixture(t, fixture)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	summary := setStartingClassTestSummaryAt(content.platform)
	data[summary+setStartingClassTestSummaryClass] = content.summaryClassID
	binary.LittleEndian.PutUint32(data[summary+setStartingClassTestSummaryLevel:], content.level)

	if content.withAnchor {
		anchor := setStartingClassTestSlotBase(content.platform) + content.anchorAt
		data[anchor+setStartingClassTestClassOffset] = content.classID
		binary.LittleEndian.PutUint32(data[anchor+setStartingClassTestRunesOffset:], content.runes)
		binary.LittleEndian.PutUint32(data[anchor+setStartingClassTestSoulMemoryOffset:], content.soulMemory)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	return path
}

func loadSetStartingClassSession(t *testing.T, content setStartingClassTestContent) (*Engine, string) {
	t.Helper()
	engine := New()
	loaded, err := engine.LoadSave(
		writeSetStartingClassFixture(t, content), string(content.platform), "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func assertSetStartingClassRejectedUnchanged(t *testing.T, engine *Engine, sessionID string, before []byte) {
	t.Helper()
	if !bytes.Equal(before, engine.sessions[sessionID].snapshot.data) {
		t.Error("rejected starting class mutation changed the private snapshot")
	}
	if revision := engine.sessions[sessionID].session.revisionString(); revision != "0" {
		t.Errorf("revision after rejection = %q, want 0", revision)
	}
	if engine.sessions[sessionID].session.dirty {
		t.Error("rejected starting class mutation marked the session dirty")
	}
}

// setStartingClassConfirmedBases is the confirmed definition of every playable
// class, written out here rather than read from the catalog: the point of these
// tests is that the setter reproduces exactly these values, so a catalog change
// that silently altered them must fail here instead of being copied into the
// expectation.
//
// The levels are the CharaInitParam soulLv facts: 0..9 confirmed against a
// native vanilla save holding all ten freshly created classes, 10 and 11 read
// from the Regulation 1.17 rows 3010 and 3011.
var setStartingClassConfirmedBases = []struct {
	id         uint8
	name       string
	level      uint32
	attributes CharacterAttributes
}{
	{0, "Vagabond", 9, CharacterAttributes{Vigor: 15, Mind: 10, Endurance: 11, Strength: 14, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7}},
	{1, "Warrior", 8, CharacterAttributes{Vigor: 11, Mind: 12, Endurance: 11, Strength: 10, Dexterity: 16, Intelligence: 10, Faith: 8, Arcane: 9}},
	{2, "Hero", 7, CharacterAttributes{Vigor: 14, Mind: 9, Endurance: 12, Strength: 16, Dexterity: 9, Intelligence: 7, Faith: 8, Arcane: 11}},
	{3, "Bandit", 5, CharacterAttributes{Vigor: 10, Mind: 11, Endurance: 10, Strength: 9, Dexterity: 13, Intelligence: 9, Faith: 8, Arcane: 14}},
	{4, "Astrologer", 6, CharacterAttributes{Vigor: 9, Mind: 15, Endurance: 9, Strength: 8, Dexterity: 12, Intelligence: 16, Faith: 7, Arcane: 9}},
	{5, "Prophet", 7, CharacterAttributes{Vigor: 10, Mind: 14, Endurance: 8, Strength: 11, Dexterity: 10, Intelligence: 7, Faith: 16, Arcane: 10}},
	{6, "Confessor", 10, CharacterAttributes{Vigor: 10, Mind: 13, Endurance: 10, Strength: 12, Dexterity: 12, Intelligence: 9, Faith: 14, Arcane: 9}},
	{7, "Samurai", 9, CharacterAttributes{Vigor: 12, Mind: 11, Endurance: 13, Strength: 12, Dexterity: 15, Intelligence: 9, Faith: 8, Arcane: 8}},
	{8, "Prisoner", 9, CharacterAttributes{Vigor: 11, Mind: 12, Endurance: 11, Strength: 11, Dexterity: 14, Intelligence: 14, Faith: 6, Arcane: 9}},
	{9, "Wretch", 1, CharacterAttributes{Vigor: 10, Mind: 10, Endurance: 10, Strength: 10, Dexterity: 10, Intelligence: 10, Faith: 10, Arcane: 10}},
	{10, "Idus Knight", 7, CharacterAttributes{Vigor: 10, Mind: 12, Endurance: 11, Strength: 13, Dexterity: 15, Intelligence: 8, Faith: 11, Arcane: 6}},
	{11, "Heavy Knight", 10, CharacterAttributes{Vigor: 14, Mind: 8, Endurance: 17, Strength: 15, Dexterity: 11, Intelligence: 7, Faith: 8, Arcane: 9}},
}

// setStartingClassExpectedImage builds the only snapshot the reset is allowed to
// produce: the image before the call with exactly the eight attributes, the
// PlayerGameData level, the PlayerGameData class byte, the ProfileSummary level
// and the ProfileSummary class byte replaced. Comparing the whole image against
// it proves in one assertion that TotalGetSoul, the held runes, HP/FP/SP, the
// name, the appearance and every other byte of the save stayed exactly as they
// were.
func setStartingClassExpectedImage(
	before []byte,
	platform Platform,
	target uint8,
	attributes CharacterAttributes,
	level uint32,
) []byte {
	expected := bytes.Clone(before)
	anchor := setStartingClassTestSlotBase(platform) + setStartingClassTestAnchorAt
	summary := setStartingClassTestSummaryAt(platform)

	for index, value := range attributes.ordered() {
		binary.LittleEndian.PutUint32(
			expected[anchor+setStartingClassTestVigorOffset+int64(index)*4:], value)
	}
	binary.LittleEndian.PutUint32(expected[anchor+setStartingClassTestLevelOffset:], level)
	expected[anchor+setStartingClassTestClassOffset] = target
	binary.LittleEndian.PutUint32(expected[summary+setStartingClassTestSummaryLevel:], level)
	expected[summary+setStartingClassTestSummaryClass] = target
	return expected
}

func assertSetStartingClassImage(
	t *testing.T,
	engine *Engine,
	sessionID string,
	before []byte,
	platform Platform,
	target uint8,
	attributes CharacterAttributes,
	level uint32,
) {
	t.Helper()
	expected := setStartingClassExpectedImage(before, platform, target, attributes, level)
	after := engine.sessions[sessionID].snapshot.data
	if len(after) != len(expected) {
		t.Fatalf("snapshot length = %d, want %d", len(after), len(expected))
	}
	for index := range expected {
		if after[index] != expected[index] {
			t.Fatalf("snapshot byte at %d = %d, want %d", index, after[index], expected[index])
		}
	}
}

// A developed character is reset to the exact base build of every one of the ten
// classes. The stored attributes are far above every class minimum, so the old
// max(current, minimum) rule would have left all of them untouched; here every
// single one must come down to the class base and the level must come from the
// class document rather than from the attribute sum.
func TestSetCharacterStartingClassResetsDevelopedCharacterToEveryClassBase(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			for _, base := range setStartingClassConfirmedBases {
				t.Run(base.name, func(t *testing.T) {
					developed := CharacterAttributes{
						Vigor: 60, Mind: 55, Endurance: 50, Strength: 45,
						Dexterity: 40, Intelligence: 35, Faith: 30, Arcane: 25,
					}
					const developedLevel = uint32(261) // 340 - 79
					const developedRunes = uint32(123_456)
					const developedSoulMemory = uint32(9_000_000)

					engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
						platform:       platform,
						active:         true,
						withAnchor:     true,
						anchorAt:       setStartingClassTestAnchorAt,
						classID:        byte(setStartingClassTestVagabond),
						summaryClassID: byte(setStartingClassTestVagabond),
						attributes:     developed,
						level:          developedLevel,
						runes:          developedRunes,
						soulMemory:     developedSoulMemory,
					})
					before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

					result, err := engine.SetCharacterStartingClass(
						sessionID, setStartingClassTestSlot, base.id, true, "0")
					if err != nil {
						t.Fatalf("SetCharacterStartingClass: %v", err)
					}

					if result.Attributes != base.attributes {
						t.Errorf("result.Attributes = %+v, want the exact class base %+v",
							result.Attributes, base.attributes)
					}
					if result.Level != base.level {
						t.Errorf("result.Level = %d, want the class document level %d",
							result.Level, base.level)
					}
					if result.SoulMemory != developedSoulMemory {
						t.Errorf("result.SoulMemory = %d, want it preserved at %d",
							result.SoulMemory, developedSoulMemory)
					}
					if result.StartingClassID != base.id || result.SaveRevision != "1" ||
						result.CharacterID != setStartingClassTestSlot || result.SaveSessionID != sessionID {
						t.Errorf("result = %+v", result)
					}

					assertSetStartingClassImage(t, engine, sessionID, before,
						platform, base.id, base.attributes, base.level)

					anchor := setStartingClassTestSlotBase(platform) + setStartingClassTestAnchorAt
					summary := setStartingClassTestSummaryAt(platform)
					data := engine.sessions[sessionID].snapshot.data

					// Both class copies and both level copies are synchronised.
					if data[anchor+setStartingClassTestClassOffset] != base.id {
						t.Errorf("PlayerGameData class byte = %d, want %d",
							data[anchor+setStartingClassTestClassOffset], base.id)
					}
					if data[summary+setStartingClassTestSummaryClass] != base.id {
						t.Errorf("ProfileSummary class byte = %d, want %d",
							data[summary+setStartingClassTestSummaryClass], base.id)
					}
					if stored := binary.LittleEndian.Uint32(
						data[anchor+setStartingClassTestLevelOffset:]); stored != base.level {
						t.Errorf("PlayerGameData level = %d, want %d", stored, base.level)
					}
					if stored := binary.LittleEndian.Uint32(
						data[summary+setStartingClassTestSummaryLevel:]); stored != base.level {
						t.Errorf("ProfileSummary level = %d, want %d", stored, base.level)
					}

					// TotalGetSoul and the held runes are untouched.
					if stored := binary.LittleEndian.Uint32(
						data[anchor+setStartingClassTestSoulMemoryOffset:]); stored != developedSoulMemory {
						t.Errorf("stored SoulMemory = %d, want it preserved at %d",
							stored, developedSoulMemory)
					}
					if stored := binary.LittleEndian.Uint32(
						data[anchor+setStartingClassTestRunesOffset:]); stored != developedRunes {
						t.Errorf("stored held runes = %d, want them preserved at %d",
							stored, developedRunes)
					}

					// The public getters agree with the receipt.
					profile, err := engine.GetCharacterProfile(sessionID, setStartingClassTestSlot)
					if err != nil {
						t.Fatalf("GetCharacterProfile: %v", err)
					}
					if profile.StartingClassID != base.id || profile.Level != base.level {
						t.Errorf("profile = %+v, want class %d level %d",
							profile, base.id, base.level)
					}
					stats, err := engine.GetCharacterStats(sessionID, setStartingClassTestSlot)
					if err != nil {
						t.Fatalf("GetCharacterStats: %v", err)
					}
					if stats.Level != base.level ||
						stats.Vigor != base.attributes.Vigor ||
						stats.Mind != base.attributes.Mind ||
						stats.Endurance != base.attributes.Endurance ||
						stats.Strength != base.attributes.Strength ||
						stats.Dexterity != base.attributes.Dexterity ||
						stats.Intelligence != base.attributes.Intelligence ||
						stats.Faith != base.attributes.Faith ||
						stats.Arcane != base.attributes.Arcane {
						t.Errorf("stats = %+v, want the class base %+v at level %d",
							stats, base.attributes, base.level)
					}
				})
			}
		})
	}
}

// The base level moves in both directions. Confessor (10) to Wretch (1) drops
// nine levels, Wretch (1) to Confessor (10) climbs nine, and in both cases the
// resulting level is the target class document level, never the attribute sum of
// the previous build.
func TestSetCharacterStartingClassMovesTheBaseLevelInBothDirections(t *testing.T) {
	const (
		confessor = uint8(6)
		wretch    = uint8(9)
	)
	confessorBase := setStartingClassConfirmedBases[confessor]
	wretchBase := setStartingClassConfirmedBases[wretch]

	for _, step := range []struct {
		name   string
		from   uint8
		fromAt struct {
			attributes CharacterAttributes
			level      uint32
		}
		to uint8
	}{
		{"high base to low base", confessor, struct {
			attributes CharacterAttributes
			level      uint32
		}{confessorBase.attributes, confessorBase.level}, wretch},
		{"low base to high base", wretch, struct {
			attributes CharacterAttributes
			level      uint32
		}{wretchBase.attributes, wretchBase.level}, confessor},
	} {
		t.Run(step.name, func(t *testing.T) {
			engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
				platform:       PlatformPC,
				active:         true,
				withAnchor:     true,
				anchorAt:       setStartingClassTestAnchorAt,
				classID:        step.from,
				summaryClassID: step.from,
				attributes:     step.fromAt.attributes,
				level:          step.fromAt.level,
				runes:          0,
				soulMemory:     0,
			})
			before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

			want := setStartingClassConfirmedBases[step.to]
			result, err := engine.SetCharacterStartingClass(
				sessionID, setStartingClassTestSlot, step.to, true, "0")
			if err != nil {
				t.Fatalf("SetCharacterStartingClass: %v", err)
			}
			if result.Level != want.level {
				t.Errorf("result.Level = %d, want %d", result.Level, want.level)
			}
			if result.Attributes != want.attributes {
				t.Errorf("result.Attributes = %+v, want %+v", result.Attributes, want.attributes)
			}
			if result.SoulMemory != 0 {
				t.Errorf("result.SoulMemory = %d, want it preserved at 0", result.SoulMemory)
			}
			assertSetStartingClassImage(t, engine, sessionID, before,
				PlatformPC, step.to, want.attributes, want.level)
		})
	}
}

// SoulMemory is preserved whatever its value: the reset neither raises a zero to
// a level floor nor lowers a large one. Both ends are pinned because the old
// contract moved this field and the new one must never touch it.
func TestSetCharacterStartingClassPreservesSoulMemory(t *testing.T) {
	for _, soulMemory := range []uint32{0, 999_999} {
		t.Run(fmt.Sprintf("soulMemory=%d", soulMemory), func(t *testing.T) {
			vagabond := setStartingClassConfirmedBases[0]
			engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
				platform:       PlatformPC,
				active:         true,
				withAnchor:     true,
				anchorAt:       setStartingClassTestAnchorAt,
				classID:        byte(setStartingClassTestVagabond),
				summaryClassID: byte(setStartingClassTestVagabond),
				attributes:     vagabond.attributes,
				level:          vagabond.level,
				runes:          7_777,
				soulMemory:     soulMemory,
			})

			result, err := engine.SetCharacterStartingClass(
				sessionID, setStartingClassTestSlot, uint8(setStartingClassTestHero), true, "0")
			if err != nil {
				t.Fatalf("SetCharacterStartingClass: %v", err)
			}
			if result.SoulMemory != soulMemory {
				t.Errorf("result.SoulMemory = %d, want it preserved at %d", result.SoulMemory, soulMemory)
			}

			anchor := setStartingClassTestSlotBase(PlatformPC) + setStartingClassTestAnchorAt
			data := engine.sessions[sessionID].snapshot.data
			if stored := binary.LittleEndian.Uint32(
				data[anchor+setStartingClassTestSoulMemoryOffset:]); stored != soulMemory {
				t.Errorf("stored SoulMemory = %d, want %d", stored, soulMemory)
			}
			if stored := binary.LittleEndian.Uint32(
				data[anchor+setStartingClassTestRunesOffset:]); stored != 7_777 {
				t.Errorf("stored held runes = %d, want 7777", stored)
			}
		})
	}
}

// An unconfirmed reset is refused before anything is read or written. Both the
// explicit refusal and the omission a transport turns into false must leave the
// snapshot, the revision and the dirty flag exactly as they were.
func TestSetCharacterStartingClassRequiresConfirmReset(t *testing.T) {
	vagabond := setStartingClassConfirmedBases[0]
	engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
		platform:       PlatformPC,
		active:         true,
		withAnchor:     true,
		anchorAt:       setStartingClassTestAnchorAt,
		classID:        byte(setStartingClassTestVagabond),
		summaryClassID: byte(setStartingClassTestVagabond),
		attributes:     vagabond.attributes,
		level:          vagabond.level,
		runes:          100,
		soulMemory:     0,
	})
	before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

	_, err := engine.SetCharacterStartingClass(
		sessionID, setStartingClassTestSlot, uint8(setStartingClassTestHero), false, "0")
	if err == nil || !strings.Contains(err.Error(), "confirmReset must be true") {
		t.Fatalf("error = %v, want the confirmReset rejection", err)
	}
	assertSetStartingClassRejectedUnchanged(t, engine, sessionID, before)

	// The gate is checked before the revision, so an unconfirmed request cannot
	// be smuggled past it by any other argument.
	_, err = engine.SetCharacterStartingClass(
		sessionID, setStartingClassTestSlot, 200, false, "not-a-revision")
	if err == nil || !strings.Contains(err.Error(), "confirmReset must be true") {
		t.Fatalf("error = %v, want the confirmReset rejection", err)
	}
	assertSetStartingClassRejectedUnchanged(t, engine, sessionID, before)

	// The very same request with confirmReset set commits.
	if _, err := engine.SetCharacterStartingClass(
		sessionID, setStartingClassTestSlot, uint8(setStartingClassTestHero), true, "0"); err != nil {
		t.Fatalf("confirmed SetCharacterStartingClass: %v", err)
	}
}

// Repeating the committed reset is a real second mutation: the class is already
// correct, but the revision still advances, because the setter reports every
// accepted commit rather than comparing the snapshot first.
func TestSetCharacterStartingClassRepeatedCallKeepsRevisionSemantics(t *testing.T) {
	vagabond := setStartingClassConfirmedBases[0]
	hero := setStartingClassConfirmedBases[2]
	engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
		platform:       PlatformPC,
		active:         true,
		withAnchor:     true,
		anchorAt:       setStartingClassTestAnchorAt,
		classID:        byte(setStartingClassTestVagabond),
		summaryClassID: byte(setStartingClassTestVagabond),
		attributes:     vagabond.attributes,
		level:          vagabond.level,
		runes:          100,
		soulMemory:     4_242,
	})

	first, err := engine.SetCharacterStartingClass(
		sessionID, setStartingClassTestSlot, hero.id, true, "0")
	if err != nil {
		t.Fatalf("first SetCharacterStartingClass: %v", err)
	}
	afterFirst := bytes.Clone(engine.sessions[sessionID].snapshot.data)

	second, err := engine.SetCharacterStartingClass(
		sessionID, setStartingClassTestSlot, hero.id, true, first.SaveRevision)
	if err != nil {
		t.Fatalf("second SetCharacterStartingClass: %v", err)
	}
	if second.SaveRevision != "2" {
		t.Errorf("second revision = %q, want 2", second.SaveRevision)
	}
	if second.Attributes != first.Attributes || second.Level != first.Level ||
		second.SoulMemory != first.SoulMemory {
		t.Errorf("second result = %+v, want the same state as %+v", second, first)
	}
	if !bytes.Equal(afterFirst, engine.sessions[sessionID].snapshot.data) {
		t.Error("the repeated reset changed the snapshot")
	}
}

func TestSetCharacterStartingClassRejections(t *testing.T) {
	t.Run("unknown startingClassID", func(t *testing.T) {
		engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
			platform:       PlatformPC,
			active:         true,
			withAnchor:     true,
			anchorAt:       setStartingClassTestAnchorAt,
			classID:        0,
			summaryClassID: 0,
			attributes:     CharacterAttributes{Vigor: 15, Mind: 10, Endurance: 11, Strength: 14, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
			level:          9,
			runes:          100,
			soulMemory:     100,
		})
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

		// 12 is the first ID above the twelve classes Regulation 1.17 declares.
		_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 12, true, "0")
		if err == nil || err.Error() != "starting class 12 is unknown; its attribute minima are not confirmed" {
			t.Fatalf("error = %v, want unknown starting class error", err)
		}
		assertSetStartingClassRejectedUnchanged(t, engine, sessionID, before)
	})

	// A stored attribute outside the confirmed range 1..99 can only come from an
	// earlier external edit. The reset does not carry it forward and does not
	// need to reject it either: every one of the eight values is overwritten with
	// the class base, so the illegal value cannot survive the call in any form.
	// This replaces the range rejection the previous max(current, minimum)
	// contract needed, and is the stronger statement of the two: it pins the
	// resulting value instead of only refusing the input.
	t.Run("a stored attribute outside 1..99 is overwritten by the class base", func(t *testing.T) {
		astrologer := setStartingClassConfirmedBases[setStartingClassTestAstrologer]
		for _, stored := range []CharacterAttributes{
			{Vigor: 15, Mind: 10, Endurance: 11, Strength: 150, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
			{Vigor: 15, Mind: 10, Endurance: 11, Strength: 0, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
			{Vigor: 15, Mind: 10, Endurance: 11, Strength: 99, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
		} {
			engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
				platform:       PlatformPC,
				active:         true,
				withAnchor:     true,
				anchorAt:       setStartingClassTestAnchorAt,
				classID:        setStartingClassTestVagabond,
				summaryClassID: setStartingClassTestVagabond,
				attributes:     stored,
				level:          144,
				runes:          100,
				soulMemory:     100,
			})
			before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

			result, err := engine.SetCharacterStartingClass(
				sessionID, setStartingClassTestSlot, uint8(setStartingClassTestAstrologer), true, "0")
			if err != nil {
				t.Fatalf("stored strength %d: SetCharacterStartingClass: %v", stored.Strength, err)
			}
			if result.Attributes != astrologer.attributes {
				t.Errorf("stored strength %d: result.Attributes = %+v, want the class base %+v",
					stored.Strength, result.Attributes, astrologer.attributes)
			}
			if result.Level != astrologer.level {
				t.Errorf("stored strength %d: result.Level = %d, want %d",
					stored.Strength, result.Level, astrologer.level)
			}
			if result.SoulMemory != 100 {
				t.Errorf("stored strength %d: result.SoulMemory = %d, want it preserved at 100",
					stored.Strength, result.SoulMemory)
			}
			assertSetStartingClassImage(t, engine, sessionID, before, PlatformPC,
				uint8(setStartingClassTestAstrologer), astrologer.attributes, astrologer.level)
		}
	})

	t.Run("stale or malformed expectedRevision", func(t *testing.T) {
		engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
			platform:       PlatformPC,
			active:         true,
			withAnchor:     true,
			anchorAt:       setStartingClassTestAnchorAt,
			classID:        0,
			summaryClassID: 0,
			attributes:     CharacterAttributes{Vigor: 15, Mind: 10, Endurance: 11, Strength: 14, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
			level:          9,
			runes:          100,
			soulMemory:     100,
		})
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

		for _, badRev := range []string{"", "1", "01", "-1", "abc"} {
			_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 1, true, badRev)
			if err == nil {
				t.Fatalf("expected error for expectedRevision %q", badRev)
			}
			assertSetStartingClassRejectedUnchanged(t, engine, sessionID, before)
		}
	})

	t.Run("inactive slot", func(t *testing.T) {
		engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
			platform:       PlatformPC,
			active:         false,
			withAnchor:     true,
			anchorAt:       setStartingClassTestAnchorAt,
			classID:        0,
			summaryClassID: 0,
			attributes:     CharacterAttributes{Vigor: 15, Mind: 10, Endurance: 11, Strength: 14, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
			level:          9,
			runes:          100,
			soulMemory:     100,
		})
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

		_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 1, true, "0")
		if err == nil || err.Error() != "character 3 is not active" {
			t.Fatalf("error = %v, want inactive character error", err)
		}
		assertSetStartingClassRejectedUnchanged(t, engine, sessionID, before)
	})

	t.Run("slot index outside 0..9", func(t *testing.T) {
		engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
			platform:       PlatformPC,
			active:         true,
			withAnchor:     true,
			anchorAt:       setStartingClassTestAnchorAt,
			classID:        0,
			summaryClassID: 0,
			attributes:     CharacterAttributes{Vigor: 15, Mind: 10, Endurance: 11, Strength: 14, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
			level:          9,
			runes:          100,
			soulMemory:     100,
		})
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

		for _, badSlot := range []int{-1, 10, 100} {
			_, err := engine.SetCharacterStartingClass(sessionID, badSlot, 1, true, "0")
			if err == nil {
				t.Fatalf("expected error for bad slot %d", badSlot)
			}
			assertSetStartingClassRejectedUnchanged(t, engine, sessionID, before)
		}
	})

	t.Run("anchorless slot", func(t *testing.T) {
		engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
			platform:       PlatformPC,
			active:         true,
			withAnchor:     false,
			anchorAt:       setStartingClassTestAnchorAt,
			classID:        0,
			summaryClassID: 0,
			attributes:     CharacterAttributes{Vigor: 15, Mind: 10, Endurance: 11, Strength: 14, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
			level:          9,
			runes:          100,
			soulMemory:     100,
		})
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

		_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 1, true, "0")
		if err == nil || err.Error() != "character 3 carries no statistics anchor" {
			t.Fatalf("error = %v, want anchor error", err)
		}
		assertSetStartingClassRejectedUnchanged(t, engine, sessionID, before)
	})
}

func TestSetCharacterStartingClassPersistence(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := gestureTestActiveFixture(
				platform, setStartingClassTestSlot, setStartingClassTestFullAnchorAt, 0)
			content.records = setGestureTestRecords()
			source := writeGestureFixture(t, content)
			data, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read full fixture: %v", err)
			}
			binary.LittleEndian.PutUint32(data[setStartingClassTestSlotBase(platform):], 0x6E)
			if err := os.WriteFile(source, data, 0o600); err != nil {
				t.Fatalf("rewrite full fixture: %v", err)
			}

			engine := New()
			loaded, err := engine.LoadSave(source, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			// Mutate to Samurai (class 7)
			result, err := engine.SetCharacterStartingClass(
				loaded.SaveSessionID, setStartingClassTestSlot, 7, true, "0")
			if err != nil {
				t.Fatalf("SetCharacterStartingClass: %v", err)
			}

			targetPath := filepath.Join(t.TempDir(), "persisted.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", targetPath); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}

			// Reload written file into a new engine
			reloadedEngine := New()
			reloadedSession, err := reloadedEngine.LoadSave(targetPath, string(platform), "local")
			if err != nil {
				t.Fatalf("reloaded LoadSave: %v", err)
			}

			profile, err := reloadedEngine.GetCharacterProfile(reloadedSession.SaveSessionID, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile: %v", err)
			}
			if profile.StartingClassID != 7 {
				t.Errorf("reloaded profile StartingClassID = %d, want 7", profile.StartingClassID)
			}
			if profile.Level != result.Level {
				t.Errorf("reloaded profile Level = %d, want %d", profile.Level, result.Level)
			}

			stats, err := reloadedEngine.GetCharacterStats(reloadedSession.SaveSessionID, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterStats: %v", err)
			}
			if stats.Vigor != result.Attributes.Vigor || stats.Level != result.Level {
				t.Errorf("reloaded stats = %+v, want attrs = %+v, level = %d", stats, result.Attributes, result.Level)
			}
		})
	}
}

func TestSetCharacterStartingClassTruncatedRange(t *testing.T) {
	engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
		platform:       PlatformPC,
		active:         true,
		withAnchor:     true,
		anchorAt:       setStartingClassTestAnchorAt,
		classID:        0,
		summaryClassID: 0,
		attributes:     CharacterAttributes{Vigor: 15, Mind: 10, Endurance: 11, Strength: 14, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
		level:          9,
		runes:          100,
		soulMemory:     100,
	})

	// Truncate snapshot data so reading the anchor/summary ranges fails
	engine.sessions[sessionID].snapshot.data = engine.sessions[sessionID].snapshot.data[:0x1000]

	_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 1, true, "0")
	if err == nil {
		t.Fatal("expected error on truncated snapshot")
	}
}

// setStartingClassRawField reads one uint32 of the addressed slot's
// PlayerGameData directly, for the two fields no public getter reports:
// TotalGetSoul and the held runes.
func setStartingClassRawField(
	t *testing.T, engine *Engine, sessionID string, platform Platform, offset int64,
) uint32 {
	t.Helper()
	at := setStartingClassTestSlotBase(platform) + setStartingClassTestAnchorAt + offset
	raw, err := engine.sessions[sessionID].snapshot.readAt(at, 4)
	if err != nil {
		t.Fatalf("read field at offset %d: %v", offset, err)
	}
	return binary.LittleEndian.Uint32(raw)
}

// The reset is destructive for the build, not irreversible inside the session.
// It has to leave the ordinary single undo point, and undoing it has to restore
// the exact prior character: the class, the level, all eight attributes, the
// lifetime runes and the held runes. The whole snapshot image is compared too,
// so a restore that reproduced the reported values while disturbing any other
// byte still fails.
func TestSetCharacterStartingClassLeavesAnUndoPointThatRestoresTheBuild(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			developed := CharacterAttributes{
				Vigor: 60, Mind: 55, Endurance: 50, Strength: 45,
				Dexterity: 40, Intelligence: 35, Faith: 30, Arcane: 25,
			}
			engine, session := loadSetStartingClassSession(t, setStartingClassTestContent{
				platform:       platform,
				active:         true,
				withAnchor:     true,
				anchorAt:       setStartingClassTestAnchorAt,
				classID:        setStartingClassTestVagabond,
				summaryClassID: setStartingClassTestVagabond,
				attributes:     developed,
				level:          261,
				runes:          777_777,
				soulMemory:     4_242_424,
			})

			imageBefore := bytes.Clone(engine.sessions[session].snapshot.data)
			statsBefore, err := engine.GetCharacterStats(session, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterStats before the reset: %v", err)
			}
			profileBefore, err := engine.GetCharacterProfile(session, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile before the reset: %v", err)
			}
			soulMemoryBefore := setStartingClassRawField(
				t, engine, session, platform, setStartingClassTestSoulMemoryOffset)
			runesBefore := setStartingClassRawField(
				t, engine, session, platform, setStartingClassTestRunesOffset)

			if _, err := engine.SetCharacterStartingClass(
				session, setStartingClassTestSlot, setStartingClassTestWretch, true, "0"); err != nil {
				t.Fatalf("SetCharacterStartingClass: %v", err)
			}

			state, err := engine.GetUndoState(session, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetUndoState: %v", err)
			}
			if !state.Available || state.UndoToken == "" {
				t.Fatalf("undo state after the reset = %+v, want an available point with a token", state)
			}
			if state.OperationID != opSetCharacterStartingClass {
				t.Errorf("operationID = %q, want %q", state.OperationID, opSetCharacterStartingClass)
			}
			if state.SaveRevision != "1" {
				t.Errorf("saveRevision of the undo point = %q, want 1", state.SaveRevision)
			}

			// The mutation really happened, so the restore below is a restore.
			if got := setStartingClassRawField(
				t, engine, session, platform, setStartingClassTestSoulMemoryOffset); got != soulMemoryBefore {
				t.Fatalf("the reset changed SoulMemory to %d, want the preserved %d", got, soulMemoryBefore)
			}
			if bytes.Equal(imageBefore, engine.sessions[session].snapshot.data) {
				t.Fatal("the reset changed nothing, so the undo below would prove nothing")
			}

			result, err := engine.UndoCharacterChanges(
				session, setStartingClassTestSlot, state.UndoToken, "1")
			if err != nil {
				t.Fatalf("UndoCharacterChanges: %v", err)
			}
			if result.UndoneOperationID != opSetCharacterStartingClass {
				t.Errorf("undoneOperationID = %q, want %q",
					result.UndoneOperationID, opSetCharacterStartingClass)
			}

			profileAfter, err := engine.GetCharacterProfile(session, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile after the undo: %v", err)
			}
			if profileAfter.StartingClassID != profileBefore.StartingClassID {
				t.Errorf("startingClassID after undo = %d, want the restored %d",
					profileAfter.StartingClassID, profileBefore.StartingClassID)
			}
			if profileAfter.Level != profileBefore.Level {
				t.Errorf("profile level after undo = %d, want the restored %d",
					profileAfter.Level, profileBefore.Level)
			}

			statsAfter, err := engine.GetCharacterStats(session, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterStats after the undo: %v", err)
			}
			if statsAfter != statsBefore {
				t.Errorf("stats after undo = %+v, want the restored %+v", statsAfter, statsBefore)
			}

			if got := setStartingClassRawField(
				t, engine, session, platform, setStartingClassTestSoulMemoryOffset); got != soulMemoryBefore {
				t.Errorf("SoulMemory after undo = %d, want the restored %d", got, soulMemoryBefore)
			}
			if got := setStartingClassRawField(
				t, engine, session, platform, setStartingClassTestRunesOffset); got != runesBefore {
				t.Errorf("held runes after undo = %d, want the restored %d", got, runesBefore)
			}
			if !bytes.Equal(imageBefore, engine.sessions[session].snapshot.data) {
				t.Error("the undo did not restore the snapshot byte for byte")
			}

			consumed, err := engine.GetUndoState(session, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetUndoState after the undo: %v", err)
			}
			if consumed.Available || consumed.UndoToken != "" {
				t.Errorf("undo state after the undo = %+v, want the point consumed", consumed)
			}
		})
	}
}
