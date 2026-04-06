package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	affinities = []string{"Heavy", "Keen", "Quality", "Fire", "Flame", "Lightning", "Sacred", "Magic", "Cold", "Poison", "Blood", "Occult"}
	reUpgrade  = regexp.MustCompile(`_(\d+)$`)
)

func main() {
	f, err := os.Open("missing_icons.txt")
	if err != nil {
		fmt.Printf("⚠️ Error opening missing_icons.txt: %v\n", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	limit := 500
	consecutiveFailures := 0
	maxConsecutiveFailures := 50

	for scanner.Scan() && count < limit {
		if consecutiveFailures >= maxConsecutiveFailures {
			fmt.Println("🛑 Too many consecutive failures. Stopping batch.")
			break
		}

		iconPath := scanner.Text()
		fullLocalPath := filepath.Join("frontend/public", iconPath)
		filename := filepath.Base(iconPath)
		nameOnly := strings.TrimSuffix(filename, ".png")

		// 1. Clean up the name
		cleanName := nameOnly

		// Handle Talismans (convert _1, _2, _3 to +1, +2, +3)
		if strings.Contains(iconPath, "talismans/") {
			if reUpgrade.MatchString(cleanName) {
				cleanName = reUpgrade.ReplaceAllString(cleanName, "_+$1")
			}
		} else {
			// Remove upgrade suffixes for weapons and ashes (_8, _10, _25 etc)
			cleanName = reUpgrade.ReplaceAllString(cleanName, "")
		}

		// Convert to Wiki Title Case
		wikiName := toWikiName(cleanName)

		// Remove Affinities for weapons
		if strings.Contains(iconPath, "weapons/") {
			for _, aff := range affinities {
				if strings.HasPrefix(wikiName, aff+"_") {
					wikiName = strings.TrimPrefix(wikiName, aff+"_")
					break
				}
			}
		}

		// Determine prefix based on category
		var prefixes []string
		if strings.Contains(iconPath, "weapons/") {
			prefixes = []string{"ER_Icon_Weapon_"}
		} else if strings.Contains(iconPath, "armor/") {
			prefixes = []string{"ER_Icon_Armor_"}
		} else if strings.Contains(iconPath, "talismans/") {
			prefixes = []string{"ER_Icon_Talisman_"}
		} else if strings.Contains(iconPath, "goods/") {
			// Check if it's an Ash (Spirit Ash)
			if strings.Contains(nameOnly, "ashes") || strings.Contains(nameOnly, "puppet") {
				prefixes = []string{"ER_Icon_Ash_", "ER_Icon_Tool_", "ER_Icon_Item_"}
			} else {
				prefixes = []string{"ER_Icon_Tool_", "ER_Icon_Item_"}
			}
		} else if strings.Contains(iconPath, "ashes/") {
			prefixes = []string{"ER_Icon_Ash_of_War_", "ER_Icon_ash_of_war_"}
		}

		success := false
		for _, prefix := range prefixes {
			fullWikiName := prefix + wikiName + ".png"

			// Try direct URL first
			remoteURL := "https://eldenring.wiki.gg/images/" + fullWikiName
			fmt.Printf("📥 [%d] Trying %s\n", count+1, remoteURL)
			err := downloadFile(remoteURL, fullLocalPath)
			if err == nil {
				fmt.Printf("✅ Success: %s\n", filename)
				count++
				success = true
				consecutiveFailures = 0
				break
			}

			// Try thumb URL as fallback
			thumbURL := "https://eldenring.wiki.gg/images/thumb/" + fullWikiName + "/600px-" + fullWikiName
			fmt.Printf("📥 [%d] Trying Thumb %s\n", count+1, thumbURL)
			err = downloadFile(thumbURL, fullLocalPath)
			if err == nil {
				fmt.Printf("✅ Success (Thumb): %s\n", filename)
				count++
				success = true
				consecutiveFailures = 0
				break
			}
		}

		if !success {
			// Try without prefix
			remoteURL := "https://eldenring.wiki.gg/images/" + wikiName + ".png"
			fmt.Printf("📥 [%d] Fallback try %s\n", count+1, remoteURL)
			err := downloadFile(remoteURL, fullLocalPath)
			if err == nil {
				fmt.Printf("✅ Success (Fallback): %s\n", filename)
				count++
				success = true
				consecutiveFailures = 0
			}
		}

		if !success {
			consecutiveFailures++
			fmt.Printf("🚫 Could not find icon for: %s (WikiName: %s)\n", filename, wikiName)
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("\n✅ Batch complete. Downloaded %d icons.\n", count)
}

func toWikiName(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			if p == "+1" || p == "+2" || p == "+3" {
				parts[i] = p
			} else {
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
	}
	return strings.Join(parts, "_")
}

func downloadFile(url, path string) error {
	os.MkdirAll(filepath.Dir(path), 0755)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
