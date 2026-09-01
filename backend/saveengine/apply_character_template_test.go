package saveengine

import (
	"bytes"
	"encoding/binary"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"unicode/utf16"
)

const (
	applyTplTestSlot                    = 0
	applyTplTestPCSlotBase              = int64(0x310)
	applyTplTestPCSlotStride            = int64(0x280010)
	applyTplTestPS4SlotBase             = int64(0x70)
	applyTplTestPS4SlotStride           = int64(0x280000)
	applyTplTestPCUserData10Base        = int64(0x19003B0)
	applyTplTestPS4UserData10Base       = int64(0x1900070)
	applyTplTestActiveFlagsOffset       = int64(0x1954)
	applyTplTestActiveFlagValue         = byte(1)
	applyTplTestSummaryOffset           = int64(0x195E)
	applyTplTestSummaryStride           = int64(0x24C)
	applyTplTestSummaryNameOffset       = int64(0x00)
	applyTplTestSummaryLevelOffset      = int64(0x22)
	applyTplTestOrdinaryAnchorAt        = int64(0xB000)
	applyTplTestPlayerNameOffset        = int64(-0x11B)
	applyTplTestStatsClassOffset        = int64(-248)
	applyTplTestStatsVigorOffset        = int64(-379)
	applyTplTestStatsLevelOffset        = int64(-335)
	applyTplTestStatsTotalGetSoulOffset = int64(-327)
	applyTplTestSpellsSectionOffset     = int64(0x9205)
	applyTplTestTalismanSlotsOffset     = int64(-241)
	applyTplTestSpellEmptyID            = uint32(0xFFFFFFFF)
	applyTplTestSpellEmptyFollower      = uint32(0x00000000)
	applyTplTestSpellOccFollower        = uint32(0xFFFFFFFF)
)

var applyTplTestVagabondAttrs = [8]uint32{15, 10, 11, 14, 13, 9, 9, 7}

