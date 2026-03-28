package core

import (
	"bytes"
	"encoding/binary"
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
	Platform      Platform
	Header        SaveHeader
	Slots         [10][]byte   // Raw slot data (0x280000 bytes each)
	SlotsMD5      [10][16]byte // PC only
	UserData10    []byte       // 0x60000 bytes
	UserData10MD5 [16]byte     // PC only
	UserData11    []byte       // 0x23FFF0 bytes (Regulation + Rest)
	UserData11MD5 [16]byte     // PC only
}

// LoadSave loads a save file from the given path, auto-detecting the platform.
func LoadSave(path string) (*SaveFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	save := &SaveFile{}

	if bytes.HasPrefix(data, []byte("BND4")) {
		save.Platform = PlatformPC
		return loadPC(data, save)
	}

	if len(data) >= 0x1960070+0x240010 {
		save.Platform = PlatformPS
		return loadPS(data, save)
	}

	return nil, fmt.Errorf("unknown save format")
}

// ImportSlot copies a character slot from source save to this save.
func (dest *SaveFile) ImportSlot(source *SaveFile, srcIdx, destIdx int) error {
	if srcIdx < 0 || srcIdx >= 10 || destIdx < 0 || destIdx >= 10 {
		return fmt.Errorf("invalid slot index")
	}

	// 1. Copy raw slot data
	newSlotData := make([]byte, 0x280000)
	copy(newSlotData, source.Slots[srcIdx])

	// 2. Overwrite version in the new slot with destination's version
	// Version is the first 4 bytes of the slot
	destVersion := dest.Slots[destIdx][:4]
	copy(newSlotData[:4], destVersion)

	dest.Slots[destIdx] = newSlotData

	// 3. Update UserData10 (Active Slot and Profile Summary)
	// This requires complex sequential parsing of UserData10 to find exact offsets.
	// Placeholder for UserData10 update logic.
	
	return nil
}

func loadPC(data []byte, save *SaveFile) (*SaveFile, error) {
	payload := data[0x30:]
	decrypted, err := DecryptSave(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt PC save: %w", err)
	}

	fullData := make([]byte, 0, 0x30+len(decrypted))
	fullData = append(fullData, data[:0x30]...)
	fullData = append(fullData, decrypted...)

	reader := bytes.NewReader(fullData)

	if err := binary.Read(reader, binary.LittleEndian, &save.Header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	reader.Seek(0x300, 0)

	for i := 0; i < 10; i++ {
		if err := binary.Read(reader, binary.LittleEndian, &save.SlotsMD5[i]); err != nil {
			return nil, err
		}
		save.Slots[i] = make([]byte, 0x280000)
		if _, err := reader.Read(save.Slots[i]); err != nil {
			return nil, err
		}
	}

	if err := binary.Read(reader, binary.LittleEndian, &save.UserData10MD5); err != nil {
		return nil, err
	}
	save.UserData10 = make([]byte, 0x60000)
	if _, err := reader.Read(save.UserData10); err != nil {
		return nil, err
	}

	if err := binary.Read(reader, binary.LittleEndian, &save.UserData11MD5); err != nil {
		return nil, err
	}
	save.UserData11 = make([]byte, 0x23FFF0)
	if _, err := reader.Read(save.UserData11); err != nil {
		return nil, err
	}

	return save, nil
}

func loadPS(data []byte, save *SaveFile) (*SaveFile, error) {
	reader := bytes.NewReader(data)

	if err := binary.Read(reader, binary.LittleEndian, &save.Header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	for i := 0; i < 10; i++ {
		save.Slots[i] = make([]byte, 0x280000)
		if _, err := reader.Read(save.Slots[i]); err != nil {
			return nil, err
		}
	}

	save.UserData10 = make([]byte, 0x60000)
	if _, err := reader.Read(save.UserData10); err != nil {
		return nil, err
	}

	save.UserData11 = make([]byte, 0x23FFF0)
	if _, err := reader.Read(save.UserData11); err != nil {
		return nil, err
	}

	return save, nil
}
