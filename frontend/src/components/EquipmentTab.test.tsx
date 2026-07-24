import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { EquipmentTab } from './EquipmentTab';

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

        expect(screen.getByText('Wondrous Physick flask')).toHaveClass('mb-3');
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
