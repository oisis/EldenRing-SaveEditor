package schema

import "fmt"

func ValidateRelation(relation Relation, sources map[SourceID]struct{}) error {
	if relation.From == 0 || relation.To == 0 {
		return fmt.Errorf("relation endpoints must be greater than zero")
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
