package gamecatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strings"
)

// AppearancePresetsPath is the catalog-relative path of the appearance preset
// file. It is the single source of truth for the backend appearance presets.
const AppearancePresetsPath = "presets/appearance.json"

// AppearanceAssetDirectory is the catalog-relative directory that holds the
// preview image of every appearance preset.
const AppearanceAssetDirectory = "assets/appearance"

// Lengths of the three appearance blobs. They are the stored contract of the
// legacy FaceData layout and are never derived from the file.
const (
	appearanceFaceShapeLength = 64
	appearanceBodyLength      = 7
	appearanceSkinLength      = 91
)

// maxAppearanceVoiceType is the highest voice identifier the legacy data
// declares (0-5: Young 1/2, Mature 1/2, Aged 1/2).
const maxAppearanceVoiceType = 5

// appearancePresetIDPattern is the stable identifier shape: lowercase
// kebab-case, so the identifier can also be the asset file name.
var appearancePresetIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// AppearancePreset is one stored appearance preset: its identity, its preview
// asset, and the complete appearance configuration the mutating endpoints need.
// The catalog owns it read-only; every accessor hands out an independent copy.
type AppearancePreset struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`

	// BodyType keeps the legacy numeric value: 1 is Type A, 0 is Type B.
	BodyType  uint8 `json:"bodyType"`
	VoiceType uint8 `json:"voiceType"`

	FaceModel     uint8 `json:"faceModel"`
	HairModel     uint8 `json:"hairModel"`
	EyeModel      uint8 `json:"eyeModel"`
	EyebrowModel  uint8 `json:"eyebrowModel"`
	BeardModel    uint8 `json:"beardModel"`
	EyepatchModel uint8 `json:"eyepatchModel"`
	DecalModel    uint8 `json:"decalModel"`
	EyelashModel  uint8 `json:"eyelashModel"`

	FaceShape [appearanceFaceShapeLength]uint8 `json:"faceShape"`
	Body      [appearanceBodyLength]uint8      `json:"body"`
	Skin      [appearanceSkinLength]uint8      `json:"skin"`

	Tags []string `json:"tags"`
}

// appearancePresetFile is the on-disk shape of one preset. The three blobs are
// slices here so their length can actually be checked; a fixed-size array would
// silently accept a short or long list.
type appearancePresetFile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`

	BodyType  int `json:"bodyType"`
	VoiceType int `json:"voiceType"`

	FaceModel     int `json:"faceModel"`
	HairModel     int `json:"hairModel"`
	EyeModel      int `json:"eyeModel"`
	EyebrowModel  int `json:"eyebrowModel"`
	BeardModel    int `json:"beardModel"`
	EyepatchModel int `json:"eyepatchModel"`
	DecalModel    int `json:"decalModel"`
	EyelashModel  int `json:"eyelashModel"`

	FaceShape []int `json:"faceShape"`
	Body      []int `json:"body"`
	Skin      []int `json:"skin"`

	Tags []string `json:"tags"`
}

// appearancePresetsFile is the on-disk shape of AppearancePresetsPath. The
// presets stay raw so each one can be checked for the fields it declares before
// it is decoded.
type appearancePresetsFile struct {
	Presets []json.RawMessage `json:"presets"`
}

// appearancePresetFields is every field a stored preset must declare. Most of
// them accept 0 or an empty list as a valid value, so a missing field cannot be
// told apart from a declared one once it has become the Go zero value; it is
// therefore rejected before decoding instead.
var appearancePresetFields = []string{
	"id",
	"name",
	"image",
	"bodyType",
	"voiceType",
	"faceModel",
	"hairModel",
	"eyeModel",
	"eyebrowModel",
	"beardModel",
	"eyepatchModel",
	"decalModel",
	"eyelashModel",
	"faceShape",
	"body",
	"skin",
	"tags",
}

