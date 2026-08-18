package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// The offsets are restated literally instead of reused from the implementation,
// so a changed base, stride or field offset fails here.
const (
	setStatsTestSlot         = 3
	setStatsTestAnchorAt     = int64(0x0640)
	setStatsTestFullAnchorAt = int64(0x20 + 5120*8 + 0x1B0)

	setStatsTestPCSlotBase    = int64(0x310)
	setStatsTestPCSlotStride  = int64(0x280010)
	setStatsTestPS4SlotBase   = int64(0x70)
	setStatsTestPS4SlotStride = int64(0x280000)

	setStatsTestSummaryOffset = int64(0x195E)
	setStatsTestSummaryStride = int64(0x24C)
	setStatsTestSummaryLevel  = int64(0x22)
	setStatsTestSummaryClass  = int64(0x243)

	setStatsTestVigorOffset      = int64(-379)
	setStatsTestLevelOffset      = int64(-335)
	setStatsTestRunesOffset      = int64(-331)
	setStatsTestSoulMemoryOffset = int64(-327)
	setStatsTestClassOffset      = int64(-248)

	setStatsTestVagabond = 0
	setStatsTestWretch   = 9
)

// setStatsTestAttributes is a legal Vagabond assignment: every value is at or
// above the Vagabond base and the sum recalculates to level 44.
var setStatsTestAttributes = CharacterAttributes{
	Vigor: 20, Mind: 15, Endurance: 16, Strength: 20,
	Dexterity: 18, Intelligence: 12, Faith: 12, Arcane: 10,
}

const (
	setStatsTestLevel              = uint32(44)
	setStatsTestRequiredSoulMemory = uint32(177_486)
)

