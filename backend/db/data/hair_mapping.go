package data

// MaleHairUIToPartsID maps male hair UI position (1-based, as shown in Mirror)
// to the internal PartsId stored in save files.
// Source: empirical mapping from save file analysis (tmp/save/ER0000-wlosy.sl2).
// Hair IDs are NOT sequential — the game sorts hair styles differently from their
// internal database IDs.
//
// Confirmed mappings:
//
// Base game (UI 1-9):
//
//	UI 1 → 0     UI 4 → 1     UI 7 → 5
//	UI 2 → 113   UI 5 → 3     UI 8 → 10
//	UI 3 → 112   UI 6 → 100   UI 9 → 101
//
// DLC (approximate UI 32-37):
//
//	UI 32 → 121   UI 35 → 120
//	UI 33 → 125   UI 36 → 123
//	UI 34 → 122   UI 37 → 124
//
// UI 10-31: unmapped — fallback to UI-1 (inaccurate)
var MaleHairUIToPartsID = map[uint8]uint8{
	// Base game (confirmed from save analysis, slots 0-8)
	1: 0,
	2: 113,
	3: 112,
	4: 1,
	5: 3,
	6: 100,
	7: 5,
	8: 10,
	9: 101,
	// UI 10-31: unmapped — need save with these styles
	// DLC / Shadow of the Erdtree (from slots 9-14, approximate UI positions 32-37)
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
