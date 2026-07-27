import { describe, it, expect } from 'vitest';
import { getAoWCompatStatus } from './WeaponEditModal';
import { WEP_TYPE_TO_BITS, AOW_HEURISTIC_WEPTYPES } from '../data/aowCompat.generated';

// Bit helper: mask with a single bit set, as a Number (safe up to bit 52).
const bit = (n: number) => 2 ** n;

describe('generated wepType → bit mapping (parity with backend)', () => {
    it('maps every base-game weapon class to its matching canMountWep column', () => {
        expect(WEP_TYPE_TO_BITS[9]).toEqual([4]);   // Curved Sword
        expect(WEP_TYPE_TO_BITS[14]).toEqual([7]);  // Twinblade
        expect(WEP_TYPE_TO_BITS[17]).toEqual([10]); // Axe
        expect(WEP_TYPE_TO_BITS[24]).toEqual([14]); // Flail
        expect(WEP_TYPE_TO_BITS[29]).toEqual([18]); // Halberd
        expect(WEP_TYPE_TO_BITS[41]).toEqual([23]); // Colossal Weapon
        expect(WEP_TYPE_TO_BITS[50]).toEqual([24]); // Light Bow
        expect(WEP_TYPE_TO_BITS[51]).toEqual([25]); // Bow
        expect(WEP_TYPE_TO_BITS[53]).toEqual([26]); // Greatbow
        expect(WEP_TYPE_TO_BITS[55]).toEqual([27]); // Crossbow
        expect(WEP_TYPE_TO_BITS[56]).toEqual([28]); // Ballista
        expect(WEP_TYPE_TO_BITS[57]).toEqual([29]); // Glintstone Staff
        expect(WEP_TYPE_TO_BITS[61]).toEqual([30]); // Sacred Seal
        expect(WEP_TYPE_TO_BITS[67]).toEqual([33]); // Medium Shield
    });

    it('maps the fixed DLC / torch wepTypes to the corrected bits', () => {
        expect(WEP_TYPE_TO_BITS[87]).toEqual([35]); // Torch (was Bow bit 25)
        expect(WEP_TYPE_TO_BITS[88]).toEqual([36]); // Hand-to-Hand
        expect(WEP_TYPE_TO_BITS[89]).toEqual([37]); // Perfume Bottles
        expect(WEP_TYPE_TO_BITS[90]).toEqual([38]); // Dueling/Thrusting Shields (was dead bit 43)
        expect(WEP_TYPE_TO_BITS[91]).toEqual([39]); // Throwing/Smithscript Blades
        expect(WEP_TYPE_TO_BITS[92]).toEqual([40]); // Backhand Blades
        expect(WEP_TYPE_TO_BITS[93]).toEqual([41]); // Light Greatswords
        expect(WEP_TYPE_TO_BITS[94]).toEqual([42]); // Great Katanas
        expect(WEP_TYPE_TO_BITS[95]).toEqual([43]); // Beast Claws
    });

    it('drops wepType 68 because no real weapon uses it', () => {
        expect(WEP_TYPE_TO_BITS[68]).toBeUndefined();
    });

    it('does not need heuristic entries for current DLC regulation data', () => {
        expect(AOW_HEURISTIC_WEPTYPES).toEqual({});
    });
});

describe('getAoWCompatStatus', () => {
    it('shows torch-compatible AoW as compatible on torches (wepType 87 → Torch bit 35)', () => {
        expect(getAoWCompatStatus(0x1234, bit(35), 87)).toBe('compatible');
    });

    it('does not leak a Bow-only AoW onto a torch', () => {
        expect(getAoWCompatStatus(0x1234, bit(25), 87)).toBe('incompatible');
    });

    it('keeps light bow, bow and greatbow AoWs in their own classes', () => {
        expect(getAoWCompatStatus(0x80009CA4, bit(24), 50)).toBe('compatible'); // Barrage
        expect(getAoWCompatStatus(0x80009CA4, bit(24), 51)).toBe('incompatible');
        expect(getAoWCompatStatus(0x80009D08, bit(24) + bit(25), 50)).toBe('compatible'); // Mighty Shot
        expect(getAoWCompatStatus(0x80009D08, bit(24) + bit(25), 51)).toBe('compatible');
        expect(getAoWCompatStatus(0x80009D08, bit(24) + bit(25), 53)).toBe('incompatible');
        expect(getAoWCompatStatus(0x80009C40, bit(26), 53)).toBe('compatible'); // Through and Through
        expect(getAoWCompatStatus(0x80009C40, bit(26), 50)).toBe('incompatible');
    });

    it('mounts a reserved-bit AoW on Dueling Shields (wepType 90 → bit 38)', () => {
        expect(getAoWCompatStatus(0x1234, bit(38), 90)).toBe('compatible');
        expect(getAoWCompatStatus(0x1234, bit(39), 90)).toBe('incompatible'); // only Throwing bit set
    });

    it('uses direct DLC bits for Milady and the other expansion classes', () => {
        expect(getAoWCompatStatus(0x80064960, bit(41), 93)).toBe('compatible'); // Wing Stance
        expect(getAoWCompatStatus(0x80002774, bit(41), 93)).toBe('compatible'); // Impaling Thrust
        expect(getAoWCompatStatus(0x80002CEC, bit(1), 93)).toBe('incompatible'); // Square Off
        expect(getAoWCompatStatus(0x800631F0, bit(43), 95)).toBe('compatible'); // Raging Beast
        expect(getAoWCompatStatus(0x800631F0, bit(43), 41)).toBe('incompatible');
    });

    it('fail-closes on unknown data', () => {
        expect(getAoWCompatStatus(0x999, 0, 1)).toBe('unknown');   // no mask, no heuristic
        expect(getAoWCompatStatus(0x999, bit(0), 0)).toBe('unknown'); // unknown wepType
        expect(getAoWCompatStatus(0x999, bit(0), 12345)).toBe('unknown'); // wepType not mapped
    });
});
