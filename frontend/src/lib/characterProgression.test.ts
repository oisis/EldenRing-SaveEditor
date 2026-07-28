import { describe, it, expect } from 'vitest';
import { calculateLevel, minimumSoulMemoryForLevel, sumStats } from './characterProgression';

// Test vectors are shared with the backend (backend/vm/validation_test.go
// TestRecalculateLevel and the runes-cost formula). Keep them in sync.

const base = {
    vigor: 10,
    mind: 10,
    endurance: 10,
    strength: 10,
    dexterity: 10,
    intelligence: 10,
    faith: 10,
    arcane: 10,
};

describe('calculateLevel', () => {
    it('Wretch base stats (8x10 = 80) → level 1', () => {
        // sum 80 - 79 = 1
        expect(calculateLevel(base)).toBe(1);
    });

    it('never drops below 1', () => {
        const low = { ...base, vigor: 1, mind: 1, endurance: 1, strength: 1, dexterity: 1, intelligence: 1, faith: 1, arcane: 1 };
        // sum 8 - 79 → clamped to 1
        expect(calculateLevel(low)).toBe(1);
    });

    it('all 99 → max level 713', () => {
        const maxed = { vigor: 99, mind: 99, endurance: 99, strength: 99, dexterity: 99, intelligence: 99, faith: 99, arcane: 99 };
        // 792 - 79 = 713
        expect(calculateLevel(maxed)).toBe(713);
    });

    it('raising one stat by N raises level by N', () => {
        expect(calculateLevel({ ...base, vigor: 60 })).toBe(calculateLevel(base) + 50);
    });

    it('ignores non-finite entries', () => {
        expect(sumStats({ ...base, vigor: NaN })).toBe(70);
    });
});

describe('minimumSoulMemoryForLevel', () => {
    it('level 1 costs 0', () => {
        expect(minimumSoulMemoryForLevel(1)).toBe(0);
    });

    it('is monotonically non-decreasing', () => {
        let prev = 0;
        for (let lvl = 1; lvl <= 150; lvl++) {
            const sm = minimumSoulMemoryForLevel(lvl);
            expect(sm).toBeGreaterThanOrEqual(prev);
            prev = sm;
        }
    });

    it.each([
        [9, 473],
        [50, 256_598],
        [150, 7_106_585],
        [713, 1_692_560_963],
    ])('matches the backend reference vector at level %i', (level, expected) => {
        expect(minimumSoulMemoryForLevel(level)).toBe(expected);
    });
});
