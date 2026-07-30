package schema

import (
	"fmt"
	"path"
	"strings"
)

func validateSourceLocation(location string) error {
	cleaned := path.Clean(location)
	if path.IsAbs(location) || strings.HasPrefix(location, `\`) {
		return fmt.Errorf("location must not be an absolute local path")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("location must not escape its logical root")
	}
	if cleaned == "tmp" || strings.HasPrefix(cleaned, "tmp/") {
		return fmt.Errorf("location must not reference a temporary working directory")
	}
	return nil
}
