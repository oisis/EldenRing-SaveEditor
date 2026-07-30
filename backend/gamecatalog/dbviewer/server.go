package dbviewer

import (
	"fmt"
	"net/http"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type Server struct {
	catalog       *gamecatalog.Catalog
	data          loader.Data
	documentsByID map[schema.ResourceID]loader.Document
	sources       map[schema.SourceID]schema.DataSource
	templates     templateSet
	handler       http.Handler
}

func New(data loader.Data) (*Server, error) {
	catalog, err := gamecatalog.New(data.Manifest, data.Resources())
	if err != nil {
		return nil, fmt.Errorf("build catalog: %w", err)
	}
	templates, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	server := &Server{
		catalog:       catalog,
		data:          data,
		documentsByID: make(map[schema.ResourceID]loader.Document, len(data.Documents)),
		sources:       make(map[schema.SourceID]schema.DataSource, len(data.Manifest.Sources)),
		templates:     templates,
	}
	for _, document := range data.Documents {
		server.documentsByID[document.Resource.ID] = document
	}
	for _, source := range data.Manifest.Sources {
		server.sources[source.ID] = source
	}
	server.handler = server.routes()
	return server, nil
}

func (server *Server) Handler() http.Handler {
	return server.handler
}
