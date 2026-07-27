package application

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/editor"
	"github.com/oisis/EldenRing-SaveForge/backend/templates"
)

func TestResolveEquipmentWrites_MatchesOwnedWeapon(t *testing.T) {
	items := []editor.EditableItem{{BaseItemID: 0x100000, OriginalHandle: 0x80000010, IsWeapon: true}}
	sec := &templates.EquipmentSection{WeaponRightHand1: &templates.EquipmentItemRef{BaseItemID: 0x100000}}
	writes, warnings, err := resolveEquipmentWritesFromItems(items, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(writes) != 1 || writes[0] != (core.EquipmentWrite{Slot: core.EquipSlotRightHandArmament1, Handle: 0x80000010}) {
		t.Fatalf("writes = %+v", writes)
	}
}

func TestResolveEquipmentWrites_SkipsTalisman(t *testing.T) {
	sec := &templates.EquipmentSection{Talisman1: &templates.EquipmentItemRef{BaseItemID: 0x200003E8}}
	writes, warnings, err := resolveEquipmentWritesFromItems(nil, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(writes) != 0 {
		t.Fatalf("talisman must not produce writes: %+v", writes)
	}
	if len(warnings) != 1 || warnings[0].Code != templates.IssueCodeEquipmentSlotInvalid {
		t.Fatalf("warnings = %+v", warnings)
	}
}

func TestResolveEquipmentWrites_ExplicitClear(t *testing.T) {
	sec := &templates.EquipmentSection{ArmorHead: &templates.EquipmentItemRef{BaseItemID: 0}}
	writes, warnings, err := resolveEquipmentWritesFromItems(nil, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(writes) != 1 || writes[0] != (core.EquipmentWrite{Slot: core.EquipSlotHead, Handle: 0}) {
		t.Fatalf("writes = %+v", writes)
	}
}

func TestResolveEquipmentWrites_AmbiguousMatchWarns(t *testing.T) {
	items := []editor.EditableItem{
		{BaseItemID: 0x100000, OriginalHandle: 0x80000010, IsWeapon: true},
		{BaseItemID: 0x100000, OriginalHandle: 0x80000011, IsWeapon: true},
	}
	sec := &templates.EquipmentSection{WeaponRightHand1: &templates.EquipmentItemRef{BaseItemID: 0x100000}}
	writes, warnings, err := resolveEquipmentWritesFromItems(items, &templates.SectionSelection{All: true}, sec)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(writes) != 1 || writes[0].Handle != 0x80000010 {
		t.Fatalf("writes = %+v", writes)
	}
	if len(warnings) != 1 || warnings[0].Code != templates.IssueCodeEquipmentItemAmbiguous {
		t.Fatalf("warnings = %+v", warnings)
	}
}
