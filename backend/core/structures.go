package core

import (
	"bytes"
	"encoding/binary"
)

// SaveHeader is the 0x70 byte header of the save file
type SaveHeader [0x70]byte

// PlayerGameData contains character stats and name
// Offset in SaveSlot: 0x15420
type PlayerGameData struct {
	Unk00          [0x94]byte
	CharacterName  [32]byte // UTF-16, 16 characters max
	UnkB4          [0x01]byte
	UnkB5          [0x01]byte
	UnkB6          [0x01]byte
	UnkB7          [0x01]byte
	Level          uint32
	UnkBC          [0x04]byte
	Vigor          uint32
	Mind           uint32
	Endurance      uint32
	Strength       uint32
	Dexterity      uint32
	Intelligence   uint32
	Faith          uint32
	Arcane         uint32
	UnkDC          [0x04]byte
	HP             uint32
	UnkE4          [0x04]byte
	FP             uint32
	UnkEC          [0x04]byte
	SP             uint32
	UnkF4          [0x0C]byte
	Souls          uint32
	TotalSouls     uint32
	Unk108         [0x18]byte
}

// GaItem represents an item in the game's global inventory
// Size: 17 bytes
type GaItem struct {
	Handle uint32 // Unique ID for this instance of the item
	ItemID uint32 // The ID of the item from the game database
	Unk08  uint32
	Unk0C  uint32
	Unk10  byte
}

// SaveSlot is the 0x280000 byte block for a single character
type SaveSlot struct {
	Version uint32
	MapID   [4]byte
	Unk08   [0x18]byte
	// GaItems: 5120 items * 17 bytes = 87040 (0x15400)
	GaItems [5120]GaItem
	Data    [0x280000 - 0x20 - 87040]byte
}

// GetPlayerGameData extracts PlayerGameData from the raw SaveSlot data
func (s *SaveSlot) GetPlayerGameData() (*PlayerGameData, error) {
	// PlayerGameData starts at 0x15420 in the slot
	// Our Data field now starts exactly at 0x15420 (0x20 + 0x15400)
	pgd := &PlayerGameData{}
	size := binary.Size(pgd)
	pgdData := s.Data[0:size]
	
	reader := bytes.NewReader(pgdData)
	if err := binary.Read(reader, binary.LittleEndian, pgd); err != nil {
		return nil, err
	}
	return pgd, nil
}

// SetPlayerGameData writes PlayerGameData back to the raw SaveSlot data
func (s *SaveSlot) SetPlayerGameData(pgd *PlayerGameData) error {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, pgd); err != nil {
		return err
	}
	copy(s.Data[0:], buf.Bytes())
	return nil
}

// PCSaveSlot includes the 16-byte MD5 checksum
type PCSaveSlot struct {
	Checksum [16]byte
	Slot     SaveSlot
}

// ProfileSummary contains basic info for the load game menu
type ProfileSummary struct {
	CharacterName [32]byte
	Level         uint32
	Unk24         [0x0C]byte
}

// UserData10 contains account info and slot status
type UserData10 struct {
	Checksum       [16]byte
	Unk10          [0x04]byte
	SteamID        uint64
	Unk1C          [0x04]byte
	ActiveSlots    [10]byte // 0x01 = active, 0x00 = empty
	Unk2A          [0x12]byte
	ProfileSummary [10]ProfileSummary
	Unk262         [0x5FD9E]byte
}

// UserData11 contains regulation data
type UserData11 struct {
	Checksum [16]byte
	Data     [0x240000]byte
}

// PCSave represents the entire decrypted PC save file
type PCSave struct {
	Header     SaveHeader
	Slots      [10]PCSaveSlot
	UserData10 UserData10
	UserData11 UserData11
}

func (s *PCSave) Read(data []byte) error {
	reader := bytes.NewReader(data)
	return binary.Read(reader, binary.LittleEndian, s)
}

func (s *PCSave) Write() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
