package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	path := "backend/db/data/spirit_ashes.go"
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening spirit_ashes.go: %v\n", err)
		return
	}
	defer file.Close()

	// Regex to match the ItemData struct fields
	reInv := regexp.MustCompile(`MaxInventory: \d+`)
	reSto := regexp.MustCompile(`MaxStorage: \d+`)
	reUpg := regexp.MustCompile(`MaxUpgrade: \d+`)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "IconPath: \"items/spirit_ashes/") {
			line = reInv.ReplaceAllString(line, "MaxInventory: 1")
			line = reSto.ReplaceAllString(line, "MaxStorage: 1")
			line = reUpg.ReplaceAllString(line, "MaxUpgrade: 10")
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading spirit_ashes.go: %v\n", err)
		return
	}

	output := strings.Join(lines, "\n") + "\n"
	err = os.WriteFile(path, []byte(output), 0644)
	if err != nil {
		fmt.Printf("Error writing spirit_ashes.go: %v\n", err)
	} else {
		fmt.Println("Successfully updated Spirit Ashes settings in spirit_ashes.go")
	}
}
