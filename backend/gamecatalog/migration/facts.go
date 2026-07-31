package migration

import (
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func knownLegacyFact[T any](value T, method string) schema.Fact[T] {
	return schema.Fact[T]{
		Known: true,
		Value: value,
		Provenance: schema.Provenance{
			Source: sourceLegacyData,
			Method: method,
		},
	}
}

func knownIconFact(value string) schema.Fact[string] {
	return schema.Fact[string]{
		Known: true,
		Value: value,
		Provenance: schema.Provenance{
			Source: sourceLegacyIcons,
			Method: "copied once from the shared legacy item asset",
		},
	}
}

func knownRegulationFact[T any](
	value T,
	table RegulationTableName,
	field string,
	rowID uint32,
) schema.Fact[T] {
	return schema.Fact[T]{
		Known: true,
		Value: value,
		Provenance: schema.Provenance{
			Source: sourceIDByRegulationTable[table],
			Method: "parsed " + field + " from row " + decimalRowID(rowID),
			Table:  string(table),
			Row:    decimalRowID(rowID),
			Field:  regulationFieldReference(field),
		},
	}
}

func knownRegulationDerivedFact[T any](
	value T,
	table RegulationTableName,
	method string,
	rowID uint32,
	fields string,
) schema.Fact[T] {
	return schema.Fact[T]{
		Known: true,
		Value: value,
		Provenance: schema.Provenance{
			Source: sourceIDByRegulationTable[table],
			Method: method,
			Table:  string(table),
			Row:    decimalRowID(rowID),
			Field:  fields,
		},
	}
}

func regulationFieldReference(description string) string {
	switch {
	case strings.HasPrefix(description, "Row ID"):
		return "Row ID"
	case strings.HasPrefix(description, "base 100 plus "):
		return strings.TrimPrefix(description, "base 100 plus ")
	case strings.HasPrefix(description, "verified nonzero "):
		return strings.TrimPrefix(description, "verified nonzero ")
	default:
		field, _, _ := strings.Cut(description, " ")
		return field
	}
}

func unknownLegacyFact[T any](method string) schema.Fact[T] {
	return schema.Fact[T]{
		Provenance: schema.Provenance{
			Source: sourceLegacyUnknown,
			Method: method,
		},
	}
}

func unknownCatalogFact[T any](method string) schema.Fact[T] {
	return schema.Fact[T]{
		Provenance: schema.Provenance{
			Source: sourceLegacyData,
			Method: method,
		},
	}
}
