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
		writeSetStartingClassFixture(t, content), string(content.platform))
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

func TestSetCharacterStartingClassNoCollision(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			// Character attributes are all 20, which is >= every class's minima.
			initialAttrs := CharacterAttributes{
				Vigor: 20, Mind: 20, Endurance: 20, Strength: 20,
				Dexterity: 20, Intelligence: 20, Faith: 20, Arcane: 20,
			}
			initialLevel := uint32(81) // 8*20 - 79 = 81
			initialSoulMemory := uint32(2_000_000)

			engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
				platform:       platform,
				active:         true,
				withAnchor:     true,
				anchorAt:       setStartingClassTestAnchorAt,
				classID:        byte(setStartingClassTestVagabond),
				summaryClassID: byte(setStartingClassTestVagabond),
				attributes:     initialAttrs,
				level:          initialLevel,
				runes:          5000,
				soulMemory:     initialSoulMemory,
			})

			before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

			// Change to Astrologer (class 4). Astrologer base attributes are <= 20 across all 8 stats.
			result, err := engine.SetCharacterStartingClass(
				sessionID, setStartingClassTestSlot, uint8(setStartingClassTestAstrologer), "0")
			if err != nil {
				t.Fatalf("SetCharacterStartingClass: %v", err)
			}

			if result.SaveSessionID != sessionID || result.SaveRevision != "1" ||
				result.CharacterID != setStartingClassTestSlot ||
				result.StartingClassID != uint8(setStartingClassTestAstrologer) ||
				result.Attributes != initialAttrs ||
				result.Level != initialLevel ||
				result.SoulMemory != initialSoulMemory ||
				result.AttributesRaised {
				t.Fatalf("result = %+v, want no-collision result", result)
			}

			after := engine.sessions[sessionID].snapshot.data
			anchor := setStartingClassTestSlotBase(platform) + setStartingClassTestAnchorAt
			summary := setStartingClassTestSummaryAt(platform)

			// Exactly the two class bytes should differ
			for i := range before {
				if i == int(anchor+setStartingClassTestClassOffset) || i == int(summary+setStartingClassTestSummaryClass) {
					if after[i] != byte(setStartingClassTestAstrologer) {
						t.Errorf("byte at %d = %d, want %d", i, after[i], setStartingClassTestAstrologer)
					}
				} else {
					if after[i] != before[i] {
						t.Fatalf("unexpected snapshot byte change at offset %d: was %d, now %d", i, before[i], after[i])
					}
				}
			}

			// Verify GetCharacterProfile and GetCharacterStats
			profile, err := engine.GetCharacterProfile(sessionID, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile: %v", err)
			}
			if profile.StartingClassID != uint8(setStartingClassTestAstrologer) {
				t.Errorf("profile.StartingClassID = %d, want %d", profile.StartingClassID, setStartingClassTestAstrologer)
			}

			stats, err := engine.GetCharacterStats(sessionID, setStartingClassTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterStats: %v", err)
			}
			if stats.Vigor != 20 || stats.Level != initialLevel {
				t.Errorf("stats = %+v", stats)
			}
		})
	}
}

