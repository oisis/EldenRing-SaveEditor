package data

// shields_subcat.go — sub-category assignment for the Shields tab.
//
// Sub-groups (in-game order):
//   1. Torches  (top of tab — left-hand light source)
//   2. Small Shields (bucklers, parry shields, light shields)
//   3. Medium Shields (kite, heater, crest)
//   4. Greatshields (towershields, full-block heavies)
//
// Classification rules (highest priority first):
//   1. Item name contains "Torch" or is "Torchpole" / "Lamenting Visage" → Torch
//   2. Base name (without infusion prefix) is in shieldsSmall set       → Small
//   3. Base name (without infusion prefix) is in shieldsGreatshield set → Greatshield
//   4. Otherwise                                                        → Medium
//
// Infusion prefixes stripped before lookup: Heavy / Keen / Quality / Fire /
// Flame Art / Lightning / Sacred / Magic / Cold / Poison / Blood / Occult /
// Bloody / Cracked. Multi-word base names (e.g., "Black Steel Greatshield")
// must appear verbatim in the lookup tables.
//
// To add a new shield in the future: only update one of the lookup sets —
// init() rebuilds SubCategory from the source-of-truth Name field.

import "strings"

var shieldInfusionPrefixes = []string{
	"Heavy ", "Keen ", "Quality ", "Fire ", "Flame Art ", "Lightning ",
	"Sacred ", "Magic ", "Cold ", "Poison ", "Blood ", "Occult ",
	"Bloody ", "Cracked ",
}

// shieldsSmall — base names that classify as Small Shields.
var shieldsSmall = map[string]struct{}{
	"Shield of Night":          {}, // DLC, parry shield
	"Buckler":                  {},
	"Perfumer's Shield":        {},
	"Man-Serpent's Shield":     {},
	"Rickety Shield":           {},
	"Pillory Shield":           {},
	"Beastman's Jar-Shield":    {},
	"Red Thorn Roundshield":    {},
	"Scripture Wooden Shield":  {},
	"Riveted Wooden Shield":    {},
	"Blue-White Wooden Shield": {},
	"Rift Shield":              {},
	"Iron Roundshield":         {},
	"Gilded Iron Shield":       {},
	"Ice Crest Shield":         {},
	"Smoldering Shield":        {},
	"Spiralhorn Shield":        {},
	"Coil Shield":              {},
	"Smithscript Shield":       {}, // DLC
	"Dueling Shield":           {}, // DLC
	"Carian Thrusting Shield":  {}, // DLC
}

// shieldsGreatshield — base names that classify as Greatshields.
var shieldsGreatshield = map[string]struct{}{
	"Dragon Towershield":           {},
	"Distinguished Greatshield":    {},
	"Crucible Hornshield":          {},
	"Dragonclaw Shield":            {},
	"Briar Greatshield":            {},
	"Erdtree Greatshield":          {},
	"Golden Beast Crest Shield":    {},
	"Jellyfish Shield":             {},
	"Fingerprint Stone Shield":     {},
	"Icon Shield":                  {},
	"One-Eyed Shield":              {},
	"Visage Shield":                {},
	"Spiked Palisade Shield":       {},
	"Manor Towershield":            {},
	"Crossed-Tree Towershield":     {},
	"Inverted Hawk Towershield":    {},
	"Redmane Greatshield":          {},
	"Eclipse Crest Greatshield":    {},
	"Cuckoo Greatshield":           {},
	"Golden Greatshield":           {},
	"Gilded Greatshield":           {},
	"Haligtree Crest Greatshield":  {},
	"Wooden Greatshield":           {},
	"Lordsworn's Shield":           {},
	"Black Steel Greatshield":      {}, // DLC
	"Verdigris Greatshield":        {}, // DLC
	"Great Turtle Shell":           {},
	"Ant's Skull Plate":            {},
}

// stripShieldInfusionPrefix returns the base shield name with any leading
// infusion qualifier removed (e.g., "Heavy Wolf Crest Shield" → "Wolf Crest Shield").
func stripShieldInfusionPrefix(name string) string {
	for _, p := range shieldInfusionPrefixes {
		if strings.HasPrefix(name, p) {
			return name[len(p):]
		}
	}
	return name
}

func classifyShield(name string) string {
	if strings.Contains(name, "Torch") || name == "Torchpole" || name == "Lamenting Visage" {
		return SubcatShieldsTorches
	}
	base := stripShieldInfusionPrefix(name)
	if _, ok := shieldsSmall[base]; ok {
		return SubcatShieldsSmall
	}
	if _, ok := shieldsGreatshield[base]; ok {
		return SubcatShieldsGreatshields
	}
	return SubcatShieldsMedium
}

func init() {
	for id, item := range Shields {
		if item.SubCategory != "" {
			continue
		}
		item.SubCategory = classifyShield(item.Name)
		Shields[id] = item
	}
}
