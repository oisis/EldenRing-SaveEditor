package data

// GestureEmptySentinel is the value used for empty gesture slots in the save file.
const GestureEmptySentinel = uint32(0xFFFFFFFE)

// GestureDef defines a gesture with both possible body-type variant IDs.
// Some gestures have a fixed ID (same for all characters); others have two
// variants: EvenID (body type B) and OddID = EvenID+1 (body type A).
// Source: binary analysis of real save files (PC + PS4), spec/08-spells-gestures.md.
type GestureDef struct {
	Name     string
	Category string
	EvenID   uint32 // primary ID — used by body type B (or fixed ID for non-variable gestures)
	OddID    uint32 // alternate ID — EvenID+1 for variable gestures, 0 if fixed
}

// AllGestures is the canonical list of all known gestures.
// Order: by category, then by ID.
var AllGestures = []GestureDef{
	// Greetings
	{Name: "Bow", Category: "Greetings", EvenID: 1, OddID: 0},                    // fixed odd
	{Name: "Polite Bow", Category: "Greetings", EvenID: 2, OddID: 3},             // variable
	{Name: "My Thanks", Category: "Greetings", EvenID: 4, OddID: 5},              // variable
	{Name: "Curtsy", Category: "Greetings", EvenID: 6, OddID: 7},                 // variable
	{Name: "Reverential Bow", Category: "Greetings", EvenID: 8, OddID: 9},        // variable
	{Name: "My Lord", Category: "Greetings", EvenID: 10, OddID: 11},              // variable
	{Name: "Warm Welcome", Category: "Greetings", EvenID: 13, OddID: 0},          // fixed odd
	{Name: "Wave", Category: "Greetings", EvenID: 15, OddID: 0},                  // fixed odd
	{Name: "Casual Greeting", Category: "Greetings", EvenID: 16, OddID: 17},      // variable
	{Name: "Strength!", Category: "Greetings", EvenID: 18, OddID: 19},            // variable
	{Name: "As You Wish", Category: "Greetings", EvenID: 20, OddID: 21},          // variable
	// Gesturing
	{Name: "Point Forwards", Category: "Gesturing", EvenID: 41, OddID: 0},        // fixed odd
	{Name: "Point Upwards", Category: "Gesturing", EvenID: 43, OddID: 0},         // fixed odd
	{Name: "Point Downwards", Category: "Gesturing", EvenID: 45, OddID: 0},       // fixed odd
	{Name: "Beckon", Category: "Gesturing", EvenID: 47, OddID: 0},                // fixed odd
	{Name: "Wait!", Category: "Gesturing", EvenID: 49, OddID: 0},                 // fixed odd
	{Name: "Calm Down!", Category: "Gesturing", EvenID: 50, OddID: 0},            // fixed even
	{Name: "Nod In Thought", Category: "Gesturing", EvenID: 60, OddID: 61},       // variable
	// Submissive
	{Name: "Extreme Repentance", Category: "Submissive", EvenID: 80, OddID: 81},  // variable
	{Name: "Grovel For Mercy", Category: "Submissive", EvenID: 82, OddID: 83},    // variable
	// Battle
	{Name: "Rallying Cry", Category: "Battle", EvenID: 101, OddID: 0},            // fixed odd
	{Name: "Heartening Cry", Category: "Battle", EvenID: 102, OddID: 0},          // fixed
	{Name: "By My Sword", Category: "Battle", EvenID: 104, OddID: 105},           // variable
	{Name: "Hoslow's Oath", Category: "Battle", EvenID: 106, OddID: 107},         // variable
	{Name: "Fire Spur Me", Category: "Battle", EvenID: 108, OddID: 109},          // variable
	{Name: "The Carian Oath", Category: "Battle", EvenID: 110, OddID: 0},         // fixed
	// Celebration
	{Name: "Bravo!", Category: "Celebration", EvenID: 120, OddID: 121},            // variable
	{Name: "Jump for Joy", Category: "Celebration", EvenID: 141, OddID: 0},        // fixed odd
	{Name: "Triumphant Delight", Category: "Celebration", EvenID: 142, OddID: 143}, // variable
	{Name: "Fancy Spin", Category: "Celebration", EvenID: 144, OddID: 0},          // fixed
	{Name: "Finger Snap", Category: "Celebration", EvenID: 146, OddID: 147},       // variable
	// Emotion
	{Name: "Dejection", Category: "Emotion", EvenID: 161, OddID: 0},              // fixed odd
	{Name: "What Do You Want?", Category: "Emotion", EvenID: 196, OddID: 197},    // variable
	// Resting
	{Name: "Patches' Crouch", Category: "Resting", EvenID: 180, OddID: 181},      // variable
	{Name: "Crossed Legs", Category: "Resting", EvenID: 182, OddID: 183},         // variable
	{Name: "Rest", Category: "Resting", EvenID: 185, OddID: 0},                   // fixed odd
	{Name: "Sitting Sideways", Category: "Resting", EvenID: 186, OddID: 187},     // variable
	{Name: "Dozing Cross-Legged", Category: "Resting", EvenID: 188, OddID: 189},  // variable
	{Name: "Spread Out", Category: "Resting", EvenID: 190, OddID: 191},           // variable
	{Name: "Fetal Position", Category: "Resting", EvenID: 192, OddID: 0},         // fixed
	{Name: "Balled Up", Category: "Resting", EvenID: 194, OddID: 195},            // variable
	// Prayer
	{Name: "Prayer", Category: "Prayer", EvenID: 200, OddID: 201},                // variable
	{Name: "Desperate Prayer", Category: "Prayer", EvenID: 202, OddID: 0},        // fixed
	{Name: "Rapture", Category: "Prayer", EvenID: 204, OddID: 205},               // variable
	{Name: "Erudition", Category: "Prayer", EvenID: 206, OddID: 207},             // variable
	{Name: "Outer Order", Category: "Prayer", EvenID: 208, OddID: 0},             // fixed
	{Name: "Inner Order", Category: "Prayer", EvenID: 210, OddID: 211},           // variable
	{Name: "Golden Order Totality", Category: "Prayer", EvenID: 212, OddID: 213}, // variable
	// Special
	{Name: "The Ring", Category: "Special", EvenID: 216, OddID: 0},               // fixed
	{Name: "The Ring (Co-op)", Category: "Special", EvenID: 218, OddID: 219},     // variable
	// DLC — Shadow of the Erdtree
	{Name: "Ring of Miquella", Category: "Special", EvenID: 220, OddID: 0},       // fixed
	{Name: "May the Best Win", Category: "Battle", EvenID: 222, OddID: 223},      // variable
	{Name: "The Two Fingers", Category: "Gesturing", EvenID: 224, OddID: 0},      // fixed
	{Name: "DLC Gesture (226)", Category: "Special", EvenID: 226, OddID: 227},    // variable, unknown name
	{Name: "Let Us Go Together", Category: "Greetings", EvenID: 228, OddID: 0},   // fixed
	{Name: "O Mother", Category: "Prayer", EvenID: 230, OddID: 231},              // variable
	{Name: "DLC Gesture (232)", Category: "Special", EvenID: 232, OddID: 0},      // fixed, unknown name
}

