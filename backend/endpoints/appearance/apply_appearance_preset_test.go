package appearance

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const (
	applyAppearanceHeaderSize       = 0x300
	applyAppearanceEntryCountOffset = 0x0C
	applyAppearanceSlotBlockSize    = 0x280010
	applyAppearanceFixtureSize      = int64(applyAppearanceHeaderSize) + 10*applyAppearanceSlotBlockSize + 0x60010
	applyAppearanceUserData10Offset = int64(applyAppearanceHeaderSize) + 10*applyAppearanceSlotBlockSize + 0x10
	applyAppearanceFlagsOffset      = 0x1954
	applyAppearanceAnchorAt         = 0x0640
	applyAppearanceBlockAt          = 0x3000
)

var applyAppearanceAnchor = []byte{
	0x00,
	0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0xFF, 0xFF, 0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

func writeApplyAppearanceFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, applyAppearanceFixtureSize)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[applyAppearanceEntryCountOffset:], 12)
	data[applyAppearanceUserData10Offset+applyAppearanceFlagsOffset] = 1

	slotBase := int64(applyAppearanceHeaderSize) + 0x10
	copy(data[slotBase+applyAppearanceAnchorAt:], applyAppearanceAnchor)
	face := data[slotBase+applyAppearanceBlockAt:]
	copy(face, []byte{0xFF, 0xFF, 0xFF, 0xFF, 'F', 'A', 'C', 'E'})
	binary.LittleEndian.PutUint32(face[0x08:], 4)
	binary.LittleEndian.PutUint32(face[0x0C:], 0x120)

	path := filepath.Join(t.TempDir(), "apply-appearance-preset.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestApplyAppearancePresetCommitsStoredTypeAAndTypeBPresets(t *testing.T) {
	tests := []struct {
		presetID string
		modelIDs [8]uint32
	}{
		{
			presetID: "geralt-of-rivia-the-witcher",
			modelIDs: [8]uint32{0, 101, 0, 1, 3, 0, 6, 2},
		},
		{
			presetID: "sekiro-the-wolf-shinobi",
			modelIDs: [8]uint32{50, 9, 0, 7, 1, 0, 1, 2},
		},
		{
			presetID: "red-skull-a-mutated-humanoid",
			modelIDs: [8]uint32{10, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			presetID: "yennefer-sorceress-from-the-witcher",
			modelIDs: [8]uint32{50, 106, 0, 14, 0, 0, 8, 3},
		},
	}

	for _, test := range tests {
		t.Run(test.presetID, func(t *testing.T) {
			engine := saveengine.New()
			loaded, err := engine.LoadSave(writeApplyAppearanceFixture(t), "", "local")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			result, err := ApplyAppearancePreset(
				engine, newCatalog(t), loaded.SaveSessionID, 0, test.presetID, "0")
			if err != nil {
				t.Fatalf("ApplyAppearancePreset: %v", err)
			}
			if result.SaveSessionID != loaded.SaveSessionID ||
				result.SaveRevision != "1" || result.CharacterID != 0 ||
				result.PresetID != test.presetID || result.Appearance.ModelIDs != test.modelIDs {
				t.Fatalf("result = %+v, want committed preset %q", result, test.presetID)
			}

			stored, err := engine.GetCharacterAppearance(loaded.SaveSessionID, 0)
			if err != nil {
				t.Fatalf("GetCharacterAppearance: %v", err)
			}
			if stored.Gender != result.Appearance.Gender ||
				stored.VoiceType != result.Appearance.VoiceType ||
				stored.ModelIDs != result.Appearance.ModelIDs ||
				stored.FaceShape != result.Appearance.FaceShape ||
				stored.Body != result.Appearance.Body || stored.Skin != result.Appearance.Skin {
				t.Fatalf("stored appearance = %+v, want %+v", stored, result.Appearance)
			}
		})
	}
}

func TestApplyAppearancePresetRejectsInvalidPresetWithoutMutation(t *testing.T) {
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeApplyAppearanceFixture(t), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	for _, presetID := range []string{"", "unknown", "Geralt-Of-Rivia-The-Witcher"} {
		result, err := ApplyAppearancePreset(
			engine, newCatalog(t), loaded.SaveSessionID, 0, presetID, "0")
		if err == nil {
			t.Fatalf("ApplyAppearancePreset(%q) accepted an invalid preset", presetID)
		}
		if !reflect.DeepEqual(result, ApplyAppearancePresetResult{}) {
			t.Fatalf("result = %+v, want zero value", result)
		}
	}

	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Fatalf("session = %+v, want unchanged dirty state", info)
	}
}

func TestApplyAppearancePresetRejectsUnconfirmedCatalogMapping(t *testing.T) {
	preset := gamecatalog.AppearancePreset{
		ID: "unconfirmed", Name: "Unconfirmed", Image: "unconfirmed.jpg",
		BodyType: 0, FaceModel: 7, HairModel: 1, EyebrowModel: 1,
		BeardModel: 1, EyepatchModel: 1, DecalModel: 1, EyelashModel: 1,
		Tags: []string{},
	}
	gameCatalog, err := gamecatalog.NewWithData(gamecatalog.CatalogData{
		Manifest: testManifest(), AppearancePresets: []gamecatalog.AppearancePreset{preset},
	})
	if err != nil {
		t.Fatalf("NewWithData: %v", err)
	}
	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeApplyAppearanceFixture(t), "", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := ApplyAppearancePreset(
		engine, gameCatalog, loaded.SaveSessionID, 0, preset.ID, "0")
	if err == nil || !strings.Contains(err.Error(), "unsupported faceModel value 7") {
		t.Fatalf("error = %v, want unsupported mapping", err)
	}
	if !reflect.DeepEqual(result, ApplyAppearancePresetResult{}) {
		t.Fatalf("result = %+v, want zero value", result)
	}

	info, err := engine.GetSessionInfo(loaded.SaveSessionID)
	if err != nil {
		t.Fatalf("GetSessionInfo: %v", err)
	}
	if info.UnsavedChanges {
		t.Fatalf("session = %+v, want unchanged dirty state", info)
	}
}
