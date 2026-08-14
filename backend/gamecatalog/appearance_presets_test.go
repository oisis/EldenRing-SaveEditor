package gamecatalog

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
)

// storedPresetCount is the number of presets presets/appearance.json holds.
const storedPresetCount = 20

// storedPresetIDs is the contractual order of the stored presets. The values
// belong to the data file; only the identifiers and their order are asserted
// here.
var storedPresetIDs = []string{
	"geralt-of-rivia-the-witcher",
	"sekiro-the-wolf-shinobi",
	"ragnar-lodbrok-a-viking-warrior",
	"trevor-belmont-vampire-hunter-from-castlevania",
	"yennefer-sorceress-from-the-witcher",
	"obi-wan-kenobi-a-jedi-master",
	"lord-voldemort-the-dark-wizard",
	"red-skull-a-mutated-humanoid",
	"isaac-the-devil-forgemaster",
	"thornkettle-the-forest-gnome",
	"kratos-the-god-of-war",
	"queen-marika-the-god-of-elden-ring",
	"ciri-the-princess-of-cintra-from-witcher",
	"makima-the-devil-hunter-from-chainsaw-man",
	"melina-the-tarnished-finger-maiden",
	"helga-the-tarnished-barbarian",
	"witch-of-salem-the-blackflame-apostle",
	"eleonora-the-sexy-twinblade-queen",
	"casca-berserks-band-of-the-falcon-commander",
	"fire-keeper-the-dark-souls-3-npc",
}

func TestLoadAppearancePresetsReadsTheStoredPresetsInOrder(t *testing.T) {
	presets, err := LoadAppearancePresets(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadAppearancePresets: %v", err)
	}
	if len(presets) != storedPresetCount {
		t.Fatalf("len(presets) = %d, want %d", len(presets), storedPresetCount)
	}
	for index, want := range storedPresetIDs {
		preset := presets[index]
		if preset.ID != want {
			t.Fatalf("presets[%d].ID = %q, want %q", index, preset.ID, want)
		}
		if preset.Image != want+".jpg" {
			t.Fatalf("presets[%d].Image = %q, want %q", index, preset.Image, want+".jpg")
		}
		if preset.Name == "" {
			t.Fatalf("presets[%d].Name is empty", index)
		}
		if preset.BodyType > 1 {
			t.Fatalf("presets[%d].BodyType = %d, want 0 or 1", index, preset.BodyType)
		}
		if preset.VoiceType > maxAppearanceVoiceType {
			t.Fatalf("presets[%d].VoiceType = %d, want at most %d", index, preset.VoiceType, maxAppearanceVoiceType)
		}
		if preset.Tags == nil || len(preset.Tags) != 0 {
			t.Fatalf("presets[%d].Tags = %#v, want an empty, non-nil slice", index, preset.Tags)
		}
	}
}

// Every preset must point at an asset that really exists, otherwise the catalog
// would advertise an image no consumer can ever load.
func TestLoadAppearancePresetsRequiresEveryAsset(t *testing.T) {
	presets, err := LoadAppearancePresets(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadAppearancePresets: %v", err)
	}
	for _, preset := range presets {
		path := AppearanceAssetDirectory + "/" + preset.Image
		info, err := fs.Stat(catalogdata.Files(), path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", path)
		}
	}
}

// presetFixture is one valid stored preset, used as the base of the rejection
// cases below so each case differs in exactly the rule it exercises.
func presetFixture() map[string]any {
	return map[string]any{
		"id":            "test-preset",
		"name":          "Test Preset",
		"image":         "test-preset.jpg",
		"bodyType":      1,
		"voiceType":     2,
		"faceModel":     1,
		"hairModel":     2,
		"eyeModel":      0,
		"eyebrowModel":  3,
		"beardModel":    4,
		"eyepatchModel": 1,
		"decalModel":    5,
		"eyelashModel":  3,
		"faceShape":     make([]int, appearanceFaceShapeLength),
		"body":          make([]int, appearanceBodyLength),
		"skin":          make([]int, appearanceSkinLength),
		"tags":          []string{},
	}
}

