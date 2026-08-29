import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { db } from '../../wailsjs/go/models';

if (typeof globalThis.localStorage === 'undefined') {
    const store = new Map<string, string>();
    Object.defineProperty(globalThis, 'localStorage', {
        configurable: true,
        value: {
            getItem: (k: string) => (store.has(k) ? store.get(k)! : null),
            setItem: (k: string, v: string) => { store.set(k, String(v)); },
            removeItem: (k: string) => { store.delete(k); },
            clear: () => { store.clear(); },
            key: (i: number) => Array.from(store.keys())[i] ?? null,
            get length() { return store.size; },
        },
    });
}

vi.mock('../../wailsjs/go/main/App', () => ({
    GetGraces: vi.fn().mockResolvedValue([]),
    SetGraceVisited: vi.fn().mockResolvedValue(undefined),
    GetBosses: vi.fn().mockResolvedValue([]),
    SetBossDefeated: vi.fn().mockResolvedValue(undefined),
    GetSummoningPools: vi.fn().mockResolvedValue([]),
    SetSummoningPoolActivated: vi.fn().mockResolvedValue(undefined),
    GetColosseums: vi.fn().mockResolvedValue([]),
    SetColosseumUnlocked: vi.fn().mockResolvedValue(undefined),
    GetMapProgress: vi.fn().mockResolvedValue([]),
    SetMapFlag: vi.fn().mockResolvedValue(undefined),
    SetMapRegionFlags: vi.fn().mockResolvedValue(undefined),
    RevealAllMap: vi.fn().mockResolvedValue(undefined),
    ResetMapExploration: vi.fn().mockResolvedValue(undefined),
    RemoveFogOfWar: vi.fn().mockResolvedValue(undefined),
    GetCookbooks: vi.fn().mockResolvedValue([]),
    SetCookbookUnlocked: vi.fn().mockResolvedValue(undefined),
    BulkSetCookbooksUnlocked: vi.fn().mockResolvedValue(undefined),
    GetGestures: vi.fn().mockResolvedValue([]),
    SetGestureUnlocked: vi.fn().mockResolvedValue(undefined),
    BulkSetGesturesUnlocked: vi.fn().mockResolvedValue(undefined),
    GetQuestNPCs: vi.fn().mockResolvedValue([]),
    GetQuestProgress: vi.fn().mockResolvedValue(null),
    SetQuestStep: vi.fn().mockResolvedValue(undefined),
    GetBellBearings: vi.fn().mockResolvedValue([]),
    SetBellBearingUnlocked: vi.fn().mockResolvedValue(undefined),
    BulkSetBellBearings: vi.fn().mockResolvedValue(undefined),
    GetWhetblades: vi.fn().mockResolvedValue([]),
    SetWhetbladeUnlocked: vi.fn().mockResolvedValue(undefined),
    GetUnlockedRegions: vi.fn().mockResolvedValue([]),
    SetRegionUnlocked: vi.fn().mockResolvedValue(undefined),
    BulkSetUnlockedRegions: vi.fn().mockResolvedValue(undefined),
    RecordDiagnosticWorldAction: vi.fn().mockResolvedValue(undefined),
    GetSpectralSteedAttire: vi.fn(),
    SetSpectralSteedAttire: vi.fn().mockResolvedValue(undefined),
    LockAllSpectralSteedAttires: vi.fn().mockResolvedValue(undefined),
    AddItemsToCharacter: vi.fn(),
}));

vi.mock('../lib/toast', () => ({
    default: { success: vi.fn(), error: vi.fn(), loading: vi.fn(), dismiss: vi.fn() },
}));

vi.mock('../state/safetyMode', () => ({
    useSafetyMode: () => ({ enabled: false, tier: 0, requireConfirmFor: () => false }),
}));

