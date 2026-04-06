package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	dbFiles := []string{"items.go", "spirit_ashes.go", "weapons.go", "armors.go", "talismans.go", "aows.go"}
	missing := []string{}

	reIcon := regexp.MustCompile(`IconPath: "(.*)"`)
	for _, f := range dbFiles {
		path := filepath.Join("backend/db/data", f)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		matches := reIcon.FindAllStringSubmatch(string(content), -1)
		for _, m := range matches {
			if len(m) == 2 && m[1] != "" {
				fullPath := filepath.Join("frontend/public", m[1])
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					missing = append(missing, m[1])
				}
			}
		}
	}

	// Remove duplicates
	uniqueMissing := make(map[string]bool)
	for _, m := range missing {
		uniqueMissing[m] = true
	}

	f, _ := os.Create("missing_icons.txt")
	defer f.Close()
	for m := range uniqueMissing {
		fmt.Fprintln(f, m)
	}
	fmt.Printf("📊 Total unique missing icons: %d\n", len(uniqueMissing))
}
