package core

// SaveHeader represents the 0x70 byte file header common to both PC and PS4.
type SaveHeader struct {
	Signature [4]byte // "BND4"
	Unk04     uint8
	Unk05     uint8
	Padding06 [2]byte
	Unk08     uint8
	BigEndian uint8
	BitBigEnd uint8
	Padding0B uint8
	FileCount int32
	HeaderEnd int64 // 0x40
	Version   [8]byte
	EntrySize int64
	Unused00  int64
	Unicode   uint8
	RawFormat uint8
	Extended  uint8
	Padding33 uint8
	Unused01  int32
	HashOff   int64
	Padding40 [0x30]byte // Remainder to 0x70
}

// PlayerGameData represents the character stats and identity.
// Located at offset 0x15420 from the start of a SaveSlot.
type PlayerGameData struct {
	Unk00          int32
	Unk04          int32
	Health         uint32
	MaxHealth      uint32
	BaseMaxHealth  uint32
	FP             uint32
	MaxFP          uint32
	BaseMaxFP      uint32
	Unk20          int32
	SP             uint32
	MaxSP          uint32
	BaseMaxSP      uint32
	Unk30          int32
	Vigor          uint32
	Mind           uint32
	Endurance      uint32
	Strength       uint32
	Dexterity      uint32
	Intelligence   uint32
	Faith          uint32
	Arcane         uint32
	Unk54          int32
	Unk58          int32
	Unk5C          int32
	Level          uint32
	Souls          uint32
	SoulsMemory    uint32
	Padding28      [0x28]byte
	CharacterName  [16]uint16 // UTF-16
	Padding02      [2]byte
	Gender         uint8
	ArcheType      uint8
	Padding03      [3]byte
	Gift           uint8
	Padding1E      [0x1E]byte
	MatchmakingLvl uint8
	Padding35      [0x35]byte
	Password       [0x12]byte
	GroupPass1     [0x12]byte
	GroupPass2     [0x12]byte
	GroupPass3     [0x12]byte
	GroupPass4     [0x12]byte
	GroupPass5     [0x12]byte
	UnkRemainder   [0x34]byte
}

// GaItem represents an inventory item. Its size is dynamic.
type GaItem struct {
	Handle   uint32
	ItemID   uint32
	Unk2     int32  // Only for Weapons/Armor
	Unk3     int32  // Only for Weapons/Armor
	AoW      uint32 // Only for Weapons
	Unk5     uint8  // Only for Weapons
}

// EquipData represents the equipped item indices.
type EquipData struct {
	LeftHandArmaments  [3]uint32
	RightHandArmaments [3]uint32
	Arrows             [2]uint32
	Bolts              [2]uint32
	Unk04              uint32
	Unk04_1            uint32
	Head               uint32
	Chest              uint32
	Arms               uint32
	Legs               uint32
	Unk04_2            uint32
	Talismans          [4]uint32
	Unk                uint32
}

// ChrAsm represents the character assembly (actual Item IDs).
type ChrAsm struct {
	ArmStyle            uint32
	LeftHandActiveSlot  uint32
	RightHandActiveSlot uint32
	LeftArrowActiveSlot uint32
	RightArrowActiveSlot uint32
	LeftBoltActiveSlot  uint32
	RightBoltActiveSlot uint32
	LeftHandArmaments   [3]uint32
	RightHandArmaments  [3]uint32
	Arrows              [2]uint32
	Bolts               [2]uint32
	Unk04               uint32
	Unk04_1             uint32
	Head                uint32
	Chest               uint32
	Arms                uint32
	Legs                uint32
	Unk04_2             uint32
	Talismans           [4]uint32
	Unk                 uint32
}

// UserData10 represents the account metadata and profile summaries.
type UserData10 struct {
	Unk3B4      int32
	SteamID     uint64
	Padding4FC  [0x140]byte
	// Profile summaries and active slots are handled manually in the parser
}

// ProfileSummary represents the character data shown in the "Load Game" menu.
type ProfileSummary struct {
	CharacterName [17]uint16
	Level         uint32
	Unk28         uint32
	Unk2C         uint32
	Unk30         uint32
	Unk34         uint32
	Unk38_150     uint32
	Unk38_8       [0x120]byte
	// Equipment data follows in the binary stream
}

// UserData11 represents the regulation and other data.
type UserData11 struct {
	Regulation []byte // 0x1c5f70 bytes
	Rest       []byte // 0x7A090 bytes
}
