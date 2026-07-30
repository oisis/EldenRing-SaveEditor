package dbviewer

import (
	"net/http"
	"strings"
)

type rawPage struct {
	Meta         pageMeta
	Name         string
	GameID       string
	GameIDPath   string
	DocumentPath string
	JSON         string
}

func (server *Server) rawItemHandler(response http.ResponseWriter, request *http.Request) {
	gameID, err := parseGameID(request.PathValue("gameID"))
	if err != nil {
		http.NotFound(response, request)
		return
	}
	resource, exists := server.catalog.ItemByGameID(gameID)
	if !exists {
		http.NotFound(response, request)
		return
	}
	document, exists := server.documentsByID[resource.ID]
	if !exists {
		http.Error(response, "Catalog document is unavailable.", http.StatusInternalServerError)
		return
	}
	baseGameID := formatGameID(resource.Item.GameID.Value)
	server.render(response, "raw", rawPage{
		Meta:         server.pageMeta(resource.Label.Value + " raw JSON"),
		Name:         resource.Label.Value,
		GameID:       baseGameID,
		GameIDPath:   strings.TrimPrefix(baseGameID, "0x"),
		DocumentPath: document.Path,
		JSON:         string(document.RawJSON),
	})
}
