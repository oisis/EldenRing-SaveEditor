package data

import "testing"

// TestContainerPickupFlags_Counts verifies each container has exactly as many
// pickup flag IDs as its MaxInventory — guarantees flag list covers every
// world pickup for that container.
func TestContainerPickupFlags_Counts(t *testing.T) {
	cases := []struct {
		container uint32
		name      string
	}{
		{CrackedPotKeyItemID, "Cracked Pot"},
		{RitualPotKeyItemID, "Ritual Pot"},
		{PerfumeBottleKeyItemID, "Perfume Bottle"},
		{HeftyCrackedPotKeyItemID, "Hefty Cracked Pot"},
	}
	for _, c := range cases {
		flags := ContainerPickupFlags[c.container]
		cap := KeyItems[c.container].MaxInventory
		if uint32(len(flags)) != cap {
			t.Errorf("%s: %d flags, want %d (MaxInventory)", c.name, len(flags), cap)
		}
	}
}

// TestContainerPickupFlags_Step10 verifies flags within a group differ by 10.
func TestContainerPickupFlags_Step10(t *testing.T) {
	for cID, flags := range ContainerPickupFlags {
		for i := 1; i < len(flags); i++ {
			if flags[i]-flags[i-1] != 10 {
				t.Errorf("container 0x%X flag[%d]=%d not step-10 from prev %d", cID, i, flags[i], flags[i-1])
			}
		}
	}
}

// TestContainerPickupFlags_NoOverlap verifies no flag is shared between
// containers — pickup flags must be unique to one container.
func TestContainerPickupFlags_NoOverlap(t *testing.T) {
	seen := map[uint32]uint32{} // flag → container
	for cID, flags := range ContainerPickupFlags {
		for _, f := range flags {
			if other, dup := seen[f]; dup {
				t.Errorf("flag %d in both 0x%X and 0x%X", f, other, cID)
			}
			seen[f] = cID
		}
	}
}

// TestContainerPickupFlags_AllContainersCovered verifies every key item ID in
// the four constants has a pickup flag list (no missing entries).
func TestContainerPickupFlags_AllContainersCovered(t *testing.T) {
	required := []uint32{
		CrackedPotKeyItemID,
		RitualPotKeyItemID,
		PerfumeBottleKeyItemID,
		HeftyCrackedPotKeyItemID,
	}
	for _, c := range required {
		if _, ok := ContainerPickupFlags[c]; !ok {
			t.Errorf("container 0x%X missing from ContainerPickupFlags", c)
		}
	}
}

// TestContainerVendorPurchaseFlags_KaleCrackedPot verifies Kale's purchase
// flag (710580) is mapped to the Cracked Pot container.
func TestContainerVendorPurchaseFlags_KaleCrackedPot(t *testing.T) {
	flags, ok := ContainerVendorPurchaseFlags[CrackedPotKeyItemID]
	if !ok {
		t.Fatalf("Cracked Pot missing from ContainerVendorPurchaseFlags")
	}
	if len(flags) != 1 || flags[0] != 710580 {
		t.Errorf("Cracked Pot vendor flags = %v, want [710580]", flags)
	}
}

// TestContainerVendorPurchaseFlags_OtherContainersHaveNone verifies the three
// other containers have no vendor flags (no vanilla merchant sells them).
func TestContainerVendorPurchaseFlags_OtherContainersHaveNone(t *testing.T) {
	for _, c := range []uint32{RitualPotKeyItemID, PerfumeBottleKeyItemID, HeftyCrackedPotKeyItemID} {
		if flags, ok := ContainerVendorPurchaseFlags[c]; ok && len(flags) > 0 {
			t.Errorf("container 0x%X unexpectedly has vendor flags %v (no vanilla merchant sells it)", c, flags)
		}
	}
}
