package schema

type ResourceKind string

const (
	ResourceKindItem          ResourceKind = "item"
	ResourceKindColosseum     ResourceKind = "colosseum"
	ResourceKindRegion        ResourceKind = "region"
	ResourceKindSummoningPool ResourceKind = "summoning_pool"
	ResourceKindGrace         ResourceKind = "grace"
	ResourceKindBoss          ResourceKind = "boss"
)

// ResourceRef is the identity of one catalog resource: the exact pair
// (Kind, Key). Kind is resolved first and Key is only meaningful inside that
// kind, so the same Key may later exist under a different Kind. Neither field
// is trimmed, normalised or matched with a fallback.
type ResourceRef struct {
	Kind ResourceKind `json:"kind"`
	Key  string       `json:"key"`
}

// Resource is the union of the document types the catalog stores. Exactly one
// document pointer matches Kind and every other one stays nil; the nil ones are
// omitted from JSON, so a document never carries a null field for a type it is
// not.
type Resource struct {
	Key           string                 `json:"key"`
	Kind          ResourceKind           `json:"kind"`
	Item          *ItemDocument          `json:"item,omitempty"`
	Colosseum     *ColosseumDocument     `json:"colosseum,omitempty"`
	Region        *RegionDocument        `json:"region,omitempty"`
	SummoningPool *SummoningPoolDocument `json:"summoningPool,omitempty"`
	Grace         *GraceDocument         `json:"grace,omitempty"`
	Boss          *BossDocument          `json:"boss,omitempty"`
}

func (resource Resource) Ref() ResourceRef {
	return ResourceRef{Kind: resource.Kind, Key: resource.Key}
}
