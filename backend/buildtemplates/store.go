package buildtemplates

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// IndexFileName is the on-disk metadata file name.
const IndexFileName = "_index.json"

// IndexVersion is the supported schema version of _index.json.
const IndexVersion = 1

// TemplateMetadata represents the lightweight metadata of a build template in
// the library index. It intentionally omits Filename, RootDir and any system paths.
type TemplateMetadata struct {
	TemplateID       string   `json:"templateID"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
	SchemaVersion    int      `json:"schemaVersion,omitempty"`
	SelectedSections []string `json:"selectedSections,omitempty"`
	InventoryItems   int      `json:"inventoryItems"`
	StorageItems     int      `json:"storageItems"`
	Warnings         int      `json:"warnings"`
}

// indexEntry is the on-disk JSON structure of an entry in _index.json.
type indexEntry struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Filename         string   `json:"filename"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
	InventoryItems   int      `json:"inventoryItems"`
	StorageItems     int      `json:"storageItems"`
	Warnings         int      `json:"warnings"`
	Version          int      `json:"version,omitempty"`
	SelectedSections []string `json:"selectedSections,omitempty"`
}

// indexFile is the on-disk JSON structure of _index.json.
type indexFile struct {
	Version int          `json:"version"`
	Entries []indexEntry `json:"entries"`
}

// Store provides read-only access to a local Build Templates library.
type Store struct {
	dir string
}

// NewStore returns a new Store reading from the specified directory.
func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// DefaultDirectory returns the canonical templates library directory
// under os.UserConfigDir()/EldenRing-SaveEditor/templates.
func DefaultDirectory() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(configDir, "EldenRing-SaveEditor", "templates"), nil
}

// NewDefaultStore returns a Store pointing to DefaultDirectory.
func NewDefaultStore() (*Store, error) {
	dir, err := DefaultDirectory()
	if err != nil {
		return nil, err
	}
	return NewStore(dir), nil
}

// ListTemplates reads the index file from the store directory and returns
// template metadata sorted by UpdatedAt descending, tie-broken by TemplateID ascending.
// If the directory or _index.json does not exist, an empty slice and nil error are returned.
// If _index.json is malformed or declares an unsupported version, an error is returned fail-closed.
func (s *Store) ListTemplates() ([]TemplateMetadata, error) {
	if s == nil {
		return nil, errors.New("store is nil")
	}
	indexPath := filepath.Join(s.dir, IndexFileName)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []TemplateMetadata{}, nil
		}
		return nil, fmt.Errorf("read index: %w", err)
	}

	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("unmarshal index: %w", err)
	}
	if idx.Version != IndexVersion {
		return nil, fmt.Errorf("unsupported index version %d; expected %d", idx.Version, IndexVersion)
	}

	entries := make([]TemplateMetadata, 0, len(idx.Entries))
	for _, e := range idx.Entries {
		var tags []string
		if len(e.Tags) > 0 {
			tags = append([]string(nil), e.Tags...)
		}
		var selectedSections []string
		if len(e.SelectedSections) > 0 {
			selectedSections = append([]string(nil), e.SelectedSections...)
		}
		entries = append(entries, TemplateMetadata{
			TemplateID:       e.ID,
			Name:             e.Name,
			Description:      e.Description,
			Tags:             tags,
			CreatedAt:        e.CreatedAt,
			UpdatedAt:        e.UpdatedAt,
			SchemaVersion:    e.Version,
			SelectedSections: selectedSections,
			InventoryItems:   e.InventoryItems,
			StorageItems:     e.StorageItems,
			Warnings:         e.Warnings,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].UpdatedAt != entries[j].UpdatedAt {
			return entries[i].UpdatedAt > entries[j].UpdatedAt
		}
		return entries[i].TemplateID < entries[j].TemplateID
	})

	return entries, nil
}
