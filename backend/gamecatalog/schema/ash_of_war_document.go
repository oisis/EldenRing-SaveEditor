package schema

type AshOfWarData struct {
	SourceRowID          Fact[uint32]   `json:"sourceRowID"`
	IconID               Fact[uint32]   `json:"iconID"`
	SortID               Fact[uint32]   `json:"sortID"`
	SortGroupID          Fact[uint8]    `json:"sortGroupID"`
	SwordArtsParamID     Fact[int32]    `json:"swordArtsParamID"`
	SwordArtsName        Fact[string]   `json:"swordArtsName"`
	DefaultAffinity      Fact[uint8]    `json:"defaultAffinity"`
	CompatibilityMask    Fact[uint64]   `json:"compatibilityMask"`
	CompatibleClassNames Fact[[]string] `json:"compatibleClassNames"`
}
