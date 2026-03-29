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

// PlayerGameData represents character stats.
type PlayerGameData struct {
	Health         uint32
	MaxHealth      uint32
	BaseMaxHealth  uint32
	FP             uint32
	MaxFP          uint32
	BaseMaxFP      uint32
	SP             uint32
	MaxSP          uint32
	BaseMaxSP      uint32
	Vigor          uint32
	Mind           uint32
	Endurance      uint32
	Strength       uint32
	Dexterity      uint32
	Intelligence   uint32
	Faith          uint32
	Arcane         uint32
	Level          uint32
	Souls          uint32
	SoulsMemory    uint32
	CharacterName  [16]uint16
}

func (p *PlayerGameData) Read(r *Reader) error {
	r.ReadI32() // _0x4
	r.ReadI32() // _0x4_1
	p.Health, _ = r.ReadU32()
	p.MaxHealth, _ = r.ReadU32()
	p.BaseMaxHealth, _ = r.ReadU32()
	p.FP, _ = r.ReadU32()
	p.MaxFP, _ = r.ReadU32()
	p.BaseMaxFP, _ = r.ReadU32()
	r.ReadI32() // _0x4_2
	p.SP, _ = r.ReadU32()
	p.MaxSP, _ = r.ReadU32()
	p.BaseMaxSP, _ = r.ReadU32()
	r.ReadI32() // _0x4_3
	p.Vigor, _ = r.ReadU32()
	p.Mind, _ = r.ReadU32()
	p.Endurance, _ = r.ReadU32()
	p.Strength, _ = r.ReadU32()
	p.Dexterity, _ = r.ReadU32()
	p.Intelligence, _ = r.ReadU32()
	p.Faith, _ = r.ReadU32()
	p.Arcane, _ = r.ReadU32()
	r.ReadI32() // _0x4_4
	r.ReadI32() // _0x4_5
	r.ReadI32() // _0x4_6
	p.Level, _ = r.ReadU32()
	p.Souls, _ = r.ReadU32()
	p.SoulsMemory, _ = r.ReadU32()
	r.ReadBytes(0x28) // _0x28
	for i := 0; i < 16; i++ {
		p.CharacterName[i], _ = r.ReadU16()
	}
	return nil
}

// ProfileSummary represents character data in the menu.
type ProfileSummary struct {
	CharacterName [17]uint16
	Level         uint32
}

func (p *ProfileSummary) Read(r *Reader) error {
	for i := 0; i < 17; i++ {
		p.CharacterName[i], _ = r.ReadU16()
	}
	p.Level, _ = r.ReadU32()
	r.ReadBytes(20)  // unks
	r.ReadBytes(288) // padding 0x120
	r.ReadBytes(120) // EquipGaitem
	r.ReadBytes(112) // EquipItem
	r.ReadBytes(6)   // unks
	r.ReadI32()      // unk
	return nil
}

// CSMenuSystemSaveLoad represents dynamic menu data.
type CSMenuSystemSaveLoad struct {
	Length uint32
	Data   []byte
}

func (c *CSMenuSystemSaveLoad) Read(r *Reader) error {
	r.ReadU32() // unk
	len, _ := r.ReadU32()
	c.Length = len
	c.Data, _ = r.ReadBytes(int(len))
	return nil
}

// SaveSlot represents a full character slot.
type SaveSlot struct {
	GaItems        [0x1400]GaItem
	PlayerGameData PlayerGameData
}

func (s *SaveSlot) Read(r *Reader) error {
	r.ReadU32()      // ver
	r.ReadBytes(4)   // map_id
	r.ReadBytes(0x18) // _0x18
	for i := 0; i < 0x1400; i++ {
		s.GaItems[i].Read(r)
	}
	return s.PlayerGameData.Read(r)
}
