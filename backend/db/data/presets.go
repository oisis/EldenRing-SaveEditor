package data

// AppearancePreset holds a complete character appearance configuration.
// Field order in FaceShape and Skin arrays matches the FaceData blob layout
// documented in spec/09-face-data.md and verified against hex dumps.
type AppearancePreset struct {
	Name      string
	Image     string // filename in frontend/public/presets/ (e.g. "geralt.jpg")
	BodyType  uint8  // 1=TypeA(male), 0=TypeB(female)
	VoiceType uint8  // 0=Young1, 1=Young2, 2=Mature1, 3=Mature2, 4=Aged1, 5=Aged2

	// Model IDs (u8 stored as u32 LE with 3 padding bytes)
	FaceModel     uint8 // Bone Structure
	HairModel     uint8 // Hair Style
	EyeModel      uint8 // Eye model (usually 0)
	EyebrowModel  uint8 // Brow Style
	BeardModel    uint8 // Beard Style
	EyepatchModel uint8 // Eyepatch
	DecalModel    uint8 // Tattoo/Mark
	EyelashModel  uint8 // Eyelashes

	// Face shape: 64 bytes at blob offset 0x30-0x6F
	// Order: apparent_age, facial_aesthetic, form_emphasis, unk,
	//   brow_ridge_height, inner_brow_ridge, outer_brow_ridge,
	//   cheekbone_height, cheekbone_depth, cheekbone_width, cheekbone_protrusion, cheeks,
	//   chin_tip_position, chin_length, chin_protrusion, chin_depth, chin_size, chin_height, chin_width,
	//   eye_position, eye_size, eye_slant, eye_spacing,
	//   nose_size, nose_forehead_ratio, unk,
	//   face_protrusion, vertical_face_ratio, facial_feature_slant, horizontal_face_ratio, unk,
	//   forehead_depth, forehead_protrusion, unk,
	//   jaw_protrusion, jaw_width, lower_jaw, jaw_contour,
	//   lip_shape, lip_size, lip_fullness, mouth_expression, lip_protrusion, lip_thickness,
	//   mouth_protrusion, mouth_slant, occlusion, mouth_position, mouth_width, mouth_chin_distance,
	//   nose_ridge_depth, nose_ridge_length, nose_position, nose_tip_height,
	//   nostril_slant, nostril_size, nostril_width,
	//   nose_protrusion, nose_bridge_height, bridge_protrusion1, bridge_protrusion2, nose_bridge_width,
	//   nose_height, nose_slant
	FaceShape [64]uint8

	// Body proportions: 7 bytes at blob offset 0xB0-0xB6
	// Order: head, chest, abdomen, arm_r, leg_r, arm_l, leg_l
	Body [7]uint8

	// Skin & cosmetics: 91 bytes at blob offset 0xB7-0x111
	// Order: skin_r/g/b, skin_luster, pores, stubble, dark_circles, dark_circle_r/g/b,
	//   cheeks_int, cheek_r/g/b, eyeliner, eyeliner_r/g/b,
	//   eyeshadow_lower, lower_r/g/b, eyeshadow_upper, upper_r/g/b,
	//   lipstick, lipstick_r/g/b, tattoo_h/v/angle/exp, tattoo_r/g/b, tattoo_unk, tattoo_flip,
	//   body_hair, body_hair_r/g/b,
	//   right_iris_r/g/b, right_iris_size, right_clouding, right_cloud_r/g/b, right_white_r/g/b, right_eye_pos,
	//   left_iris_r/g/b, left_iris_size, left_clouding, left_cloud_r/g/b, left_white_r/g/b, left_eye_pos,
	//   hair_r/g/b, luster, root, white,
	//   beard_r/g/b, luster, root, white,
	//   brow_r/g/b, luster, root, white,
	//   lash_r/g/b, patch_r/g/b
	Skin [91]uint8
}

// Presets contains all available appearance presets.
// Generated from tmp/characters/characters.md by scripts/parse_presets.go.
// Source: https://eldensliders.com/
var Presets = GeneratedPresets

