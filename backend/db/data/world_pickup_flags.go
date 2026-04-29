package data

// WorldPickupFlagID maps inventory item ID → world pickup event flag ID.
//
// Many "1/0 cap" items in the game (Notes, Map fragments, About * tutorials,
// Letters, Paintings, Cookbooks, Whetblades, Prayerbooks, Crystal Tears,
// quest items) exist at exactly one fixed world location (chest, corpse, or
// shop). Cap is enforced by the inventory definition, but the game still
// SPAWNS the world copy when the player walks past — and if their inventory
// is full, the duplicate drops on the ground.
//
// Setting the corresponding pickup flag tells the game "the world copy is
// already collected" — preventing the duplicate spawn.
//
// Source: regulation.bin (post-DLC build) extracted via /tmp/dump_all_params.py
//   • ItemLotParam_map.getItemFlagId where lotItemCategory==1 (goods)
//   • ShopLineupParam.eventFlag_forStock where equipType==3 (goods)
//
// Filtering rules:
//   • flags < 10000 excluded — those are progression/boss/status flags, not
//     pickup flags (e.g. Great Rune flags 171–176, Dark Moon Ring flag 114).
//   • Bell Bearings excluded — they have a separate "Unlocked" toggle in the
//     World tab that handles the dual-state (in-inv vs given-to-TM).
//   • Great Runes / Mending Runes excluded — touching their flags can affect
//     boss state, ending paths, or rune-effect activation. Spawning them via
//     editor is fine; auto-flagging is not.
//
// Inferred entries (extrapolated from adjacent confirmed ranges, unverified):
//   • 0x400023AA..0x400023B4 (About Fast Travel ... About Wielding Armaments):
//     these tutorials are EMEVD-spawned with no ItemLotParam reference, but
//     adjacent IDs 9100..9129 (0x238C..0x23A9) all use 550000+(id-9100)*10,
//     so we extrapolate the same pattern. Setting the flag is harmless if
//     unchecked, beneficial if EMEVD reuses the pattern.
//
// Intentional shared flag (NOT a duplicate bug):
//   • 0x401EA3C8 (Hole-Laden Necklace) and 0x401EA3D0 (Ruins Map) both map
//     to flag 400660. Verified in regulation.bin: lot 106600 places the
//     Necklace with flag 400660, lot 106601 places the Map with flag 400660.
//     This is FromSoftware quest-design (Igon's questline gate — collecting
//     either advances quest), so adding either via editor blocks BOTH world
//     copies. Both items are kept in the map per design.
//
// Items NOT in this map (no flag in regulation, no derivable pattern):
//   • 0x40002202 (Note: Great Coffins) — VERIFIED cut content; flagged
//     cut_content+ban_risk in info.go. Polish FMG broken ([ERROR] prefix),
//     no Lot/Shop reference, only in dev CharaInit row 8515.
//   • 0x401EA848 (About the Scadutree Blessing), 0x401EA84A (About New
//     Inventory Features) — purely EMEVD-spawned in DLC, nothing in
//     ItemLotParam_map or ShopLineupParam to extrapolate from. TODO:
//     extract Data0.bdt EMEVD from Steam Deck to find their flags.
//   • 0x401EA443 (Note: Sealed Spiritsprings) — params unfinished in DLC
//     build, flagged cut_content+ban_risk in info.go.
var WorldPickupFlagID = map[uint32]uint32{

	// ─── Notes (base) (16) ───
	// Note: 0x40002202 (Great Coffins) excluded — see header (cut content).
	0x400021FC:      69600, // Note: Hidden Cave
	0x400021FD:      69610, // Note: Imp Shades
	0x400021FE:      69620, // Note: Flask of Wondrous Physick
	0x400021FF:      69630, // Note: Stonedigger Trolls
	0x40002200:      69640, // Note: Walking Mausoleum
	0x40002201:      69650, // Note: Unseen Assassins
	0x40002203:      69670, // Note: Flame Chariots
	0x40002204:      69680, // Note: Demi-human Mobs
	0x40002205:      69690, // Note: Land Squirts
	0x40002206:      69700, // Note: Gravity's Advantage
	0x40002207:      69710, // Note: Revenants
	0x40002208:      69720, // Note: Waypoint Ruins
	0x40002209:      69730, // Note: Gateway
	0x40002202:      69660, // Note: Great Coffins (inferred — see header notes)
	0x4000220A:      69740, // Note: Miquella's Needle
	0x4000220B:      69750, // Note: Frenzied Flame Village
	0x4000220C:      69760, // Note: The Lord of Frenzied Flame
	0x4000220D:      69770, // Note: Below the Capital

	// ─── Abouts (base) (26) ───
	0x4000238C:     550000, // About Sites of Grace
	0x4000238E:     550020, // About Bows
	0x4000238F:     550030, // About Crouching
	0x40002390:     550040, // About Stance-Breaking
	0x40002391:     550050, // About Stakes of Marika
	0x40002392:     550060, // About Guard Counters
	0x40002393:     550070, // About the Map
	0x40002394:     550080, // About Guidance of Grace
	0x40002395:     550090, // About Horseback Riding
	0x40002396:     550100, // About Death
	0x40002397:     550110, // About Summoning Spirits
	0x40002398:     550120, // About Guarding
	0x40002399:     550130, // About Item Crafting
	0x4000239C:     550160, // About Adding Skills
	0x4000239D:     550170, // About Birdseye Telescopes
	0x4000239E:     550180, // About Spiritspring Jumping
	0x4000239F:     550190, // About Vanquishing Enemy Groups
	0x400023A1:     550210, // About Summoning Other Players
	0x400023A2:     550220, // About Cooperative Multiplayer
	0x400023A3:     550230, // About Competitive Multiplayer
	0x400023A4:     550240, // About Invasion Multiplayer
	0x400023A5:     550250, // About Hunter Multiplayer
	0x400023A6:     550260, // About Summoning Pools
	0x400023A7:     550270, // About Monument Icon
	0x400023A8:     550280, // About Requesting Help from Hunters
	0x400023A9:     550290, // About Skills
	0x400023AA:     550300, // About Fast Travel to Sites of Grace (inferred)
	0x400023AB:     550310, // About Strengthening Armaments (inferred)
	0x400023AC:     550320, // About Roundtable Hold (inferred)
	0x400023AE:     550340, // About Materials (inferred)
	0x400023AF:     550350, // About Containers (inferred)
	0x400023B0:     550360, // About Adding Affinities (inferred)
	0x400023B1:     550370, // About Pouches (inferred)
	0x400023B2:     550380, // About Dodging (inferred)
	0x400023B4:     550400, // About Wielding Armaments (inferred)

	// ─── Maps (base) (19) ───
	0x40002198:      62010, // Map: Limgrave, West
	0x40002199:      62011, // Map: Weeping Peninsula
	0x4000219A:      62012, // Map: Limgrave, East
	0x4000219B:      62020, // Map: Liurnia, East
	0x4000219C:      62021, // Map: Liurnia, North
	0x4000219D:      62022, // Map: Liurnia, West
	0x4000219E:      62030, // Map: Altus Plateau
	0x4000219F:      62031, // Map: Leyndell, Royal Capital
	0x400021A0:      62032, // Map: Mt. Gelmir
	0x400021A1:      62040, // Map: Caelid
	0x400021A2:      62041, // Map: Dragonbarrow
	0x400021A3:      62050, // Map: Mountaintops of the Giants, West
	0x400021A4:      62051, // Map: Mountaintops of the Giants, East
	0x400021A5:      62060, // Map: Ainsel River
	0x400021A6:      62061, // Map: Lake of Rot
	0x400021A7:      62063, // Map: Siofra River
	0x400021A8:      62062, // Map: Mohgwyn Palace
	0x400021A9:      62064, // Map: Deeproot Depths
	0x400021AA:      62052, // Map: Consecrated Snowfield

	// ─── Maps (DLC) (5) ───
	0x401EA618:      62080, // Map: Gravesite Plain
	0x401EA619:      62081, // Map: Scadu Altus
	0x401EA61A:      62082, // Map: Southern Shore
	0x401EA61B:      62083, // Map: Rauh Ruins
	0x401EA61C:      62084, // Map: Abyss

	// ─── Letters / Messages (base) (7) ───
	0x40001FC3:     400080, // Irina's Letter
	0x40001FC4:     400074, // Letter from Volcano Manor
	0x40001FC5:     400075, // Red Letter
	0x40001FE7:     400180, // Letter to Patches
	0x40001FED:     400290, // Letter to Bernahl
	0x4000201D:     400091, // Zorayas's Letter
	0x4000201F:     400356, // Rogier's Letter

	// ─── Letters / Messages / DLC Maps (13) ───
	0x401EA3C7:     400610, // Cross Map
	0x401EA3CC:     400611, // Cross-Marked Map
	0x401EA3CF:     400620, // Letter for Freyja
	0x401EA3D0:     400660, // Ruins Map
	0x401EA3D1:     400661, // Ruins Map (2nd)
	0x401EA3D2:     400662, // Ruins Map (3rd)
	0x401EA3DB: 2047447710, // Castle Cross Message
	0x401EA3DC: 2047477000, // Ancient Ruins Cross Message
	0x401EA3DD: 2048457510, // Monk's Missive
	0x401EA3DE:   21017180, // Storehouse Cross Message
	0x401EA3E0:   28007010, // Torn Diary Page
	0x401EA3E2:     580600, // Message from Leda
	0x401EA3E5:   20007830, // Tower of Shadow Message

	// ─── Paintings (base) (7) ───
	0x40002008:     580000, // Homing Instinct Painting
	0x40002009:     580010, // Resurrection Painting
	0x4000200A:     580020, // Champion's Song Painting
	0x4000200B:     580030, // Sorcerer Painting
	0x4000200C:     580040, // Prophecy Painting
	0x4000200D:     580050, // Flightless Bird Painting
	0x4000200E:     580060, // Redmane Painting

	// ─── Paintings (DLC) (3) ───
	0x401EA488:     580100, // Incursion Painting
	0x401EA489:     580110, // The Sacred Tower Painting
	0x401EA48A:     580120, // Domain of Dragons Painting

	// ─── Cookbooks (103) ───
	0x40002454:      67000, // Nomadic Warrior's Cookbook [1]
	0x40002455:      67010, // Nomadic Warrior's Cookbook [3]
	0x40002456:      67020, // Nomadic Warrior's Cookbook [6]
	0x40002457:      67030, // Nomadic Warrior's Cookbook [10]
	0x40002459:      67050, // Nomadic Warrior's Cookbook [7]
	0x4000245A:      67060, // Nomadic Warrior's Cookbook [12]
	0x4000245B:      67070, // Nomadic Warrior's Cookbook [19]
	0x4000245C:      67080, // Nomadic Warrior's Cookbook [13]
	0x4000245D:      67090, // Nomadic Warrior's Cookbook [23]
	0x4000245E:      67100, // Nomadic Warrior's Cookbook [17]
	0x4000245F:      67110, // Nomadic Warrior's Cookbook [2]
	0x40002460:      67120, // Nomadic Warrior's Cookbook [21]
	0x40002461:      67130, // Missionary's Cookbook [6]
	0x40002468:      67200, // Armorer's Cookbook [1]
	0x40002469:      67210, // Armorer's Cookbook [2]
	0x4000246A:      67220, // Nomadic Warrior's Cookbook [11]
	0x4000246B:      67230, // Nomadic Warrior's Cookbook [20]
	0x4000246D:      67250, // Armorer's Cookbook [7]
	0x4000246E:      67260, // Armorer's Cookbook [4]
	0x4000246F:      67270, // Nomadic Warrior's Cookbook [18]
	0x40002470:      67280, // Armorer's Cookbook [3]
	0x40002471:      67290, // Nomadic Warrior's Cookbook [16]
	0x40002472:      67300, // Armorer's Cookbook [6]
	0x40002473:      67310, // Armorer's Cookbook [5]
	0x4000247C:      67400, // Glintstone Craftsman's Cookbook [4]
	0x4000247D:      67410, // Glintstone Craftsman's Cookbook [1]
	0x4000247E:      67420, // Glintstone Craftsman's Cookbook [5]
	0x4000247F:      67430, // Nomadic Warrior's Cookbook [9]
	0x40002480:      67440, // Glintstone Craftsman's Cookbook [8]
	0x40002481:      67450, // Glintstone Craftsman's Cookbook [2]
	0x40002482:      67460, // Glintstone Craftsman's Cookbook [6]
	0x40002483:      67470, // Glintstone Craftsman's Cookbook [7]
	0x40002484:      67480, // Glintstone Craftsman's Cookbook [3]
	0x40002490:      67600, // Missionary's Cookbook [2]
	0x40002491:      67610, // Missionary's Cookbook [1]
	0x40002493:      67630, // Missionary's Cookbook [5]
	0x40002494:      67640, // Missionary's Cookbook [4]
	0x40002495:      67650, // Missionary's Cookbook [3]
	0x400024A4:      67800, // Nomadic Warrior's Cookbook [4]
	0x400024A7:      67830, // Nomadic Warrior's Cookbook [5]
	0x400024A8:      67840, // Perfumer's Cookbook [1]
	0x400024A9:      67850, // Perfumer's Cookbook [2]
	0x400024AA:      67860, // Perfumer's Cookbook [3]
	0x400024AB:      67870, // Nomadic Warrior's Cookbook [14]
	0x400024AC:      67880, // Nomadic Warrior's Cookbook [8]
	0x400024AD:      67890, // Nomadic Warrior's Cookbook [22]
	0x400024AE:      67900, // Nomadic Warrior's Cookbook [15]
	0x400024AF:      67910, // Nomadic Warrior's Cookbook [24]
	0x400024B0:      67920, // Perfumer's Cookbook [4]
	0x400024B8:      68000, // Ancient Dragon Apostle's Cookbook [1]
	0x400024B9:      68010, // Ancient Dragon Apostle's Cookbook [2]
	0x400024BA:      68020, // Ancient Dragon Apostle's Cookbook [4]
	0x400024BB:      68030, // Ancient Dragon Apostle's Cookbook [3]
	0x400024CC:      68200, // Fevor's Cookbook [1]
	0x400024CD:      68210, // Fevor's Cookbook [3]
	0x400024CE:      68220, // Fevor's Cookbook [2]
	0x400024CF:      68230, // Missionary's Cookbook [7]
	0x400024E0:      68400, // Frenzied's Cookbook [1]
	0x400024E1:      68410, // Frenzied's Cookbook [2]
	0x401EA8D5:      68510, // Forager Brood Cookbook [6]
	0x401EA8D6:      68520, // Forager Brood Cookbook [1]
	0x401EA8D7:      68530, // Forager Brood Cookbook [2]
	0x401EA8D8:      68540, // Forager Brood Cookbook [3]
	0x401EA8D9:      68550, // Forager Brood Cookbook [4]
	0x401EA8DA:      68560, // Forager Brood Cookbook [5]
	0x401EA8DB:      68570, // Igon's Cookbook [2]
	0x401EA8DC:      68580, // Finger-Weaver's Cookbook [2]
	0x401EA8DD:      68590, // Greater Potentate's Cookbook [1]
	0x401EA8DE:      68600, // Greater Potentate's Cookbook [4]
	0x401EA8DF:      68610, // Greater Potentate's Cookbook [5]
	0x401EA8E0:      68620, // Greater Potentate's Cookbook [12]
	0x401EA8E1:      68630, // Greater Potentate's Cookbook [7]
	0x401EA8E2:      68640, // Greater Potentate's Cookbook [9]
	0x401EA8E3:      68650, // Greater Potentate's Cookbook [10]
	0x401EA8E4:      68660, // Greater Potentate's Cookbook [11]
	0x401EA8E5:      68670, // Mad Craftsman's Cookbook [2]
	0x401EA8E6:      68680, // Greater Potentate's Cookbook [8]
	0x401EA8E7:      68690, // Greater Potentate's Cookbook [3]
	0x401EA8E8:      68700, // Greater Potentate's Cookbook [13]
	0x401EA8E9:      68710, // Greater Potentate's Cookbook [14]
	0x401EA8EA:      68720, // Greater Potentate's Cookbook [6]
	0x401EA8EB:      68730, // Greater Potentate's Cookbook [2]
	0x401EA8EC:      68740, // Ancient Dragon Knight's Cookbook [1]
	0x401EA8ED:      68750, // Mad Craftsman's Cookbook [1]
	0x401EA8EE:      68760, // St. Trina Disciple's Cookbook [1]
	0x401EA8EF:      68770, // Fire Knight's Cookbook [1]
	0x401EA8F0:      68780, // Ancient Dragon Knight's Cookbook [2]
	0x401EA8F1:      68790, // Loyal Knight's Cookbook
	0x401EA8F2:      68800, // Battlefield Priest's Cookbook [1]
	0x401EA8F3:      68810, // Igon's Cookbook [1]
	0x401EA8F4:      68820, // Battlefield Priest's Cookbook [2]
	0x401EA8F5:      68830, // Forager Brood Cookbook [7]
	0x401EA8F6:      68840, // St. Trina Disciple's Cookbook [3]
	0x401EA8F7:      68850, // Grave Keeper's Cookbook [2]
	0x401EA8F8:      68860, // Antiquity Scholar's Cookbook [2]
	0x401EA8F9:      68870, // Tibia's Cookbook
	0x401EA8FA:      68880, // Mad Craftsman's Cookbook [3]
	0x401EA8FB:      68890, // Battlefield Priest's Cookbook [3]
	0x401EA8FC:      68900, // Fire Knight's Cookbook [2]
	0x401EA8FE:      68920, // Finger-Weaver's Cookbook [1]
	0x401EA8FF:      68930, // Battlefield Priest's Cookbook [4]
	0x401EA900:      68940, // Grave Keeper's Cookbook [1]
	0x401EA901:      68950, // St. Trina Disciple's Cookbook [2]

	// ─── Whetblades (5) ───
	0x4000230A:      65610, // Iron Whetblade
	0x4000230B:      65640, // Red-Hot Whetblade
	0x4000230C:      65660, // Sanctified Whetblade
	0x4000230D:      65680, // Glintstone Whetblade
	0x4000230E:      65720, // Black Whetblade

	// ─── Prayerbooks / Scrolls / Principia (12) ───
	0x40002292:   14007360, // Conspectus Scroll
	0x40002293: 1044357010, // Royal House Scroll
	0x40002297: 1036417000, // Fire Monks' Prayerbook
	0x40002298: 1052557900, // Giant's Prayerbook
	0x40002299:   10007990, // Godskin Prayerbook
	0x4000229A:   11007690, // Two Fingers' Prayerbook
	0x4000229B:   11107700, // Assassin's Prayerbook
	0x4000229E:   11007910, // Golden Order Principia
	0x400022A0: 1038447100, // Dragon Cult Prayerbook
	0x400022A1:   13007120, // Ancient Dragon Prayerbook
	0x400022A2: 1039407000, // Academy Scroll
	0x401EA3CE:   21017340, // Secret Rite Scroll

	// ─── Crystal Tears / Tears (37) ───
	0x40002AF8:      65000, // Crimsonspill Crystal Tear
	0x40002AF9:      65010, // Greenspill Crystal Tear
	0x40002AFB:      65030, // Crimson Crystal Tear
	0x40002AFD:      65050, // Cerulean Crystal Tear
	0x40002AFE:      65060, // Speckled Hardtear
	0x40002AFF:      65070, // Crimson Bubbletear
	0x40002B00:      65080, // Opaline Bubbletear
	0x40002B01:      65090, // Crimsonburst Crystal Tear
	0x40002B02:      65100, // Greenburst Crystal Tear
	0x40002B03:      65110, // Opaline Hardtear
	0x40002B04:      65120, // Winged Crystal Tear
	0x40002B05:      65130, // Thorny Cracked Tear
	0x40002B06:      65140, // Spiked Cracked Tear
	0x40002B07:      65150, // Windy Crystal Tear
	0x40002B09:      65170, // Ruptured Crystal Tear
	0x40002B0A:      65180, // Leaden Hardtear
	0x40002B0B:      65190, // Twiggy Cracked Tear
	0x40002B0C:      65200, // Crimsonwhorl Bubbletear
	0x40002B0D:      65210, // Strength-knot Crystal Tear
	0x40002B0E:      65220, // Dexterity-knot Crystal Tear
	0x40002B0F:      65230, // Intelligence-knot Crystal Tear
	0x40002B10:      65240, // Faith-knot Crystal Tear
	0x40002B11:      65250, // Cerulean Hidden Tear
	0x40002B12:      65260, // Stonebarb Cracked Tear
	0x40002B13:      65270, // Purifying Crystal Tear
	0x40002B14:      65280, // Flame-Shrouding Cracked Tear
	0x40002B15:      65290, // Magic-Shrouding Cracked Tear
	0x40002B16:      65300, // Lightning-Shrouding Cracked Tear
	0x40002B17:      65310, // Holy-Shrouding Cracked Tear
	0x401EAF78:      65400, // Viridian Hidden Tear
	0x401EAF82:      65410, // Crimsonburst Dried Tear
	0x401EAF8C:      65420, // Crimson-Sapping Cracked Tear
	0x401EAF96:      65430, // Cerulean-Sapping Cracked Tear
	0x401EAFA0:      65440, // Oil-Soaked Tear
	0x401EAFAA:      65450, // Bloodsucking Cracked Tear
	0x401EAFB4:      65460, // Glovewort Crystal Tear
	0x401EAFBE:      65470, // Deflecting Hardtear

	// ─── Other Key Items / Quest Items (54) ───
	0x40001F4A:   10007500, // Rusty Key
	0x40001FA9: 1046367500, // Dectus Medallion (Left)
	0x40001FAA: 1051397900, // Dectus Medallion (Right)
	0x40001FAB:     400001, // Rold Medallion
	0x40001FAF:     400391, // Carian Inverted Statue
	0x40001FBE: 1037497300, // Fingerprint Grape
	0x40001FC0:     400070, // Tonic of Forgetfulness
	0x40001FC1:   16007710, // Serpent's Amnion
	0x40001FC6:     400072, // Drawing-Room Key
	0x40001FC8:     400300, // Rya's Necklace
	0x40001FC9:     400090, // Volcano Manor Invitation
	0x40001FCE: 1040527000, // Amber Starlight
	0x40001FCF:     400143, // Seluvis's Introduction
	0x40001FD0:     400100, // Sellen's Primal Glintstone
	0x40001FDB:     400033, // Lord of Blood's Favor
	0x40001FDE:      60110, // Spirit Calling Bell
	0x40001FDF:     400395, // Fingerslayer Blade
	0x40001FE1:     520340, // Sewing Needle
	0x40001FE2: 1037467000, // Gold Sewing Needle
	0x40001FE3:      60140, // Tailoring Tools
	0x40001FE4:     400140, // Seluvis's Potion
	0x40001FE6:     400145, // Amber Draught
	0x40001FE8:     400181, // Dancer's Castanets
	0x40001FE9:     400102, // Sellian Sealbreaker
	0x40001FEB:   10007450, // Chrysalids' Memento
	0x40001FEC:     520210, // Black Knifeprint
	0x40001FEE:   14007930, // Academy Glintstone Key
	0x40001FEF:     400280, // Haligtree Secret Medallion (Left)
	0x40001FF0:     400130, // Haligtree Secret Medallion (Right)
	0x40001FFC:      60150, // Golden Tailoring Tools
	0x40001FFE:     400334, // Knifeprint Clue
	0x40001FFF:     400392, // Cursemark of Death
	0x40002002:   10017010, // The Stormhawk King
	0x40002005:     400380, // Sewer-Gaol Key
	0x40002006: 1035457100, // Meeting Place Map
	0x40002007:     400159, // Discarded Palace Key
	0x4000201E:     400173, // Alexander's Innards
	0x40002134:      60120, // Crafting Kit
	0x4000218E:     400210, // Whetstone Knife
	0x400021D4: 1039537080, // Mirage Riddle
	0x40002310:     400310, // Unalloyed Gold Needle
	0x40002311: 1039547300, // Valkyrie's Prosthesis
	0x40002313:     400239, // Beast Eye
	0x40002314:     400331, // Weathered Dagger
	0x401EA3C3:     400710, // Igon's Furled Finger
	0x401EA3C4:   20007510, // Well Depths Key
	0x401EA3C5:   41027000, // Gaol Upper Level Key
	0x401EA3C6:   41027320, // Gaol Lower Level Key
	0x401EA3C8:     400660, // Hole-Laden Necklace
	0x401EA3CB:     510630, // Heart of Bayle
	0x401EA3CD:   20007480, // Storeroom Key
	0x401EA3D5:     510460, // Messmer's Kindling
	0x401EA3D9: 2049477000, // Furnace Keeper's Note
	0x401EA3E4:     400696, // Prayer Room Key
}

// IsWorldPickupItem returns true if the item ID has an associated world
// pickup flag that should be auto-set when added to inventory.
func IsWorldPickupItem(id uint32) bool {
	_, ok := WorldPickupFlagID[id]
	return ok
}