// gestureByID is a reverse lookup: save slot ID → index in AllGestures.
// Built once at init, includes both EvenID and OddID entries.
var gestureByID map[uint32]int

func init() {
	gestureByID = make(map[uint32]int, len(AllGestures)*2)
	for i, g := range AllGestures {
		gestureByID[g.EvenID] = i
		if g.OddID != 0 {
			gestureByID[g.OddID] = i
		}
	}
}

// LookupGestureBySlotID returns the gesture index and true if found, -1 and false otherwise.
func LookupGestureBySlotID(id uint32) (int, bool) {
	idx, ok := gestureByID[id]
	return idx, ok
}

// DetectBodyTypeOffset examines existing gesture slot values and returns the
// body-type offset: 0 for type B (even IDs), 1 for type A (odd IDs).
// Only checks variable gestures (those with OddID != 0).
func DetectBodyTypeOffset(slotValues []uint32) uint32 {
	evenCount := 0
	oddCount := 0
	for _, v := range slotValues {
		if v == GestureEmptySentinel || v == 0 {
			continue
		}
		idx, ok := gestureByID[v]
		if !ok {
			continue
		}
		g := AllGestures[idx]
		if g.OddID == 0 {
			continue // fixed gesture, skip
		}
		if v == g.EvenID {
			evenCount++
		} else {
			oddCount++
		}
	}
	if oddCount > evenCount {
		return 1
	}
	return 0
}

