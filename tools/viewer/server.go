package dbviewer

import (
	"fmt"
	"net/http"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type Server struct {
	catalog        *gamecatalog.Catalog
	data           loader.Data
	documentsByRef map[schema.ResourceRef]loader.Document
	sources        map[schema.SourceID]schema.DataSource
	templates      templateSet
	handler        http.Handler
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
		catalog:        catalog,
		data:           data,
		documentsByRef: make(map[schema.ResourceRef]loader.Document, len(data.Documents)),
		sources:        make(map[schema.SourceID]schema.DataSource, len(data.Manifest.Sources)),
		templates:      templates,
	}
	for _, document := range data.Documents {
		server.documentsByRef[document.Resource.Ref()] = document
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
