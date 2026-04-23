package data

// AshOfWarFlagData holds metadata for an Ash of War duplication flag.
type AshOfWarFlagData struct {
	Name string
}

// AshOfWarFlags maps event flag ID → Ash of War duplication menu unlock.
// When set, the AoW appears in the Lost Grace duplication menu.
// Source: er-save-manager/event_flags_db.py
var AshOfWarFlags = map[uint32]AshOfWarFlagData{
	// Base game
	65810: {Name: "Impaling Thrust"},
	65811: {Name: "Piercing Fang"},
	65812: {Name: "Spinning Slash"},
	65813: {Name: "Repeating Thrust"},
	65814: {Name: "Double Slash"},
	65815: {Name: "Unsheathe"},
	65816: {Name: "Sword Dance"},
	65818: {Name: "Quickstep"},
	65819: {Name: "Bloodhound's Step"},
	65820: {Name: "Lion's Claw"},
	65821: {Name: "Stamp (Upward Cut)"},
	65822: {Name: "Stamp (Sweep)"},
	65823: {Name: "Wild Strikes"},
	65824: {Name: "Earthshaker"},
	65825: {Name: "Kick"},
	65826: {Name: "Ground Slam"},
	65827: {Name: "Hoarah Loux's Earthshaker"},
	65828: {Name: "Barbaric Roar"},
	65829: {Name: "War Cry"},
	65830: {Name: "Troll's Roar"},
	65831: {Name: "Braggart's Roar"},
	65832: {Name: "Endure"},
	65833: {Name: "Charge Forth"},
	65834: {Name: "Square Off"},
	65835: {Name: "Giant Hunt"},
	65836: {Name: "Spinning Strikes"},
	65837: {Name: "Storm Assault"},
	65838: {Name: "Stormcaller"},
	65839: {Name: "Storm Blade"},
	65840: {Name: "Vacuum Strike"},
	65841: {Name: "Storm Stomp"},
	65842: {Name: "Determination"},
	65843: {Name: "Royal Knight's Resolve"},
	65844: {Name: "Prelate's Charge"},
	65845: {Name: "Eruption"},
	65846: {Name: "Flaming Strike"},
	65847: {Name: "Black Flame Tornado"},
	65848: {Name: "Flame of the Redmanes"},
	65849: {Name: "Thunderbolt"},
	65850: {Name: "Lightning Slash"},
	65851: {Name: "Lightning Ram"},
	65852: {Name: "Loretta's Slash"},
	65853: {Name: "Spinning Weapon"},
	65854: {Name: "Glintblade Phalanx"},
	65855: {Name: "Glintstone Pebble"},
	65856: {Name: "Gravitas"},
	65857: {Name: "Carian Grandeur"},
	65858: {Name: "Carian Greatsword"},
	65859: {Name: "Waves of Darkness"},
	65860: {Name: "Cragblade"},
	65861: {Name: "Sacred Blade"},
	65862: {Name: "Prayerful Strike"},
	65863: {Name: "Golden Land"},
	65864: {Name: "Sacred Ring of Light"},
	65865: {Name: "Golden Slam"},
	65866: {Name: "Golden Vow"},
	65867: {Name: "Sacred Order"},
	65868: {Name: "Shared Order"},
	65869: {Name: "Beast's Roar"},
	65870: {Name: "Phantom Slash"},
	65871: {Name: "Spectral Lance"},
	65872: {Name: "Raptor of the Mists"},
	65873: {Name: "White Shadow's Lure"},
	65874: {Name: "Poison Moth Flight"},
	65875: {Name: "Poison Mist"},
	65876: {Name: "Blood Tax"},
	65877: {Name: "Bloody Slash"},
	65878: {Name: "Lifesteal Fist"},
	65879: {Name: "Blood Blade"},
	65880: {Name: "Assassin's Gambit"},
	65881: {Name: "Seppuku"},
	65882: {Name: "Ice Spear"},
	65883: {Name: "Chilling Mist"},
	65884: {Name: "Hoarfrost Stomp"},
	65885: {Name: "No Skill"},
	65886: {Name: "Shield Bash"},
	65887: {Name: "Shield Crash"},
	65888: {Name: "Barricade Shield"},
	65889: {Name: "Parry"},
	65890: {Name: "Carian Retaliation"},
	65891: {Name: "Storm Wall"},
	65892: {Name: "Golden Parry"},
	65893: {Name: "Thops's Barrier"},
	65894: {Name: "Holy Ground"},
	65895: {Name: "Vow of the Indomitable"},
	65896: {Name: "Barrage"},
	65897: {Name: "Mighty Shot"},
	65898: {Name: "Sky Shot"},
	65899: {Name: "Through and Through"},
	65900: {Name: "Enchanted Shot"},
	65901: {Name: "Rain of Arrows"},

	// DLC — Shadow of the Erdtree
	65910: {Name: "Dryleaf Whirlwind"},
	65911: {Name: "Aspects of the Crucible: Wings"},
	65912: {Name: "Spinning Gravity Thrust"},
	65913: {Name: "Palm Blast"},
	65914: {Name: "Piercing Throw"},
	65915: {Name: "Scattershot Throw"},
	65916: {Name: "Wall of Sparks"},
	65917: {Name: "Rolling Sparks"},
	65918: {Name: "Raging Beast"},
	65919: {Name: "Savage Claws"},
	65920: {Name: "Blind Spot"},
	65921: {Name: "Swift Slash"},
	65922: {Name: "Overhead Stance"},
	65923: {Name: "Wing Stance"},
	65924: {Name: "Blinkbolt"},
	65925: {Name: "Flame Skewer"},
	65926: {Name: "Savage Lion's Claw"},
	65927: {Name: "Divine Beast Frost Stomp"},
	65928: {Name: "Flame Spear"},
	65929: {Name: "Carian Sovereignty"},
	65930: {Name: "Shriek of Sorrow"},
	65931: {Name: "Ghostflame Call"},
	65932: {Name: "The Poison Flower Blooms Twice"},
	65933: {Name: "Igon's Drake Hunt"},
	65934: {Name: "Shield Strike"},
}

