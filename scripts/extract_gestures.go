package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	itemsFile := "backend/db/data/items.go"
	gesturesFile := "backend/db/data/gestures.go"
	oldIconDir := "frontend/public/items/goods"
	newIconDir := "frontend/public/items/gestures"

	// Create new icon directory
	os.MkdirAll(newIconDir, 0755)

	file, err := os.Open(itemsFile)
	if err != nil {
		fmt.Println("Error opening items.go:", err)
		return
	}
	defer file.Close()

	var newItemsContent []string
	var gesturesContent []string

	gesturesContent = append(gesturesContent, "package data")
	gesturesContent = append(gesturesContent, "")
	gesturesContent = append(gesturesContent, "// Gestures contains all gestures.")
	gesturesContent = append(gesturesContent, "var Gestures = map[uint32]ItemData{")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		isGesture := false
		if strings.Contains(line, "0x400023") || strings.Contains(line, "0xB00023") || strings.Contains(line, "0x401EA7A") || strings.Contains(line, "0xB01EA7A") {
			parts := strings.Split(strings.TrimSpace(line), ":")
			if len(parts) > 0 {
				idStr := strings.TrimSpace(parts[0])
				if strings.HasPrefix(idStr, "0x") {
					id, err := strconv.ParseUint(idStr[2:], 16, 32)
					if err == nil {
						if (id >= 0x40002328 && id <= 0x4000235A) ||
							(id >= 0xB0002328 && id <= 0xB000235A) ||
							(id >= 0x401EA7A8 && id <= 0x401EA7AC) ||
							(id >= 0xB01EA7A8 && id <= 0xB01EA7AC) {
							isGesture = true
						}
					}
				}
			}
		}

		if isGesture {
			// Extract icon filename
			iconStart := strings.Index(line, "IconPath: \"items/goods/")
			if iconStart != -1 {
				iconStart += len("IconPath: \"items/goods/")
				iconEnd := strings.Index(line[iconStart:], "\"")
				if iconEnd != -1 {
					iconFilename := line[iconStart : iconStart+iconEnd]

					// Move icon file
					oldPath := filepath.Join(oldIconDir, iconFilename)
					newPath := filepath.Join(newIconDir, iconFilename)

					// Only move if it exists in the old path
					if _, err := os.Stat(oldPath); err == nil {
						err = os.Rename(oldPath, newPath)
						if err != nil {
							fmt.Printf("Error moving icon %s: %v\n", iconFilename, err)
						} else {
							fmt.Printf("Moved icon: %s\n", iconFilename)
						}
					}
				}
			}

			// Replace IconPath in the line
			line = strings.Replace(line, "items/goods/", "items/gestures/", 1)
			gesturesContent = append(gesturesContent, line)
		} else {
			newItemsContent = append(newItemsContent, line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading items.go:", err)
		return
	}

	gesturesContent = append(gesturesContent, "}")

	// Write new items.go
	err = os.WriteFile(itemsFile, []byte(strings.Join(newItemsContent, "\n")+"\n"), 0644)
	if err != nil {
		fmt.Println("Error writing items.go:", err)
		return
	}

	// Write gestures.go
	err = os.WriteFile(gesturesFile, []byte(strings.Join(gesturesContent, "\n")+"\n"), 0644)
	if err != nil {
		fmt.Println("Error writing gestures.go:", err)
		return
	}

	fmt.Println("Successfully extracted gestures to", gesturesFile)
}
