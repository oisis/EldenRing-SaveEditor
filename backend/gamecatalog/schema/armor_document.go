package schema

type ArmorData struct {
	SourceRowID   Fact[uint32]  `json:"sourceRowID"`
	IconIDMale    Fact[uint32]  `json:"iconIDMale"`
	IconIDFemale  Fact[uint32]  `json:"iconIDFemale"`
	SortID        Fact[uint32]  `json:"sortID"`
	SortGroupID   Fact[uint8]   `json:"sortGroupID"`
	Weight        Fact[float64] `json:"weight"`
	Physical      Fact[float64] `json:"physical"`
	Strike        Fact[float64] `json:"strike"`
	Slash         Fact[float64] `json:"slash"`
	Pierce        Fact[float64] `json:"pierce"`
	Magic         Fact[float64] `json:"magic"`
	Fire          Fact[float64] `json:"fire"`
	Lightning     Fact[float64] `json:"lightning"`
	Holy          Fact[float64] `json:"holy"`
	Immunity      Fact[uint32]  `json:"immunity"`
	Robustness    Fact[uint32]  `json:"robustness"`
	Focus         Fact[uint32]  `json:"focus"`
	Vitality      Fact[uint32]  `json:"vitality"`
	Poise         Fact[float64] `json:"poise"`
	HeadEquipable Fact[bool]    `json:"headEquipable"`
	BodyEquipable Fact[bool]    `json:"bodyEquipable"`
	ArmEquipable  Fact[bool]    `json:"armEquipable"`
	LegEquipable  Fact[bool]    `json:"legEquipable"`
}
