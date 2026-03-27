package vm

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// CharacterViewModel represents the editable character data for the UI.
type CharacterViewModel struct {
	Name         string `json:"name"`
	Level        uint32 `json:"level"`
	Souls        uint32 `json:"souls"`
	Vigor        uint32 `json:"vigor"`
	Mind         uint32 `json:"mind"`
	Endurance    uint32 `json:"endurance"`
	Strength     uint32 `json:"strength"`
	Dexterity    uint32 `json:"dexterity"`
	Intelligence uint32 `json:"intelligence"`
	Faith        uint32 `json:"faith"`
	Arcane       uint32 `json:"arcane"`
}

const (
	PlayerGameDataOffset = 0x15420
	NameOffset           = PlayerGameDataOffset + 0x94
	StatsOffset          = PlayerGameDataOffset + 0x34
	LevelOffset          = PlayerGameDataOffset + 0x68
	SoulsOffset          = PlayerGameDataOffset + 0x6C
)

// MapSlotToVM extracts character data from raw slot bytes.
func MapSlotToVM(slotData []byte) (*CharacterViewModel, error) {
	if len(slotData) < PlayerGameDataOffset+0x200 {
		return nil, fmt.Errorf("slot data too short")
	}

	vm := &CharacterViewModel{}

	// Read Stats (Attributes)
	vm.Vigor = binary.LittleEndian.Uint32(slotData[StatsOffset+0x00:])
	vm.Mind = binary.LittleEndian.Uint32(slotData[StatsOffset+0x04:])
	vm.Endurance = binary.LittleEndian.Uint32(slotData[StatsOffset+0x08:])
	vm.Strength = binary.LittleEndian.Uint32(slotData[StatsOffset+0x0C:])
	vm.Dexterity = binary.LittleEndian.Uint32(slotData[StatsOffset+0x10:])
	vm.Intelligence = binary.LittleEndian.Uint32(slotData[StatsOffset+0x14:])
	vm.Faith = binary.LittleEndian.Uint32(slotData[StatsOffset+0x18:])
	vm.Arcane = binary.LittleEndian.Uint32(slotData[StatsOffset+0x1C:])

	// Read Level and Souls
	vm.Level = binary.LittleEndian.Uint32(slotData[LevelOffset:])
	vm.Souls = binary.LittleEndian.Uint32(slotData[SoulsOffset:])

	// Read Name (UTF-16, 16 characters max)
	nameRaw := slotData[NameOffset : NameOffset+32]
	u16 := make([]uint16, 16)
	for i := 0; i < 16; i++ {
		val := binary.LittleEndian.Uint16(nameRaw[i*2:])
		if val == 0 {
			u16 = u16[:i]
			break
		}
		u16[i] = val
	}
	vm.Name = string(utf16.Decode(u16))

	return vm, nil
}

// ApplyVMToSlot writes the ViewModel data back to raw slot bytes.
func ApplyVMToSlot(vm *CharacterViewModel, slotData []byte) error {
	if len(slotData) < PlayerGameDataOffset+0x200 {
		return fmt.Errorf("slot data too short")
	}

	// Write Stats
	binary.LittleEndian.PutUint32(slotData[StatsOffset+0x00:], vm.Vigor)
	binary.LittleEndian.PutUint32(slotData[StatsOffset+0x04:], vm.Mind)
	binary.LittleEndian.PutUint32(slotData[StatsOffset+0x08:], vm.Endurance)
	binary.LittleEndian.PutUint32(slotData[StatsOffset+0x0C:], vm.Strength)
	binary.LittleEndian.PutUint32(slotData[StatsOffset+0x10:], vm.Dexterity)
	binary.LittleEndian.PutUint32(slotData[StatsOffset+0x14:], vm.Intelligence)
	binary.LittleEndian.PutUint32(slotData[StatsOffset+0x18:], vm.Faith)
	binary.LittleEndian.PutUint32(slotData[StatsOffset+0x1C:], vm.Arcane)

	// Write Level and Souls
	binary.LittleEndian.PutUint32(slotData[LevelOffset:], vm.Level)
	binary.LittleEndian.PutUint32(slotData[SoulsOffset:], vm.Souls)

	// Write Name (UTF-16)
	nameU16 := utf16.Encode([]rune(vm.Name))
	for i := 0; i < 16; i++ {
		var val uint16
		if i < len(nameU16) {
			val = nameU16[i]
		}
		binary.LittleEndian.PutUint16(slotData[NameOffset+i*2:], val)
	}

	return nil
}
