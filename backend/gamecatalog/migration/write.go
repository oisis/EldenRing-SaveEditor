package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type WriteOptions struct {
	OutputDirectory     string
	LegacyIconDirectory string
	Replace             bool
}

type catalogIndex struct {
	Manifest  schema.Manifest `json:"manifest"`
	Documents []string        `json:"documents"`
}

func WriteCatalog(catalog GeneratedCatalog, options WriteOptions) error {
	if strings.TrimSpace(options.OutputDirectory) == "" {
		return fmt.Errorf("output directory is required")
	}
	if strings.TrimSpace(options.LegacyIconDirectory) == "" {
		return fmt.Errorf("legacy icon directory is required")
	}
	parent := filepath.Dir(options.OutputDirectory)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create output parent: %w", err)
	}
	stage, err := os.MkdirTemp(parent, ".gamecatalog-migration-")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	documents, err := writeStagedDocuments(stage, catalog)
	if err != nil {
		return err
	}
	if err := writeStagedCatalogIndex(stage, catalog, documents); err != nil {
		return err
	}
	if err := copyStagedIcons(stage, options.LegacyIconDirectory, catalog.IconSources); err != nil {
		return err
	}
	if err := validateStagedCatalog(stage); err != nil {
		return err
	}
	if err := validateStagedIconVersion(stage, catalog); err != nil {
		return err
	}
	return installStagedCatalog(stage, options.OutputDirectory, options.Replace)
}

func validateStagedCatalog(stage string) error {
	data, err := loader.LoadDir(stage)
	if err != nil {
		return fmt.Errorf("validate staged catalog files: %w", err)
	}
	if _, err := gamecatalog.New(data.Manifest, data.Resources()); err != nil {
		return fmt.Errorf("validate staged catalog model: %w", err)
	}
	return nil
}

func validateStagedIconVersion(
	stage string,
	catalog GeneratedCatalog,
) error {
	expected := ""
	for _, source := range catalog.Manifest.Sources {
		if source.ID == sourceLegacyIcons {
			expected = source.Version
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("manifest has no legacy icon source version")
	}
	actual, err := hashStagedIcons(stage, catalog.IconSources)
	if err != nil {
		return fmt.Errorf("hash staged icons: %w", err)
	}
	if actual != expected {
		return fmt.Errorf(
			"staged icon source version %s differs from generated version %s",
			actual,
			expected,
		)
	}
	return nil
}

func hashStagedIcons(
	stage string,
	iconSources map[string]string,
) (string, error) {
	destinationsBySource := make(map[string]string, len(iconSources))
	sources := make([]string, 0, len(iconSources))
	for destination, source := range iconSources {
		cleanSource := filepath.ToSlash(filepath.Clean(source))
		cleanDestination := filepath.ToSlash(filepath.Clean(destination))
		if cleanSource == "." ||
			cleanSource == ".." ||
			strings.HasPrefix(cleanSource, "../") ||
			filepath.IsAbs(source) {
			return "", fmt.Errorf("unsafe icon source path %q", source)
		}
		if cleanDestination == "." ||
			cleanDestination == ".." ||
			strings.HasPrefix(cleanDestination, "../") ||
			filepath.IsAbs(destination) {
			return "", fmt.Errorf("unsafe icon destination path %q", destination)
		}
		if previous, duplicate := destinationsBySource[cleanSource]; duplicate {
			if previous != cleanDestination {
				return "", fmt.Errorf(
					"icon source %q maps to multiple destinations",
					cleanSource,
				)
			}
			continue
		}
		destinationsBySource[cleanSource] = cleanDestination
		sources = append(sources, cleanSource)
	}
	sort.Strings(sources)

	sum := sha256.New()
	for _, source := range sources {
		content, err := os.ReadFile(filepath.Join(
			stage,
			filepath.FromSlash(destinationsBySource[source]),
		))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", source, err)
		}
		sum.Write([]byte(source))
		sum.Write([]byte{0})
		sum.Write(content)
		sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func writeStagedDocuments(
	stage string,
	catalog GeneratedCatalog,
) ([]string, error) {
	documents := make([]string, 0, len(catalog.Resources))
	for _, resource := range catalog.Resources {
		if resource.Item == nil {
			return nil, fmt.Errorf("resource %q has no item document", resource.Key)
		}
		relative := filepath.ToSlash(filepath.Join(
			"items",
			string(resource.Item.Family.Value),
			fmt.Sprintf("%08x.json", resource.Item.GameID.Value),
		))
		if err := writeJSONFile(filepath.Join(stage, filepath.FromSlash(relative)), resource); err != nil {
			return nil, err
		}
		documents = append(documents, relative)
	}
	sort.Strings(documents)
	return documents, nil
}

func writeStagedCatalogIndex(
	stage string,
	catalog GeneratedCatalog,
	documents []string,
) error {
	return writeJSONFile(filepath.Join(stage, "catalog.json"), catalogIndex{
		Manifest:  catalog.Manifest,
		Documents: documents,
	})
}

func writeJSONFile(filename string, value any) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("create %s parent: %w", filename, err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filename, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filename, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filename, err)
	}
	return nil
}

