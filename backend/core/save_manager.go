package core

import (
	"bytes"
	"fmt"
	"os"
)

type Platform string

const (
	PlatformPC Platform = "PC"
	PlatformPS Platform = "PS4"
)

// SaveFile represents the entire save file state in memory.
type SaveFile struct {
	Platform         Platform
	Slots            [10]SaveSlot
	ActiveSlots      [10]bool
	ProfileSummaries [10]ProfileSummary
}

// LoadSave loads a save file from the given path, auto-detecting the platform.
func LoadSave(path string) (*SaveFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	save := &SaveFile{}

	// 1. Detect Platform & Decrypt if needed
	if bytes.HasPrefix(data, []byte("BND4")) {
		save.Platform = PlatformPC
		return loadPCSequential(NewReader(data), save)
	}

	// Try decrypting (PC saves start with 16-byte IV)
	decrypted, err := DecryptSave(data)
	if err == nil && bytes.HasPrefix(decrypted, []byte("BND4")) {
		save.Platform = PlatformPC
		return loadPCSequential(NewReader(decrypted), save)
	}

	// Default to PS4
	save.Platform = PlatformPS
	return loadPSSequential(NewReader(data), save)
}

func loadPCSequential(r *Reader, save *SaveFile) (*SaveFile, error) {
	// 1. Skip Header (0x300)
	r.ReadBytes(0x300)

	// 2. Read 10 Slots
	for i := 0; i < 10; i++ {
		slotStart := r.Pos()
		// PC slots have a 16-byte MD5 checksum prefix
		r.ReadBytes(0x10)
		
		if err := save.Slots[i].Read(r); err != nil {
			return nil, fmt.Errorf("failed to read slot %d: %w", i, err)
		}
		// Each slot is exactly 0x280000 bytes (excluding checksum). Skip remainder.
		r.Seek(int64(slotStart+0x10+0x280000), 0)
	}

	// 3. Read UserData10
	// PC UserData10 also has a 16-byte MD5 checksum
	r.ReadBytes(0x10)
	
	r.ReadI32()       // _0x19003b4
	r.ReadU64()       // steam_id
	r.ReadBytes(0x140) // _0x19004fc
	
	var menu CSMenuSystemSaveLoad
	menu.Read(r)

	// Active Slots
	for i := 0; i < 10; i++ {
		b, _ := r.ReadU8()
		save.ActiveSlots[i] = b == 1
	}

	// Profile Summaries
	for i := 0; i < 10; i++ {
		save.ProfileSummaries[i].Read(r)
	}

	return save, nil
}

func loadPSSequential(r *Reader, save *SaveFile) (*SaveFile, error) {
	// 1. Skip Header (0x70)
	r.ReadBytes(0x70)

	// 2. Read 10 Slots
	for i := 0; i < 10; i++ {
		slotStart := r.Pos()
		if err := save.Slots[i].Read(r); err != nil {
			return nil, fmt.Errorf("failed to read slot %d: %w", i, err)
		}
		// Each slot is exactly 0x280000 bytes. Skip remainder.
		r.Seek(int64(slotStart+0x280000), 0)
	}

	// 3. Read UserData10
	r.ReadI32()       // _0x19003b4
	r.ReadU64()       // steam_id
	r.ReadBytes(0x140) // _0x19004fc
	
	var menu CSMenuSystemSaveLoad
	menu.Read(r)

	// Active Slots
	for i := 0; i < 10; i++ {
		b, _ := r.ReadU8()
		save.ActiveSlots[i] = b == 1
	}

	// Profile Summaries
	for i := 0; i < 10; i++ {
		save.ProfileSummaries[i].Read(r)
	}

	return save, nil
}

func (s *SaveFile) Write(path string) error {
	return fmt.Errorf("write not implemented in sequential mode yet")
}

// ImportSlot copies a slot and its metadata from another SaveFile.
func (s *SaveFile) ImportSlot(source *SaveFile, srcIdx, destIdx int) error {
	if srcIdx < 0 || srcIdx >= 10 || destIdx < 0 || destIdx >= 10 {
		return fmt.Errorf("invalid slot index")
	}
	
	// Copy slot data
	s.Slots[destIdx] = source.Slots[srcIdx]
	
	// Copy activity status
	s.ActiveSlots[destIdx] = source.ActiveSlots[srcIdx]
	
	// Copy profile summary
	s.ProfileSummaries[destIdx] = source.ProfileSummaries[srcIdx]
	
	return nil
}
