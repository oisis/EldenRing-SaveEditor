package equipment

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// validateEquipmentSlotCompatibility is the single family-and-capability rule of
// this domain. The hand, armor, talisman, Quick Item and Pouch setters validate
// an assignment with it and GetEquipmentCandidates offers by it, so a candidate
// the picker shows is one the setter recognises and neither side can drift.
//
// Only the wording is per caller: capability names the group in the missing
// capability failure and slotPhrase names the position in the compatibility
// failure, so every setter keeps the exact message its contract documents.
//
// The Physick and spell rules are deliberately not expressed here. Physick adds
// confirmed game-ID conditions of its own and stays in physickTearGameID; a
// spell is not addressed by an equipment slot at all and stays in
// gamecatalog.ValidateSpellResource.
func validateEquipmentSlotCompatibility(
	resource schema.Resource,
	family schema.ItemFamily,
	slot schema.EquipmentSlot,
	capability string,
	slotPhrase string,
) error {
	item := resource.Item
	if item == nil {
		return fmt.Errorf("resource kind %q key %q has no item document",
			resource.Kind, resource.Key)
	}
	if !item.Family.Known || item.Family.Value != family {
		return fmt.Errorf("resource kind %q key %q has item family %q, want %q",
			resource.Kind, resource.Key, item.Family.Value, family)
	}
	equipment := item.Capabilities.Equipment
	if !equipment.Known || !equipment.Enabled || equipment.Rules == nil {
		return fmt.Errorf("resource kind %q key %q has no confirmed %s equipment capability",
			resource.Kind, resource.Key, capability)
	}
	for _, allowed := range equipment.Rules.AllowedSlots {
		if allowed == slot {
			return nil
		}
	}
	return fmt.Errorf("resource kind %q key %q cannot be equipped in %s",
		resource.Kind, resource.Key, slotPhrase)
}