var applyTplTestAnchor = []byte{
	0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

type applyTplByteRange struct {
	start int64
	end   int64
}

func applyTplInAllowedRanges(offset int64, ranges []applyTplByteRange) bool {
	for _, r := range ranges {
		if offset >= r.start && offset < r.end {
			return true
		}
	}
	return false
}

func writeApplyTemplateFixture(t *testing.T, platform Platform, occupySlot13 bool) string {
	t.Helper()

	var data []byte
	var userData10Base, slotBase int64
	switch platform {
	case PlatformPC:
		data = make([]byte, pcFixtureSize)
		copy(data, pcHeader())
		userData10Base = applyTplTestPCUserData10Base
		slotBase = applyTplTestPCSlotBase + int64(applyTplTestSlot)*applyTplTestPCSlotStride
	case PlatformPS4:
		data = make([]byte, ps4FixtureSize)
		copy(data, ps4Header())
		userData10Base = applyTplTestPS4UserData10Base
		slotBase = applyTplTestPS4SlotBase + int64(applyTplTestSlot)*applyTplTestPS4SlotStride
	default:
		t.Fatalf("unknown platform %q", platform)
	}

	// 1. Mark character active.
	data[userData10Base+applyTplTestActiveFlagsOffset+int64(applyTplTestSlot)] = applyTplTestActiveFlagValue

	// 2. Set profile summary name and level.
	summaryBase := userData10Base + applyTplTestSummaryOffset + int64(applyTplTestSlot)*applyTplTestSummaryStride
	initialName := utf16.Encode([]rune("Tarnished"))
	for i, u := range initialName {
		binary.LittleEndian.PutUint16(data[summaryBase+applyTplTestSummaryNameOffset+int64(i*2):], u)
	}
	binary.LittleEndian.PutUint32(data[summaryBase+applyTplTestSummaryLevelOffset:], 10)
	data[summaryBase+0x243] = 0 // Vagabond

	// 3. Slot version and anchor. The anchor sits behind the complete zero-filled
	// GaItem table so the fixture survives WriteSave reload validation.
	binary.LittleEndian.PutUint32(data[slotBase:], 0x6E)
	anchorAt := applyTplTestOrdinaryAnchorAt
	copy(data[slotBase+anchorAt:], applyTplTestAnchor)

	// 4. PlayerGameData name.
	playerAt := anchorAt + applyTplTestPlayerNameOffset
	for i, u := range initialName {
		binary.LittleEndian.PutUint16(data[slotBase+playerAt+int64(i*2):], u)
	}

	// 5. PlayerGameData starting class (Vagabond = 0).
	data[slotBase+anchorAt+applyTplTestStatsClassOffset] = 0

	// 6. PlayerGameData attributes & stats (Vagabond base).
	for i, v := range applyTplTestVagabondAttrs {
		binary.LittleEndian.PutUint32(data[slotBase+anchorAt+applyTplTestStatsVigorOffset+int64(i*4):], v)
	}
	binary.LittleEndian.PutUint32(data[slotBase+anchorAt+applyTplTestStatsLevelOffset:], 10)
	binary.LittleEndian.PutUint32(data[slotBase+anchorAt+applyTplTestStatsTotalGetSoulOffset:], 1000)

	// 7. EquippedSpells (all 14 empty by default).
	sectionAt := anchorAt + applyTplTestSpellsSectionOffset
	for i := 0; i < 14; i++ {
		binary.LittleEndian.PutUint32(data[slotBase+sectionAt+int64(i*8):], applyTplTestSpellEmptyID)
		binary.LittleEndian.PutUint32(data[slotBase+sectionAt+int64(i*8+4):], applyTplTestSpellEmptyFollower)
	}
	if occupySlot13 {
		// Physical spell fields store raw MagicParam IDs (0x00000FA0, not game ID 0x40000FA0)
		binary.LittleEndian.PutUint32(data[slotBase+sectionAt+int64(12*8):], 0x00000FA0)
		binary.LittleEndian.PutUint32(data[slotBase+sectionAt+int64(12*8+4):], applyTplTestSpellOccFollower)
	}
	binary.LittleEndian.PutUint32(data[slotBase+sectionAt+112:], applyTplTestSpellEmptyID) // Active index

	// 8. Memory stone count = 0, talisman slot count = 0
	data[slotBase+anchorAt+applyTplTestTalismanSlotsOffset] = 0

	full := filepath.Join(t.TempDir(), "save.sl2")
	if err := os.WriteFile(full, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return full
}

func TestApplyCharacterTemplate_CombinedSuccess(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			fixturePath := writeApplyTemplateFixture(t, platform, false)
			engine := New()
			session, err := engine.LoadSave(fixturePath, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			snapshotBefore := bytes.Clone(engine.sessions[session.SaveSessionID].snapshot.data)

			targetName := "EldenLord"
			targetAttrs := CharacterAttributes{
				Vigor:        40,
				Mind:         20,
				Endurance:    25,
				Strength:     30,
				Dexterity:    20,
				Intelligence: 15,
				Faith:        15,
				Arcane:       10,
			}
			// sum = 40+20+25+30+20+15+15+10 = 175. Level = 175 - 79 = 96.
			expectedLevel := uint32(96)
			targetSpells := CharacterSpellsPlan{
				RawMagicParamIDs: []uint32{0x00000FA0, 0x00000FA1}, // 2 spells
				UsedMemorySlots:  2,
			}

			plan := ApplyCharacterTemplatePlan{
				Name:       &targetName,
				Attributes: &targetAttrs,
				Spells:     &targetSpells,
			}

			initialRev := "0"
			res, err := engine.ApplyCharacterTemplate(
				session.SaveSessionID,
				applyTplTestSlot,
				plan,
				initialRev,
			)
			if err != nil {
				t.Fatalf("ApplyCharacterTemplate: %v", err)
			}

			if res.SaveRevision == initialRev {
				t.Errorf("expected revision to advance, got %q", res.SaveRevision)
			}
			if res.CharacterID != applyTplTestSlot {
				t.Errorf("CharacterID = %d, want %d", res.CharacterID, applyTplTestSlot)
			}

			// Verify character profile name and level.
			prof, err := engine.GetCharacterProfile(session.SaveSessionID, applyTplTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile: %v", err)
			}
			if prof.Name != targetName {
				t.Errorf("profile.Name = %q, want %q", prof.Name, targetName)
			}
			if prof.Level != expectedLevel {
				t.Errorf("profile.Level = %d, want %d", prof.Level, expectedLevel)
			}

			// Verify character stats.
			stats, err := engine.GetCharacterStats(session.SaveSessionID, applyTplTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterStats: %v", err)
			}
			if stats.Vigor != 40 || stats.Mind != 20 || stats.Level != expectedLevel {
				t.Errorf("stats mismatch: vigor=%d level=%d", stats.Vigor, stats.Level)
			}

			// Verify equipped spells.
			spells, err := engine.GetEquippedSpells(session.SaveSessionID, applyTplTestSlot)
			if err != nil {
				t.Fatalf("GetEquippedSpells: %v", err)
			}
			if spells.Spells[0] != 0x00000FA0 || spells.Spells[1] != 0x00000FA1 {
				t.Errorf("spells[0..1] mismatch: %v", spells.Spells[:2])
			}
			// Verify physical slots 13 and 14 are untouched sentinel.
			if spells.Spells[12] != applyTplTestSpellEmptyID || spells.Spells[13] != applyTplTestSpellEmptyID {
				t.Errorf("slots 13-14 modified: 12=0x%08X 13=0x%08X", spells.Spells[12], spells.Spells[13])
			}

			// Verify that only allowed ranges changed in snapshot.
			var slotBase, userDataBase int64
			if platform == PlatformPS4 {
				slotBase = applyTplTestPS4SlotBase + int64(applyTplTestSlot)*applyTplTestPS4SlotStride
				userDataBase = applyTplTestPS4UserData10Base
			} else {
				slotBase = applyTplTestPCSlotBase + int64(applyTplTestSlot)*applyTplTestPCSlotStride
				userDataBase = applyTplTestPCUserData10Base
			}
			playerAt := slotBase + applyTplTestOrdinaryAnchorAt + applyTplTestPlayerNameOffset
			summaryBase := userDataBase + applyTplTestSummaryOffset + int64(applyTplTestSlot)*applyTplTestSummaryStride
			summaryNameAt := summaryBase + applyTplTestSummaryNameOffset
			statsBlockAt := slotBase + applyTplTestOrdinaryAnchorAt + applyTplTestStatsVigorOffset
			summaryLevelAt := summaryBase + applyTplTestSummaryLevelOffset
			spellsSectionAt := slotBase + applyTplTestOrdinaryAnchorAt + applyTplTestSpellsSectionOffset

			allowedRanges := []applyTplByteRange{
				{start: playerAt, end: playerAt + 32},
				{start: summaryNameAt, end: summaryNameAt + 32},
				{start: statsBlockAt, end: statsBlockAt + 56},
				{start: summaryLevelAt, end: summaryLevelAt + 4},
				{start: spellsSectionAt, end: spellsSectionAt + 96},
				{start: spellsSectionAt + 112, end: spellsSectionAt + 116},
			}

			snapshotAfter := engine.sessions[session.SaveSessionID].snapshot.data
			for i := int64(0); i < int64(len(snapshotBefore)); i++ {
				if snapshotBefore[i] != snapshotAfter[i] {
					if !applyTplInAllowedRanges(i, allowedRanges) {
						t.Fatalf("unexpected byte change at 0x%X: before=0x%02X, after=0x%02X", i, snapshotBefore[i], snapshotAfter[i])
					}
				}
			}

			// Verify undo state.
			undoState, err := engine.GetUndoState(session.SaveSessionID, applyTplTestSlot)
			if err != nil {
				t.Fatalf("GetUndoState: %v", err)
			}
			if !undoState.Available {
				t.Fatalf("expected undo point to be available")
			}
			if undoState.OperationKind != kindApplyBuildTemplate {
				t.Errorf("undo operationKind = %q, want %q", undoState.OperationKind, kindApplyBuildTemplate)
			}

			// Undo the mutation and verify complete byte-for-byte restoration.
			undoRes, err := engine.UndoCharacterChanges(
				session.SaveSessionID,
				applyTplTestSlot,
				undoState.UndoToken,
				res.SaveRevision,
			)
			if err != nil {
				t.Fatalf("UndoCharacterChanges: %v", err)
			}
			if undoRes.UndoneOperationKind != kindApplyBuildTemplate {
				t.Errorf("undoneOperationKind = %q, want %q", undoRes.UndoneOperationKind, kindApplyBuildTemplate)
			}

			snapshotAfterUndo := engine.sessions[session.SaveSessionID].snapshot.data
			if !bytes.Equal(snapshotAfterUndo, snapshotBefore) {
				t.Error("snapshot after undo does not match pre-mutation snapshot byte-for-byte")
			}

			restoredProf, err := engine.GetCharacterProfile(session.SaveSessionID, applyTplTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile after undo: %v", err)
			}
			if restoredProf.Name != "Tarnished" {
				t.Errorf("restored name = %q, want 'Tarnished'", restoredProf.Name)
			}
			if restoredProf.Level != 10 {
				t.Errorf("restored level = %d, want 10", restoredProf.Level)
			}
		})
	}
}

