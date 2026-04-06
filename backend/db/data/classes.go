package data

type StarterStats struct {
	Level, Vigor, Mind, Endurance, Strength, Dexterity, Intelligence, Faith, Arcane uint32
}

var StarterClasses = map[uint8]StarterStats{
	0: {Level: 9, Vigor: 15, Mind: 10, Endurance: 11, Strength: 14, Dexterity: 13, Intelligence: 9, Faith: 9, Arcane: 7},    // Vagabond
	1: {Level: 8, Vigor: 11, Mind: 12, Endurance: 11, Strength: 10, Dexterity: 16, Intelligence: 10, Faith: 8, Arcane: 9},   // Warrior
	2: {Level: 7, Vigor: 14, Mind: 9, Endurance: 12, Strength: 16, Dexterity: 9, Intelligence: 7, Faith: 8, Arcane: 11},     // Hero
	3: {Level: 5, Vigor: 10, Mind: 11, Endurance: 10, Strength: 9, Dexterity: 13, Intelligence: 9, Faith: 8, Arcane: 14},    // Bandit
	4: {Level: 6, Vigor: 9, Mind: 15, Endurance: 9, Strength: 8, Dexterity: 12, Intelligence: 16, Faith: 7, Arcane: 9},      // Astrologer
	5: {Level: 7, Vigor: 10, Mind: 14, Endurance: 8, Strength: 11, Dexterity: 10, Intelligence: 7, Faith: 16, Arcane: 10},   // Prophet
	7: {Level: 9, Vigor: 12, Mind: 11, Endurance: 13, Strength: 12, Dexterity: 15, Intelligence: 9, Faith: 8, Arcane: 8},    // Samurai
	8: {Level: 9, Vigor: 11, Mind: 12, Endurance: 11, Strength: 11, Dexterity: 14, Intelligence: 14, Faith: 6, Arcane: 9},   // Prisoner
	6: {Level: 10, Vigor: 10, Mind: 13, Endurance: 10, Strength: 12, Dexterity: 12, Intelligence: 9, Faith: 14, Arcane: 9},  // Confessor
	9: {Level: 1, Vigor: 10, Mind: 10, Endurance: 10, Strength: 10, Dexterity: 10, Intelligence: 10, Faith: 10, Arcane: 10}, // Wretch
}
