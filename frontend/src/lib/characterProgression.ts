// characterProgression — shared, framework-free helpers for the two
// Elden Ring progression formulas the UI needs client-side:
//
//   - calculateLevel: Level = max(1, sum(8 stats) - 79)
//   - minimumSoulMemoryForLevel: total runes required to reach a level
//
// Both mirror the authoritative Go implementations exactly
// (backend/vm/validation.go RecalculateLevel and
// backend/vm/character_vm.go runesCostForLevel / MinimumSoulMemoryForLevel).
// The backend stays the source of truth — it re-validates every apply — but
// CharacterTab and the Templates "Apply with overrides" panel both need a
// synchronous preview, so this single module is shared by both instead of
// each keeping a private copy that could drift.

const U32_MAX = 4_294_967_295;

export interface StatBlock {
    vigor: number;
    mind: number;
    endurance: number;
    strength: number;
    dexterity: number;
    intelligence: number;
    faith: number;
    arcane: number;
}

export const STAT_KEYS: ReadonlyArray<keyof StatBlock> = [
    'vigor',
    'mind',
    'endurance',
    'strength',
    'dexterity',
    'intelligence',
    'faith',
    'arcane',
];

// sumStats totals the eight attributes. Non-finite / missing entries count as 0
// so a partially-filled draft never yields NaN.
export function sumStats(stats: StatBlock): number {
    let sum = 0;
    for (const k of STAT_KEYS) {
        const v = stats[k];
        if (Number.isFinite(v)) sum += v;
    }
    return sum;
}

// calculateLevel mirrors vm.RecalculateLevel: Level = max(1, sum - 79).
export function calculateLevel(stats: StatBlock): number {
    return Math.max(1, sumStats(stats) - 79);
}

// minimumSoulMemoryForLevel mirrors vm.MinimumSoulMemoryForLevel /
// runesCostForLevel: the cumulative runes cost to reach `level`, using the
// official per-level cost formula floored to 0 for low levels and clamped to
// uint32. Level <= 1 costs 0.
export function minimumSoulMemoryForLevel(level: number): number {
    let total = 0;
    for (let n = 2; n <= level; n++) {
        let cost = Math.trunc(
            (2 * n * n * n + 306 * n * n + 10_560 * n - 89_500) / 100,
        );
        if (n === 45 || n === 205 || n === 257 || n === 282 || n === 410 || n === 707) {
            cost--;
        }
        if (cost > 0) total += cost;
    }
    return Math.min(total, U32_MAX);
}
