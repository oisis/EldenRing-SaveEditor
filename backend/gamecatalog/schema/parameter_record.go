package schema

type ParameterRecord struct {
	Table      string           `json:"table"`
	RowID      int64            `json:"rowID"`
	Fields     []ParameterField `json:"fields,omitempty"`
	Provenance Provenance       `json:"provenance"`
}

type ParameterField struct {
	Name string `json:"name"`
}
