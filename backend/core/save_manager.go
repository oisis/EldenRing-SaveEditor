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

	// 1. Detect Platform (PC uses BND4 container)
	if bytes.HasPrefix(data, []byte("BND4")) {
		save.Platform = PlatformPC
		return loadPC(data, save)
	}

	// 2. Detect PlayStation (Save Wizard export)
	// Check for Regulation MD5 at offset 0x1960070
	if len(data) >= 0x1960070+0x240010 {
		save.Platform = PlatformPS
		return loadPS(data, save)
	}

	return nil, fmt.Errorf("unknown save format")
}

func loadPC(data []byte, save *SaveFile) (*SaveFile, error) {
	// PC saves are AES encrypted starting from offset 0x30
	// The first 16 bytes of the encrypted block are the IV
	payload := data[0x30:]
	decrypted, err := DecryptSave(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt PC save: %w", err)
	}

	// Reconstruct full data for easier parsing (Header is not encrypted)
	fullData := make([]byte, 0, 0x30+len(decrypted))
	fullData = append(fullData, data[:0x30]...)
	fullData = append(fullData, decrypted...)

	reader := bytes.NewReader(fullData)

	// Read Header (0x70 bytes)
	if err := binary.Read(reader, binary.LittleEndian, &save.Header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Jump to slots (PC slots start after 0x70 header + padding to 0x300?)
	// Based on Rust: save.header = SaveHeader::read(br)?; 
	// Then it immediately reads slots.
	reader.Seek(0x300, 0) // PC offset for slots

	// Read 10 Slots (MD5 + 0x280000 bytes each)
	for i := 0; i < 10; i++ {
		if err := binary.Read(reader, binary.LittleEndian, &save.SlotsMD5[i]); err != nil {
			return nil, err
		}
		save.Slots[i] = make([]byte, 0x280000)
		if _, err := reader.Read(save.Slots[i]); err != nil {
			return nil, err
		}
	}

	// Read UserData10 (MD5 + 0x60000)
	if err := binary.Read(reader, binary.LittleEndian, &save.UserData10MD5); err != nil {
		return nil, err
	}
	save.UserData10 = make([]byte, 0x60000)
	if _, err := reader.Read(save.UserData10); err != nil {
		return nil, err
	}

	// Read UserData11 (MD5 + 0x23FFF0)
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

	// Read Header (0x70 bytes)
	if err := binary.Read(reader, binary.LittleEndian, &save.Header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Read 10 Slots (0x280000 bytes each, no MD5)
	for i := 0; i < 10; i++ {
		save.Slots[i] = make([]byte, 0x280000)
		if _, err := reader.Read(save.Slots[i]); err != nil {
			return nil, err
		}
	}

	// Read UserData10 (0x60000 bytes)
	save.UserData10 = make([]byte, 0x60000)
	if _, err := reader.Read(save.UserData10); err != nil {
		return nil, err
	}

	// Read UserData11 (0x23FFF0 bytes)
	save.UserData11 = make([]byte, 0x23FFF0)
	if _, err := reader.Read(save.UserData11); err != nil {
		return nil, err
	}

	return save, nil
}
