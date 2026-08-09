package catalogdata

import (
	"embed"
	"io/fs"
)

//go:embed catalog.json items/*/*.json regulation/*.json assets/icons/items/*/*.png
var catalogFiles embed.FS

func Files() fs.FS {
	return catalogFiles
}
