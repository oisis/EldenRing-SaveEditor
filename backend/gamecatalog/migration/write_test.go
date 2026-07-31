package migration

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestWriteCatalogCopiesSharedIconOnceAndIsDeterministic(t *testing.T) {
	legacyRoot := t.TempDir()
	iconSource := filepath.Join(legacyRoot, "items", "shared.png")
	if err := os.MkdirAll(filepath.Dir(iconSource), 0o755); err != nil {
		t.Fatal(err)
	}
	iconBytes := writeTestPNG(t, iconSource)
	output := filepath.Join(t.TempDir(), "catalog")
	manifest, resources := prototype.Data()
	manifest.DataVersion = strings.Repeat("a", 64)
	iconVersion, err := hashRelativeFiles(
		legacyRoot,
		[]string{"items/shared.png"},
	)
	if err != nil {
		t.Fatal(err)
	}
	foundIconSource := false
	for index := range manifest.Sources {
		if manifest.Sources[index].ID == sourceLegacyIcons {
			manifest.Sources[index].Version = iconVersion
			foundIconSource = true
		}
	}
	if !foundIconSource {
		manifest.Sources = append(manifest.Sources, schema.DataSource{
			ID:       sourceLegacyIcons,
			Kind:     "legacy_item_assets",
			Location: "frontend/public/items",
			Version:  iconVersion,
			Evidence: schema.EvidenceCurated,
			Reviewed: true,
		})
	}
	for index := range resources {
		resources[index].Item.Presentation.IconPath.Value =
			"assets/icons/items/shared.png"
		for variantIndex := range resources[index].Item.Variants {
			resources[index].Item.Variants[variantIndex].
				Data.Presentation.IconPath.Value = "assets/icons/items/shared.png"
		}
	}
	catalog := GeneratedCatalog{
		Manifest:  manifest,
		Resources: resources,
		IconSources: map[string]string{
			"assets/icons/items/shared.png": "items/shared.png",
		},
	}
	options := WriteOptions{
		OutputDirectory:     output,
		LegacyIconDirectory: legacyRoot,
	}
	if err := WriteCatalog(catalog, options); err != nil {
		t.Fatalf("first WriteCatalog: %v", err)
	}
	firstIndex := mustReadFile(t, filepath.Join(output, "catalog.json"))
	firstDocument := mustReadFile(t, filepath.Join(output, "items", "weapon", "000f4240.json"))
	copiedIcon := mustReadFile(t, filepath.Join(output, "assets", "icons", "items", "shared.png"))
	if !bytes.Equal(copiedIcon, iconBytes) {
		t.Fatalf("copied icon = %q, want %q", copiedIcon, iconBytes)
	}
	iconFiles := 0
	err = filepath.WalkDir(
		filepath.Join(output, "assets", "icons"),
		func(_ string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() {
				iconFiles++
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if iconFiles != 1 {
		t.Fatalf("copied icon files = %d, want one shared file", iconFiles)
	}
	if err := WriteCatalog(catalog, options); err == nil ||
		!strings.Contains(err.Error(), "replacement was not authorized") {
		t.Fatalf("second WriteCatalog error = %v, want replacement refusal", err)
	}
	unmanagedPath := filepath.Join(output, "embed.go")
	unmanagedContent := []byte("package catalogdata\n")
	if err := os.WriteFile(unmanagedPath, unmanagedContent, 0o644); err != nil {
		t.Fatal(err)
	}
	options.Replace = true

	invalidCatalog := catalog
	invalidCatalog.Resources = append(
		[]schema.Resource(nil),
		catalog.Resources...,
	)
	duplicateItem := *invalidCatalog.Resources[1].Item
	duplicateItem.GameID.Value = invalidCatalog.Resources[0].Item.GameID.Value
	invalidCatalog.Resources[1].Item = &duplicateItem
	if err := WriteCatalog(invalidCatalog, options); err == nil ||
		!strings.Contains(err.Error(), "duplicate item game ID") {
		t.Fatalf("invalid catalog replacement error = %v", err)
	}
	if got := mustReadFile(t, filepath.Join(output, "catalog.json")); !bytes.Equal(got, firstIndex) {
		t.Fatal("catalog.json changed after rejected catalog model")
	}

	if err := os.WriteFile(iconSource, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteCatalog(catalog, options); err == nil ||
		!strings.Contains(err.Error(), "decode icon asset") {
		t.Fatalf("invalid replacement error = %v, want staged icon rejection", err)
	}
	if got := mustReadFile(t, filepath.Join(output, "catalog.json")); !bytes.Equal(got, firstIndex) {
		t.Fatal("catalog.json changed after rejected staged output")
	}
	if got := mustReadFile(t, unmanagedPath); !bytes.Equal(got, unmanagedContent) {
		t.Fatal("unmanaged file changed after rejected staged output")
	}
	if err := os.WriteFile(iconSource, iconBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	differentIcon := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	differentIcon.Set(0, 0, color.NRGBA{R: 0xCC, G: 0x22, B: 0x11, A: 0xFF})
	if err := writePNG(iconSource, differentIcon); err != nil {
		t.Fatal(err)
	}
	if err := WriteCatalog(catalog, options); err == nil ||
		!strings.Contains(err.Error(), "staged icon source version") {
		t.Fatalf("changed valid icon error = %v, want source-version rejection", err)
	}
	if got := mustReadFile(t, filepath.Join(output, "catalog.json")); !bytes.Equal(got, firstIndex) {
		t.Fatal("catalog.json changed after rejected icon source mismatch")
	}
	if err := os.WriteFile(iconSource, iconBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := WriteCatalog(catalog, options); err != nil {
		t.Fatalf("replacement WriteCatalog: %v", err)
	}
	if got := mustReadFile(t, filepath.Join(output, "catalog.json")); !bytes.Equal(got, firstIndex) {
		t.Fatal("catalog.json changed across deterministic replacement")
	}
	if got := mustReadFile(t, filepath.Join(output, "items", "weapon", "000f4240.json")); !bytes.Equal(got, firstDocument) {
		t.Fatal("item document changed across deterministic replacement")
	}
	if got := mustReadFile(t, unmanagedPath); !bytes.Equal(got, unmanagedContent) {
		t.Fatal("replacement did not preserve unmanaged file")
	}
}

func writeTestPNG(t *testing.T, filename string) []byte {
	t.Helper()
	value := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	value.Set(0, 0, color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: 0xFF})
	if err := writePNG(filename, value); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func writePNG(filename string, value image.Image) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	if err := png.Encode(file, value); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func mustReadFile(t *testing.T, filename string) []byte {
	t.Helper()
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