// AoWItemToFlagID maps AoW inventory item ID → duplication event flag ID.
// When an AoW item is added to inventory, set the corresponding flag
// so the AoW appears in the Lost Grace duplication menu.
var AoWItemToFlagID = map[uint32]uint32{
	0x80002710: 65820, // Lion's Claw
	0x80002774: 65810, // Impaling Thrust
	0x800027D8: 65811, // Piercing Fang
	0x8000283C: 65812, // Spinning Slash
	0x80002904: 65833, // Charge Forth
	0x80002968: 65821, // Stamp (Upward Cut)
	0x800029CC: 65822, // Stamp (Sweep)
	0x80002A30: 65876, // Blood Tax
	0x80002A94: 65813, // Repeating Thrust
	0x80002AF8: 65823, // Wild Strikes
	0x80002B5C: 65836, // Spinning Strikes
	0x80002BC0: 65814, // Double Slash
	0x80002C24: 65844, // Prelate's Charge
	0x80002C88: 65815, // Unsheathe
	0x80002CEC: 65834, // Square Off
	0x80002D50: 65835, // Giant Hunt
	0x80002E18: 65852, // Loretta's Slash
	0x80002E7C: 65874, // Poison Moth Flight
	0x80002EE0: 65853, // Spinning Weapon
	0x80002FA8: 65837, // Storm Assault
	0x8000300C: 65838, // Stormcaller
	0x80003070: 65816, // Sword Dance
	0x80004E20: 65854, // Glintblade Phalanx
	0x80004E84: 65861, // Sacred Blade
	0x80004EE8: 65882, // Ice Spear
	0x80004F4C: 65855, // Glintstone Pebble
	0x80004FB0: 65877, // Bloody Slash
	0x80005014: 65878, // Lifesteal Fist
	0x800050DC: 65845, // Eruption
	0x80005140: 65862, // Prayerful Strike
	0x800051A4: 65856, // Gravitas
	0x80005208: 65839, // Storm Blade
	0x800052D0: 65824, // Earthshaker
	0x80005334: 65863, // Golden Land
	0x80005398: 65846, // Flaming Strike
	0x80005460: 65849, // Thunderbolt
	0x800054C4: 65850, // Lightning Slash
	0x80005528: 65857, // Carian Grandeur
	0x8000558C: 65858, // Carian Greatsword
	0x800055F0: 65840, // Vacuum Slice → Vacuum Strike
	0x80005654: 65847, // Black Flame Tornado
	0x800056B8: 65864, // Sacred Ring of Light
	0x80005780: 65879, // Blood Blade
	0x800057E4: 65870, // Phantom Slash
	0x80005848: 65871, // Spectral Lance
	0x800058AC: 65883, // Chilling Mist
	0x80005910: 65875, // Poisonous Mist → Poison Mist
	0x80007530: 65886, // Shield Bash
	0x80007594: 65888, // Barricade Shield
	0x800075F8: 65889, // Parry
	0x80007724: 65890, // Carian Retaliation
	0x80007788: 65891, // Storm Wall
	0x800077EC: 65892, // Golden Parry
	0x80007850: 65887, // Shield Crash
	0x800078B4: 65885, // No Skill
	0x80007918: 65893, // Thops's Barrier
	0x80009C40: 65899, // Through and Through
	0x80009CA4: 65896, // Barrage
	0x80009D08: 65897, // Mighty Shot
	0x80009DD0: 65900, // Enchanted Shot
	0x80009E34: 65898, // Sky Shot
	0x80009E98: 65901, // Rain of Arrows
	0x8000C3B4: 65884, // Hoarfrost Stomp
	0x8000C418: 65841, // Storm Stomp
	0x8000C47C: 65825, // Kick
	0x8000C4E0: 65851, // Lightning Ram
	0x8000C544: 65848, // Flame of the Redmanes
	0x8000C5A8: 65826, // Ground Slam
	0x8000C60C: 65865, // Golden Slam
	0x8000C670: 65859, // Waves of Darkness
	0x8000C6D4: 65827, // Hoarah Loux's Earthshaker
	0x8000EA60: 65842, // Determination
	0x8000EAC4: 65843, // Royal Knight's Resolve
	0x8000EB28: 65880, // Assassin's Gambit
	0x8000EB8C: 65866, // Golden Vow
	0x8000EBF0: 65867, // Sacred Order
	0x8000EC54: 65868, // Shared Order
	0x8000ECB8: 65881, // Seppuku
	0x8000ED1C: 65860, // Cragblade
	0x8000FDE8: 65828, // Barbaric Roar
	0x8000FE4C: 65829, // War Cry
	0x8000FEB0: 65869, // Beast's Roar
	0x8000FF14: 65830, // Troll's Roar
	0x8000FF78: 65831, // Braggart's Roar
	0x80011170: 65832, // Endure
	0x800111D4: 65895, // Vow of the Indomitable
	0x80011238: 65894, // Holy Ground
	0x80013880: 65818, // Quickstep
	0x800138E4: 65819, // Bloodhound's Step
	0x80013948: 65872, // Raptor of the Mists
	0x80014C08: 65873, // White Shadow's Lure
	0x80030D40: 65910, // Dryleaf Whirlwind
	0x80030DA4: 65911, // Aspects of the Crucible: Wings
	0x80061A80: 65912, // Spinning Gravity Thrust
	0x80061E68: 65913, // Palm Blast
	0x80062250: 65914, // Piercing Throw
	0x80062638: 65915, // Scattershot Throw
	0x80062A20: 65916, // Wall of Sparks
	0x80062E08: 65917, // Rolling Sparks
	0x800631F0: 65918, // Raging Beast
	0x800635D8: 65919, // Savage Claws
	0x80063DA8: 65920, // Blind Spot
	0x80064190: 65921, // Swift Slash
	0x80064578: 65922, // Overhead Stance
	0x80064960: 65923, // Wing Stance
	0x80064D48: 65924, // Blinkbolt
	0x80065130: 65925, // Flame Skewer
	0x80065518: 65926, // Savage Lion's Claw
	0x80065900: 65927, // Divine Beast Frost Stomp
	0x80065CE8: 65928, // Flame Spear
	0x800660D0: 65929, // Carian Sovereignty
	0x800664B8: 65930, // Shriek of Sorrow
	0x80067070: 65931, // Ghostflame Call
	0x8007B4A8: 65932, // The Poison Flower Blooms Twice
	0x80085CA0: 65933, // Igon's Drake Hunt
	0x800C3500: 65934, // Shield Strike
}
