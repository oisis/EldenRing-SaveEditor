package schema

type AshOfWarData struct {
	SourceRowID       Fact[uint32]
	CompatibilityMask Fact[uint64]
}
