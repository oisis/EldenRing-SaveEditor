package data

// ContainerPickupFlags maps a container key item ID to the list of event flag
// IDs the game uses to gate world pickups for that container. Setting flag N
// (0-indexed) marks the (N+1)th world pickup as already collected, so the
// game won't offer that container key item to the player again — preventing
// stack duplication when the editor pre-fills the container.
//
// Flag groups (step 10) per er-save-manager event_flags_db.py:
//   Cracked Pot:       66000–66190 (20 used, 66200–66230 unused per param)
//   Ritual Pot:        66400–66490 (10 used, 66500–66590 unused)
//   Perfume Bottle:    66700–66790 (10 used)
//   Hefty Cracked Pot: 66900–66990 (10 used)
//
// Only "used" flags are listed (= active world pickup locations). The number
// of entries per container matches the container's MaxInventory cap, so
// auto-setting flags 1..MaxInventory covers every world pickup for that
// container and the editor never produces an inconsistent state.
var ContainerPickupFlags = map[uint32][]uint32{
	CrackedPotKeyItemID: {
		66000, 66010, 66020, 66030, 66040, 66050, 66060, 66070, 66080, 66090,
		66100, 66110, 66120, 66130, 66140, 66150, 66160, 66170, 66180, 66190,
	},
	RitualPotKeyItemID: {
		66400, 66410, 66420, 66430, 66440, 66450, 66460, 66470, 66480, 66490,
	},
	PerfumeBottleKeyItemID: {
		66700, 66710, 66720, 66730, 66740, 66750, 66760, 66770, 66780, 66790,
	},
	HeftyCrackedPotKeyItemID: {
		66900, 66910, 66920, 66930, 66940, 66950, 66960, 66970, 66980, 66990,
	},
}

// ContainerVendorPurchaseFlags maps a container key item ID to additional
// event flags that gate vendor sales. Separate from world-pickup flags —
// vendors track their own purchase state. Setting these flags removes the
// container from the listed vendor's stock for the current NG cycle.
//
// IMPORTANT: NG+ resets these flags (vanilla behavior). The editor cannot
// prevent NG+ from re-stocking vendors — that's by-design vanilla behavior.
//
// Source: er-save-manager quest_flags_db.py "Kale" → "Purchasing Cracked Pot".
// In vanilla, only Kale (Church of Elleh, Limgrave) sells a container key
// item, and only Cracked Pot. Ritual Pot, Hefty Cracked Pot, and Perfume
// Bottle have no merchant — only world pickups.
var ContainerVendorPurchaseFlags = map[uint32][]uint32{
	CrackedPotKeyItemID: {710580}, // Kale "Purchasing Cracked Pot"
}
