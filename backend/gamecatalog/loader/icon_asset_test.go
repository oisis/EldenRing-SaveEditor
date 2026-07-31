package loader_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

func TestLoadFSRejectsMissingKnownIconAsset(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	delete(catalogFS, "assets/icons/items/melee_armaments/dagger.png")

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "read icon asset") {
		t.Fatalf("LoadFS error = %v, want missing icon rejection", err)
	}
}

func TestLoadFSRejectsEscapingIconPath(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	documentPath := "items/weapon/000f4240.json"
	content := string(catalogFS[documentPath].Data)
	content = strings.Replace(
		content,
		"assets/icons/items/melee_armaments/dagger.png",
		"../frontend/public/items/melee_armaments/dagger.png",
		1,
	)
	catalogFS[documentPath].Data = []byte(content)

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "must be a relative, slash-separated path inside the catalog") {
		t.Fatalf("LoadFS error = %v, want unsafe icon path rejection", err)
	}
}

func TestLoadFSRejectsIconOutsideDedicatedAssetDirectory(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	documentPath := "items/weapon/000f4240.json"
	content := string(catalogFS[documentPath].Data)
	content = strings.Replace(
		content,
		"assets/icons/items/melee_armaments/dagger.png",
		"other/icons/000f4240.png",
		1,
	)
	catalogFS[documentPath].Data = []byte(content)
	catalogFS["other/icons/000f4240.png"] = catalogFS["assets/icons/items/melee_armaments/dagger.png"]

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "must be inside assets/icons/items/") {
		t.Fatalf("LoadFS error = %v, want icon directory rejection", err)
	}
}

func TestLoadFSRejectsUnsupportedIconAssetSuffix(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	documentPath := "items/weapon/000f4240.json"
	content := string(catalogFS[documentPath].Data)
	content = strings.Replace(
		content,
		"assets/icons/items/melee_armaments/dagger.png",
		"assets/icons/items/weapon/000f4240.webp",
		1,
	)
	catalogFS[documentPath].Data = []byte(content)

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "must use the legacy .png catalog asset suffix") {
		t.Fatalf("LoadFS error = %v, want legacy suffix rejection", err)
	}
}

func TestLoadFSRejectsInvalidPNGIcon(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	catalogFS["assets/icons/items/melee_armaments/dagger.png"].Data = []byte("not a PNG")

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "decode icon asset") {
		t.Fatalf("LoadFS error = %v, want invalid PNG rejection", err)
	}
}

func TestLoadFSRejectsEmptyPNGIcon(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	catalogFS["assets/icons/items/melee_armaments/dagger.png"].Data = nil

	_, err := loader.LoadFS(catalogFS)
	if err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("LoadFS error = %v, want empty icon rejection", err)
	}
}

func TestLoadFSAllowsWebPBytesAtLegacyPNGPath(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	iconPath := "assets/icons/items/melee_armaments/dagger.png"
	webp := []byte{'R', 'I', 'F', 'F', 4, 0, 0, 0, 'W', 'E', 'B', 'P'}
	catalogFS[iconPath].Data = append([]byte(nil), webp...)

	data, err := loader.LoadFS(catalogFS)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	content, mediaType, exists := data.ReadAssetWithMediaType(iconPath)
	if !exists || mediaType != "image/webp" || !bytes.Equal(content, webp) {
		t.Fatalf("WebP asset = %d bytes, %q, %t", len(content), mediaType, exists)
	}
}

func TestLoadFSLoadsSharedCanonicalIconForFullVariants(t *testing.T) {
	catalogFS := embeddedCatalogFS(t)
	data, err := loader.LoadFS(catalogFS)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	iconPath := "assets/icons/items/melee_armaments/dagger.png"
	if content, exists := data.ReadAsset(iconPath); !exists || len(content) == 0 {
		t.Fatalf("ReadAsset(%q) = %d bytes, %t", iconPath, len(content), exists)
	}
}
