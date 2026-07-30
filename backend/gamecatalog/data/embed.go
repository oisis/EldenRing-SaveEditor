package catalogdata

import (
	"embed"
	"io/fs"
)

//go:embed catalog.json items/*/*.json
var catalogFiles embed.FS

func Files() fs.FS {
	return catalogFiles
}