import {
    AddItemsToCharacter,
    GetSpectralSteedAttire,
    LockAllSpectralSteedAttires,
    SetSpectralSteedAttire,
} from '../../wailsjs/go/main/App';
import { RISK_INFO } from '../data/riskInfo';
import toast from '../lib/toast';
import { WorldTab } from './WorldTab';

const TREE_SENTINEL_ITEM = 0x401EAA00;
const SILVER_OF_CARIA_ITEM = 0x401EAA0A;
const FUNEREAL_NIGHT_ITEM = 0x401EAA14;

// Mirrors the backend db.SpectralSteedAttireState shape.
function attireState(overrides: Partial<{ activeId: number; status: string; owned: number[] }> = {}) {
    const owned = overrides.owned ?? [];
    return {
        status: overrides.status ?? 'resolved',
        activeId: overrides.activeId ?? 6700,
        entries: [
            { id: 6700, name: 'Default Appearance', itemId: 0, iconPath: '', owned: true },
            { id: 6701, name: 'Tree Sentinel Spectral Steed Attire', itemId: TREE_SENTINEL_ITEM, iconPath: 'items/key_items/tree_sentinel_spectral_steed_attire.png', owned: owned.includes(6701) },
            { id: 6702, name: 'Silver of Caria Spectral Steed Attire', itemId: SILVER_OF_CARIA_ITEM, iconPath: 'items/key_items/silver_of_caria_spectral_steed_attire.png', owned: owned.includes(6702) },
            { id: 6703, name: 'Funereal Night Spectral Steed Attire', itemId: FUNEREAL_NIGHT_ITEM, iconPath: 'items/key_items/funereal_night_spectral_steed_attire.png', owned: owned.includes(6703) },
        ],
    } as unknown as db.SpectralSteedAttireState;
}

async function openAttireSection(platform = 'PC') {
    render(<WorldTab charIdx={0} platform={platform} />);
    fireEvent.click(await screen.findByRole('button', { name: /unlocks/i }));
    const header = await screen.findByText('Spectral Steed Attire');
    fireEvent.click(header);
    await screen.findByTestId('attire-row-6700');
}

const row = (id: number) => within(screen.getByTestId(`attire-row-${id}`));
const addButton = (id: number) => row(id).getByRole('button', { name: 'Add' });
const setButton = (id: number) => row(id).getByRole('button', { name: 'Set' });
const attireHeader = () => {
    const header = screen.getByText('Spectral Steed Attire').closest('button');
    if (!header) throw new Error('missing Spectral Steed Attire header button');
    return within(header);
};

