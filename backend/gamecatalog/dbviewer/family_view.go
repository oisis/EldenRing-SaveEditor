package dbviewer

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (server *Server) familyFacts(item *schema.ItemDocument) []factView {
	if item.Weapon != nil {
		weapon := item.Weapon
		return []factView{
			server.fact("Source row ID", weapon.SourceRowID.Known, weapon.SourceRowID.Value, weapon.SourceRowID.Provenance),
			server.fact("Weapon type ID", weapon.WeaponTypeID.Known, weapon.WeaponTypeID.Value, weapon.WeaponTypeID.Provenance),
			server.fact("Weight", weapon.Weight.Known, weapon.Weight.Value, weapon.Weight.Provenance),
			server.fact("Physical attack", weapon.AttackPhysical.Known, weapon.AttackPhysical.Value, weapon.AttackPhysical.Provenance),
			server.fact("Required strength", weapon.RequiredStrength.Known, weapon.RequiredStrength.Value, weapon.RequiredStrength.Provenance),
			server.fact("Required dexterity", weapon.RequiredDexterity.Known, weapon.RequiredDexterity.Value, weapon.RequiredDexterity.Provenance),
			server.fact("Critical", weapon.Critical.Known, weapon.Critical.Value, weapon.Critical.Provenance),
		}
	}
	if item.AshOfWar != nil {
		ash := item.AshOfWar
		return []factView{
			server.fact("Source row ID", ash.SourceRowID.Known, ash.SourceRowID.Value, ash.SourceRowID.Provenance),
			server.fact(
				"Compatibility mask",
				ash.CompatibilityMask.Known,
				fmt.Sprintf("0x%X", ash.CompatibilityMask.Value),
				ash.CompatibilityMask.Provenance,
			),
		}
	}
	return nil
}

type variantView struct {
	GameID      string
	Affinity    string
	SourceRowID uint32
}

func (server *Server) variantViews(item *schema.ItemDocument) []variantView {
	variants := make([]variantView, 0, len(item.Variants))
	for _, variant := range item.Variants {
		variants = append(variants, variantView{
			GameID:      formatGameID(variant.GameID.Value),
			Affinity:    string(variant.Affinity.Value),
			SourceRowID: variant.SourceRowID.Value,
		})
	}
	return variants
}
