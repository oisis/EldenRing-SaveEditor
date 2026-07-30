package schema

type AshOfWarData struct {
	SourceRowID       Fact[uint32] `json:"sourceRowID"`
	CompatibilityMask Fact[uint64] `json:"compatibilityMask"`
}
