package migration

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGoodsStorageRegulationValuesDiscardReviewedSaveForgeValues(t *testing.T) {
	context := generationContext{}
	item := seed{
		HasLegacyItem:         true,
		Category:              "tools",
		MaxInventory:          7,
		MaxStorage:            8,
		GameMaxInventoryKnown: true,
		GameMaxInventory:      99,
		GameMaxStorageKnown:   true,
		GameMaxStorage:        100,
		GameLimits: &gameLimitsSeed{
			InventoryKnown: true,
			MaxInventory:   101,
			StorageKnown:   true,
			MaxStorage:     102,
		},
	}
	row := ParameterRow{
		RowID: 123,
		Fields: []ParameterField{
			{Name: "maxNum", RawValue: "17"},
			{Name: "maxRepositoryNum", RawValue: "18"},
		},
	}
	storage, err := context.buildStorage(
		item,
		schema.ItemFamilyGoods,
		row,
		true,
	)
	if err != nil {
		t.Fatalf("buildStorage: %v", err)
	}
	if storage.MaxInventory.Value != 17 || storage.MaxStorage.Value != 18 {
		t.Fatalf(
			"storage limits = %d/%d, want Regulation 17/18",
			storage.MaxInventory.Value,
			storage.MaxStorage.Value,
		)
	}
	if storage.GameMaxInventory.Value != 17 || storage.GameMaxStorage.Value != 18 {
		t.Fatalf(
			"game limits = %d/%d, want Regulation 17/18",
			storage.GameMaxInventory.Value,
			storage.GameMaxStorage.Value,
		)
	}
	if storage.GameMaxInventory.Provenance.Source !=
		sourceIDByRegulationTable[RegulationTableGoods] ||
		storage.GameMaxStorage.Provenance.Source !=
			sourceIDByRegulationTable[RegulationTableGoods] {
		t.Fatalf(
			"game-limit sources = %q/%q",
			storage.GameMaxInventory.Provenance.Source,
			storage.GameMaxStorage.Provenance.Source,
		)
	}
	if storage.MaxInventorySFV != nil || storage.MaxStorageSFV != nil {
		t.Fatalf("reviewed tool retains SaveForge storage values: %#v", storage)
	}
	if storage.GameMaxInventorySFV == nil ||
		storage.GameMaxInventorySFV.Value != 99 ||
		storage.GameMaxStorageSFV == nil ||
		storage.GameMaxStorageSFV.Value != 100 {
		t.Fatalf("SaveForge technical limits = %#v", storage)
	}
}

func TestStoragePromotesLegacyLimitsToSafeMode(t *testing.T) {
	tests := []struct {
		name     string
		category string
	}{
		{name: "bolstering material", category: bolsteringMaterialsCategory},
		{name: "incantation", category: incantationsCategory},
		{name: "sorcery", category: sorceriesCategory},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context := generationContext{}
			item := seed{
				HasLegacyItem: true,
				Category:      test.category,
				MaxInventory:  7,
				MaxStorage:    8,
			}
			row := ParameterRow{
				RowID: 123,
				Fields: []ParameterField{
					{Name: "maxNum", RawValue: "17"},
					{Name: "maxRepositoryNum", RawValue: "18"},
				},
			}
			storage, err := context.buildStorage(
				item,
				schema.ItemFamilyGoods,
				row,
				true,
			)
			if err != nil {
				t.Fatalf("buildStorage: %v", err)
			}
			if storage.MaxInventory.Value != 17 || storage.MaxStorage.Value != 18 {
				t.Fatalf("Regulation limits = %d/%d, want 17/18", storage.MaxInventory.Value, storage.MaxStorage.Value)
			}
			if storage.MaxInventorySFV != nil || storage.MaxStorageSFV != nil {
				t.Fatalf("item retains SaveForge limits: %#v", storage)
			}
			if storage.SafeModeMaxInventory == nil || storage.SafeModeMaxInventory.Value != 7 ||
				storage.SafeModeMaxStorage == nil || storage.SafeModeMaxStorage.Value != 8 {
				t.Fatalf("Safe Mode limits = %#v, want 7/8", storage)
			}
		})
	}
}

func TestCrystalTorrentStorageUsesRegulationAndDiscardsReviewedSaveForgeValues(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	context := generationContext{regulation: regulation}
	item := findSeed(t, collectLegacySnapshot().Items, 0x400011A8)
	identity, err := primaryRegulationForLegacyItem(*item)
	if err != nil {
		t.Fatal(err)
	}
	primary, exists, err := regulation.LookupFamilyRow(
		identity.Family,
		RegulationTableRolePrimary,
		identity.RowID,
	)
	if err != nil || !exists {
		t.Fatalf("Magic row lookup = %+v, %t, %v", primary, exists, err)
	}
	storage, err := context.buildStorage(
		*item,
		schema.ItemFamilySpell,
		primary.Row,
		true,
	)
	if err != nil {
		t.Fatalf("buildStorage: %v", err)
	}
	if storage.MaxInventory.Value != 99 ||
		storage.MaxStorage.Value != 600 ||
		storage.GameMaxInventory.Value != 99 ||
		storage.GameMaxStorage.Value != 600 {
		t.Fatalf("Crystal Torrent Regulation limits = %#v", storage)
	}
	if storage.MaxInventorySFV != nil || storage.MaxStorageSFV != nil {
		t.Fatalf("Crystal Torrent retains reviewed SaveForge limits: %#v", storage)
	}
}

