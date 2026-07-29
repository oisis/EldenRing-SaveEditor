package application

import (
	"bytes"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/templates"
)

// Phase 7b.1 (P3 follow-up) — the equipment ↔ Inventory mutual-exclusion
// rule is defended at the direct Apply boundary for the whole conflicting
// family (inventory.workspace / items / inventoryLayout / storageLayout),
// not just inventory.workspace. These table-driven tests prove the guard
// rejects each combo with zero slot and zero workspace mutation — the
// boundary defends even though a session is open and valid.

// equipmentComboSection returns a valid v2 template combining an equipment
// selection with one conflicting Inventory-mutating section. The template
// validates so the combo guard (not a structural error) is the surfacing
// failure. The equipment slot never needs to resolve: the guard fires
// before the equipment resolver runs.
func equipmentComboSection(name string) *templates.BuildTemplate {
	tpl := &templates.BuildTemplate{
		Schema:     templates.SchemaKey,
		Version:    2,
		AppVersion: "test",
		CreatedAt:  "2026-06-02T00:00:00Z",
		Selection:  &templates.TemplateSelection{Equipment: &templates.SectionSelection{All: true}},
		Sections: templates.TemplateSections{
			Equipment: &templates.EquipmentSection{
				WeaponRightHand1: &templates.EquipmentItemRef{BaseItemID: 0x100000},
			},
		},
	}
	meleeEntry := templates.TemplateItemEntryV2{
		EntryID:      "w1",
		ItemID:       0x003D9700,
		Name:         "Greatsword",
		Category:     templates.ItemCategoryMeleeArmaments,
		Quantity:     1,
		Location:     templates.ItemLocationInventory,
		UpgradeKind:  templates.UpgradeKindStandard,
		UpgradeLevel: u8ptr(25),
	}
	switch name {
	case "inventory.workspace":
		tpl.Selection.InventoryWorkspace = &templates.SectionSelection{All: true}
		tpl.Sections.InventoryWorkspace = &templates.InventoryWorkspaceSection{
			InventoryItems: []templates.TemplateItem{},
			StorageItems:   []templates.TemplateItem{},
		}
	case "items":
		tpl.Selection.Items = &templates.SectionSelection{All: true}
		tpl.Sections.Items = &templates.ItemsSection{Entries: []templates.TemplateItemEntryV2{meleeEntry}}
	case "inventoryLayout":
		// items present (for the layout to reference) but NOT selected —
		// isolates inventoryLayout as the sole Inventory conflict.
		tpl.Selection.InventoryLayout = &templates.SectionSelection{All: true}
		tpl.Sections.Items = &templates.ItemsSection{Entries: []templates.TemplateItemEntryV2{meleeEntry}}
		tpl.Sections.InventoryLayout = &templates.InventoryLayoutSection{
			Entries: []templates.LayoutEntry{{EntryRef: "w1", Position: 0}},
		}
	case "storageLayout":
		tpl.Selection.StorageLayout = &templates.SectionSelection{All: true}
		tpl.Sections.Items = &templates.ItemsSection{Entries: []templates.TemplateItemEntryV2{meleeEntry}}
		tpl.Sections.StorageLayout = &templates.StorageLayoutSection{
			Entries: []templates.LayoutEntry{{EntryRef: "w1", Position: 0}},
		}
	}
	return tpl
}

func TestApplyBuildTemplateV2_EquipmentInventoryCombo_RejectedFamily(t *testing.T) {
	for _, name := range []string{"inventory.workspace", "items", "inventoryLayout", "storageLayout"} {
		t.Run(name, func(t *testing.T) {
			// A valid, open session exists — the guard must still reject
			// before touching it, proving a boundary defense rather than
			// mere propagation from a separate Preview call.
			app, sessionID := freshItemsFixture(t)

			preSlot := append([]byte(nil), app.save.Slots[0].Data...)
			preWS, err := app.GetInventoryEditSession(sessionID)
			if err != nil {
				t.Fatalf("GetInventoryEditSession: %v", err)
			}
			preInv, preSto, preDirty := len(preWS.InventoryItems), len(preWS.StorageItems), preWS.Dirty

			jsonText := mustMarshalTpl(t, equipmentComboSection(name))
			res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{SessionID: sessionID})
			if err != nil {
				t.Fatalf("apply: unexpected Go error %v", err)
			}

			if res.Applied {
				t.Fatalf("Applied=true for equipment+%s, want false", name)
			}
			if res.Preview.OK {
				t.Errorf("Preview.OK=true for equipment+%s, want false", name)
			}
			if !hasIssue(res.Preview.Errors, templates.IssueCodeEquipmentInventoryComboUnsupported) {
				t.Errorf("missing %s in errors: %+v", templates.IssueCodeEquipmentInventoryComboUnsupported, res.Preview.Errors)
			}

			// Byte-identical slot — no mutation.
			if !bytes.Equal(preSlot, app.save.Slots[0].Data) {
				t.Errorf("slot.Data mutated for equipment+%s (pre=%d post=%d bytes)",
					name, len(preSlot), len(app.save.Slots[0].Data))
			}

			// Untouched workspace + Dirty state.
			postWS, err := app.GetInventoryEditSession(sessionID)
			if err != nil {
				t.Fatalf("GetInventoryEditSession (post): %v", err)
			}
			if postWS.Dirty != preDirty {
				t.Errorf("workspace Dirty changed pre=%v post=%v", preDirty, postWS.Dirty)
			}
			if len(postWS.InventoryItems) != preInv || len(postWS.StorageItems) != preSto {
				t.Errorf("workspace items changed inv %d->%d, sto %d->%d",
					preInv, len(postWS.InventoryItems), preSto, len(postWS.StorageItems))
			}
		})
	}
}
