package loader

import (
	"fmt"
	"io/fs"
	"path"
)

func validateDocumentPath(documentPath string) error {
	if !fs.ValidPath(documentPath) {
		return fmt.Errorf("must be a relative, slash-separated path inside the catalog")
	}
	if documentPath == CatalogFileName {
		return fmt.Errorf("must not point to %s", CatalogFileName)
	}
	if path.Ext(documentPath) != ".json" {
		return fmt.Errorf("must point to a JSON document")
	}
	return nil
}
