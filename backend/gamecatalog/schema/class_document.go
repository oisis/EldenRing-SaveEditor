package schema

// ClassDocument declares one playable starting class resolved from the save's
// starting-class ID, carrying its ID, official name and eight base attributes.
// Each fact retains its independent source provenance.
type ClassDocument struct {
	StartingClassID Fact[uint32] `json:"startingClassID"`
	Name            Fact[string] `json:"name"`
	Vigor           Fact[uint32] `json:"vigor"`
	Mind            Fact[uint32] `json:"mind"`
	Endurance       Fact[uint32] `json:"endurance"`
	Strength        Fact[uint32] `json:"strength"`
	Dexterity       Fact[uint32] `json:"dexterity"`
	Intelligence    Fact[uint32] `json:"intelligence"`
	Faith           Fact[uint32] `json:"faith"`
	Arcane          Fact[uint32] `json:"arcane"`
}
