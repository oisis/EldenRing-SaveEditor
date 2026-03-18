from construct import Struct, Bytes, Int32ul, Int32sl, Array, PaddedString, Const, Padding, Int16ul, Float32l, Int64ul, Byte

# Character Stats (PlayerGameData in Rust)
PLAYER_GAME_DATA = Struct(
    "unk0" / Bytes(8),
    "health" / Int32ul,
    "max_health" / Int32ul,
    "base_max_health" / Int32ul,
    "fp" / Int32ul,
    "max_fp" / Int32ul,
    "base_max_fp" / Int32ul,
    "unk1" / Int32sl,
    "sp" / Int32ul,
    "max_sp" / Int32ul,
    "base_max_sp" / Int32ul,
    "unk2" / Int32sl,
    "vigor" / Int32ul,
    "mind" / Int32ul,
    "endurance" / Int32ul,
    "strength" / Int32ul,
    "dexterity" / Int32ul,
    "intelligence" / Int32ul,
    "faith" / Int32ul,
    "arcane" / Int32ul,
    "unk3" / Bytes(12),
    "level" / Int32ul,
    "souls" / Int32ul,
    "soulsmemory" / Int32ul,
    "padding1" / Bytes(0x28),
    "name" / PaddedString(32, "utf-16"),
    "padding2" / Bytes(2),
    "gender" / Byte,
    "archetype" / Byte,
    "padding3" / Bytes(3),
    "gift" / Byte,
    "padding4" / Bytes(0x1e),
    "weapon_level" / Byte,
    "padding5" / Bytes(0x35),
    "password" / Bytes(0x12),
    "group_passwords" / Array(5, Bytes(0x12)),
    "unk_end" / Bytes(0x34)
)

# Individual Item in Inventory
GA_ITEM = Struct(
    "handle" / Int32ul,
    "id" / Int32ul,
    "data" / Struct(
        "unk2" / Int32sl,
        "unk3" / Int32sl,
        "aow_handle" / Int32ul,
        "unk5" / Byte
    )
)

# Full Save Slot Structure (0x280000 bytes)
SAVE_SLOT = Struct(
    "ver" / Int32ul,
    "map_id" / Bytes(4),
    "unk0" / Bytes(0x18),
    "ga_items" / Array(0x1400, GA_ITEM),
    "player_game_data" / PLAYER_GAME_DATA,
    "padding_after_stats" / Bytes(0xd0),
    # ... more structures will be added here for full parity ...
    "data_rest" / Bytes(0x280000 - (4 + 4 + 0x18 + 0x1400 * 17 + 0x138 + 0xd0)) 
)

# BND4 Header (PC only)
BND4_HEADER = Struct(
    "signature" / Const(b"BND4"),
    "unknown1" / Bytes(8),
    "file_count" / Int32ul,
    "header_size" / Int32ul,
    "unknown2" / Bytes(8),
    "file_headers" / Array(lambda ctx: ctx.file_count, Struct(
        "unknown" / Bytes(12),
        "offset" / Int32ul,
        "size" / Int32ul,
        "unknown2" / Bytes(8),
        "name_offset" / Int32ul,
    ))
)
