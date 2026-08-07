package schema

import (
	"fmt"
	"reflect"
)

func validateItemDocument(item ItemDocument, sources map[SourceID]struct{}) error {
	if err := validateFact("item.gameID", item.GameID, sources); err != nil {
		return err
	}
	if !item.GameID.Known || item.GameID.Value == 0 {
		return fmt.Errorf("item.gameID must be known and greater than zero")
	}
	if err := validateFact("item.family", item.Family, sources); err != nil {
		return err
	}
	if !item.Family.Known {
		return fmt.Errorf("item.family must be known")
	}
	if err := validateOptionalNonEmptyString("item.category", item.Category, sources); err != nil {
		return err
	}
	if err := validateFact("item.subcategory", item.Subcategory, sources); err != nil {
		return err
	}
	if item.Subcategory.Known && item.Subcategory.Value == "" {
		return fmt.Errorf("item.subcategory cannot be empty when known")
	}
	if err := validatePresentation(item.Presentation, sources); err != nil {
		return err
	}
	if err := validateStorage(item.Storage, sources); err != nil {
		return err
	}
	if err := validateSafety(item.Safety, sources); err != nil {
		return err
	}
	if err := validateAcquisition(item.Acquisition, sources); err != nil {
		return err
	}
	if err := validateModifiers(item.Modifiers, sources); err != nil {
		return err
	}
	if err := validateItemLinks("item.links", item.Links, item.GameID.Value, item.Unlocks, sources); err != nil {
		return err
	}
	if err := validateRelatedTechnicalRecords(
		"item.relatedTechnicalRecords",
		item.RelatedTechnicalRecords,
		item.GameID.Value,
		sources,
	); err != nil {
		return err
	}
	if err := validateCapabilities(item.Capabilities, sources); err != nil {
		return err
	}
	if err := validateVariants(item, sources); err != nil {
		return err
	}
	if err := validateAliases(item.Aliases, item, sources); err != nil {
		return err
	}
	if err := validateUnlocks(item.Unlocks, sources); err != nil {
		return err
	}
	if err := validateParameterRecords("item.sourceRecords", item.SourceRecords, sources); err != nil {
		return err
	}
	if err := validateFamilyDocument(item, sources); err != nil {
		return err
	}
	if err := validateSaveForgeValues(
		"item",
		reflect.ValueOf(item),
		sources,
	); err != nil {
		return err
	}
	document := item
	document.Variants = nil
	document.Aliases = nil
	document.RelatedTechnicalRecords = nil
	document.SourceRecords = nil
	return validateRegulationProvenanceCoverage(
		"item",
		document,
		item.SourceRecords,
	)
}

func validatePresentation(presentation ItemPresentation, sources map[SourceID]struct{}) error {
	if err := validateOptionalNonEmptyString("item.presentation.name", presentation.Name, sources); err != nil {
		return err
	}
	optional := []struct {
		name string
		fact Fact[string]
	}{
		{"item.presentation.caption", presentation.Caption},
		{"item.presentation.description", presentation.Description},
		{"item.presentation.location", presentation.Location},
		{"item.presentation.iconPath", presentation.IconPath},
	}
	for _, entry := range optional {
		if err := validateOptionalFact(entry.name, entry.fact, sources); err != nil {
			return err
		}
		if entry.fact.Known && entry.fact.Value == "" {
			return fmt.Errorf("%s cannot be empty when known", entry.name)
		}
	}
	return validateTextMetadata(presentation.TextMetadata, sources)
}

func validateStorage(storage ItemStorage, sources map[SourceID]struct{}) error {
	if err := validateFact("item.storage.recordMode", storage.RecordMode, sources); err != nil {
		return err
	}
	if storage.RecordMode.Known &&
		storage.RecordMode.Value != RecordModeQuantityStack &&
		storage.RecordMode.Value != RecordModeSeparateInstances {
		return fmt.Errorf("item.storage.recordMode has unsupported value %q", storage.RecordMode.Value)
	}
	if err := validateFact("item.storage.maxInventory", storage.MaxInventory, sources); err != nil {
		return err
	}
	if storage.SafeModeMaxInventory != nil {
		if err := validateFact("item.storage.safeModeMaxInventory", *storage.SafeModeMaxInventory, sources); err != nil {
			return err
		}
	}
	if err := validateFact("item.storage.maxStorage", storage.MaxStorage, sources); err != nil {
		return err
	}
	if storage.SafeModeMaxStorage != nil {
		if err := validateFact("item.storage.safeModeMaxStorage", *storage.SafeModeMaxStorage, sources); err != nil {
			return err
		}
	}
	if err := validateOptionalFact("item.storage.gameMaxInventory", storage.GameMaxInventory, sources); err != nil {
		return err
	}
	return validateOptionalFact("item.storage.gameMaxStorage", storage.GameMaxStorage, sources)
}

func validateSafety(safety ItemSafety, sources map[SourceID]struct{}) error {
	if err := validateFact("item.safety.cutContent", safety.CutContent, sources); err != nil {
		return err
	}
	if err := validateFact("item.safety.banRisk", safety.BanRisk, sources); err != nil {
		return err
	}
	optional := []struct {
		name string
		fact Fact[bool]
	}{
		{"item.safety.dlc", safety.DLC},
		{"item.safety.noDatabase", safety.NoDatabase},
		{"item.safety.scalesWithNG", safety.ScalesWithNG},
		{"item.safety.preOrder", safety.PreOrder},
	}
	for _, entry := range optional {
		if err := validateOptionalFact(entry.name, entry.fact, sources); err != nil {
			return err
		}
	}
	return nil
}

func validateOptionalNonEmptyString(name string, fact Fact[string], sources map[SourceID]struct{}) error {
	if err := validateOptionalFact(name, fact, sources); err != nil {
		return err
	}
	if fact.Known && fact.Value == "" {
		return fmt.Errorf("%s cannot be empty when known", name)
	}
	return nil
}

func validateOptionalStringList(name string, fact Fact[[]string], sources map[SourceID]struct{}) error {
	if err := validateOptionalFact(name, fact, sources); err != nil {
		return err
	}
	if !fact.Known {
		return nil
	}
	seen := make(map[string]struct{}, len(fact.Value))
	for index, value := range fact.Value {
		if value == "" {
			return fmt.Errorf("%s[%d] cannot be empty", name, index)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%s contains duplicate value %q", name, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}