func copyStagedIcons(
	stage string,
	legacyIconDirectory string,
	iconSources map[string]string,
) error {
	destinations := make([]string, 0, len(iconSources))
	for destination := range iconSources {
		destinations = append(destinations, destination)
	}
	sort.Strings(destinations)
	for _, destination := range destinations {
		source := iconSources[destination]
		if err := copyFile(
			filepath.Join(legacyIconDirectory, filepath.FromSlash(source)),
			filepath.Join(stage, filepath.FromSlash(destination)),
		); err != nil {
			return fmt.Errorf("copy icon %s: %w", source, err)
		}
	}
	return nil
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func installStagedCatalog(stage, output string, replace bool) error {
	managed := []string{"catalog.json", "items", filepath.Join("assets", "icons", "items")}
	if !replace {
		for _, relative := range managed {
			if _, err := os.Stat(filepath.Join(output, relative)); err == nil {
				return fmt.Errorf(
					"managed output %s already exists; replacement was not authorized",
					relative,
				)
			} else if !os.IsNotExist(err) {
				return err
			}
		}
	}
	if info, err := os.Stat(output); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("output %s is not a directory", output)
		}
		if err := copyUnmanagedOutput(output, stage); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(output); os.IsNotExist(err) {
		if err := os.Rename(stage, output); err != nil {
			return fmt.Errorf("install catalog: %w", err)
		}
		return nil
	}

	backupRoot, err := os.MkdirTemp(filepath.Dir(output), ".gamecatalog-backup-")
	if err != nil {
		return fmt.Errorf("create catalog backup directory: %w", err)
	}
	defer os.RemoveAll(backupRoot)
	backup := filepath.Join(backupRoot, "catalog")
	if err := os.Rename(output, backup); err != nil {
		return fmt.Errorf("back up existing catalog: %w", err)
	}
	if err := os.Rename(stage, output); err != nil {
		if rollbackErr := os.Rename(backup, output); rollbackErr != nil {
			return fmt.Errorf(
				"install catalog: %w; rollback failed: %v",
				err,
				rollbackErr,
			)
		}
		return fmt.Errorf("install catalog: %w; previous catalog restored", err)
	}
	return nil
}

func copyUnmanagedOutput(sourceRoot, destinationRoot string) error {
	return filepath.WalkDir(sourceRoot, func(source string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, source)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if isManagedOutputPath(relative) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("unmanaged output %s is a symbolic link", relative)
		}
		destination := filepath.Join(destinationRoot, relative)
		if entry.IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return fmt.Errorf("copy unmanaged directory %s: %w", relative, err)
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unmanaged output %s is not a regular file", relative)
		}
		if err := copyFile(source, destination); err != nil {
			return fmt.Errorf("copy unmanaged file %s: %w", relative, err)
		}
		return nil
	})
}

func isManagedOutputPath(relative string) bool {
	clean := filepath.Clean(relative)
	managed := []string{
		"catalog.json",
		"items",
		filepath.Join("assets", "icons", "items"),
	}
	for _, root := range managed {
		if clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
