package data

// BellBearingItemToFlagID maps Bell Bearing inventory item ID → acquisition event flag ID.
// When a BB is added to inventory via the editor, set this flag so Twin Maiden Husks
// recognise the BB as legitimately turned in (expands their wares).
// Flag IDs come from BellBearings (er-save-manager event_flags_db.py).
// 59 entries matched by exact name; 3 by manual alias (Kale, Spell-Machinist, String-seller).
// Cut-content item Nomadic [11] (0x400022E6, flagged ban_risk) is intentionally excluded.
var BellBearingItemToFlagID = map[uint32]uint32{
	0x400022CE: 11109710, // Pidia's Bell Bearing
	0x400022CF: 11109711, // Seluvis's Bell Bearing
	0x400022D0: 11109712, // Patches' Bell Bearing
	0x400022D1: 11109713, // Sellen's Bell Bearing
	0x400022D3: 11109715, // D's Bell Bearing
	0x400022D4: 11109716, // Bernahl's Bell Bearing
	0x400022D5: 11109717, // Miriel's Bell Bearing
	0x400022D6: 11109718, // Gostoc's Bell Bearing
	0x400022D7: 11109719, // Thops's Bell Bearing
	0x400022D8: 11109720, // Kalé's Bell Bearing
	0x400022D9: 11109721, // Nomadic Merchant's Bell Bearing [1]
	0x400022DA: 11109722, // Nomadic Merchant's Bell Bearing [2]
	0x400022DB: 11109723, // Nomadic Merchant's Bell Bearing [3]
	0x400022DC: 11109724, // Nomadic Merchant's Bell Bearing [4]
	0x400022DD: 11109725, // Nomadic Merchant's Bell Bearing [5]
	0x400022DE: 11109726, // Isolated Merchant's Bell Bearing [1]
	0x400022DF: 11109727, // Isolated Merchant's Bell Bearing [2]
	0x400022E0: 11109728, // Nomadic Merchant's Bell Bearing [6]
	0x400022E1: 11109729, // Hermit Merchant's Bell Bearing [1]
	0x400022E2: 11109730, // Nomadic Merchant's Bell Bearing [7]
	0x400022E3: 11109731, // Nomadic Merchant's Bell Bearing [8]
	0x400022E4: 11109732, // Nomadic Merchant's Bell Bearing [9]
	0x400022E5: 11109733, // Nomadic Merchant's Bell Bearing [10]
	0x400022E7: 11109735, // Isolated Merchant's Bell Bearing [3]
	0x400022E8: 11109736, // Hermit Merchant's Bell Bearing [2]
	0x400022E9: 11109737, // Abandoned Merchant's Bell Bearing
	0x400022EA: 11109738, // Hermit Merchant's Bell Bearing [3]
	0x400022EB: 11109739, // Imprisoned Merchant's Bell Bearing
	0x400022EC: 11109740, // Iji's Bell Bearing
	0x400022ED: 11109741, // Rogier's Bell Bearing
	0x400022EE: 11109742, // Blackguard's Bell Bearing
	0x400022EF: 11109743, // Corhyn's Bell Bearing
	0x400022F0: 11109744, // Gowry's Bell Bearing
	0x400022F1: 11109745, // Bone Peddler's Bell Bearing
	0x400022F2: 11109746, // Meat Peddler's Bell Bearing
	0x400022F3: 11109747, // Medicine Peddler's Bell Bearing
	0x400022F4: 11109748, // Gravity Stone Peddler's Bell Bearing
	0x400022F7: 11109751, // Smithing-Stone Miner's Bell Bearing [1]
	0x400022F8: 11109752, // Smithing-Stone Miner's Bell Bearing [2]
	0x400022F9: 11109753, // Smithing-Stone Miner's Bell Bearing [3]
	0x400022FA: 11109754, // Smithing-Stone Miner's Bell Bearing [4]
	0x400022FB: 11109755, // Somberstone Miner's Bell Bearing [1]
	0x400022FC: 11109756, // Somberstone Miner's Bell Bearing [2]
	0x400022FD: 11109757, // Somberstone Miner's Bell Bearing [3]
	0x400022FE: 11109758, // Somberstone Miner's Bell Bearing [4]
	0x400022FF: 11109759, // Somberstone Miner's Bell Bearing [5]
	0x40002300: 11109760, // Glovewort Picker's Bell Bearing [1]
	0x40002301: 11109761, // Glovewort Picker's Bell Bearing [2]
	0x40002302: 11109762, // Glovewort Picker's Bell Bearing [3]
	0x40002303: 11109763, // Ghost-Glovewort Picker's Bell Bearing [1]
	0x40002304: 11109764, // Ghost-Glovewort Picker's Bell Bearing [2]
	0x40002305: 11109765, // Ghost-Glovewort Picker's Bell Bearing [3]
	0x401EA744: 11109790, // Moore's Bell Bearing
	0x401EA745: 11109791, // Ymir's Bell Bearing
	0x401EA746: 11109792, // Herbalist's Bell Bearing
	0x401EA747: 11109793, // Mushroom-Seller's Bell Bearing [1]
	0x401EA748: 11109794, // Mushroom-Seller's Bell Bearing [2]
	0x401EA749: 11109795, // Greasemonger's Bell Bearing
	0x401EA74A: 11109796, // Moldmonger's Bell Bearing
	0x401EA74B: 11109797, // Igon's Bell Bearing
	0x401EA74C: 11109798, // Spellmachinist's Bell Bearing
	0x401EA74D: 11109799, // String-Seller's Bell Bearing
}

// BellBearingFlagToItemID is the reverse of BellBearingItemToFlagID.
// Used by World → Unlocks toggle to add/remove the corresponding key item
// from inventory when the acquisition flag is flipped.
var BellBearingFlagToItemID = func() map[uint32]uint32 {
	out := make(map[uint32]uint32, len(BellBearingItemToFlagID))
	for itemID, flagID := range BellBearingItemToFlagID {
		out[flagID] = itemID
	}
	return out
}()
