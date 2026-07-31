package migration

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func resolvedVariantWeapon(
	t *testing.T,
	item *schema.ItemDocument,
	variant schema.ItemVariant,
) *schema.WeaponData {
	t.Helper()
	if item == nil || item.Weapon == nil || variant.Data.Weapon == nil {
		t.Fatal("weapon variant or canonical weapon data is missing")
	}
	resolved := *variant.Data.Weapon
	return &resolved
}
