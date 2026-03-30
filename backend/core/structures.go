package core

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

// GaItem2 represents an item in the GaItemData block.
type GaItem2 struct {
	ID            uint32
	Unk           uint32
	ReinforceType uint32
	Unk1          uint32
}

func (g *GaItem2) Read(r *Reader) error {
	var err error
	g.ID, err = r.ReadU32()
	if err != nil { return err }
	g.Unk, err = r.ReadU32()
	if err != nil { return err }
	g.ReinforceType, err = r.ReadU32()
	if err != nil { return err }
	g.Unk1, err = r.ReadU32()
	if err != nil { return err }
	return nil
}

func (g *GaItem2) Write(w *Writer) error {
	w.WriteU32(g.ID)
	w.WriteU32(g.Unk)
	w.WriteU32(g.ReinforceType)
	w.WriteU32(g.Unk1)
	return nil
}

// GaItemData represents the acquired items database in the save slot.
type GaItemData struct {
	AcquiredCount int32
	Unk1          int32
	Items         [0x1b58]GaItem2
}

func (g *GaItemData) Read(r *Reader) error {
	var err error
	g.AcquiredCount, err = r.ReadI32()
	if err != nil { return err }
	g.Unk1, err = r.ReadI32()
	if err != nil { return err }
	for i := 0; i < 0x1b58; i++ {
		g.Items[i].Read(r)
	}
	return nil
}

func (g *GaItemData) Write(w *Writer) error {
	w.WriteI32(g.AcquiredCount)
	w.WriteI32(g.Unk1)
	for i := 0; i < 0x1b58; i++ {
		g.Items[i].Write(w)
	}
	return nil
}

// PlayerGameData represents character stats.
type PlayerGameData struct {
	Header         []byte // 8 bytes
	Health         uint32
	MaxHealth      uint32
	BaseMaxHealth  uint32
	FP             uint32
	MaxFP          uint32
	BaseMaxFP      uint32
	Unk1           int32
	SP             uint32
	MaxSP          uint32
	BaseMaxSP      uint32
	Unk2           int32
	Vigor          uint32
	Mind           uint32
	Endurance      uint32
	Strength       uint32
	Dexterity      uint32
	Intelligence   uint32
	Faith          uint32
	Arcane         uint32
	Unk3           []byte // 12 bytes
	Level          uint32
	Souls          uint32
	SoulsMemory    uint32
	Unk4           []byte // 0x28 bytes
	CharacterName  [16]uint16
	Gender         uint8
	Class          uint8
	Unk5           []byte // 128 bytes (gift, match_making_wpn_lvl, passwords)
}

func (p *PlayerGameData) Read(r *Reader) error {
	p.Header, _ = r.ReadBytes(8)
	p.Health, _ = r.ReadU32()
	p.MaxHealth, _ = r.ReadU32()
	p.BaseMaxHealth, _ = r.ReadU32()
	p.FP, _ = r.ReadU32()
	p.MaxFP, _ = r.ReadU32()
	p.BaseMaxFP, _ = r.ReadU32()
	p.Unk1, _ = r.ReadI32()
	p.SP, _ = r.ReadU32()
	p.MaxSP, _ = r.ReadU32()
	p.BaseMaxSP, _ = r.ReadU32()
	p.Unk2, _ = r.ReadI32()
	p.Vigor, _ = r.ReadU32()
	p.Mind, _ = r.ReadU32()
	p.Endurance, _ = r.ReadU32()
	p.Strength, _ = r.ReadU32()
	p.Dexterity, _ = r.ReadU32()
	p.Intelligence, _ = r.ReadU32()
	p.Faith, _ = r.ReadU32()
	p.Arcane, _ = r.ReadU32()
	p.Unk3, _ = r.ReadBytes(12)
	p.Level, _ = r.ReadU32()
	p.Souls, _ = r.ReadU32()
	p.SoulsMemory, _ = r.ReadU32()
	p.Unk4, _ = r.ReadBytes(0x28)
	for i := 0; i < 16; i++ {
		p.CharacterName[i], _ = r.ReadU16()
	}
	p.Gender, _ = r.ReadU8()
	p.Class, _ = r.ReadU8()
	p.Unk5, _ = r.ReadBytes(128)
	return nil
}

