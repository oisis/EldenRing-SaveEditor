package buildtemplates

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// IndexFileName is the on-disk metadata file name.
const IndexFileName = "_index.json"

// IndexVersion is the supported schema version of _index.json.
const IndexVersion = 1

// ErrNotFound indicates that the requested template ID does not exist in the store index.
var ErrNotFound = errors.New("template not found")

// ErrStaleRevision indicates that the supplied templateRevision no longer
// matches the library entry, so the caller acted on a stale view of the
// template. Nothing was written.
var ErrStaleRevision = errors.New("stale template revision")

func withoutPath(err error) error {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}

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
	TemplateRevision string   `json:"templateRevision"`
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
	Revision         uint64   `json:"revision,omitempty"`
}

// indexFile is the on-disk JSON structure of _index.json.
type indexFile struct {
	Version int          `json:"version"`
	Entries []indexEntry `json:"entries"`
}

// formatRevision renders the persistent per-template revision counter as the
// canonical decimal templateRevision token. An index entry written before the
// counter existed carries no revision field and therefore reports "0".
func formatRevision(revision uint64) string {
	return strconv.FormatUint(revision, 10)
}

// validateRevisionToken accepts only the canonical decimal form a caller can
// have received from formatRevision. It round-trips the parse, so a leading
// sign, padding, whitespace, a fractional part, or a value beyond uint64 is
// rejected before any file is touched.
func validateRevisionToken(token string) error {
	value, err := strconv.ParseUint(token, 10, 64)
	if err != nil || formatRevision(value) != token {
		return fmt.Errorf("templateRevision %q is not a canonical decimal revision token", token)
	}
	return nil
}

