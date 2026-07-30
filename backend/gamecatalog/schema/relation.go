package schema

type RelationKind string

const (
	RelationCompatibleWithAshOfWar RelationKind = "compatible_with_aow"
)

type Relation struct {
	From       ResourceID
	To         ResourceID
	Kind       RelationKind
	Provenance Provenance
}
