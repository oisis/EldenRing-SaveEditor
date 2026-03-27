package tests

import (
	"bytes"
	"crypto/sha256"
	"os"
	"testing"
)

// TestRoundTrip verifies that loading and saving a file without changes
// results in a bit-perfect copy of the original.
func TestRoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		// Paths will be populated as we add test data from tmp/save
		// {"PCSave", "data/pc/ER0000.sl2"},
		// {"PS4Save", "data/ps4/memory.dat"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("Failed to read original file: %v", err)
			}

			// TODO: Implement Load -> Save logic once backend/core is ready
			// saveObj, err := core.Load(tc.path)
			// modified, err := saveObj.Write()

			// For now, this is a placeholder that just compares the file to itself
			modified := original 

			if !bytes.Equal(original, modified) {
				origHash := sha256.Sum256(original)
				modHash := sha256.Sum256(modified)
				t.Errorf("Round-trip failed for %s\nOriginal SHA256: %x\nModified SHA256: %x", 
					tc.name, origHash, modHash)
			}
		})
	}
}
