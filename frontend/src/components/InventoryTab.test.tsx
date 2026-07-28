import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// Equipment tab: subcategory filtering, Owned scoping and numeric sorting.
// Virtualized table rows don't render under jsdom, so order assertions read
// from the grid view (which shares the same sorted list).

vi.mock('../../wailsjs/go/main/App', () => ({
    GetCharacter: vi.fn(),
    SaveCharacter: vi.fn(),
    RemoveItemsFromCharacter: vi.fn(),
    GetItemDetail: vi.fn(),
    StartInventoryEditSession: vi.fn(),
    DiscardInventoryEditSession: vi.fn(),
    SaveInventoryWorkspaceChanges: vi.fn(),
    UpdateInventoryWorkspaceWeapon: vi.fn(),
    ValidateInventoryWorkspace: vi.fn(),
    AddInventoryWorkspaceItem: vi.fn(),
    MoveInventoryWorkspaceItem: vi.fn(),
    RemoveInventoryWorkspaceItem: vi.fn(),
    ReorderInventoryWorkspaceItems: vi.fn(),
    TransferInventoryWorkspaceItem: vi.fn(),
}));

vi.mock('./WeaponEditModal', () => ({
    WeaponEditModal: ({ workspaceItem }: { workspaceItem: { name: string } }) =>
        <div data-testid="weapon-edit-modal">Weapon editor: {workspaceItem.name}</div>,
}));

vi.mock('../lib/toast', () => {
    const fn = vi.fn() as unknown as Record<string, unknown> & ((...a: unknown[]) => void);
    fn.success = vi.fn();
    fn.error = vi.fn();
    return { default: fn };
});

vi.mock('../state/favorites', () => ({
    useFavorites: () => ({ isFav: () => false, toggle: vi.fn() }),
}));

// jsdom has no layout, so the real virtualizer renders zero rows — the editable
// quantity inputs never mount. Render every row so the Save flow is reachable.
vi.mock('@tanstack/react-virtual', () => ({
    useVirtualizer: (opts: { count: number }) => ({
        getVirtualItems: () => Array.from({ length: opts.count }, (_, index) => ({ index, start: 0, size: 40, key: index })),
        getTotalSize: () => opts.count * 40,
        measureElement: () => {},
    }),
}));

import * as App from '../../wailsjs/go/main/App';
import { InventoryTab } from './InventoryTab';

const mocks = App as unknown as Record<string, ReturnType<typeof vi.fn>>;

// Stackable owned rows in the melee_armaments category, split across two
// subgroups (Straight Sword x2, Axe x1).
function owned(id: number, name: string, subGroup: string, quantity = 1) {
    return {
        id, baseId: id, name, category: 'Item',
        subCategory: 'melee_armaments', subGroup,
        maxInventory: 99, maxStorage: 600, maxUpgrade: 0, currentUpgrade: 0,
        quantity, handle: id, iconPath: '', flags: [] as string[], readOnly: false,
        recordMode: 'quantity_stack',
        gameMaxInventory: 99, gameMaxStorage: 600,
        gameMaxInventoryKnown: true, gameMaxStorageKnown: true,
    };
}

function emptyWorkspace(overrides: Record<string, unknown> = {}) {
    return {
        sessionID: 'inventory-session',
        characterIndex: 0,
        inventoryItems: [],
        storageItems: [],
        dirty: false,
        validation: null,
        ...overrides,
    };
}

function tabElement(overrides: Partial<Parameters<typeof InventoryTab>[0]> = {}) {
    return (
        <InventoryTab
            charIndex={0}
            inventoryVersion={0}
            columnVisibility={{ id: true, category: true }}
            showFlaggedItems={true}
            category="melee_armaments"
            setCategory={vi.fn()}
            {...overrides}
        />
    );
}