func TestObservedCanonicalStorageOverrideRemains999(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	context := generationContext{regulation: regulation}
	snapshot := collectLegacySnapshot()
	for _, itemID := range []uint32{0x40000096} {
		item := findSeed(t, snapshot.Items, itemID)
		identity, err := primaryRegulationForLegacyItem(*item)
		if err != nil {
			t.Fatalf("item 0x%08X identity: %v", itemID, err)
		}
		primary, exists, err := regulation.LookupFamilyRow(
			identity.Family,
			RegulationTableRolePrimary,
			identity.RowID,
		)
		if err != nil || !exists {
			t.Fatalf("item 0x%08X lookup = %+v, %t, %v", itemID, primary, exists, err)
		}
		storage, err := context.buildStorage(
			*item,
			schema.ItemFamilyGoods,
			primary.Row,
			true,
		)
		if err != nil {
			t.Fatalf("item 0x%08X buildStorage: %v", itemID, err)
		}
		if storage.GameMaxInventory.Value != 999 ||
			storage.GameMaxStorage.Value != 999 ||
			storage.GameMaxInventory.Provenance.Source != sourceIDByRegulationTable[RegulationTableGoods] ||
			storage.GameMaxStorage.Provenance.Source != sourceIDByRegulationTable[RegulationTableGoods] {
			t.Fatalf("item 0x%08X protected limits = %#v", itemID, storage)
		}
		rawStorage, err := regulationUint32(primary.Row, "maxRepositoryNum")
		if err != nil {
			t.Fatal(err)
		}
		if rawStorage != 0 {
			t.Fatalf("item 0x%08X Regulation maxRepositoryNum = %d, want sentinel 0", itemID, rawStorage)
		}
	}
}

func TestSpellAndGoodsStorageGameLimitsMatchRegulation(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	context := generationContext{regulation: regulation}
	snapshot := collectLegacySnapshot()
	tests := []struct {
		name   string
		itemID uint32
		family schema.ItemFamily
	}{
		{name: "Glintstone Pebble", itemID: 0x40000FA0, family: schema.ItemFamilySpell},
		{name: "Fire Pot", itemID: 0x4000012C, family: schema.ItemFamilyGoods},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item := findSeed(t, snapshot.Items, test.itemID)
			identity, err := primaryRegulationForLegacyItem(*item)
			if err != nil {
				t.Fatalf("primaryRegulationForLegacyItem: %v", err)
			}
			primary, exists, err := regulation.LookupFamilyRow(
				identity.Family,
				RegulationTableRolePrimary,
				identity.RowID,
			)
			if err != nil || !exists {
				t.Fatalf("primary lookup = %+v, %t, %v", primary, exists, err)
			}
			storage, err := context.buildStorage(*item, test.family, primary.Row, true)
			if err != nil {
				t.Fatalf("buildStorage: %v", err)
			}
			goods, exists, err := regulation.LookupFamilyRow(
				RegulationFamilyGoods,
				RegulationTableRolePrimary,
				identity.RowID,
			)
			if err != nil || !exists {
				t.Fatalf("Goods lookup = %+v, %t, %v", goods, exists, err)
			}
			wantInventory, err := regulationUint32(goods.Row, "maxNum")
			if err != nil {
				t.Fatal(err)
			}
			wantStorage, err := regulationUint32(goods.Row, "maxRepositoryNum")
			if err != nil {
				t.Fatal(err)
			}
			if !storage.GameMaxInventory.Known || !storage.GameMaxStorage.Known {
				t.Fatalf("game limits are unknown: %#v", storage)
			}
			if wantInventory == 0 {
				wantInventory = storage.MaxInventory.Value
			}
			if wantStorage == 0 {
				if item.GameMaxStorageKnown {
					wantStorage = item.GameMaxStorage
				} else if item.GameLimits != nil && item.GameLimits.StorageKnown {
					wantStorage = item.GameLimits.MaxStorage
				}
			}
			if storage.GameMaxInventory.Provenance.Source != sourceIDByRegulationTable[RegulationTableGoods] ||
				storage.GameMaxStorage.Provenance.Source != sourceIDByRegulationTable[RegulationTableGoods] {
				t.Fatalf(
					"game-limit sources = %q/%q, want Regulation",
					storage.GameMaxInventory.Provenance.Source,
					storage.GameMaxStorage.Provenance.Source,
				)
			}
			if storage.GameMaxInventory.Value != wantInventory ||
				storage.GameMaxStorage.Value != wantStorage {
				t.Fatalf(
					"game limits = %d/%d, want %d/%d",
					storage.GameMaxInventory.Value,
					storage.GameMaxStorage.Value,
					wantInventory,
					wantStorage,
				)
			}
		})
	}
}
