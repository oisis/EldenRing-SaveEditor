package loader

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func LoadDir(directory string) (Data, error) {
	return LoadFS(os.DirFS(directory))
}

func LoadFS(catalogFS fs.FS) (Data, error) {
	indexJSON, err := fs.ReadFile(catalogFS, CatalogFileName)
	if err != nil {
		return Data{}, fmt.Errorf("read %s: %w", CatalogFileName, err)
	}

	var index CatalogFile
	if err := decodeJSON(indexJSON, &index); err != nil {
		return Data{}, fmt.Errorf("decode %s: %w", CatalogFileName, err)
	}
	if len(index.Documents) == 0 {
		return Data{}, fmt.Errorf("%s: at least one document is required", CatalogFileName)
	}

	data := Data{
		Manifest:      index.Manifest,
		Documents:     make([]Document, 0, len(index.Documents)),
		sourceFS:      catalogFS,
		documentPaths: make(map[string]struct{}, len(index.Documents)),
		assets:        make(map[string]string),
	}
	seenPaths := make(map[string]struct{}, len(index.Documents))
	for index, documentPath := range index.Documents {
		if err := validateDocumentPath(documentPath); err != nil {
			return Data{}, fmt.Errorf("document %d path %q: %w", index, documentPath, err)
		}
		if _, exists := seenPaths[documentPath]; exists {
			return Data{}, fmt.Errorf("document %d path %q: duplicate path", index, documentPath)
		}
		seenPaths[documentPath] = struct{}{}
		data.documentPaths[documentPath] = struct{}{}

		rawJSON, err := fs.ReadFile(catalogFS, documentPath)
		if err != nil {
			return Data{}, fmt.Errorf("read document %q: %w", documentPath, err)
		}
		var resource schema.Resource
		if err := decodeJSON(rawJSON, &resource); err != nil {
			return Data{}, fmt.Errorf("decode document %q: %w", documentPath, err)
		}
		if err := loadKnownIconAssets(catalogFS, resource, data.assets); err != nil {
			return Data{}, fmt.Errorf("document %q: %w", documentPath, err)
		}
		data.Documents = append(data.Documents, Document{
			Path:     documentPath,
			Resource: resource,
		})
	}
	if err := loadAppearanceAssets(catalogFS, data.assets); err != nil {
		return Data{}, err
	}
	return data, nil
}
