package saveengine

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	setAppearanceTestSlot            = 3
	setAppearanceTestAnchorAt        = 0xB000
	setAppearanceTestProjectileCount = 0x931D
	setAppearanceTestArmamentsSize   = 0x9C
	setAppearanceTestPhysicsSize     = 0x0C
	setAppearanceTestPCSlotBase      = 0x310
	setAppearanceTestPCSlotStride    = 0x280010
	setAppearanceTestPS4SlotBase     = 0x70
	setAppearanceTestPS4SlotStride   = 0x280000
	setAppearanceTestLaterFaceAt     = 0x200000
	setAppearanceTestFaceSize        = 0x12F
	setAppearanceTestSexFlagsOffset  = 0x125
	setAppearanceTestUnknownStart    = 0x70
	setAppearanceTestUnknownEnd      = 0xB0
	setAppearanceTestGenderOffset    = -249
	setAppearanceTestVoiceTypeOffset = -245
	setAppearanceTestGaItemTableSize = 0x6E
)

func setAppearanceTestValues(seed uint8) CharacterAppearanceValues {
	raw := appearanceValues(seed)
	return CharacterAppearanceValues{
		Gender:    raw.Gender,
		VoiceType: raw.VoiceType,
		ModelIDs:  raw.ModelIDs,
		FaceShape: raw.FaceShape,
		Body:      raw.Body,
		Skin:      raw.Skin,
	}
}

func setAppearanceTestSlotBase(platform Platform) int64 {
	if platform == PlatformPS4 {
		return setAppearanceTestPS4SlotBase +
			setAppearanceTestSlot*setAppearanceTestPS4SlotStride
	}
	return setAppearanceTestPCSlotBase +
		setAppearanceTestSlot*setAppearanceTestPCSlotStride
}

func writeSetAppearanceFace(block []byte, values CharacterAppearanceValues) {
	copy(block, []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'})
	binary.LittleEndian.PutUint32(block[0x08:], 4)
	binary.LittleEndian.PutUint32(block[0x0C:], 0x120)
	for index, modelID := range values.ModelIDs {
		binary.LittleEndian.PutUint32(block[0x10+index*4:], modelID)
	}
	copy(block[0x30:], values.FaceShape[:])
	copy(block[0xB0:], values.Body[:])
	copy(block[0xB7:], values.Skin[:])
}

func writeSetAppearanceFixture(
	t *testing.T,
	platform Platform,
	values CharacterAppearanceValues,
) (string, int64, int64, int64) {
	t.Helper()

	content := gestureTestActiveFixture(
		platform, setAppearanceTestSlot, setAppearanceTestAnchorAt, 0)
	path := writeGestureFixture(t, content)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	slotBase := setAppearanceTestSlotBase(platform)
	binary.LittleEndian.PutUint32(data[slotBase:], setAppearanceTestGaItemTableSize)
	anchor := slotBase + setAppearanceTestAnchorAt
	genderAt := anchor + setAppearanceTestGenderOffset
	voiceAt := anchor + setAppearanceTestVoiceTypeOffset
	faceAt := anchor + setAppearanceTestProjectileCount + 4 +
		setAppearanceTestArmamentsSize + setAppearanceTestPhysicsSize
	data[genderAt] = values.Gender
	data[voiceAt] = values.VoiceType
	writeSetAppearanceFace(data[faceAt:faceAt+setAppearanceTestFaceSize], values)
	for index := setAppearanceTestUnknownStart; index < setAppearanceTestUnknownEnd; index++ {
		data[faceAt+int64(index)] = byte(index ^ 0xA5)
	}
	data[faceAt+setAppearanceTestSexFlagsOffset] = 0x7A
	data[faceAt+setAppearanceTestSexFlagsOffset+1] = 0xB4

	// A second valid FACE block proves the mutation addresses only the first one.
	laterAt := slotBase + setAppearanceTestLaterFaceAt
	writeSetAppearanceFace(
		data[laterAt:laterAt+setAppearanceTestFaceSize], setAppearanceTestValues(0xC1))

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
	return path, genderAt, voiceAt, faceAt
}

func TestSetCharacterAppearanceWritesExactFieldsAndReloadsOnBothPlatforms(t *testing.T) {
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			beforeValues := setAppearanceTestValues(0x11)
			afterValues := setAppearanceTestValues(0x53)
			afterValues.Gender = 0
			afterValues.VoiceType = 5
			path, genderAt, voiceAt, faceAt := writeSetAppearanceFixture(
				t, platform, beforeValues)
			sourceBefore, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}

			engine := New()
			loaded, err := engine.LoadSave(path, string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			result, err := engine.SetCharacterAppearance(
				loaded.SaveSessionID, setAppearanceTestSlot, afterValues, "0")
			if err != nil {
				t.Fatalf("SetCharacterAppearance: %v", err)
			}
			wantResult := SetCharacterAppearanceResult{
				SaveSessionID: loaded.SaveSessionID,
				SaveRevision:  "1",
				CharacterID:   setAppearanceTestSlot,
				Appearance:    afterValues,
			}
			if !reflect.DeepEqual(result, wantResult) {
				t.Errorf("result = %+v, want %+v", result, wantResult)
			}

			expected := bytes.Clone(sourceBefore)
			expected[genderAt] = afterValues.Gender
			expected[voiceAt] = afterValues.VoiceType
			writeSetAppearanceFace(
				expected[faceAt:faceAt+setAppearanceTestFaceSize], afterValues)
			expected[faceAt+setAppearanceTestSexFlagsOffset] = 0
			expected[faceAt+setAppearanceTestSexFlagsOffset+1] = 0
			if !bytes.Equal(engine.sessions[loaded.SaveSessionID].snapshot.data, expected) {
				t.Error("mutation changed bytes outside the confirmed appearance fields")
			}
			sourceAfter, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read source after mutation: %v", err)
			}
			if !bytes.Equal(sourceBefore, sourceAfter) {
				t.Error("in-memory mutation changed the source file")
			}

			readBack, err := engine.GetCharacterAppearance(
				loaded.SaveSessionID, setAppearanceTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterAppearance: %v", err)
			}
			if readBack.Gender != afterValues.Gender ||
				readBack.VoiceType != afterValues.VoiceType ||
				readBack.ModelIDs != afterValues.ModelIDs ||
				readBack.FaceShape != afterValues.FaceShape ||
				readBack.Body != afterValues.Body || readBack.Skin != afterValues.Skin {
				t.Errorf("read-back appearance = %+v, want %+v", readBack, afterValues)
			}

			target := filepath.Join(t.TempDir(), "appearance.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloadedEngine := New()
			reloaded, err := reloadedEngine.LoadSave(target, string(platform))
			if err != nil {
				t.Fatalf("reload target: %v", err)
			}
			reloadedAppearance, err := reloadedEngine.GetCharacterAppearance(
				reloaded.SaveSessionID, setAppearanceTestSlot)
			if err != nil {
				t.Fatalf("GetCharacterAppearance after reload: %v", err)
			}
			if reloadedAppearance.ModelIDs != afterValues.ModelIDs ||
				reloadedAppearance.FaceShape != afterValues.FaceShape ||
				reloadedAppearance.Body != afterValues.Body ||
				reloadedAppearance.Skin != afterValues.Skin {
				t.Error("reloaded save lost the appearance assignment")
			}
		})
	}
}

