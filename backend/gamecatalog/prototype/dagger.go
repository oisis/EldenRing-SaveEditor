package prototype

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func daggerResource() schema.Resource {
	regulationWeapon := provenance(sourceRegulationWeapon, "normalized from EquipParamWeapon rows 1000000-1001200")
	regulationText := provenance(sourceRegulationText, "normalized from WeaponName and WeaponCaption FMG entries 1000000")
	legacy := provenance(sourceLegacyData, "copied as data from the legacy catalog")
	research := provenance(sourceSaveResearch, "normalized from verified save-format research")
	unknown := provenance(sourceUnknown, "not established in the prototype")

	upgradeRules := schema.UpgradeRules{
		Model:    schema.UpgradeModelStandard,
		MaxLevel: 25,
	}
	infusionRules := schema.InfusionRules{
		AllowedAffinities: []schema.Affinity{
			schema.AffinityStandard,
			schema.AffinityHeavy,
			schema.AffinityKeen,
			schema.AffinityQuality,
			schema.AffinityFire,
			schema.AffinityFlameArt,
			schema.AffinityLightning,
			schema.AffinitySacred,
			schema.AffinityMagic,
			schema.AffinityCold,
			schema.AffinityPoison,
			schema.AffinityBlood,
			schema.AffinityOccult,
		},
	}
	mountRules := schema.AshOfWarMountRules{
		Mode:             schema.AshOfWarMountModeCustom,
		WeaponType:       "dagger",
		CompatibilityBit: 0,
	}
	equipmentRules := schema.EquipmentRules{
		AllowedSlots: []schema.EquipmentSlot{
			schema.EquipmentSlotLeftHand,
			schema.EquipmentSlotRightHand,
		},
	}

	return schema.Resource{
		ID:    1,
		Key:   "item:000F4240",
		Kind:  schema.ResourceKindItem,
		Label: known("Dagger", regulationText),
		Item: &schema.ItemDocument{
			GameID:      known(DaggerGameID, regulationWeapon),
			Family:      known(schema.ItemFamilyWeapon, regulationWeapon),
			Subcategory: known("dagger", research),
			Presentation: schema.ItemPresentation{
				CanonicalName: known("Dagger", regulationText),
				Description: known(
					"A standard dagger with a straight blade.\n\n"+
						"Though modest in reach and capacity for harm, this weapon is light enough to jab in rapid succession and delivers devastating critical hits.",
					regulationText,
				),
				IconPath: known("items/melee_armaments/dagger.png", legacy),
			},
			Storage: schema.ItemStorage{
				RecordMode:   known(schema.RecordModeSeparateInstances, research),
				MaxInventory: unknownFact[uint32](unknown),
				MaxStorage:   unknownFact[uint32](unknown),
			},
			Capabilities: schema.ItemCapabilities{
				Upgrade:       enabled(upgradeRules, regulationWeapon),
				Infusion:      enabled(infusionRules, regulationWeapon),
				AshOfWarMount: enabled(mountRules, regulationWeapon),
				Stack:         disabled[schema.StackRules](research),
				Equipment:     enabled(equipmentRules, regulationWeapon),
			},
			Safety: schema.ItemSafety{
				CutContent: unknownFact[bool](unknown),
				BanRisk:    unknownFact[bool](unknown),
			},
			Variants: daggerVariants(regulationWeapon),
			Weapon: &schema.WeaponData{
				SourceRowID:       known(uint32(1000000), regulationWeapon),
				WeaponTypeID:      known(uint16(1), regulationWeapon),
				Weight:            known(float32(1.5), regulationWeapon),
				AttackPhysical:    known(uint32(74), regulationWeapon),
				RequiredStrength:  known(uint16(5), regulationWeapon),
				RequiredDexterity: known(uint16(9), regulationWeapon),
				Critical:          known(uint16(130), regulationWeapon),
			},
		},
	}
}
