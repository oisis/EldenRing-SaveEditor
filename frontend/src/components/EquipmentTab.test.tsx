import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { EquipmentTab } from './EquipmentTab';
import { GetEquipmentSnapshot } from '../../wailsjs/go/main/App';

vi.mock('../../wailsjs/go/main/App', () => ({
    GetEquipmentSnapshot: vi.fn(),
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
        currentEquipLoad: 1.8,
        equipLoadKnown: true,
        equipLoadClass: 'Medium',
        maxEquipLoad: 64.1,
        activeTalismanSlots: 4,
        ...over,
    };
}

beforeEach(() => {
    vi.mocked(GetEquipmentSnapshot).mockReset();
    vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot() as never);
});

describe('EquipmentTab', () => {
    it('renders the approved equipment layout', () => {
        render(<EquipmentTab />);

        expect(screen.getByText('Equipment slots')).toBeInTheDocument();
        expect(screen.getByText('Quick pouch')).toBeInTheDocument();
        expect(screen.getByText('Wondrous Physick flask')).toBeInTheDocument();
        expect(screen.getByText('Equip Load', { exact: false })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Save changes' })).toBeInTheDocument();
    });

    it('opens an empty modal from every equipment field and closes it with both actions', () => {
        render(<EquipmentTab />);

        fireEvent.click(screen.getByRole('button', { name: 'Weapon slot 2' }));
        expect(screen.getByRole('dialog', { name: 'Equipment slot' })).toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
        expect(screen.queryByRole('dialog', { name: 'Equipment slot' })).not.toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Physick tear 1' }));
        fireEvent.click(screen.getByRole('button', { name: 'Ok' }));
        expect(screen.queryByRole('dialog', { name: 'Equipment slot' })).not.toBeInTheDocument();
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

        expect(screen.getByText('Equipment slots').parentElement?.parentElement).toHaveClass('grid-cols-[499px_255px]');
        expect(screen.getByText('Quick pouch').parentElement).toHaveClass('pl-[26px]');
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

    it('leaves Physick unchanged when a snapshot loads', async () => {
        vi.mocked(GetEquipmentSnapshot).mockResolvedValue(makeSnapshot({
            rightHandArmaments: fill(3, { occupied: true, resolved: true, name: 'Claymore', iconPath: 'items/weapons/claymore.png' }),
        }) as never);

        render(<EquipmentTab charIdx={0} />);
        await screen.findByAltText('Claymore');

        expect(screen.getAllByRole('tooltip', { name: 'Crystal Tears' })).toHaveLength(2);
    });
});
