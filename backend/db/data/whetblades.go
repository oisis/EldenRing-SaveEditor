package data

// WhetbladeData holds metadata for a whetblade unlock.
type WhetbladeData struct {
	Name string
}

// Whetblades maps event flag ID → whetblade metadata.
// Whetblades unlock weapon affinities at the smithing table.
// Source: er-save-manager/event_flags_db.py
var Whetblades = map[uint32]WhetbladeData{
	65610: {Name: "Iron Whetblade"},
	65640: {Name: "Red-Hot Whetblade"},
	65660: {Name: "Sanctified Whetblade"},
	65680: {Name: "Glintstone Whetblade"},
	65700: {Name: "Black Whetblade"},
}
