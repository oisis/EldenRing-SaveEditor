// @vitest-environment jsdom
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { editor, application as main } from '../../wailsjs/go/models';

// vi.hoisted runs before module imports so the shared mock fns are
// available inside vi.mock's hoisted factory. Each test reassigns the
// per-call behavior via mockResolvedValue / mockImplementation.
const mocks = vi.hoisted(() => ({
    GetInfuseTypes: vi.fn(),
    GetItemList: vi.fn(),
    GetCharacter: vi.fn(),
    GetAoWAvailability: vi.fn(),
}));

vi.mock('../../wailsjs/go/main/App', () => ({
    GetInfuseTypes: mocks.GetInfuseTypes,
    GetItemList: mocks.GetItemList,
    GetCharacter: mocks.GetCharacter,
    GetAoWAvailability: mocks.GetAoWAvailability,
}));

beforeEach(() => {
    mocks.GetInfuseTypes.mockResolvedValue([]);
    mocks.GetItemList.mockResolvedValue([]);
    mocks.GetCharacter.mockResolvedValue({ inventory: [], storage: [] });
    mocks.GetAoWAvailability.mockResolvedValue([]);
});

afterEach(() => {
    vi.clearAllMocks();
});

import { WeaponEditModal } from './WeaponEditModal';

function makeOrderItem(overrides: Partial<main.InventoryOrderItem> = {}): main.InventoryOrderItem {
    return main.InventoryOrderItem.createFrom({
        handle: 0x80800001,
        itemId: 0x000F4240,
        name: 'Dagger',
        category: 'melee_armaments',
        acquisitionIndex: 1000,
        currentUpgrade: 0,
        maxUpgrade: 25,
        ...overrides,
    });
}

function makeWorkspaceItem(overrides: Partial<editor.EditableItem> = {}): editor.EditableItem {
    return editor.EditableItem.createFrom({
        uid: 'hnd:0x80800001',
        source: 'original',
        container: 'inventory',
        position: 0,
        originalHandle: 0x80800001,
        itemID: 0x000F4240,
        baseItemID: 0x000F4240,
        name: 'Dagger',
        category: 'melee_armaments',
        quantity: 1,
        acquisitionIndex: 1000,
        currentUpgrade: 0,
        maxUpgrade: 25,
        hasGaItem: true,
        isWeapon: true,
        isArmor: false,
        isTalisman: false,
        canChangeAffinity: true,
        ...overrides,
    });
}

async function renderModal(props: Parameters<typeof WeaponEditModal>[0]) {
    let result!: ReturnType<typeof render>;
    await act(async () => {
        result = render(<WeaponEditModal {...props} />);
    });
    return result;
}

