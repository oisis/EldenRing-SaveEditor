package migration

import (
	"fmt"
	"path"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) buildResource(
	group legacyItemGroup,
) (schema.Resource, error) {
	item := group.Canonical
	family, table, err := itemFamily(item)
	if err != nil {
		return schema.Resource{}, err
	}
	sourceRecords, err := context.sourceRecordsForItem(item)
	if err != nil {
		return schema.Resource{}, err
	}
	identity, err := primaryRegulationForLegacyItem(item)
	if err != nil {
		return schema.Resource{}, err
	}
	primary, primaryExists, err := context.regulation.LookupFamilyRow(
		identity.Family,
		RegulationTableRolePrimary,
		identity.RowID,
	)
	if err != nil {
		return schema.Resource{}, err
	}
	provenanceTable := table
	provenanceRowID := identity.RowID
	if !primaryExists {
		gestureRows := context.gestureRows[item.ID&0x0FFFFFFF]
		if len(gestureRows) == 0 {
			provenanceTable = ""
		} else {
			provenanceTable = RegulationTableGesture
			provenanceRowID = gestureRows[0].RowID
		}
	}
	aliases, err := context.buildAliases(item.ID)
	if err != nil {
		return schema.Resource{}, err
	}
	data, err := context.buildDocumentData(
		item,
		family,
		primary.Row,
		primaryExists,
	)
	if err != nil {
		return schema.Resource{}, err
	}
	if family == schema.ItemFamilySpiritAsh {
		upgradeRecords, upgradeErr := context.spiritAshUpgradeSourceRecords(primary.Row)
		if upgradeErr != nil {
			return schema.Resource{}, upgradeErr
		}
		sourceRecords, upgradeErr = mergeParameterRecords(sourceRecords, upgradeRecords)
		if upgradeErr != nil {
			return schema.Resource{}, upgradeErr
		}
	}
	variants, err := context.buildVariants(group.Variants, data)
	if err != nil {
		return schema.Resource{}, err
	}
	document := schema.ItemDocument{
		GameID:                  itemIdentityFact(item.ID, provenanceTable, provenanceRowID),
		Family:                  itemFamilyFact(family, provenanceTable, provenanceRowID),
		Category:                data.Category,
		Subcategory:             data.Subcategory,
		Presentation:            data.Presentation,
		Storage:                 data.Storage,
		Capabilities:            data.Capabilities,
		Safety:                  data.Safety,
		Acquisition:             data.Acquisition,
		Modifiers:               data.Modifiers,
		Links:                   data.Links,
		Variants:                variants,
		Aliases:                 aliases,
		Unlocks:                 data.Unlocks,
		RelatedTechnicalRecords: data.RelatedTechnicalRecords,
		SourceRecords:           sourceRecords,
		Weapon:                  data.Weapon,
		SpiritAsh:               data.SpiritAsh,
	}
	if family != schema.ItemFamilyWeapon && family != schema.ItemFamilySpiritAsh {
		if err := context.attachFamilyData(&document, item, family); err != nil {
			return schema.Resource{}, err
		}
	}
	document.SourceRecords = enrichParameterRecordFields(document.SourceRecords, document)
	return schema.Resource{
		Key:  fmt.Sprintf("%08X", item.ID),
		Kind: schema.ResourceKindItem,
		Item: &document,
	}, nil
}

func itemIdentityFact(
	id uint32,
	table RegulationTableName,
	sourceRowID uint32,
) schema.Fact[uint32] {
	if table == "" {
		return schema.Fact[uint32]{
			Known: true,
			Value: id,
			Provenance: schema.Provenance{
				Source: sourceLegacyUnknown,
				Method: "copied from the legacy gesture record; primary regulation row is absent",
			},
		}
	}
	field := "Row ID plus item-family prefix"
	if table == RegulationTableGesture {
		field = "itemId plus goods item prefix"
	}
	return knownRegulationFact(id, table, field, sourceRowID)
}

