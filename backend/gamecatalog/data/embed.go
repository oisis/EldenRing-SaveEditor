package catalogdata

import (
	"embed"
	"io/fs"
)

//go:embed catalog.json items/*/*.json colosseums/*.json regulation/*.json presets/*.json assets/icons/items/*/*.png assets/appearance/*.jpg
var catalogFiles embed.FS

func Files() fs.FS {
	return catalogFiles
}
