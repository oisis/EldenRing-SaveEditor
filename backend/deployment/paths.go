package deployment

import "path/filepath"

// localJoin joins a local target path in the host filesystem's own syntax.
func localJoin(directory string, name string) string { return filepath.Join(directory, name) }