func itemFamilyFact(
	family schema.ItemFamily,
	table RegulationTableName,
	rowID uint32,
) schema.Fact[schema.ItemFamily] {
	if table == "" {
		return schema.Fact[schema.ItemFamily]{
			Known: true,
			Value: family,
			Provenance: schema.Provenance{
				Source: sourceLegacyUnknown,
				Method: "classified from the legacy gesture record without a primary regulation row",
			},
		}
	}
	if table == RegulationTableGesture {
		return knownRegulationDerivedFact(
			family,
			table,
			"classified from the exact GestureParam itemId relationship",
			rowID,
			"itemId",
		)
	}
	return knownRegulationDerivedFact(
		family,
		table,
		"classified from the primary parameter table",
		rowID,
		"Row ID",
	)
}

func itemCategoryFact(item seed) schema.Fact[string] {
	if item.RegulationOnlyVariant {
		return knownLegacyFact(
			item.Category,
			"copied from the canonical legacy ItemData category for a Regulation-only variant",
		)
	}
	if !item.HasLegacyItem {
		return knownLegacyFact(item.Category, "normalized from legacy AllGestures")
	}
	return knownLegacyFact(item.Category, "copied from legacy ItemData.Category")
}

func (context *generationContext) buildPresentation(
	item seed,
	family schema.ItemFamily,
	safety schema.ItemSafety,
	isVariant bool,
) (schema.ItemPresentation, error) {
	result := emptyPresentation()
	if item.Text != nil {
		result.Caption = optionalLegacyString(item.Text.Caption, "copied from legacy ItemTexts.Caption")
		result.Description = optionalLegacyString(item.Text.Description, "copied from legacy ItemTexts.Description")
		result.Location = optionalLegacyString(item.Text.Location, "copied from legacy ItemTexts.Location")
		result.TextMetadata = schema.ItemTextMetadata{
			CaptionSource:     optionalLegacyString(item.Text.CaptionSource, "copied from legacy ItemTexts.CaptionSource"),
			DescriptionSource: optionalLegacyString(item.Text.DescriptionSource, "copied from legacy ItemTexts.DescriptionSource"),
			LocationSource:    optionalLegacyString(item.Text.LocationSource, "copied from legacy ItemTexts.LocationSource"),
			DLCSource:         optionalLegacyString(item.Text.DLCSource, "copied from legacy ItemTexts.DLCSource"),
			Notes:             optionalLegacyString(item.Text.Notes, "copied from legacy ItemTexts.Notes"),
		}
	} else if item.Description != nil {
		result.Description = optionalLegacyString(item.Description.Description, "copied from legacy Descriptions.Description")
		result.Location = optionalLegacyString(item.Description.Location, "copied from legacy Descriptions.Location")
	}
	name, err := context.itemNameFact(item.ID, family, safety, isVariant)
	if err != nil {
		return schema.ItemPresentation{}, err
	}
	result.Name = name
	if item.IconPath != "" {
		result.IconPath = knownIconFact("assets/icons/" + path.Clean(item.IconPath))
	} else {
		result.IconPath = unknownCatalogFact[string]("legacy item has no icon path")
	}
	return result, nil
}

// visionOfGraceGameID is the single base item the official FMG name catalogs do
// not cover. It keeps its safety flags and carries an unknown name.
const visionOfGraceGameID uint32 = 0x400000A6

// itemNameFact resolves the one official FMG name for a game ID. The name is
// unknown only for Vision of Grace, for cut-content or ban-risk items whose FMG
// entry is empty, missing or an "[ERROR]" placeholder, and for variants whose
// FMG entry is an "[ERROR]" placeholder. Anything else is a generation error.
func (context *generationContext) itemNameFact(
	gameID uint32,
	family schema.ItemFamily,
	safety schema.ItemSafety,
	isVariant bool,
) (schema.Fact[string], error) {
	entry, exists, err := context.gameText.lookupItemName(family, gameID)
	if err != nil {
		return schema.Fact[string]{}, fmt.Errorf("item 0x%08X: %w", gameID, err)
	}
	placeholder := exists && strings.HasPrefix(entry.text, gameTextErrorPrefix)
	if exists && !placeholder && entry.text != "" {
		return schema.Fact[string]{
			Known: true,
			Value: entry.text,
			Provenance: schema.Provenance{
				Source: entry.source,
				Method: fmt.Sprintf(
					"resolved the official English %s entry %d",
					entry.fmgFile,
					entry.entryID,
				),
			},
		}, nil
	}
	switch {
	case gameID == visionOfGraceGameID:
		return unknownCatalogFact[string](
			"no official FMG name entry exists for this item",
		), nil
	case isVariant && placeholder:
		return unknownCatalogFact[string](fmt.Sprintf(
			"official %s entry %d is an %s placeholder",
			entry.fmgFile,
			entry.entryID,
			gameTextErrorPrefix,
		)), nil
	case safety.CutContent.Value || safety.BanRisk.Value:
		return unknownCatalogFact[string](
			"cut-content or ban-risk item has no usable official FMG name entry",
		), nil
	}
	return schema.Fact[string]{}, fmt.Errorf(
		"item 0x%08X (family %q) has no usable official FMG name entry %d",
		gameID,
		family,
		gameTextEntryID(gameID),
	)
}

