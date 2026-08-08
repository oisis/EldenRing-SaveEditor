package schema

import "fmt"

func ValidateRelation(relation Relation, sources map[SourceID]struct{}) error {
	if err := validateRelationEndpoint("from", relation.From); err != nil {
		return err
	}
	if err := validateRelationEndpoint("to", relation.To); err != nil {
		return err
	}
	if relation.From == relation.To {
		return fmt.Errorf("relation cannot point to the same resource")
	}
	if relation.Kind != RelationCompatibleWithAshOfWar &&
		relation.Kind != RelationRequiresContainer {
		return fmt.Errorf("unsupported relation kind %q", relation.Kind)
	}
	return validateProvenance("relation", relation.Provenance, sources)
}

func validateRelationEndpoint(side string, ref ResourceRef) error {
	if ref.Kind == "" {
		return fmt.Errorf("relation %s endpoint requires a resource kind", side)
	}
	if ref.Kind != ResourceKindItem {
		return fmt.Errorf("relation %s endpoint has unsupported kind %q", side, ref.Kind)
	}
	if ref.Key == "" {
		return fmt.Errorf("relation %s endpoint requires a resource key", side)
	}
	return nil
}
