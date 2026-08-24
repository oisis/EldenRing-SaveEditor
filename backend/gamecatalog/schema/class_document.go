package schema

// ClassDocument declares one playable starting class resolved from the save's
// starting-class ID, carrying its ID, official name, base level and eight base
// attributes. Each fact retains its independent source provenance.
//
// Level is the CharaInitParam soulLv of the class row, read directly from the
// regulation. It is never derived from the eight attributes: the level formula
// is a rule of SetCharacterStats, not the source of this fact.
type ClassDocument struct {
	StartingClassID Fact[uint32] `json:"startingClassID"`
	Name            Fact[string] `json:"name"`
	Level           Fact[uint32] `json:"level"`
	Vigor           Fact[uint32] `json:"vigor"`
	Mind            Fact[uint32] `json:"mind"`
	Endurance       Fact[uint32] `json:"endurance"`
	Strength        Fact[uint32] `json:"strength"`
	Dexterity       Fact[uint32] `json:"dexterity"`
	Intelligence    Fact[uint32] `json:"intelligence"`
	Faith           Fact[uint32] `json:"faith"`
	Arcane          Fact[uint32] `json:"arcane"`
}