describe('WeaponEditModal (workspace mode)', () => {
    it('renders the pending assign banner when workspaceItem has pendingAoWItemID', async () => {
        const workspace = { sessionID: 'ses-1', updateWeapon: vi.fn() };
        const item = makeOrderItem();
        const workspaceItem = makeWorkspaceItem({
            pendingAoWItemID: 0x80002710,
            pendingAoWName: "Lion's Claw",
        });

        await renderModal({
            charIndex: 0,
            item,
            source: 'inventory',
            onClose: () => {},
            workspace,
            workspaceItem,
        });

        expect(screen.getByText(/Pending save:\s*Lion's Claw/)).toBeInTheDocument();
    });

    it('renders the pending clear banner when workspaceItem has pendingAoWClear', async () => {
        const workspace = { sessionID: 'ses-2', updateWeapon: vi.fn() };
        const item = makeOrderItem();
        const workspaceItem = makeWorkspaceItem({ pendingAoWClear: true });

        await renderModal({
            charIndex: 0,
            item,
            source: 'inventory',
            onClose: () => {},
            workspace,
            workspaceItem,
        });

        expect(screen.getByText(/built-in skill will be restored/)).toBeInTheDocument();
    });

    it('does not render the pending banner when no pending edits exist', async () => {
        const workspace = { sessionID: 'ses-3', updateWeapon: vi.fn() };
        await renderModal({
            charIndex: 0,
            item: makeOrderItem(),
            source: 'inventory',
            onClose: () => {},
            workspace,
            workspaceItem: makeWorkspaceItem(),
        });

        expect(screen.queryByText(/Pending save/)).not.toBeInTheDocument();
    });

    it('shows the workspace item name in the header', async () => {
        const workspace = { sessionID: 'ses-header', updateWeapon: vi.fn() };
        await renderModal({
            charIndex: 0,
            item: makeOrderItem({ name: 'Claymore' }),
            source: 'inventory',
            onClose: () => {},
            workspace,
            workspaceItem: makeWorkspaceItem({ name: 'Claymore' }),
        });

        expect(screen.getByText('Claymore')).toBeInTheDocument();
    });

    it('renders through a top-level portal with compact level and infusion editors', async () => {
        const host = document.createElement('div');
        document.body.appendChild(host);

        let result!: ReturnType<typeof render>;
        await act(async () => {
            result = render(
                <WeaponEditModal
                    charIndex={0}
                    item={makeOrderItem()}
                    source="inventory"
                    onClose={() => {}}
                    workspace={{ sessionID: 'ses-layout', updateWeapon: vi.fn() }}
                    workspaceItem={makeWorkspaceItem()}
                />,
                { container: host },
            );
        });

        const dialog = screen.getByRole('dialog', { name: 'Edit Dagger' });
        expect(dialog.parentElement).toBe(document.body.lastElementChild);
        expect(dialog.parentElement).toHaveClass('z-[100]', 'bg-black/60');

        const editors = screen.getByTestId('weapon-value-editors');
        expect(editors).toHaveClass('grid-cols-2');
        expect(screen.getByLabelText('Upgrade level')).toHaveClass('h-8');

        result.unmount();
        host.remove();
    });

    it('shows default AoW name when no custom AoW is attached', async () => {
        mocks.GetItemList.mockImplementation(async (cat: string) => {
            if (cat !== 'ashes_of_war') return [];
            return [{
                id: 0x80002710,
                name: 'Quickstep',
                iconPath: '/items/ashes_of_war/quickstep.png',
                aowCompatBitmask: 1,
            }];
        });
        const workspace = { sessionID: 'ses-default-aow', updateWeapon: vi.fn() };
        await renderModal({
            charIndex: 0,
            item: makeOrderItem(),
            source: 'inventory',
            onClose: () => {},
            workspace,
            workspaceItem: makeWorkspaceItem({
                canMountAoW: true,
                currentAoWItemID: 0,
                defaultAoWID: 800,
                defaultAoWName: 'Quickstep',
            }),
        });

        expect(screen.getByText('Quickstep')).toBeInTheDocument();
        expect(screen.queryByText('Built-in skill')).not.toBeInTheDocument();
        const icon = await screen.findByTestId('current-aow-icon');
        expect(icon.querySelector('img')).toHaveAttribute('src', '/items/ashes_of_war/quickstep.png');
    });

    // Regression: workspace-mode WeaponEditModal must render compatible
    // Ashes of War without depending on GetCharacter. Previously the modal
    // fell back to GetCharacter for wepType/canMountAoW; for newly-added
    // weapons (no save-side handle) or after a Save that re-allocated
    // handles, the lookup missed and the modal filtered every AoW out as
    // unknown-compat. The workspace item now carries wepType/canMountAoW
    // directly, so the modal can resolve compatibility off the workspace
    // snapshot alone.
    it('renders compatible AoWs from workspace metadata without GetCharacter', async () => {
        // Intentionally make GetCharacter return nothing matching — the
        // modal must NOT depend on it in workspace mode.
        mocks.GetCharacter.mockResolvedValue({ inventory: [], storage: [] });
        mocks.GetAoWAvailability.mockResolvedValue([
            {
                itemId: 0x80003070,
                totalCopies: 1,
                availableCopies: 1,
                usedCopies: 0,
                usedByWeaponHandles: [],
                isMissing: false,
                hasSharedHandleConflict: false,
            },
        ]);
        mocks.GetItemList.mockImplementation(async (cat: string) => {
            if (cat !== 'ashes_of_war') return [];
            return [
                {
                    id: 0x80003070,
                    name: 'Sword Dance',
                    iconPath: '',
                    aowCompatBitmask: 1, // bit 0 set ⇒ compatible with wepType=1 (Dagger)
                },
            ];
        });

        const workspace = { sessionID: 'ses-aow', updateWeapon: vi.fn() };
        await renderModal({
            charIndex: 0,
            item: makeOrderItem(),
            source: 'inventory',
            onClose: () => {},
            workspace,
            workspaceItem: makeWorkspaceItem({
                wepType: 1,        // Dagger
                canMountAoW: true, // gemMountType==2
            }),
        });

        // The list should contain the compatible AoW and not show the
        // "No compatible Ashes of War available." fallback.
        await waitFor(() => {
            expect(screen.getByText('Sword Dance')).toBeInTheDocument();
        });
        expect(screen.queryByText(/No compatible Ashes of War available/i))
            .not.toBeInTheDocument();
    });

    it('renders Ashes of War as an icon grid instead of list rows', async () => {
        mocks.GetAoWAvailability.mockResolvedValue([]);
        mocks.GetItemList.mockImplementation(async (cat: string) => {
            if (cat !== 'ashes_of_war') return [];
            return [
                {
                    id: 0x80003070,
                    name: 'Sword Dance',
                    iconPath: '/items/ashes_of_war/sword_dance.png',
                    aowCompatBitmask: 1,
                },
            ];
        });

        await renderModal({
            charIndex: 0,
            item: makeOrderItem(),
            source: 'inventory',
            onClose: () => {},
            workspace: { sessionID: 'ses-aow-icons', updateWeapon: vi.fn() },
            workspaceItem: makeWorkspaceItem({
                wepType: 1,
                canMountAoW: true,
            }),
        });

        const grid = await screen.findByTestId('aow-icon-grid');
        expect(grid).toHaveClass('grid-cols-[repeat(auto-fill,minmax(104px,1fr))]');

        const card = screen.getByRole('button', { name: 'Select Sword Dance' });
        expect(card).toHaveAttribute('data-aow-icon-card');
        expect(card.querySelector('img')).toHaveClass('h-full', 'w-full', 'object-contain');
    });

 it('allows applying compatible AoW when no free copy exists in the save', async () => {
 mocks.GetAoWAvailability.mockResolvedValue([]);
 mocks.GetItemList.mockImplementation(async (cat: string) => {
 if (cat !== 'ashes_of_war') return [];
 return [
 {
 id: 0x80003070,
 name: 'Sword Dance',
 iconPath: '',
 aowCompatBitmask: 1, // bit 0 set ⇒ compatible with wepType=1 (Dagger)
 },
 ];
 });

 const workspace = {
 sessionID: 'ses-aow-missing',
 updateWeapon: vi.fn().mockResolvedValue(makeWorkspaceItem({
 pendingAoWItemID: 0x80003070,
 pendingAoWName: 'Sword Dance',
 })),
 };
 await renderModal({
 charIndex: 0,
 item: makeOrderItem(),
 source: 'inventory',
 onClose: () => {},
 workspace,
 workspaceItem: makeWorkspaceItem({
 wepType: 1,
 canMountAoW: true,
 }),
 });

 await waitFor(() => {
 expect(screen.getByText('Sword Dance')).toBeInTheDocument();
 });
 expect(screen.queryByText(/No compatible Ashes of War available/i))
 .not.toBeInTheDocument();

 fireEvent.click(screen.getByText('Sword Dance'));
 fireEvent.click(screen.getByRole('button', { name: /Apply Ash of War/i }));

 await waitFor(() => {
 expect(workspace.updateWeapon).toHaveBeenCalledTimes(1);
 });
 expect(workspace.updateWeapon).toHaveBeenCalledWith(
 'hnd:0x80800001',
 expect.objectContaining({ aowItemID: 0x80003070 }),
 );
 });

    it('shows direct Light Greatsword AoWs for Milady and toggles incompatible entries', async () => {
        mocks.GetAoWAvailability.mockResolvedValue([]);
        mocks.GetItemList.mockImplementation(async (cat: string) => {
            if (cat !== 'ashes_of_war') return [];
            return [
                {
                    id: 0x80002774,
                    name: 'Impaling Thrust',
                    iconPath: '',
                    aowCompatBitmask: 2 ** 41,
                },
                {
                    id: 0x80003070,
                    name: 'Sword Dance',
                    iconPath: '',
                    aowCompatBitmask: 2 ** 41,
                },
                {
                    id: 0x80064960,
                    name: 'Wing Stance',
                    iconPath: '',
                    aowCompatBitmask: 2 ** 41,
                },
                {
                    id: 0x80002CEC,
                    name: 'Square Off',
                    iconPath: '',
                    aowCompatBitmask: 2 ** 1,
                },
            ];
        });

        await renderModal({
            charIndex: 0,
            item: makeOrderItem({ name: 'Milady', itemId: 0x0405F7E0 }),
            source: 'inventory',
            onClose: () => {},
            workspace: { sessionID: 'ses-aow-dlc', updateWeapon: vi.fn() },
            workspaceItem: makeWorkspaceItem({
                name: 'Milady',
                itemID: 0x0405F7E0,
                baseItemID: 0x0405F7E0,
                wepType: 93,
                canMountAoW: true,
            }),
        });

        await waitFor(() => {
            expect(screen.getByText('Impaling Thrust')).toBeInTheDocument();
            expect(screen.getByText('Sword Dance')).toBeInTheDocument();
            expect(screen.getByText('Wing Stance')).toBeInTheDocument();
        });
        expect(screen.queryByText('Square Off')).not.toBeInTheDocument();

        const showUnavailable = screen.getByRole('checkbox', { name: 'Show unavailable' });
        expect(showUnavailable.closest('label')).toHaveClass('h-9');
        expect(screen.getByPlaceholderText('Search Ashes of War...')).toHaveClass('h-9');
        fireEvent.click(showUnavailable);

        expect(await screen.findByText('Square Off')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Select Square Off' }))
            .toHaveAttribute('data-aow-compat', 'incompatible');
    });

    it('shows only light-bow AoWs and hides affinity editing for Shortbow', async () => {
        mocks.GetInfuseTypes.mockResolvedValue([
            { name: 'Standard', offset: 0 },
            { name: 'Heavy', offset: 100 },
        ]);
        mocks.GetItemList.mockImplementation(async (cat: string) => {
            if (cat !== 'ashes_of_war') return [];
            return [
                { id: 0x80009CA4, name: 'Barrage', iconPath: '', aowCompatBitmask: 2 ** 24 },
                { id: 0x80003070, name: 'Sword Dance', iconPath: '', aowCompatBitmask: 1 },
            ];
        });

        await renderModal({
            charIndex: 0,
            item: makeOrderItem({
                itemId: 0x02625A00,
                name: 'Shortbow',
                category: 'ranged_and_catalysts',
            }),
            source: 'inventory',
            onClose: () => {},
            workspace: { sessionID: 'ses-shortbow', updateWeapon: vi.fn() },
            workspaceItem: makeWorkspaceItem({
                itemID: 0x02625A00,
                baseItemID: 0x02625A00,
                name: 'Shortbow',
                category: 'ranged_and_catalysts',
                wepType: 50,
                canMountAoW: true,
                canChangeAffinity: false,
            }),
        });

        await waitFor(() => expect(screen.getByText('Barrage')).toBeInTheDocument());
        expect(screen.queryByText('Sword Dance')).not.toBeInTheDocument();
        expect(screen.queryByRole('button', { name: /Apply Infusion/i })).not.toBeInTheDocument();
    });

    it('keeps level editing but hides AoW and affinity controls for Steel-Wire Torch', async () => {
        mocks.GetInfuseTypes.mockResolvedValue([
            { name: 'Standard', offset: 0 },
            { name: 'Heavy', offset: 100 },
        ]);

        await renderModal({
            charIndex: 0,
            item: makeOrderItem({
                itemId: 0x016E8420,
                name: 'Steel-Wire Torch',
                category: 'shields',
            }),
            source: 'inventory',
            onClose: () => {},
            workspace: { sessionID: 'ses-torch', updateWeapon: vi.fn() },
            workspaceItem: makeWorkspaceItem({
                itemID: 0x016E8420,
                baseItemID: 0x016E8420,
                name: 'Steel-Wire Torch',
                category: 'shields',
                wepType: 87,
                canMountAoW: false,
                canChangeAffinity: false,
            }),
        });

        expect(screen.getByRole('button', { name: /Apply Level/i })).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: /Apply Infusion/i })).not.toBeInTheDocument();
        expect(screen.queryByText(/No compatible Ashes of War available/i)).not.toBeInTheDocument();
    });
 });
