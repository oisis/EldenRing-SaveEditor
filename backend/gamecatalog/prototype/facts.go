package prototype

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func provenance(source schema.SourceID, method string) schema.Provenance {
	return schema.Provenance{
		Source: source,
		Method: method,
	}
}

func known[T any](value T, provenance schema.Provenance) schema.Fact[T] {
	return schema.Fact[T]{
		Known:      true,
		Value:      value,
		Provenance: provenance,
	}
}

func unknownFact[T any](provenance schema.Provenance) schema.Fact[T] {
	return schema.Fact[T]{
		Provenance: provenance,
	}
}

func enabled[T any](rules T, provenance schema.Provenance) schema.Capability[T] {
	return schema.Capability[T]{
		Known:      true,
		Enabled:    true,
		Rules:      &rules,
		Provenance: provenance,
	}
}

func disabled[T any](provenance schema.Provenance) schema.Capability[T] {
	return schema.Capability[T]{
		Known:      true,
		Provenance: provenance,
	}
}
