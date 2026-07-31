package loader

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"io/fs"
	"path"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const iconAssetPrefix = "assets/icons/items/"

const (
	iconMediaTypePNG  = "image/png"
	iconMediaTypeWebP = "image/webp"
)

var (
	pngSignature  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	webpContainer = []byte("RIFF")
	webpFormat    = []byte("WEBP")
)

func loadKnownIconAssets(catalogFS fs.FS, resource schema.Resource, assets map[string]string) error {
	if resource.Item == nil {
		return nil
	}

	if resource.Item.Presentation.IconPath.Known {
		if err := loadIconAsset(catalogFS, resource.Item.Presentation.IconPath.Value, assets); err != nil {
			return fmt.Errorf("canonical icon: %w", err)
		}
	}
	for _, variant := range resource.Item.Variants {
		if !variant.Data.Presentation.IconPath.Known {
			continue
		}
		if err := loadIconAsset(catalogFS, variant.Data.Presentation.IconPath.Value, assets); err != nil {
			return fmt.Errorf("variant 0x%08X icon: %w", variant.GameID.Value, err)
		}
	}
	return nil
}

func loadIconAsset(catalogFS fs.FS, iconPath string, assets map[string]string) error {
	if err := validateIconAssetPath(iconPath); err != nil {
		return err
	}
	if _, exists := assets[iconPath]; exists {
		return nil
	}
	icon, err := catalogFS.Open(iconPath)
	if err != nil {
		return fmt.Errorf("read icon asset %q: %w", iconPath, err)
	}
	defer icon.Close()

	info, err := icon.Stat()
	if err != nil {
		return fmt.Errorf("stat icon asset %q: %w", iconPath, err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("icon asset %q is empty", iconPath)
	}
	reader := bufio.NewReader(icon)
	mediaType, err := iconMediaType(reader)
	if err != nil {
		return fmt.Errorf("decode icon asset %q: %w", iconPath, err)
	}
	assets[iconPath] = mediaType
	return nil
}

func iconMediaType(reader *bufio.Reader) (string, error) {
	header, err := reader.Peek(12)
	if err != nil {
		return "", fmt.Errorf("read signature: %w", err)
	}
	switch {
	case bytes.Equal(header[:len(pngSignature)], pngSignature):
		if _, err := png.DecodeConfig(reader); err != nil {
			return "", err
		}
		return iconMediaTypePNG, nil
	case bytes.Equal(header[:4], webpContainer) && bytes.Equal(header[8:12], webpFormat):
		return iconMediaTypeWebP, nil
	default:
		return "", errors.New("unsupported image signature")
	}
}

func validateIconAssetPath(iconPath string) error {
	if !fs.ValidPath(iconPath) {
		return fmt.Errorf("icon path %q must be a relative, slash-separated path inside the catalog", iconPath)
	}
	if !strings.HasPrefix(iconPath, iconAssetPrefix) {
		return fmt.Errorf("icon path %q must be inside %s", iconPath, iconAssetPrefix)
	}
	if path.Ext(iconPath) != ".png" {
		return fmt.Errorf("icon path %q must use the legacy .png catalog asset suffix", iconPath)
	}
	return nil
}
