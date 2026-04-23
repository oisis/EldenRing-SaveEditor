package data

import "strings"

// CookbookData holds metadata for a cookbook item.
type CookbookData struct {
	Name     string
	Category string // series name for grouping
}

// Cookbooks maps event flag ID → cookbook metadata.
// Event flags sourced from ER-Save-Editor (Rust) cookbooks.rs.
var Cookbooks = map[uint32]CookbookData{
	// Nomadic Warrior's Cookbook (24)
	67000: {Name: "Nomadic Warrior's Cookbook [1]", Category: "Nomadic Warrior's Cookbook"},
	67110: {Name: "Nomadic Warrior's Cookbook [2]", Category: "Nomadic Warrior's Cookbook"},
	67010: {Name: "Nomadic Warrior's Cookbook [3]", Category: "Nomadic Warrior's Cookbook"},
	67800: {Name: "Nomadic Warrior's Cookbook [4]", Category: "Nomadic Warrior's Cookbook"},
	67830: {Name: "Nomadic Warrior's Cookbook [5]", Category: "Nomadic Warrior's Cookbook"},
	67020: {Name: "Nomadic Warrior's Cookbook [6]", Category: "Nomadic Warrior's Cookbook"},
	67050: {Name: "Nomadic Warrior's Cookbook [7]", Category: "Nomadic Warrior's Cookbook"},
	67880: {Name: "Nomadic Warrior's Cookbook [8]", Category: "Nomadic Warrior's Cookbook"},
	67430: {Name: "Nomadic Warrior's Cookbook [9]", Category: "Nomadic Warrior's Cookbook"},
	67030: {Name: "Nomadic Warrior's Cookbook [10]", Category: "Nomadic Warrior's Cookbook"},
	67220: {Name: "Nomadic Warrior's Cookbook [11]", Category: "Nomadic Warrior's Cookbook"},
	67060: {Name: "Nomadic Warrior's Cookbook [12]", Category: "Nomadic Warrior's Cookbook"},
	67080: {Name: "Nomadic Warrior's Cookbook [13]", Category: "Nomadic Warrior's Cookbook"},
	67870: {Name: "Nomadic Warrior's Cookbook [14]", Category: "Nomadic Warrior's Cookbook"},
	67900: {Name: "Nomadic Warrior's Cookbook [15]", Category: "Nomadic Warrior's Cookbook"},
	67290: {Name: "Nomadic Warrior's Cookbook [16]", Category: "Nomadic Warrior's Cookbook"},
	67100: {Name: "Nomadic Warrior's Cookbook [17]", Category: "Nomadic Warrior's Cookbook"},
	67270: {Name: "Nomadic Warrior's Cookbook [18]", Category: "Nomadic Warrior's Cookbook"},
	67070: {Name: "Nomadic Warrior's Cookbook [19]", Category: "Nomadic Warrior's Cookbook"},
	67230: {Name: "Nomadic Warrior's Cookbook [20]", Category: "Nomadic Warrior's Cookbook"},
	67120: {Name: "Nomadic Warrior's Cookbook [21]", Category: "Nomadic Warrior's Cookbook"},
	67890: {Name: "Nomadic Warrior's Cookbook [22]", Category: "Nomadic Warrior's Cookbook"},
	67090: {Name: "Nomadic Warrior's Cookbook [23]", Category: "Nomadic Warrior's Cookbook"},
	67910: {Name: "Nomadic Warrior's Cookbook [24]", Category: "Nomadic Warrior's Cookbook"},

	// Missionary's Cookbook (7)
	67610: {Name: "Missionary's Cookbook [1]", Category: "Missionary's Cookbook"},
	67600: {Name: "Missionary's Cookbook [2]", Category: "Missionary's Cookbook"},
	67650: {Name: "Missionary's Cookbook [3]", Category: "Missionary's Cookbook"},
	67640: {Name: "Missionary's Cookbook [4]", Category: "Missionary's Cookbook"},
	67630: {Name: "Missionary's Cookbook [5]", Category: "Missionary's Cookbook"},
	67130: {Name: "Missionary's Cookbook [6]", Category: "Missionary's Cookbook"},
	68230: {Name: "Missionary's Cookbook [7]", Category: "Missionary's Cookbook"},

	// Armorer's Cookbook (7)
	67200: {Name: "Armorer's Cookbook [1]", Category: "Armorer's Cookbook"},
	67210: {Name: "Armorer's Cookbook [2]", Category: "Armorer's Cookbook"},
	67280: {Name: "Armorer's Cookbook [3]", Category: "Armorer's Cookbook"},
	67260: {Name: "Armorer's Cookbook [4]", Category: "Armorer's Cookbook"},
	67310: {Name: "Armorer's Cookbook [5]", Category: "Armorer's Cookbook"},
	67300: {Name: "Armorer's Cookbook [6]", Category: "Armorer's Cookbook"},
	67250: {Name: "Armorer's Cookbook [7]", Category: "Armorer's Cookbook"},

	// Ancient Dragon Apostle's Cookbook (4)
	68000: {Name: "Ancient Dragon Apostle's Cookbook [1]", Category: "Ancient Dragon Apostle's Cookbook"},
	68010: {Name: "Ancient Dragon Apostle's Cookbook [2]", Category: "Ancient Dragon Apostle's Cookbook"},
	68030: {Name: "Ancient Dragon Apostle's Cookbook [3]", Category: "Ancient Dragon Apostle's Cookbook"},
	68020: {Name: "Ancient Dragon Apostle's Cookbook [4]", Category: "Ancient Dragon Apostle's Cookbook"},

	// Fevor's Cookbook (3)
	68200: {Name: "Fevor's Cookbook [1]", Category: "Fevor's Cookbook"},
	68220: {Name: "Fevor's Cookbook [2]", Category: "Fevor's Cookbook"},
	68210: {Name: "Fevor's Cookbook [3]", Category: "Fevor's Cookbook"},

	// Perfumer's Cookbook (4)
	67840: {Name: "Perfumer's Cookbook [1]", Category: "Perfumer's Cookbook"},
	67850: {Name: "Perfumer's Cookbook [2]", Category: "Perfumer's Cookbook"},
	67860: {Name: "Perfumer's Cookbook [3]", Category: "Perfumer's Cookbook"},
	67920: {Name: "Perfumer's Cookbook [4]", Category: "Perfumer's Cookbook"},

	// Glintstone Craftsman's Cookbook (8)
	67410: {Name: "Glintstone Craftsman's Cookbook [1]", Category: "Glintstone Craftsman's Cookbook"},
	67450: {Name: "Glintstone Craftsman's Cookbook [2]", Category: "Glintstone Craftsman's Cookbook"},
	67480: {Name: "Glintstone Craftsman's Cookbook [3]", Category: "Glintstone Craftsman's Cookbook"},
	67400: {Name: "Glintstone Craftsman's Cookbook [4]", Category: "Glintstone Craftsman's Cookbook"},
	67420: {Name: "Glintstone Craftsman's Cookbook [5]", Category: "Glintstone Craftsman's Cookbook"},
	67460: {Name: "Glintstone Craftsman's Cookbook [6]", Category: "Glintstone Craftsman's Cookbook"},
	67470: {Name: "Glintstone Craftsman's Cookbook [7]", Category: "Glintstone Craftsman's Cookbook"},
	67440: {Name: "Glintstone Craftsman's Cookbook [8]", Category: "Glintstone Craftsman's Cookbook"},

	// Frenzied's Cookbook (2)
	68400: {Name: "Frenzied's Cookbook [1]", Category: "Frenzied's Cookbook"},
	68410: {Name: "Frenzied's Cookbook [2]", Category: "Frenzied's Cookbook"},
}

