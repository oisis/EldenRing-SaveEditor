package schema

type RelationKind string

const (
	RelationCompatibleWithAshOfWar RelationKind = "compatible_with_aow"
)

type Relation struct {
	From       ResourceID   `json:"from"`
	To         ResourceID   `json:"to"`
	Kind       RelationKind `json:"kind"`
	Provenance Provenance   `json:"provenance"`
}
