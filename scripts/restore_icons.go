package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	files := []string{"items.go", "weapons.go", "armors.go", "talismans.go", "aows.go"}
	for _, f := range files {
		restoreIcons(f)
	}
}

func restoreIcons(filename string) {
	path := "backend/db/data/" + filename
	fmt.Printf("🔄 Restoring icons for %s...\n", filename)

	// 1. Get old icon paths from git
	oldIcons := make(map[string]string)
	cmd := exec.Command("git", "show", "HEAD:"+path)
	output, err := cmd.Output()
	if err == nil {
		re := regexp.MustCompile(`(0x[0-9A-Fa-f]+): \{.*IconPath: "(.*)"\}`)
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := scanner.Text()
			matches := re.FindStringSubmatch(line)
			if len(matches) == 3 {
				oldIcons[matches[1]] = matches[2]
			}
		}
	}

	// 2. Read current file
	currentContent, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("⚠️ Error reading %s: %v\n", path, err)
		return
	}

	// 3. Process line by line and restore/generate icons
	var newLines []string
	scanner := bufio.NewScanner(strings.NewReader(string(currentContent)))
	
	// Determine category for new icons
	category := "goods"
	switch filename {
	case "weapons.go": category = "weapons"
	case "armors.go": category = "armor"
	case "talismans.go": category = "talismans"
	case "aows.go": category = "ashes"
	}

	reLine := regexp.MustCompile(`(\t(0x[0-9A-Fa-f]+): \{Name: "(.*)", MaxInventory: (\d+), MaxStorage: (\d+), MaxUpgrade: (\d+), IconPath: ""\},)`)
	
	for scanner.Scan() {
		line := scanner.Text()
		matches := reLine.FindStringSubmatch(line)
		
		if len(matches) == 7 {
			id := matches[2]
			name := matches[3]
			inv := matches[4]
			storage := matches[5]
			upgrade := matches[6]
			
			iconPath := ""
			if oldPath, ok := oldIcons[id]; ok && oldPath != "" {
				iconPath = oldPath
			} else {
				// Generate new path
				cleanName := strings.ToLower(name)
				cleanName = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(cleanName, "_")
				cleanName = strings.Trim(cleanName, "_")
				iconPath = fmt.Sprintf("items/%s/%s.png", category, cleanName)
			}
			
			// Reconstruct line EXACTLY as it was, only changing IconPath
			newLine := fmt.Sprintf("\t%s: {Name: \"%s\", MaxInventory: %s, MaxStorage: %s, MaxUpgrade: %s, IconPath: \"%s\"},", 
				id, name, inv, storage, upgrade, iconPath)
			newLines = append(newLines, newLine)
		} else {
			newLines = append(newLines, line)
		}
	}

	// 4. Write back
	err = os.WriteFile(path, []byte(strings.Join(newLines, "\n")+"\n"), 0644)
	if err != nil {
		fmt.Printf("⚠️ Error writing %s: %v\n", path, err)
	} else {
		fmt.Printf("✅ Icons restored for %s\n", filename)
	}
}