func TestSetCharacterStartingClassCollision(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			// Hero starting attributes: Vigor 14, Mind 9, Endurance 12, Strength 16, Dexterity 9, Intelligence 7, Faith 8, Arcane 11 (Level 7)
			initialAttrs := CharacterAttributes{
				Vigor: 14, Mind: 9, Endurance: 12, Strength: 16,
				Dexterity: 9, Intelligence: 7, Faith: 8, Arcane: 11,
			}
			initialLevel := uint32(7)
			initialSoulMemory := uint32(100)

			engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
				platform:       platform,
				active:         true,
				withAnchor:     true,
				anchorAt:       setStartingClassTestAnchorAt,
				classID:        byte(setStartingClassTestHero),
				summaryClassID: byte(setStartingClassTestHero),
				attributes:     initialAttrs,
				level:          initialLevel,
				runes:          1000,
				soulMemory:     initialSoulMemory,
			})

			// Change to Astrologer (class 4):
			// Astrologer base: Vigor 9, Mind 15, Endurance 9, Strength 8, Dexterity 12, Intelligence 16, Faith 7, Arcane 9
			// Collisions:
			// Vigor: max(14, 9) = 14 (untouched)
			// Mind: max(9, 15) = 15 (raised from 9)
			// Endurance: max(12, 9) = 12 (untouched)
			// Strength: max(16, 8) = 16 (untouched)
			// Dexterity: max(9, 12) = 12 (raised from 9)
			// Intelligence: max(7, 16) = 16 (raised from 7)
			// Faith: max(8, 7) = 8 (untouched)
			// Arcane: max(11, 9) = 11 (untouched)
			// Resulting attributes: [14, 15, 12, 16, 12, 16, 8, 11]
			// Sum = 14 + 15 + 12 + 16 + 12 + 16 + 8 + 11 = 104
			// Recalculated level = 104 - 79 = 25.
			// Required SoulMemory for level 25 is minimumSoulMemoryForLevel(25).
			wantAttrs := CharacterAttributes{
				Vigor: 14, Mind: 15, Endurance: 12, Strength: 16,
				Dexterity: 12, Intelligence: 16, Faith: 8, Arcane: 11,
			}
			wantLevel := uint32(25)
			wantSoulMemory := minimumSoulMemoryForLevel(wantLevel)

			result, err := engine.SetCharacterStartingClass(
				sessionID, setStartingClassTestSlot, uint8(setStartingClassTestAstrologer), "0")
			if err != nil {
				t.Fatalf("SetCharacterStartingClass: %v", err)
			}

			if !result.AttributesRaised {
				t.Errorf("result.AttributesRaised = false, want true")
			}
			if result.Attributes != wantAttrs {
				t.Errorf("result.Attributes = %+v, want %+v", result.Attributes, wantAttrs)
			}
			if result.Level != wantLevel {
				t.Errorf("result.Level = %d, want %d", result.Level, wantLevel)
			}
			if result.SoulMemory != wantSoulMemory {
				t.Errorf("result.SoulMemory = %d, want %d", result.SoulMemory, wantSoulMemory)
			}
			if result.StartingClassID != uint8(setStartingClassTestAstrologer) {
				t.Errorf("result.StartingClassID = %d, want %d", result.StartingClassID, setStartingClassTestAstrologer)
			}

			// Verify PlayerGameData class and ProfileSummary class both equal 4
			anchor := setStartingClassTestSlotBase(platform) + setStartingClassTestAnchorAt
			summary := setStartingClassTestSummaryAt(platform)
			data := engine.sessions[sessionID].snapshot.data
			if data[anchor+setStartingClassTestClassOffset] != byte(setStartingClassTestAstrologer) {
				t.Errorf("PlayerGameData class byte = %d, want %d",
					data[anchor+setStartingClassTestClassOffset], setStartingClassTestAstrologer)
			}
			if data[summary+setStartingClassTestSummaryClass] != byte(setStartingClassTestAstrologer) {
				t.Errorf("ProfileSummary class byte = %d, want %d",
					data[summary+setStartingClassTestSummaryClass], setStartingClassTestAstrologer)
			}
			// Verify ProfileSummary level
			summaryLevel := binary.LittleEndian.Uint32(data[summary+setStartingClassTestSummaryLevel:])
			if summaryLevel != wantLevel {
				t.Errorf("ProfileSummary level = %d, want %d", summaryLevel, wantLevel)
			}
		})
	}
}

