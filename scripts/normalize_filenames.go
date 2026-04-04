package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func normalize(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	
	// Lowercase
	res := strings.ToLower(base)
	
	// Replace spaces and hyphens with underscores
	res = strings.ReplaceAll(res, " ", "_")
	res = strings.ReplaceAll(res, "-", "_")
	
	// Remove special characters
	reg := regexp.MustCompile(`[^a-z0-9_]`)
	res = reg.ReplaceAllString(res, "")
	
	// Collapse multiple underscores
	regMulti := regexp.MustCompile(`_+`)
	res = regMulti.ReplaceAllString(res, "_")
	
	// Trim underscores from ends
	res = strings.Trim(res, "_")
	
	return res + ext
}

func main() {
	root := "frontend/public/items"
	
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		
		dir := filepath.Dir(path)
		oldName := filepath.Base(path)
		newName := normalize(oldName)
		
		if oldName != newName {
			newPath := filepath.Join(dir, newName)
			fmt.Printf("Renaming: %s -> %s\n", path, newPath)
			
			// Check if destination exists to avoid overwriting
			if _, err := os.Stat(newPath); err == nil {
				fmt.Printf("WARNING: Destination %s already exists, skipping.\n", newPath)
				return nil
			}
			
			err := os.Rename(path, newPath)
			if err != nil {
				fmt.Printf("ERROR renaming %s: %v\n", path, err)
			}
		}
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("Error walking path: %v\n", err)
	}
}
