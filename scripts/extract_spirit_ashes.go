package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	itemsFile := "backend/db/data/items.go"
	spiritAshesFile := "backend/db/data/spirit_ashes.go"

	file, err := os.Open(itemsFile)
	if err != nil {
		fmt.Println("Error opening items.go:", err)
		return
	}
	defer file.Close()

	var newItemsContent []string
	var spiritAshesContent []string

	spiritAshesContent = append(spiritAshesContent, "package data")
	spiritAshesContent = append(spiritAshesContent, "")
	spiritAshesContent = append(spiritAshesContent, "// SpiritAshes contains all spirit ashes and puppets.")
	spiritAshesContent = append(spiritAshesContent, "var SpiritAshes = map[uint32]ItemData{")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "IconPath: \"items/spirit_ashes/") {
			spiritAshesContent = append(spiritAshesContent, line)
		} else {
			newItemsContent = append(newItemsContent, line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading items.go:", err)
		return
	}

	spiritAshesContent = append(spiritAshesContent, "}")

	// Write new items.go
	err = os.WriteFile(itemsFile, []byte(strings.Join(newItemsContent, "\n")+"\n"), 0644)
	if err != nil {
		fmt.Println("Error writing items.go:", err)
		return
	}

	// Write spirit_ashes.go
	err = os.WriteFile(spiritAshesFile, []byte(strings.Join(spiritAshesContent, "\n")+"\n"), 0644)
	if err != nil {
		fmt.Println("Error writing spirit_ashes.go:", err)
		return
	}

	fmt.Println("Successfully extracted spirit ashes to", spiritAshesFile)
}