// Gestures is the item database for gestures (for the item browser / DatabaseTab).
// These use ITEM IDs (0x4000xxxx), not gesture slot IDs.
var Gestures = map[uint32]ItemData{
	0x40002328: {Name: "Bow", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/bow.png"},
	0x40002329: {Name: "Polite Bow", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/polite_bow.png"},
	0x4000232A: {Name: "My Thanks", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/my_thanks.png"},
	0x4000232B: {Name: "Curtsy", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/curtsy.png"},
	0x4000232C: {Name: "Reverential Bow", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/reverential_bow.png"},
	0x4000232D: {Name: "My Lord", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/my_lord.png"},
	0x4000232E: {Name: "Warm Welcome", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/warm_welcome.png"},
	0x4000232F: {Name: "Wave", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/wave.png"},
	0x40002330: {Name: "Casual Greeting", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/casual_greeting.png"},
	0x40002331: {Name: "Strength!", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/strength.png"},
	0x40002332: {Name: "As You Wish", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/as_you_wish.png"},
	0x40002333: {Name: "Point Forwards", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/point_forwards.png"},
	0x40002334: {Name: "Point Upwards", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/point_upwards.png"},
	0x40002335: {Name: "Point Downwards", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/point_downwards.png"},
	0x40002336: {Name: "Beckon", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/beckon.png"},
	0x40002337: {Name: "Wait!", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/wait.png"},
	0x40002338: {Name: "Calm Down!", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/calm_down.png"},
	0x40002339: {Name: "Nod In Thought", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/nod_in_thought.png"},
	0x4000233A: {Name: "Extreme Repentance", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/extreme_repentance.png"},
	0x4000233B: {Name: "Grovel For Mercy", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/grovel_for_mercy.png"},
	0x4000233C: {Name: "Rallying Cry", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/rallying_cry.png"},
	0x4000233D: {Name: "Heartening Cry", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/heartening_cry.png"},
	0x4000233E: {Name: "By My Sword", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/by_my_sword.png"},
	0x4000233F: {Name: "Hoslow's Oath", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/hoslows_oath.png"},
	0x40002340: {Name: "Fire Spur Me", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/fire_spur_me.png"},
	0x40002341: {Name: "The Carian Oath", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/the_carian_oath.png"},
	0x40002342: {Name: "Bravo!", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/bravo.png"},
	0x40002343: {Name: "Jump for Joy", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/jump_for_joy.png"},
	0x40002344: {Name: "Triumphant Delight", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/triumphant_delight.png"},
	0x40002345: {Name: "Fancy Spin", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/fancy_spin.png"},
	0x40002346: {Name: "Finger Snap", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/finger_snap.png"},
	0x40002347: {Name: "Dejection", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/dejection.png"},
	0x40002348: {Name: "Patches' Crouch", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/patches_crouch.png"},
	0x40002349: {Name: "Crossed Legs", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/crossed_legs.png"},
	0x4000234A: {Name: "Rest", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/rest.png"},
	0x4000234B: {Name: "Sitting Sideways", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/sitting_sideways.png"},
	0x4000234C: {Name: "Dozing Cross-Legged", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/dozing_cross_legged.png"},
	0x4000234D: {Name: "Spread Out", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/spread_out.png"},
	0x4000234E: {Name: "Fetal Position", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/fetal_position.png"},
	0x4000234F: {Name: "Balled Up", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/balled_up.png"},
	0x40002350: {Name: "What Do You Want?", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/what_do_you_want.png"},
	0x40002351: {Name: "Prayer", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/prayer.png"},
	0x40002352: {Name: "Desperate Prayer", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/desperate_prayer.png"},
	0x40002353: {Name: "Rapture", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/rapture.png"},
	0x40002355: {Name: "Erudition", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/erudition.png"},
	0x40002356: {Name: "Outer Order", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/outer_order.png"},
	0x40002357: {Name: "Inner Order", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/inner_order.png"},
	0x40002358: {Name: "Golden Order Totality", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/golden_order_totality.png"},
	0x4000235A: {Name: "The Ring", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/the_ring.png"},
	0x401EA7A8: {Name: "Ring of Miquella", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/ring_of_miquella.png"},
	0x401EA7A9: {Name: "May the Best Win", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/may_the_best_win.png"},
	0x401EA7AA: {Name: "The Two Fingers", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/the_two_fingers.png"},
	0x401EA7AB: {Name: "Let Us Go Together", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/let_us_go_together.png"},
	0x401EA7AC: {Name: "O Mother", Category: "gestures", MaxInventory: 1, MaxStorage: 0, MaxUpgrade: 0, IconPath: "items/gestures/o_mother.png"},
}
