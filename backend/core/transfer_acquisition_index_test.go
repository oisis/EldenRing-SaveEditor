package core

import "testing"

func TestAssignDestIndex_InventoryStartsInFreshBucketAboveReservedBoundary(t *testing.T) {
	slot := &SaveSlot{}
	slot.Inventory.NextAcquisitionSortId = InvEquipReservedMax

	got := assignDestIndex(slot, containerInventory)
	if got <= InvEquipReservedMax {
		t.Fatalf("assigned Index = %d, want above %d", got, InvEquipReservedMax)
	}
	if got>>1 == InvEquipReservedMax>>1 {
		t.Fatalf("assigned Index %d shares bucket %d with native boundary Index %d",
			got, got>>1, InvEquipReservedMax)
	}
}