// validateEntryFilename enforces that an index entry names a plain file inside
// the store directory.
func validateEntryFilename(templateID string, filename string) error {
	if filename == "" {
		return fmt.Errorf("template %q index entry has empty filename", templateID)
	}
	if filename == "." || filename == ".." ||
		filepath.IsAbs(filename) || filepath.Clean(filename) != filename ||
		strings.ContainsAny(filename, `/\`) {
		return fmt.Errorf("template %q index entry has invalid filename %q", templateID, filename)
	}
	return nil
}

// decodeIndexForWrite parses _index.json bytes fail-closed for mutations:
// it disallows unknown fields and enforces that the data contains exactly one
// JSON document. This prevents a writer from silently dropping fields added by
// a newer version or external tool.
func decodeIndexForWrite(data []byte) (indexFile, error) {
	var idx indexFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&idx); err != nil {
		return indexFile{}, fmt.Errorf("unmarshal index: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return indexFile{}, errors.New("unmarshal index: trailing data after index JSON")
	}
	return idx, nil
}

// validateIndexForWrite enforces the writer-side invariants of a complete
// index: a supported version, unique template IDs, and one safe filename owned
// by exactly one entry. A writer must refuse to rewrite an index it cannot
// fully account for. The getters keep their own, narrower rules, so a library
// that only ever gets read is unaffected by this stricter check.
func validateIndexForWrite(idx indexFile) error {
	if idx.Version != IndexVersion {
		return fmt.Errorf("unsupported index version %d; expected %d", idx.Version, IndexVersion)
	}
	seenIDs := make(map[string]bool, len(idx.Entries))
	seenFilenames := make(map[string]bool, len(idx.Entries))
	for _, e := range idx.Entries {
		if seenIDs[e.ID] {
			return fmt.Errorf("index contains duplicate template ID %q", e.ID)
		}
		seenIDs[e.ID] = true
		if err := validateEntryFilename(e.ID, e.Filename); err != nil {
			return err
		}
		if seenFilenames[e.Filename] {
			return fmt.Errorf("index entry %q shares filename %q with another entry", e.ID, e.Filename)
		}
		seenFilenames[e.Filename] = true
	}
	return nil
}

// Store provides access to a local Build Templates library. One instance owns
// one directory; mu serialises every read and write it performs there.
// ponytail: a single in-process mutex per Store, no lockfile. The library is
// local and single-user; add inter-process locking only if a second writer ever
// exists.
type Store struct {
	dir string
	mu  sync.Mutex
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
	s.mu.Lock()
	defer s.mu.Unlock()

	indexPath := filepath.Join(s.dir, IndexFileName)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []TemplateMetadata{}, nil
		}
		return nil, fmt.Errorf("read index: %w", withoutPath(err))
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
			TemplateRevision: formatRevision(e.Revision),
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

// GetTemplate loads, decodes, and validates a build template by templateID.
// It resolves the template only via _index.json and rejects symlinks or paths outside the store.
func (s *Store) GetTemplate(templateID string) (*BuildTemplate, string, error) {
	if s == nil {
		return nil, "", errors.New("store is nil")
	}
	if templateID == "" {
		return nil, "", errors.New("templateID must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	indexPath := filepath.Join(s.dir, IndexFileName)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", fmt.Errorf("template %q: %w", templateID, ErrNotFound)
		}
		return nil, "", fmt.Errorf("read index: %w", withoutPath(err))
	}

	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, "", fmt.Errorf("unmarshal index: %w", err)
	}
	if idx.Version != IndexVersion {
		return nil, "", fmt.Errorf("unsupported index version %d; expected %d", idx.Version, IndexVersion)
	}

	seenIDs := make(map[string]bool, len(idx.Entries))
	var matched *indexEntry
	for i := range idx.Entries {
		e := &idx.Entries[i]
		if seenIDs[e.ID] {
			return nil, "", fmt.Errorf("index contains duplicate template ID %q", e.ID)
		}
		seenIDs[e.ID] = true
		if e.ID == templateID {
			matched = e
		}
	}
	if matched == nil {
		return nil, "", fmt.Errorf("template %q: %w", templateID, ErrNotFound)
	}

	if err := validateEntryFilename(templateID, matched.Filename); err != nil {
		return nil, "", err
	}

	targetPath := filepath.Join(s.dir, matched.Filename)

	evalTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", fmt.Errorf("template payload not found for %q: %w", templateID, ErrNotFound)
		}
		return nil, "", fmt.Errorf("resolve payload for template %q: %w", templateID, withoutPath(err))
	}
	evalDir, err := filepath.EvalSymlinks(s.dir)
	if err != nil {
		return nil, "", fmt.Errorf("resolve template store: %w", withoutPath(err))
	}
	rel, err := filepath.Rel(evalDir, evalTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return nil, "", fmt.Errorf("template %q target escapes store directory", templateID)
	}

	payload, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, "", fmt.Errorf("read template payload for %q: %w", templateID, withoutPath(err))
	}

	tpl, err := DecodeTemplate(payload)
	if err != nil {
		return nil, "", fmt.Errorf("template %q: %w", templateID, err)
	}

	if matched.Version != 0 && tpl.Version != matched.Version {
		return nil, "", fmt.Errorf("template %q schema version mismatch: index=%d payload=%d", templateID, matched.Version, tpl.Version)
	}
	payloadName := ""
	payloadDescription := ""
	var payloadTags []string
	if tpl.Metadata != nil {
		payloadName = tpl.Metadata.Name
		payloadDescription = tpl.Metadata.Description
		payloadTags = tpl.Metadata.Tags
	}
	if matched.Name != payloadName || matched.Description != payloadDescription || !slices.Equal(matched.Tags, payloadTags) {
		return nil, "", fmt.Errorf("template %q metadata mismatch with its index entry", templateID)
	}

	return tpl, formatRevision(matched.Revision), nil
}

// DeleteTemplate removes one template from the local library: it drops the
// entry from _index.json and then unlinks the payload the entry pointed at.
//
// templateRevision must be the canonical decimal token the getters reported for
// this templateID, which is "0" for an entry written before the revision
// counter existed. A token that no longer matches the entry yields
// ErrStaleRevision and writes nothing; Delete never advances the counter.
//
// The index is decoded strictly (disallowing unknown fields) and validated
// before writing. The index is committed first, so an interrupted delete can
// leave an unreferenced payload behind but can never leave an entry pointing at
// a file that is already gone.
func (s *Store) DeleteTemplate(templateID string, templateRevision string) error {
	if s == nil {
		return errors.New("store is nil")
	}
	if templateID == "" {
		return errors.New("templateID must not be empty")
	}
	if err := validateRevisionToken(templateRevision); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(filepath.Join(s.dir, IndexFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("template %q: %w", templateID, ErrNotFound)
		}
		return fmt.Errorf("read index: %w", withoutPath(err))
	}
	idx, err := decodeIndexForWrite(data)
	if err != nil {
		return err
	}
	if err := validateIndexForWrite(idx); err != nil {
		return err
	}

	matched := -1
	for i := range idx.Entries {
		if idx.Entries[i].ID == templateID {
			matched = i
			break
		}
	}
	if matched < 0 {
		return fmt.Errorf("template %q: %w", templateID, ErrNotFound)
	}
	if formatRevision(idx.Entries[matched].Revision) != templateRevision {
		return fmt.Errorf("template %q: %w", templateID, ErrStaleRevision)
	}
	filename := idx.Entries[matched].Filename

	remaining := make([]indexEntry, 0, len(idx.Entries)-1)
	remaining = append(remaining, idx.Entries[:matched]...)
	remaining = append(remaining, idx.Entries[matched+1:]...)
	next := indexFile{Version: idx.Version, Entries: remaining}

	encoded, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode index: %w", err)
	}
	// Re-read what is about to hit the disk, so a serialisation defect is caught
	// while the old index is still the live one.
	verify, err := decodeIndexForWrite(encoded)
	if err != nil {
		return fmt.Errorf("verify new index: %w", err)
	}
	if err := validateIndexForWrite(verify); err != nil {
		return fmt.Errorf("verify new index: %w", err)
	}
	for _, e := range verify.Entries {
		if e.ID == templateID {
			return fmt.Errorf("verify new index: template %q still present", templateID)
		}
	}

	if err := s.writeIndexAtomic(encoded); err != nil {
		return err
	}

	// The index no longer references this payload, so the delete is already
	// committed and must not be reported as retryable. A failed unlink leaves an
	// invisible orphan file behind; nothing cleans it up.
	_ = os.Remove(filepath.Join(s.dir, filename))
	return nil
}

// writeIndexAtomic replaces _index.json through a temporary file in the same
// directory, so a crash leaves either the complete old index or the complete
// new one.
// ponytail: the file is synced but the directory is not, so a power loss can
// still lose the rename itself. Add a directory fsync only if the library ever
// needs a crash-durability guarantee.
func (s *Store) writeIndexAtomic(encoded []byte) error {
	tmp, err := os.CreateTemp(s.dir, "._index-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary index: %w", withoutPath(err))
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary index: %w", withoutPath(err))
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary index: %w", withoutPath(err))
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary index: %w", withoutPath(err))
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		return fmt.Errorf("set index permissions: %w", withoutPath(err))
	}
	if err := os.Rename(tmpName, filepath.Join(s.dir, IndexFileName)); err != nil {
		return fmt.Errorf("replace index: %w", withoutPath(err))
	}
	return nil
}
