package migration

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestEquipmentCapabilityUsesExplicitRegulationFlags(t *testing.T) {
	weaponRow := equipmentFlagsRow(map[string]string{
		"rightHandEquipable": "0",
		"leftHandEquipable":  "0",
		"bothHandEquipable":  "0",
		"arrowSlotEquipable": "1",
		"boltSlotEquipable":  "0",
	})
	weapon, err := buildEquipmentCapability(schema.ItemFamilyWeapon, weaponRow, true)
	if err != nil {
		t.Fatalf("weapon capability: %v", err)
	}
	if !weapon.Known || !weapon.Enabled || weapon.Rules == nil {
		t.Fatalf("weapon capability = %#v", weapon)
	}
	if want := []schema.EquipmentSlot{schema.EquipmentSlotArrow}; !reflect.DeepEqual(weapon.Rules.AllowedSlots, want) {
		t.Fatalf("weapon slots = %#v, want %#v", weapon.Rules.AllowedSlots, want)
	}
	if weapon.Provenance.Source != sourceIDByRegulationTable[RegulationTableWeapon] {
		t.Fatalf("weapon source = %q", weapon.Provenance.Source)
	}

	armorRow := equipmentFlagsRow(map[string]string{
		"headEquip": "0",
		"bodyEquip": "0",
		"armEquip":  "0",
		"legEquip":  "1",
	})
	armor, err := buildEquipmentCapability(schema.ItemFamilyArmor, armorRow, true)
	if err != nil {
		t.Fatalf("armor capability: %v", err)
	}
	if !armor.Known || !armor.Enabled || armor.Rules == nil {
		t.Fatalf("armor capability = %#v", armor)
	}
	if want := []schema.EquipmentSlot{schema.EquipmentSlotLegs}; !reflect.DeepEqual(armor.Rules.AllowedSlots, want) {
		t.Fatalf("armor slots = %#v, want %#v", armor.Rules.AllowedSlots, want)
	}
	if armor.Provenance.Source != sourceIDByRegulationTable[RegulationTableProtector] {
		t.Fatalf("armor source = %q", armor.Provenance.Source)
	}
}

func TestEquipmentCapabilityClassifiesGoodsFamilies(t *testing.T) {
	tests := []struct {
		name    string
		family  schema.ItemFamily
		row     ParameterRow
		enabled bool
		slots   []schema.EquipmentSlot
	}{
		{
			name:   "Spirit Ash uses quick item and pouch",
			family: schema.ItemFamilySpiritAsh,
			row: equipmentFlagsRow(map[string]string{
				"goodsType": "8",
				"isEquip":   "1",
			}),
			enabled: true,
			slots: []schema.EquipmentSlot{
				schema.EquipmentSlotQuickItem,
				schema.EquipmentSlotPouch,
			},
		},
		{
			name:   "Wondrous Physick tear uses Physick slot",
			family: schema.ItemFamilyGoods,
			row: equipmentFlagsRow(map[string]string{
				"goodsType": "10",
				"isEquip":   "0",
			}),
			enabled: true,
			slots:   []schema.EquipmentSlot{schema.EquipmentSlotPhysick},
		},
		{
			name:   "non-equipable goods are disabled",
			family: schema.ItemFamilyGoods,
			row: equipmentFlagsRow(map[string]string{
				"goodsType": "1",
				"isEquip":   "0",
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capability, err := buildEquipmentCapability(test.family, test.row, true)
			if err != nil {
				t.Fatalf("buildEquipmentCapability: %v", err)
			}
			if !capability.Known || capability.Enabled != test.enabled {
				t.Fatalf("capability = %#v", capability)
			}
			if test.enabled {
				if capability.Rules == nil ||
					!reflect.DeepEqual(capability.Rules.AllowedSlots, test.slots) {
					t.Fatalf("slots = %#v, want %#v", capability.Rules, test.slots)
				}
			} else if capability.Rules != nil {
				t.Fatalf("disabled capability has rules %#v", capability.Rules)
			}
		})
	}
}

func TestEquipmentCapabilityDisablesNonEquipmentFamilies(t *testing.T) {
	for _, family := range []schema.ItemFamily{
		schema.ItemFamilyGesture,
		schema.ItemFamilyAshOfWar,
	} {
		capability, err := buildEquipmentCapability(family, ParameterRow{}, false)
		if err != nil {
			t.Fatalf("%s capability: %v", family, err)
		}
		if !capability.Known || capability.Enabled || capability.Rules != nil {
			t.Fatalf("%s capability = %#v", family, capability)
		}
	}
}

func equipmentFlagsRow(values map[string]string) ParameterRow {
	fields := make([]ParameterField, 0, len(values))
	for _, name := range []string{
		"rightHandEquipable",
		"leftHandEquipable",
		"bothHandEquipable",
		"arrowSlotEquipable",
		"boltSlotEquipable",
		"headEquip",
		"bodyEquip",
		"armEquip",
		"legEquip",
		"goodsType",
		"isEquip",
	} {
		if value, exists := values[name]; exists {
			fields = append(fields, ParameterField{Name: name, RawValue: value})
		}
	}
	return ParameterRow{RowID: 123, Fields: fields}
}
