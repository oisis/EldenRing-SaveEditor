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

// EquipInventoryData represents the inventory and storage lists.
type EquipInventoryData struct {
	CommonItems []EquipInventoryItem
	KeyItems    []EquipInventoryItem
}

func (e *EquipInventoryData) Read(r *Reader, commonLen, keyLen int) error {
	r.ReadU32() // common_inventory_items_distinct_count
	for i := 0; i < commonLen; i++ {
		item := EquipInventoryItem{}
		item.Read(r)
		e.CommonItems = append(e.CommonItems, item)
	}
	r.ReadU32() // key_inventory_items_distinct_count
	for i := 0; i < keyLen; i++ {
		item := EquipInventoryItem{}
		item.Read(r)
		e.KeyItems = append(e.KeyItems, item)
	}
	r.ReadU32() // next_equip_index
	r.ReadU32() // next_acquisition_sort_id
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
	GaItems            [0x1400]GaItem
	PlayerGameData     PlayerGameData
	EquipInventoryData EquipInventoryData
	GaItemData         GaItemData
}

func (s *SaveSlot) Read(r *Reader) error {
	r.ReadU32()      // ver
	r.ReadBytes(4)   // map_id
	r.ReadBytes(0x18) // _0x18
	for i := 0; i < 0x1400; i++ {
		s.GaItems[i].Read(r)
	}
	s.PlayerGameData.Read(r)

	r.ReadBytes(0xd0) // _0xd0
	r.ReadBytes(88)   // EquipData
	r.ReadBytes(112)  // ChrAsm
	r.ReadBytes(88)   // ChrAsm2

	s.EquipInventoryData.Read(r, 0xa80, 0x180)

	r.ReadBytes(40)   // EquipMagicData
	r.ReadBytes(104)  // EquipItemData
	r.ReadBytes(24)   // equip_gesture_data
	r.ReadBytes(64)   // EquipProjectileData
	r.ReadBytes(112)  // EquippedItems
	r.ReadBytes(12)   // EquipPhysicsData
	r.ReadBytes(4)    // _0x4
	r.ReadBytes(0x12f) // _face_data
	
	// storage_inventory_data (EquipInventoryData)
	storage := EquipInventoryData{}
	storage.Read(r, 0x780, 0x80)

	r.ReadBytes(256) // gesture_game_data
	r.ReadBytes(16)  // regions
	r.ReadBytes(40)  // ride_game_data
	r.ReadBytes(1)   // _0x1
	r.ReadBytes(0x40) // _0x40
	r.ReadBytes(12)  // _0x4_1, _0x4_2, _0x4_3
	r.ReadBytes(0x1008) // _menu_profile_save_load
	r.ReadBytes(0x34)   // _trophy_equip_data

	s.GaItemData.Read(r)

	return nil
}