func (p *PlayerGameData) Write(w *Writer) error {
	w.WriteBytes(p.Header)
	w.WriteU32(p.Health)
	w.WriteU32(p.MaxHealth)
	w.WriteU32(p.BaseMaxHealth)
	w.WriteU32(p.FP)
	w.WriteU32(p.MaxFP)
	w.WriteU32(p.BaseMaxFP)
	w.WriteI32(p.Unk1)
	w.WriteU32(p.SP)
	w.WriteU32(p.MaxSP)
	w.WriteU32(p.BaseMaxSP)
	w.WriteI32(p.Unk2)
	w.WriteU32(p.Vigor)
	w.WriteU32(p.Mind)
	w.WriteU32(p.Endurance)
	w.WriteU32(p.Strength)
	w.WriteU32(p.Dexterity)
	w.WriteU32(p.Intelligence)
	w.WriteU32(p.Faith)
	w.WriteU32(p.Arcane)
	w.WriteBytes(p.Unk3)
	w.WriteU32(p.Level)
	w.WriteU32(p.Souls)
	w.WriteU32(p.SoulsMemory)
	w.WriteBytes(p.Unk4)
	for i := 0; i < 16; i++ {
		w.WriteU16(p.CharacterName[i])
	}
	w.WriteU8(p.Gender)
	w.WriteU8(p.Class)
	w.WriteBytes(p.Unk5)
	return nil
}

// EquipInventoryItem represents an item entry in the inventory list.
type EquipInventoryItem struct {
	GaItemHandle   uint32
	Quantity       uint32
	InventoryIndex uint32
}

func (e *EquipInventoryItem) Read(r *Reader) error {
	var err error
	e.GaItemHandle, err = r.ReadU32()
	if err != nil { return err }
	e.Quantity, err = r.ReadU32()
	if err != nil { return err }
	e.InventoryIndex, err = r.ReadU32()
	if err != nil { return err }
	return nil
}

func (e *EquipInventoryItem) Write(w *Writer) error {
	w.WriteU32(e.GaItemHandle)
	w.WriteU32(e.Quantity)
	w.WriteU32(e.InventoryIndex)
	return nil
}

// EquipInventoryData represents the inventory and storage lists.
type EquipInventoryData struct {
	CommonCount uint32
	CommonItems []EquipInventoryItem
	KeyCount    uint32
	KeyItems    []EquipInventoryItem
	Unk1        uint32
	Unk2        uint32
}

func (e *EquipInventoryData) Read(r *Reader, commonLen, keyLen int) error {
	e.CommonCount, _ = r.ReadU32()
	e.CommonItems = make([]EquipInventoryItem, commonLen)
	for i := 0; i < commonLen; i++ {
		e.CommonItems[i].Read(r)
	}
	e.KeyCount, _ = r.ReadU32()
	e.KeyItems = make([]EquipInventoryItem, keyLen)
	for i := 0; i < keyLen; i++ {
		e.KeyItems[i].Read(r)
	}
	e.Unk1, _ = r.ReadU32()
	e.Unk2, _ = r.ReadU32()
	return nil
}

func (e *EquipInventoryData) Write(w *Writer) error {
	w.WriteU32(e.CommonCount)
	for i := 0; i < len(e.CommonItems); i++ {
		e.CommonItems[i].Write(w)
	}
	w.WriteU32(e.KeyCount)
	for i := 0; i < len(e.KeyItems); i++ {
		e.KeyItems[i].Write(w)
	}
	w.WriteU32(e.Unk1)
	w.WriteU32(e.Unk2)
	return nil
}

// ProfileSummary represents character data in the menu.
type ProfileSummary struct {
	CharacterName [17]uint16
	Level         uint32
	Unk1          []byte // 20 bytes
	Padding       []byte // 288 bytes
	EquipGaitem   []byte // 120 bytes
	EquipItem     []byte // 112 bytes
	Unk2          []byte // 6 bytes
	Unk3          int32
}

func (p *ProfileSummary) Read(r *Reader) error {
	for i := 0; i < 17; i++ {
		p.CharacterName[i], _ = r.ReadU16()
	}
	p.Level, _ = r.ReadU32()
	p.Unk1, _ = r.ReadBytes(20)
	p.Padding, _ = r.ReadBytes(288)
	p.EquipGaitem, _ = r.ReadBytes(120)
	p.EquipItem, _ = r.ReadBytes(112)
	p.Unk2, _ = r.ReadBytes(6)
	p.Unk3, _ = r.ReadI32()
	return nil
}

