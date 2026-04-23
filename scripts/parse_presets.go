// parse_presets reads tmp/characters/characters.md and generates backend/db/data/presets_generated.go
// Usage: go run scripts/parse_presets.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type Preset struct {
	Name      string
	Image     string
	BodyType  uint8  // 1=male, 0=female
	VoiceType uint8  // 0=Young1, 1=Young2, 2=Mature1, 3=Mature2, 4=Aged1, 5=Aged2
	Models    [8]uint8 // face, hair, eye, brow, beard, patch, decal, lash
	FaceShape [64]uint8
	Body      [7]uint8
	Skin      [91]uint8
}

var voiceMap = map[string]uint8{
	"young voice 1":  0,
	"young voice 2":  1,
	"mature voice 1": 2,
	"mature voice 2": 3,
	"aged voice 1":   4,
	"aged voice 2":   5,
}

// imageFiles maps character name substrings to actual image filenames.
var imageFiles = map[string]string{
	"geralt":      "geralt-of-rivia-the-witcher.jpg",
	"sekiro":      "sekiro-the-wolf-shinobi.jpg",
	"ragnar":      "ragnar-lodbrok-a-viking-warrior.jpg",
	"trevor":      "trevor-belmont-vampire-hunter-from-castlevania.jpg",
	"yennefer":    "yennefer-sorceress-from-the-witcher.jpg",
	"obi-wan":     "obi-wan-kenobi-a-jedi-master.jpg",
	"voldemort":   "lord-voldemort-the-dark-wizard.jpg",
	"red skull":   "red-skull-a-mutated-humanoid.jpg",
	"isaac":       "isaac-the-devil-forgemaster.jpg",
	"thornkettle": "thornkettle-the-forest-gnome.jpg",
	"kratos":      "kratos-the-god-of-war.jpg",
	"queen marika":"queen-marika-the-god-of-elden-ring.jpg",
	"ciri":        "ciri-the-princess-of-cintra-from-witcher.jpg",
	"makima":      "makima-the-devil-hunter-from-chainsaw-man.jpg",
	"melina":      "melina-the-tarnished-finger-maiden.jpg",
	"helga":       "helga-the-tarnished-barbarian.jpg",
	"witch of salem": "witch-of-salem-the-blackflame-apostle.jpg",
	"eleonora":    "eleonora-the-sexy-twinblade-queen.jpg",
	"casca":       "casca-berserks-band-of-the-falcon-commander.jpg",
	"fire keeper": "fire-keeper-the-dark-souls-3-npc.jpg",
}

func findImage(name string) string {
	lower := strings.ToLower(name)
	for key, file := range imageFiles {
		if strings.Contains(lower, key) {
			return file
		}
	}
	return toSlug(name) + ".jpg"
}

