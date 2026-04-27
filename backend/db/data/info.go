package data

// Information holds items that appear on the in-game "Information" tab
// (Polish: "Informacje"). The split was created after the user verified in-game
// that several letters and maps that er-save-manager classifies in
// `KeyItems.txt` / `Tools.txt` actually live in the Information tab.
//
// Source of truth: Fextralife "Info Items" master list cross-checked against
// per-item Fextralife pages and in-game verification by the user.
// See spec/33-info-tab-category.md for the audit trail.
var Information = map[uint32]ItemData{
	// ─── About * tutorial messages (base) ───────────────────────────────
	0x4000238C: {Name: "About Sites of Grace", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_sites_of_grace.png"},
	0x4000238E: {Name: "About Bows", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_bows.png"},
	0x4000238F: {Name: "About Crouching", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_crouching.png"},
	0x40002390: {Name: "About Stance-Breaking", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_stance_breaking.png"},
	0x40002391: {Name: "About Stakes of Marika", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_stakes_of_marika.png"},
	0x40002392: {Name: "About Guard Counters", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_guard_counters.png"},
	0x40002393: {Name: "About the Map", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/about_the_map.png"},
	0x40002394: {Name: "About Guidance of Grace", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_guidance_of_grace.png"},
	0x40002395: {Name: "About Horseback Riding", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_horseback_riding.png"},
	0x40002396: {Name: "About Death", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_death.png"},
	0x40002397: {Name: "About Summoning Spirits", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_summoning_spirits.png"},
	0x40002398: {Name: "About Guarding", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_guarding.png"},
	0x40002399: {Name: "About Item Crafting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_item_crafting.png"},
	0x4000239C: {Name: "About Adding Skills", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_adding_skills.png"},
	0x4000239D: {Name: "About Birdseye Telescopes", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_birdseye_telescopes.png"},
	0x4000239E: {Name: "About Spiritspring Jumping", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_spiritspring_jumping.png"},
	0x4000239F: {Name: "About Vanquishing Enemy Groups", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_vanquishing_enemy_groups.png"},
	0x400023A1: {Name: "About Summoning Other Players", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_summoning_other_players.png"},
	0x400023A2: {Name: "About Cooperative Multiplayer", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_cooperative_multiplayer.png"},
	0x400023A3: {Name: "About Competitive Multiplayer", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_competitive_multiplayer.png"},
	0x400023A4: {Name: "About Invasion Multiplayer", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_invasion_multiplayer.png"},
	0x400023A5: {Name: "About Hunter Multiplayer", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_hunter_multiplayer.png"},
	0x400023A6: {Name: "About Summoning Pools", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_summoning_pools.png"},
	// 0x400023A7: removed in patch 1.06 — was reachable on disc v1.0.
	// Spawning it now triggers EAC soft-ban (ban_risk). Not "cut content"
	// (it shipped legitimately), so cut_content flag intentionally dropped.
	0x400023A7: {Name: "About Monument Icon", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_monument_icon.png", Flags: []string{"ban_risk"}},
	0x400023A8: {Name: "About Requesting Help from Hunters", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_requesting_help_from_hunters.png"},
	0x400023A9: {Name: "About Skills", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_skills.png"},
	0x400023AA: {Name: "About Fast Travel to Sites of Grace", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_fast_travel_to_sites_of_grace.png"},
	0x400023AB: {Name: "About Strengthening Armaments", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_strengthening_armaments.png"},
	0x400023AC: {Name: "About Roundtable Hold", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_roundtable_hold.png"},
	0x400023AE: {Name: "About Materials", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_materials.png"},
	0x400023AF: {Name: "About Containers", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_containers.png"},
	0x400023B0: {Name: "About Adding Affinities", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_adding_affinities.png"},
	0x400023B1: {Name: "About Pouches", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_pouches.png"},
	0x400023B2: {Name: "About Dodging", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_dodging.png"},
	0x400023B4: {Name: "About Wielding Armaments", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_wielding_armaments.png"},
	// 0x400023EB: cut content (never shipped). Spawned copies carry [ERROR]
	// prefix at runtime — Fextralife "About Multiplayer" page + Unobtainable Items list.
	0x400023EB: {Name: "About Multiplayer", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_multiplayer.png", Flags: []string{"cut_content", "ban_risk"}},

	// ─── About * tutorial messages (DLC) ────────────────────────────────
	0x401EA848: {Name: "About the Scadutree Blessing", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_the_scadutree_blessing.png", Flags: []string{"dlc"}},
	0x401EA84A: {Name: "About New Inventory Features", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/about_new_inventory_features.png", Flags: []string{"dlc"}},

	// ─── Paintings ──────────────────────────────────────────────────────
	0x40002008: {Name: "Homing Instinct Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/homing_instinct_painting.png"},
	0x40002009: {Name: "Resurrection Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/resurrection_painting.png"},
	0x4000200A: {Name: "Champion's Song Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/champions_song_painting.png"},
	0x4000200B: {Name: "Sorcerer Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/sorcerer_painting.png"},
	0x4000200C: {Name: "Prophecy Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/prophecy_painting.png"},
	0x4000200D: {Name: "Flightless Bird Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/flightless_bird_painting.png"},
	0x4000200E: {Name: "Redmane Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/redmane_painting.png"},
	0x401EA488: {Name: "Incursion Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/incursion_painting.png", Flags: []string{"dlc"}},
	0x401EA489: {Name: "The Sacred Tower Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/the_sacred_tower_painting.png", Flags: []string{"dlc"}},
	0x401EA48A: {Name: "Domain of Dragons Painting", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/domain_of_dragons_painting.png", Flags: []string{"dlc"}},

	// ─── Letters (base) ─────────────────────────────────────────────────
	// User verified all of these appear in the Information tab in-game,
	// even though er-save-manager classifies them under KeyItems.txt.
	0x40001FC3: {Name: "Irina's Letter", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/irinas_letter.png"},
	0x40001FC4: {Name: "Letter from Volcano Manor", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/letter_from_volcano_manor.png"},
	0x40001FC5: {Name: "Red Letter", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/red_letter.png"},
	0x40001FE7: {Name: "Letter to Patches", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/letter_to_patches.png"},
	0x40001FED: {Name: "Letter to Bernahl", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/letter_to_bernahl.png"},
	// 0x40001FF5: Burial Crow's Letter — Fextralife per-item page calls it
	// cut content, but the user verified it does appear in the Information
	// tab in-game on a save that received it. Keep cut+ban flags.
	0x40001FF5: {Name: "Burial Crow's Letter", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/burial_crows_letter.png", Flags: []string{"cut_content", "ban_risk"}},
	0x4000201D: {Name: "Zorayas's Letter", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/zorayass_letter.png"},
	0x4000201F: {Name: "Rogier's Letter", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/rogiers_letter.png"},

	// ─── Letters / messages (DLC) ──────────────────────────────────────
	// 0x401EA3CF: Letter for Freyja — Fextralife master list says Information,
	// per-item page says Key Item. User has not verified in-game yet.
	// Tagged Information per master list; revisit if user finds it elsewhere.
	0x401EA3C7: {Name: "Cross Map", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/cross_map.png", Flags: []string{"dlc"}},
	0x401EA3CF: {Name: "Letter for Freyja", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/letter_for_freyja.png", Flags: []string{"dlc"}},
	0x401EA3D0: {Name: "Ruins Map", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/ruins_map.png", Flags: []string{"dlc"}},
	0x401EA3D1: {Name: "Ruins Map (2nd)", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/ruins_map_2nd.png", Flags: []string{"dlc"}},
	0x401EA3D2: {Name: "Ruins Map (3rd)", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/ruins_map_3rd.png", Flags: []string{"dlc"}},
	0x401EA3DB: {Name: "Castle Cross Message", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/castle_cross_message.png", Flags: []string{"dlc"}},
	0x401EA3DC: {Name: "Ancient Ruins Cross Message", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/ancient_ruins_cross_message.png", Flags: []string{"dlc"}},
	0x401EA3DD: {Name: "Monk's Missive", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/monks_missive.png", Flags: []string{"dlc"}},
	0x401EA3DE: {Name: "Storehouse Cross Message", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/storehouse_cross_message.png", Flags: []string{"dlc"}},
	0x401EA3E0: {Name: "Torn Diary Page", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/torn_diary_page.png", Flags: []string{"dlc"}},
	0x401EA3E2: {Name: "Message from Leda", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/message_from_leda.png", Flags: []string{"dlc"}},
	0x401EA3E5: {Name: "Tower of Shadow Message", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/tower_of_shadow_message.png", Flags: []string{"dlc"}},

	// ─── Notes (base) ───────────────────────────────────────────────────
	0x4000220A: {Name: "Note: Miquella's Needle", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_miquellas_needle.png"},
	0x4000220C: {Name: "Note: The Lord of Frenzied Flame", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_the_lord_of_frenzied_flame.png"},
	0x4000222E: {Name: "Note: Hidden Cave", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_hidden_cave.png"},
	0x4000222F: {Name: "Note: Imp Shades", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_imp_shades.png"},
	0x40002230: {Name: "Note: Flask of Wondrous Physick", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/sacred_flasks/note_flask_of_wondrous_physick.png"},
	0x40002231: {Name: "Note: Stonedigger Trolls", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/throwables/note_stonedigger_trolls.png"},
	0x40002232: {Name: "Note: Walking Mausoleum", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_walking_mausoleum.png"},
	0x40002233: {Name: "Note: Unseen Assassins", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_unseen_assassins.png"},
	0x40002234: {Name: "Note: Great Coffins", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_great_coffins.png"},
	0x40002235: {Name: "Note: Flame Chariots", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_flame_chariots.png"},
	0x40002236: {Name: "Note: Demi-human Mobs", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_demi_human_mobs.png"},
	0x40002237: {Name: "Note: Land Squirts", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_land_squirts.png"},
	0x40002238: {Name: "Note: Gravity's Advantage", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_gravitys_advantage.png"},
	0x40002239: {Name: "Note: Revenants", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_revenants.png"},
	0x4000223A: {Name: "Note: Waypoint Ruins", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_waypoint_ruins.png"},
	0x4000223B: {Name: "Note: Gateway", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_gateway.png"},
	0x4000223D: {Name: "Note: Frenzied Flame Village", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_frenzied_flame_village.png"},

	// ─── Notes (DLC) ────────────────────────────────────────────────────
	0x401EA3D9: {Name: "Furnace Keeper's Note", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/furnace_keepers_note.png", Flags: []string{"dlc"}},
	0x401EA443: {Name: "Note: Sealed Spiritsprings", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/note_sealed_spiritsprings.png", Flags: []string{"dlc"}},

	// ─── Region maps (base) ─────────────────────────────────────────────
	0x40002198: {Name: "Map: Limgrave, West", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_limgrave_west.png"},
	0x40002199: {Name: "Map: Weeping Peninsula", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_weeping_peninsula.png"},
	0x4000219A: {Name: "Map: Limgrave, East", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_limgrave_east.png"},
	0x4000219B: {Name: "Map: Liurnia, East", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_liurnia_east.png"},
	0x4000219C: {Name: "Map: Liurnia, North", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_liurnia_north.png"},
	0x4000219D: {Name: "Map: Liurnia, West", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_liurnia_west.png"},
	0x4000219E: {Name: "Map: Altus Plateau", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_altus_plateau.png"},
	0x4000219F: {Name: "Map: Leyndell, Royal Capital", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_leyndell_royal_capital.png"},
	0x400021A0: {Name: "Map: Mt. Gelmir", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_mt_gelmir.png"},
	0x400021A1: {Name: "Map: Caelid", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_caelid.png"},
	0x400021A2: {Name: "Map: Dragonbarrow", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/key_items/map_dragonbarrow.png"},
	0x400021A3: {Name: "Map: Mountaintops of the Giants, West", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_mountaintops_of_the_giants_west.png"},
	0x400021A4: {Name: "Map: Mountaintops of the Giants, East", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_mountaintops_of_the_giants_east.png"},
	0x400021A5: {Name: "Map: Ainsel River", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_ainsel_river.png"},
	0x400021A6: {Name: "Map: Lake of Rot", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_lake_of_rot.png"},
	0x400021A7: {Name: "Map: Siofra River", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_siofra_river.png"},
	0x400021A8: {Name: "Map: Mohgwyn Palace", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_mohgwyn_palace.png"},
	0x400021A9: {Name: "Map: Deeproot Depths", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_deeproot_depths.png"},
	0x400021AA: {Name: "Map: Consecrated Snowfield", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_consecrated_snowfield.png"},

	// ─── Region maps (DLC) ──────────────────────────────────────────────
	0x401EA618: {Name: "Map: Gravesite Plain", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_gravesite_plain.png", Flags: []string{"dlc"}},
	0x401EA619: {Name: "Map: Scadu Altus", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_scadu_altus.png", Flags: []string{"dlc"}},
	0x401EA61A: {Name: "Map: Southern Shore", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_southern_shore.png", Flags: []string{"dlc"}},
	0x401EA61B: {Name: "Map: Rauh Ruins", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_rauh_ruins.png", Flags: []string{"dlc"}},
	0x401EA61C: {Name: "Map: Abyss", Category: "info", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/tools/quest/map_abyss.png", Flags: []string{"dlc"}},
}
