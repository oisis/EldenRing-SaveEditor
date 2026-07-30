package schema

type SourceID string

type EvidenceLevel string

const (
	EvidenceRegulation       EvidenceLevel = "regulation"
	EvidenceVerifiedResearch EvidenceLevel = "verified_research"
	EvidenceCurated          EvidenceLevel = "curated"
	EvidenceUnknown          EvidenceLevel = "unknown"
)

type DataSource struct {
	ID       SourceID
	Kind     string
	Location string
	Version  string
	Evidence EvidenceLevel
	Reviewed bool
}

type Manifest struct {
	SchemaVersion uint32
	DataVersion   string
	GameVersion   string
	Sources       []DataSource
}

type Provenance struct {
	Source SourceID
	Method string
}

type Fact[T any] struct {
	Known      bool
	Value      T
	Provenance Provenance
}
