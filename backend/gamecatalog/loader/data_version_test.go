package loader_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"sort"
	"strings"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
)

// TestManifestDataVersionMatchesTheShippedTree keeps manifest.dataVersion honest
// now that the class documents are maintained by hand instead of regenerated.
// Editing a document without recomputing the fingerprint leaves a manifest that
// claims a data set the tree no longer contains, and nothing else in the project
// would notice.
//
// It is a check, never a generator: it reads the shipped tree, computes the
// fingerprint and compares it with the value the manifest carries. It writes
// nothing and has no expected constant of its own, so the manifest stays the
// single place the fingerprint is stated.
func TestManifestDataVersionMatchesTheShippedTree(t *testing.T) {
	catalogFS := catalogdata.Files()
	data, err := loader.LoadFS(catalogFS)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}

	computed, err := computeDataVersion(data, catalogFS)
	if err != nil {
		t.Fatalf("compute dataVersion: %v", err)
	}
	if computed != data.Manifest.DataVersion {
		t.Errorf("manifest.dataVersion is stale:\n stored   = %s\n computed = %s\n"+
			"the catalog data changed without the fingerprint being recomputed",
			data.Manifest.DataVersion, computed)
	}
}

// TestDataVersionIsDeterministic proves the fingerprint depends on the tree
// alone: two runs over the same data agree, so a failure of the check above is
// always a data change and never map-iteration order.
func TestDataVersionIsDeterministic(t *testing.T) {
	catalogFS := catalogdata.Files()
	data, err := loader.LoadFS(catalogFS)
	if err != nil {
		t.Fatalf("LoadFS: %v", err)
	}
	first, err := computeDataVersion(data, catalogFS)
	if err != nil {
		t.Fatalf("compute dataVersion: %v", err)
	}
	second, err := computeDataVersion(data, catalogFS)
	if err != nil {
		t.Fatalf("recompute dataVersion: %v", err)
	}
	if first != second {
		t.Errorf("dataVersion is not deterministic: %s then %s", first, second)
	}
}

// computeDataVersion is the fingerprint rule of the shipped catalog tree:
//
//	SHA-256 over
//	  the manifest with an empty dataVersion, marshalled through schema.Manifest,
//	  followed by 0x00;
//	  every resource in the catalog.json document order, marshalled through
//	  schema.Resource, each followed by 0x00;
//	  every file under assets/ in lexicographic path order as the path, 0x00, the
//	  raw 32-byte SHA-256 of its content, 0x00.
//
// The manifest carries the result, so it is excluded from its own input.
func computeDataVersion(data loader.Data, catalogFS fs.FS) (string, error) {
	digest := sha256.New()

	manifest := data.Manifest
	manifest.DataVersion = ""
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	digest.Write(encoded)
	digest.Write([]byte{0})

	for _, document := range data.Documents {
		encoded, err := json.Marshal(document.Resource)
		if err != nil {
			return "", err
		}
		digest.Write(encoded)
		digest.Write([]byte{0})
	}

	assetPaths, err := shippedAssetPaths(catalogFS)
	if err != nil {
		return "", err
	}
	for _, assetPath := range assetPaths {
		content, err := fs.ReadFile(catalogFS, assetPath)
		if err != nil {
			return "", err
		}
		assetDigest := sha256.Sum256(content)
		digest.Write([]byte(assetPath))
		digest.Write([]byte{0})
		digest.Write(assetDigest[:])
		digest.Write([]byte{0})
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}

// shippedAssetPaths lists every file the catalog ships under assets/, sorted, so
// the fingerprint covers the icons and the appearance images alike and does not
// depend on which of them a resource happens to reference.
func shippedAssetPaths(catalogFS fs.FS) ([]string, error) {
	var paths []string
	err := fs.WalkDir(catalogFS, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasPrefix(path, "assets/") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}
