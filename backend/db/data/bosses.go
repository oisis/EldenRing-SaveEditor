package data

// BossData holds the static definition of a boss encounter.
type BossData struct {
	Name        string
	Region      string
	Type        string // "main" or "field"
	Remembrance bool   // Drops a Remembrance item
}

// Bosses maps synchronized defeat event flag IDs (9xxx range) to boss definitions.
// These global flags are set by the game when a boss is defeated.
// Flag IDs use standard event flag formula: byte = id/8, bit = 7-(id%8).
// Source: soulsmods/elden-ring-eventparam, er-save-manager/boss_data.py, SoulSplitter Boss.cs
//
// ~97 open-world field bosses (Night's Cavalry, Deathbirds, Dragons, Evergaol, etc.)
// only have per-map flags (10000000+) which exceed the event flags bitfield size
// and are NOT included here.
var Bosses = map[uint32]BossData{
	// ============================================================
	// BASE GAME — Legacy Dungeons & Main Story (9100-9135)
	// ============================================================

	// Stormveil Castle
	9100: {Name: "Margit, the Fell Omen", Region: "Stormveil Castle", Type: "main"},
	9101: {Name: "Godrick the Grafted", Region: "Stormveil Castle", Type: "main", Remembrance: true},

	// Chapel of Anticipation
	9103: {Name: "Grafted Scion", Region: "Chapel of Anticipation", Type: "field"},

	// Leyndell, Royal Capital
	9104: {Name: "Morgott, the Omen King", Region: "Leyndell, Royal Capital", Type: "main", Remembrance: true},
	9105: {Name: "Godfrey, First Elden Lord (Golden Shade)", Region: "Leyndell, Royal Capital", Type: "main"},

	// Leyndell, Ashen Capital
	9106: {Name: "Sir Gideon Ofnir, the All-Knowing", Region: "Leyndell, Ashen Capital", Type: "main"},
	9107: {Name: "Hoarah Loux, Warrior", Region: "Leyndell, Ashen Capital", Type: "main", Remembrance: true},

	// Grand Cloister
	9108: {Name: "Astel, Naturalborn of the Void", Region: "Grand Cloister", Type: "main", Remembrance: true},

	// Ainsel River
	9109: {Name: "Dragonkin Soldier of Nokstella", Region: "Ainsel River", Type: "field"},

	// Siofra Aqueduct
	9110: {Name: "Valiant Gargoyles", Region: "Siofra Aqueduct", Type: "field"},

	// Deeproot Depths
	9111: {Name: "Lichdragon Fortissax", Region: "Deeproot Depths", Type: "main", Remembrance: true},

	// Mohgwyn Palace
	9112: {Name: "Mohg, Lord of Blood", Region: "Mohgwyn Palace", Type: "main", Remembrance: true},

	// Crumbling Farum Azula
	9114: {Name: "Godskin Duo", Region: "Crumbling Farum Azula", Type: "main"},
	9115: {Name: "Dragonlord Placidusax", Region: "Crumbling Farum Azula", Type: "main", Remembrance: true},
	9116: {Name: "Maliketh, the Black Blade", Region: "Crumbling Farum Azula", Type: "main", Remembrance: true},

	// Academy of Raya Lucaria
	9117: {Name: "Red Wolf of Radagon", Region: "Academy of Raya Lucaria", Type: "main"},
	9118: {Name: "Rennala, Queen of the Full Moon", Region: "Academy of Raya Lucaria", Type: "main", Remembrance: true},

	// Miquella's Haligtree
	9119: {Name: "Loretta, Knight of the Haligtree", Region: "Miquella's Haligtree", Type: "main"},
	9120: {Name: "Malenia, Blade of Miquella", Region: "Miquella's Haligtree", Type: "main", Remembrance: true},

	// Volcano Manor
	9121: {Name: "Godskin Noble", Region: "Volcano Manor", Type: "main"},
	9122: {Name: "Rykard, Lord of Blasphemy", Region: "Volcano Manor", Type: "main", Remembrance: true},

	// Elden Throne
	9123: {Name: "Elden Beast", Region: "Elden Throne", Type: "main", Remembrance: true},

	// Subterranean Shunning-Grounds
	9125: {Name: "Mohg, the Omen", Region: "Subterranean Shunning-Grounds", Type: "field"},

	// Ruin-Strewn Precipice
	9126: {Name: "Magma Wyrm Makar", Region: "Ruin-Strewn Precipice", Type: "field"},

	// Fringefolk Hero's Grave
	9128: {Name: "Ulcerated Tree Spirit", Region: "Fringefolk Hero's Grave", Type: "field"},

	// Volcano Manor (Abductor)
	9129: {Name: "Abductor Virgins", Region: "Volcano Manor", Type: "field"},

	// Redmane Castle
	9130: {Name: "Starscourge Radahn", Region: "Redmane Castle", Type: "main", Remembrance: true},

	// Mountaintops of the Giants
	9131: {Name: "Fire Giant", Region: "Mountaintops of the Giants", Type: "main", Remembrance: true},

	// Siofra River
	9132: {Name: "Ancestor Spirit", Region: "Siofra River", Type: "field"},

	// Ancestral Woods / Nokron
	9133: {Name: "Regal Ancestor Spirit", Region: "Ancestral Woods", Type: "main", Remembrance: true},

	// Nokron, Eternal City
	9134: {Name: "Mimic Tear", Region: "Nokron, Eternal City", Type: "field"},

	// Deeproot Depths
	9135: {Name: "Fia's Champions", Region: "Deeproot Depths", Type: "field"},

	// ============================================================
	// BASE GAME — Divine Towers (9170-9175)
	// ============================================================
	9173: {Name: "Godskin Apostle", Region: "Divine Tower of Caelid", Type: "field"},
	9174: {Name: "Fell Twins", Region: "Divine Tower of East Altus", Type: "field"},

	// ============================================================
	// BASE GAME — Fortress Bosses (9180-9184)
	// ============================================================
	9180: {Name: "Leonine Misbegotten", Region: "Castle Morne", Type: "field"},
	9181: {Name: "Royal Knight Loretta", Region: "Carian Manor", Type: "field"},
	9182: {Name: "Elemer of the Briar", Region: "The Shaded Castle", Type: "field"},
	9183: {Name: "Crucible Knight & Misbegotten Warrior", Region: "Redmane Castle", Type: "field"},
	9184: {Name: "Commander Niall", Region: "Castle Sol", Type: "field"},

	// ============================================================
	// BASE GAME — Catacomb Bosses (9200-9222)
	// ============================================================
	9200: {Name: "Cemetery Shade", Region: "Tombsward Catacombs", Type: "field"},
	9201: {Name: "Erdtree Burial Watchdog", Region: "Impaler's Catacombs", Type: "field"},
	9202: {Name: "Erdtree Burial Watchdog", Region: "Stormfoot Catacombs", Type: "field"},
	9203: {Name: "Black Knife Assassin", Region: "Deathtouched Catacombs", Type: "field"},
	9204: {Name: "Grave Warden Duelist", Region: "Murkwater Catacombs", Type: "field"},
	9205: {Name: "Cemetery Shade", Region: "Black Knife Catacombs", Type: "field"},
	9206: {Name: "Spirit-Caller Snail", Region: "Road's End Catacombs", Type: "field"},
	9207: {Name: "Erdtree Burial Watchdog", Region: "Cliffbottom Catacombs", Type: "field"},
	9208: {Name: "Ancient Hero of Zamor", Region: "Sainted Hero's Grave", Type: "field"},
	9209: {Name: "Red Wolf of the Champion", Region: "Gelmir Hero's Grave", Type: "field"},
	9210: {Name: "Crucible Knight & Crucible Knight Ordovis", Region: "Auriza Hero's Grave", Type: "field"},
	9211: {Name: "Perfumer Tricia & Misbegotten Warrior", Region: "Unsightly Catacombs", Type: "field"},
	9212: {Name: "Erdtree Burial Watchdog", Region: "Wyndham Catacombs", Type: "field"},
	9213: {Name: "Grave Warden Duelist", Region: "Auriza Side Tomb", Type: "field"},
	9214: {Name: "Erdtree Burial Watchdogs", Region: "Minor Erdtree Catacombs", Type: "field"},
	9215: {Name: "Cemetery Shade", Region: "Caelid Catacombs", Type: "field"},
	9216: {Name: "Putrid Tree Spirit", Region: "War-Dead Catacombs", Type: "field"},
	9217: {Name: "Ancient Hero of Zamor", Region: "Giant-Conquering Hero's Grave", Type: "field"},
	9218: {Name: "Ulcerated Tree Spirit", Region: "Giants' Mountaintop Catacombs", Type: "field"},
	9219: {Name: "Putrid Grave Warden Duelist", Region: "Consecrated Snowfield Catacombs", Type: "field"},
	9220: {Name: "Stray Mimic Tear", Region: "Hidden Path to the Haligtree", Type: "field"},
	9221: {Name: "Black Knife Assassin", Region: "Black Knife Catacombs", Type: "field"},
	9222: {Name: "Esgar, Priest of Blood", Region: "Leyndell Catacombs", Type: "field"},

	// ============================================================
	// BASE GAME — Cave Bosses (9230-9249)
	// ============================================================
	9230: {Name: "Miranda the Blighted Bloom", Region: "Tombsward Cave", Type: "field"},
	9231: {Name: "Runebear", Region: "Earthbore Cave", Type: "field"},
	9233: {Name: "Beastman of Farum Azula", Region: "Groveside Cave", Type: "field"},
	9234: {Name: "Demi-Human Chiefs", Region: "Coastal Cave", Type: "field"},
	9235: {Name: "Guardian Golem", Region: "Highroad Cave", Type: "field"},
	9236: {Name: "Cleanrot Knight", Region: "Stillwater Cave", Type: "field"},
	9237: {Name: "Bloodhound Knight", Region: "Lakeside Crystal Cave", Type: "field"},
	9238: {Name: "Crystalian Duo", Region: "Academy Crystal Cave", Type: "field"},
	9239: {Name: "Kindred of Rot Duo", Region: "Seethewater Cave", Type: "field"},
	9240: {Name: "Demi-Human Queen Margot", Region: "Volcano Cave", Type: "field"},
	9241: {Name: "Omenkiller & Miranda, the Blighted Bloom", Region: "Perfumer's Grotto", Type: "field"},
	9242: {Name: "Black Knife Assassin", Region: "Sage's Cave", Type: "field"},
	9243: {Name: "Frenzied Duelist", Region: "Gaol Cave", Type: "field"},
	9244: {Name: "Beastmen of Farum Azula", Region: "Dragonbarrow Cave", Type: "field"},
	9245: {Name: "Cleanrot Knight Duo", Region: "Abandoned Cave", Type: "field"},
	9246: {Name: "Putrid Crystalian Trio", Region: "Sellia Hideaway", Type: "field"},
	9247: {Name: "Misbegotten Crusader", Region: "Cave of the Forlorn", Type: "field"},
	9248: {Name: "Godskin Apostle & Noble", Region: "Spiritcaller's Cave", Type: "field"},
	9249: {Name: "Necromancer Garris", Region: "Sage's Cave", Type: "field"},

	// ============================================================
	// BASE GAME — Tunnel Bosses (9260-9268)
	// ============================================================
	9260: {Name: "Scaly Misbegotten", Region: "Morne Tunnel", Type: "field"},
	9261: {Name: "Stonedigger Troll", Region: "Limgrave Tunnels", Type: "field"},
	9262: {Name: "Crystalian", Region: "Raya Lucaria Crystal Tunnel", Type: "field"},
	9263: {Name: "Stonedigger Troll", Region: "Old Altus Tunnel", Type: "field"},
	9264: {Name: "Onyx Lord", Region: "Sealed Tunnel", Type: "field"},
	9265: {Name: "Crystalian Duo", Region: "Altus Tunnel", Type: "field"},
	9266: {Name: "Magma Wyrm", Region: "Gael Tunnel", Type: "field"},
	9267: {Name: "Fallingstar Beast", Region: "Sellia Crystal Tunnel", Type: "field"},
	9268: {Name: "Astel, Stars of Darkness", Region: "Yelough Anix Tunnel", Type: "field"},

	// ============================================================
	// SHADOW OF THE ERDTREE — Main / Remembrance Bosses
	// ============================================================
	9140: {Name: "Divine Beast Dancing Lion", Region: "Belurat, Tower Settlement", Type: "main", Remembrance: true},
	9143: {Name: "Promised Consort Radahn", Region: "Enir-Ilim", Type: "main", Remembrance: true},
	9144: {Name: "Golden Hippopotamus", Region: "Shadow Keep", Type: "main"},
	9146: {Name: "Messmer the Impaler", Region: "Shadow Keep", Type: "main", Remembrance: true},
	9148: {Name: "Putrescent Knight", Region: "Stone Coffin Fissure", Type: "main", Remembrance: true},
	9155: {Name: "Metyr, Mother of Fingers", Region: "Cathedral of Manus Metyr", Type: "main", Remembrance: true},
	9156: {Name: "Midra, Lord of Frenzied Flame", Region: "Midra's Manse", Type: "main", Remembrance: true},
	9160: {Name: "Romina, Saint of the Bud", Region: "Church of the Bud", Type: "main", Remembrance: true},
	9161: {Name: "Jori, Elder Inquisitor", Region: "Abyssal Woods", Type: "main"},
	9162: {Name: "Scadutree Avatar", Region: "Scadutree Base", Type: "main", Remembrance: true},
	9163: {Name: "Bayle the Dread", Region: "Jagged Peak", Type: "main", Remembrance: true},
	9164: {Name: "Commander Gaius", Region: "Scaduview", Type: "main", Remembrance: true},
	9190: {Name: "Rellana, Twin Moon Knight", Region: "Castle Ensis", Type: "main", Remembrance: true},

	// ============================================================
	// SHADOW OF THE ERDTREE — Dungeon Bosses (9270-9281)
	// ============================================================
	9270: {Name: "Death Knight", Region: "Fogrift Catacombs", Type: "field"},
	9271: {Name: "Death Knight", Region: "Scorpion River Catacombs", Type: "field"},
	9275: {Name: "Demi-Human Swordmaster Onze", Region: "Belurat Gaol", Type: "field"},
	9276: {Name: "Curseblade Labirith", Region: "Bonny Gaol", Type: "field"},
	9277: {Name: "Lamenter", Region: "Lamenter's Gaol", Type: "field"},
	9280: {Name: "Chief Bloodfiend", Region: "Rivermouth Cave", Type: "field"},
	9281: {Name: "Ancient Dragon-Man", Region: "Dragon's Pit", Type: "field"},
}