// decodeAppearancePreset rejects a preset that omits a required field and then
// decodes it, refusing any field the contract does not declare.
func decodeAppearancePreset(raw json.RawMessage) (appearancePresetFile, error) {
	var declared map[string]json.RawMessage
	if err := json.Unmarshal(raw, &declared); err != nil {
		return appearancePresetFile{}, err
	}
	for _, field := range appearancePresetFields {
		if _, exists := declared[field]; !exists {
			return appearancePresetFile{}, fmt.Errorf("field %q is required", field)
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var stored appearancePresetFile
	if err := decoder.Decode(&stored); err != nil {
		return appearancePresetFile{}, err
	}
	return stored, nil
}

// LoadAppearancePresets reads and validates AppearancePresetsPath from the
// catalog data filesystem, keeping the file order. An absent, malformed, or
// inconsistent file is an error instead of a partially populated catalog, and a
// preset whose image is missing under AppearanceAssetDirectory is rejected here
// rather than at display time.
func LoadAppearancePresets(catalogFS fs.FS) ([]AppearancePreset, error) {
	content, err := fs.ReadFile(catalogFS, AppearancePresetsPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", AppearancePresetsPath, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var file appearancePresetsFile
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("decode %s: %w", AppearancePresetsPath, err)
	}
	// The file must hold exactly one document: a second value, or any trailing
	// data that is not whitespace, would silently be ignored otherwise.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode %s: multiple JSON values are not allowed", AppearancePresetsPath)
		}
		return nil, fmt.Errorf("decode %s: trailing JSON data: %w", AppearancePresetsPath, err)
	}

	if len(file.Presets) == 0 {
		return nil, fmt.Errorf("%s: at least one preset is required", AppearancePresetsPath)
	}

	presets := make([]AppearancePreset, 0, len(file.Presets))
	for index, raw := range file.Presets {
		stored, err := decodeAppearancePreset(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: preset %d: %w", AppearancePresetsPath, index, err)
		}
		preset, err := stored.toPreset()
		if err != nil {
			return nil, fmt.Errorf("%s: preset %d: %w", AppearancePresetsPath, index, err)
		}
		assetPath := AppearanceAssetDirectory + "/" + preset.Image
		if _, err := fs.Stat(catalogFS, assetPath); err != nil {
			return nil, fmt.Errorf("%s: preset %d: read asset %s: %w", AppearancePresetsPath, index, assetPath, err)
		}
		presets = append(presets, preset)
	}

	if err := validateAppearancePresets(presets); err != nil {
		return nil, fmt.Errorf("%s: %w", AppearancePresetsPath, err)
	}
	return presets, nil
}