func main() {
	f, err := os.Open("tmp/characters/characters.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	var presets []Preset
	scanner := bufio.NewScanner(f)

	headerRe := regexp.MustCompile(`^## \d+\.\s+(.+)$`)
	tableRe := regexp.MustCompile(`^\|\s*(.+?)\s*\|\s*(.+?)\s*\|$`)
	imageRe := regexp.MustCompile(`-\s*\*\*Image\*\*:\s*(.+)$`)

	var cur *Preset
	var section string

	// Temp storage for "Match Hair" resolution
	var hairR, hairG, hairB, hairLuster, hairRoot, hairWhite uint8

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// New character
		if m := headerRe.FindStringSubmatch(line); m != nil {
			if cur != nil {
				presets = append(presets, *cur)
			}
			cur = &Preset{Name: m[1]}
			section = ""
			hairR, hairG, hairB, hairLuster, hairRoot, hairWhite = 0, 0, 0, 0, 0, 0
			continue
		}

		if cur == nil {
			continue
		}

		// Image URL → derive filename from mapping
		if m := imageRe.FindStringSubmatch(line); m != nil {
			cur.Image = findImage(cur.Name)
			continue
		}

		// Section header
		if strings.HasPrefix(line, "### ") {
			section = strings.TrimPrefix(line, "### ")
			continue
		}

		// Table row
		m := tableRe.FindStringSubmatch(line)
		if m == nil || m[1] == "Parameter" || strings.HasPrefix(m[1], "---") {
			continue
		}
		param, value := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])

		switch section {
		case "Base":
			switch param {
			case "Body Type":
				if value == "Type A" {
					cur.BodyType = 1
				}
			case "Voice":
				cur.VoiceType = voiceMap[strings.ToLower(value)]
			case "Skin Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[0], cur.Skin[1], cur.Skin[2] = rgb[0], rgb[1], rgb[2]
			}

		case "Face Template":
			switch param {
			case "Bone Structure":
				cur.Models[0] = uint8(parseInt(value))
			case "Form Emphasis":
				cur.FaceShape[2] = uint8(parseInt(value))
			case "Apparent Age":
				cur.FaceShape[0] = uint8(parseInt(value))
			case "Facial Aesthetic":
				cur.FaceShape[1] = uint8(parseInt(value))
			}

		case "Facial Balance":
			switch param {
			case "Nose Size":
				cur.FaceShape[23] = uint8(parseInt(value))
			case "Nose/Forehead Ratio":
				cur.FaceShape[24] = uint8(parseInt(value))
			case "Face Protrusion":
				cur.FaceShape[26] = uint8(parseInt(value))
			case "Vertical Face Ratio":
				cur.FaceShape[27] = uint8(parseInt(value))
			case "Facial Feature Slant":
				cur.FaceShape[28] = uint8(parseInt(value))
			case "Horizontal Face Ratio":
				cur.FaceShape[29] = uint8(parseInt(value))
			}

		case "Forehead/Glabella":
			switch param {
			case "Forehead Depth":
				cur.FaceShape[31] = uint8(parseInt(value))
			case "Forehead Protrusion":
				cur.FaceShape[32] = uint8(parseInt(value))
			case "Nose Bridge Height":
				cur.FaceShape[58] = uint8(parseInt(value))
			case "Bridge Protrusion 1":
				cur.FaceShape[59] = uint8(parseInt(value))
			case "Bridge Protrusion 2":
				cur.FaceShape[60] = uint8(parseInt(value))
			case "Nose Bridge Width":
				cur.FaceShape[61] = uint8(parseInt(value))
			}

		case "Brow Ridge":
			switch param {
			case "Brow Ridge Height":
				cur.FaceShape[4] = uint8(parseInt(value))
			case "Inner Brow Ridge":
				cur.FaceShape[5] = uint8(parseInt(value))
			case "Outer Brow Ridge":
				cur.FaceShape[6] = uint8(parseInt(value))
			}

		case "Eyes":
			switch param {
			case "Eye Position":
				cur.FaceShape[19] = uint8(parseInt(value))
			case "Eye Size":
				cur.FaceShape[20] = uint8(parseInt(value))
			case "Eye Slant":
				cur.FaceShape[21] = uint8(parseInt(value))
			case "Eye Spacing":
				cur.FaceShape[22] = uint8(parseInt(value))
			}

		case "Nose Ridge":
			switch param {
			case "Nose Ridge Depth":
				cur.FaceShape[50] = uint8(parseInt(value))
			case "Nose Ridge Length":
				cur.FaceShape[51] = uint8(parseInt(value))
			case "Nose Position":
				cur.FaceShape[52] = uint8(parseInt(value))
			case "Nose Tip Height":
				cur.FaceShape[53] = uint8(parseInt(value))
			case "Nose Protrusion":
				cur.FaceShape[57] = uint8(parseInt(value))
			case "Nose Height":
				cur.FaceShape[62] = uint8(parseInt(value))
			case "Nose Slant":
				cur.FaceShape[63] = uint8(parseInt(value))
			}

		case "Nostrils":
			switch param {
			case "Nostril Slant":
				cur.FaceShape[54] = uint8(parseInt(value))
			case "Nostril Size":
				cur.FaceShape[55] = uint8(parseInt(value))
			case "Nostril Width":
				cur.FaceShape[56] = uint8(parseInt(value))
			}

		case "Cheeks":
			switch param {
			case "Cheekbone Height":
				cur.FaceShape[7] = uint8(parseInt(value))
			case "Cheekbone Depth":
				cur.FaceShape[8] = uint8(parseInt(value))
			case "Cheekbone Width":
				cur.FaceShape[9] = uint8(parseInt(value))
			case "Cheekbone Protrusion":
				cur.FaceShape[10] = uint8(parseInt(value))
			case "Cheeks":
				cur.FaceShape[11] = uint8(parseInt(value))
			}

		case "Lips":
			switch param {
			case "Lip Shape":
				cur.FaceShape[38] = uint8(parseInt(value))
			case "Mouth Expression":
				cur.FaceShape[41] = uint8(parseInt(value))
			case "Lip Fullness":
				cur.FaceShape[40] = uint8(parseInt(value))
			case "Lip Size":
				cur.FaceShape[39] = uint8(parseInt(value))
			case "Lip Protrusion":
				cur.FaceShape[42] = uint8(parseInt(value))
			case "Lip Thickness":
				cur.FaceShape[43] = uint8(parseInt(value))
			}

		case "Mouth":
			switch param {
			case "Mouth Protrusion":
				cur.FaceShape[44] = uint8(parseInt(value))
			case "Mouth Slant":
				cur.FaceShape[45] = uint8(parseInt(value))
			case "Occlusion":
				cur.FaceShape[46] = uint8(parseInt(value))
			case "Mouth Position":
				cur.FaceShape[47] = uint8(parseInt(value))
			case "Mouth Width":
				cur.FaceShape[48] = uint8(parseInt(value))
			case "Mouth-Chin Distance":
				cur.FaceShape[49] = uint8(parseInt(value))
			}

		case "Chin":
			switch param {
			case "Chin Tip Position":
				cur.FaceShape[12] = uint8(parseInt(value))
			case "Chin Length":
				cur.FaceShape[13] = uint8(parseInt(value))
			case "Chin Protrusion":
				cur.FaceShape[14] = uint8(parseInt(value))
			case "Chin Depth":
				cur.FaceShape[15] = uint8(parseInt(value))
			case "Chin Size":
				cur.FaceShape[16] = uint8(parseInt(value))
			case "Chin Height":
				cur.FaceShape[17] = uint8(parseInt(value))
			case "Chin Width":
				cur.FaceShape[18] = uint8(parseInt(value))
			}

		case "Jaw":
			switch param {
			case "Jaw Protrusion":
				cur.FaceShape[34] = uint8(parseInt(value))
			case "Jaw Width":
				cur.FaceShape[35] = uint8(parseInt(value))
			case "Lower Jaw":
				cur.FaceShape[36] = uint8(parseInt(value))
			case "Jaw Contour":
				cur.FaceShape[37] = uint8(parseInt(value))
			}

		case "Hair":
			switch param {
			case "Hair Style":
				cur.Models[1] = uint8(parseInt(value))
			case "Hair Color (RGB)":
				rgb := parseRGB(value)
				hairR, hairG, hairB = rgb[0], rgb[1], rgb[2]
				cur.Skin[67], cur.Skin[68], cur.Skin[69] = rgb[0], rgb[1], rgb[2]
			case "Hair Luster":
				v := uint8(parseInt(value))
				hairLuster = v
				cur.Skin[70] = v
			case "Hair Root Darkness":
				v := uint8(parseInt(value))
				hairRoot = v
				cur.Skin[71] = v
			case "Hair White Hairs":
				v := uint8(parseInt(value))
				hairWhite = v
				cur.Skin[72] = v
			}

		case "Eyebrows":
			switch param {
			case "Brow Style":
				cur.Models[3] = uint8(parseInt(value))
			case "Brow Color (RGB)":
				rgb := resolveMatchHair(value, hairR, hairG, hairB)
				cur.Skin[79], cur.Skin[80], cur.Skin[81] = rgb[0], rgb[1], rgb[2]
			case "Luster":
				cur.Skin[82] = resolveMatchHairSingle(value, hairLuster)
			case "Root Darkness":
				cur.Skin[83] = resolveMatchHairSingle(value, hairRoot)
			case "White Hairs":
				cur.Skin[84] = resolveMatchHairSingle(value, hairWhite)
			}

		case "Facial Hair":
			switch param {
			case "Beard Style":
				cur.Models[4] = uint8(parseInt(value))
			case "Beard Color (RGB)":
				rgb := resolveMatchHair(value, hairR, hairG, hairB)
				cur.Skin[73], cur.Skin[74], cur.Skin[75] = rgb[0], rgb[1], rgb[2]
			case "Stubble":
				cur.Skin[5] = uint8(parseInt(value))
			case "Luster":
				cur.Skin[76] = resolveMatchHairSingle(value, hairLuster)
			case "Root Darkness":
				cur.Skin[77] = resolveMatchHairSingle(value, hairRoot)
			case "White Hairs":
				cur.Skin[78] = resolveMatchHairSingle(value, hairWhite)
			}

		case "Eyelashes":
			switch param {
			case "Eyelashes":
				cur.Models[7] = uint8(parseInt(value))
			case "Eyelash Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[85], cur.Skin[86], cur.Skin[87] = rgb[0], rgb[1], rgb[2]
			}

		case "Right Eye":
			switch param {
			case "Iris Size":
				cur.Skin[46] = uint8(parseInt(value))
			case "Iris Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[43], cur.Skin[44], cur.Skin[45] = rgb[0], rgb[1], rgb[2]
			case "Clouding":
				cur.Skin[47] = uint8(parseInt(value))
			case "Clouding Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[48], cur.Skin[49], cur.Skin[50] = rgb[0], rgb[1], rgb[2]
			case "White Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[51], cur.Skin[52], cur.Skin[53] = rgb[0], rgb[1], rgb[2]
			case "Eye Position":
				cur.Skin[54] = uint8(parseInt(value))
			}

		case "Left Eye":
			switch param {
			case "Iris Size":
				cur.Skin[58] = uint8(parseInt(value))
			case "Iris Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[55], cur.Skin[56], cur.Skin[57] = rgb[0], rgb[1], rgb[2]
			case "Clouding":
				cur.Skin[59] = uint8(parseInt(value))
			case "Clouding Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[60], cur.Skin[61], cur.Skin[62] = rgb[0], rgb[1], rgb[2]
			case "White Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[63], cur.Skin[64], cur.Skin[65] = rgb[0], rgb[1], rgb[2]
			case "Eye Position":
				cur.Skin[66] = uint8(parseInt(value))
			}

		case "Skin Features":
			switch param {
			case "Pores":
				cur.Skin[4] = uint8(parseInt(value))
			case "Skin Luster":
				cur.Skin[3] = uint8(parseInt(value))
			case "Dark Circles":
				cur.Skin[6] = uint8(parseInt(value))
			case "Dark Circle Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[7], cur.Skin[8], cur.Skin[9] = rgb[0], rgb[1], rgb[2]
			}

		case "Cosmetics":
			switch param {
			case "Eyeliner":
				cur.Skin[14] = uint8(parseInt(value))
			case "Eyeliner Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[15], cur.Skin[16], cur.Skin[17] = rgb[0], rgb[1], rgb[2]
			case "Eyeshadow Upper":
				cur.Skin[22] = uint8(parseInt(value))
			case "Eyeshadow Upper Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[23], cur.Skin[24], cur.Skin[25] = rgb[0], rgb[1], rgb[2]
			case "Eyeshadow Lower":
				cur.Skin[18] = uint8(parseInt(value))
			case "Eyeshadow Lower Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[19], cur.Skin[20], cur.Skin[21] = rgb[0], rgb[1], rgb[2]
			case "Cheeks":
				cur.Skin[10] = uint8(parseInt(value))
			case "Cheek Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[11], cur.Skin[12], cur.Skin[13] = rgb[0], rgb[1], rgb[2]
			case "Lipstick":
				cur.Skin[26] = uint8(parseInt(value))
			case "Lipstick Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[27], cur.Skin[28], cur.Skin[29] = rgb[0], rgb[1], rgb[2]
			}

		case "Tattoo/Mark":
			switch param {
			case "Tattoo Mark":
				cur.Models[6] = uint8(parseInt(value))
			case "Tattoo Mark Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[34], cur.Skin[35], cur.Skin[36] = rgb[0], rgb[1], rgb[2]
			case "Eyepatch":
				cur.Models[5] = uint8(parseInt(value))
			case "Eyepatch Color (RGB)":
				rgb := parseRGB(value)
				cur.Skin[88], cur.Skin[89], cur.Skin[90] = rgb[0], rgb[1], rgb[2]
			case "Position Vertical":
				cur.Skin[31] = uint8(parseInt(value))
			case "Position Horizontal":
				cur.Skin[30] = uint8(parseInt(value))
			case "Angle":
				cur.Skin[32] = uint8(parseInt(value))
			case "Expansion":
				cur.Skin[33] = uint8(parseInt(value))
			case "Flip":
				if strings.ToUpper(value) == "ON" {
					cur.Skin[38] = 1
				}
			}

		case "Body":
			switch param {
			case "Head":
				cur.Body[0] = uint8(parseInt(value))
			case "Chest":
				cur.Body[1] = uint8(parseInt(value))
			case "Abdomen":
				cur.Body[2] = uint8(parseInt(value))
			case "Arms":
				v := uint8(parseInt(value))
				cur.Body[3] = v
				cur.Body[5] = v // arm_l = arm_r
			case "Legs":
				v := uint8(parseInt(value))
				cur.Body[4] = v
				cur.Body[6] = v // leg_l = leg_r
			case "Body Hair":
				cur.Skin[39] = uint8(parseInt(value))
			case "Body Hair Color":
				if strings.Contains(strings.ToLower(value), "match hair") {
					cur.Skin[40], cur.Skin[41], cur.Skin[42] = hairR, hairG, hairB
				} else {
					rgb := parseRGB(value)
					cur.Skin[40], cur.Skin[41], cur.Skin[42] = rgb[0], rgb[1], rgb[2]
				}
			}
		}
	}
	if cur != nil {
		presets = append(presets, *cur)
	}

	// Tattoo unk byte (index 37) defaults to 128
	for i := range presets {
		presets[i].Skin[37] = 128
	}

	// Generate Go source
	out, err := os.Create("backend/db/data/presets_generated.go")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	fmt.Fprintln(out, "// Code generated by scripts/parse_presets.go — DO NOT EDIT.")
	fmt.Fprintln(out, "package data")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "// GeneratedPresets contains all appearance presets parsed from tmp/characters/characters.md.")
	fmt.Fprintln(out, "var GeneratedPresets = []AppearancePreset{")
	for _, p := range presets {
		fmt.Fprintf(out, "\t{ // %s\n", p.Name)
		fmt.Fprintf(out, "\t\tName: %q, Image: %q,\n", p.Name, p.Image)
		fmt.Fprintf(out, "\t\tBodyType: %d, VoiceType: %d,\n", p.BodyType, p.VoiceType)
		fmt.Fprintf(out, "\t\tFaceModel: %d, HairModel: %d, EyeModel: %d, EyebrowModel: %d,\n",
			p.Models[0], p.Models[1], p.Models[2], p.Models[3])
		fmt.Fprintf(out, "\t\tBeardModel: %d, EyepatchModel: %d, DecalModel: %d, EyelashModel: %d,\n",
			p.Models[4], p.Models[5], p.Models[6], p.Models[7])
		fmt.Fprintf(out, "\t\tFaceShape: [64]uint8{%s},\n", formatArray(p.FaceShape[:]))
		fmt.Fprintf(out, "\t\tBody: [7]uint8{%s},\n", formatArray(p.Body[:]))
		fmt.Fprintf(out, "\t\tSkin: [91]uint8{%s},\n", formatArray(p.Skin[:]))
		fmt.Fprintln(out, "\t},")
	}
	fmt.Fprintln(out, "}")

	fmt.Printf("Generated %d presets → backend/db/data/presets_generated.go\n", len(presets))
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	v, _ := strconv.Atoi(s)
	return v
}

func parseRGB(s string) [3]uint8 {
	parts := strings.Split(s, ",")
	if len(parts) < 3 {
		return [3]uint8{}
	}
	return [3]uint8{
		uint8(parseInt(parts[0])),
		uint8(parseInt(parts[1])),
		uint8(parseInt(parts[2])),
	}
}

func resolveMatchHair(value string, r, g, b uint8) [3]uint8 {
	if strings.Contains(strings.ToLower(value), "match hair") {
		return [3]uint8{r, g, b}
	}
	return parseRGB(value)
}

func resolveMatchHairSingle(value string, hairVal uint8) uint8 {
	if strings.Contains(strings.ToLower(value), "match hair") {
		return hairVal
	}
	return uint8(parseInt(value))
}

func formatArray(a []uint8) string {
	parts := make([]string, len(a))
	for i, v := range a {
		parts[i] = strconv.Itoa(int(v))
	}
	return strings.Join(parts, ", ")
}

func toSlug(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
