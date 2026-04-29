// Maps lower-tier talisman ID → its highest-tier variant ID.
// Used by DatabaseTab when "Talismans: highest only" toggle is on,
// to filter lower-tier variants from the visible list.
//
// Each family has one "highest" entry (a value in the map) and one or more
// "lower" entries (keys mapping to the highest). Highest IDs are NOT keys.
export const TALISMAN_LOWER_TO_HIGHEST: Record<number, number> = {
    // Crimson Amber Medallion → +3 (DLC)
    0x200003E8: 0x20001B58, 0x200003E9: 0x20001B58, 0x200003EA: 0x20001B58,
    // Cerulean Amber Medallion → +3 (DLC)
    0x200003F2: 0x20001B62, 0x200003F3: 0x20001B62, 0x200003F4: 0x20001B62,
    // Viridian Amber Medallion → +3 (DLC)
    0x200003FC: 0x20001B6C, 0x200003FD: 0x20001B6C, 0x200003FE: 0x20001B6C,
    // Erdtree's Favor → +2 (max in game, no DLC +3)
    0x20000410: 0x20000412, 0x20000411: 0x20000412,
    // Stalwart Horn Charm → +2 (DLC)
    0x20000488: 0x20001B80, 0x20000489: 0x20001B80,
    // Immunizing Horn Charm → +2 (DLC)
    0x20000492: 0x20001B8A, 0x20000493: 0x20001B8A,
    // Clarifying Horn Charm → +2 (DLC)
    0x2000049C: 0x20001B94, 0x2000049D: 0x20001B94,
    // Mottled Necklace → +2 (DLC)
    0x200004B0: 0x20001BA8, 0x200004B1: 0x20001BA8,
    // Dragoncrest Shield Talisman → Dragoncrest Greatshield Talisman (named +3, base game)
    0x20000FA0: 0x20000FA3, 0x20000FA1: 0x20000FA3, 0x20000FA2: 0x20000FA3,
    // Spelldrake Talisman → +3 (DLC)
    0x20000FAA: 0x20001BB2, 0x20000FAB: 0x20001BB2, 0x20000FAC: 0x20001BB2,
    // Flamedrake Talisman → +3 (DLC)
    0x20000FB4: 0x20001BBC, 0x20000FB5: 0x20001BBC, 0x20000FB6: 0x20001BBC,
    // Boltdrake Talisman → +3 (DLC)
    0x20000FBE: 0x20001BC6, 0x20000FBF: 0x20001BC6, 0x20000FC0: 0x20001BC6,
    // Haligdrake Talisman → Golden Braid (named +3, DLC)
    0x20000FC8: 0x20001BD0, 0x20000FC9: 0x20001BD0, 0x20000FCA: 0x20001BD0,
    // Pearldrake Talisman → +3 (DLC)
    0x20000FD2: 0x20001BDA, 0x20000FD3: 0x20001BDA, 0x20000FD4: 0x20001BDA,
    // Crimson Seed Talisman → +1 (DLC)
    0x20001388: 0x20001BE4,
    // Cerulean Seed Talisman → +1 (DLC)
    0x20001392: 0x20001BEE,
    // Arsenal Charm → Great-Jar's Arsenal (named upgrade, base game)
    0x20000406: 0x20000408, 0x20000407: 0x20000408,
    // Green Turtle Talisman → Two-Headed Turtle Talisman (DLC)
    0x2000047E: 0x20001B76,
    // Radagon's Scarseal → Radagon's Soreseal
    0x2000041A: 0x2000041B,
    // Marika's Scarseal → Marika's Soreseal
    0x200004C4: 0x200004C5,
};

export function isLowerTierTalisman(id: number): boolean {
    return id in TALISMAN_LOWER_TO_HIGHEST;
}
