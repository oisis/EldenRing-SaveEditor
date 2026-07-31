package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

const validRegulationFixture = "Row ID;Name;Value\n0;Zero;empty\n1;One;plain\n1000;Weapon;\"value;with:semicolon\"\n"

func TestReadRegulationCSVDirectory(t *testing.T) {
	directory := t.TempDir()
	for _, spec := range regulationTableSpecs {
		path := filepath.Join(directory, spec.filename)
		if err := os.WriteFile(path, []byte(validRegulationFixture), 0o600); err != nil {
			t.Fatalf("write %s: %v", spec.filename, err)
		}
	}

	data, err := ReadRegulationCSVDirectory(directory)
	if err != nil {
		t.Fatalf("ReadRegulationCSVDirectory: %v", err)
	}
	if len(data.tables) != len(regulationTableSpecs) {
		t.Fatalf("loaded table count = %d, want %d", len(data.tables), len(regulationTableSpecs))
	}
}

func TestRegulationReaderPreservesFieldsOrderProvenanceAndImmutability(t *testing.T) {
	data, err := readRegulationFS(newRegulationMapFS())
	if err != nil {
		t.Fatalf("readRegulationFS: %v", err)
	}

	table, exists := data.Table(RegulationTableWeapon)
	if !exists {
		t.Fatal("weapon table is not loaded")
	}
	if table.RowCount() != 3 {
		t.Fatalf("RowCount = %d, want 3", table.RowCount())
	}

	source := table.Source()
	if source.Location != "regulation.bin/csv/EquipParamWeapon.csv" {
		t.Fatalf("source location = %q", source.Location)
	}
	sum := sha256.Sum256([]byte(validRegulationFixture))
	if source.Version != hex.EncodeToString(sum[:]) {
		t.Fatalf("source version = %q, want SHA-256 %x", source.Version, sum)
	}

	row, exists := table.Row(1000)
	if !exists {
		t.Fatal("Row ID 1000 is not indexed")
	}
	wantFields := []ParameterField{
		{Name: "Row ID", RawValue: "1000"},
		{Name: "Name", RawValue: "Weapon"},
		{Name: "Value", RawValue: "value;with:semicolon"},
	}
	if len(row.Fields) != len(wantFields) {
		t.Fatalf("field count = %d, want %d", len(row.Fields), len(wantFields))
	}
	for index := range wantFields {
		if row.Fields[index] != wantFields[index] {
			t.Fatalf("field %d = %+v, want %+v", index, row.Fields[index], wantFields[index])
		}
	}
	if value, ok := row.Field("Value"); !ok || value != "value;with:semicolon" {
		t.Fatalf("Field(Value) = %q, %t", value, ok)
	}

	row.Fields[0].RawValue = "mutated"
	again, _ := table.Row(1000)
	if again.Fields[0].RawValue != "1000" {
		t.Fatalf("stored row mutated through returned slice: %q", again.Fields[0].RawValue)
	}

	rows := table.Rows()
	if len(rows) != 3 || rows[0].RowID != 0 || rows[1].RowID != 1 || rows[2].RowID != 1000 {
		t.Fatalf("Rows order = %+v", rows)
	}
}

func TestRegulationReaderRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "missing header", content: "", want: "missing header"},
		{name: "missing Row ID", content: "Name;Value\nOne;1\n", want: `missing required "Row ID" header`},
		{name: "duplicate header", content: "Row ID;Name;Name\n1;a;b\n", want: `duplicate header column "Name"`},
		{name: "invalid Row ID", content: "Row ID;Name\nnot-an-id;One\n", want: "not an unsigned 32-bit decimal integer"},
		{name: "negative Row ID", content: "Row ID;Name\n-1;One\n", want: "not an unsigned 32-bit decimal integer"},
		{name: "duplicate Row ID", content: "Row ID;Name\n1;One\n1;Duplicate\n", want: "duplicate Row ID 1"},
		{name: "short row", content: "Row ID;Name;Value\n1;One\n", want: "wrong number of fields"},
		{name: "malformed quote", content: "Row ID;Name\n1;\"unterminated\n", want: "extraneous or missing"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := newRegulationMapFS()
			source["EquipParamWeapon.csv"] = &fstest.MapFile{Data: []byte(test.content)}

			_, err := readRegulationFS(source)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestRegulationReaderRejectsMissingRequiredTable(t *testing.T) {
	source := newRegulationMapFS()
	delete(source, "Magic.csv")

	_, err := readRegulationFS(source)
	if err == nil || !strings.Contains(err.Error(), "read Magic.csv") {
		t.Fatalf("error = %v, want missing Magic.csv", err)
	}
}

func newRegulationMapFS() fstest.MapFS {
	source := make(fstest.MapFS, len(regulationTableSpecs))
	for _, spec := range regulationTableSpecs {
		source[spec.filename] = &fstest.MapFile{Data: []byte(validRegulationFixture)}
	}
	return source
}
