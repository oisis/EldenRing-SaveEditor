package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func computeCatalogDataVersion(
	manifest schema.Manifest,
	resources []schema.Resource,
	iconSources map[string]string,
) (string, error) {
	manifest.DataVersion = ""
	sum := sha256.New()
	if err := writeCanonicalHashValue(sum, manifest); err != nil {
		return "", fmt.Errorf("hash manifest: %w", err)
	}
	for _, resource := range resources {
		if err := writeCanonicalHashValue(sum, resource); err != nil {
			return "", fmt.Errorf("hash resource %q: %w", resource.Key, err)
		}
	}
	destinations := make([]string, 0, len(iconSources))
	for destination := range iconSources {
		destinations = append(destinations, destination)
	}
	sort.Strings(destinations)
	for _, destination := range destinations {
		sum.Write([]byte(destination))
		sum.Write([]byte{0})
		sum.Write([]byte(iconSources[destination]))
		sum.Write([]byte{0})
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func writeCanonicalHashValue(
	sum interface{ Write([]byte) (int, error) },
	value any,
) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := sum.Write(raw); err != nil {
		return err
	}
	_, err = sum.Write([]byte{0})
	return err
}
