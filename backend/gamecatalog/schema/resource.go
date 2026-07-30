package schema

type ResourceID uint32

type ResourceKind string

const (
	ResourceKindItem ResourceKind = "item"
)

type Resource struct {
	ID    ResourceID
	Key   string
	Kind  ResourceKind
	Label Fact[string]
	Item  *ItemDocument
}