// fixtureFS builds a catalog filesystem holding the given raw document and one
// asset per named image.
func fixtureFS(t *testing.T, document string, images ...string) fs.FS {
	t.Helper()

	catalogFS := fstest.MapFS{
		AppearancePresetsPath: &fstest.MapFile{Data: []byte(document)},
	}
	for _, image := range images {
		catalogFS[AppearanceAssetDirectory+"/"+image] = &fstest.MapFile{Data: []byte{0xFF, 0xD8, 0xFF}}
	}
	return catalogFS
}

// documentOf serialises the presets into the stored file shape.
func documentOf(t *testing.T, presets ...map[string]any) string {
	t.Helper()

	raw, err := json.Marshal(map[string]any{"presets": presets})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return string(raw)
}

func TestLoadAppearancePresetsRejectsInvalidData(t *testing.T) {
	duplicateID := presetFixture()
	duplicateIDSecond := presetFixture()
	duplicateIDSecond["name"] = "Other Preset"

	duplicateImageFirst := presetFixture()
	duplicateImageSecond := presetFixture()
	duplicateImageSecond["id"] = "other-preset"
	duplicateImageSecond["name"] = "Other Preset"
	// The image no longer matches its own id, which is the rule that fires first.

	invalidBodyType := presetFixture()
	invalidBodyType["bodyType"] = 2

	shortFaceShape := presetFixture()
	shortFaceShape["faceShape"] = make([]int, appearanceFaceShapeLength-1)

	longBody := presetFixture()
	longBody["body"] = make([]int, appearanceBodyLength+1)

	shortSkin := presetFixture()
	shortSkin["skin"] = make([]int, appearanceSkinLength-1)

	// A missing field must not silently become the Go zero value, which would be
	// a valid value for both of these.
	missingBodyType := presetFixture()
	delete(missingBodyType, "bodyType")

	missingTags := presetFixture()
	delete(missingTags, "tags")

	valid := presetFixture()

	cases := []struct {
		name     string
		document string
		images   []string
		want     string
	}{
		{
			name:     "malformed JSON",
			document: `{"presets": [`,
			want:     "decode",
		},
		{
			name:     "trailing JSON",
			document: documentOf(t, valid) + `{}`,
			images:   []string{"test-preset.jpg"},
			want:     "multiple JSON values are not allowed",
		},
		{
			name:     "duplicate id",
			document: documentOf(t, duplicateID, duplicateIDSecond),
			images:   []string{"test-preset.jpg"},
			want:     "duplicate preset ID",
		},
		{
			name:     "duplicate image",
			document: documentOf(t, duplicateImageFirst, duplicateImageSecond),
			images:   []string{"test-preset.jpg"},
			want:     `image "test-preset.jpg" must be "other-preset.jpg"`,
		},
		{
			name:     "invalid bodyType",
			document: documentOf(t, invalidBodyType),
			images:   []string{"test-preset.jpg"},
			want:     "bodyType 2 must be 0 (Type B) or 1 (Type A)",
		},
		{
			name:     "short faceShape",
			document: documentOf(t, shortFaceShape),
			images:   []string{"test-preset.jpg"},
			want:     "faceShape has 63 values, want exactly 64",
		},
		{
			name:     "long body",
			document: documentOf(t, longBody),
			images:   []string{"test-preset.jpg"},
			want:     "body has 8 values, want exactly 7",
		},
		{
			name:     "short skin",
			document: documentOf(t, shortSkin),
			images:   []string{"test-preset.jpg"},
			want:     "skin has 90 values, want exactly 91",
		},
		{
			name:     "missing bodyType",
			document: documentOf(t, missingBodyType),
			images:   []string{"test-preset.jpg"},
			want:     `preset 0: field "bodyType" is required`,
		},
		{
			name:     "missing tags",
			document: documentOf(t, missingTags),
			images:   []string{"test-preset.jpg"},
			want:     `preset 0: field "tags" is required`,
		},
		{
			name:     "missing asset",
			document: documentOf(t, valid),
			want:     "read asset assets/appearance/test-preset.jpg",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			presets, err := LoadAppearancePresets(fixtureFS(t, testCase.document, testCase.images...))
			if err == nil {
				t.Fatalf("LoadAppearancePresets = %#v, want a rejection", presets)
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("error = %q, want it to contain %q", err.Error(), testCase.want)
			}
			if presets != nil {
				t.Fatalf("presets = %#v, want nil on a rejection", presets)
			}
		})
	}
}

