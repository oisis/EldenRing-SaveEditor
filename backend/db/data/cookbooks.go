package data

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
