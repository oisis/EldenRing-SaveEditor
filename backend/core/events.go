package core

import (
	"fmt"
)

// EventFlags offset in SaveSlot: starts after many other structures.
// Based on analiza_rusta.md, it's a large bit array.
// In the original Rust code, it's located at a specific offset.
const EventFlagsOffset = 0x1D700 // Approximate offset for EventFlags in Elden Ring save slots

// SetEventFlag sets a specific bit in the EventFlags array
func (s *SaveSlot) SetEventFlag(flagID uint32, enabled bool) {
	// Elden Ring uses a specific mapping for flagID to byte/bit.
	// Most flags follow: byte = flagID / 8, bit = flagID % 8
	// However, some categories might have offsets. 
	// For Graces and Bosses, this simple mapping is usually correct within the EventFlags block.
	
	byteIdx := flagID / 8
	bitIdx := flagID % 8
	
	// Ensure we are within bounds of the Data array
	// EventFlags is a large part of the 0x280000 slot.
	actualOffset := EventFlagsOffset + int(byteIdx)
	if actualOffset >= len(s.Data) {
		fmt.Printf("Warning: Flag ID %d (offset %d) out of bounds\n", flagID, actualOffset)
		return
	}

	if enabled {
		s.Data[actualOffset] |= (1 << bitIdx)
	} else {
		s.Data[actualOffset] &= ^(1 << bitIdx)
	}
}

// GetEventFlag checks if a specific bit is set
func (s *SaveSlot) GetEventFlag(flagID uint32) bool {
	byteIdx := flagID / 8
	bitIdx := flagID % 8
	actualOffset := EventFlagsOffset + int(byteIdx)
	
	if actualOffset >= len(s.Data) {
		return false
	}
	
	return (s.Data[actualOffset] & (1 << bitIdx)) != 0
}
