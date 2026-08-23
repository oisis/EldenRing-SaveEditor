package data

// faceBoneUIToPartsID maps the Face / Bone Structure UI position (1-6, as shown
// in the character creator) to the raw PartsId stored in save files. The table
// is shared by Type A and Type B: both body types select the same six bone
// structures and store them at the same raw IDs, in steps of ten.
//
// Native evidence from controlled test saves:
//   - Type A: UI 1→0, 3→20, 4→30, 5→40, 6→50.
//   - Type B: UI 1→0, 2→10, 3→20, 4→30, 5→40, plus the previously confirmed 6→50.
//
// Type A UI 2 has no separate native Type A artifact and Type B UI 6 has no new
// native artifact; both follow from the shared table above together with the
// matching native value of the other body type. The user reviewed that evidence
// and approved the complete 1-6 table.
//
// There is no fallback: a UI value outside 1-6 is rejected.
var faceBoneUIToPartsID = map[uint8]uint8{1: 0, 2: 10, 3: 20, 4: 30, 5: 40, 6: 50}

// LookupFaceBonePartsID returns the raw save-file PartsId for a Face / Bone
// Structure UI value. It is the single source of truth for both Type A and
// Type B; ok is false for any UI value outside the verified 1-6 range.
func LookupFaceBonePartsID(ui uint8) (uint8, bool) {
	partsID, ok := faceBoneUIToPartsID[ui]
	return partsID, ok
}
