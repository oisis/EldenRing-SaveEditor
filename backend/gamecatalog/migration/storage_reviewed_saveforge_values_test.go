package migration

import "testing"

func TestReviewedStorageSaveForgeValuesAreDiscarded(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	wait := findGeneratedItem(t, catalog, waitGestureItemID).Storage
	if wait.MaxInventory.Value != 0 ||
		wait.MaxInventorySFV != nil ||
		wait.MaxStorage.Value != 0 {
		t.Fatalf("Wait! storage = %+v", wait)
	}

	theRing := findGeneratedItem(t, catalog, 0x4000235A).Storage
	if theRing.MaxStorage.Value != 1 ||
		theRing.MaxStorageSFV == nil ||
		theRing.MaxStorageSFV.Value != 0 {
		t.Fatalf("The Ring storage = %+v", theRing)
	}
}
