package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	missingFile, err := os.Open("missing_icons.txt")
	if err != nil {
		fmt.Printf("⚠️ Error opening missing_icons.txt: %v\n", err)
		return
	}
	defer missingFile.Close()

	missingIcons := make(map[string]string)
	scanner := bufio.NewScanner(missingFile)
	for scanner.Scan() {
		path := scanner.Text()
		filename := filepath.Base(path)
		nameOnly := strings.TrimSuffix(filename, ".png")
		missingIcons[nameOnly] = path
	}

	fmt.Printf("🔍 Loaded %d missing icons from list.\n", len(missingIcons))

	// Map of normalized name to full local path in tmp/icons
	localIcons := make(map[string]string)
	err = filepath.Walk("tmp/icons", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".png") {
			filename := filepath.Base(path)
			// Pattern: "Item Name - MENU_Knowledge_ID.png" or just "Item Name.png"
			namePart := filename
			if strings.Contains(filename, " - ") {
				namePart = strings.Split(filename, " - ")[0]
			} else {
				namePart = strings.TrimSuffix(filename, filepath.Ext(filename))
			}

			normalized := normalizeName(namePart)
			localIcons[normalized] = path
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️ Error walking tmp/icons: %v\n", err)
		return
	}

	fmt.Printf("📂 Found %d icons in tmp/icons.\n", len(localIcons))

	importedCount := 0
	for name, targetPath := range missingIcons {
		// Try exact match
		if sourcePath, ok := localIcons[name]; ok {
			err := copyFile(sourcePath, filepath.Join("frontend/public", targetPath))
			if err == nil {
				importedCount++
				delete(missingIcons, name)
				continue
			}
		}

		// Try fuzzy match (e.g. removing upgrade suffixes like _10)
		baseName := name
		if strings.Contains(name, "_") {
			parts := strings.Split(name, "_")
			if len(parts) > 1 {
				// Check if last part is a number
				lastPart := parts[len(parts)-1]
				isNumber := true
				for _, char := range lastPart {
					if char < '0' || char > '9' {
						isNumber = false
						break
					}
				}
				if isNumber {
					baseName = strings.Join(parts[:len(parts)-1], "_")
					if sourcePath, ok := localIcons[baseName]; ok {
						err := copyFile(sourcePath, filepath.Join("frontend/public", targetPath))
						if err == nil {
							importedCount++
							delete(missingIcons, name)
							continue
						}
					}
				}
			}
		}
	}

	fmt.Printf("✅ Imported %d icons from local tmp/icons.\n", importedCount)

	// Update missing_icons.txt
	f, _ := os.Create("missing_icons.txt")
	defer f.Close()
	for _, path := range missingIcons {
		fmt.Fprintln(f, path)
	}
	fmt.Printf("📝 Updated missing_icons.txt with %d remaining icons.\n", len(missingIcons))
}

func normalizeName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "!", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, "__", "_")
	return strings.Trim(s, "_")
}

func copyFile(src, dst string) error {
	os.MkdirAll(filepath.Dir(dst), 0755)
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
