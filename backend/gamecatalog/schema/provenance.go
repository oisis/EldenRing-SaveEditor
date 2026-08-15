package schema

import "strings"

type SourceID string

const SourceSaveForgeLegacy SourceID = "legacy_db_data"

const (
	MinimumSchemaVersion uint32 = 1
	CurrentSchemaVersion uint32 = 11
)

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

// NotApplicableMethodPrefix opens the Provenance method of a fact whose field
// cannot apply to the item that carries it. Such a fact stays unknown with a
// zero value and keeps complete provenance.
const NotApplicableMethodPrefix = "not applicable"

// MarksNotApplicable reports whether the provenance declares its fact as not
// applicable instead of merely unresolved.
func (provenance Provenance) MarksNotApplicable() bool {
	return strings.HasPrefix(provenance.Method, NotApplicableMethodPrefix)
}

type Fact[T any] struct {
	Known      bool       `json:"known"`
	Value      T          `json:"value"`
	Provenance Provenance `json:"provenance"`
}