describe('WorldTab — Spectral Steed Attire', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        localStorage.clear();
        vi.mocked(GetSpectralSteedAttire).mockResolvedValue(attireState({ owned: [6702] }));
        vi.mocked(AddItemsToCharacter).mockResolvedValue({ added: 1, requested: 1, capHit: '' } as never);
    });

    it('renders exactly four appearances and the DLC warning', async () => {
        await openAttireSection();
        expect(screen.getByText(RISK_INFO.tarnished_edition_dlc.whyBan)).toBeTruthy();
        for (const id of [6700, 6701, 6702, 6703]) {
            expect(screen.getByTestId(`attire-row-${id}`)).toBeTruthy();
        }
        expect(screen.queryByTestId('attire-row-6704')).toBeNull();
    });

    it('shows owned progress and bulk actions in the collapsed section header', async () => {
        vi.mocked(GetSpectralSteedAttire).mockResolvedValue(attireState());
        render(<WorldTab charIdx={0} platform="PC" />);
        fireEvent.click(await screen.findByRole('button', { name: /unlocks/i }));
        await screen.findByText('Spectral Steed Attire');
        expect(attireHeader().getByText('1/4')).toBeTruthy();
        expect(attireHeader().getByRole('button', { name: 'Unlock All' })).toBeTruthy();
        expect(attireHeader().getByRole('button', { name: 'Lock All' })).toBeTruthy();
    });

    it('gives the default appearance no Add button and no DLC confirmation', async () => {
        await openAttireSection();
        expect(row(6700).queryByRole('button', { name: 'Add' })).toBeNull();
        // The only risk affordance in the default row would be the ⚠ info icon
        // that RiskActionButton renders next to a risk-carrying button.
        expect(row(6700).queryByLabelText(/why is this risky/i)).toBeNull();
        expect(row(6701).getByRole('button', { name: 'Add' })).toBeTruthy();
    });

    it('opens and closes an enlarged attire preview from the thumbnail', async () => {
        await openAttireSection();
        fireEvent.click(row(6701).getByRole('button', { name: 'Preview Tree Sentinel Spectral Steed Attire' }));
        const dialog = screen.getByRole('dialog', { name: 'Spectral Steed Attire preview' });
        const image = within(dialog).getByRole('img', { name: 'Tree Sentinel Spectral Steed Attire' });
        expect(image.getAttribute('src')).toBe('items/key_items/tree_sentinel_spectral_steed_attire.png');
        expect(image).toHaveClass('h-96', 'w-96');
        fireEvent.click(within(dialog).getByRole('button', { name: 'Close attire preview' }));
        expect(screen.queryByRole('dialog', { name: 'Spectral Steed Attire preview' })).toBeNull();
    });

    it('disables Add for an owned attire and Set without the item', async () => {
        await openAttireSection();
        expect((addButton(6702) as HTMLButtonElement).disabled).toBe(true);
        expect((addButton(6701) as HTMLButtonElement).disabled).toBe(false);
        expect((setButton(6701) as HTMLButtonElement).disabled).toBe(true);
        expect((setButton(6702) as HTMLButtonElement).disabled).toBe(false);
    });

    it('disables Set for the active appearance', async () => {
        vi.mocked(GetSpectralSteedAttire).mockResolvedValue(attireState({ activeId: 6702, owned: [6702] }));
        await openAttireSection();
        expect((setButton(6702) as HTMLButtonElement).disabled).toBe(true);
        expect((setButton(6700) as HTMLButtonElement).disabled).toBe(false);
    });

    it('Add uses AddItemsToCharacter with one item, Inventory=1 and Storage=0, then refetches', async () => {
        await openAttireSection();
        fireEvent.click(addButton(6703));
        await waitFor(() => expect(AddItemsToCharacter).toHaveBeenCalledTimes(1));
        expect(vi.mocked(AddItemsToCharacter).mock.calls[0]).toEqual([0, [FUNEREAL_NIGHT_ITEM], 0, 0, 0, 0, 1, 0]);
        await waitFor(() => expect(vi.mocked(GetSpectralSteedAttire).mock.calls.length).toBeGreaterThan(1));
    });

    it('acknowledges the DLC warning after the first Add operation only', async () => {
        await openAttireSection();
        const riskLabel = /why is this risky/i;
        expect(row(6701).getByLabelText(riskLabel)).toBeTruthy();
        fireEvent.click(addButton(6701));
        await waitFor(() => expect(AddItemsToCharacter).toHaveBeenCalledTimes(1));
        await waitFor(() => expect(row(6701).queryByLabelText(riskLabel)).toBeNull());
        expect(row(6703).queryByLabelText(riskLabel)).toBeNull();
    });

    it('Unlock All adds every missing attire in one Inventory batch', async () => {
        vi.mocked(AddItemsToCharacter).mockResolvedValue({ added: 2, requested: 2, capHit: '' } as never);
        await openAttireSection();
        fireEvent.click(attireHeader().getByRole('button', { name: 'Unlock All' }));
        await waitFor(() => expect(AddItemsToCharacter).toHaveBeenCalledWith(
            0,
            [TREE_SENTINEL_ITEM, FUNEREAL_NIGHT_ITEM],
            0, 0, 0, 0, 1, 0,
        ));
    });

    it('Lock All uses the atomic backend reset and does not add items', async () => {
        await openAttireSection();
        fireEvent.click(attireHeader().getByRole('button', { name: 'Lock All' }));
        await waitFor(() => expect(LockAllSpectralSteedAttires).toHaveBeenCalledWith(0));
        expect(AddItemsToCharacter).not.toHaveBeenCalled();
    });

    it('Set calls the backend with the matching flag and refetches', async () => {
        await openAttireSection();
        fireEvent.click(setButton(6702));
        await waitFor(() => expect(SetSpectralSteedAttire).toHaveBeenCalledWith(0, 6702));
        expect(AddItemsToCharacter).not.toHaveBeenCalled();
        await waitFor(() => expect(vi.mocked(GetSpectralSteedAttire).mock.calls.length).toBeGreaterThan(1));
    });

    it('routes only attire Add through the tarnished_edition_dlc risk key', async () => {
        await openAttireSection();
        const title = RISK_INFO.tarnished_edition_dlc.title;
        expect(row(6701).getAllByLabelText(new RegExp(`why is this risky.*${title}`, 'i')).length).toBe(1);
        expect(row(6702).getAllByLabelText(new RegExp(`why is this risky.*${title}`, 'i')).length).toBe(1);
    });

    it('renders no attire rows and toasts when the getter rejects', async () => {
        vi.mocked(GetSpectralSteedAttire).mockRejectedValue(new Error('event flags offset not computed for slot 0'));
        render(<WorldTab charIdx={0} platform="PC" />);
        fireEvent.click(await screen.findByRole('button', { name: /unlocks/i }));
        fireEvent.click(await screen.findByText('Spectral Steed Attire'));
        await waitFor(() => expect(vi.mocked(toast.error).mock.calls.length).toBe(1));
        expect(String(vi.mocked(toast.error).mock.calls[0][0])).toContain('Failed to load Spectral Steed Attire');
        for (const id of [6700, 6701, 6702, 6703]) {
            expect(screen.queryByTestId(`attire-row-${id}`)).toBeNull();
        }
        expect(screen.queryByRole('button', { name: 'Add' })).toBeNull();
        expect(screen.queryByRole('button', { name: 'Set' })).toBeNull();
    });

    it('keeps every action available on a PS4/PS5 save', async () => {
        await openAttireSection('PS4');
        // Ownership and active state are the only gates: 6702 is owned, 6700 active.
        expect((addButton(6701) as HTMLButtonElement).disabled).toBe(false);
        expect((addButton(6702) as HTMLButtonElement).disabled).toBe(true);
        expect((setButton(6702) as HTMLButtonElement).disabled).toBe(false);
        expect((setButton(6700) as HTMLButtonElement).disabled).toBe(true);
        expect((attireHeader().getByRole('button', { name: 'Unlock All' }) as HTMLButtonElement).disabled).toBe(false);
        expect((attireHeader().getByRole('button', { name: 'Lock All' }) as HTMLButtonElement).disabled).toBe(false);

        fireEvent.click(setButton(6702));
        await waitFor(() => expect(SetSpectralSteedAttire).toHaveBeenCalledWith(0, 6702));
        fireEvent.click(addButton(6701));
        await waitFor(() => expect(AddItemsToCharacter).toHaveBeenCalledWith(0, [TREE_SENTINEL_ITEM], 0, 0, 0, 0, 1, 0));
        fireEvent.click(attireHeader().getByRole('button', { name: 'Lock All' }));
        await waitFor(() => expect(LockAllSpectralSteedAttires).toHaveBeenCalledWith(0));
    });

    it('adds no platform gate when the platform is unknown', async () => {
        await openAttireSection('');
        expect((addButton(6701) as HTMLButtonElement).disabled).toBe(false);
        expect((addButton(6702) as HTMLButtonElement).disabled).toBe(true);
        expect((setButton(6702) as HTMLButtonElement).disabled).toBe(false);
        expect((setButton(6700) as HTMLButtonElement).disabled).toBe(true);
        expect(setButton(6702).getAttribute('title')).toBeNull();
    });
});
