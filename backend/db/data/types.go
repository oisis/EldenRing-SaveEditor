package data

// ItemData represents the metadata for an item in the game database.
type ItemData struct {
	Name         string
	Category     string
	MaxInventory uint32
	MaxStorage   uint32
	MaxUpgrade   uint32
	IconPath     string
	Flags        []string
}
