package schema

import "fmt"

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
	if err := validateFact("item.subcategory", item.Subcategory, sources); err != nil {
		return err
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
	if err := validateCapabilities(item.Capabilities, sources); err != nil {
		return err
	}
	if err := validateVariants(item.Variants, sources); err != nil {
		return err
	}
	return validateFamilyDocument(item, sources)
}

func validatePresentation(presentation ItemPresentation, sources map[SourceID]struct{}) error {
	if err := validateFact("item.presentation.canonicalName", presentation.CanonicalName, sources); err != nil {
		return err
	}
	if !presentation.CanonicalName.Known || presentation.CanonicalName.Value == "" {
		return fmt.Errorf("item.presentation.canonicalName must be known and non-empty")
	}
	optional := []struct {
		name string
		fact Fact[string]
	}{
		{"item.presentation.description", presentation.Description},
		{"item.presentation.iconPath", presentation.IconPath},
	}
	for _, entry := range optional {
		if err := validateFact(entry.name, entry.fact, sources); err != nil {
			return err
		}
		if entry.fact.Known && entry.fact.Value == "" {
			return fmt.Errorf("%s cannot be empty when known", entry.name)
		}
	}
	return nil
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
	return validateFact("item.storage.maxStorage", storage.MaxStorage, sources)
}

func validateSafety(safety ItemSafety, sources map[SourceID]struct{}) error {
	if err := validateFact("item.safety.cutContent", safety.CutContent, sources); err != nil {
		return err
	}
	return validateFact("item.safety.banRisk", safety.BanRisk, sources)
}

func validateVariants(variants []ItemVariant, sources map[SourceID]struct{}) error {
	seen := make(map[uint32]struct{}, len(variants))
	for index, variant := range variants {
		if err := validateFact(fmt.Sprintf("item.variants[%d].gameID", index), variant.GameID, sources); err != nil {
			return err
		}
		if !variant.GameID.Known || variant.GameID.Value == 0 {
			return fmt.Errorf("item.variants[%d].gameID must be known and greater than zero", index)
		}
		if _, exists := seen[variant.GameID.Value]; exists {
			return fmt.Errorf("item.variants[%d]: duplicate game ID 0x%08X", index, variant.GameID.Value)
		}
		seen[variant.GameID.Value] = struct{}{}
		if err := validateFact(fmt.Sprintf("item.variants[%d].affinity", index), variant.Affinity, sources); err != nil {
			return err
		}
		if !variant.Affinity.Known || !validAffinity(variant.Affinity.Value) {
			return fmt.Errorf("item.variants[%d].affinity must be known and supported", index)
		}
		if err := validateFact(fmt.Sprintf("item.variants[%d].sourceRowID", index), variant.SourceRowID, sources); err != nil {
			return err
		}
		if !variant.SourceRowID.Known || variant.SourceRowID.Value == 0 {
			return fmt.Errorf("item.variants[%d].sourceRowID must be known and greater than zero", index)
		}
	}
	return nil
}