// The catalog data is read-only: mutating a returned preset must not change the
// stored one, and a later call must see the original values.
func TestCatalogAppearancePresetsAreIndependentPerCall(t *testing.T) {
	presets, err := LoadAppearancePresets(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadAppearancePresets: %v", err)
	}
	catalog := &Catalog{appearancePresets: cloneAppearancePresets(presets)}

	first, err := catalog.AppearancePresets()
	if err != nil {
		t.Fatalf("AppearancePresets: %v", err)
	}
	want := first[0]
	want.Tags = append([]string(nil), first[0].Tags...)

	first[0].ID = "mutated"
	first[0].Name = "mutated"
	first[0].FaceShape[0] = first[0].FaceShape[0] + 1
	first[0].Tags = append(first[0].Tags, "injected")

	second, err := catalog.AppearancePresets()
	if err != nil {
		t.Fatalf("AppearancePresets: %v", err)
	}
	if !reflect.DeepEqual(second[0].ID, want.ID) || !reflect.DeepEqual(second[0].FaceShape, want.FaceShape) {
		t.Fatalf("presets[0] = %#v, want the stored values %#v", second[0], want)
	}
	if len(second[0].Tags) != len(want.Tags) {
		t.Fatalf("presets[0].Tags = %#v, want %#v", second[0].Tags, want.Tags)
	}
	// The caller's own slice must not be reachable through the catalog either.
	presets[0].ID = "mutated source"
	third, err := catalog.AppearancePresets()
	if err != nil {
		t.Fatalf("AppearancePresets: %v", err)
	}
	if third[0].ID != want.ID {
		t.Fatalf("presets[0].ID = %q, want %q", third[0].ID, want.ID)
	}
}

func TestCatalogAppearancePresetLookupIsExactAndIndependent(t *testing.T) {
	presets, err := LoadAppearancePresets(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadAppearancePresets: %v", err)
	}
	catalog := &Catalog{appearancePresets: cloneAppearancePresets(presets)}

	got, err := catalog.AppearancePreset(defaultTypeAAppearancePresetID)
	if err != nil {
		t.Fatalf("AppearancePreset: %v", err)
	}
	if got.ID != defaultTypeAAppearancePresetID || got.BodyType != 1 {
		t.Fatalf("preset = %+v, want the default Type A preset", got)
	}
	got.Tags = append(got.Tags, "mutated")
	again, err := catalog.AppearancePreset(defaultTypeAAppearancePresetID)
	if err != nil {
		t.Fatalf("AppearancePreset again: %v", err)
	}
	if len(again.Tags) != 0 {
		t.Fatalf("stored tags = %#v, want an independent empty slice", again.Tags)
	}

	for _, presetID := range []string{"", "Geralt-Of-Rivia-The-Witcher", "unknown"} {
		if _, err := catalog.AppearancePreset(presetID); err == nil {
			t.Fatalf("AppearancePreset(%q) accepted an invalid ID", presetID)
		}
	}
}

func TestCatalogDefaultAppearancePresetMapsBothBodyTypes(t *testing.T) {
	presets, err := LoadAppearancePresets(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadAppearancePresets: %v", err)
	}
	catalog := &Catalog{appearancePresets: cloneAppearancePresets(presets)}

	for bodyType, wantID := range map[uint8]string{
		0: defaultTypeBAppearancePresetID,
		1: defaultTypeAAppearancePresetID,
	} {
		got, err := catalog.DefaultAppearancePreset(bodyType)
		if err != nil {
			t.Fatalf("DefaultAppearancePreset(%d): %v", bodyType, err)
		}
		if got.ID != wantID || got.BodyType != bodyType {
			t.Fatalf("DefaultAppearancePreset(%d) = %+v, want %q", bodyType, got, wantID)
		}
	}
	if _, err := catalog.DefaultAppearancePreset(2); err == nil ||
		!strings.Contains(err.Error(), "gender 2 is outside the range 0..1") {
		t.Fatalf("DefaultAppearancePreset(2) error = %v", err)
	}

	for index := range presets {
		if presets[index].ID == defaultTypeBAppearancePresetID {
			presets[index].BodyType = 1
			break
		}
	}
	inconsistent := &Catalog{appearancePresets: cloneAppearancePresets(presets)}
	if _, err := inconsistent.DefaultAppearancePreset(0); err == nil ||
		!strings.Contains(err.Error(), "has bodyType 1, want 0") {
		t.Fatalf("inconsistent default error = %v", err)
	}
}
