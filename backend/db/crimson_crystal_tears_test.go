package db

import "testing"

func TestCrimsonCrystalTears_AreDistinctAndOrdered(t *testing.T) {
	const (
		flaskPickupID = uint32(0x40002AFA)
		secondTearID  = uint32(0x40002AFB)
	)

	items := GetItemsByCategory("key_items", "PC")
	indexes := map[uint32]int{}
	for index, item := range items {
		if item.ID == flaskPickupID || item.ID == secondTearID {
			if item.Name != "Crimson Crystal Tear" {
				t.Errorf("0x%08X name = %q, want Crimson Crystal Tear", item.ID, item.Name)
			}
			indexes[item.ID] = index
		}
	}
	if _, ok := indexes[flaskPickupID]; !ok {
		t.Fatal("Flask pickup Crimson Crystal Tear is missing from Item Database")
	}
	if _, ok := indexes[secondTearID]; !ok {
		t.Fatal("second Crimson Crystal Tear is missing from Item Database")
	}
	if indexes[flaskPickupID] >= indexes[secondTearID] {
		t.Errorf("Flask pickup Crimson tear index = %d, second tear index = %d; want Flask pickup first", indexes[flaskPickupID], indexes[secondTearID])
	}
}
