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
		theRing.MaxStorageSFV != nil {
		t.Fatalf("The Ring storage = %+v", theRing)
	}

	gestureCount := 0
	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil || item.Category.Value != gesturesCategory {
			continue
		}
		gestureCount++
		if item.Storage.MaxStorageSFV != nil {
			t.Fatalf("gesture 0x%08X retains maxStorage-sfv: %+v", item.GameID.Value, item.Storage)
		}
	}
	if gestureCount != 56 {
		t.Fatalf("gesture records = %d, want 56", gestureCount)
	}

	infoCount := 0
	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil || item.Category.Value != infoCategory {
			continue
		}
		infoCount++
		storage := item.Storage
		if storage.MaxInventory.Value != 1 || storage.MaxStorage.Value != 1 {
			t.Fatalf("info item 0x%08X Regulation limits = %+v, want 1/1", item.GameID.Value, storage)
		}
		if storage.MaxStorageSFV != nil {
			t.Fatalf("info item 0x%08X retains maxStorage-sfv: %+v", item.GameID.Value, storage)
		}
	}
	if infoCount != 102 {
		t.Fatalf("info records = %d, want 102", infoCount)
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

func TestSpellLegacyLimitsAreSafeModeCaps(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	wantByCategory := map[string]int{
		incantationsCategory: 128,
		sorceriesCategory:    85,
	}
	actualByCategory := make(map[string]int, len(wantByCategory))
	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil {
			continue
		}
		if _, wanted := wantByCategory[item.Category.Value]; !wanted {
			continue
		}
		actualByCategory[item.Category.Value]++
		storage := item.Storage
		if storage.MaxInventory.Value != 99 || storage.MaxStorage.Value != 600 {
			t.Fatalf("item 0x%08X Regulation limits = %+v, want 99/600", item.GameID.Value, storage)
		}
		if storage.MaxInventorySFV != nil || storage.MaxStorageSFV != nil {
			t.Fatalf("item 0x%08X retains legacy suffixes: %+v", item.GameID.Value, storage)
		}
		if storage.SafeModeMaxInventory == nil || storage.SafeModeMaxInventory.Value != 1 ||
			storage.SafeModeMaxStorage == nil || storage.SafeModeMaxStorage.Value != 0 {
			t.Fatalf("item 0x%08X Safe Mode limits = %+v, want 1/0", item.GameID.Value, storage)
		}
	}
	for category, want := range wantByCategory {
		if actualByCategory[category] != want {
			t.Fatalf("%s records = %d, want %d", category, actualByCategory[category], want)
		}
	}
}
