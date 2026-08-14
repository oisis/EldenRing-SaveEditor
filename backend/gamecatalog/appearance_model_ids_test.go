package gamecatalog

import (
	"reflect"
	"strings"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
)

func TestAppearanceModelIDsResolvesConfirmedMappings(t *testing.T) {
	tests := []struct {
		name   string
		preset AppearancePreset
		want   [8]uint32
	}{
		{
			name: "Type A",
			preset: AppearancePreset{
				ID: "type-a", BodyType: 1, FaceModel: 6, HairModel: 37,
				EyeModel: 1, EyebrowModel: 15, BeardModel: 2,
				EyepatchModel: 1, DecalModel: 12, EyelashModel: 4,
			},
			want: [8]uint32{5, 124, 0, 14, 1, 0, 11, 3},
		},
		{
			name: "Type B",
			preset: AppearancePreset{
				ID: "type-b", BodyType: 0, FaceModel: 5, HairModel: 24,
				EyeModel: 0, EyebrowModel: 15, BeardModel: 1,
				EyepatchModel: 1, DecalModel: 12, EyelashModel: 4,
			},
			want: [8]uint32{40, 109, 0, 14, 0, 0, 11, 3},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := AppearanceModelIDs(test.preset)
			if err != nil {
				t.Fatalf("AppearanceModelIDs: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("AppearanceModelIDs = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestAppearanceModelIDsResolvesEveryStoredPreset(t *testing.T) {
	presets, err := LoadAppearancePresets(catalogdata.Files())
	if err != nil {
		t.Fatalf("LoadAppearancePresets: %v", err)
	}
	for _, preset := range presets {
		if _, err := AppearanceModelIDs(preset); err != nil {
			t.Errorf("AppearanceModelIDs(%q): %v", preset.ID, err)
		}
	}
}

func TestAppearanceModelIDsRejectsUnconfirmedValues(t *testing.T) {
	tests := []struct {
		name   string
		preset AppearancePreset
		want   string
	}{
		{
			name:   "Type A hair",
			preset: AppearancePreset{ID: "unknown-a", BodyType: 1, HairModel: 38},
			want:   "unsupported Type A hair model 38",
		},
		{
			name: "Type B face",
			preset: AppearancePreset{
				ID: "unknown-b", BodyType: 0, FaceModel: 4, HairModel: 1,
				EyebrowModel: 1, BeardModel: 1, EyepatchModel: 1,
				DecalModel: 1, EyelashModel: 1,
			},
			want: "unsupported Type B faceModel value 4",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := AppearanceModelIDs(test.preset)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want it to contain %q", err, test.want)
			}
			if got != ([8]uint32{}) {
				t.Fatalf("AppearanceModelIDs = %#v, want zero value", got)
			}
		})
	}
}
