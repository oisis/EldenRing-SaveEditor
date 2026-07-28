import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ItemCapacityBadge } from './ItemCapacityBadge';

describe('ItemCapacityBadge', () => {
    it('shows unavailable only for a quantity stack with a zero cap', () => {
        render(<ItemCapacityBadge owned={0} max={0} label="Storage" />);
        expect(screen.getByLabelText('Storage unavailable')).toHaveTextContent('×');
    });

    it('uses presence semantics for separate physical instances', () => {
        const { rerender } = render(
            <ItemCapacityBadge owned={2} max={1} label="Inventory" mode="instance" />,
        );
        expect(screen.getByLabelText('Inventory present')).toHaveTextContent('✓');
        expect(screen.getByLabelText('Inventory present')).toHaveAttribute('title', 'Inventory: 2 copies');

        rerender(<ItemCapacityBadge owned={0} max={0} label="Storage" mode="instance" />);
        expect(screen.getByLabelText('Storage not present')).toHaveTextContent('—');
    });

    describe('instance-count mode', () => {
        it('shows the real copy count over a fixed 1 denominator', () => {
            render(<ItemCapacityBadge owned={3} max={1} label="Inventory" mode="instance-count" />);
            expect(screen.getByLabelText('Inventory 3 of 1')).toHaveTextContent('3 / 1');
        });

        it('keeps the container prefix in compact form', () => {
            render(<ItemCapacityBadge owned={3} max={1} label="Inventory" prefix="I" compact mode="instance-count" />);
            expect(screen.getByLabelText('Inventory 3 of 1')).toHaveTextContent('I:3/1');
        });

        it('renders 0 / 1 for an unowned instance item instead of a dash', () => {
            render(<ItemCapacityBadge owned={0} max={1} label="Storage" mode="instance-count" />);
            const badge = screen.getByLabelText('Storage 0 of 1');
            expect(badge).toHaveTextContent('0 / 1');
            expect(badge).not.toHaveTextContent('—');
        });
    });
});
