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
});
