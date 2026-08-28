package data

import "testing"

// Regression anchors for the Regulation 1.17 data refresh.
//
// 1.17 rebalanced a small set of weapons and shields. These cases pin the
// representative changes plus one unchanged neighbour, so a stale or
// accidentally reverted generated table fails here instead of silently
// showing 1.16 numbers in the UI.
func TestRegulation117_WeaponStatChanges(t *testing.T) {
	cases := []struct {
		id    uint32
		name  string
		field string
		got   func(WeaponStatsV1) float64
		want  float64
	}{
		// Changed in 1.17.
		{0x00269AD0, "Carian Sorcery Sword", "AttackPhysical",
			func(s WeaponStatsV1) float64 { return float64(s.AttackPhysical) }, 38},
		{0x03AA6A60, "Firespark Perfume Bottle", "AttackFire",
			func(s WeaponStatsV1) float64 { return float64(s.AttackFire) }, 118},
		{0x01E84800, "Dragon Towershield", "GuardBoost",
			func(s WeaponStatsV1) float64 { return float64(s.GuardBoost) }, 71},
		{0x01E84800, "Dragon Towershield", "Weight",
			func(s WeaponStatsV1) float64 { return s.Weight }, 16.5},
		// Unchanged in 1.17 — guards against a blanket rewrite of the table.
		{0x01EA43D0, "Fingerprint Stone Shield", "GuardBoost",
			func(s WeaponStatsV1) float64 { return float64(s.GuardBoost) }, 77},
		{0x01EA43D0, "Fingerprint Stone Shield", "Weight",
			func(s WeaponStatsV1) float64 { return s.Weight }, 29},
	}
	for _, tc := range cases {
		s, ok := WeaponStatsV1ByID[tc.id]
		if !ok {
			t.Errorf("WeaponStatsV1ByID[0x%08X] (%s): missing entry", tc.id, tc.name)
			continue
		}
		if got := tc.got(s); got != tc.want {
			t.Errorf("0x%08X (%s) %s = %g, want %g", tc.id, tc.name, tc.field, got, tc.want)
		}
	}
}

// TestRegulation117_DragonTowershieldWeight anchors the shared ItemWeights
// table, which is generated independently of WeaponStatsV1 and must agree.
func TestRegulation117_DragonTowershieldWeight(t *testing.T) {
	const id = uint32(0x01E84800)
	w, ok := ItemWeights[id]
	if !ok {
		t.Fatalf("ItemWeights[0x%08X] (Dragon Towershield): missing entry", id)
	}
	if w != 16.5 {
		t.Errorf("ItemWeights[0x%08X] (Dragon Towershield) = %g, want 16.5", id, w)
	}
}

// TestRegulation117_DuelingShieldSortId anchors the 1.17 sort renumbering of
// the DLC thrusting-shield block.
func TestRegulation117_DuelingShieldSortId(t *testing.T) {
	const id = uint32(0x03B9ACA0)
	k, ok := ItemSortKeys[id]
	if !ok {
		t.Fatalf("ItemSortKeys[0x%08X] (Dueling Shield): missing entry", id)
	}
	if k.SortId != 7400500 {
		t.Errorf("ItemSortKeys[0x%08X] (Dueling Shield).SortId = %d, want 7400500", id, k.SortId)
	}
}

// TestRegulation117_NewGoodsLimits pins the technical limits of the three goods
// rows added by 1.17 (Spectral Steed Attire). The rows are now also present in
// the public item catalog as Key Items; this test covers only their generated
// regulation limits.
func TestRegulation117_NewGoodsLimits(t *testing.T) {
	for _, id := range []uint32{0x401EAA00, 0x401EAA0A, 0x401EAA14} {
		l, ok := GameLimitsByItemID[id]
		if !ok {
			t.Errorf("GameLimitsByItemID[0x%08X]: missing entry", id)
			continue
		}
		if l.MaxInventory != 1 || l.MaxStorage != 1 {
			t.Errorf("GameLimitsByItemID[0x%08X] = inv %d/storage %d, want 1/1",
				id, l.MaxInventory, l.MaxStorage)
		}
		if !l.InventoryKnown || !l.StorageKnown {
			t.Errorf("GameLimitsByItemID[0x%08X] = inventoryKnown %t/storageKnown %t, want true/true",
				id, l.InventoryKnown, l.StorageKnown)
		}
	}
}