func TestSetCharacterStartingClassSoulMemoryNotLowered(t *testing.T) {
	// Character already has high SoulMemory.
	initialAttrs := CharacterAttributes{
		Vigor: 15, Mind: 10, Endurance: 11, Strength: 14,
		Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7,
	}
	highSoulMemory := uint32(999_999)

	engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
		platform:       PlatformPC,
		active:         true,
		withAnchor:     true,
		anchorAt:       setStartingClassTestAnchorAt,
		classID:        byte(setStartingClassTestVagabond),
		summaryClassID: byte(setStartingClassTestVagabond),
		attributes:     initialAttrs,
		level:          9,
		runes:          1000,
		soulMemory:     highSoulMemory,
	})

	// Changing to Hero (class 2)
	result, err := engine.SetCharacterStartingClass(
		sessionID, setStartingClassTestSlot, uint8(setStartingClassTestHero), "0")
	if err != nil {
		t.Fatalf("SetCharacterStartingClass: %v", err)
	}

	if result.SoulMemory != highSoulMemory {
		t.Errorf("result.SoulMemory = %d, want untouched %d", result.SoulMemory, highSoulMemory)
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

		_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 10, "0")
		if err == nil || err.Error() != "starting class 10 is unknown; its attribute minima are not confirmed" {
			t.Fatalf("error = %v, want unknown starting class error", err)
		}
		assertSetStartingClassRejectedUnchanged(t, engine, sessionID, before)
	})

	// A stored attribute outside 1..99 can only come from an earlier external
	// edit. Carrying it forward would let this mutation bless a state
	// SetCharacterStats rejects, so the class change fails closed instead. The
	// range itself is enforced once, inside recalculateCharacterLevel, which
	// this mutation shares with SetCharacterStats; these cases pin that the
	// shared rule is actually reached through the class-change path. The raise
	// to the class minima happens first, so only the upper bound is reachable
	// here: a stored value below the minimum is lifted into range.
	t.Run("stored attribute above the confirmed maximum", func(t *testing.T) {
		engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
			platform:       PlatformPC,
			active:         true,
			withAnchor:     true,
			anchorAt:       setStartingClassTestAnchorAt,
			classID:        setStartingClassTestVagabond,
			summaryClassID: setStartingClassTestVagabond,
			attributes:     CharacterAttributes{Vigor: 15, Mind: 10, Endurance: 11, Strength: 150, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
			level:          144,
			runes:          100,
			soulMemory:     100,
		})
		before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

		_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, uint8(setStartingClassTestAstrologer), "0")
		if err == nil {
			t.Fatal("a stored attribute of 150 was accepted, but the confirmed range is 1..99")
		}
		if !strings.Contains(err.Error(), "attributes.strength 150") {
			t.Errorf("error = %v, want it to name the rejected attribute and value", err)
		}
		assertSetStartingClassRejectedUnchanged(t, engine, sessionID, before)
	})

	t.Run("stored attribute exactly at the confirmed maximum is accepted", func(t *testing.T) {
		engine, sessionID := loadSetStartingClassSession(t, setStartingClassTestContent{
			platform:       PlatformPC,
			active:         true,
			withAnchor:     true,
			anchorAt:       setStartingClassTestAnchorAt,
			classID:        setStartingClassTestVagabond,
			summaryClassID: setStartingClassTestVagabond,
			attributes:     CharacterAttributes{Vigor: 15, Mind: 10, Endurance: 11, Strength: 99, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},
			level:          94,
			runes:          100,
			soulMemory:     100,
		})

		result, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, uint8(setStartingClassTestAstrologer), "0")
		if err != nil {
			t.Fatalf("SetCharacterStartingClass: %v", err)
		}
		if result.Attributes.Strength != 99 {
			t.Errorf("attributes.strength = %d, want it kept at the maximum 99", result.Attributes.Strength)
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
			_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 1, badRev)
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

		_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 1, "0")
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
			_, err := engine.SetCharacterStartingClass(sessionID, badSlot, 1, "0")
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

		_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 1, "0")
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
			loaded, err := engine.LoadSave(source, string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			// Mutate to Samurai (class 7)
			result, err := engine.SetCharacterStartingClass(
				loaded.SaveSessionID, setStartingClassTestSlot, 7, "0")
			if err != nil {
				t.Fatalf("SetCharacterStartingClass: %v", err)
			}

			targetPath := filepath.Join(t.TempDir(), "persisted.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", targetPath); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}

			// Reload written file into a new engine
			reloadedEngine := New()
			reloadedSession, err := reloadedEngine.LoadSave(targetPath, string(platform))
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

	_, err := engine.SetCharacterStartingClass(sessionID, setStartingClassTestSlot, 1, "0")
	if err == nil {
		t.Fatal("expected error on truncated snapshot")
	}
}
