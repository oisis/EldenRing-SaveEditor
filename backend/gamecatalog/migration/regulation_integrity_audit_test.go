package migration

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestEquipLoadModifiersMatchReferencedRegulationEffects(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	context := generationContext{regulation: regulation}
	snapshot := collectLegacySnapshot()
	checked := 0
	for index := range snapshot.Items {
		item := snapshot.Items[index]
		if item.EquipLoad == nil {
			continue
		}
		family, _, err := itemFamily(item)
		if err != nil {
			t.Fatalf("item 0x%08X family: %v", item.ID, err)
		}
		identity, err := primaryRegulationForLegacyItem(item)
		if err != nil {
			t.Fatalf("item 0x%08X identity: %v", item.ID, err)
		}
		primary, exists, err := regulation.LookupFamilyRow(
			identity.Family,
			RegulationTableRolePrimary,
			identity.RowID,
		)
		if err != nil || !exists {
			t.Fatalf(
				"item 0x%08X primary row = %+v, %t, %v",
				item.ID,
				primary,
				exists,
				err,
			)
		}
		modifiers, err := context.buildModifiers(item, family, primary.Row, true)
		if err != nil {
			t.Fatalf("item 0x%08X buildModifiers: %v", item.ID, err)
		}
		if modifiers.EquipLoad == nil {
			t.Fatalf("item 0x%08X has no generated equip-load modifier", item.ID)
		}
		if modifiers.EquipLoad.EnduranceBonus.Value != item.EquipLoad.EnduranceBonus ||
			modifiers.EquipLoad.EquipLoadRate.Value != item.EquipLoad.EquipLoadRate {
			t.Fatalf(
				"item 0x%08X Regulation modifier = %+v, legacy = %+v",
				item.ID,
				modifiers.EquipLoad,
				item.EquipLoad,
			)
		}
		if modifiers.EquipLoad.EnduranceBonusSFV != nil ||
			modifiers.EquipLoad.EquipLoadRateSFV != nil {
			t.Fatalf("item 0x%08X has unexpected SFV modifier: %+v", item.ID, modifiers.EquipLoad)
		}
		checked++
	}
	if checked != 11 {
		t.Fatalf("verified equip-load modifiers = %d, want 11", checked)
	}
}

func TestEquipLoadModifierPreservesConflictingSaveForgeValue(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	context := generationContext{regulation: regulation}
	item := *findSeed(t, collectLegacySnapshot().Items, 0x20000406)
	item.EquipLoad = &equipLoadSeed{EnduranceBonus: 4, EquipLoadRate: 0.25}
	identity, err := primaryRegulationForLegacyItem(item)
	if err != nil {
		t.Fatal(err)
	}
	primary, exists, err := regulation.LookupFamilyRow(
		identity.Family,
		RegulationTableRolePrimary,
		identity.RowID,
	)
	if err != nil || !exists {
		t.Fatalf("primary row = %+v, %t, %v", primary, exists, err)
	}
	modifiers, err := context.buildModifiers(
		item,
		schema.ItemFamilyTalisman,
		primary.Row,
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if modifiers.EquipLoad.EnduranceBonus.Value != 0 ||
		modifiers.EquipLoad.EquipLoadRate.Value != 0.15 ||
		modifiers.EquipLoad.EnduranceBonusSFV == nil ||
		modifiers.EquipLoad.EnduranceBonusSFV.Value != 4 ||
		modifiers.EquipLoad.EquipLoadRateSFV == nil ||
		modifiers.EquipLoad.EquipLoadRateSFV.Value != 0.25 {
		t.Fatalf("equip-load Regulation/SFV split = %+v", modifiers.EquipLoad)
	}
}

func TestGestureSlotsUseExactGestureParamRows(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	rows, err := indexGestureRows(regulation)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := collectLegacySnapshot()
	context := generationContext{
		regulation:     regulation,
		gestureRows:    rows,
		gesturesByItem: make(map[uint32][]gestureSlotSeed),
	}
	for _, slot := range snapshot.GestureSlots {
		context.gesturesByItem[slot.ItemID] = append(
			context.gesturesByItem[slot.ItemID],
			slot,
		)
	}

	bow := *findSeed(t, snapshot.Items, 0x40002328)
	data, err := context.buildGestureData(bow, ParameterRow{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Slots) != 1 ||
		data.Slots[0].SlotID.Value != 1 ||
		data.Slots[0].SlotID.Provenance.Source != sourceIDByRegulationTable[RegulationTableGesture] ||
		data.Slots[0].ItemID.Value != bow.ID ||
		len(data.Slots[0].SourceRecords) != 1 {
		t.Fatalf("Bow GestureParam mapping = %+v", data.Slots)
	}

	cutContent := seed{ID: 0x40002354}
	data, err = context.buildGestureData(cutContent, ParameterRow{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Slots) != 1 ||
		data.Slots[0].SlotID.Value != 221 ||
		data.Slots[0].SlotID.Provenance.Source != sourceLegacyData ||
		len(data.Slots[0].SourceRecords) != 0 {
		t.Fatalf("unmapped cut-content gesture = %+v", data.Slots)
	}
}

func TestGestureSlotMismatchFailsClosed(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	rows, err := indexGestureRows(regulation)
	if err != nil {
		t.Fatal(err)
	}
	item := seed{ID: 0x40002328}
	context := generationContext{
		regulation:  regulation,
		gestureRows: rows,
		gesturesByItem: map[uint32][]gestureSlotSeed{
			item.ID: {{SlotID: 2, ItemID: item.ID, Name: "Bow", Category: "Greetings"}},
		},
	}
	_, err = context.buildGestureData(item, ParameterRow{}, false)
	if err == nil || !strings.Contains(err.Error(), "0 matching GestureParam rows") {
		t.Fatalf("buildGestureData error = %v", err)
	}
}

func TestWeaponLegacyIdentityMismatchFailsClosed(t *testing.T) {
	context := generationContext{}
	data := &schema.WeaponData{}
	item := seed{
		ID:            0x000F4240,
		HasLegacyItem: true,
		WeaponStats: &weaponStatsSeed{
			ItemID:      0x000F4241,
			SourceRowID: 1_000_000,
		},
	}
	err := context.attachWeaponSaveForgeValues(
		data,
		item,
		regulationWeaponCoreStats{},
		0,
		1_000_000,
	)
	if err == nil || !strings.Contains(err.Error(), "does not match item") {
		t.Fatalf("ItemID mismatch error = %v", err)
	}

	item.WeaponStats.ItemID = item.ID
	item.WeaponStats.SourceRowID = 1_000_001
	err = context.attachWeaponSaveForgeValues(
		data,
		item,
		regulationWeaponCoreStats{},
		0,
		1_000_000,
	)
	if err == nil || !strings.Contains(err.Error(), "does not match EquipParamWeapon row") {
		t.Fatalf("SourceRowID mismatch error = %v", err)
	}
}
