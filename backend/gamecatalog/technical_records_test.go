package gamecatalog_test

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestBareArmorTechnicalRecordsAreHiddenAndSlotSpecific(t *testing.T) {
	t.Parallel()

	catalog := newRealCatalog(t)
	for _, want := range []struct {
		key      string
		gameID   uint32
		name     string
		category string
		slot     schema.EquipmentSlot
		rowID    uint32
	}{
		{key: "10002710", gameID: 0x10002710, name: "Bare Head", category: "head", slot: schema.EquipmentSlotHead, rowID: 10000},
		{key: "10002774", gameID: 0x10002774, name: "Bare Body", category: "chest", slot: schema.EquipmentSlotChest, rowID: 10100},
		{key: "100027D8", gameID: 0x100027D8, name: "Bare Arms", category: "arms", slot: schema.EquipmentSlotArms, rowID: 10200},
		{key: "1000283C", gameID: 0x1000283C, name: "Bare Legs", category: "legs", slot: schema.EquipmentSlotLegs, rowID: 10300},
	} {
		want := want
		t.Run(want.key, func(t *testing.T) {
			t.Parallel()

			resource, found := catalog.ItemByGameID(want.gameID)
			if !found {
				t.Fatalf("ItemByGameID(0x%08X) did not resolve the technical record", want.gameID)
			}
			if resource.Key != want.key || resource.Kind != schema.ResourceKindItem || resource.Item == nil {
				t.Fatalf("ItemByGameID(0x%08X) = %+v, want item key %q", want.gameID, resource, want.key)
			}

			item := resource.Item
			if item.Family.Value != schema.ItemFamilyArmor || item.Category.Value != want.category {
				t.Errorf("item classification = (%q, %q), want (%q, %q)",
					item.Family.Value, item.Category.Value, schema.ItemFamilyArmor, want.category)
			}
			if !item.Presentation.Name.Known || item.Presentation.Name.Value != want.name ||
				item.Presentation.Name.Provenance.Source != schema.SourceSaveForgeLegacy {
				t.Errorf("technical record name = %+v, want legacy fallback %q",
					item.Presentation.Name, want.name)
			}
			if !item.Safety.NoDatabase.Known || !item.Safety.NoDatabase.Value {
				t.Error("technical record is no longer hidden from the item database")
			}
			if !item.Capabilities.Equipment.Known || !item.Capabilities.Equipment.Enabled ||
				item.Capabilities.Equipment.Rules == nil ||
				len(item.Capabilities.Equipment.Rules.AllowedSlots) != 1 ||
				item.Capabilities.Equipment.Rules.AllowedSlots[0] != want.slot {
				t.Errorf("equipment capability = %+v, want only slot %q",
					item.Capabilities.Equipment, want.slot)
			}
			if item.Armor == nil || !item.Armor.SourceRowID.Known || item.Armor.SourceRowID.Value != want.rowID {
				t.Errorf("armor source row = %+v, want EquipParamProtector row %d", item.Armor, want.rowID)
			}
		})
	}
}

func TestFilledPhysickStateIsAnAliasNotASeparateItem(t *testing.T) {
	t.Parallel()

	catalog := newRealCatalog(t)
	const (
		aliasID     = uint32(0x400000FA)
		canonicalID = uint32(0x400000FB)
	)

	resource, found := catalog.ItemByGameID(aliasID)
	if !found || resource.Key != "400000FB" || resource.Item == nil {
		t.Fatalf("ItemByGameID(0x%08X) = (%+v, %t), want canonical item 400000FB", aliasID, resource, found)
	}
	if resource.Item.GameID.Value != canonicalID {
		t.Errorf("alias changed the canonical game ID to 0x%08X, want 0x%08X",
			resource.Item.GameID.Value, canonicalID)
	}
	if len(resource.Item.Aliases) != 1 || !resource.Item.Aliases[0].GameID.Known ||
		resource.Item.Aliases[0].GameID.Value != aliasID {
		t.Errorf("canonical item aliases = %+v, want only 0x%08X", resource.Item.Aliases, aliasID)
	}
	if _, err := catalog.ResourceByKindAndKey(schema.ResourceKindItem, "400000FA"); err == nil {
		t.Fatal("the Physick save-side alias was also published as a standalone item")
	}
	if _, found := catalog.ItemByGameID(0x400000F9); found {
		t.Fatal("an adjacent unknown Physick game ID unexpectedly resolved")
	}
}

func TestUnarmedIsAHiddenTechnicalWeaponWithoutAFakeSubcategory(t *testing.T) {
	t.Parallel()

	resource, found := newRealCatalog(t).ItemByGameID(0x0001ADB0)
	if !found || resource.Key != "0001ADB0" || resource.Item == nil {
		t.Fatalf("ItemByGameID(Unarmed) = (%+v, %t), want item 0001ADB0", resource, found)
	}

	item := resource.Item
	if item.Subcategory.Known || item.Subcategory.Value != "" {
		t.Errorf("Unarmed subcategory = %+v, want an explicitly unknown empty value", item.Subcategory)
	}
	if !item.Safety.NoDatabase.Known || !item.Safety.NoDatabase.Value {
		t.Error("Unarmed is no longer hidden from the item database")
	}
	if !item.Capabilities.Equipment.Known || !item.Capabilities.Equipment.Enabled ||
		item.Capabilities.Equipment.Rules == nil {
		t.Fatalf("Unarmed equipment capability = %+v, want an enabled capability",
			item.Capabilities.Equipment)
	}
	wantSlots := []schema.EquipmentSlot{schema.EquipmentSlotLeftHand, schema.EquipmentSlotRightHand}
	if len(item.Capabilities.Equipment.Rules.AllowedSlots) != len(wantSlots) {
		t.Fatalf("Unarmed slots = %v, want %v", item.Capabilities.Equipment.Rules.AllowedSlots, wantSlots)
	}
	for index, want := range wantSlots {
		if item.Capabilities.Equipment.Rules.AllowedSlots[index] != want {
			t.Errorf("Unarmed slot[%d] = %q, want %q",
				index, item.Capabilities.Equipment.Rules.AllowedSlots[index], want)
		}
	}
}
