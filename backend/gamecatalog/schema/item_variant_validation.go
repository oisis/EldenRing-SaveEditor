package schema

import "fmt"

func validateVariants(
	item ItemDocument,
	sources map[SourceID]struct{},
) error {
	variants := item.Variants
	seen := make(map[uint32]struct{}, len(variants))
	for index, variant := range variants {
		name := fmt.Sprintf("item.variants[%d]", index)
		if err := validateFact(name+".gameID", variant.GameID, sources); err != nil {
			return err
		}
		if !variant.GameID.Known || variant.GameID.Value == 0 {
			return fmt.Errorf("%s.gameID must be known and greater than zero", name)
		}
		if _, exists := seen[variant.GameID.Value]; exists {
			return fmt.Errorf("%s: duplicate game ID 0x%08X", name, variant.GameID.Value)
		}
		seen[variant.GameID.Value] = struct{}{}
		if err := validateFact(name+".sourceRowID", variant.SourceRowID, sources); err != nil {
			return err
		}
		if !variant.SourceRowID.Known || variant.SourceRowID.Value == 0 {
			return fmt.Errorf("%s.sourceRowID must be known and greater than zero", name)
		}
		if err := validateVariantKind(name, variant, item, sources); err != nil {
			return err
		}
		if !isOmittedFact(variant.Kind) {
			if err := validateVariantDocumentData(
				name+".data",
				variant.Data,
				variant,
				item,
				sources,
			); err != nil {
				return err
			}
		}
		if err := validateParameterRecords(name+".sourceRecords", variant.SourceRecords, sources); err != nil {
			return err
		}
	}
	return nil
}

func validateVariantKind(
	name string,
	variant ItemVariant,
	item ItemDocument,
	sources map[SourceID]struct{},
) error {
	if isOmittedFact(variant.Kind) {
		if err := validateFact(name+".affinity", variant.Affinity, sources); err != nil {
			return err
		}
		if !variant.Affinity.Known || !validAffinity(variant.Affinity.Value) {
			return fmt.Errorf("%s.affinity must be known and supported", name)
		}
		return nil
	}
	if err := validateFact(name+".kind", variant.Kind, sources); err != nil {
		return err
	}
	if !variant.Kind.Known {
		return fmt.Errorf("%s.kind must be known", name)
	}
	switch variant.Kind.Value {
	case ItemVariantAffinity:
		if err := validateKnownAffinity(name, variant.Affinity, sources); err != nil {
			return err
		}
		if err := validateKnownUpgradeLevel(name, variant.UpgradeLevel, sources); err != nil {
			return err
		}
		if variant.UpgradeLevel.Value != 0 {
			return fmt.Errorf("%s.upgradeLevel must be zero for affinity variant", name)
		}
	case ItemVariantUpgrade:
		if err := validateKnownUpgradeLevel(name, variant.UpgradeLevel, sources); err != nil {
			return err
		}
		if item.Family.Value == ItemFamilySpiritAsh && isNotApplicableFact(variant.Affinity) {
			if err := validateFact(name+".affinity", variant.Affinity, sources); err != nil {
				return err
			}
			return nil
		}
		if !isOmittedFact(variant.Affinity) {
			return fmt.Errorf("%s.affinity must be omitted for upgrade variant", name)
		}
	case ItemVariantAffinityUpgrade:
		if err := validateKnownAffinity(name, variant.Affinity, sources); err != nil {
			return err
		}
		if err := validateKnownUpgradeLevel(name, variant.UpgradeLevel, sources); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%s.kind has unsupported value %q", name, variant.Kind.Value)
	}
	return nil
}

func validateKnownAffinity(name string, affinity Fact[Affinity], sources map[SourceID]struct{}) error {
	if err := validateFact(name+".affinity", affinity, sources); err != nil {
		return err
	}
	if !affinity.Known || !validAffinity(affinity.Value) {
		return fmt.Errorf("%s.affinity must be known and supported", name)
	}
	return nil
}

func validateKnownUpgradeLevel(name string, level Fact[uint8], sources map[SourceID]struct{}) error {
	if err := validateFact(name+".upgradeLevel", level, sources); err != nil {
		return err
	}
	if !level.Known {
		return fmt.Errorf("%s.upgradeLevel must be known", name)
	}
	return nil
}

func validateVariantDocumentData(
	name string,
	data VariantDocumentData,
	variant ItemVariant,
	item ItemDocument,
	sources map[SourceID]struct{},
) error {
	if !data.Family.Known || data.Family.Value != item.Family.Value {
		return fmt.Errorf(
			"%s.family must be known and match canonical family %q",
			name,
			item.Family.Value,
		)
	}
	materialized := MaterializeVariant(item, variant)
	materialized.Variants = nil
	materialized.Aliases = nil
	if err := validateItemDocument(materialized, sources); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
