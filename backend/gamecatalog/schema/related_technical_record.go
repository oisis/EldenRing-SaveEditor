package schema

type TechnicalRecordKind string

const (
	TechnicalRecordAppearanceState TechnicalRecordKind = "appearance_state"
)

type RelatedTechnicalRecord struct {
	Kind             Fact[TechnicalRecordKind] `json:"kind"`
	GameID           Fact[uint32]              `json:"gameID"`
	Description      ItemDescriptionRecord     `json:"description"`
	GameMaxInventory Fact[uint32]              `json:"gameMaxInventory"`
	GameMaxStorage   Fact[uint32]              `json:"gameMaxStorage"`
	SourceRecords    []ParameterRecord         `json:"sourceRecords"`
}