// classID is the authoritative PlayerGameData class the mutation validates
// against; summaryClassID is the ProfileSummary copy, which the mutation must
// ignore. They are separate fields so a stale summary can be reproduced.
type setStatsTestContent struct {
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

func setStatsTestSlotBase(platform Platform) int64 {
	if platform == PlatformPS4 {
		return setStatsTestPS4SlotBase + setStatsTestSlot*setStatsTestPS4SlotStride
	}
	return setStatsTestPCSlotBase + setStatsTestSlot*setStatsTestPCSlotStride
}

func setStatsTestSummaryAt(platform Platform) int64 {
	base := pcUserData10DataOffset
	if platform == PlatformPS4 {
		base = ps4UserData10DataOffset
	}
	return base + setStatsTestSummaryOffset + setStatsTestSlot*setStatsTestSummaryStride
}

// writeSetStatsFixture builds a synthetic save inside t.TempDir() whose one slot
// carries the requested activity, anchor, both starting-class copies, attributes,
// level, held runes and TotalGetSoul. No real save file is opened or modified.
func writeSetStatsFixture(t *testing.T, content setStatsTestContent) string {
	t.Helper()

	fixture := statsFixture{
		platform: content.platform,
		slot:     setStatsTestSlot,
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

	summary := setStatsTestSummaryAt(content.platform)
	data[summary+setStatsTestSummaryClass] = content.summaryClassID
	binary.LittleEndian.PutUint32(data[summary+setStatsTestSummaryLevel:], content.level)

	if content.withAnchor {
		anchor := setStatsTestSlotBase(content.platform) + content.anchorAt
		data[anchor+setStatsTestClassOffset] = content.classID
		binary.LittleEndian.PutUint32(data[anchor+setStatsTestRunesOffset:], content.runes)
		binary.LittleEndian.PutUint32(data[anchor+setStatsTestSoulMemoryOffset:], content.soulMemory)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	return path
}

// setStatsTestActiveContent is an active Vagabond slot that already carries the
// requested attributes' predecessor values and needs its SoulMemory raised.
func setStatsTestActiveContent(platform Platform) setStatsTestContent {
	return setStatsTestContent{
		platform:       platform,
		active:         true,
		withAnchor:     true,
		anchorAt:       setStatsTestAnchorAt,
		classID:        setStatsTestVagabond,
		summaryClassID: setStatsTestVagabond,
		attributes: CharacterAttributes{
			Vigor: 15, Mind: 10, Endurance: 11, Strength: 14,
			Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7,
		},
		level:      9,
		runes:      4_242,
		soulMemory: 5_000,
	}
}

func loadSetStatsSession(t *testing.T, content setStatsTestContent) (*Engine, string) {
	t.Helper()
	engine := New()
	loaded, err := engine.LoadSave(
		writeSetStatsFixture(t, content), string(content.platform))
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

func assertSetStatsRejectedUnchanged(t *testing.T, engine *Engine, sessionID string, before []byte) {
	t.Helper()
	if !bytes.Equal(before, engine.sessions[sessionID].snapshot.data) {
		t.Error("rejected statistics mutation changed the private snapshot")
	}
	if revision := engine.sessions[sessionID].session.revisionString(); revision != "0" {
		t.Errorf("revision after rejection = %q, want 0", revision)
	}
	if engine.sessions[sessionID].session.dirty {
		t.Error("rejected statistics mutation marked the session dirty")
	}
}

// expectedSetStatsSnapshot returns the only bytes a successful mutation may
// change: the eight attributes, the PlayerGameData level, TotalGetSoul and the
// ProfileSummary level. Everything else must stay identical.
func expectedSetStatsSnapshot(
	before []byte,
	platform Platform,
	anchorAt int64,
	attributes CharacterAttributes,
	level uint32,
	soulMemory uint32,
) []byte {
	expected := bytes.Clone(before)
	anchor := setStatsTestSlotBase(platform) + anchorAt
	ordered := []uint32{
		attributes.Vigor, attributes.Mind, attributes.Endurance, attributes.Strength,
		attributes.Dexterity, attributes.Intelligence, attributes.Faith, attributes.Arcane,
	}
	for index, value := range ordered {
		binary.LittleEndian.PutUint32(
			expected[anchor+setStatsTestVigorOffset+int64(index)*4:], value)
	}
	binary.LittleEndian.PutUint32(expected[anchor+setStatsTestLevelOffset:], level)
	binary.LittleEndian.PutUint32(expected[anchor+setStatsTestSoulMemoryOffset:], soulMemory)
	binary.LittleEndian.PutUint32(
		expected[setStatsTestSummaryAt(platform)+setStatsTestSummaryLevel:], level)
	return expected
}

func TestSetCharacterStatsWritesTheConfirmedFieldsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			engine, sessionID := loadSetStatsSession(t, setStatsTestActiveContent(platform))
			before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

			result, err := engine.SetCharacterStats(
				sessionID, setStatsTestSlot, setStatsTestAttributes, LevelPolicyRecalculate, "0")
			if err != nil {
				t.Fatalf("SetCharacterStats: %v", err)
			}

			want := SetCharacterStatsResult{
				SaveSessionID: sessionID,
				SaveRevision:  "1",
				CharacterID:   setStatsTestSlot,
				Attributes:    setStatsTestAttributes,
				Level:         setStatsTestLevel,
				SoulMemory:    setStatsTestRequiredSoulMemory,
			}
			if result != want {
				t.Errorf("result = %+v, want %+v", result, want)
			}

			expected := expectedSetStatsSnapshot(before, platform, setStatsTestAnchorAt,
				setStatsTestAttributes, setStatsTestLevel, setStatsTestRequiredSoulMemory)
			if !bytes.Equal(engine.sessions[sessionID].snapshot.data, expected) {
				t.Error("mutation changed bytes outside the attributes, level, TotalGetSoul and summary level")
			}
			if !engine.sessions[sessionID].session.dirty {
				t.Error("successful statistics mutation did not mark the session dirty")
			}
		})
	}
}

func TestSetCharacterStatsKeepsASufficientSoulMemory(t *testing.T) {
	content := setStatsTestActiveContent(PlatformPC)
	content.soulMemory = setStatsTestRequiredSoulMemory + 1
	engine, sessionID := loadSetStatsSession(t, content)

	result, err := engine.SetCharacterStats(
		sessionID, setStatsTestSlot, setStatsTestAttributes, LevelPolicyRecalculate, "0")
	if err != nil {
		t.Fatalf("SetCharacterStats: %v", err)
	}
	if result.SoulMemory != content.soulMemory {
		t.Errorf("soulMemory = %d, want the stored %d left untouched",
			result.SoulMemory, content.soulMemory)
	}

	anchor := setStatsTestSlotBase(PlatformPC) + setStatsTestAnchorAt
	stored, err := engine.sessions[sessionID].snapshot.uint32At(anchor + setStatsTestSoulMemoryOffset)
	if err != nil {
		t.Fatalf("read TotalGetSoul: %v", err)
	}
	if stored != content.soulMemory {
		t.Errorf("stored TotalGetSoul = %d, want %d", stored, content.soulMemory)
	}
}

