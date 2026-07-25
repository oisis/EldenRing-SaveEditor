import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { EquipmentTab } from './EquipmentTab';
import {
    GetArmsSlotEligibleItems,
    GetArrowSlotEligibleItems,
    GetBoltSlotEligibleItems,
    GetCharacter,
    GetChestSlotEligibleItems,
    GetEquipmentSnapshot,
    GetHandArmamentEligibleItems,
    GetHeadSlotEligibleItems,
    GetInfuseTypes,
    GetItemList,
    GetLegsSlotEligibleItems,
    GetPhysickEligibleItems,
    GetPouchEligibleItems,
    GetQuickItemEligibleItems,
    GetTalismanSlotEligibleItems,
} from '../../wailsjs/go/main/App';

vi.mock('../../wailsjs/go/main/App', () => ({
    GetEquipmentSnapshot: vi.fn(),
    GetCharacter: vi.fn(),
    GetHandArmamentEligibleItems: vi.fn(),
    GetArrowSlotEligibleItems: vi.fn(),
    GetBoltSlotEligibleItems: vi.fn(),
    GetHeadSlotEligibleItems: vi.fn(),
    GetInfuseTypes: vi.fn(),
    GetChestSlotEligibleItems: vi.fn(),
    GetArmsSlotEligibleItems: vi.fn(),
    GetLegsSlotEligibleItems: vi.fn(),
    GetTalismanSlotEligibleItems: vi.fn(),
    GetQuickItemEligibleItems: vi.fn(),
    GetPouchEligibleItems: vi.fn(),
    GetPhysickEligibleItems: vi.fn(),
    GetItemList: vi.fn(),
}));

const emptyView = { occupied: false, rawId: 0, name: '', iconPath: '', resolved: false };
const view = (over: Partial<typeof emptyView>) => ({ ...emptyView, ...over });
const fill = (n: number, over?: Partial<typeof emptyView>) =>
    Array.from({ length: n }, (_, i) => (i === 0 && over ? view(over) : { ...emptyView }));

function makeSnapshot(over: Record<string, unknown> = {}) {
    return {
        rightHandArmaments: fill(3),
        leftHandArmaments: fill(3),
        arrows: fill(2),
        bolts: fill(2),
        armor: fill(4),
        talismans: fill(4),
        quickItems: fill(10),
        pouch: fill(6),
        physick: fill(2),
        currentEquipLoad: 1.8,
        equipLoadKnown: true,
        equipLoadClass: 'Medium',
        maxEquipLoad: 64.1,
        activeTalismanSlots: 4,
        activeSpellSlots: 10,
        ...over,
    };
}

beforeEach(() => {
    vi.mocked(GetEquipmentSnapshot).mockReset();
    vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot() as never);
    const eligibilityEndpoints = [
        GetHandArmamentEligibleItems,
        GetArrowSlotEligibleItems,
        GetBoltSlotEligibleItems,
        GetHeadSlotEligibleItems,
        GetInfuseTypes,
        GetChestSlotEligibleItems,
        GetArmsSlotEligibleItems,
        GetLegsSlotEligibleItems,
        GetTalismanSlotEligibleItems,
        GetQuickItemEligibleItems,
        GetPouchEligibleItems,
        GetPhysickEligibleItems,
        GetItemList,
    ];
    eligibilityEndpoints.forEach(endpoint => {
        vi.mocked(endpoint).mockReset();
        vi.mocked(endpoint).mockResolvedValue([] as never);
    });
    vi.mocked(GetCharacter).mockReset();
    vi.mocked(GetCharacter).mockResolvedValue({ inventory: [] } as never);
    vi.mocked(GetInfuseTypes).mockReset();
    vi.mocked(GetInfuseTypes).mockResolvedValue([] as never);
});

