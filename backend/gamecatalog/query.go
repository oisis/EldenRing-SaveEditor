package gamecatalog

import (
	"fmt"
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type ItemView struct {
	Resource          schema.Resource
	OutgoingRelations []schema.Relation
	IncomingRelations []schema.Relation
	RelatedResources  []schema.Resource
}

// ResourceByKindAndKey resolves one resource by the exact (kind, key) pair. The
// kind is selected first and the key is matched only inside that kind; neither
// value is trimmed, normalised, or retried under another kind. The returned
// resource is an independent deep copy and carries no relations.
func (catalog *Catalog) ResourceByKindAndKey(
	kind schema.ResourceKind,
	key string,
) (schema.Resource, error) {
	resource, err := catalog.requireResource(kind, key)
	if err != nil {
		return schema.Resource{}, err
	}
	return cloneResource(resource), nil
}

// RelationsByKindAndKey returns the relations of one resource resolved by the
// exact (kind, key) pair. The kind is resolved first and the key is matched only
// inside that kind, so an unknown kind and an unknown key stay the two distinct
// errors ResourceByKindAndKey reports. Both slices are independent copies in
// catalog order, so the internal relation maps are never exposed, and a known
// resource without relations returns two empty, non-nil slices. The documents of
// the related resources are not part of the result.
func (catalog *Catalog) RelationsByKindAndKey(
	kind schema.ResourceKind,
	key string,
) (outgoing []schema.Relation, incoming []schema.Relation, err error) {
	resource, err := catalog.requireResource(kind, key)
	if err != nil {
		return nil, nil, err
	}
	ref := resource.Ref()
	return append([]schema.Relation{}, catalog.outgoing[ref]...),
		append([]schema.Relation{}, catalog.incoming[ref]...),
		nil
}

// requireResource is the single place that turns a (kind, key) pair into the
// stored resource, so every getter built on that pair reports the same two
// failures with the same wording. The returned resource is the stored value and
// must never leave the package without a clone.
func (catalog *Catalog) requireResource(
	kind schema.ResourceKind,
	key string,
) (schema.Resource, error) {
	byKey, exists := catalog.byKind[kind]
	if !exists {
		return schema.Resource{}, fmt.Errorf("unknown resource kind %q", kind)
	}
	resource, exists := byKey[key]
	if !exists {
		return schema.Resource{}, fmt.Errorf("unknown resource key %q in kind %q", key, kind)
	}
	return resource, nil
}

func (catalog *Catalog) resourceByRef(ref schema.ResourceRef) (schema.Resource, bool) {
	byKey, exists := catalog.byKind[ref.Kind]
	if !exists {
		return schema.Resource{}, false
	}
	resource, exists := byKey[ref.Key]
	if !exists {
		return schema.Resource{}, false
	}
	return cloneResource(resource), true
}

func (catalog *Catalog) ItemByGameID(gameID uint32) (schema.Resource, bool) {
	ref, exists := catalog.byItemGameID[gameID]
	if !exists {
		return schema.Resource{}, false
	}
	resource, exists := catalog.resourceByRef(ref)
	if !exists || resource.Item == nil || resource.Item.GameID.Value == gameID {
		return resource, exists
	}
	for _, variant := range resource.Item.Variants {
		if !variant.GameID.Known || variant.GameID.Value != gameID {
			continue
		}
		materialized := schema.MaterializeVariant(*resource.Item, variant)
		resource.Item = &materialized
		return resource, true
	}
	for _, alias := range resource.Item.Aliases {
		if alias.GameID.Known && alias.GameID.Value == gameID {
			return resource, true
		}
	}
	return schema.Resource{}, false
}

func (catalog *Catalog) ItemViewByGameID(gameID uint32) (ItemView, bool) {
	resource, exists := catalog.ItemByGameID(gameID)
	if !exists {
		return ItemView{}, false
	}

	ref := resource.Ref()
	outgoing := cloneRelations(catalog.outgoing[ref])
	incoming := cloneRelations(catalog.incoming[ref])
	relatedRefs := make(map[schema.ResourceRef]struct{}, len(outgoing)+len(incoming))
	for _, relation := range outgoing {
		relatedRefs[relation.To] = struct{}{}
	}
	for _, relation := range incoming {
		relatedRefs[relation.From] = struct{}{}
	}
	refs := make([]schema.ResourceRef, 0, len(relatedRefs))
	for relatedRef := range relatedRefs {
		refs = append(refs, relatedRef)
	}
	sortResourceRefs(refs)

	related := make([]schema.Resource, 0, len(refs))
	for _, relatedRef := range refs {
		relatedResource, found := catalog.resourceByRef(relatedRef)
		if !found {
			continue
		}
		related = append(related, relatedResource)
	}
	return ItemView{
		Resource:          resource,
		OutgoingRelations: outgoing,
		IncomingRelations: incoming,
		RelatedResources:  related,
	}, true
}

// sortResourceRefs orders references by kind and only then by key, which is the
// deterministic order every catalog-derived list uses.
func sortResourceRefs(refs []schema.ResourceRef) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Kind != refs[j].Kind {
			return refs[i].Kind < refs[j].Kind
		}
		return refs[i].Key < refs[j].Key
	})
}
