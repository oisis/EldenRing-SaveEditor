package gamecatalog

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type ItemView struct {
	Resource          schema.Resource
	OutgoingRelations []schema.Relation
	IncomingRelations []schema.Relation
	RelatedResources  []schema.Resource
}

func (catalog *Catalog) ResourceByID(id schema.ResourceID) (schema.Resource, bool) {
	resource, exists := catalog.byID[id]
	if !exists {
		return schema.Resource{}, false
	}
	return cloneResource(resource), true
}

func (catalog *Catalog) ItemByGameID(gameID uint32) (schema.Resource, bool) {
	id, exists := catalog.byItemGameID[gameID]
	if !exists {
		return schema.Resource{}, false
	}
	return catalog.ResourceByID(id)
}

func (catalog *Catalog) ItemViewByGameID(gameID uint32) (ItemView, bool) {
	resource, exists := catalog.ItemByGameID(gameID)
	if !exists {
		return ItemView{}, false
	}

	outgoing := cloneRelations(catalog.outgoing[resource.ID])
	incoming := cloneRelations(catalog.incoming[resource.ID])
	relatedIDs := make(map[schema.ResourceID]struct{}, len(outgoing)+len(incoming))
	for _, relation := range outgoing {
		relatedIDs[relation.To] = struct{}{}
	}
	for _, relation := range incoming {
		relatedIDs[relation.From] = struct{}{}
	}
	ids := make([]int, 0, len(relatedIDs))
	for id := range relatedIDs {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	related := make([]schema.Resource, 0, len(ids))
	for _, rawID := range ids {
		related = append(related, cloneResource(catalog.byID[schema.ResourceID(rawID)]))
	}
	return ItemView{
		Resource:          resource,
		OutgoingRelations: outgoing,
		IncomingRelations: incoming,
		RelatedResources:  related,
	}, true
}