// CookbookFlagToItemID maps cookbook event flag ID → inventory item ID (Key Items).
// Built by matching cookbook names between Cookbooks and KeyItems.
var CookbookFlagToItemID = map[uint32]uint32{
	// Nomadic Warrior's Cookbook [1-24]
	67000: 0x40002454, // [1]
	67110: 0x4000245F, // [2]
	67010: 0x40002455, // [3]
	67800: 0x400024A4, // [4]
	67830: 0x400024A7, // [5]
	67020: 0x40002456, // [6]
	67050: 0x40002459, // [7]
	67880: 0x400024AC, // [8]
	67430: 0x4000247F, // [9]
	67030: 0x40002457, // [10]
	67220: 0x4000246A, // [11]
	67060: 0x4000245A, // [12]
	67080: 0x4000245C, // [13]
	67870: 0x400024AB, // [14]
	67900: 0x400024AE, // [15]
	67290: 0x40002471, // [16]
	67100: 0x4000245E, // [17]
	67270: 0x4000246F, // [18]
	67070: 0x4000245B, // [19]
	67230: 0x4000246B, // [20]
	67120: 0x40002460, // [21]
	67890: 0x400024AD, // [22]
	67090: 0x4000245D, // [23]
	67910: 0x400024AF, // [24]

	// Missionary's Cookbook [1-7]
	67610: 0x40002491, // [1]
	67600: 0x40002490, // [2]
	67650: 0x40002495, // [3]
	67640: 0x40002494, // [4]
	67630: 0x40002493, // [5]
	67130: 0x40002461, // [6]
	68230: 0x400024CF, // [7]

	// Armorer's Cookbook [1-7]
	67200: 0x40002468, // [1]
	67210: 0x40002469, // [2]
	67280: 0x40002470, // [3]
	67260: 0x4000246E, // [4]
	67310: 0x40002473, // [5]
	67300: 0x40002472, // [6]
	67250: 0x4000246D, // [7]

	// Ancient Dragon Apostle's Cookbook [1-4]
	68000: 0x400024B8, // [1]
	68010: 0x400024B9, // [2]
	68030: 0x400024BB, // [3]
	68020: 0x400024BA, // [4]

	// Fevor's Cookbook [1-3]
	68200: 0x400024CC, // [1]
	68220: 0x400024CE, // [2]
	68210: 0x400024CD, // [3]

	// Perfumer's Cookbook [1-4]
	67840: 0x400024A8, // [1]
	67850: 0x400024A9, // [2]
	67860: 0x400024AA, // [3]
	67920: 0x400024B0, // [4]

	// Glintstone Craftsman's Cookbook [1-8]
	67410: 0x4000247D, // [1]
	67450: 0x40002481, // [2]
	67480: 0x40002484, // [3]
	67400: 0x4000247C, // [4]
	67420: 0x4000247E, // [5]
	67460: 0x40002482, // [6]
	67470: 0x40002483, // [7]
	67440: 0x40002480, // [8]

	// Frenzied's Cookbook [1-2]
	68400: 0x400024E0, // [1]
	68410: 0x400024E1, // [2]
}

// cookbookItemIDs is a set of all cookbook inventory item IDs for filtering.
var cookbookItemIDs map[uint32]bool

func init() {
	cookbookItemIDs = make(map[uint32]bool, len(CookbookFlagToItemID))
	for _, itemID := range CookbookFlagToItemID {
		cookbookItemIDs[itemID] = true
	}
}

// IsCookbookItemID returns true if the item ID is a cookbook Key Item.
// Checks both the mapped IDs and any Key Item with "Cookbook" in the name.
func IsCookbookItemID(id uint32) bool {
	if cookbookItemIDs[id] {
		return true
	}
	if item, ok := KeyItems[id]; ok {
		return strings.Contains(item.Name, "Cookbook")
	}
	return false
}
