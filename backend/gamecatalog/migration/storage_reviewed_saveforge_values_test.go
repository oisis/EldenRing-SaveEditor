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

func TestBolsteringMaterialLegacyLimitsAreSafeModeCaps(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expected := map[uint32]struct {
		inventory uint32
		storage   uint32
	}{
		0x4000271A: {inventory: 30, storage: 0},  // Golden Seed
		0x40002724: {inventory: 12, storage: 0},  // Sacred Tear
		0x4000279C: {inventory: 18, storage: 18}, // Ancient Dragon Smithing Stone
		0x400027B8: {inventory: 15, storage: 15}, // Somber Ancient Dragon Smithing Stone
		0x40002A9D: {inventory: 12, storage: 12}, // Great Grave Glovewort
		0x40002AA7: {inventory: 9, storage: 9},   // Great Ghost Glovewort
		0x401EAB90: {inventory: 50, storage: 0},  // Scadutree Fragment
		0x401EABF4: {inventory: 25, storage: 0},  // Revered Spirit Ash
	}

	for itemID, want := range expected {
		storage := findGeneratedItem(t, catalog, itemID).Storage
		if storage.MaxInventorySFV != nil || storage.MaxStorageSFV != nil {
			t.Fatalf("item 0x%08X retains legacy suffixes: %+v", itemID, storage)
		}
		if storage.SafeModeMaxInventory == nil ||
			storage.SafeModeMaxInventory.Value != want.inventory ||
			storage.SafeModeMaxStorage == nil ||
			storage.SafeModeMaxStorage.Value != want.storage {
			t.Fatalf(
				"item 0x%08X Safe Mode limits = %+v, want %d/%d",
				itemID,
				storage,
				want.inventory,
				want.storage,
			)
		}
	}
}