describe('EquipmentTab', () => {
    it('renders the approved equipment layout', () => {
        render(<EquipmentTab />);

        expect(screen.getByText('Equipment slots')).toBeInTheDocument();
        expect(screen.getByText('Quick pouch')).toBeInTheDocument();
        expect(screen.getByText('Wondrous Physick flask')).toBeInTheDocument();
        expect(screen.getByText('Spell slots')).toBeInTheDocument();
        expect(screen.getByText('Equip Load', { exact: false })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Save changes' })).toBeInTheDocument();
    });

    it('shows a visible remove icon in the lower-left corner of every equipment field', () => {
        render(<EquipmentTab />);

        expect(screen.getAllByTestId('slot-remove-icon')).toHaveLength(46);
        expect(screen.getAllByTestId('slot-remove-icon')[0]).toHaveClass('bottom-0.5', 'left-1', 'text-lg', 'text-red-600');
    });

    it('opens the item picker from every equipment field and closes it with both actions', () => {
        render(<EquipmentTab />);

        fireEvent.click(screen.getByRole('button', { name: 'Weapon slot 2' }));
        expect(screen.getByRole('dialog', { name: 'Select equipment item' })).toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
        expect(screen.queryByRole('dialog', { name: 'Select equipment item' })).not.toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Physick tear 1' }));
        fireEvent.click(screen.getByRole('button', { name: 'Ok' }));
        expect(screen.queryByRole('dialog', { name: 'Select equipment item' })).not.toBeInTheDocument();
    });

    it('filters owned items by slot eligibility and exposes the database picker controls', async () => {
        vi.mocked(GetHandArmamentEligibleItems).mockResolvedValue([
            { id: 0x1234, name: 'Database Claymore', category: 'melee_armaments', iconPath: 'items/weapons/claymore.png' },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [
                { id: 0x1234, baseId: 0x1234, name: 'Owned Claymore', category: 'melee_armaments', iconPath: 'items/weapons/claymore.png', quantity: 1 },
                { id: 0x5678, baseId: 0x5678, name: 'Ineligible Tool', category: 'tools', iconPath: 'items/tools/lantern.png', quantity: 1 },
            ],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Weapon slot 1' }));
        await screen.findByPlaceholderText('Search items...');
        expect(await screen.findByText('Owned Claymore')).toBeInTheDocument();
        expect(screen.queryByText('Ineligible Tool')).not.toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Item Database' }));
        expect(await screen.findByText('Database Claymore')).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Add & Equip' })).not.toBeInTheDocument();
        expect(screen.getByText('Items from Item Database will be added to Inventory before equipping.')).toBeInTheDocument();
        expect(screen.getByAltText('Database Claymore')).toHaveClass('h-20', 'w-20');
        expect(screen.getByTestId('equipment-picker-toolbar')).toHaveClass('justify-start');
        expect(screen.getByPlaceholderText('Search items...')).toHaveClass('h-[32px]', 'w-[448px]');
        expect(screen.getByPlaceholderText('Search items...')).toHaveClass('max-w-full');
        expect(screen.getByLabelText('Item source')).toHaveClass('ml-auto');
        expect(screen.getByLabelText('Sort items')).toHaveValue('alphabetical');
        expect(screen.getByLabelText('Sort items')).toHaveClass('h-[32px]');
        expect(screen.getByLabelText('View mode')).toHaveClass('h-[32px]');
        expect(screen.getByLabelText('Item source')).toHaveClass('h-[32px]');

        fireEvent.click(screen.getByRole('button', { name: 'List view' }));
        expect(screen.getByRole('button', { name: 'List view' })).toHaveAttribute('aria-pressed', 'true');
        expect(screen.getByRole('button', { name: 'Select Database Claymore' }).querySelector('img')).toHaveClass('h-12', 'w-12');
    });

    it('closes the item picker with Escape', () => {
        render(<EquipmentTab />);

        fireEvent.click(screen.getByRole('button', { name: 'Weapon slot 1' }));
        expect(screen.getByRole('dialog', { name: 'Select equipment item' })).toBeInTheDocument();

        fireEvent.keyDown(window, { key: 'Escape' });
        expect(screen.queryByRole('dialog', { name: 'Select equipment item' })).not.toBeInTheDocument();
    });

    it('shows an icon-view quantity only for stackable owned items', async () => {
        vi.mocked(GetHandArmamentEligibleItems).mockResolvedValue([
            { id: 1, name: 'Stackable item', category: 'tools', iconPath: 'items/tools/a.png', maxInventory: 99 },
            { id: 2, name: 'Single item', category: 'melee_armaments', iconPath: 'items/weapons/b.png', maxInventory: 1 },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [
                { id: 1, baseId: 1, name: 'Stackable item', category: 'tools', iconPath: 'items/tools/a.png', quantity: 1, maxInventory: 99 },
                { id: 2, baseId: 2, name: 'Single item', category: 'melee_armaments', iconPath: 'items/weapons/b.png', quantity: 1, maxInventory: 1 },
            ],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Weapon slot 1' }));
        await screen.findByText('Stackable item');

        expect(screen.getAllByText('×1')).toHaveLength(1);
        expect(screen.getByText('×1')).toHaveClass('absolute', 'right-2', 'top-2', 'text-xs');
    });

    it('renders duplicate inventory instances as icon cards after switching from list view', async () => {
        vi.mocked(GetHandArmamentEligibleItems).mockResolvedValue([
            { id: 0x1234, name: 'Duplicate armor', category: 'chest', iconPath: 'items/chest/duplicate.png', maxInventory: 1 },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [
                { handle: 11, id: 0x1234, baseId: 0x1234, name: 'Duplicate armor', category: 'chest', iconPath: 'items/chest/duplicate.png', quantity: 1, maxInventory: 1 },
                { handle: 12, id: 0x1234, baseId: 0x1234, name: 'Duplicate armor', category: 'chest', iconPath: 'items/chest/duplicate.png', quantity: 1, maxInventory: 1 },
            ],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Weapon slot 1' }));
        await screen.findAllByText('Duplicate armor');
        fireEvent.click(screen.getByRole('button', { name: 'List view' }));
        fireEvent.click(screen.getByRole('button', { name: 'Icon view' }));

        const cards = screen.getAllByRole('button', { name: 'Select Duplicate armor' });
        expect(cards).toHaveLength(2);
        cards.forEach(card => expect(card).toHaveClass('flex-col'));
    });

    it('hides risky eligible items in Safe Mode from both picker sources', async () => {
        vi.mocked(GetHandArmamentEligibleItems).mockResolvedValue([
            { id: 1, name: 'Safe item', category: 'melee_armaments', iconPath: 'items/weapons/safe.png' },
            { id: 2, name: 'Cut item', category: 'melee_armaments', iconPath: 'items/weapons/cut.png', flags: ['cut_content', 'ban_risk'] },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [
                { handle: 11, id: 1, baseId: 1, name: 'Safe item', category: 'melee_armaments', iconPath: 'items/weapons/safe.png', quantity: 1, maxInventory: 1 },
                { handle: 12, id: 2, baseId: 2, name: 'Cut item', category: 'melee_armaments', iconPath: 'items/weapons/cut.png', quantity: 1, maxInventory: 1 },
            ],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Weapon slot 1' }));
        expect(await screen.findByText('Safe item')).toBeInTheDocument();
        expect(screen.queryByText('Cut item')).not.toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Item Database' }));
        expect(await screen.findByText('Safe item')).toBeInTheDocument();
        expect(screen.queryByText('Cut item')).not.toBeInTheDocument();
    });

    it('shows weapon level in icons and level, infusion and AoW columns in list view', async () => {
        vi.mocked(GetHandArmamentEligibleItems).mockResolvedValue([
            { id: 1000, name: 'Test weapon', category: 'melee_armaments', iconPath: 'items/weapons/test.png', maxInventory: 1 },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [{ handle: 1, id: 1305, baseId: 1000, name: 'Test weapon', category: 'Weapon', iconPath: 'items/weapons/test.png', quantity: 1, maxInventory: 1, currentUpgrade: 5, aowId: 0x80002C88 }],
        } as never);
        vi.mocked(GetInfuseTypes).mockResolvedValue([{ name: 'Quality', offset: 300 }] as never);
        vi.mocked(GetItemList).mockImplementation((category: string) => Promise.resolve(category === 'ashes_of_war'
            ? [{ id: 0x80002C88, name: 'Unsheathe', category: 'ashes_of_war', iconPath: '' }]
            : []) as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Weapon slot 1' }));
        expect(await screen.findByText('+5')).toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'List view' }));
        expect(screen.getByText('Level')).toBeInTheDocument();
        expect(screen.getByText('Infuse')).toBeInTheDocument();
        expect(screen.getByText('Ashes of War')).toHaveClass('text-right');
        expect(screen.getByText('Ashes of War').parentElement).toHaveClass('grid-cols-[minmax(0,1fr)_max-content_max-content_max-content]');
        expect(screen.getByText('Quality')).toBeInTheDocument();
        expect(screen.getByText('Unsheathe')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Select Test weapon' })).toHaveTextContent('Test weapon');
        expect(screen.queryByText('Weapon · 1')).not.toBeInTheDocument();
    });

    it('shows armor names larger without the category and quantity line in list view', async () => {
        vi.mocked(GetChestSlotEligibleItems).mockResolvedValue([
            { id: 1, name: 'Test armor', category: 'chest', iconPath: 'items/chest/test.png', maxInventory: 1 },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [{ handle: 1, id: 1, baseId: 1, name: 'Test armor', category: 'Armor', iconPath: 'items/chest/test.png', quantity: 1, maxInventory: 1 }],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Knight Armor' }));
        await screen.findByText('Test armor');
        fireEvent.click(screen.getByRole('button', { name: 'List view' }));

        expect(screen.getByText('Test armor')).toHaveClass('text-sm');
        expect(screen.queryByText('Armor · 1')).not.toBeInTheDocument();
    });

    it('shows talisman names larger without the category and quantity line in list view', async () => {
        vi.mocked(GetTalismanSlotEligibleItems).mockResolvedValue([
            { id: 1, name: 'Test talisman', category: 'talismans', iconPath: 'items/talismans/test.png', maxInventory: 1 },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [{ handle: 1, id: 1, baseId: 1, name: 'Test talisman', category: 'Talisman', iconPath: 'items/talismans/test.png', quantity: 1, maxInventory: 1 }],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Axe Talisman' }));
        await screen.findByText('Test talisman');
        fireEvent.click(screen.getByRole('button', { name: 'List view' }));

        expect(screen.getByText('Test talisman')).toHaveClass('text-sm');
        expect(screen.queryByText('Talisman · 1')).not.toBeInTheDocument();
    });

    it('shows ammunition names larger without the category and quantity line in list view', async () => {
        vi.mocked(GetArrowSlotEligibleItems).mockResolvedValue([
            { id: 1, name: 'Test arrow', category: 'arrows_and_bolts', iconPath: 'items/ammo/test.png', maxInventory: 99 },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [{ handle: 1, id: 1, baseId: 1, name: 'Test arrow', category: 'Ammunition', iconPath: 'items/ammo/test.png', quantity: 10, maxInventory: 99 }],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Arrow slot 1' }));
        await screen.findByText('Test arrow');
        fireEvent.click(screen.getByRole('button', { name: 'List view' }));

        expect(screen.getByText('Test arrow')).toHaveClass('text-sm');
        expect(screen.queryByText('Ammunition · 10')).not.toBeInTheDocument();
    });

    it('shows quick-pouch and quick-item names larger without category and quantity lines in list view', async () => {
        const eligibleItem = { id: 1, name: 'Test tool', category: 'tools', iconPath: 'items/tools/test.png', maxInventory: 99 };
        vi.mocked(GetPouchEligibleItems).mockResolvedValue([eligibleItem] as never);
        vi.mocked(GetQuickItemEligibleItems).mockResolvedValue([eligibleItem] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [{ handle: 1, id: 1, baseId: 1, name: 'Test tool', category: 'Tool', iconPath: 'items/tools/test.png', quantity: 10, maxInventory: 99 }],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Quick pouch up' }));
        await screen.findByText('Test tool');
        fireEvent.click(screen.getByRole('button', { name: 'List view' }));
        expect(screen.getByText('Test tool')).toHaveClass('text-sm');
        expect(screen.queryByText('Tool · 10')).not.toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

        fireEvent.click(screen.getByRole('button', { name: 'Quick item 1' }));
        await screen.findByText('Test tool');
        fireEvent.click(screen.getByRole('button', { name: 'List view' }));
        expect(screen.getByText('Test tool')).toHaveClass('text-sm');
        expect(screen.queryByText('Tool · 10')).not.toBeInTheDocument();
    });

    it('shows spell names larger without category and quantity lines in list view', async () => {
        vi.mocked(GetItemList).mockImplementation((category: string) => Promise.resolve(category === 'sorceries'
            ? [{ id: 1, name: 'Test sorcery', category: 'sorceries', iconPath: 'items/sorceries/test.png', maxInventory: 1 }]
            : []) as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [{ handle: 1, id: 1, baseId: 1, name: 'Test sorcery', category: 'Sorcery', iconPath: 'items/sorceries/test.png', quantity: 1, maxInventory: 1 }],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Spell slot 1' }));
        await screen.findByText('Test sorcery');
        fireEvent.click(screen.getByRole('button', { name: 'List view' }));

        expect(screen.getByText('Test sorcery')).toHaveClass('text-sm');
        expect(screen.queryByText('Sorcery · 1')).not.toBeInTheDocument();
    });

    it('shows Physick tear names larger without the key-item category and quantity line in list view', async () => {
        vi.mocked(GetPhysickEligibleItems).mockResolvedValue([
            { id: 1, name: 'Test crystal tear', category: 'key_items', iconPath: 'items/key_items/test.png', maxInventory: 1 },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [{ handle: 1, id: 1, baseId: 1, name: 'Test crystal tear', category: 'Key Item', iconPath: 'items/key_items/test.png', quantity: 1, maxInventory: 1 }],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Physick tear 1' }));
        await screen.findByText('Test crystal tear');
        fireEvent.click(screen.getByRole('button', { name: 'List view' }));

        expect(screen.getByText('Test crystal tear')).toHaveClass('text-sm');
        expect(screen.queryByText('Key Item · 1')).not.toBeInTheDocument();
    });

    it('shows an owned technical Crimson Crystal Tear variant as the canonical Physick tear', async () => {
        vi.mocked(GetPhysickEligibleItems).mockResolvedValue([
            { id: 0x40002AFB, name: 'Crimson Crystal Tear', category: 'key_items', iconPath: 'items/key_items/crimson_crystal_tear.png', maxInventory: 1 },
            { id: 0x40002B02, name: 'Greenburst Crystal Tear', category: 'key_items', iconPath: 'items/key_items/greenburst_crystal_tear.png', maxInventory: 1 },
        ] as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [
                { handle: 0xB0002AFA, id: 0x40002AFA, baseId: 0x40002AFA, name: 'Crimson Crystal Tear (Variant)', category: 'Key Item', iconPath: 'items/key_items/crimson_crystal_tear.png', quantity: 1, maxInventory: 1 },
                { handle: 0xB0002B02, id: 0x40002B02, baseId: 0x40002B02, name: 'Greenburst Crystal Tear', category: 'Key Item', iconPath: 'items/key_items/greenburst_crystal_tear.png', quantity: 1, maxInventory: 1 },
            ],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        fireEvent.click(screen.getByRole('button', { name: 'Physick tear 1' }));

        expect(await screen.findByText('Crimson Crystal Tear')).toBeInTheDocument();
        expect(screen.getByText('Greenburst Crystal Tear')).toBeInTheDocument();
        expect(screen.queryByText('Crimson Crystal Tear (Variant)')).not.toBeInTheDocument();
    });

    it('shows the eligible item types in a tooltip for each slot family', () => {
        render(<EquipmentTab />);

        expect(screen.getAllByRole('tooltip', { name: 'Weapons, shields, staves, seals and torches' })).toHaveLength(6);
        expect(screen.getAllByRole('tooltip', { name: 'Arrows and greatarrows' })).toHaveLength(2);
        expect(screen.getAllByRole('tooltip', { name: 'Bolts and greatbolts' })).toHaveLength(2);
        expect(screen.getByRole('tooltip', { name: 'Helms' })).toBeInTheDocument();
        expect(screen.getByRole('tooltip', { name: 'Chest armor' })).toBeInTheDocument();
        expect(screen.getByRole('tooltip', { name: 'Gauntlets' })).toBeInTheDocument();
        expect(screen.getByRole('tooltip', { name: 'Leg armor' })).toBeInTheDocument();
        expect(screen.getAllByRole('tooltip', { name: 'Talismans' })).toHaveLength(4);
        expect(screen.getAllByRole('tooltip', { name: 'Tools and Spirit Ashes' })).toHaveLength(16);
        expect(screen.getAllByRole('tooltip', { name: 'Crystal Tears' })).toHaveLength(2);
        expect(screen.getAllByRole('tooltip', { name: 'Sorceries and Incantations' })).toHaveLength(10);
    });

    it('keeps rows 3–6 on the fixed-width mockup grid', () => {
        render(<EquipmentTab />);

        expect(screen.getByRole('button', { name: 'Knight Helm' }).parentElement).toHaveClass('grid-cols-[repeat(4,82px)]');
        expect(screen.getByRole('button', { name: 'Axe Talisman' }).parentElement).toHaveClass('grid-cols-[repeat(4,82px)]');
        expect(screen.getByRole('button', { name: 'Quick item 1' }).parentElement).toHaveClass('grid-cols-[repeat(5,82px)]');
        expect(screen.getByRole('button', { name: 'Quick item 6' }).parentElement).toHaveClass('grid-cols-[repeat(5,82px)]');
    });

    it('keeps equal spacing on both sides of the section divider', () => {
        render(<EquipmentTab />);

        expect(screen.getByText('Equipment slots').parentElement?.parentElement).toHaveClass('mx-auto', 'grid-cols-[499px_255px_199px]');
        expect(screen.getByText('Quick pouch').parentElement).toHaveClass('pl-[26px]');
        expect(screen.getByText('Spell slots').parentElement).toHaveClass('border-l', 'pl-[26px]');
    });

    it('matches quick-pouch fields to the equipment-slot size', () => {
        render(<EquipmentTab />);

        expect(screen.getByRole('button', { name: 'Quick pouch up' })).toHaveClass('h-[82px]', 'w-[82px]');
        expect(screen.getByRole('button', { name: 'Quick pouch up' }).parentElement).toHaveClass('grid-cols-[82px_82px]');
        expect(screen.getByRole('button', { name: 'Physick tear 1' })).toHaveClass('h-[82px]', 'w-[82px]');
        expect(screen.getByRole('button', { name: 'Physick tear 1' }).parentElement).toHaveClass('grid-cols-[82px_82px]');
    });

    it('uses the standard heading spacing above Physick fields', () => {
        render(<EquipmentTab />);

        expect(screen.getByText('Wondrous Physick flask')).toHaveClass('mb-3', 'whitespace-pre-line');
    });

    it('aligns right-side headings to their two-slot grids', () => {
        render(<EquipmentTab />);

        expect(screen.getByText('Quick pouch')).toHaveClass('w-[173px]');
        expect(screen.getByText('Wondrous Physick flask')).toHaveClass('w-[173px]');
        expect(screen.getByText('Spell slots')).toHaveClass('w-[173px]');
    });

    it('lays out spell slots top-to-bottom in two five-slot columns with mixed spell placeholders', () => {
        render(<EquipmentTab />);

        const grid = screen.getByTestId('spell-primary-grid');
        expect(grid).toHaveClass('grid-flow-col', 'grid-cols-[repeat(2,82px)]', 'grid-rows-[repeat(5,82px)]');
        const placeholderSources = [
            '/items/sorceries/comet_azur.png',
            '/items/incantations/dragonfire.png',
            '/items/sorceries/carian_slicer.png',
            '/items/incantations/scarlet_aeonia.png',
            '/items/sorceries/rock_sling.png',
            '/items/incantations/lightning_spear.png',
            '/items/sorceries/rennalas_full_moon.png',
            '/items/incantations/black_flame.png',
            '/items/sorceries/oracle_bubbles.png',
            '/items/incantations/heal.png',
            '/items/sorceries/founding_rain_of_stars.png',
            '/items/incantations/frenzied_burst.png',
        ];
        for (let index = 1; index <= 10; index++) {
            const slot = screen.getByRole('button', { name: `Spell slot ${index}` });
            expect(slot).toBeInTheDocument();
            expect(slot.querySelector('img')).toHaveAttribute('src', placeholderSources[index - 1]);
        }
        expect(screen.queryByRole('button', { name: 'Spell slot 11' })).not.toBeInTheDocument();
        expect(screen.getByTestId('spell-slot-area')).toHaveClass('justify-center');
    });

    it('adds the bottom spell row immediately when Moon of Nokstella increases the snapshot to twelve slots', async () => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({ activeSpellSlots: 12 }) as never);
        render(<EquipmentTab charIdx={0} />);

        await screen.findByRole('button', { name: 'Spell slot 11' });
        expect(screen.getByRole('button', { name: 'Spell slot 12' })).toBeInTheDocument();
        expect(screen.getByTestId('spell-slot-area')).toHaveClass('justify-start');
        expect(screen.getByRole('button', { name: 'Spell slot 11' }).parentElement).toHaveClass('grid-cols-[repeat(2,82px)]');
    });

    it('refreshes the visible spell slots after an equipment revision', async () => {
        vi.mocked(GetEquipmentSnapshot)
            .mockResolvedValueOnce(makeSnapshot({ activeSpellSlots: 10 }) as never)
            .mockResolvedValueOnce(makeSnapshot({ activeSpellSlots: 12 }) as never);

        const { rerender } = render(<EquipmentTab charIdx={0} equipmentRevision={1} />);
        await screen.findByRole('button', { name: 'Spell slot 10' });
        expect(screen.queryByRole('button', { name: 'Spell slot 11' })).not.toBeInTheDocument();

        rerender(<EquipmentTab charIdx={0} equipmentRevision={2} />);
        expect(await screen.findByRole('button', { name: 'Spell slot 11' })).toBeInTheDocument();
    });

    it('drives visuals from theme tokens instead of hard-coded light colors', () => {
        const { container } = render(<EquipmentTab />);

        // Card surface and primary action follow the semantic theme palette.
        const section = container.querySelector('section');
        expect(section).toHaveClass('bg-card', 'text-card-foreground', 'border-border');
        expect(screen.getByRole('button', { name: 'Save changes' })).toHaveClass('bg-primary', 'text-primary-foreground');
        expect(screen.getByText('Wondrous Physick flask')).toHaveClass('text-muted-foreground');

        // Equipment-specific visuals resolve to the per-theme --eq-* tokens.
        expect(screen.getByRole('button', { name: 'Weapon slot 1' })).toHaveClass('border-[color:var(--eq-slot-border)]');
    });
});

describe('EquipmentTab equipped-item projection', () => {
    it('renders a populated equipment slot with its icon and item-name tooltip', async () => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({
            rightHandArmaments: fill(3, { occupied: true, resolved: true, name: 'Claymore +12', iconPath: 'items/weapons/claymore.png', rawId: 0x80abcdef }),
        }) as never);

        render(<EquipmentTab charIdx={0} />);

        const icon = await screen.findByAltText('Claymore +12');
        expect(icon).toHaveAttribute('src', '/items/weapons/claymore.png');
        expect(screen.getByRole('tooltip', { name: 'Claymore +12' })).toBeInTheDocument();
    });

    it('renders a populated quick pouch slot with its icon and item-name tooltip', async () => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({
            pouch: fill(6, { occupied: true, resolved: true, name: 'Spirit Jellyfish Ashes', iconPath: 'items/ashes/spirit_jellyfish.png', rawId: 0xb0000123 }),
        }) as never);

        render(<EquipmentTab charIdx={0} />);

        const icon = await screen.findByAltText('Spirit Jellyfish Ashes');
        expect(icon).toHaveAttribute('src', '/items/ashes/spirit_jellyfish.png');
        expect(screen.getByRole('tooltip', { name: 'Spirit Jellyfish Ashes' })).toBeInTheDocument();
    });

    it('shows inventory quantities in the upper-right corner of equipped arrow and bolt slots', async () => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({
            arrows: [view({ occupied: true, rawId: 101, name: 'Test arrow', iconPath: 'items/arrows_and_bolts/arrow.png', resolved: true }), view({ occupied: true, rawId: 102, name: 'Test greatarrow', iconPath: 'items/arrows_and_bolts/greatarrow.png', resolved: true })],
            bolts: [view({ occupied: true, rawId: 201, name: 'Test bolt', iconPath: 'items/arrows_and_bolts/bolt.png', resolved: true }), view({ occupied: true, rawId: 202, name: 'Test greatbolt', iconPath: 'items/arrows_and_bolts/greatbolt.png', resolved: true })],
        }) as never);
        vi.mocked(GetCharacter).mockResolvedValue({
            inventory: [
                { id: 101, baseId: 101, subCategory: 'arrows_and_bolts', quantity: 30 },
                { id: 102, baseId: 102, subCategory: 'arrows_and_bolts', quantity: 20 },
                { id: 201, baseId: 201, subCategory: 'arrows_and_bolts', quantity: 40 },
                { id: 202, baseId: 202, subCategory: 'arrows_and_bolts', quantity: 10 },
            ],
        } as never);
        render(<EquipmentTab charIdx={0} />);

        await waitFor(() => expect(screen.getByRole('button', { name: 'Arrow slot 1' })).toHaveTextContent('30'));
        expect(screen.getByRole('button', { name: 'Arrow slot 2' })).toHaveTextContent('20');
        expect(screen.getByRole('button', { name: 'Bolt slot 1' })).toHaveTextContent('40');
        expect(screen.getByRole('button', { name: 'Bolt slot 2' })).toHaveTextContent('10');
        expect(screen.getByText('30')).toHaveClass('left-1.5', 'top-1', 'text-xs');
    });

    it('shows the raw-ID label for a populated but unresolved slot', async () => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({
            talismans: fill(4, { occupied: true, resolved: false, name: 'Unknown item (0x2000FFFF)', rawId: 0x2000ffff }),
        }) as never);

        render(<EquipmentTab charIdx={0} />);

        expect(await screen.findByRole('tooltip', { name: 'Unknown item (0x2000FFFF)' })).toBeInTheDocument();
    });

    it.each([
        [1, ['Axe Talisman']],
        [2, ['Axe Talisman', 'Claw Talisman']],
        [3, ['Axe Talisman', 'Claw Talisman', 'Companion Jar']],
    ])('hides locked talisman slots from the right when only %i are unlocked', async (activeTalismanSlots, visibleLabels) => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({ activeTalismanSlots }) as never);

        render(<EquipmentTab charIdx={0} />);

        await screen.findByRole('button', { name: 'Axe Talisman' });
        for (const label of visibleLabels) {
            expect(screen.getByRole('button', { name: label })).toBeInTheDocument();
        }
        for (const label of ['Axe Talisman', 'Claw Talisman', 'Companion Jar', 'Gold Scarab']) {
            if (!visibleLabels.includes(label)) {
                expect(screen.queryByRole('button', { name: label })).not.toBeInTheDocument();
            }
        }
    });

    it('keeps the eligibility tooltip on an empty slot', async () => {
        render(<EquipmentTab charIdx={0} />);
        // All-empty snapshot: weapon slots keep their eligibility text.
        expect(await screen.findAllByRole('tooltip', { name: 'Weapons, shields, staves, seals and torches' })).toHaveLength(6);
    });

    it('renders current and maximum Equip Load returned from the save', async () => {
        render(<EquipmentTab charIdx={0} />);

        expect(await screen.findByText('Medium')).toHaveClass('text-orange-500');
        expect(screen.getByText('1.8 / 64.1', { exact: false })).toBeInTheDocument();
    });

    it('reloads load and movement class when the equipped-save revision changes', async () => {
        vi.mocked(GetEquipmentSnapshot)
            .mockResolvedValueOnce(makeSnapshot({ currentEquipLoad: 20, maxEquipLoad: 64.1, equipLoadClass: 'Medium' }) as never)
            .mockResolvedValueOnce(makeSnapshot({ currentEquipLoad: 35, maxEquipLoad: 76.3, equipLoadClass: 'Heavy' }) as never);

        const { rerender } = render(<EquipmentTab charIdx={0} saveLoadKey={1} />);
        expect(await screen.findByText('20.0 / 64.1', { exact: false })).toBeInTheDocument();

        rerender(<EquipmentTab charIdx={0} saveLoadKey={2} />);
        await waitFor(() => expect(GetEquipmentSnapshot).toHaveBeenCalledTimes(2));
        expect(await screen.findByText('35.0 / 76.3', { exact: false })).toBeInTheDocument();
        expect(screen.getByText('Heavy')).toHaveClass('text-red-600');
    });

    it('keeps the eligibility tooltip on empty Physick slots', async () => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({
            rightHandArmaments: fill(3, { occupied: true, resolved: true, name: 'Claymore', iconPath: 'items/weapons/claymore.png' }),
        }) as never);

        render(<EquipmentTab charIdx={0} />);
        await screen.findByAltText('Claymore');

        expect(screen.getAllByRole('tooltip', { name: 'Crystal Tears' })).toHaveLength(2);
    });

    it('renders both Physick tears from the snapshot with canonical names and icons', async () => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({
            physick: [
                view({ occupied: true, resolved: true, name: 'Crimson Crystal Tear', iconPath: 'items/key_items/crimson_crystal_tear.png', rawId: 0x40002afa }),
                view({ occupied: true, resolved: true, name: 'Greenspill Crystal Tear', iconPath: 'items/key_items/greenspill_crystal_tear.png', rawId: 0x40002af9 }),
            ],
        }) as never);

        render(<EquipmentTab charIdx={0} />);

        const crimson = await screen.findByAltText('Crimson Crystal Tear');
        expect(crimson).toHaveAttribute('src', '/items/key_items/crimson_crystal_tear.png');
        const greenspill = screen.getByAltText('Greenspill Crystal Tear');
        expect(greenspill).toHaveAttribute('src', '/items/key_items/greenspill_crystal_tear.png');
        // Canonical display: the technical variant suffix must not leak through.
        expect(screen.queryByText('Crimson Crystal Tear (Variant)')).not.toBeInTheDocument();
        expect(screen.getByRole('tooltip', { name: 'Crimson Crystal Tear' })).toBeInTheDocument();
    });

    it('shows an unresolved Physick tear as a visible unknown, not an empty placeholder', async () => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({
            physick: [
                view({ occupied: true, resolved: false, name: 'Unknown item (0xFFFFFFFF)', rawId: 0xffffffff }),
                { occupied: false, rawId: 0, name: '', iconPath: '', resolved: false },
            ],
        }) as never);

        render(<EquipmentTab charIdx={0} />);

        // The raw-ID unknown label is visible and carries a tooltip; no ghost placeholder.
        expect(await screen.findByRole('tooltip', { name: 'Unknown item (0xFFFFFFFF)' })).toBeInTheDocument();
        expect(screen.getByTestId('physick-unknown')).toBeInTheDocument();
        const tear1 = screen.getByRole('button', { name: 'Physick tear 1' });
        expect(tear1.querySelector('img')).toBeNull();
        // The still-empty second slot keeps its eligibility tooltip.
        expect(screen.getByRole('tooltip', { name: 'Crystal Tears' })).toBeInTheDocument();
    });
});
