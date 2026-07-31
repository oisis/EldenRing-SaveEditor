package schema

type ArmorData struct {
	SourceRowID    Fact[uint32]   `json:"sourceRowID"`
	IconIDMale     Fact[uint32]   `json:"iconIDMale"`
	IconIDFemale   Fact[uint32]   `json:"iconIDFemale"`
	SortID         Fact[uint32]   `json:"sortID"`
	SortIDSFV      *Fact[uint32]  `json:"sortID-sfv,omitempty"`
	SortGroupID    Fact[uint8]    `json:"sortGroupID"`
	SortGroupIDSFV *Fact[uint8]   `json:"sortGroupID-sfv,omitempty"`
	Weight         Fact[float64]  `json:"weight"`
	WeightSFV      *Fact[float64] `json:"weight-sfv,omitempty"`
	Physical       Fact[float64]  `json:"physical"`
	PhysicalSFV    *Fact[float64] `json:"physical-sfv,omitempty"`
	Strike         Fact[float64]  `json:"strike"`
	StrikeSFV      *Fact[float64] `json:"strike-sfv,omitempty"`
	Slash          Fact[float64]  `json:"slash"`
	SlashSFV       *Fact[float64] `json:"slash-sfv,omitempty"`
	Pierce         Fact[float64]  `json:"pierce"`
	PierceSFV      *Fact[float64] `json:"pierce-sfv,omitempty"`
	Magic          Fact[float64]  `json:"magic"`
	MagicSFV       *Fact[float64] `json:"magic-sfv,omitempty"`
	Fire           Fact[float64]  `json:"fire"`
	FireSFV        *Fact[float64] `json:"fire-sfv,omitempty"`
	Lightning      Fact[float64]  `json:"lightning"`
	LightningSFV   *Fact[float64] `json:"lightning-sfv,omitempty"`
	Holy           Fact[float64]  `json:"holy"`
	HolySFV        *Fact[float64] `json:"holy-sfv,omitempty"`
	Immunity       Fact[uint32]   `json:"immunity"`
	ImmunitySFV    *Fact[uint32]  `json:"immunity-sfv,omitempty"`
	Robustness     Fact[uint32]   `json:"robustness"`
	RobustnessSFV  *Fact[uint32]  `json:"robustness-sfv,omitempty"`
	Focus          Fact[uint32]   `json:"focus"`
	FocusSFV       *Fact[uint32]  `json:"focus-sfv,omitempty"`
	Vitality       Fact[uint32]   `json:"vitality"`
	VitalitySFV    *Fact[uint32]  `json:"vitality-sfv,omitempty"`
	Poise          Fact[float64]  `json:"poise"`
	PoiseSFV       *Fact[float64] `json:"poise-sfv,omitempty"`
	HeadEquipable  Fact[bool]     `json:"headEquipable"`
	BodyEquipable  Fact[bool]     `json:"bodyEquipable"`
	ArmEquipable   Fact[bool]     `json:"armEquipable"`
	LegEquipable   Fact[bool]     `json:"legEquipable"`
}