// --- Geralt of Rivia, the Witcher ---
var geralt = AppearancePreset{
	Name:      "Geralt of Rivia",
	Image:     "geralt.jpg",
	BodyType:  1, // Type A
	VoiceType: 2, // Mature Voice 1
	// Models
	FaceModel: 1, HairModel: 9, EyeModel: 0, EyebrowModel: 2,
	BeardModel: 4, EyepatchModel: 1, DecalModel: 7, EyelashModel: 3,
	// Face shape
	FaceShape: [64]uint8{
		255, 100, 0, 0, // apparent_age, facial_aesthetic, form_emphasis, unk
		158, 128, 178,   // brow_ridge_height, inner, outer
		225, 148, 128, 255, 218, // cheekbone: height, depth, width, protrusion, cheeks
		138, 0, 98, 128, 138, 138, 118, // chin: tip, length, protrusion, depth, size, height, width
		198, 170, 128, 111, // eye: position, size, slant, spacing
		88, 78, 0,         // nose_size, nose_forehead_ratio, unk
		118, 108, 225, 95, 0, // face_protrusion, vert_ratio, feature_slant, horiz_ratio, unk
		108, 128, 0,          // forehead_depth, forehead_protrusion, unk
		128, 38, 38, 148,     // jaw: protrusion, width, lower, contour
		98, 178, 128, 18, 78, 128, // lip: shape, size, fullness, expression, protrusion, thickness
		108, 128, 138, 108, 128, 108, // mouth: protrusion, slant, occlusion, position, width, chin_dist
		138, 108, 78, 105,    // nose_ridge: depth, length, position, tip_height
		83, 108, 143,         // nostril: slant, size, width
		28, 128, 128, 128, 148, // nose: protrusion, bridge_height, bridge1, bridge2, bridge_width
		128, 128,             // nose_height, nose_slant
	},
	// Body: head, chest, abdomen, arm_r, leg_r, arm_l, leg_l
	Body: [7]uint8{128, 218, 148, 218, 208, 218, 208},
	// Skin
	Skin: [91]uint8{
		180, 131, 113, 178, // skin RGB + luster
		35,                  // pores
		255,                 // stubble (beard)
		210,                 // dark_circles
		40, 30, 35,          // dark_circle RGB
		0,                   // cheeks_intensity
		0, 0, 0,             // cheek RGB
		25,                  // eyeliner
		30, 20, 20,          // eyeliner RGB
		30,                  // eyeshadow_lower
		50, 25, 0,           // eyeshadow_lower RGB
		36,                  // eyeshadow_upper
		60, 80, 90,          // eyeshadow_upper RGB
		18,                  // lipstick
		183, 133, 111,       // lipstick RGB
		88, 220, 98, 218,    // tattoo: horiz, vert, angle, expansion
		81, 27, 24,          // tattoo RGB
		128,                 // tattoo_unk
		0,                   // tattoo_flip (OFF)
		0,                   // body_hair
		209, 190, 177,       // body_hair RGB (Match Hair)
		// Right eye
		255, 155, 55, 190, 31, // iris RGB, size, clouding
		255, 155, 55,          // clouding RGB
		255, 255, 255,         // white RGB
		128,                   // eye_pos
		// Left eye
		255, 155, 55, 190, 31,
		255, 155, 55,
		255, 255, 255,
		128,
		// Hair
		209, 190, 177, 78, 225, 0, // hair RGB, luster, root, white
		// Beard (Match Hair for color)
		209, 190, 177, 78, 129, 128, // beard RGB, luster, root, white
		// Eyebrows
		29, 20, 17, 78, 255, 0, // brow RGB, luster(match), root, white(match)
		// Eyelashes
		30, 20, 5, // lash RGB
		// Eyepatch
		0, 0, 0, // patch RGB
	},
}

// --- Sekiro, the Wolf Shinobi ---
var sekiro = AppearancePreset{
	Name:      "Sekiro",
	Image:     "sekiro.jpg",
	BodyType:  1, // Type A
	VoiceType: 2, // Mature Voice 1
	FaceModel: 6, HairModel: 10, EyeModel: 0, EyebrowModel: 8,
	BeardModel: 2, EyepatchModel: 1, DecalModel: 2, EyelashModel: 3,
	FaceShape: [64]uint8{
		255, 108, 0, 0,
		178, 98, 108,
		68, 190, 128, 218, 108,
		168, 38, 68, 128, 158, 68, 191,
		0, 106, 216, 118,
		148, 206, 0,
		108, 88, 98, 115, 0,
		198, 128, 0,
		128, 148, 38, 128,
		48, 154, 128, 150, 228, 118,
		78, 168, 68, 0, 105, 118,
		108, 178, 98, 138,
		148, 100, 168,
		195, 68, 158, 85, 108,
		158, 100,
	},
	Body: [7]uint8{128, 138, 138, 138, 138, 138, 138},
	Skin: [91]uint8{
		153, 113, 82, 160, // skin RGB + luster
		255,                // pores
		255,                // stubble
		120,                // dark_circles
		128, 128, 128,      // dark_circle RGB (default)
		0,                  // cheeks
		128, 128, 128,      // cheek RGB
		10,                 // eyeliner
		128, 128, 128,      // eyeliner RGB
		120,                // eyeshadow_lower
		128, 128, 128,      // eyeshadow_lower RGB
		10,                 // eyeshadow_upper
		128, 128, 128,      // eyeshadow_upper RGB
		50,                 // lipstick
		128, 128, 128,      // lipstick RGB
		255, 210, 10, 218,  // tattoo: horiz, vert, angle, expansion
		128, 128, 128,      // tattoo RGB
		128,                // tattoo_unk
		0,                  // tattoo_flip
		0,                  // body_hair
		58, 46, 38,         // body_hair RGB
		// Right eye
		26, 15, 5, 225, 0,
		128, 128, 128,
		255, 255, 255,
		138,
		// Left eye
		26, 15, 5, 225, 0,
		128, 128, 128,
		255, 255, 255,
		138,
		// Hair
		58, 46, 38, 78, 128, 124,
		// Beard (Match Hair)
		58, 46, 38, 78, 128, 124,
		// Eyebrows (Match Hair)
		58, 46, 38, 78, 128, 124,
		// Eyelashes (Match Hair)
		58, 46, 38,
		// Eyepatch
		0, 0, 0,
	},
}

