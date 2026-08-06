package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type builtDocumentData struct {
	Category                schema.Fact[string]
	Subcategory             schema.Fact[string]
	Presentation            schema.ItemPresentation
	Storage                 schema.ItemStorage
	Capabilities            schema.ItemCapabilities
	Safety                  schema.ItemSafety
	Acquisition             schema.ItemAcquisition
	Modifiers               schema.ItemModifiers
	Links                   schema.ItemLinks
	Unlocks                 []schema.ItemUnlock
	RelatedTechnicalRecords []schema.RelatedTechnicalRecord
	Weapon                  *schema.WeaponData
	SpiritAsh               *schema.SpiritAshData
}

func (context *generationContext) buildDocumentData(
	item seed,
	family schema.ItemFamily,
	row ParameterRow,
	hasPrimaryRow bool,
) (builtDocumentData, error) {
	return context.buildDocumentDataWithCapabilities(
		item,
		family,
		row,
		hasPrimaryRow,
		nil,
	)
}

func (context *generationContext) buildDocumentDataWithCapabilities(
	item seed,
	family schema.ItemFamily,
	row ParameterRow,
	hasPrimaryRow bool,
	capabilitiesOverride *schema.ItemCapabilities,
) (builtDocumentData, error) {
	var capabilities schema.ItemCapabilities
	var err error
	if capabilitiesOverride == nil {
		capabilities, err = context.buildCapabilities(
			item,
			family,
			row,
			hasPrimaryRow,
		)
		if err != nil {
			return builtDocumentData{}, err
		}
	} else {
		capabilities = *capabilitiesOverride
	}
	storage, err := context.buildStorage(item, family, row, hasPrimaryRow)
	if err != nil {
		return builtDocumentData{}, err
	}
	relatedTechnicalRecords, err := context.buildRelatedTechnicalRecords(
		family,
		row,
		hasPrimaryRow,
	)
	if err != nil {
		return builtDocumentData{}, err
	}
	modifiers, err := context.buildModifiers(item, family, row, hasPrimaryRow)
	if err != nil {
		return builtDocumentData{}, err
	}
	data := builtDocumentData{
		Category:                itemCategoryFact(item),
		Subcategory:             itemSubcategoryFact(item),
		Presentation:            buildPresentation(item),
		Storage:                 storage,
		Capabilities:            capabilities,
		Safety:                  context.buildSafety(item),
		Acquisition:             buildAcquisition(item, requiredContainerIsNotApplicable(item, family)),
		Modifiers:               modifiers,
		Links:                   buildLinks(item.Links, family == schema.ItemFamilySpiritAsh),
		Unlocks:                 buildUnlocks(item.Unlocks),
		RelatedTechnicalRecords: relatedTechnicalRecords,
	}
	switch family {
	case schema.ItemFamilyWeapon:
		if !hasPrimaryRow {
			return builtDocumentData{}, fmt.Errorf(
				"weapon item 0x%08X has no primary regulation row",
				item.ID,
			)
		}
		data.Weapon, err = context.buildWeaponData(item, row)
	case schema.ItemFamilySpiritAsh:
		if !hasPrimaryRow {
			return builtDocumentData{}, fmt.Errorf(
				"spirit ash item 0x%08X has no primary regulation row",
				item.ID,
			)
		}
		data.SpiritAsh, err = buildSpiritAshData(item, row)
	}
	return data, err
}

func itemSubcategoryFact(item seed) schema.Fact[string] {
	method := "copied from legacy ItemData.SubCategory"
	if item.RegulationOnlyVariant {
		method = "copied from the canonical legacy ItemData subcategory for a Regulation-only variant"
	}
	return optionalLegacyString(item.Subcategory, method)
}