func TestSetCharacterStatsAcceptsTheMaximumAttributesAndLevel(t *testing.T) {
	engine, sessionID := loadSetStatsSession(t, setStatsTestActiveContent(PlatformPC))

	maximum := CharacterAttributes{
		Vigor: 99, Mind: 99, Endurance: 99, Strength: 99,
		Dexterity: 99, Intelligence: 99, Faith: 99, Arcane: 99,
	}
	result, err := engine.SetCharacterStats(
		sessionID, setStatsTestSlot, maximum, LevelPolicyRecalculate, "0")
	if err != nil {
		t.Fatalf("SetCharacterStats: %v", err)
	}
	if result.Level != 713 || result.SoulMemory != 1_692_560_963 {
		t.Errorf("result = %+v, want level 713 and soulMemory 1692560963", result)
	}
}

func TestSetCharacterStatsRejectsIllegalAttributesAndPolicies(t *testing.T) {
	below := setStatsTestAttributes
	below.Vigor = 14 // one below the Vagabond base
	zero := setStatsTestAttributes
	zero.Mind = 0
	above := setStatsTestAttributes
	above.Arcane = 100
	minimal := CharacterAttributes{
		Vigor: 1, Mind: 1, Endurance: 1, Strength: 1,
		Dexterity: 1, Intelligence: 1, Faith: 1, Arcane: 1,
	}

	cases := map[string]struct {
		attributes  CharacterAttributes
		levelPolicy string
		want        string
	}{
		"below class minimum": {below, LevelPolicyRecalculate,
			"attributes.vigor 14 is below the starting-class minimum 15"},
		"attribute zero": {zero, LevelPolicyRecalculate,
			"attributes.mind 0 is outside the range 1..99"},
		"attribute above maximum": {above, LevelPolicyRecalculate,
			"attributes.arcane 100 is outside the range 1..99"},
		"level below the legal range": {minimal, LevelPolicyRecalculate,
			"recalculated level -71 is outside the range 1..713"},
		"unknown level policy": {setStatsTestAttributes, "preserve",
			`levelPolicy must be "recalculate"; got "preserve"`},
		"padded level policy": {setStatsTestAttributes, " recalculate",
			`levelPolicy must be "recalculate"; got " recalculate"`},
	}

	engine, sessionID := loadSetStatsSession(t, setStatsTestActiveContent(PlatformPC))
	before := bytes.Clone(engine.sessions[sessionID].snapshot.data)
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := engine.SetCharacterStats(
				sessionID, setStatsTestSlot, testCase.attributes, testCase.levelPolicy, "0")
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
	assertSetStatsRejectedUnchanged(t, engine, sessionID, before)
}

func TestSetCharacterStatsRejectsAnUnknownStartingClass(t *testing.T) {
	content := setStatsTestActiveContent(PlatformPC)
	content.classID = 10
	content.summaryClassID = 10
	engine, sessionID := loadSetStatsSession(t, content)
	before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

	_, err := engine.SetCharacterStats(
		sessionID, setStatsTestSlot, setStatsTestAttributes, LevelPolicyRecalculate, "0")
	want := "starting class 10 is unknown; its attribute minima are not confirmed"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	assertSetStatsRejectedUnchanged(t, engine, sessionID, before)
}

// TestSetCharacterStatsValidatesTheMinimaAgainstPlayerGameDataClass pins the
// class source. The slot's PlayerGameData carries Vagabond, whose vigor base is
// 15, while its ProfileSummary carries the stale Wretch copy, whose base is 10.
// A vigor of 10 is legal for the summary copy and illegal for the real class, so
// it must be rejected without touching the snapshot, revision or dirty state.
func TestSetCharacterStatsValidatesTheMinimaAgainstPlayerGameDataClass(t *testing.T) {
	content := setStatsTestActiveContent(PlatformPC)
	content.classID = setStatsTestVagabond
	content.summaryClassID = setStatsTestWretch
	engine, sessionID := loadSetStatsSession(t, content)
	before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

	attributes := setStatsTestAttributes
	attributes.Vigor = 10

	_, err := engine.SetCharacterStats(
		sessionID, setStatsTestSlot, attributes, LevelPolicyRecalculate, "0")
	want := "attributes.vigor 10 is below the starting-class minimum 15"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	assertSetStatsRejectedUnchanged(t, engine, sessionID, before)
}

