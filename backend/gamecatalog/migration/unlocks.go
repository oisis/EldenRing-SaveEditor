package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func buildUnlocks(values []unlockSeed) []schema.ItemUnlock {
	result := make([]schema.ItemUnlock, len(values))
	for index, value := range values {
		result[index] = schema.ItemUnlock{
			Kind:        knownLegacyFact(value.Kind, "copied from legacy item-to-event-flag mapping"),
			EventFlagID: knownLegacyFact(value.FlagID, "copied from legacy item-to-event-flag mapping"),
			Name:        optionalLegacyString(value.Name, "copied from legacy unlock metadata"),
			Category:    optionalLegacyString(value.Category, "copied from legacy unlock metadata"),
		}
	}
	return result
}
