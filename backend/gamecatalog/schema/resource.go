package schema

type ResourceKind string

const (
	ResourceKindItem ResourceKind = "item"
)

// ResourceRef is the identity of one catalog resource: the exact pair
// (Kind, Key). Kind is resolved first and Key is only meaningful inside that
// kind, so the same Key may later exist under a different Kind. Neither field
// is trimmed, normalised or matched with a fallback.
type ResourceRef struct {
	Kind ResourceKind `json:"kind"`
	Key  string       `json:"key"`
}

type Resource struct {
	Key  string        `json:"key"`
	Kind ResourceKind  `json:"kind"`
	Item *ItemDocument `json:"item"`
}

func (resource Resource) Ref() ResourceRef {
	return ResourceRef{Kind: resource.Kind, Key: resource.Key}
}
