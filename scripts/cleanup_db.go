package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func cleanupWeapons() {
	path := "backend/db/data/weapons.go"
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening weapons: %v\n", err)
		return
	}
	defer file.Close()

	affixes := []string{
		"Heavy ", "Keen ", "Quality ", "Fire ", "Flame Art ",
		"Lightning ", "Sacred ", "Magic ", "Cold ", "Poison ",
		"Blood ", "Occult ",
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		isAffinity := false
		
		if strings.Contains(line, ": \"") {
			isAmmo := strings.Contains(line, "Arrow") || 
					  strings.Contains(line, "Bolt") || 
					  strings.Contains(line, "Great Arrow") || 
					  strings.Contains(line, "Great Bolt") ||
					  strings.Contains(line, "Harpoon")

			if !isAmmo {
				for _, affix := range affixes {
					if strings.Contains(line, affix) {
						idxStart := strings.Index(line, ": \"") + 3
						idxEnd := strings.LastIndex(line, "\"")
						if idxStart < 0 || idxEnd < 0 || idxStart >= idxEnd {
							continue
						}
						namePart := line[idxStart:idxEnd]
						
						if strings.HasPrefix(namePart, affix) || strings.Contains(namePart, "'s "+affix) {
							isAffinity = true
							break
						}
					}
				}
			}
		}

		if !isAffinity {
			lines = append(lines, line)
		}
	}

	output := strings.Join(lines, "\n") + "\n"
	err = os.WriteFile(path, []byte(output), 0644)
	if err != nil {
		fmt.Printf("Error writing weapons: %v\n", err)
	} else {
		fmt.Println("Cleaned weapons.go")
	}
}

func cleanupItems() {
	path := "backend/db/data/items.go"
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Error opening items: %v\n", err)
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.Contains(line, " +") {
			for i := 10; i >= 1; i-- {
				suffix := fmt.Sprintf(" +%d", i)
				if strings.Contains(line, suffix) {
					line = strings.ReplaceAll(line, suffix, "")
					break
				}
			}
		}
		
		lines = append(lines, line)
	}

	output := strings.Join(lines, "\n") + "\n"
	err = os.WriteFile(path, []byte(output), 0644)
	if err != nil {
		fmt.Printf("Error writing items: %v\n", err)
	} else {
		fmt.Println("Cleaned items.go")
	}
}

func cleanupAows() {
	path := "backend/db/data/aows.go"
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading aows: %v\n", err)
		return
	}
	
	content := string(data)
	content = strings.ReplaceAll(content, "Ash of War: ", "")
	
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error writing aows: %v\n", err)
	} else {
		fmt.Println("Cleaned aows.go")
	}
}

func main() {
	cleanupWeapons()
	cleanupItems()
	cleanupAows()
}
