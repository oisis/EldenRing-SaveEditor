package templates

import (
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/editor"
)

// gateEquipmentSource / gateItemsSource are complete, non-nil sources so the
// BuildV2Template section-combination gate is the check under test — not the
// earlier "source was not provided" guard.
func gateEquipmentSource() *EquipmentSection {
	return &EquipmentSection{WeaponRightHand1: &EquipmentItemRef{BaseItemID: 0x100000}}
}

func gateItemsSource() *ItemsLayoutSource {
	return &ItemsLayoutSource{
		InventoryItems: []editor.EditableItem{
			editableWeapon("uid-w-0", 0x001E8480, "Longsword", editor.ContainerInventory, 0, 0, 25),
		},
	}
}

// Equipment applies against items already present in the target inventory, so
// it cannot be combined with the sections that add / reorder those items. The
// builder rejects the combination even for a direct caller.
func TestBuildV2Template_Gate_EquipmentWithItems_Rejected(t *testing.T) {
	_, err := BuildV2Template(ExportV2Options{
		Now:         fixedNow(),
		Equipment:   gateEquipmentSource(),
		ItemsSource: gateItemsSource(),
		Selection: &TemplateSelection{
			Equipment: &SectionSelection{All: true},
			Items:     &SectionSelection{All: true},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "equipment cannot be combined") {
		t.Errorf("expected equipment/items section-conflict error, got %v", err)
	}
}

func TestBuildV2Template_Gate_EquipmentWithInventoryLayout_Rejected(t *testing.T) {
	_, err := BuildV2Template(ExportV2Options{
		Now:         fixedNow(),
		Equipment:   gateEquipmentSource(),
		ItemsSource: gateItemsSource(),
		Selection: &TemplateSelection{
			Equipment:       &SectionSelection{All: true},
			InventoryLayout: &SectionSelection{All: true},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "equipment cannot be combined") {
		t.Errorf("expected equipment/inventoryLayout section-conflict error, got %v", err)
	}
}

func TestBuildV2Template_Gate_EquipmentWithStorageLayout_Rejected(t *testing.T) {
	_, err := BuildV2Template(ExportV2Options{
		Now:         fixedNow(),
		Equipment:   gateEquipmentSource(),
		ItemsSource: gateItemsSource(),
		Selection: &TemplateSelection{
			Equipment:     &SectionSelection{All: true},
			StorageLayout: &SectionSelection{All: true},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "equipment cannot be combined") {
		t.Errorf("expected equipment/storageLayout section-conflict error, got %v", err)
	}
}

// Spells touch a disjoint section, so equipment + spells is allowed and both
// sections land.
func TestBuildV2Template_Gate_EquipmentWithSpells_Allowed(t *testing.T) {
	tpl, err := BuildV2Template(ExportV2Options{
		Now:               fixedNow(),
		Equipment:         gateEquipmentSource(),
		EquippedSpellsRaw: emptySpellLoadout(),
		Selection: &TemplateSelection{
			Equipment: &SectionSelection{All: true},
			Spells:    &SectionSelection{All: true},
		},
	})
	if err != nil {
		t.Fatalf("equipment + spells must be allowed, got %v", err)
	}
	if tpl.Sections.Equipment == nil {
		t.Error("equipment section missing in equipment+spells template")
	}
	if tpl.Sections.Spells == nil {
		t.Error("spells section missing in equipment+spells template")
	}
}
