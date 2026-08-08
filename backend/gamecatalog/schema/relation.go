package schema

type RelationKind string

const (
	RelationCompatibleWithAshOfWar RelationKind = "compatible_with_aow"
	RelationRequiresContainer      RelationKind = "requires_container"
)

type Relation struct {
	From       ResourceRef  `json:"from"`
	To         ResourceRef  `json:"to"`
	Kind       RelationKind `json:"kind"`
	Provenance Provenance   `json:"provenance"`
}