func emptyPresentation() schema.ItemPresentation {
	return schema.ItemPresentation{
		Name:        unknownCatalogFact[string]("official FMG name is unknown"),
		Caption:     unknownCatalogFact[string]("legacy caption is unknown"),
		Description: unknownCatalogFact[string]("legacy description is unknown"),
		Location:    unknownCatalogFact[string]("legacy location is unknown"),
		IconPath:    unknownCatalogFact[string]("legacy icon path is unknown"),
		TextMetadata: schema.ItemTextMetadata{
			CaptionSource:     unknownCatalogFact[string]("legacy caption source is unknown"),
			DescriptionSource: unknownCatalogFact[string]("legacy description source is unknown"),
			LocationSource:    unknownCatalogFact[string]("legacy location source is unknown"),
			DLCSource:         unknownCatalogFact[string]("legacy DLC text source is unknown"),
			Notes:             unknownCatalogFact[string]("legacy text notes are unknown"),
		},
	}
}

func optionalLegacyString(value, method string) schema.Fact[string] {
	if strings.TrimSpace(value) == "" {
		return unknownCatalogFact[string]("legacy field is empty")
	}
	return knownLegacyFact(value, method)
}

func (context *generationContext) buildStorage(
	item seed,
	family schema.ItemFamily,
	primaryRow ParameterRow,
	hasPrimaryRow bool,
) (schema.ItemStorage, error) {
	storage := authoredStorage(item)
	legacyGameInventory, legacyGameStorage := legacyGameLimits(item)
	goodsRow, goodsExists, err := context.goodsStorageRow(
		item,
		family,
		primaryRow,
		hasPrimaryRow,
	)
	if err != nil {
		return schema.ItemStorage{}, err
	}
	if goodsExists {
		maxInventory, err := regulationUint32(goodsRow, "maxNum")
		if err != nil {
			return schema.ItemStorage{}, err
		}
		maxStorage, err := regulationUint32(goodsRow, "maxRepositoryNum")
		if err != nil {
			return schema.ItemStorage{}, err
		}
		authoritativeInventory := regulationGameLimitFact(
			maxInventory,
			legacyGameInventory,
			"maxNum",
			goodsRow.RowID,
		)
		authoritativeStorage := regulationGameLimitFact(
			maxStorage,
			legacyGameStorage,
			"maxRepositoryNum",
			goodsRow.RowID,
		)
		if item.ID == furlcallingFingerRemedyItemID {
			authoritativeStorage = knownRegulationFact(
				maxStorage,
				RegulationTableGoods,
				"maxRepositoryNum",
				goodsRow.RowID,
			)
		}
		storage.MaxInventorySFV = saveForgeValue(
			item.HasLegacyItem,
			storage.MaxInventory.Value,
			authoritativeInventory.Value,
			"preserved conflicting legacy ItemData.MaxInventory",
		)
		storage.MaxStorageSFV = saveForgeValue(
			item.HasLegacyItem,
			storage.MaxStorage.Value,
			authoritativeStorage.Value,
			"preserved conflicting legacy ItemData.MaxStorage",
		)
		storage.MaxInventory = authoritativeInventory
		storage.MaxStorage = authoritativeStorage
		promoteSafeModeStorageLimits(&storage, item)
		promoteKeyItemNG0InventoryLimits(&storage, item)
		discardReviewedStorageSaveForgeValues(&storage, item, family)
		return storage, nil
	}
	return storage, nil
}

