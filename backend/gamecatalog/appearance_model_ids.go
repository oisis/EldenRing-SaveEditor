package gamecatalog

import "fmt"

// Type A hair styles are not stored in UI order. These PartsIds were verified
// from controlled save analysis; every other Type A model uses the confirmed
// one-based UI index to zero-based PartsId conversion.
var typeAHairPartsIDs = map[uint8]uint8{
	1: 0, 2: 113, 3: 112, 4: 1, 5: 3, 6: 100, 7: 5, 8: 10, 9: 101,
	10: 9, 11: 8, 12: 6, 13: 7, 14: 115, 15: 114, 16: 2, 17: 4,
	18: 102, 19: 103, 20: 104, 21: 105, 22: 106, 23: 107, 24: 109,
	25: 108, 26: 111, 27: 110, 28: 117, 29: 119, 30: 118, 31: 116,
	32: 121, 33: 125, 34: 122, 35: 120, 36: 123, 37: 124,
}

// Face / Bone Structure is the one appearance model both body types share:
// Type A and Type B select the same six bone structures and store them at the
// same raw PartsIds, in steps of ten. This map is the single source of truth
// for index 0 of both branches; a UI value outside 1-6 is rejected and never
// guessed or approximated.
var faceBonePartsIDs = map[uint8]uint8{1: 0, 2: 10, 3: 20, 4: 30, 5: 40, 6: 50}

// Type B PartsIds use independent, non-sequential namespaces. The finite maps
// contain only values verified from controlled saves and used by the current
// preset set. An absent value is unsupported, never guessed. Index 0 is unused:
// Face / Bone Structure resolves through faceBonePartsIDs for both body types.
var typeBModelPartsIDs = [8]map[uint8]uint8{
	nil,
	{1: 0, 6: 100, 22: 106, 24: 109, 31: 116, 37: 124},
	{0: 0},
	{1: 0, 3: 2, 10: 9, 15: 14, 16: 15},
	{1: 0},
	{1: 0},
	{1: 0, 9: 8, 12: 11, 18: 17, 29: 29, 33: 33},
	{1: 0, 3: 2, 4: 3},
}

var appearanceModelFieldNames = [8]string{
	"faceModel", "hairModel", "eyeModel", "eyebrowModel",
	"beardModel", "eyepatchModel", "decalModel", "eyelashModel",
}

// AppearanceModelIDs resolves one catalog preset's UI-facing model selections
// to the raw PartsIds written by SaveEngine. The returned order is face, hair,
// eyes, eyebrows, beard, eyepatch, decal, eyelashes.
func AppearanceModelIDs(preset AppearancePreset) ([8]uint32, error) {
	ui := [8]uint8{
		preset.FaceModel,
		preset.HairModel,
		preset.EyeModel,
		preset.EyebrowModel,
		preset.BeardModel,
		preset.EyepatchModel,
		preset.DecalModel,
		preset.EyelashModel,
	}
	if preset.BodyType != 0 && preset.BodyType != 1 {
		return [8]uint32{}, fmt.Errorf(
			"appearance preset %q has unsupported bodyType %d", preset.ID, preset.BodyType)
	}

	var resolved [8]uint32
	face, known := faceBonePartsIDs[preset.FaceModel]
	if !known {
		return [8]uint32{}, fmt.Errorf(
			"appearance preset %q has unsupported faceModel value %d outside the confirmed 1-6 Face / Bone Structure range",
			preset.ID, preset.FaceModel)
	}
	resolved[0] = uint32(face)

	if preset.BodyType == 1 {
		for index, value := range ui {
			if index > 0 && value > 0 {
				resolved[index] = uint32(value - 1)
			}
		}
		if preset.HairModel == 0 {
			return resolved, nil
		}
		hair, known := typeAHairPartsIDs[preset.HairModel]
		if !known {
			return [8]uint32{}, fmt.Errorf(
				"appearance preset %q has unsupported Type A hair model %d",
				preset.ID, preset.HairModel)
		}
		resolved[1] = uint32(hair)
		return resolved, nil
	}
	for index, value := range ui {
		if index == 0 {
			continue
		}
		partsID, known := typeBModelPartsIDs[index][value]
		if !known {
			return [8]uint32{}, fmt.Errorf(
				"appearance preset %q has unsupported Type B %s value %d",
				preset.ID, appearanceModelFieldNames[index], value)
		}
		resolved[index] = uint32(partsID)
	}
	return resolved, nil
}