// --- Ragnar Lodbrok, a Viking Warrior ---
var ragnar = AppearancePreset{
	Name:      "Ragnar Lodbrok",
	Image:     "ragnar.jpg",
	BodyType:  1, // Type A
	VoiceType: 4, // Aged Voice 1
	FaceModel: 1, HairModel: 15, EyeModel: 0, EyebrowModel: 7,
	BeardModel: 3, EyepatchModel: 1, DecalModel: 8, EyelashModel: 3,
	FaceShape: [64]uint8{
		200, 0, 0, 0,
		155, 97, 140,
		125, 145, 173, 147, 125,
		168, 125, 128, 128, 122, 107, 107,
		183, 126, 135, 135,
		130, 130, 0,
		130, 130, 128, 128, 0,
		125, 149, 0,
		148, 143, 85, 130,
		65, 127, 75, 0, 127, 104,
		146, 0, 66, 175, 216, 153,
		120, 100, 115, 50,
		135, 155, 204,
		49, 125, 105, 145, 165,
		100, 145,
	},
	Body: [7]uint8{130, 255, 168, 255, 255, 255, 255},
	Skin: [91]uint8{
		169, 124, 110, 255, // skin RGB + luster
		255,                 // pores
		255,                 // stubble
		0,                   // dark_circles
		128, 128, 128,       // dark_circle RGB
		80,                  // cheeks
		227, 148, 148,       // cheek RGB
		0,                   // eyeliner
		128, 128, 128,       // eyeliner RGB
		0,                   // eyeshadow_lower
		128, 128, 128,       // eyeshadow_lower RGB
		0,                   // eyeshadow_upper
		128, 128, 128,       // eyeshadow_upper RGB
		50,                  // lipstick
		200, 119, 119,       // lipstick RGB
		51, 190, 92, 190,    // tattoo: horiz, vert, angle, expansion
		110, 30, 30,         // tattoo RGB
		128,                 // tattoo_unk
		0,                   // tattoo_flip (OFF)
		131,                 // body_hair
		73, 52, 31,          // body_hair RGB
		// Right eye
		145, 225, 65, 130, 0,
		128, 128, 128,
		255, 255, 255,
		150,
		// Left eye
		145, 225, 65, 130, 0,
		128, 128, 128,
		255, 255, 255,
		150,
		// Hair
		167, 98, 48, 66, 127, 117,
		// Beard (Match Hair)
		167, 98, 48, 66, 127, 117,
		// Eyebrows (Match Hair)
		167, 98, 48, 66, 127, 117,
		// Eyelashes (Match Hair)
		167, 98, 48,
		// Eyepatch
		0, 0, 0,
	},
}