func TestApplyCharacterTemplate_RejectsInvalidRequests(t *testing.T) {
	fixturePathNormal := writeApplyTemplateFixture(t, PlatformPC, false)
	fixturePathSlot13 := writeApplyTemplateFixture(t, PlatformPC, true)

	validName := "ValidName"
	validSpells := CharacterSpellsPlan{
		RawMagicParamIDs: []uint32{0x00000FA0},
		UsedMemorySlots:  1,
	}

	invalidAttrsMinima := CharacterAttributes{
		Vigor:        10, // Vagabond base Vigor is 15
		Mind:         10,
		Endurance:    11,
		Strength:     14,
		Dexterity:    13,
		Intelligence: 9,
		Faith:        9,
		Arcane:       7,
	}

	tooManySpells := make([]uint32, 13)
	for i := range tooManySpells {
		tooManySpells[i] = uint32(0x00000FA0 + i)
	}

	cases := []struct {
		name             string
		fixturePath      string
		charID           int
		plan             ApplyCharacterTemplatePlan
		expectedRevision string
	}{
		{
			name:             "stale expectedRevision",
			fixturePath:      fixturePathNormal,
			charID:           applyTplTestSlot,
			plan:             ApplyCharacterTemplatePlan{Name: &validName},
			expectedRevision: "9999",
		},
		{
			name:             "non-canonical expectedRevision 01",
			fixturePath:      fixturePathNormal,
			charID:           applyTplTestSlot,
			plan:             ApplyCharacterTemplatePlan{Name: &validName},
			expectedRevision: "01",
		},
		{
			name:             "non-canonical expectedRevision +1",
			fixturePath:      fixturePathNormal,
			charID:           applyTplTestSlot,
			plan:             ApplyCharacterTemplatePlan{Name: &validName},
			expectedRevision: "+1",
		},
		{
			name:             "attribute below starting class minima",
			fixturePath:      fixturePathNormal,
			charID:           applyTplTestSlot,
			plan:             ApplyCharacterTemplatePlan{Attributes: &invalidAttrsMinima},
			expectedRevision: "0",
		},
		{
			name:             "occupied physical slot 13",
			fixturePath:      fixturePathSlot13,
			charID:           applyTplTestSlot,
			plan:             ApplyCharacterTemplatePlan{Spells: &validSpells},
			expectedRevision: "0",
		},
		{
			name:        "negative UsedMemorySlots",
			fixturePath: fixturePathNormal,
			charID:      applyTplTestSlot,
			plan: ApplyCharacterTemplatePlan{
				Spells: &CharacterSpellsPlan{
					RawMagicParamIDs: []uint32{0x00000FA0},
					UsedMemorySlots:  -1,
				},
			},
			expectedRevision: "0",
		},
		{
			name:             "negative characterID",
			fixturePath:      fixturePathNormal,
			charID:           -1,
			plan:             ApplyCharacterTemplatePlan{Name: &validName},
			expectedRevision: "0",
		},
		{
			name:             "out of bounds characterID",
			fixturePath:      fixturePathNormal,
			charID:           10,
			plan:             ApplyCharacterTemplatePlan{Name: &validName},
			expectedRevision: "0",
		},
		{
			name:             "empty plan",
			fixturePath:      fixturePathNormal,
			charID:           applyTplTestSlot,
			plan:             ApplyCharacterTemplatePlan{},
			expectedRevision: "0",
		},
		{
			name:        "more than 12 spells",
			fixturePath: fixturePathNormal,
			charID:      applyTplTestSlot,
			plan: ApplyCharacterTemplatePlan{
				Spells: &CharacterSpellsPlan{
					RawMagicParamIDs: tooManySpells,
					UsedMemorySlots:  13,
				},
			},
			expectedRevision: "0",
		},
		{
			name:        "spell rawMagicParamID is 0",
			fixturePath: fixturePathNormal,
			charID:      applyTplTestSlot,
			plan: ApplyCharacterTemplatePlan{
				Spells: &CharacterSpellsPlan{
					RawMagicParamIDs: []uint32{0},
					UsedMemorySlots:  1,
				},
			},
			expectedRevision: "0",
		},
		{
			name:        "spell rawMagicParamID exceeds limit",
			fixturePath: fixturePathNormal,
			charID:      applyTplTestSlot,
			plan: ApplyCharacterTemplatePlan{
				Spells: &CharacterSpellsPlan{
					RawMagicParamIDs: []uint32{equippedSpellRawIDLimit},
					UsedMemorySlots:  1,
				},
			},
			expectedRevision: "0",
		},
		{
			name:        "duplicate rawMagicParamID",
			fixturePath: fixturePathNormal,
			charID:      applyTplTestSlot,
			plan: ApplyCharacterTemplatePlan{
				Spells: &CharacterSpellsPlan{
					RawMagicParamIDs: []uint32{0x00000FA0, 0x00000FA0},
					UsedMemorySlots:  2,
				},
			},
			expectedRevision: "0",
		},
		{
			name:        "used memory slots exceeds character capacity",
			fixturePath: fixturePathNormal,
			charID:      applyTplTestSlot,
			plan: ApplyCharacterTemplatePlan{
				Spells: &CharacterSpellsPlan{
					RawMagicParamIDs: []uint32{0x00000FA0},
					UsedMemorySlots:  5, // Vagabond base capacity is 2
				},
			},
			expectedRevision: "0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := New()
			session, err := engine.LoadSave(tc.fixturePath, "pc", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			loaded := engine.sessions[session.SaveSessionID]
			snapshotBefore := bytes.Clone(loaded.snapshot.data)
			revBefore := loaded.session.revision
			dirtyBefore := loaded.session.dirty
			undoBefore := loaded.session.undo
			ownedSeqBefore := loaded.session.ownedSeq
			ownedByLocatorBefore := maps.Clone(loaded.session.ownedByLocator)
			ownedByIDBefore := maps.Clone(loaded.session.ownedByID)

			res, err := engine.ApplyCharacterTemplate(
				session.SaveSessionID,
				tc.charID,
				tc.plan,
				tc.expectedRevision,
			)
			if err == nil {
				t.Fatalf("ApplyCharacterTemplate(%s) accepted invalid request", tc.name)
			}
			if !reflect.DeepEqual(res, ApplyCharacterTemplateResult{}) {
				t.Errorf("ApplyCharacterTemplate(%s) returned non-zero result on error: %+v", tc.name, res)
			}
			if !bytes.Equal(loaded.snapshot.data, snapshotBefore) {
				t.Errorf("ApplyCharacterTemplate(%s) mutated snapshot bytes on error", tc.name)
			}
			if loaded.session.revision != revBefore {
				t.Errorf("ApplyCharacterTemplate(%s) advanced revision on error: got %d, want %d", tc.name, loaded.session.revision, revBefore)
			}
			if loaded.session.dirty != dirtyBefore {
				t.Errorf("ApplyCharacterTemplate(%s) changed dirty flag on error: got %v, want %v", tc.name, loaded.session.dirty, dirtyBefore)
			}
			if loaded.session.undo != undoBefore {
				t.Errorf("ApplyCharacterTemplate(%s) mutated undo state on error", tc.name)
			}
			if loaded.session.ownedSeq != ownedSeqBefore {
				t.Errorf("ApplyCharacterTemplate(%s) changed ownedSeq on error: got %d, want %d", tc.name, loaded.session.ownedSeq, ownedSeqBefore)
			}
			if !reflect.DeepEqual(loaded.session.ownedByLocator, ownedByLocatorBefore) {
				t.Errorf("ApplyCharacterTemplate(%s) changed ownedByLocator on error", tc.name)
			}
			if !reflect.DeepEqual(loaded.session.ownedByID, ownedByIDBefore) {
				t.Errorf("ApplyCharacterTemplate(%s) changed ownedByID on error", tc.name)
			}
		})
	}
}

