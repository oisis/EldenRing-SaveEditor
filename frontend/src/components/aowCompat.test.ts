import { describe, it, expect } from 'vitest';
import { getAoWCompatStatus } from './WeaponEditModal';
import { WEP_TYPE_TO_BITS, AOW_HEURISTIC_WEPTYPES } from '../data/aowCompat.generated';

// Bit helper: mask with a single bit set, as a Number (safe up to bit 52).
const bit = (n: number) => 2 ** n;

describe('generated wepType → bit mapping (parity with backend)', () => {
    it('maps the fixed DLC / torch wepTypes to the corrected bits', () => {
        expect(WEP_TYPE_TO_BITS[87]).toEqual([35]); // Torch (was Bow bit 25)
        expect(WEP_TYPE_TO_BITS[88]).toEqual([36]); // Hand-to-Hand
        expect(WEP_TYPE_TO_BITS[89]).toEqual([37]); // Perfume Bottles
        expect(WEP_TYPE_TO_BITS[90]).toEqual([38]); // Dueling/Thrusting Shields (was dead bit 43)
        expect(WEP_TYPE_TO_BITS[91]).toEqual([39]); // Throwing/Smithscript Blades
        expect(WEP_TYPE_TO_BITS[94]).toEqual([6]);  // Great Katana → base katana bit
        expect(WEP_TYPE_TO_BITS[95]).toEqual([21]); // Beast Claw → base claw bit
    });

    it('drops wepType 68 (no real weapon uses it) and the heuristic-only classes', () => {
        expect(WEP_TYPE_TO_BITS[68]).toBeUndefined();
        expect(WEP_TYPE_TO_BITS[92]).toBeUndefined(); // resolved via heuristic
        expect(WEP_TYPE_TO_BITS[93]).toBeUndefined(); // resolved via heuristic
    });

    it('exposes the DLC heuristic arts', () => {
        expect(AOW_HEURISTIC_WEPTYPES[0x800631F0]).toEqual([95]); // Raging Beast
        expect(AOW_HEURISTIC_WEPTYPES[0x80063DA8]).toEqual([92]); // Blind Spot
        expect(AOW_HEURISTIC_WEPTYPES[0x80064578]).toEqual([94]); // Overhead Stance
        expect(AOW_HEURISTIC_WEPTYPES[0x80064960]).toEqual([93]); // Wing Stance
    });
});

describe('getAoWCompatStatus', () => {
    it('shows torch-compatible AoW as compatible on torches (wepType 87 → Torch bit 35)', () => {
        expect(getAoWCompatStatus(0x1234, bit(35), 87)).toBe('compatible');
    });

    it('does not leak a Bow-only AoW onto a torch', () => {
        expect(getAoWCompatStatus(0x1234, bit(25), 87)).toBe('incompatible');
    });

    it('mounts a reserved-bit AoW on Dueling Shields (wepType 90 → bit 38)', () => {
        expect(getAoWCompatStatus(0x1234, bit(38), 90)).toBe('compatible');
        expect(getAoWCompatStatus(0x1234, bit(39), 90)).toBe('incompatible'); // only Throwing bit set
    });

    it('resolves DLC heuristic arts and blocks them on other classes', () => {
        // Raging Beast (mask 0) is compatible only with Beast Claws (95).
        expect(getAoWCompatStatus(0x800631F0, 0, 95)).toBe('compatible');
        expect(getAoWCompatStatus(0x800631F0, 0, 41)).toBe('incompatible'); // normal claw
        expect(getAoWCompatStatus(0x800631F0, 0, 1)).toBe('incompatible');  // dagger
    });

    it('fail-closes on unknown data', () => {
        expect(getAoWCompatStatus(0x999, 0, 1)).toBe('unknown');   // no mask, no heuristic
        expect(getAoWCompatStatus(0x999, bit(0), 0)).toBe('unknown'); // unknown wepType
        expect(getAoWCompatStatus(0x999, bit(0), 12345)).toBe('unknown'); // wepType not mapped
    });
});