func regulationGameLimitFact(
	raw uint32,
	legacy schema.Fact[uint32],
	field string,
	rowID uint32,
) schema.Fact[uint32] {
	if raw != 0 {
		return knownRegulationFact(raw, RegulationTableGoods, field, rowID)
	}
	if legacy.Known {
		return knownRegulationDerivedFact(
			legacy.Value,
			RegulationTableGoods,
			field+" is the zero sentinel; retained the migrated effective limit",
			rowID,
			field,
		)
	}
	return knownRegulationFact(uint32(0), RegulationTableGoods, field, rowID)
}

func authoredStorage(item seed) schema.ItemStorage {
	if !item.HasLegacyItem && !item.RegulationOnlyVariant {
		return schema.ItemStorage{
			RecordMode:   unknownLegacyFact[schema.RecordMode]("slot-only gesture has no legacy ItemData record mode"),
			MaxInventory: unknownLegacyFact[uint32]("slot-only gesture has no legacy ItemData.MaxInventory"),
			MaxStorage:   unknownLegacyFact[uint32]("slot-only gesture has no legacy ItemData.MaxStorage"),
		}
	}
	recordModeMethod := "normalized from the verified item record family"
	maxInventoryMethod := "copied from legacy ItemData.MaxInventory"
	maxStorageMethod := "copied from legacy ItemData.MaxStorage"
	if item.RegulationOnlyVariant {
		recordModeMethod = "copied from the canonical item record family for a Regulation-only variant"
		maxInventoryMethod = "copied from the canonical legacy ItemData.MaxInventory for a Regulation-only variant"
		maxStorageMethod = "copied from the canonical legacy ItemData.MaxStorage for a Regulation-only variant"
	}
	return schema.ItemStorage{
		RecordMode:   knownLegacyFact(recordMode(item), recordModeMethod),
		MaxInventory: knownLegacyFact(item.MaxInventory, maxInventoryMethod),
		MaxStorage:   knownLegacyFact(item.MaxStorage, maxStorageMethod),
	}
}

func (context *generationContext) goodsStorageRow(
	item seed,
	family schema.ItemFamily,
	primaryRow ParameterRow,
	hasPrimaryRow bool,
) (ParameterRow, bool, error) {
	switch family {
	case schema.ItemFamilyGoods, schema.ItemFamilyGesture, schema.ItemFamilySpiritAsh:
		if hasPrimaryRow {
			return primaryRow, true, nil
		}
	case schema.ItemFamilySpell:
		lookup, exists, err := context.regulation.LookupFamilyRow(
			RegulationFamilyGoods,
			RegulationTableRolePrimary,
			item.ID&0x0FFFFFFF,
		)
		if err != nil {
			return ParameterRow{}, false, err
		}
		if exists {
			return lookup.Row, true, nil
		}
	}
	return ParameterRow{}, false, nil
}

// legacyGameLimits reports the migrated legacy technical limits that back the
// Regulation zero-sentinel fallback in regulationGameLimitFact.
func legacyGameLimits(item seed) (schema.Fact[uint32], schema.Fact[uint32]) {
	inventory := unknownCatalogFact[uint32]("legacy game inventory limit is unknown")
	storage := unknownCatalogFact[uint32]("legacy game storage limit is unknown")
	if item.GameMaxInventoryKnown {
		inventory = knownLegacyFact(
			item.GameMaxInventory,
			"copied from legacy ItemData.GameMaxInventory",
		)
	} else if item.GameLimits != nil && item.GameLimits.InventoryKnown {
		inventory = knownLegacyFact(
			item.GameLimits.MaxInventory,
			"copied from legacy GameLimitsByItemID.MaxInventory",
		)
	}
	if item.GameMaxStorageKnown {
		storage = knownLegacyFact(
			item.GameMaxStorage,
			"copied from legacy ItemData.GameMaxStorage",
		)
	} else if item.GameLimits != nil && item.GameLimits.StorageKnown {
		storage = knownLegacyFact(
			item.GameLimits.MaxStorage,
			"copied from legacy GameLimitsByItemID.MaxStorage",
		)
	}
	return inventory, storage
}

func recordMode(item seed) schema.RecordMode {
	if item.Category == "arrows_and_bolts" || item.ID&0xF0000000 == 0x40000000 {
		return schema.RecordModeQuantityStack
	}
	return schema.RecordModeSeparateInstances
}