// TestApplyCharacterTemplate_PersistsThroughWriteSaveAndReload proves that a combined
// name/statistics/spells mutation survives serialization: the plan is applied, written to an
// explicit new target, and read back semantically through a fresh Engine and the public getters.
func TestApplyCharacterTemplate_PersistsThroughWriteSaveAndReload(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			sourcePath := writeApplyTemplateFixture(t, platform, false)
			engine := New()
			session, err := engine.LoadSave(sourcePath, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			targetName := "EldenLord"
			targetAttrs := CharacterAttributes{
				Vigor:        40,
				Mind:         20,
				Endurance:    25,
				Strength:     30,
				Dexterity:    20,
				Intelligence: 15,
				Faith:        15,
				Arcane:       10,
			}
			// sum = 175. Level = 175 - 79 = 96.
			expectedLevel := uint32(96)
			targetSpells := []uint32{0x00000FA0, 0x00000FA1}

			applied, err := engine.ApplyCharacterTemplate(
				session.SaveSessionID,
				applyTplTestSlot,
				ApplyCharacterTemplatePlan{
					Name:       &targetName,
					Attributes: &targetAttrs,
					Spells:     &CharacterSpellsPlan{RawMagicParamIDs: targetSpells, UsedMemorySlots: 2},
				},
				"0",
			)
			if err != nil {
				t.Fatalf("ApplyCharacterTemplate: %v", err)
			}
			if applied.SaveRevision != "1" {
				t.Fatalf("SaveRevision = %q, want %q", applied.SaveRevision, "1")
			}

			targetPath := filepath.Join(t.TempDir(), "written.sl2")
			if _, err := engine.WriteSave(session.SaveSessionID, applied.SaveRevision, targetPath); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}

			reloaded := New()
			reloadedSession, err := reloaded.LoadSave(targetPath, string(platform), "local")
			if err != nil {
				t.Fatalf("LoadSave persisted target: %v", err)
			}

			prof, err := reloaded.GetCharacterProfile(reloadedSession.SaveSessionID, applyTplTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterProfile: %v", err)
			}
			if prof.Name != targetName || prof.Level != expectedLevel {
				t.Errorf("persisted profile = %q level %d, want %q level %d",
					prof.Name, prof.Level, targetName, expectedLevel)
			}

			stats, err := reloaded.GetCharacterStats(reloadedSession.SaveSessionID, applyTplTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterStats: %v", err)
			}
			persistedAttrs := CharacterAttributes{
				Vigor:        stats.Vigor,
				Mind:         stats.Mind,
				Endurance:    stats.Endurance,
				Strength:     stats.Strength,
				Dexterity:    stats.Dexterity,
				Intelligence: stats.Intelligence,
				Faith:        stats.Faith,
				Arcane:       stats.Arcane,
			}
			if persistedAttrs != targetAttrs || stats.Level != expectedLevel {
				t.Errorf("persisted stats = %+v level %d, want %+v level %d",
					persistedAttrs, stats.Level, targetAttrs, expectedLevel)
			}

			spells, err := reloaded.GetEquippedSpells(reloadedSession.SaveSessionID, applyTplTestSlot)
			if err != nil {
				t.Fatalf("GetEquippedSpells: %v", err)
			}
			// The compact target list, the empty tail and the untouched physical
			// positions 13-14 are one comparable array.
			var expectedSpells [equippedSpellSlotCount]uint32
			for index := range expectedSpells {
				expectedSpells[index] = applyTplTestSpellEmptyID
			}
			copy(expectedSpells[:], targetSpells)
			if spells.Spells != expectedSpells {
				t.Errorf("persisted spells = %v, want %v", spells.Spells, expectedSpells)
			}
		})
	}
}