func (p *ProfileSummary) Write(w *Writer) error {
	for i := 0; i < 17; i++ {
		w.WriteU16(p.CharacterName[i])
	}
	w.WriteU32(p.Level)
	w.WriteBytes(p.Unk1)
	w.WriteBytes(p.Padding)
	w.WriteBytes(p.EquipGaitem)
	w.WriteBytes(p.EquipItem)
	w.WriteBytes(p.Unk2)
	w.WriteI32(p.Unk3)
	return nil
}

// CSMenuSystemSaveLoad represents dynamic menu data.
type CSMenuSystemSaveLoad struct {
	Unk    uint32
	Length uint32
	Data   []byte
}

func (c *CSMenuSystemSaveLoad) Read(r *Reader) error {
	c.Unk, _ = r.ReadU32()
	len, _ := r.ReadU32()
	c.Length = len
	c.Data, _ = r.ReadBytes(int(len))
	return nil
}

func (c *CSMenuSystemSaveLoad) Write(w *Writer) error {
	w.WriteU32(c.Unk)
	w.WriteU32(c.Length)
	w.WriteBytes(c.Data)
	return nil
}

// SaveSlot represents a full character slot.
type SaveSlot struct {
	Header                []byte // 0x20 bytes
	PCHeader              []byte // 0x290 bytes
	GaItems               [0x1400]GaItem
	PlayerGameData        PlayerGameData
	Unk1                  []byte // 0xd0 bytes
	EquipData             []byte // 88 bytes
	ChrAsm                []byte // 116 bytes
	ChrAsm2               []byte // 88 bytes
	EquipInventoryData    EquipInventoryData
	StorageInventoryData  EquipInventoryData
	Skip1                 []byte // magic, item, gesture, projectile, equipped, physics, etc.
	Skip2                 []byte // gesture_game, regions, ride, unks, menu, trophy
	GaItemData            GaItemData
	Padding               []byte // Rest of the slot (0x280000 - current pos)
}

func (s *SaveSlot) Read(r *Reader, platform string) error {
	startPos := r.Pos()

	// 1. Header
	s.Header, _ = r.ReadBytes(0x20)

	// PC has an extra header block
	if platform == "PC" {
		s.PCHeader, _ = r.ReadBytes(0x290)
	}

	// 2. GaItems (5120 items)
	for i := 0; i < 0x1400; i++ {
		s.GaItems[i].Read(r)
	}

	// 3. PlayerGameData
	s.PlayerGameData.Read(r)

	s.Unk1, _ = r.ReadBytes(0xd0)
	s.EquipData, _ = r.ReadBytes(88)
	s.ChrAsm, _ = r.ReadBytes(116)
	s.ChrAsm2, _ = r.ReadBytes(88)

	// 5. Inventory
	s.EquipInventoryData.Read(r, 0xa80, 0x180)

	// 6. Skip to Storage (EquipInventoryData 2)
	s.Skip1, _ = r.ReadBytes(40 + 104 + 24 + 64 + 116 + 12 + 4 + 0x12f)

	// 7. Storage
	s.StorageInventoryData.Read(r, 0x780, 0x80)

	// 8. Skip to GaItemData
	s.Skip2, _ = r.ReadBytes(256 + 16 + 40 + 1 + 0x40 + 12 + 0x1008 + 0x34)

	s.GaItemData.Read(r)

	// 9. Padding
	currentPos := r.Pos()
	remaining := (startPos + 0x280000) - currentPos
	if remaining > 0 {
		s.Padding, _ = r.ReadBytes(int(remaining))
	}

	return nil
}

func (s *SaveSlot) Write(w *Writer, platform string) error {
	w.WriteBytes(s.Header)
	if platform == "PC" {
		w.WriteBytes(s.PCHeader)
	}
	for i := 0; i < 0x1400; i++ {
		s.GaItems[i].Write(w)
	}
	s.PlayerGameData.Write(w)
	w.WriteBytes(s.Unk1)
	w.WriteBytes(s.EquipData)
	w.WriteBytes(s.ChrAsm)
	w.WriteBytes(s.ChrAsm2)
	s.EquipInventoryData.Write(w)
	w.WriteBytes(s.Skip1)
	s.StorageInventoryData.Write(w)
	w.WriteBytes(s.Skip2)
	s.GaItemData.Write(w)
	w.WriteBytes(s.Padding)
	return nil
}
