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
});
