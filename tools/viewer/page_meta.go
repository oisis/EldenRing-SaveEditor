package dbviewer

import (
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type pageMeta struct {
	Title              string
	Manifest           schema.Manifest
	DisplayGameVersion string
	ResourceCount      int
}

func (server *Server) pageMeta(title string) pageMeta {
	manifest := server.catalog.Manifest()
	return pageMeta{
		Title:              title,
		Manifest:           manifest,
		DisplayGameVersion: displayGameVersion(manifest.GameVersion),
		ResourceCount:      server.catalog.ResourceCount(),
	}
}

func displayGameVersion(version string) string {
	return strings.TrimSuffix(version, "-class")
}
