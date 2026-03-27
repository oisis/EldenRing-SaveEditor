package core

import (
	"encoding/binary"
	"fmt"
)

// UpdateUserDataSteamID updates the SteamID in the UserData10 block.
// On PC, it also recalculates the MD5 checksum of the block.
func UpdateUserDataSteamID(data []byte, newID uint64, isPC bool) error {
	offset := 4 // PS4 offset (Unk3B4)
	if isPC {
		if len(data) < 16+4+8 {
			return fmt.Errorf("UserData10 block too short")
		}
		offset = 16 + 4 // PC offset (MD5 + Unk3B4)
	}

	if len(data) < offset+8 {
		return fmt.Errorf("data buffer too small for SteamID update")
	}

	binary.LittleEndian.PutUint64(data[offset:], newID)

	if isPC {
		// Recalculate MD5 for the whole block (excluding the first 16 bytes of checksum)
		hash := ComputeMD5(data[16:])
		copy(data[:16], hash[:])
	}
	return nil
}

// UpdateSlotSteamID updates the SteamID in a SaveSlot block.
// Note: This requires the slot data to be decrypted (if PC).
func UpdateSlotSteamID(slotData []byte, newID uint64, isPC bool) error {
	// On PC, each slot is preceded by 16 bytes of MD5.
	// The SteamID offset within the 0x280000 data block needs to be exact.
	// Based on Rust analysis, it's located after event_flags and net_data_chunks.
	
	// TODO: Implement exact offset calculation after full slot structure mapping.
	// For now, this is a placeholder to be completed in Phase 4.
	return nil
}
