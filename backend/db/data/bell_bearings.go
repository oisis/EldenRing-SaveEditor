package data

// BellBearingData holds metadata for a bell bearing.
type BellBearingData struct {
	Name     string
	Category string // "npc", "merchant", "smithing", "peddler", "dlc"
}

// BellBearings maps event flag ID → bell bearing metadata.
// Source: er-save-manager/event_flags_db.py
var BellBearings = map[uint32]BellBearingData{
	// NPC Bell Bearings
	11109710: {Name: "Pidia's Bell Bearing", Category: "npc"},
	11109711: {Name: "Seluvis's Bell Bearing", Category: "npc"},
	11109712: {Name: "Patches' Bell Bearing", Category: "npc"},
	11109713: {Name: "Sellen's Bell Bearing", Category: "npc"},
	11109715: {Name: "D's Bell Bearing", Category: "npc"},
	11109716: {Name: "Bernahl's Bell Bearing", Category: "npc"},
	11109717: {Name: "Miriel's Bell Bearing", Category: "npc"},
	11109718: {Name: "Gostoc's Bell Bearing", Category: "npc"},
	11109719: {Name: "Thops's Bell Bearing", Category: "npc"},
	11109720: {Name: "Kale's Bell Bearing", Category: "npc"},
	11109740: {Name: "Iji's Bell Bearing", Category: "npc"},
	11109741: {Name: "Rogier's Bell Bearing", Category: "npc"},
	11109742: {Name: "Blackguard's Bell Bearing", Category: "npc"},
	11109743: {Name: "Corhyn's Bell Bearing", Category: "npc"},
	11109744: {Name: "Gowry's Bell Bearing", Category: "npc"},

	// Merchant Bell Bearings
	11109721: {Name: "Nomadic Merchant's Bell Bearing [1]", Category: "merchant"},
	11109722: {Name: "Nomadic Merchant's Bell Bearing [2]", Category: "merchant"},
	11109723: {Name: "Nomadic Merchant's Bell Bearing [3]", Category: "merchant"},
	11109724: {Name: "Nomadic Merchant's Bell Bearing [4]", Category: "merchant"},
	11109725: {Name: "Nomadic Merchant's Bell Bearing [5]", Category: "merchant"},
	11109726: {Name: "Isolated Merchant's Bell Bearing [1]", Category: "merchant"},
	11109727: {Name: "Isolated Merchant's Bell Bearing [2]", Category: "merchant"},
	11109728: {Name: "Nomadic Merchant's Bell Bearing [6]", Category: "merchant"},
	11109729: {Name: "Hermit Merchant's Bell Bearing [1]", Category: "merchant"},
	11109730: {Name: "Nomadic Merchant's Bell Bearing [7]", Category: "merchant"},
	11109731: {Name: "Nomadic Merchant's Bell Bearing [8]", Category: "merchant"},
	11109732: {Name: "Nomadic Merchant's Bell Bearing [9]", Category: "merchant"},
	11109733: {Name: "Nomadic Merchant's Bell Bearing [10]", Category: "merchant"},
	11109735: {Name: "Isolated Merchant's Bell Bearing [3]", Category: "merchant"},
	11109736: {Name: "Hermit Merchant's Bell Bearing [2]", Category: "merchant"},
	11109737: {Name: "Abandoned Merchant's Bell Bearing", Category: "merchant"},
	11109738: {Name: "Hermit Merchant's Bell Bearing [3]", Category: "merchant"},
	11109739: {Name: "Imprisoned Merchant's Bell Bearing", Category: "merchant"},

	// Peddler Bell Bearings
	11109745: {Name: "Bone Peddler's Bell Bearing", Category: "peddler"},
	11109746: {Name: "Meat Peddler's Bell Bearing", Category: "peddler"},
	11109747: {Name: "Medicine Peddler's Bell Bearing", Category: "peddler"},
	11109748: {Name: "Gravity Stone Peddler's Bell Bearing", Category: "peddler"},

	// Smithing / Somber / Glovewort Bell Bearings
	11109751: {Name: "Smithing-Stone Miner's Bell Bearing [1]", Category: "smithing"},
	11109752: {Name: "Smithing-Stone Miner's Bell Bearing [2]", Category: "smithing"},
	11109753: {Name: "Smithing-Stone Miner's Bell Bearing [3]", Category: "smithing"},
	11109754: {Name: "Smithing-Stone Miner's Bell Bearing [4]", Category: "smithing"},
	11109755: {Name: "Somberstone Miner's Bell Bearing [1]", Category: "smithing"},
	11109756: {Name: "Somberstone Miner's Bell Bearing [2]", Category: "smithing"},
	11109757: {Name: "Somberstone Miner's Bell Bearing [3]", Category: "smithing"},
	11109758: {Name: "Somberstone Miner's Bell Bearing [4]", Category: "smithing"},
	11109759: {Name: "Somberstone Miner's Bell Bearing [5]", Category: "smithing"},
	11109760: {Name: "Glovewort Picker's Bell Bearing [1]", Category: "smithing"},
	11109761: {Name: "Glovewort Picker's Bell Bearing [2]", Category: "smithing"},
	11109762: {Name: "Glovewort Picker's Bell Bearing [3]", Category: "smithing"},
	11109763: {Name: "Ghost-Glovewort Picker's Bell Bearing [1]", Category: "smithing"},
	11109764: {Name: "Ghost-Glovewort Picker's Bell Bearing [2]", Category: "smithing"},
	11109765: {Name: "Ghost-Glovewort Picker's Bell Bearing [3]", Category: "smithing"},

	// DLC Bell Bearings
	11109790: {Name: "Moore's Bell Bearing", Category: "dlc"},
	11109791: {Name: "Ymir's Bell Bearing", Category: "dlc"},
	11109792: {Name: "Herbalist's Bell Bearing", Category: "dlc"},
	11109793: {Name: "Mushroom-Seller's Bell Bearing [1]", Category: "dlc"},
	11109794: {Name: "Mushroom-Seller's Bell Bearing [2]", Category: "dlc"},
	11109795: {Name: "Greasemonger's Bell Bearing", Category: "dlc"},
	11109796: {Name: "Moldmonger's Bell Bearing", Category: "dlc"},
	11109797: {Name: "Igon's Bell Bearing", Category: "dlc"},
	11109798: {Name: "Spell-Machinist Bell Bearing", Category: "dlc"},
	11109799: {Name: "String-seller's Bell Bearing", Category: "dlc"},
}