func TestSetCharacterStatsRejectsInvalidSessionSlotRevisionSlotStateAndAnchor(t *testing.T) {
	engine, sessionID := loadSetStatsSession(t, setStatsTestActiveContent(PlatformPC))
	before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

	cases := map[string]struct {
		sessionID        string
		characterID      int
		expectedRevision string
		want             string
	}{
		"empty session":   {"", setStatsTestSlot, "0", "saveSessionID is required"},
		"unknown session": {"missing", setStatsTestSlot, "0", `unknown save session "missing"`},
		"character below": {sessionID, -1, "0", "characterID -1 is outside the range 0..9"},
		"character above": {sessionID, 10, "0", "characterID 10 is outside the range 0..9"},
		"noncanonical revision": {sessionID, setStatsTestSlot, "00",
			`expectedRevision must be a canonical decimal saveRevision; got "00"`},
		"stale revision": {sessionID, setStatsTestSlot, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`},
		"inactive slot": {sessionID, 5, "0", "character 5 is not active"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := engine.SetCharacterStats(testCase.sessionID, testCase.characterID,
				setStatsTestAttributes, LevelPolicyRecalculate, testCase.expectedRevision)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
		})
	}
	assertSetStatsRejectedUnchanged(t, engine, sessionID, before)

	missingAnchor := setStatsTestActiveContent(PlatformPS4)
	missingAnchor.withAnchor = false
	anchorEngine, anchorSession := loadSetStatsSession(t, missingAnchor)
	anchorBefore := bytes.Clone(anchorEngine.sessions[anchorSession].snapshot.data)
	_, err := anchorEngine.SetCharacterStats(anchorSession, setStatsTestSlot,
		setStatsTestAttributes, LevelPolicyRecalculate, "0")
	if err == nil || err.Error() != "character 3 carries no statistics anchor" {
		t.Fatalf("error = %v, want missing-anchor error", err)
	}
	assertSetStatsRejectedUnchanged(t, anchorEngine, anchorSession, anchorBefore)
}

func TestSetCharacterStatsIdempotentAssignmentAdvancesRevision(t *testing.T) {
	content := setStatsTestActiveContent(PlatformPC)
	content.classID = setStatsTestWretch
	content.summaryClassID = setStatsTestWretch
	content.attributes = setStatsTestAttributes
	content.level = setStatsTestLevel
	content.soulMemory = setStatsTestRequiredSoulMemory
	engine, sessionID := loadSetStatsSession(t, content)
	before := bytes.Clone(engine.sessions[sessionID].snapshot.data)

	result, err := engine.SetCharacterStats(
		sessionID, setStatsTestSlot, setStatsTestAttributes, LevelPolicyRecalculate, "0")
	if err != nil {
		t.Fatalf("SetCharacterStats: %v", err)
	}
	if result.SaveRevision != "1" {
		t.Errorf("saveRevision = %q, want 1", result.SaveRevision)
	}
	if !bytes.Equal(before, engine.sessions[sessionID].snapshot.data) {
		t.Error("idempotent assignment changed the snapshot")
	}
}

func TestSetCharacterStatsPersistsAndReloadsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := gestureTestActiveFixture(
				platform, setStatsTestSlot, setStatsTestFullAnchorAt, 0)
			content.records = setGestureTestRecords()
			source := writeGestureFixture(t, content)
			data, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read full fixture: %v", err)
			}
			binary.LittleEndian.PutUint32(data[setStatsTestSlotBase(platform):], 0x6E)
			if err := os.WriteFile(source, data, 0o600); err != nil {
				t.Fatalf("rewrite full fixture: %v", err)
			}

			engine := New()
			loaded, err := engine.LoadSave(source, string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			if _, err := engine.SetCharacterStats(loaded.SaveSessionID, setStatsTestSlot,
				setStatsTestAttributes, LevelPolicyRecalculate, "0"); err != nil {
				t.Fatalf("SetCharacterStats: %v", err)
			}

			target := filepath.Join(t.TempDir(), "stats.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform))
			if err != nil {
				t.Fatalf("reload target: %v", err)
			}

			stats, err := reloadedEngine.GetCharacterStats(reloaded.SaveSessionID, setStatsTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterStats after reload: %v", err)
			}
			if stats.Vigor != setStatsTestAttributes.Vigor ||
				stats.Arcane != setStatsTestAttributes.Arcane ||
				stats.Level != setStatsTestLevel {
				t.Errorf("reloaded statistics = %+v, want the committed attributes at level %d",
					stats, setStatsTestLevel)
			}
			profile, err := reloadedEngine.GetCharacterProfile(reloaded.SaveSessionID, setStatsTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile after reload: %v", err)
			}
			if profile.Level != setStatsTestLevel {
				t.Errorf("reloaded summary level = %d, want %d", profile.Level, setStatsTestLevel)
			}
		})
	}
}

// TestMinimumSoulMemoryForLevelReproducesTheConfirmedVectors pins the results of
// the deterministic integer evaluation SaveForge 1.6.8 introduced and covered by
// its own reference tests, so this reimplementation cannot drift from it. 1.5.8
// summed the same per-level cost in float64, so its results are not a reference:
// they depend on the host's floating-point behaviour and may differ.
func TestMinimumSoulMemoryForLevelReproducesTheConfirmedVectors(t *testing.T) {
	for _, testCase := range []struct {
		level uint32
		want  uint32
	}{
		{1, 0},
		{9, 473},
		{44, 177_486},
		{50, 256_598},
		{150, 7_106_585},
		{713, 1_692_560_963},
	} {
		if got := minimumSoulMemoryForLevel(testCase.level); got != testCase.want {
			t.Errorf("minimumSoulMemoryForLevel(%d) = %d, want %d",
				testCase.level, got, testCase.want)
		}
	}
}

func TestPlanCharacterStats_ReadOnly(t *testing.T) {
	savePath := writeSetStatsFixture(t, setStatsTestContent{
		platform:   PlatformPC,
		active:     true,
		withAnchor: true,
		anchorAt:   setStatsTestAnchorAt,
		classID:    setStatsTestVagabond,
		attributes: CharacterAttributes{
			Vigor: 15, Mind: 10, Endurance: 11, Strength: 14,
			Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7,
		},
		level:      9,
		soulMemory: 473,
	})

	engine := New()
	loaded, err := engine.LoadSave(savePath, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	engine.mutex.Lock()
	sessionBefore := engine.sessions[loaded.SaveSessionID]
	snapshotBefore := bytes.Clone(sessionBefore.snapshot.data)
	revBefore := sessionBefore.session.revisionString()
	dirtyBefore := sessionBefore.session.dirty
	undoBefore := sessionBefore.session.undo
	ownedSeqBefore := sessionBefore.session.ownedSeq
	ownedByIDCountBefore := len(sessionBefore.session.ownedByID)
	ownedByLocCountBefore := len(sessionBefore.session.ownedByLocator)
	engine.mutex.Unlock()

	level, soulMemory, err := engine.PlanCharacterStats(
		loaded.SaveSessionID, setStatsTestSlot, setStatsTestAttributes)
	if err != nil {
		t.Fatalf("PlanCharacterStats: %v", err)
	}
	if level != setStatsTestLevel {
		t.Errorf("level = %d, want %d", level, setStatsTestLevel)
	}
	if soulMemory != setStatsTestRequiredSoulMemory {
		t.Errorf("soulMemory = %d, want %d", soulMemory, setStatsTestRequiredSoulMemory)
	}

	engine.mutex.Lock()
	sessionAfter := engine.sessions[loaded.SaveSessionID]
	snapshotAfter := sessionAfter.snapshot.data
	revAfter := sessionAfter.session.revisionString()
	dirtyAfter := sessionAfter.session.dirty
	undoAfter := sessionAfter.session.undo
	ownedSeqAfter := sessionAfter.session.ownedSeq
	ownedByIDCountAfter := len(sessionAfter.session.ownedByID)
	ownedByLocCountAfter := len(sessionAfter.session.ownedByLocator)
	engine.mutex.Unlock()

	if !bytes.Equal(snapshotBefore, snapshotAfter) {
		t.Error("PlanCharacterStats mutated snapshot bytes")
	}
	if revBefore != revAfter {
		t.Errorf("revision mutated from %q to %q", revBefore, revAfter)
	}
	if dirtyBefore != dirtyAfter {
		t.Errorf("dirty changed from %v to %v", dirtyBefore, dirtyAfter)
	}
	if undoBefore != undoAfter {
		t.Errorf("undo pointer changed from %p to %p", undoBefore, undoAfter)
	}
	if ownedSeqBefore != ownedSeqAfter {
		t.Errorf("ownedSeq changed from %d to %d", ownedSeqBefore, ownedSeqAfter)
	}
	if ownedByIDCountBefore != ownedByIDCountAfter {
		t.Errorf("ownedByID count changed from %d to %d", ownedByIDCountBefore, ownedByIDCountAfter)
	}
	if ownedByLocCountBefore != ownedByLocCountAfter {
		t.Errorf("ownedByLocator count changed from %d to %d", ownedByLocCountBefore, ownedByLocCountAfter)
	}
}

// TestLegalAttributesFor covers the rule application GetRepairPlan derives a
// corrected attribute set from. Both bounds and the class minimum are exercised
// together, because the class minimum is the stricter one wherever they overlap
// and a correction that applied only the range would still be rejected by
// SetCharacterStats.
func TestLegalAttributesFor(t *testing.T) {
	// Vagabond, class 0, minima {15, 10, 11, 14, 13, 9, 9, 7}.
	const vagabond = uint8(0)

	t.Run("a legal set is returned unchanged", func(t *testing.T) {
		legal := CharacterAttributes{
			Vigor: 20, Mind: 30, Endurance: 11, Strength: 14,
			Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7,
		}
		got, err := LegalAttributesFor(legal, vagabond)
		if err != nil {
			t.Fatalf("LegalAttributesFor: %v", err)
		}
		if got != legal {
			t.Errorf("a legal set was altered: %+v -> %+v", legal, got)
		}
	})

	t.Run("each attribute moves the smallest legal distance", func(t *testing.T) {
		got, err := LegalAttributesFor(CharacterAttributes{
			Vigor: 0, Mind: 200, Endurance: 40, Strength: 12,
			Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7,
		}, vagabond)
		if err != nil {
			t.Fatalf("LegalAttributesFor: %v", err)
		}
		want := CharacterAttributes{
			// 0 is below both bounds; the class minimum 15 is the stricter one.
			Vigor: 15,
			// 200 is above the absolute maximum.
			Mind: 99,
			// 40 is legal on both rules and must not move.
			Endurance: 40,
			// 12 is inside 1..99 but below the class minimum 14.
			Strength:  14,
			Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7,
		}
		if got != want {
			t.Errorf("LegalAttributesFor = %+v, want %+v", got, want)
		}
	})

	t.Run("the result is always writable by SetCharacterStats", func(t *testing.T) {
		// Every confirmed class sums to at least 80, so a corrected set can never
		// produce a level below the minimum 1. This is the invariant that lets a
		// plan promise the executing endpoint will accept its target.
		for class := uint8(0); class < 10; class++ {
			got, err := LegalAttributesFor(CharacterAttributes{}, class)
			if err != nil {
				t.Fatalf("class %d: LegalAttributesFor: %v", class, err)
			}
			sum := got.Vigor + got.Mind + got.Endurance + got.Strength +
				got.Dexterity + got.Intelligence + got.Faith + got.Arcane
			if sum < 80 {
				t.Errorf("class %d minima sum to %d, which produces a level below 1", class, sum)
			}
		}
	})

	t.Run("an unknown starting class is rejected", func(t *testing.T) {
		if _, err := LegalAttributesFor(CharacterAttributes{
			Vigor: 20, Mind: 20, Endurance: 20, Strength: 20,
			Dexterity: 20, Intelligence: 20, Faith: 20, Arcane: 20,
		}, 10); err == nil {
			t.Error("class 10 was accepted, but it carries no confirmed minima")
		}
	})
}
