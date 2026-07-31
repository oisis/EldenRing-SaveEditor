package migration

import (
	"strings"
	"testing"
)

func TestValidateGeneratedCatalogRejectsRuntimeIdentityConflict(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	firstID := catalog.Resources[0].Item.GameID
	catalog.Resources[1].Item.GameID.Value = firstID.Value
	err = validateGeneratedCatalog(catalog)
	if err == nil ||
		!strings.Contains(err.Error(), "build runtime catalog") ||
		!strings.Contains(err.Error(), "duplicate item game ID") {
		t.Fatalf("validateGeneratedCatalog error = %v", err)
	}
}
