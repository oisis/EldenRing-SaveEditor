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
			want: [8]uint32{50, 124, 0, 14, 1, 0, 11, 3},
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

// TestAppearanceModelIDsResolvesSharedFaceBoneMapping pins the six Face / Bone
// Structure positions confirmed in SaveForge 1.6.13. Both body types resolve
// through one shared table, so neither the Type A one-based conversion nor a
// Type B specific table may reappear for this field.
func TestAppearanceModelIDsResolvesSharedFaceBoneMapping(t *testing.T) {
	for ui, want := range map[uint8]uint32{1: 0, 2: 10, 3: 20, 4: 30, 5: 40, 6: 50} {
		for _, bodyType := range []uint8{1, 0} {
			preset := AppearancePreset{
				ID: "face-bone", BodyType: bodyType, FaceModel: ui, HairModel: 1,
				EyeModel: 0, EyebrowModel: 1, BeardModel: 1,
				EyepatchModel: 1, DecalModel: 1, EyelashModel: 1,
			}
			got, err := AppearanceModelIDs(preset)
			if err != nil {
				t.Fatalf("AppearanceModelIDs(bodyType %d, faceModel %d): %v", bodyType, ui, err)
			}
			if got[0] != want {
				t.Errorf("bodyType %d faceModel %d resolved to %d, want %d",
					bodyType, ui, got[0], want)
			}
		}
	}
}

// TestAppearanceModelIDsRejectsFaceBoneOutsideConfirmedRange proves the shared
// table has no fallback for either body type.
func TestAppearanceModelIDsRejectsFaceBoneOutsideConfirmedRange(t *testing.T) {
	for _, ui := range []uint8{0, 7, 99, 255} {
		for _, bodyType := range []uint8{1, 0} {
			preset := AppearancePreset{
				ID: "face-bone", BodyType: bodyType, FaceModel: ui, HairModel: 1,
				EyeModel: 0, EyebrowModel: 1, BeardModel: 1,
				EyepatchModel: 1, DecalModel: 1, EyelashModel: 1,
			}
			got, err := AppearanceModelIDs(preset)
			if err == nil || !strings.Contains(err.Error(),
				"unsupported faceModel value") {
				t.Fatalf("bodyType %d faceModel %d error = %v, want a rejected faceModel",
					bodyType, ui, err)
			}
			if got != ([8]uint32{}) {
				t.Fatalf("AppearanceModelIDs = %#v, want zero value", got)
			}
		}
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
			preset: AppearancePreset{ID: "unknown-a", BodyType: 1, FaceModel: 1, HairModel: 38},
			want:   "unsupported Type A hair model 38",
		},
		{
			name: "Type B hair",
			preset: AppearancePreset{
				ID: "unknown-b", BodyType: 0, FaceModel: 4, HairModel: 38,
				EyebrowModel: 1, BeardModel: 1, EyepatchModel: 1,
				DecalModel: 1, EyelashModel: 1,
			},
			want: "unsupported Type B hairModel value 38",
		},
		{
			name: "Type A face below range",
			preset: AppearancePreset{
				ID: "unknown-a-face", BodyType: 1, FaceModel: 0, HairModel: 1,
			},
			want: "unsupported faceModel value 0",
		},
		{
			name: "Type B face above range",
			preset: AppearancePreset{
				ID: "unknown-b-face", BodyType: 0, FaceModel: 7, HairModel: 1,
				EyebrowModel: 1, BeardModel: 1, EyepatchModel: 1,
				DecalModel: 1, EyelashModel: 1,
			},
			want: "unsupported faceModel value 7",
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