// --- Trevor Belmont, Vampire Hunter ---
var trevorBelmont = AppearancePreset{
	Name:      "Trevor Belmont",
	Image:     "trevor-belmont.jpg",
	BodyType:  1, // Type A
	VoiceType: 2, // Mature Voice 1
	FaceModel: 1, HairModel: 8, EyeModel: 0, EyebrowModel: 9,
	BeardModel: 2, EyepatchModel: 1, DecalModel: 4, EyelashModel: 2,
	FaceShape: [64]uint8{
		0, 0, 0, 0,
		128, 78, 138,
		125, 128, 100, 95, 108,
		125, 157, 58, 118, 188, 108, 108,
		120, 124, 118, 131,
		138, 130, 0,
		140, 48, 145, 115, 0,
		170, 135, 0,
		148, 78, 118, 131,
		135, 132, 110, 108, 135, 134,
		126, 140, 118, 121, 125, 133,
		118, 128, 88, 125,
		153, 118, 133,
		148, 124, 138, 125, 118,
		138, 128,
	},
	Body: [7]uint8{138, 138, 108, 168, 128, 168, 128},
	Skin: [91]uint8{
		235, 165, 125, 78, // skin RGB + luster
		0,                  // pores
		255,                // stubble
		20,                 // dark_circles
		128, 128, 128,      // dark_circle RGB
		0,                  // cheeks
		128, 128, 128,      // cheek RGB
		0,                  // eyeliner
		128, 128, 128,      // eyeliner RGB
		80,                 // eyeshadow_lower
		128, 128, 128,      // eyeshadow_lower RGB
		10,                 // eyeshadow_upper
		128, 128, 128,      // eyeshadow_upper RGB
		0,                  // lipstick
		128, 128, 128,      // lipstick RGB
		60, 164, 118, 188,  // tattoo: horiz, vert, angle, expansion
		128, 128, 128,      // tattoo RGB
		128,                // tattoo_unk
		0,                  // tattoo_flip
		200,                // body_hair
		80, 60, 43,         // body_hair RGB
		// Right eye
		87, 171, 255, 200, 0,
		128, 128, 128,
		255, 255, 255,
		128,
		// Left eye
		87, 171, 255, 200, 0,
		128, 128, 128,
		255, 255, 255,
		128,
		// Hair
		80, 60, 43, 0, 125, 0,
		// Beard (Match Hair)
		80, 60, 43, 0, 125, 0,
		// Eyebrows (Match Hair)
		80, 60, 43, 0, 125, 0,
		// Eyelashes (Match Hair)
		80, 60, 43,
		// Eyepatch
		0, 0, 0,
	},
}

// --- Yennefer, Sorceress from the Witcher ---
var yennefer = AppearancePreset{
	Name:      "Yennefer",
	Image:     "yennefer.jpg",
	BodyType:  0, // Type B (female)
	VoiceType: 2, // Mature Voice 1
	FaceModel: 6, HairModel: 22, EyeModel: 0, EyebrowModel: 15,
	BeardModel: 1, EyepatchModel: 1, DecalModel: 9, EyelashModel: 4,
	FaceShape: [64]uint8{
		200, 152, 0, 0,
		150, 109, 130,
		82, 148, 110, 130, 154,
		160, 120, 124, 144, 127, 141, 103,
		131, 129, 159, 96,
		145, 65, 0,
		100, 105, 70, 183, 0,
		140, 181, 0,
		144, 42, 109, 78,
		214, 130, 130, 151, 204, 127,
		118, 160, 157, 218, 127, 137,
		90, 160, 124, 150,
		133, 110, 111,
		125, 150, 210, 164, 131,
		141, 100,
	},
	Body: [7]uint8{150, 36, 56, 110, 146, 110, 146},
	Skin: [91]uint8{
		250, 185, 180, 170, // skin RGB + luster
		255,                 // pores
		0,                   // stubble
		57,                  // dark_circles
		0, 0, 0,             // dark_circle RGB
		75,                  // cheeks
		115, 26, 43,         // cheek RGB
		57,                  // eyeliner
		0, 0, 0,             // eyeliner RGB
		205,                 // eyeshadow_lower
		0, 0, 0,             // eyeshadow_lower RGB
		150,                 // eyeshadow_upper
		100, 102, 158,       // eyeshadow_upper RGB
		110,                 // lipstick
		166, 35, 125,        // lipstick RGB
		78, 70, 128, 58,     // tattoo: horiz, vert, angle, expansion
		0, 0, 0,             // tattoo RGB
		128,                 // tattoo_unk
		0,                   // tattoo_flip (OFF)
		0,                   // body_hair
		0, 0, 0,             // body_hair RGB (Match Hair = black)
		// Right eye
		70, 52, 178, 240, 30,
		0, 0, 0,
		255, 255, 255,
		115,
		// Left eye
		70, 52, 178, 240, 30,
		0, 0, 0,
		255, 255, 255,
		115,
		// Hair
		0, 0, 0, 200, 105, 0,
		// Beard
		0, 0, 0, 200, 105, 0,
		// Eyebrows
		0, 0, 0, 0, 105, 0, // luster=0 (not match)
		// Eyelashes
		0, 0, 0,
		// Eyepatch
		0, 0, 0,
	},
}
