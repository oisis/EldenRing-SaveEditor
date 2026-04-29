package data

// AboutTutorialID maps "About *" / Note / tutorial-trigger Goods item ID →
// TutorialParam row ID that the game appends to slot's TutorialData list when
// the corresponding popup fires.
//
// Mechanism (verified via save diff at slot 5, 2026-04-29):
//   • EquipParamGoods.itemGetTutorialFlagId is 0 for About items themselves,
//     non-zero for "trigger" items (e.g. Crafting Kit 8500 → flag 710570).
//   • When the game would give an About item, it first checks slot's
//     TutorialData chunk. If the row ID is already in the list → skip the
//     give (= no inventory add → no on-ground duplicate when cap=1).
//   • Therefore: pre-populating the list with the tutorial row ID prevents
//     the world copy from spawning when player has the inv copy already.
//
// This map is grown empirically from save diffs. Each entry is one verified
// "trigger event → tutorial row added" observation. Unverified inferred
// entries may be added with the comment "(inferred)" — to be confirmed.
//
// EMPTY for now — populate as we test each About item.
//
// First confirmed entry (pending in-game verification):
//   • 0x40002399 (About Item Crafting) → 2010 — observed when player bought
//     Crafting Kit at Kalé; ID 2010 was the only new entry appended to the
//     TutorialData list in the save diff.
var AboutTutorialID = map[uint32]uint32{
	0x40002399: 2010, // About Item Crafting — empirically verified via save diff
}

// HasTutorialMapping returns true if we know which TutorialParam row to
// append for the given Goods item ID.
func HasTutorialMapping(id uint32) bool {
	_, ok := AboutTutorialID[id]
	return ok
}
