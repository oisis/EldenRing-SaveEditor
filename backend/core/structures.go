package core

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// MagicPattern is the bit pattern used to locate character stats (PlayerGameData).
// It's a long sequence of 00 FF FF FF FF... found in every save slot.
var MagicPattern = []byte{
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF,
}

// GaItem represents an inventory item with dynamic size.
type GaItem struct {
	Handle uint32
	ItemID uint32
	Unk2   int32
	Unk3   int32
	AoW    uint32
	Unk5   uint8
}

func (g *GaItem) Read(r *Reader) error {
	var err error
	g.Handle, err = r.ReadU32()
	if err != nil { return err }
	g.ItemID, err = r.ReadU32()
	if err != nil { return err }

	// Weapon Logic: (id != 0 && (id & 0xf0000000) == 0)
	if g.ItemID != 0 && (g.ItemID&0xf0000000) == 0 {
		g.Unk2, _ = r.ReadI32()
		g.Unk3, _ = r.ReadI32()
		g.AoW, _ = r.ReadU32()
		g.Unk5, _ = r.ReadU8()
	} else if g.ItemID != 0 && (g.ItemID&0xf0000000) == 0x10000000 {
		// Armor Logic: (id != 0 && (id & 0xf0000000) == 0x10000000)
		g.Unk2, _ = r.ReadI32()
		g.Unk3, _ = r.ReadI32()
	}
	return nil
}

func (g *GaItem) Write(w *Writer) error {
	w.WriteU32(g.Handle)
	w.WriteU32(g.ItemID)

	if g.ItemID != 0 && (g.ItemID&0xf0000000) == 0 {
		w.WriteI32(g.Unk2)
		w.WriteI32(g.Unk3)
		w.WriteU32(g.AoW)
		w.WriteU8(g.Unk5)
	} else if g.ItemID != 0 && (g.ItemID&0xf0000000) == 0x10000000 {
		w.WriteI32(g.Unk2)
		w.WriteI32(g.Unk3)
	}
	return nil
}

// PlayerGameData represents character stats mapped via relative offsets from MagicPattern.
type PlayerGameData struct {
	Level         uint32
	Vigor         uint32
	Mind          uint32
	Endurance     uint32
	Strength      uint32
	Dexterity     uint32
	Intelligence  uint32
	Faith         uint32
	Arcane        uint32
	Souls         uint32
	CharacterName [16]uint16
	Gender        uint8
	Class         uint8
	
	// Offsets relative to the start of the slot
	MagicOffset int
}

func (p *PlayerGameData) Read(r *Reader, slotStart int) error {
	// 1. Find Magic Pattern within the slot
	magicPos := r.FindPattern(MagicPattern)
	if magicPos == -1 {
		return fmt.Errorf("magic pattern not found in slot")
	}
	p.MagicOffset = magicPos

	// 2. Read Stats using relative offsets from Python project
	// Distances: Level:-335, Vigor:-379, Mind:-375, Endurance:-371, Strength:-367, 
	// Dexterity:-363, Intelligence:-359, Faith:-355, Arcane:-351, Souls:-331, Gender:-249, Class:-248
	
	p.Level = binary.LittleEndian.Uint32(r.data[magicPos-335:])
	p.Vigor = binary.LittleEndian.Uint32(r.data[magicPos-379:])
	p.Mind = binary.LittleEndian.Uint32(r.data[magicPos-375:])
	p.Endurance = binary.LittleEndian.Uint32(r.data[magicPos-371:])
	p.Strength = binary.LittleEndian.Uint32(r.data[magicPos-367:])
	p.Dexterity = binary.LittleEndian.Uint32(r.data[magicPos-363:])
	p.Intelligence = binary.LittleEndian.Uint32(r.data[magicPos-359:])
	p.Faith = binary.LittleEndian.Uint32(r.data[magicPos-355:])
	p.Arcane = binary.LittleEndian.Uint32(r.data[magicPos-351:])
	p.Souls = binary.LittleEndian.Uint32(r.data[magicPos-331:])
	
	p.Gender = r.data[magicPos-249]
	p.Class = r.data[magicPos-248]

	// 3. Read Character Name (magic_offset - 0x11b)
	nameOffset := magicPos - 0x11b
	for i := 0; i < 16; i++ {
		p.CharacterName[i] = binary.LittleEndian.Uint16(r.data[nameOffset+i*2:])
	}

	return nil
}

