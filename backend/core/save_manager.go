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
	UserData11    []byte       // 0x240010 bytes (Regulation + Rest)
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

// Write saves the current state to a file.
func (s *SaveFile) Write(path string) error {
	var buf bytes.Buffer

	if s.Platform == PlatformPC {
		// PC Write Logic
		binary.Write(&buf, binary.LittleEndian, &s.Header)
		padding := make([]byte, 0x300-buf.Len())
		buf.Write(padding)

		for i := 0; i < 10; i++ {
			buf.Write(s.SlotsMD5[i][:])
			buf.Write(s.Slots[i])
		}

		buf.Write(s.UserData10MD5[:])
		buf.Write(s.UserData10)

		buf.Write(s.UserData11MD5[:])
		buf.Write(s.UserData11)

		return os.WriteFile(path, buf.Bytes(), 0644)
	} else {
		// PS4 Write Logic (Raw)
		binary.Write(&buf, binary.LittleEndian, &s.Header)
		for i := 0; i < 10; i++ {
			buf.Write(s.Slots[i])
		}
		buf.Write(s.UserData10)
		buf.Write(s.UserData11)
		return os.WriteFile(path, buf.Bytes(), 0644)
	}
}

// GetActiveSlots returns a boolean array indicating which slots are active.
func (s *SaveFile) GetActiveSlots() []bool {
	active := make([]bool, 10)
	reader := bytes.NewReader(s.UserData10)
	reader.Seek(4+8+0x140, 1)
	var unk, length uint32
	if err := binary.Read(reader, binary.LittleEndian, &unk); err != nil { return active }
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil { return active }
	reader.Seek(int64(length), 1)
	for i := 0; i < 10; i++ {
		b, err := reader.ReadByte()
		if err != nil { break }
		active[i] = b == 1
	}
	return active
}

// SetSlotActivity toggles the active flag for a specific slot.
func (s *SaveFile) SetSlotActivity(index int, active bool) error {
	if index < 0 || index >= 10 {
		return fmt.Errorf("invalid slot index")
	}
	reader := bytes.NewReader(s.UserData10)
	reader.Seek(4+8+0x140, 1)
	var unk, length uint32
	binary.Read(reader, binary.LittleEndian, &unk)
	binary.Read(reader, binary.LittleEndian, &length)
	offset := 4 + 8 + 0x140 + 4 + 4 + int(length) + index
	val := byte(0)
	if active { val = 1 }
	s.UserData10[offset] = val
	return nil
}

// ImportSlot copies a character slot from source save to this save.
func (dest *SaveFile) ImportSlot(source *SaveFile, srcIdx, destIdx int) error {
	if srcIdx < 0 || srcIdx >= 10 || destIdx < 0 || destIdx >= 10 {
		return fmt.Errorf("invalid slot index")
	}
	newSlotData := make([]byte, 0x280000)
	copy(newSlotData, source.Slots[srcIdx])
	destVersion := dest.Slots[destIdx][:4]
	copy(newSlotData[:4], destVersion)
	dest.Slots[destIdx] = newSlotData
	return dest.SetSlotActivity(destIdx, true)
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
	save.UserData11 = make([]byte, 0x240010)
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

	save.UserData11 = make([]byte, 0x240010)
	if _, err := reader.Read(save.UserData11); err != nil {
		return nil, err
	}

	return save, nil
}
