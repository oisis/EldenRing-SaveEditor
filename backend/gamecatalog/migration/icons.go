package migration

import (
	"path"
	"strings"
)

func collectIconSources(items []seed) map[string]string {
	result := make(map[string]string)
	for _, item := range items {
		if item.IconPath == "" {
			continue
		}
		source := path.Clean(item.IconPath)
		if source == "." || source == ".." || strings.HasPrefix(source, "../") ||
			!strings.HasPrefix(source, "items/") {
			continue
		}
		result["assets/icons/"+source] = source
	}
	return result
}
