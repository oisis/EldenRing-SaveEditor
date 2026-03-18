package db

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed json/*.json
var jsonFiles embed.FS

type Item struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}

type ItemGroup struct {
	Category string `json:"category"`
	Items    []Item `json:"items"`
}

type Database struct {
	Weapons   []ItemGroup
	Armors    []ItemGroup
	Items     []ItemGroup
	Talismans []ItemGroup
	Graces    []Item
	Bosses    []Item
}

var instance *Database

func GetInstance() *Database {
	if instance == nil {
		instance = &Database{}
		instance.load()
	}
	return instance
}

func (db *Database) load() {
	db.Weapons = loadGroups("json/weapons.json")
	db.Armors = loadGroups("json/armors.json")
	db.Items = loadGroups("json/items.json")
	db.Talismans = loadGroups("json/talismans.json")
	db.Graces = loadItems("json/graces.json")
	db.Bosses = loadItems("json/bosses.json")
}

func loadGroups(path string) []ItemGroup {
	data, err := jsonFiles.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", path, err)
		return nil
	}
	var groups []ItemGroup
	if err := json.Unmarshal(data, &groups); err != nil {
		fmt.Printf("Error unmarshaling %s: %v\n", path, err)
		return nil
	}
	return groups
}

func loadItems(path string) []Item {
	data, err := jsonFiles.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", path, err)
		return nil
	}
	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		fmt.Printf("Error unmarshaling %s: %v\n", path, err)
		return nil
	}
	return items
}
