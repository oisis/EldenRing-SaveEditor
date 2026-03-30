package core

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

var MagicPattern = []byte{
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

type GaItem struct {
	Handle uint32
	ItemID uint32
}

type InventoryItem struct {
	GaItemHandle uint32
	Quantity     uint32
}

type EquipInventoryData struct {
	CommonItems []InventoryItem
	KeyItems    []InventoryItem
}

func (e *EquipInventoryData) Read(r *Reader, commonCount, keyCount int) {
	e.CommonItems = make([]InventoryItem, commonCount)
	for i := 0; i < commonCount; i++ {
		e.CommonItems[i].GaItemHandle, _ = r.ReadU32()
		e.CommonItems[i].Quantity, _ = r.ReadU32()
		r.ReadBytes(2) // Skip padding
	}
	e.KeyItems = make([]InventoryItem, keyCount)
	for i := 0; i < keyCount; i++ {
		e.KeyItems[i].GaItemHandle, _ = r.ReadU32()
		e.KeyItems[i].Quantity, _ = r.ReadU32()
		r.ReadBytes(2) // Skip padding
	}
}

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
}

type SaveSlot struct {
	Data    []byte
	Player  PlayerGameData
	GaMap   map[uint32]uint32 // Handle -> ItemID
	Inventory EquipInventoryData
	Storage   EquipInventoryData
	SteamID uint64
	
	MagicOffset      int
	InventoryEnd     int
	EventFlagsOffset int
}

func (s *SaveSlot) Read(r *Reader, platform string) error {
	var err error
	s.Data, err = r.ReadBytes(0x280000)
	if err != nil { return err }

	s.MagicOffset = NewReader(s.Data).FindPattern(MagicPattern)
	if s.MagicOffset == -1 {
		return fmt.Errorf("magic pattern not found")
	}

	s.mapStats()

	startGa := 0x310
	if platform == "PS4" { startGa = 0x20 }
	s.scanGaItems(startGa)

	// Inwentarz i Skrzynia mają stałe offsety relatywne do PlayerGameData lub Inwentarza
	// Dla uproszczenia i stabilności użyjemy offsetów, które Python wylicza dynamicznie
	s.mapInventory()

	if platform == "PC" {
		s.SteamID = binary.LittleEndian.Uint64(s.Data[0x280000-8:])
	}
	return nil
}

func (s *SaveSlot) mapStats() {
	mo := s.MagicOffset
	s.Player.Level = binary.LittleEndian.Uint32(s.Data[mo-335:])
	s.Player.Vigor = binary.LittleEndian.Uint32(s.Data[mo-379:])
	s.Player.Mind = binary.LittleEndian.Uint32(s.Data[mo-375:])
	s.Player.Endurance = binary.LittleEndian.Uint32(s.Data[mo-371:])
	s.Player.Strength = binary.LittleEndian.Uint32(s.Data[mo-367:])
	s.Player.Dexterity = binary.LittleEndian.Uint32(s.Data[mo-363:])
	s.Player.Intelligence = binary.LittleEndian.Uint32(s.Data[mo-359:])
	s.Player.Faith = binary.LittleEndian.Uint32(s.Data[mo-355:])
	s.Player.Arcane = binary.LittleEndian.Uint32(s.Data[mo-351:])
	s.Player.Souls = binary.LittleEndian.Uint32(s.Data[mo-331:])
	s.Player.Gender = s.Data[mo-249]
	s.Player.Class = s.Data[mo-248]

	nameOff := mo - 0x11b
	for i := 0; i < 16; i++ {
		s.Player.CharacterName[i] = binary.LittleEndian.Uint16(s.Data[nameOff+i*2:])
	}
}

func (s *SaveSlot) scanGaItems(start int) {
	s.GaMap = make(map[uint32]uint32)
	curr := start
	for i := 0; i < 5120; i++ {
		if curr+8 > len(s.Data) { break }
		handle := binary.LittleEndian.Uint32(s.Data[curr:])
		itemID := binary.LittleEndian.Uint32(s.Data[curr+4:])
		
		if handle != 0 && handle != 0xFFFFFFFF {
			s.GaMap[handle] = itemID
		}
		
		if itemID != 0 && (itemID&0xf0000000) == 0 {
			curr += 17
		} else if itemID != 0 && (itemID&0xf0000000) == 0x10000000 {
			curr += 16
		} else {
			curr += 8
		}
		s.InventoryEnd = curr
	}
}

func (s *SaveSlot) mapInventory() {
	if s.InventoryEnd == 0 { return }
	
	// Offsety oparte na łańcuchu z Pythona
	playerData := s.InventoryEnd + 0x1B0
	invStart := playerData + 0xD0 + 0x58 + 0x1C + 0x58 + 0x58
	
	ir := NewReader(s.Data)
	ir.Seek(int64(invStart), 0)
	s.Inventory.Read(ir, 0xa80, 0x180)
	
	// Skrzynia (Storage)
	faceData := invStart + (0xa80+0x180)*10 + 0x74 + 0x8c + 0x18
	// Python: storage_box = face_data + 0x6010
	storageStart := faceData + 0x6010
	ir.Seek(int64(storageStart), 0)
	s.Storage.Read(ir, 0x780, 0x80)
}

func (s *SaveSlot) Write(platform string) []byte {
	mo := s.MagicOffset
	binary.LittleEndian.PutUint32(s.Data[mo-335:], s.Player.Level)
	binary.LittleEndian.PutUint32(s.Data[mo-379:], s.Player.Vigor)
	binary.LittleEndian.PutUint32(s.Data[mo-375:], s.Player.Mind)
	binary.LittleEndian.PutUint32(s.Data[mo-371:], s.Player.Endurance)
	binary.LittleEndian.PutUint32(s.Data[mo-367:], s.Player.Strength)
	binary.LittleEndian.PutUint32(s.Data[mo-363:], s.Player.Dexterity)
	binary.LittleEndian.PutUint32(s.Data[mo-359:], s.Player.Intelligence)
	binary.LittleEndian.PutUint32(s.Data[mo-355:], s.Player.Faith)
	binary.LittleEndian.PutUint32(s.Data[mo-351:], s.Player.Arcane)
	binary.LittleEndian.PutUint32(s.Data[mo-331:], s.Player.Souls)
	s.Data[mo-249] = s.Player.Gender
	s.Data[mo-248] = s.Player.Class
	nameOff := mo - 0x11b
	for i := 0; i < 16; i++ {
		binary.LittleEndian.PutUint16(s.Data[nameOff+i*2:], s.Player.CharacterName[i])
	}
	if platform == "PC" {
		binary.LittleEndian.PutUint64(s.Data[0x280000-8:], s.SteamID)
	}
	return s.Data
}

type ProfileSummary struct {
	CharacterName [16]uint16
	Level         uint32
}

func (p *ProfileSummary) Read(r *Reader) error {
	for i := 0; i < 16; i++ {
		p.CharacterName[i], _ = r.ReadU16()
	}
	p.Level, _ = r.ReadU32()
	r.ReadBytes(0x100 - 36)
	return nil
}

type CSMenuSystemSaveLoad struct {
	Data []byte
}

func (c *CSMenuSystemSaveLoad) Read(r *Reader) {
	c.Data, _ = r.ReadBytes(0x60000)
}

func UTF16ToString(u16 []uint16) string {
	for i, v := range u16 {
		if v == 0 {
			u16 = u16[:i]
			break
		}
	}
	return string(utf16.Decode(u16))
}
