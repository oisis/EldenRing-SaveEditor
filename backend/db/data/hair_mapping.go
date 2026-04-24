package data

// MaleHairUIToPartsID maps male hair UI position (1-based, as shown in Mirror)
// to the internal PartsId stored in save files.
// Source: empirical mapping from save file analysis (tmp/save/ER0000-wlosy.sl2).
// Hair IDs are NOT sequential — the game sorts hair styles differently from their
// internal database IDs.
//
// Confirmed mappings (all 37 styles):
//
// Base game (UI 1-9, from save slot analysis):
//
//	UI 1 → 0     UI 4 → 1     UI 7 → 5
//	UI 2 → 113   UI 5 → 3     UI 8 → 10
//	UI 3 → 112   UI 6 → 100   UI 9 → 101
//
// Base game (UI 10-31, from Mirror Favorites preset extraction):
//
//	UI 10 → 9     UI 14 → 115   UI 18 → 102   UI 22 → 106   UI 26 → 111   UI 30 → 118
//	UI 11 → 8     UI 15 → 114   UI 19 → 103   UI 23 → 107   UI 27 → 110   UI 31 → 116
//	UI 12 → 6     UI 16 → 2     UI 20 → 104   UI 24 → 109   UI 28 → 117
//	UI 13 → 7     UI 17 → 4     UI 21 → 105   UI 25 → 108   UI 29 → 119
//
// DLC / Shadow of the Erdtree (UI 32-37, confirmed):
//
//	UI 32 → 121   UI 34 → 122   UI 36 → 123
//	UI 33 → 125   UI 35 → 120   UI 37 → 124
var MaleHairUIToPartsID = map[uint8]uint8{
	// Base game (UI 1-9, from save slot analysis)
	1: 0,
	2: 113,
	3: 112,
	4: 1,
	5: 3,
	6: 100,
	7: 5,
	8: 10,
	9: 101,
	// Base game (UI 10-31, from Mirror Favorites preset extraction)
	10: 9,
	11: 8,
	12: 6,
	13: 7,
	14: 115,
	15: 114,
	16: 2,
	17: 4,
	18: 102,
	19: 103,
	20: 104,
	21: 105,
	22: 106,
	23: 107,
	24: 109,
	25: 108,
	26: 111,
	27: 110,
	28: 117,
	29: 119,
	30: 118,
	31: 116,
	// DLC / Shadow of the Erdtree (UI 32-37, confirmed)
	32: 121,
	33: 125,
	34: 122,
	35: 120,
	36: 123,
	37: 124,
}

// LookupMaleHairPartsID returns the save-file PartsId for a male hair UI index.
// Returns (partsId, true) if found, or (0, false) if the UI index is not in the mapping.
func LookupMaleHairPartsID(uiIndex uint8) (uint8, bool) {
	id, ok := MaleHairUIToPartsID[uiIndex]
	return id, ok
}
