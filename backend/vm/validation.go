package vm

const PlayerGameDataOffset = 0x15420

// RecalculateLevel updates the character level based on current attributes.
// Formula: Level = Vigor + Mind + Endurance + Strength + Dexterity + Intelligence + Faith + Arcane - 79
func (vm *CharacterViewModel) RecalculateLevel() {
	// Base attributes sum for level 1 is 80 (standard for FromSoftware games)
	// But Elden Ring uses 79 as the offset for the sum of all attributes.
	sum := vm.Vigor + vm.Mind + vm.Endurance + vm.Strength + vm.Dexterity + vm.Intelligence + vm.Faith + vm.Arcane
	if sum > 79 {
		vm.Level = sum - 79
	} else {
		vm.Level = 1
	}
}

// UpdateMatchmakingLevel scans the inventory and updates the matchmaking weapon level.
// Located at PlayerGameDataOffset + 0x93 (0x154B3).
func UpdateMatchmakingLevel(slotData []byte, maxUpgrade uint8) {
	const MatchmakingLvlOffset = PlayerGameDataOffset + 0x93

	// If the current matchmaking level in the save is lower than the found max upgrade, update it.
	currentLvl := slotData[MatchmakingLvlOffset]
	if maxUpgrade > currentLvl {
		slotData[MatchmakingLvlOffset] = maxUpgrade
	}
}

// ValidateStats ensures all attributes are within legal game limits (1-99).
func (vm *CharacterViewModel) ValidateStats() {
	limit := func(val *uint32) {
		if *val > 99 {
			*val = 99
		}
		if *val < 1 {
			*val = 1
		}
	}
	limit(&vm.Vigor)
	limit(&vm.Mind)
	limit(&vm.Endurance)
	limit(&vm.Strength)
	limit(&vm.Dexterity)
	limit(&vm.Intelligence)
	limit(&vm.Faith)
	limit(&vm.Arcane)
}