func (p *PlayerGameData) Write(data []byte) {
	// Write stats back using the same relative offsets
	binary.LittleEndian.PutUint32(data[p.MagicOffset-335:], p.Level)
	binary.LittleEndian.PutUint32(data[p.MagicOffset-379:], p.Vigor)
	binary.LittleEndian.PutUint32(data[p.MagicOffset-375:], p.Mind)
	binary.LittleEndian.PutUint32(data[p.MagicOffset-371:], p.Endurance)
	binary.LittleEndian.PutUint32(data[p.MagicOffset-367:], p.Strength)
	binary.LittleEndian.PutUint32(data[p.MagicOffset-363:], p.Dexterity)
	binary.LittleEndian.PutUint32(data[p.MagicOffset-359:], p.Intelligence)
	binary.LittleEndian.PutUint32(data[p.MagicOffset-355:], p.Faith)
	binary.LittleEndian.PutUint32(data[p.MagicOffset-351:], p.Arcane)
	binary.LittleEndian.PutUint32(data[p.MagicOffset-331:], p.Souls)
	
	data[p.MagicOffset-249] = p.Gender
	data[p.MagicOffset-248] = p.Class

	nameOffset := p.MagicOffset - 0x11b
	for i := 0; i < 16; i++ {
		binary.LittleEndian.PutUint16(data[nameOffset+i*2:], p.CharacterName[i])
	}
}

// SaveSlot represents a character slot. We now store the raw data to preserve 
// everything except the parts we explicitly modify.
type SaveSlot struct {
	Data    []byte
	Player  PlayerGameData
	SteamID uint64
}

func (s *SaveSlot) Read(r *Reader, platform string) error {
	var err error
	s.Data, err = r.ReadBytes(0x280000)
	if err != nil {
		return err
	}

	// Use a temporary reader for the slot data to find patterns
	slotReader := NewReader(s.Data)
	if err := s.Player.Read(slotReader, 0); err != nil {
		return err
	}

	// SteamID is at the end of the slot on PC
	if platform == "PC" {
		// Based on Python's logic, it's near the end. 
		// In our previous spec it was at the very end before padding.
		s.SteamID = binary.LittleEndian.Uint64(s.Data[0x280000-8:])
	}

	return nil
}

func (s *SaveSlot) Write(platform string) []byte {
	// Update player data in the raw byte slice
	s.Player.Write(s.Data)

	if platform == "PC" {
		binary.LittleEndian.PutUint64(s.Data[0x280000-8:], s.SteamID)
	}

	return s.Data
}

// ProfileSummary represents character info in the main menu.
type ProfileSummary struct {
	CharacterName [16]uint16
	Level         uint32
	Souls         uint32 // Added to match Python's more complete view
	Data          []byte // Raw data for the rest
}

func (p *ProfileSummary) Read(r *Reader) error {
	start := r.Pos()
	for i := 0; i < 16; i++ {
		p.CharacterName[i], _ = r.ReadU16()
	}
	p.Level, _ = r.ReadU32()
	// Skip some bytes to get to souls if needed, or just store raw
	r.Seek(int64(start), 0)
	p.Data, _ = r.ReadBytes(0x100) // Profile summaries are usually around this size
	return nil
}

// CSMenuSystemSaveLoad matches the UserData10 block.
type CSMenuSystemSaveLoad struct {
	Data []byte
}

func (c *CSMenuSystemSaveLoad) Read(r *Reader) {
	c.Data, _ = r.ReadBytes(0x60000)
}

// UTF16ToString converts a UTF-16 slice to a Go string, trimming the null terminator.
func UTF16ToString(u16 []uint16) string {
	for i, v := range u16 {
		if v == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16))
}
