package schema

type AshOfWarData struct {
	SourceRowID          Fact[uint32]   `json:"sourceRowID"`
	IconID               Fact[uint32]   `json:"iconID"`
	SortID               Fact[uint32]   `json:"sortID"`
	SortIDSFV            *Fact[uint32]  `json:"sortID-sfv,omitempty"`
	SortGroupID          Fact[uint8]    `json:"sortGroupID"`
	SortGroupIDSFV       *Fact[uint8]   `json:"sortGroupID-sfv,omitempty"`
	SwordArtsParamID     Fact[int32]    `json:"swordArtsParamID"`
	SwordArtsName        Fact[string]   `json:"swordArtsName"`
	SwordArtsNameSFV     *Fact[string]  `json:"swordArtsName-sfv,omitempty"`
	DefaultAffinity      Fact[uint8]    `json:"defaultAffinity"`
	CompatibilityMask    Fact[uint64]   `json:"compatibilityMask"`
	CompatibilityMaskSFV *Fact[uint64]  `json:"compatibilityMask-sfv,omitempty"`
	CompatibleClassNames Fact[[]string] `json:"compatibleClassNames"`
}
