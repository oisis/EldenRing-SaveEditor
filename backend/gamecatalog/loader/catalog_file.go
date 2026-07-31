package loader

import (
	"io/fs"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const CatalogFileName = "catalog.json"

type CatalogFile struct {
	Manifest  schema.Manifest `json:"manifest"`
	Documents []string        `json:"documents"`
}

type Document struct {
	Path     string
	Resource schema.Resource
}

type Data struct {
	Manifest      schema.Manifest
	Documents     []Document
	sourceFS      fs.FS
	documentPaths map[string]struct{}
	assets        map[string]string
}

func (data Data) Resources() []schema.Resource {
	resources := make([]schema.Resource, 0, len(data.Documents))
	for _, document := range data.Documents {
		resources = append(resources, document.Resource)
	}
	return resources
}

func (data Data) ReadAsset(assetPath string) ([]byte, bool) {
	content, _, exists := data.ReadAssetWithMediaType(assetPath)
	return content, exists
}

func (data Data) ReadAssetWithMediaType(assetPath string) ([]byte, string, bool) {
	mediaType, exists := data.assets[assetPath]
	if !exists {
		return nil, "", false
	}
	content, err := fs.ReadFile(data.sourceFS, assetPath)
	if err != nil {
		return nil, "", false
	}
	return append([]byte(nil), content...), mediaType, true
}

func (data Data) ReadDocument(documentPath string) ([]byte, bool) {
	if _, exists := data.documentPaths[documentPath]; !exists {
		return nil, false
	}
	content, err := fs.ReadFile(data.sourceFS, documentPath)
	if err != nil {
		return nil, false
	}
	return append([]byte(nil), content...), true
}
