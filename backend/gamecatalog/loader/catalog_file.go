package loader

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

const CatalogFileName = "catalog.json"

type CatalogFile struct {
	Manifest  schema.Manifest `json:"manifest"`
	Documents []string        `json:"documents"`
}

type Document struct {
	Path     string
	Resource schema.Resource
	RawJSON  []byte
}

type Data struct {
	Manifest  schema.Manifest
	Documents []Document
}

func (data Data) Resources() []schema.Resource {
	resources := make([]schema.Resource, 0, len(data.Documents))
	for _, document := range data.Documents {
		resources = append(resources, document.Resource)
	}
	return resources
}
