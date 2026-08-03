package migration

import (
	"strings"
	"testing"
)

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

func TestPotLegacyInventoryLimitsAreDiscarded(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	potIDs := []uint32{
		0x4000012C, // Fire Pot
		0x40000140, // Lightning Pot
		0x4000014A, // Fetid Pot
		0x40000154, // Swarm Pot
		0x4000015E, // Holy Water Pot
		0x40000172, // Poison Pot
		0x4000017C, // Oil Pot
		0x40000190, // Roped Fire Pot
		0x400001A4, // Roped Lightning Pot
		0x400001AE, // Roped Fetid Pot
		0x400001B8, // Roped Poison Pot
		0x400001C2, // Roped Oil Pot
		0x400001CC, // Roped Magic Pot
		0x400001D6, // Roped Fly Pot
		0x400001EA, // Roped Volcano Pot
		0x400001FE, // Roped Holy Water Pot
		0x40000258, // Volcano Pot
		0x40000280, // Sleep Pot
		0x4000028A, // Rancor Pot
		0x40000294, // Magic Pot
		0x401E873C, // Red Lightning Pot
		0x401E8746, // Frenzied Flame Pot
		0x401E8778, // Roped Frenzied Flame Pot
	}
	for _, itemID := range potIDs {
		storage := findGeneratedItem(t, catalog, itemID).Storage
		if storage.MaxInventory.Value != 10 || storage.MaxInventorySFV != nil {
			t.Fatalf("pot 0x%08X storage = %+v, want Regulation maxInventory 10 without maxInventory-sfv", itemID, storage)
		}
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

func TestSpiritAshLegacyStorageLimitIsSafeModeCap(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	count := 0
	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil || item.Category.Value != ashesCategory {
			continue
		}
		count++
		storage := item.Storage
		if storage.MaxInventory.Value != 1 || storage.MaxStorage.Value != 600 {
			t.Fatalf("item 0x%08X Regulation limits = %+v, want 1/600", item.GameID.Value, storage)
		}
		if storage.MaxStorageSFV != nil {
			t.Fatalf("item 0x%08X retains maxStorage-sfv: %+v", item.GameID.Value, storage)
		}
		if storage.SafeModeMaxStorage == nil || storage.SafeModeMaxStorage.Value != 1 {
			t.Fatalf("item 0x%08X Safe Mode storage = %+v, want 1", item.GameID.Value, storage)
		}
	}
	if count != 84 {
		t.Fatalf("spirit ash records = %d, want 84", count)
	}
}

func TestPrattlingPateLegacyStorageLimitIsSafeModeCap(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	count := 0
	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil || item.Category.Value != toolsCategory ||
			!strings.HasPrefix(item.Presentation.DisplayName.Value, "Prattling Pate") {
			continue
		}
		count++
		storage := item.Storage
		if storage.MaxInventory.Value != 1 || storage.MaxStorage.Value != 600 {
			t.Fatalf("item 0x%08X Regulation limits = %+v, want 1/600", item.GameID.Value, storage)
		}
		if storage.MaxInventorySFV != nil || storage.MaxStorageSFV != nil {
			t.Fatalf("item 0x%08X retains legacy suffixes: %+v", item.GameID.Value, storage)
		}
		if storage.SafeModeMaxInventory != nil ||
			storage.SafeModeMaxStorage == nil || storage.SafeModeMaxStorage.Value != 0 {
			t.Fatalf("item 0x%08X Safe Mode limits = %+v, want nil/0", item.GameID.Value, storage)
		}
	}
	if count != 9 {
		t.Fatalf("Prattling Pates = %d, want 9", count)
	}
}

func TestRemembranceLegacyLimitsAreSafeModeCaps(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	count := 0
	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil || item.Category.Value != toolsCategory ||
			(item.Presentation.DisplayName.Value != "Elden Remembrance" &&
				!strings.HasPrefix(item.Presentation.DisplayName.Value, "Remembrance")) {
			continue
		}
		count++
		storage := item.Storage
		if storage.MaxInventory.Value != 99 || storage.MaxStorage.Value != 600 {
			t.Fatalf("item 0x%08X Regulation limits = %+v, want 99/600", item.GameID.Value, storage)
		}
		if storage.MaxInventorySFV != nil || storage.MaxStorageSFV != nil {
			t.Fatalf("item 0x%08X retains legacy suffixes: %+v", item.GameID.Value, storage)
		}
		if storage.SafeModeMaxInventory == nil || storage.SafeModeMaxInventory.Value != 2 ||
			storage.SafeModeMaxStorage == nil || storage.SafeModeMaxStorage.Value != 0 {
			t.Fatalf("item 0x%08X Safe Mode limits = %+v, want 2/0", item.GameID.Value, storage)
		}
	}
	if count != 25 {
		t.Fatalf("Remembrances = %d, want 25", count)
	}
}

