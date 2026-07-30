package dbviewer

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

type pageMeta struct {
	Title         string
	Manifest      schema.Manifest
	ResourceCount int
}

func (server *Server) pageMeta(title string) pageMeta {
	return pageMeta{
		Title:         title,
		Manifest:      server.catalog.Manifest(),
		ResourceCount: server.catalog.ResourceCount(),
	}
}