// toPreset validates one stored preset and rewrites its variable-length blobs
// into the fixed-size arrays the catalog stores.
func (stored appearancePresetFile) toPreset() (AppearancePreset, error) {
	if stored.ID == "" {
		return AppearancePreset{}, fmt.Errorf("id must not be empty")
	}
	if !appearancePresetIDPattern.MatchString(stored.ID) {
		return AppearancePreset{}, fmt.Errorf("id %q must be lowercase kebab-case", stored.ID)
	}
	if stored.Name == "" {
		return AppearancePreset{}, fmt.Errorf("name must not be empty")
	}
	if stored.Image == "" {
		return AppearancePreset{}, fmt.Errorf("image must not be empty")
	}
	// The image is a bare file name inside AppearanceAssetDirectory, so a path
	// separator or a parent reference can never escape that directory.
	if strings.ContainsAny(stored.Image, `/\`) || stored.Image == "." || stored.Image == ".." {
		return AppearancePreset{}, fmt.Errorf("image %q must be a file name without a path", stored.Image)
	}
	if want := stored.ID + ".jpg"; stored.Image != want {
		return AppearancePreset{}, fmt.Errorf("image %q must be %q", stored.Image, want)
	}
	if stored.BodyType != 0 && stored.BodyType != 1 {
		return AppearancePreset{}, fmt.Errorf("bodyType %d must be 0 (Type B) or 1 (Type A)", stored.BodyType)
	}
	if stored.VoiceType < 0 || stored.VoiceType > maxAppearanceVoiceType {
		return AppearancePreset{}, fmt.Errorf(
			"voiceType %d must be between 0 and %d",
			stored.VoiceType,
			maxAppearanceVoiceType,
		)
	}

	preset := AppearancePreset{
		ID:       stored.ID,
		Name:     stored.Name,
		Image:    stored.Image,
		BodyType: uint8(stored.BodyType),
		//nolint:gosec // The range is checked above.
		VoiceType: uint8(stored.VoiceType),
		Tags:      make([]string, 0, len(stored.Tags)),
	}

	models := []struct {
		name   string
		value  int
		target *uint8
	}{
		{"faceModel", stored.FaceModel, &preset.FaceModel},
		{"hairModel", stored.HairModel, &preset.HairModel},
		{"eyeModel", stored.EyeModel, &preset.EyeModel},
		{"eyebrowModel", stored.EyebrowModel, &preset.EyebrowModel},
		{"beardModel", stored.BeardModel, &preset.BeardModel},
		{"eyepatchModel", stored.EyepatchModel, &preset.EyepatchModel},
		{"decalModel", stored.DecalModel, &preset.DecalModel},
		{"eyelashModel", stored.EyelashModel, &preset.EyelashModel},
	}
	for _, model := range models {
		if model.value < 0 || model.value > 255 {
			return AppearancePreset{}, fmt.Errorf("%s %d is outside the uint8 range", model.name, model.value)
		}
		*model.target = uint8(model.value)
	}

	blobs := []struct {
		name   string
		values []int
		target []uint8
	}{
		{"faceShape", stored.FaceShape, preset.FaceShape[:]},
		{"body", stored.Body, preset.Body[:]},
		{"skin", stored.Skin, preset.Skin[:]},
	}
	for _, blob := range blobs {
		if len(blob.values) != len(blob.target) {
			return AppearancePreset{}, fmt.Errorf(
				"%s has %d values, want exactly %d",
				blob.name,
				len(blob.values),
				len(blob.target),
			)
		}
		for index, value := range blob.values {
			if value < 0 || value > 255 {
				return AppearancePreset{}, fmt.Errorf("%s[%d] = %d is outside the uint8 range", blob.name, index, value)
			}
			blob.target[index] = uint8(value)
		}
	}

	seenTags := make(map[string]struct{}, len(stored.Tags))
	for index, tag := range stored.Tags {
		if tag == "" {
			return AppearancePreset{}, fmt.Errorf("tag %d must not be empty", index)
		}
		if _, duplicate := seenTags[tag]; duplicate {
			return AppearancePreset{}, fmt.Errorf("tag %d: duplicate tag %q", index, tag)
		}
		seenTags[tag] = struct{}{}
		preset.Tags = append(preset.Tags, tag)
	}
	return preset, nil
}

// validateAppearancePresets is the single place that decides whether a preset
// list may enter the catalog, so the loader and the constructor reject the same
// data. It holds the list-wide rules only; the per-preset rules belong to
// toPreset, and the asset check needs the catalog filesystem the loader owns.
func validateAppearancePresets(presets []AppearancePreset) error {
	seenIDs := make(map[string]struct{}, len(presets))
	seenNames := make(map[string]struct{}, len(presets))
	seenImages := make(map[string]struct{}, len(presets))
	for index, preset := range presets {
		if preset.ID == "" || preset.Name == "" || preset.Image == "" {
			return fmt.Errorf("preset %d: id, name and image are required", index)
		}
		if _, duplicate := seenIDs[preset.ID]; duplicate {
			return fmt.Errorf("preset %d: duplicate preset ID %q", index, preset.ID)
		}
		if _, duplicate := seenNames[preset.Name]; duplicate {
			return fmt.Errorf("preset %d: duplicate preset name %q", index, preset.Name)
		}
		if _, duplicate := seenImages[preset.Image]; duplicate {
			return fmt.Errorf("preset %d: duplicate preset image %q", index, preset.Image)
		}
		seenIDs[preset.ID] = struct{}{}
		seenNames[preset.Name] = struct{}{}
		seenImages[preset.Image] = struct{}{}
	}
	return nil
}

// AppearancePresets returns the stored presets in catalog order as an
// independent copy, so a caller can never change the catalog content. A catalog
// built without appearance presets reports that instead of returning an empty
// list.
func (catalog *Catalog) AppearancePresets() ([]AppearancePreset, error) {
	if len(catalog.appearancePresets) == 0 {
		return nil, fmt.Errorf("appearance presets are not loaded")
	}
	return cloneAppearancePresets(catalog.appearancePresets), nil
}

// cloneAppearancePresets copies the only reference type a preset carries, so
// neither the stored slice nor its tags are shared with a caller.
func cloneAppearancePresets(presets []AppearancePreset) []AppearancePreset {
	cloned := make([]AppearancePreset, len(presets))
	for index, preset := range presets {
		// A non-nil empty slice keeps "no tags" distinct from a missing field in
		// the public JSON of every consumer.
		tags := make([]string, len(preset.Tags))
		copy(tags, preset.Tags)
		preset.Tags = tags
		cloned[index] = preset
	}
	return cloned
}