func TestSelectedToolStorageSaveForgeValues(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	telescope := findGeneratedItem(t, catalog, 0x400007F8).Storage
	if telescope.MaxInventory.Value != 1 || telescope.MaxStorage.Value != 600 ||
		telescope.MaxInventorySFV != nil || telescope.MaxStorageSFV != nil ||
		telescope.SafeModeMaxInventory != nil ||
		telescope.SafeModeMaxStorage == nil || telescope.SafeModeMaxStorage.Value != 0 {
		t.Fatalf("Telescope storage = %+v, want Regulation 1/600 and Safe Mode nil/0", telescope)
	}

	physick := findGeneratedItem(t, catalog, 0x400000FB).Storage
	if physick.MaxInventory.Value != 1 || physick.MaxStorage.Value != 1 ||
		physick.MaxInventorySFV != nil || physick.MaxStorageSFV != nil ||
		physick.SafeModeMaxInventory != nil || physick.SafeModeMaxStorage != nil {
		t.Fatalf("Flask of Wondrous Physick storage = %+v, want Regulation 1/1 without Safe Mode values", physick)
	}

	festering := findGeneratedItem(t, catalog, 0x4000006F).Storage
	if festering.MaxInventory.Value != 99 || festering.MaxStorage.Value != 99 ||
		festering.MaxInventorySFV != nil || festering.MaxStorageSFV != nil ||
		festering.SafeModeMaxInventory != nil || festering.SafeModeMaxStorage != nil {
		t.Fatalf("Festering Bloody Finger storage = %+v, want Regulation 99/99 without Safe Mode values", festering)
	}
}

func TestTearsFlaskLegacyLimitsAreDiscarded(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	countByFamily := map[string]int{}
	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil || item.Category.Value != toolsCategory {
			continue
		}

		name := item.Presentation.DisplayName.Value
		family := ""
		switch {
		case strings.HasPrefix(name, "Flask of Crimson Tears"):
			family = "crimson"
		case strings.HasPrefix(name, "Flask of Cerulean Tears"):
			family = "cerulean"
		default:
			continue
		}
		countByFamily[family]++

		storage := item.Storage
		if storage.MaxInventory.Value != 20 || storage.MaxStorage.Value != 20 {
			t.Fatalf("%s Regulation limits = %+v, want 20/20", name, storage)
		}
		if storage.MaxInventorySFV != nil || storage.MaxStorageSFV != nil ||
			storage.SafeModeMaxInventory != nil || storage.SafeModeMaxStorage != nil {
			t.Fatalf("%s retains reviewed storage values: %+v", name, storage)
		}
		if item.Goods == nil || item.Goods.IsDepositable.Value {
			t.Fatalf("%s isDepositable = %+v, want false", name, item.Goods)
		}
	}

	if countByFamily["crimson"] != 13 || countByFamily["cerulean"] != 13 {
		t.Fatalf("Tears flask records = %#v, want crimson/cerulean 13/13", countByFamily)
	}
}

func TestCutContentToolStorageSaveForgeValuesAreDiscarded(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	expectedLimits := map[string]struct {
		inventory uint32
		storage   uint32
	}{
		"?GoodsName? Holy Water Pot": {inventory: 999, storage: 600},
		"Deathsbane Jerky":           {inventory: 10, storage: 999},
		"Deathsbane White Jerky":     {inventory: 10, storage: 999},
		"Drawstring Freezing Grease": {inventory: 30, storage: 600},
		"Holy Water Grease":          {inventory: 5, storage: 600},
		"Roped Freezing Pot":         {inventory: 10, storage: 600},
	}

	for resourceIndex := range catalog.Resources {
		item := catalog.Resources[resourceIndex].Item
		if item == nil || item.Category.Value != toolsCategory {
			continue
		}

		name := item.Presentation.DisplayName.Value
		limits, found := expectedLimits[name]
		if !found {
			continue
		}
		delete(expectedLimits, name)

		if !item.Safety.CutContent.Known || !item.Safety.CutContent.Value {
			t.Fatalf("%s cutContent = %+v, want known true", name, item.Safety.CutContent)
		}
		storage := item.Storage
		if storage.MaxInventory.Value != limits.inventory || storage.MaxStorage.Value != limits.storage {
			t.Fatalf("%s Regulation limits = %+v, want %d/%d", name, storage, limits.inventory, limits.storage)
		}
		if storage.MaxInventorySFV != nil || storage.MaxStorageSFV != nil ||
			storage.SafeModeMaxInventory != nil || storage.SafeModeMaxStorage != nil {
			t.Fatalf("%s retains reviewed storage values: %+v", name, storage)
		}
	}

	if len(expectedLimits) != 0 {
		t.Fatalf("cut-content tools missing from catalog: %#v", expectedLimits)
	}
}
