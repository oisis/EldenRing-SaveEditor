package prototype

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func determinationResource() schema.Resource {
	regulationGem := provenance(sourceRegulationGem, "normalized from EquipParamGem row 60000")
	regulationText := provenance(sourceRegulationText, "normalized from GemName and GemCaption FMG entries 60000")
	legacy := provenance(sourceLegacyData, "copied as data from the legacy catalog")
	research := provenance(sourceSaveResearch, "normalized from verified save-format research")
	unknown := provenance(sourceUnknown, "not established in the prototype")

	return schema.Resource{
		ID:    2,
		Key:   "item:8000EA60",
		Kind:  schema.ResourceKindItem,
		Label: known("Ash of War: Determination", regulationText),
		Item: &schema.ItemDocument{
			GameID:      known(DeterminationGameID, regulationGem),
			Family:      known(schema.ItemFamilyAshOfWar, regulationGem),
			Subcategory: unknownFact[string](unknown),
			Presentation: schema.ItemPresentation{
				CanonicalName: known("Ash of War: Determination", regulationText),
				Description: known(
					"This Ash of War grants an armament the Quality affinity and the following skill:\n\n"+
						"\"Determination: A knightly skill. Hold the flat of the armament to your face and pledge your resolve, powering up your next attack.\"\n\n"+
						"Usable on all melee armaments.",
					regulationText,
				),
				IconPath: known("items/ashes_of_war/determination.png", legacy),
			},
			Storage: schema.ItemStorage{
				RecordMode:   known(schema.RecordModeSeparateInstances, research),
				MaxInventory: unknownFact[uint32](unknown),
				MaxStorage:   unknownFact[uint32](unknown),
			},
			Capabilities: schema.ItemCapabilities{
				Upgrade:       disabled[schema.UpgradeRules](regulationGem),
				Infusion:      disabled[schema.InfusionRules](regulationGem),
				AshOfWarMount: disabled[schema.AshOfWarMountRules](research),
				Stack:         disabled[schema.StackRules](research),
				Equipment:     disabled[schema.EquipmentRules](research),
			},
			Safety: schema.ItemSafety{
				CutContent: unknownFact[bool](unknown),
				BanRisk:    unknownFact[bool](unknown),
			},
			AshOfWar: &schema.AshOfWarData{
				SourceRowID:       known(uint32(60000), regulationGem),
				CompatibilityMask: known(uint64(0xF4000FEFFFF), regulationGem),
			},
		},
	}
}
