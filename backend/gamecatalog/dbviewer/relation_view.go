package dbviewer

import (
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type relationView struct {
	Direction  string
	Kind       schema.RelationKind
	Name       string
	GameID     string
	GameIDPath string
}

func (server *Server) relationViews(view gamecatalog.ItemView) []relationView {
	resources := make(map[schema.ResourceID]schema.Resource, len(view.RelatedResources))
	for _, resource := range view.RelatedResources {
		resources[resource.ID] = resource
	}
	relations := make([]relationView, 0, len(view.OutgoingRelations)+len(view.IncomingRelations))
	for _, relation := range view.OutgoingRelations {
		relations = append(relations, newRelationView("outgoing", relation.Kind, resources[relation.To]))
	}
	for _, relation := range view.IncomingRelations {
		relations = append(relations, newRelationView("incoming", relation.Kind, resources[relation.From]))
	}
	return relations
}

func newRelationView(direction string, kind schema.RelationKind, resource schema.Resource) relationView {
	gameID := formatGameID(resource.Item.GameID.Value)
	return relationView{
		Direction:  direction,
		Kind:       kind,
		Name:       itemName(resource),
		GameID:     gameID,
		GameIDPath: strings.TrimPrefix(gameID, "0x"),
	}
}
