package migration

import (
	"slices"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func weaponStatusFact(
	value int32,
	field string,
	warnings []string,
) schema.Fact[int32] {
	if slices.Contains(warnings, "status-deferred") {
		return unknownCatalogFact[int32](
			"legacy WeaponStatsV1." + field +
				" is zero because status derivation is deferred",
		)
	}
	return knownLegacyFact(
		value,
		"copied from legacy WeaponStatsV1."+field,
	)
}
