package migration

import (
	"fmt"
	"math"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type legacyCandidate[T comparable] struct {
	available bool
	value     T
	source    string
}

func saveForgeValue[T comparable](
	available bool,
	legacy T,
	authoritative T,
	method string,
) *schema.Fact[T] {
	if !available || legacy == authoritative {
		return nil
	}
	fact := knownLegacyFact(legacy, method)
	return &fact
}

func legacyConsensus[T comparable](
	field string,
	candidates ...legacyCandidate[T],
) (T, bool, string, error) {
	var result T
	var source string
	found := false
	for _, candidate := range candidates {
		if !candidate.available {
			continue
		}
		if !found {
			result = candidate.value
			source = candidate.source
			found = true
			continue
		}
		if candidate.value != result {
			return result, false, "", fmt.Errorf(
				"legacy %s conflicts between %s and %s",
				field,
				source,
				candidate.source,
			)
		}
	}
	return result, found, source, nil
}

func saveForgeConsensusValue[T comparable](
	field string,
	authoritative T,
	candidates ...legacyCandidate[T],
) (*schema.Fact[T], error) {
	conflicts := make([]legacyCandidate[T], 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.available && candidate.value != authoritative {
			conflicts = append(conflicts, candidate)
		}
	}
	legacy, available, sources, err := legacyConsensus(field, conflicts...)
	if err != nil {
		return nil, err
	}
	return saveForgeValue(
		available,
		legacy,
		authoritative,
		"preserved conflicting SaveForge value from "+sources,
	), nil
}

func saveForgeWeightValue(
	authoritative float64,
	candidates ...legacyCandidate[float64],
) (*schema.Fact[float64], error) {
	const decimalPlaces = 10
	normalize := func(value float64) float64 {
		return math.Round(value*decimalPlaces) / decimalPlaces
	}
	normalizedAuthoritative := normalize(authoritative)
	var legacy legacyCandidate[float64]
	for _, candidate := range candidates {
		if !candidate.available ||
			normalize(candidate.value) == normalizedAuthoritative {
			continue
		}
		if !legacy.available {
			legacy = candidate
			continue
		}
		if normalize(candidate.value) != normalize(legacy.value) {
			return nil, fmt.Errorf(
				"legacy weight conflicts between %s and %s",
				legacy.source,
				candidate.source,
			)
		}
	}
	return saveForgeValue(
		legacy.available,
		legacy.value,
		authoritative,
		"preserved conflicting SaveForge value from "+legacy.source+
			" after one-decimal weight normalization",
	), nil
}
