package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func normalize(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	
	// Remove " - MENU_Knowledge..." suffix
	if idx := strings.Index(base, " - MENU_Knowledge"); idx != -1 {
		base = base[:idx]
	}
	
	// Lowercase
	res := strings.ToLower(base)
	
	// Handle possessive 's that was often replaced by _s in source filenames
	res = strings.ReplaceAll(res, "_s ", "s ")
	res = strings.ReplaceAll(res, "_s_", "s_")
	
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
	
	return res + ".png"
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func main() {
	srcRoot := "tmp/icons"
	dstRoot := "frontend/public/items"

	mapping := map[string]string{
		"Armor":                    "armor",
		"Ashes of War":             "ashes",
		"Melee Armaments":          "weapons",
		"Ranged Weapons-Catalysts": "weapons",
		"Shields":                  "weapons",
		"Arrows-Bolts":             "weapons",
		"Talismans":                "talismans",
		"Spirit Ashes":             "goods",
		"Key Items":                "goods",
		"Tools":                    "goods",
		"Bolstering Materials":     "goods",
		"Crafting Materials":       "goods",
		"Misc":                     "goods",
		"Incantations":             "goods",
		"Sorceries":                "goods",
	}

	err := filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Skip files that are just MENU_Knowledge without names (DLC items)
		if strings.HasPrefix(info.Name(), "MENU_Knowledge") {
			return nil
		}

		category := ""
		for srcDir, dstCat := range mapping {
			if strings.Contains(path, "/"+srcDir+"/") {
				category = dstCat
				break
			}
		}

		if category == "" {
			return nil
		}

		dstDir := filepath.Join(dstRoot, category)
		os.MkdirAll(dstDir, 0755)

		newName := normalize(info.Name())
		dstPath := filepath.Join(dstDir, newName)

		fmt.Printf("Migrating: %s -> %s\n", path, dstPath)
		err = copyFile(path, dstPath)
		if err != nil {
			fmt.Printf("Error copying %s: %v\n", path, err)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking path: %v\n", err)
	}
}
