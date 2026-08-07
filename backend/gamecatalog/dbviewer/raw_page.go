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
	rawJSON, exists := server.data.ReadDocument(document.Path)
	if !exists {
		http.Error(response, "Raw catalog document is unavailable.", http.StatusInternalServerError)
		return
	}
	name := itemName(resource)
	if variant, variantExists := findVariant(resource.Item, gameID); variantExists {
		name = variantName(resource, variant)
	}
	requestedGameID := formatGameID(gameID)
	server.render(response, "raw", rawPage{
		Meta:         server.pageMeta(name + " raw JSON"),
		Name:         name,
		GameID:       requestedGameID,
		GameIDPath:   strings.TrimPrefix(requestedGameID, "0x"),
		DocumentPath: document.Path,
		JSON:         string(rawJSON),
	})
}