beforeEach(() => {
    mocks.StartInventoryEditSession.mockResolvedValue(emptyWorkspace());
    mocks.DiscardInventoryEditSession.mockResolvedValue(undefined);
    mocks.SaveInventoryWorkspaceChanges.mockResolvedValue(emptyWorkspace());
    mocks.GetCharacter.mockResolvedValue({
        inventory: [
            owned(0x11, 'Longsword', 'Straight Sword', 5),
            owned(0x12, 'Broadsword', 'Straight Sword', 1),
            owned(0x13, 'Battle Axe', 'Axe', 10),
        ],
        storage: [],
    });
});

afterEach(() => vi.clearAllMocks());

describe('InventoryTab (Equipment)', () => {
    it('resets the subcategory filter to All when the category changes', async () => {
        const { rerender } = render(tabElement({ category: 'melee_armaments' }));
        const sub = await screen.findByLabelText('Subcategory') as HTMLSelectElement;
        await waitFor(() => expect(sub.options.length).toBeGreaterThan(1));
        fireEvent.change(sub, { target: { value: 'Axe' } });
        expect(sub.value).toBe('Axe');

        rerender(tabElement({ category: 'head' }));
        await waitFor(() =>
            expect((screen.getByLabelText('Subcategory') as HTMLSelectElement).value).toBe('all'));
    });

    it('Owned counts the whole category, then narrows to the subcategory', async () => {
        render(tabElement({ category: 'melee_armaments' }));
        const badge = (await screen.findByText('Owned:')).parentElement!;
        await waitFor(() => expect(badge).toHaveTextContent('Owned:3'));

        fireEvent.change(await screen.findByLabelText('Subcategory'), { target: { value: 'Axe' } });
        await waitFor(() => expect(badge).toHaveTextContent('Owned:1'));
    });

    it('text search does not change the Owned count', async () => {
        render(tabElement({ category: 'melee_armaments' }));
        const badge = (await screen.findByText('Owned:')).parentElement!;
        await waitFor(() => expect(badge).toHaveTextContent('Owned:3'));

        fireEvent.change(screen.getByPlaceholderText('Search owned items...'), { target: { value: 'zzz-no-match' } });
        expect(badge).toHaveTextContent('Owned:3');
    });

    it('fires onMutate after a successful SaveCharacter so App can bump inventoryVersion', async () => {
        mocks.SaveCharacter.mockResolvedValue(undefined);
        const onMutate = vi.fn();
        render(tabElement({ category: 'melee_armaments', onMutate }));

        // Edit a quantity so the Save Changes button appears, then save.
        const qtyInput = (await screen.findAllByRole('spinbutton'))[0];
        fireEvent.change(qtyInput, { target: { value: '7' } });
        fireEvent.click(await screen.findByText('Save Changes'));

        await waitFor(() => expect(mocks.SaveCharacter).toHaveBeenCalled());
        await waitFor(() => expect(onMutate).toHaveBeenCalled());
    });

    it('sorts by owned Inventory quantity', async () => {
        render(tabElement({ category: 'melee_armaments' }));
        fireEvent.click(await screen.findByText(/^Inventory/)); // table header → ascending
        fireEvent.click(screen.getByTitle('Grid view'));

        await waitFor(() => expect(screen.getByText('Broadsword')).toBeInTheDocument());
        const order = screen.getAllByText(/Longsword|Broadsword|Battle Axe/).map(e => e.textContent);
        expect(order).toEqual(['Broadsword', 'Longsword', 'Battle Axe']); // 1, 5, 10
    });

    it('uses a red X for an impossible container and a green check for 1 / 1', async () => {
        mocks.GetCharacter.mockResolvedValue({
            inventory: [{
                ...owned(0x21, 'Sacred Key', 'Key', 1),
                maxInventory: 1,
                maxStorage: 0,
            }],
            storage: [],
        });
        render(tabElement());

        expect(await screen.findByLabelText('Inventory 1 of 1')).toHaveTextContent('✓');
        expect(screen.getByLabelText('Storage unavailable')).toHaveTextContent('×');
    });

    it('uses Expanded Limits caps for quantity stacks such as Spirit Ashes', async () => {
        mocks.GetCharacter.mockResolvedValue({
            inventory: [{
                ...owned(0x400318f8, 'Fanged Imp Ashes', '', 1),
                handle: 0xb00318f8,
                category: 'Item',
                subCategory: 'ashes',
                maxInventory: 1,
                maxStorage: 1,
                gameMaxInventory: 1,
                gameMaxStorage: 600,
                gameMaxInventoryKnown: true,
                gameMaxStorageKnown: true,
                recordMode: 'quantity_stack',
            }],
            storage: [],
        });
        render(tabElement({ category: 'ashes' }));
        act(() => {
            window.dispatchEvent(new CustomEvent('safetyProfileChanged', { detail: 'expanded_limits' }));
        });
        fireEvent.click(screen.getByTitle('Grid view'));

        expect(await screen.findByLabelText('Inventory 1 of 1')).toHaveTextContent('✓');
        expect(screen.getByLabelText('Storage 0 of 600')).toHaveTextContent('S:0');
    });

    it('uses the observed Expanded Limits storage cap for Furlcalling Finger Remedy', async () => {
        mocks.GetCharacter.mockResolvedValue({
            inventory: [{
                ...owned(0x40000096, 'Furlcalling Finger Remedy', '', 3),
                handle: 0xb0000096,
                category: 'Item',
                subCategory: 'tools',
                maxInventory: 999,
                maxStorage: 0,
                gameMaxInventory: 999,
                gameMaxStorage: 999,
                gameMaxInventoryKnown: true,
                gameMaxStorageKnown: true,
                recordMode: 'quantity_stack',
            }],
            storage: [],
        });
        render(tabElement({ category: 'tools' }));
        act(() => {
            window.dispatchEvent(new CustomEvent('safetyProfileChanged', { detail: 'expanded_limits' }));
        });
        fireEvent.click(screen.getByTitle('Grid view'));

        expect(await screen.findByLabelText('Inventory 3 of 999')).toHaveTextContent('I:3');
        expect(screen.getByLabelText('Storage 0 of 999')).toHaveTextContent('S:0');
    });

    it('persists Expanded Limits quantities with the technical-cap writer mode', async () => {
        const spell = {
            ...owned(0x40000fa0, 'Glintstone Pebble', '', 1),
            handle: 0xb0000fa0,
            category: 'Item',
            subCategory: 'sorceries',
            maxInventory: 1,
            maxStorage: 0,
            gameMaxInventory: 99,
            gameMaxStorage: 600,
            gameMaxInventoryKnown: true,
            gameMaxStorageKnown: true,
            recordMode: 'quantity_stack',
        };
        mocks.GetCharacter.mockResolvedValue({ inventory: [spell], storage: [] });
        mocks.SaveCharacter.mockResolvedValue(undefined);
        render(tabElement({ category: 'sorceries' }));
        act(() => {
            window.dispatchEvent(new CustomEvent('safetyProfileChanged', { detail: 'expanded_limits' }));
        });

        const quantity = await screen.findByRole('spinbutton');
        fireEvent.change(quantity, { target: { value: '50' } });
        fireEvent.click(await screen.findByText('Save Changes'));

        await waitFor(() => expect(mocks.SaveCharacter).toHaveBeenCalled());
        const submitted = mocks.SaveCharacter.mock.calls[0][1];
        expect(submitted.useTechnicalItemCaps).toBe(true);
        expect(submitted.inventory[0].quantity).toBe(50);
    });

    it('keeps attached Ashes of War separate from Inventory and Storage records', async () => {
        const aow = (handle: number, equipped = false) => ({
            ...owned(0x80002710, "Lion's Claw", '', 1),
            handle,
            category: 'Ash of War',
            subCategory: 'ashes_of_war',
            maxInventory: 1,
            maxStorage: 1,
            recordMode: 'separate_instances',
            isEquippedAoW: equipped,
            equippedByWeaponHandle: equipped ? 0x80800002 : 0,
            equippedByWeaponName: equipped ? 'Claymore' : '',
            readOnly: equipped,
        });
        mocks.GetCharacter.mockResolvedValue({
            inventory: [aow(0xc0800002)],
            storage: [],
            attachedItems: [aow(0xc0800001, true)],
        });
        render(tabElement({ category: 'ashes_of_war' }));

        expect(await screen.findAllByText("Lion's Claw")).toHaveLength(2);
        const equipped = screen.getByTitle('Equipped on Claymore');
        expect(equipped).toHaveClass('bg-green-500/[0.06]');
        expect(equipped).not.toHaveClass('outline-green-500/60');
        expect(within(equipped).getByLabelText('Inventory not present')).toBeInTheDocument();
        expect(within(equipped).getByLabelText('Storage not present')).toBeInTheDocument();
        expect(screen.getByTitle('Attached to Claymore. Edit it from the weapon.')).toBeInTheDocument();
        // Regression: the Item Database's instance-count format (N / 1) must not
        // leak into the Inventory tab — separate instances keep the ✓ / —
        // presence semantics here.
        expect(screen.queryByText('1 / 1')).not.toBeInTheDocument();
        expect(screen.queryByLabelText(/Inventory \d+ of 1/)).not.toBeInTheDocument();
    });

    it('shows the AoW column and opens the shared weapon editor from the red edit button', async () => {
        const weapon = {
            ...owned(0x003085e0, 'Claymore', 'Greatswords', 1),
            handle: 0x80800011,
            category: 'Weapon',
            maxInventory: 1,
            maxStorage: 1,
            maxUpgrade: 25,
            recordMode: 'separate_instances',
        };
        const customWeapon = {
            ...weapon,
            id: 0x0030acf0,
            baseId: 0x0030acf0,
            handle: 0x80800012,
            name: 'Zweihander',
        };
        mocks.GetCharacter.mockResolvedValue({ inventory: [weapon, customWeapon], storage: [] });
        mocks.StartInventoryEditSession.mockResolvedValue(emptyWorkspace({
            inventoryItems: [{
                uid: 'weapon-1',
                originalHandle: weapon.handle,
                itemID: weapon.id,
                baseItemID: weapon.id,
                name: weapon.name,
                category: 'melee_armaments',
                currentUpgrade: 0,
                maxUpgrade: 25,
                infusionName: '',
                quantity: 1,
                acquisitionIndex: 1,
                iconPath: '',
                isWeapon: true,
                defaultAoWName: 'Lion\'s Claw',
                hasCurrentAoW: false,
            }, {
                uid: 'weapon-2',
                originalHandle: customWeapon.handle,
                itemID: customWeapon.id,
                baseItemID: customWeapon.id,
                name: customWeapon.name,
                category: 'melee_armaments',
                currentUpgrade: 0,
                maxUpgrade: 25,
                infusionName: '',
                quantity: 1,
                acquisitionIndex: 2,
                iconPath: '',
                isWeapon: true,
                defaultAoWName: 'Stamp',
                hasCurrentAoW: true,
                currentAoWName: 'Bloody Slash',
            }],
        }));
        render(tabElement({ category: 'melee_armaments' }));

        expect(await screen.findByText('Ash of War')).toBeInTheDocument();
        expect(await screen.findByText("Lion's Claw")).toHaveClass('text-muted-foreground');
        expect(await screen.findByText('Bloody Slash')).toHaveClass('text-green-500');
        fireEvent.click(await screen.findByLabelText('Edit Claymore'));
        expect(await screen.findByTestId('weapon-edit-modal')).toHaveTextContent('Weapon editor: Claymore');
    });

    it('persists staged weapon edits through the shared Save Changes button', async () => {
        mocks.StartInventoryEditSession.mockResolvedValue(emptyWorkspace({ dirty: true }));
        mocks.SaveInventoryWorkspaceChanges.mockResolvedValue(emptyWorkspace({ dirty: false }));
        const onMutate = vi.fn();
        render(tabElement({ onMutate }));

        fireEvent.click(await screen.findByText('Save Changes'));
        await waitFor(() => expect(mocks.SaveInventoryWorkspaceChanges).toHaveBeenCalledWith('inventory-session'));
        expect(mocks.SaveCharacter).not.toHaveBeenCalled();
        await waitFor(() => expect(onMutate).toHaveBeenCalled());
    });
});
