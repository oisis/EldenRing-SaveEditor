package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type migrationSourceVersions struct {
	legacyData  string
	legacyIcons string
}

func hashMigrationSources(
	options GenerateOptions,
	iconSources map[string]string,
	snapshot legacySnapshot,
) (migrationSourceVersions, error) {
	if strings.TrimSpace(options.LegacyIconDirectory) == "" {
		return migrationSourceVersions{}, fmt.Errorf("legacy icon directory is required")
	}

	legacyData, err := hashLegacySnapshot(snapshot)
	if err != nil {
		return migrationSourceVersions{}, err
	}
	iconPaths := make([]string, 0, len(iconSources))
	for _, sourcePath := range iconSources {
		iconPaths = append(iconPaths, sourcePath)
	}
	sort.Strings(iconPaths)
	legacyIcons, err := hashRelativeFiles(options.LegacyIconDirectory, iconPaths)
	if err != nil {
		return migrationSourceVersions{}, fmt.Errorf("hash legacy icons: %w", err)
	}

	return migrationSourceVersions{
		legacyData:  legacyData,
		legacyIcons: legacyIcons,
	}, nil
}

func hashLegacySnapshot(snapshot legacySnapshot) (string, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf("marshal legacy snapshot: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func hashRelativeFiles(root string, paths []string) (string, error) {
	sum := sha256.New()
	seen := make(map[string]struct{}, len(paths))
	for _, relative := range paths {
		clean := filepath.ToSlash(filepath.Clean(relative))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(relative) {
			return "", fmt.Errorf("unsafe relative source path %q", relative)
		}
		if _, duplicate := seen[clean]; duplicate {
			continue
		}
		seen[clean] = struct{}{}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(clean)))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", clean, err)
		}
		sum.Write([]byte(clean))
		sum.Write([]byte{0})
		sum.Write(content)
		sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}
