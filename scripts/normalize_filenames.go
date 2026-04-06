package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	dbFiles := []string{"items.go", "weapons.go", "armors.go", "talismans.go", "aows.go"}
	iconPaths := make(map[string]bool)

	// 1. Extract all IconPaths from DB
	reIcon := regexp.MustCompile(`IconPath: "(.*)"`)
	for _, f := range dbFiles {
		path := filepath.Join("backend/db/data", f)
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("⚠️ Error reading %s: %v\n", path, err)
			continue
		}
		matches := reIcon.FindAllStringSubmatch(string(content), -1)
		for _, m := range matches {
			if len(m) == 2 && m[1] != "" {
				iconPaths[m[1]] = true
			}
		}
	}

	fmt.Printf("🔍 Found %d unique IconPaths in DB\n", len(iconPaths))

	// 2. Inventory of existing files
	existingFiles := make(map[string]string) // normalized_name -> full_path
	err := filepath.Walk("frontend/public/items", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".png") {
			return nil
		}
		name := strings.ToLower(filepath.Base(path))
		name = strings.TrimSuffix(name, ".png")
		// Normalize name for matching (remove non-alphanumeric)
		norm := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(name, "")
		existingFiles[norm] = path
		return nil
	})
	if err != nil {
		fmt.Printf("⚠️ Error walking icons: %v\n", err)
		return
	}

	// 3. Rename files to match DB
	renamedCount := 0
	missingCount := 0
	for targetPath := range iconPaths {
		fullTargetPath := filepath.Join("frontend/public", targetPath)

		// If file already exists at target path, skip
		if _, err := os.Stat(fullTargetPath); err == nil {
			continue
		}

		// Try to find a match among existing files
		targetName := strings.ToLower(filepath.Base(targetPath))
		targetName = strings.TrimSuffix(targetName, ".png")
		normTarget := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(targetName, "")

		if sourcePath, ok := existingFiles[normTarget]; ok {
			// Ensure target directory exists
			os.MkdirAll(filepath.Dir(fullTargetPath), 0755)

			fmt.Printf("🚚 Renaming: %s -> %s\n", sourcePath, fullTargetPath)
			err := os.Rename(sourcePath, fullTargetPath)
			if err != nil {
				fmt.Printf("❌ Error renaming %s: %v\n", sourcePath, err)
			} else {
				renamedCount++
				// Update existingFiles map to avoid double renaming
				delete(existingFiles, normTarget)
			}
		} else {
			missingCount++
		}
	}

	fmt.Printf("\n✅ Done!\n")
	fmt.Printf("📊 Renamed: %d\n", renamedCount)
	fmt.Printf("📊 Missing: %d\n", missingCount)
}