func TestSetCharacterAppearanceRejectsInvalidInputWithoutMutation(t *testing.T) {
	values := setAppearanceTestValues(0x21)
	path, _, _, _ := writeSetAppearanceFixture(t, PlatformPC, values)
	engine := New()
	loaded, err := engine.LoadSave(path, "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := bytes.Clone(engine.sessions[loaded.SaveSessionID].snapshot.data)

	cases := map[string]struct {
		appearance       CharacterAppearanceValues
		expectedRevision string
		want             string
	}{
		"gender": {func() CharacterAppearanceValues {
			invalid := values
			invalid.Gender = 2
			return invalid
		}(), "0", "appearance.gender 2 is outside the range 0..1"},
		"voice type": {func() CharacterAppearanceValues {
			invalid := values
			invalid.VoiceType = 6
			return invalid
		}(), "0", "appearance.voiceType 6 is outside the range 0..5"},
		"revision": {values, "1",
			`expectedRevision "1" does not match the current saveRevision "0"`},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := engine.SetCharacterAppearance(
				loaded.SaveSessionID,
				setAppearanceTestSlot,
				testCase.appearance,
				testCase.expectedRevision,
			)
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("error = %v, want %q", err, testCase.want)
			}
			if !reflect.DeepEqual(result, SetCharacterAppearanceResult{}) {
				t.Errorf("result = %+v, want the zero value", result)
			}
		})
	}
	if !bytes.Equal(before, engine.sessions[loaded.SaveSessionID].snapshot.data) {
		t.Error("rejected mutation changed the private snapshot")
	}
	if engine.sessions[loaded.SaveSessionID].session.revisionString() != "0" ||
		engine.sessions[loaded.SaveSessionID].session.dirty {
		t.Error("rejected mutation changed revision or dirty state")
	}
}
