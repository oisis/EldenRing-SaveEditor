package schema

type SourceID string

type EvidenceLevel string

const (
	EvidenceRegulation       EvidenceLevel = "regulation"
	EvidenceGameData         EvidenceLevel = "game_data"
	EvidenceVerifiedResearch EvidenceLevel = "verified_research"
	EvidenceCurated          EvidenceLevel = "curated"
	EvidenceUnknown          EvidenceLevel = "unknown"
)

type DataSource struct {
	ID       SourceID      `json:"id"`
	Kind     string        `json:"kind"`
	Location string        `json:"location"`
	Version  string        `json:"version"`
	Evidence EvidenceLevel `json:"evidence"`
	Reviewed bool          `json:"reviewed"`
}

type Manifest struct {
	SchemaVersion uint32       `json:"schemaVersion"`
	DataVersion   string       `json:"dataVersion"`
	GameVersion   string       `json:"gameVersion"`
	Sources       []DataSource `json:"sources"`
}

type Provenance struct {
	Source SourceID `json:"source"`
	Method string   `json:"method"`
	Table  string   `json:"table,omitempty"`
	Row    string   `json:"row,omitempty"`
	Field  string   `json:"field,omitempty"`
}

type Fact[T any] struct {
	Known      bool       `json:"known"`
	Value      T          `json:"value"`
	Provenance Provenance `json:"provenance"`
}
