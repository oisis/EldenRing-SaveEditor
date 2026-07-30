package dbviewer

import (
	"net/http"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

type itemPage struct {
	Meta           pageMeta
	Name           string
	GameID         string
	GameIDPath     string
	Key            string
	DocumentPath   string
	Identity       []factView
	Presentation   []factView
	Storage        []factView
	Safety         []factView
	Capabilities   []capabilityView
	FamilyData     []factView
	Variants       []variantView
	Relations      []relationView
	UnknownCount   int
	HasDescription bool
}

func (server *Server) itemHandler(response http.ResponseWriter, request *http.Request) {
	gameID, err := parseGameID(request.PathValue("gameID"))
	if err != nil {
		http.NotFound(response, request)
		return
	}
	view, exists := server.catalog.ItemViewByGameID(gameID)
	if !exists {
		http.NotFound(response, request)
		return
	}
	document, exists := server.documentsByID[view.Resource.ID]
	if !exists {
		http.Error(response, "Catalog document is unavailable.", http.StatusInternalServerError)
		return
	}
	server.render(response, "item", server.buildItemPage(view, document))
}

func (server *Server) buildItemPage(view gamecatalog.ItemView, document loader.Document) itemPage {
	resource := view.Resource
	item := resource.Item
	gameID := formatGameID(item.GameID.Value)
	return itemPage{
		Meta:         server.pageMeta(resource.Label.Value),
		Name:         resource.Label.Value,
		GameID:       gameID,
		GameIDPath:   strings.TrimPrefix(gameID, "0x"),
		Key:          resource.Key,
		DocumentPath: document.Path,
		Identity: []factView{
			server.fact("Game ID", item.GameID.Known, gameID, item.GameID.Provenance),
			server.fact("Family", item.Family.Known, item.Family.Value, item.Family.Provenance),
			server.fact("Subcategory", item.Subcategory.Known, item.Subcategory.Value, item.Subcategory.Provenance),
		},
		Presentation: []factView{
			server.fact("Canonical name", item.Presentation.CanonicalName.Known, item.Presentation.CanonicalName.Value, item.Presentation.CanonicalName.Provenance),
			server.fact("Description", item.Presentation.Description.Known, item.Presentation.Description.Value, item.Presentation.Description.Provenance),
			server.fact("Icon path", item.Presentation.IconPath.Known, item.Presentation.IconPath.Value, item.Presentation.IconPath.Provenance),
		},
		Storage: []factView{
			server.fact("Record mode", item.Storage.RecordMode.Known, item.Storage.RecordMode.Value, item.Storage.RecordMode.Provenance),
			server.fact("Maximum inventory", item.Storage.MaxInventory.Known, item.Storage.MaxInventory.Value, item.Storage.MaxInventory.Provenance),
			server.fact("Maximum storage", item.Storage.MaxStorage.Known, item.Storage.MaxStorage.Value, item.Storage.MaxStorage.Provenance),
		},
		Safety: []factView{
			server.fact("Cut content", item.Safety.CutContent.Known, item.Safety.CutContent.Value, item.Safety.CutContent.Provenance),
			server.fact("Ban risk", item.Safety.BanRisk.Known, item.Safety.BanRisk.Value, item.Safety.BanRisk.Provenance),
		},
		Capabilities:   server.capabilityViews(item),
		FamilyData:     server.familyFacts(item),
		Variants:       server.variantViews(item),
		Relations:      server.relationViews(view),
		UnknownCount:   countUnknownFacts(resource),
		HasDescription: item.Presentation.Description.Known,
	}
}
