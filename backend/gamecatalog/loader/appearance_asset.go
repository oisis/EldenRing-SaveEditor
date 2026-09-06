package loader

import (
	"bytes"
	"fmt"
	"io/fs"
)

// appearanceAssetPrefix is the catalog-relative directory that holds the
// preview image of every appearance preset. The presets themselves are a
// separate data set, so the assets are registered from the authored directory
// instead of from a resource document.
const appearanceAssetPrefix = "assets/appearance/"

const appearanceMediaTypeJPEG = "image/jpeg"

// jpegSignature is the SOI marker every JFIF/EXIF preview image starts with.
var jpegSignature = []byte{0xFF, 0xD8, 0xFF}

// loadAppearanceAssets registers the appearance preview images of the catalog
// filesystem. A catalog without the directory registers nothing; a file that is
// not a JPEG is a data defect and stops loading rather than being served with a
// wrong media type.
func loadAppearanceAssets(catalogFS fs.FS, assets map[string]string) error {
	matches, err := fs.Glob(catalogFS, appearanceAssetPrefix+"*.jpg")
	if err != nil {
		return fmt.Errorf("list appearance assets: %w", err)
	}
	for _, assetPath := range matches {
		content, err := fs.ReadFile(catalogFS, assetPath)
		if err != nil {
			return fmt.Errorf("read appearance asset %q: %w", assetPath, err)
		}
		if !bytes.HasPrefix(content, jpegSignature) {
			return fmt.Errorf("appearance asset %q: unsupported image signature", assetPath)
		}
		assets[assetPath] = appearanceMediaTypeJPEG
	}
	return nil
}
