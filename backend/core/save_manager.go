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
	reader := NewReader(data)

	// 1. Detect Platform
	if bytes.HasPrefix(data, []byte("BND4")) {
		save.Platform = PlatformPC
		return nil, fmt.Errorf("PC support temporarily disabled for PS4 fix")
	} else {
		save.Platform = PlatformPS
		return loadPSSequential(reader, save)
	}
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
