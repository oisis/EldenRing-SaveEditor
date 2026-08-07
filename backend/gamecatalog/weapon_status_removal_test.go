package gamecatalog_test

import (
	"io/fs"
	"strings"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
)

func TestEmbeddedDocumentsHaveNoRemovedWeaponStatusFields(t *testing.T) {
	removed := []string{
		`"statusPoison"`,
		`"statusBleed"`,
		`"statusFrost"`,
		`"statusSleep"`,
		`"statusMadness"`,
		`"statusScarletRot"`,
	}

	err := fs.WalkDir(catalogdata.Files(), "items", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		raw, err := fs.ReadFile(catalogdata.Files(), path)
		if err != nil {
			return err
		}
		for _, field := range removed {
			if strings.Contains(string(raw), field) {
				t.Errorf("%s still contains %s", path, field)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded documents: %v", err)
	}
}
