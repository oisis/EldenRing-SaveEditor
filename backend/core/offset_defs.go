package core

// SlotSize is the fixed size of each save slot in bytes (2,621,440 = 0x280000).
// Source: SPEC.md §3.1 BND4 Container.
const SlotSize = 0x280000

// FallbackMagicBase is the hardcoded base used when MagicPattern is not found.
const FallbackMagicBase = 0x15420 + 432

// Offsets relative to MagicOffset (negative = before the pattern).
// Source: SPEC.md §5.2 PlayerGameData.
const (
	OffLevel               = -335
	OffVigor               = -379
	OffMind                = -375
	OffEndurance           = -371
	OffStrength            = -367
	OffDexterity           = -363
	OffIntelligence        = -359
	OffFaith               = -355
	OffArcane              = -351
	OffSouls               = -331
	OffGender              = -249
	OffClass               = -248
	OffScadutreeBlessing   = -187
	OffShadowRealmBlessing = -186
	OffCharacterName       = -0x11B // 16 x uint16 UTF-16LE

	// MagicOffset must be at least this value; otherwise negative stat offsets
	// would access memory before the start of the slot buffer.
	MinMagicOffset = 400 // abs(OffVigor) + margin
)

// GaItems section.
// Source: SPEC.md §5.3 GaItems.
const (
	GaItemsStart = 0x20 // scan starts here
)

// GaItem record sizes by handle type prefix (upper nibble).
// Source: SPEC.md §5.3 GaItems.
const (
	GaRecordWeapon    = 21
	GaRecordArmor     = 16
	GaRecordAccessory = 8
	GaRecordItem      = 8
	GaRecordAoW       = 8
)

// GaItem handle constants.
const (
	GaHandleEmpty    = 0x00000000
	GaHandleInvalid  = 0xFFFFFFFF
	GaHandleTypeMask = 0xF0000000 // upper nibble = item type
)

// Inventory layout (relative to MagicOffset).
// Source: SPEC.md §5.4 Dynamic Offsets.
const (
	InvStartFromMagic = 505       // MagicOffset + 505 — points to first common item (common_count header at -4)
	CommonItemCount   = 0xA80     // 2688 common item slots
	KeyItemCount      = 0x180     // 384 key item slots
	StorageItemCount  = 2048      // storage box capacity (read limit for ReadStorage)
	StorageCommonCount = 0x780    // 1920 actual common item slots in storage
	StorageKeyCount   = 0x80     // 128 key item slots in storage
	InvRecordLen      = 12        // bytes per inventory record (handle + qty + index)
	InvSafetyMargin   = 0x9000    // max distance from invStart to validate section
	StorageSafetyMarg = 0x6000    // max distance from storageStart to validate section
	StorageHeaderSkip = 4         // skip 4-byte header at StorageBoxOffset
	InvKeyCountHeader = 4         // 4-byte key_count header between common and key items

	// Offsets of trailing counters relative to (StorageBoxOffset + StorageHeaderSkip).
	// Layout: StorageCommonCount×12 + key_count(4) + StorageKeyCount×12 + next_equip_index(4) + next_acq_sort_id(4)
	StorageNextEquipIdxRel = StorageCommonCount*InvRecordLen + InvKeyCountHeader + StorageKeyCount*InvRecordLen
	StorageNextAcqSortRel  = StorageNextEquipIdxRel + 4
)

// Dynamic offset chain constants (relative to InventoryEnd).
// Source: SPEC.md §5.4 Dynamic Offsets.
const (
	DynPlayerData           = 0x1B0
	DynSpEffect             = 0xD0
	DynEquipedItemIndex     = 0x58
	DynActiveEquipedItems   = 0x1C
	DynEquipedItemsID       = 0x58
	DynActiveEquipedItemsGa = 0x58
	DynInventoryHeld        = 0x9011
	DynEquipedSpells        = 0x74
	DynEquipedItems         = 0x8C
	DynEquipedGestures      = 0x18
	DynEquipedArmaments     = 0x9C
	DynEquipePhysics        = 0x0C
	DynFaceData             = 0x12F
	DynStorageBox           = 0x6010
	DynStorageToGestures    = 0x100
	DynHorse                = 0x29
	DynBloodStain           = 0x4C
	DynMenuProfile          = 0x103C
	DynGaItemsOther         = 0x1B588
	DynTutorialData         = 0x40B
	DynIngameTimer          = 0x1A
	DynEventFlags           = 0
)

// Sanity limits for dynamic size reads from untrusted save data.
const (
	MaxProjCount      = 200000 // max acquired_projectiles count (projSkip = count*8+4; observed: 67584 PC, 103168 PS4)
	MaxUnlockedRegCnt = 20000  // max unlocked_regions count (regSkip = count*4+4)
	MaxHandleAttempts = 10000  // max iterations for generateUniqueHandle
)

// GaItemData section (distinct_acquired_items_count + GaItem2 array).
// Source: ER-Save-Editor save_slot.rs, GaItemData struct.
// GaItemData records every weapon/AoW ID ever acquired. The game looks up weapon properties
// (reinforce_type etc.) from this list on load. Missing entry → crash.
const (
	GaItemDataEntryLen = 16   // id(4) + unk(4) + reinforce_type(4) + unk1(4)
	GaItemDataArrayOff = 8    // array starts after distinct_count(4) + unk1(4)
	GaItemDataMaxCount = 7000 // 0x1B58 max entries (matches DynGaItemsOther / GaItemDataEntryLen)
)

// DLC section constants.
// CSDlc is 0x32 (50) bytes located at SlotSize - 0xB2 (before PlayerGameDataHash).
// Byte[0] = pre-order gesture "The Ring"
// Byte[1] = Shadow of the Erdtree entry flag (non-zero = entered DLC; causes infinite loading without DLC)
// Bytes[2] = pre-order gesture "Ring of Miquella"
// Bytes[3-49] = must be 0x00
const (
	DlcSectionSize   = 0x32                          // 50 bytes
	DlcSectionOffset = SlotSize - HashSize - DlcSectionSize // SlotSize - 0xB2
	DlcEntryFlagByte = 1                              // byte index within DLC section for SotE entry flag
)

// InvEquipReservedMax is the highest CSGaItemIns index reserved for equipment slots (0-432).
// Inventory items added via save editor must have Index > InvEquipReservedMax.
// If next_acquisition_sort_id from the save is ≤ InvEquipReservedMax or overlaps an existing
// item's index, the game dereferences the wrong CSGaItemIns entry → EXCEPTION_ACCESS_VIOLATION.
const InvEquipReservedMax = 432

// GaItem entry counts by slot version.
// Source: ER-Save-Editor save_slot.rs, er-save-manager user_data_x.py
const (
	GaItemCountOld     = 5118 // 0x13FE — version ≤ 81
	GaItemCountNew     = 5120 // 0x1400 — version > 81
	GaItemVersionBreak = 81   // version threshold for GaItem count change
)
