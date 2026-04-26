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
	// Note: multi-word affinities like "Flame_Art" must come BEFORE single-word "Flame"
	// to avoid prefix-shadowing (e.g. "Flame_Art_Main_Gauche" must strip "Flame_Art_" not "Flame_").
	// "Bloody" is NOT an affinity — it's part of unique DLC weapon names (Bloody Lance, Bloody Buckler).
	// Stripping "Bloody" would yield "Lance"/"Buckler" which are different base weapons. Excluded.
	affinities = []string{"Flame_Art", "Heavy", "Keen", "Quality", "Fire", "Flame", "Lightning", "Sacred", "Magic", "Cold", "Poison", "Blood", "Occult"}
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
	limit := 1000
	consecutiveFailures := 0
	maxConsecutiveFailures := 200

	for scanner.Scan() && count < limit {
		if consecutiveFailures >= maxConsecutiveFailures {
			fmt.Println("🛑 Too many consecutive failures. Stopping batch.")
			break
		}

		iconPath := scanner.Text()
		fullLocalPath := filepath.Join("frontend/public", iconPath)
		filename := filepath.Base(iconPath)

		// Skip already downloaded files
		if _, err := os.Stat(fullLocalPath); err == nil {
			fmt.Printf("⏭️  Skip (exists): %s\n", filename)
			count++
			consecutiveFailures = 0
			continue
		}
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
		isWeapon := strings.Contains(iconPath, "melee_armaments/") || strings.Contains(iconPath, "weapons/")
		isArmor := strings.Contains(iconPath, "/head/") || strings.Contains(iconPath, "/chest/") ||
			strings.Contains(iconPath, "/arms/") || strings.Contains(iconPath, "/legs/") ||
			strings.Contains(iconPath, "armor/")
		isArrow := strings.Contains(iconPath, "arrows_and_bolts/")
		isRanged := strings.Contains(iconPath, "ranged_and_catalysts/")
		isShield := strings.Contains(iconPath, "shields/")
		isTalisman := strings.Contains(iconPath, "talismans/")
		isAoW := strings.Contains(iconPath, "ashes_of_war/") || strings.Contains(iconPath, "/ashes/")
		isSpiritAsh := strings.Contains(iconPath, "/ashes/") && !strings.Contains(iconPath, "ashes_of_war/")
		isTools := strings.Contains(iconPath, "tools/") || strings.Contains(iconPath, "key_items/") || strings.Contains(iconPath, "goods/")
		isCrafting := strings.Contains(iconPath, "crafting_materials/") || strings.Contains(iconPath, "bolstering_materials/")
		isSorcery := strings.Contains(iconPath, "sorceries/")
		isIncantation := strings.Contains(iconPath, "incantations/")

		if isWeapon || isRanged || isShield || isArrow {
			for _, aff := range affinities {
				if strings.HasPrefix(wikiName, aff+"_") {
					wikiName = strings.TrimPrefix(wikiName, aff+"_")
					break
				}
			}
		}

		// Determine prefix based on category
		var prefixes []string
		if isWeapon {
			prefixes = []string{"ER_Icon_Weapon_"}
		} else if isArrow {
			prefixes = []string{"ER_Icon_Weapon_"}
		} else if isRanged {
			prefixes = []string{"ER_Icon_Weapon_"}
		} else if isShield {
			prefixes = []string{"ER_Icon_Weapon_"}
		} else if isArmor {
			prefixes = []string{"ER_Icon_Armor_"}
		} else if isTalisman {
			prefixes = []string{"ER_Icon_Talisman_"}
		} else if isSpiritAsh {
			prefixes = []string{"ER_Icon_Ash_", "ER_Icon_Tool_", "ER_Icon_Item_"}
		} else if isAoW {
			prefixes = []string{"ER_Icon_Ash_of_War_", "ER_Icon_ash_of_war_"}
		} else if isTools {
			prefixes = []string{"ER_Icon_Tool_", "ER_Icon_Item_", "ER_Icon_Key_Item_", "ER_Icon_Note_", "ER_Icon_Info_", "ER_Icon_Map_", "ER_Info_"}
		} else if isCrafting {
			isBolstering := strings.Contains(iconPath, "bolstering_materials/")
			if isBolstering {
				prefixes = []string{"ER_Icon_Bolstering_Material_", "ER_Icon_Crafting_Material_", "ER_Icon_Item_"}
			} else {
				prefixes = []string{"ER_Icon_Crafting_Material_", "ER_Icon_Item_", "ER_Icon_Tool_"}
			}
		} else if isSorcery || isIncantation {
			prefixes = []string{"ER_Icon_Spell_"}
		}
		_ = isSpiritAsh

		// Hosts to try in order: eldenring.wiki.gg first, then commons.wiki.gg as fallback
		// (some images live on the commons subdomain across the wiki.gg network).
		hosts := []string{"https://eldenring.wiki.gg", "https://commons.wiki.gg"}

		// Generate name variants: original + possessive variants (insert %27 before trailing 's')
		// e.g. "Tibias_Cookbook" → "Tibia%27s_Cookbook"
		nameVariants := []string{wikiName}
		nameVariants = append(nameVariants, insertPossessiveVariants(wikiName)...)
		// For weapons with stripped affinity, also try the UN-stripped name as fallback
		// (e.g. "Bloody Buckler" is a unique DLC weapon, not an affinity of "Buckler").
		if (isWeapon || isRanged || isShield || isArrow) && wikiName != toWikiName(cleanName) {
			fullName := toWikiName(cleanName)
			nameVariants = append(nameVariants, fullName)
			nameVariants = append(nameVariants, insertPossessiveVariants(fullName)...)
		}

		success := false
		for _, prefix := range prefixes {
			if success {
				break
			}
			for _, nv := range nameVariants {
				if success {
					break
				}
				fullWikiName := prefix + nv + ".png"
				for _, host := range hosts {
					// Try direct URL first
					remoteURL := host + "/images/" + fullWikiName
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
					thumbURL := host + "/images/thumb/" + fullWikiName + "/600px-" + fullWikiName
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
			}
		}

		// Try with original numbered suffix (e.g. Cookbook_1 instead of Cookbook)
		wikiNameWithNum := toWikiName(nameOnly)
		if !success && wikiNameWithNum != wikiName {
			for _, prefix := range prefixes {
				fullWikiName := prefix + wikiNameWithNum + ".png"
				remoteURL := "https://eldenring.wiki.gg/images/" + fullWikiName
				fmt.Printf("📥 [%d] Trying (numbered) %s\n", count+1, remoteURL)
				err := downloadFile(remoteURL, fullLocalPath)
				if err == nil {
					fmt.Printf("✅ Success (Numbered): %s\n", filename)
					count++
					success = true
					consecutiveFailures = 0
					break
				}
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

// Wiki keeps short prepositions/articles in lowercase (e.g. "Rain_of_Fire", "Note:_The_Lord").
var lowercaseWords = map[string]bool{
	"of": true, "the": true, "a": true, "an": true, "and": true,
	"in": true, "to": true, "with": true, "from": true, "for": true,
	"on": true, "at": true, "by": true, "or": true, "as": true,
}

// insertPossessiveVariants returns 0..N variants of `s` where the first qualifying word
// has its trailing 's' replaced by URL-encoded possessive `%27s`. Wiki filenames preserve
// apostrophes (e.g. "Tibia's_Cookbook"); local filenames strip them ("tibias_cookbook"
// or "thopss_barrier" when the source name was "Thops's").
//
// Variants generated:
//   "Tibias_Cookbook"  → ["Tibia%27s_Cookbook"]
//   "Thopss_Barrier"   → ["Thops%27s_Barrier", "Thopss%27_Barrier"]
//   "Items_Foo"        → []  (too short to disambiguate)
func insertPossessiveVariants(s string) []string {
	parts := strings.Split(s, "_")
	var out []string
	for i, p := range parts {
		if len(p) < 4 || !strings.HasSuffix(p, "s") {
			continue
		}
		// Variant 1: drop trailing 's', insert %27s
		v1 := append([]string(nil), parts...)
		v1[i] = p[:len(p)-1] + "%27s"
		out = append(out, strings.Join(v1, "_"))
		// Variant 2 (only for double-s): keep both s's, insert %27 BEFORE final s
		// e.g. "Thopss" → "Thops%27s" already handled by Variant 1, but original
		// "Thopss" might also genuinely be "Thops" + plural — try unchanged too.
		if strings.HasSuffix(strings.ToLower(p), "ss") {
			// Edge case: "Hess" → drop one 's' to get "Hes%27s"? Rare. Skip.
			_ = p
		}
		// Only first qualifying word — break.
		break
	}
	return out
}

func toWikiName(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			if p == "+1" || p == "+2" || p == "+3" {
				parts[i] = p
			} else if i > 0 && lowercaseWords[strings.ToLower(p)] {
				parts[i] = strings.ToLower(p)
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
