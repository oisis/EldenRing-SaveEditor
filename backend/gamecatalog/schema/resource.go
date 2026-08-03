package schema

type ResourceID uint32

type ResourceKind string

const (
	ResourceKindItem ResourceKind = "item"
)

type Resource struct {
	ID   ResourceID    `json:"id"`
	Key  string        `json:"key"`
	Kind ResourceKind  `json:"kind"`
	Item *ItemDocument `json:"item"`
}
