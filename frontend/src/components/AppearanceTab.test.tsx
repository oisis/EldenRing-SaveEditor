import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const localStorageEntries = new Map<string, string>();
Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
        getItem: (key: string) => localStorageEntries.get(key) ?? null,
        setItem: (key: string, value: string) => { localStorageEntries.set(key, String(value)); },
        removeItem: (key: string) => { localStorageEntries.delete(key); },
        clear: () => { localStorageEntries.clear(); },
    },
});

vi.mock('../../wailsjs/go/main/App', () => ({
    ListAppearancePresets: vi.fn(),
    ApplyMirrorFavoriteToCharacter: vi.fn(),
    WriteSelectedToFavorites: vi.fn(),
    GetFavoritesStatus: vi.fn(),
    RemoveFavoritePreset: vi.fn(),
}));

import {
    GetFavoritesStatus,
    ListAppearancePresets,
} from '../../wailsjs/go/main/App';
import { AppearanceTab } from './AppearanceTab';

const presets = [
    { name: 'Preset One', bodyType: 'Type A', image: 'one.png' },
    { name: 'Preset Two', bodyType: 'Type B', image: 'two.png' },
    { name: 'Preset Three', bodyType: 'Type A', image: 'three.png' },
];

beforeEach(() => {
    localStorage.clear();
    vi.mocked(ListAppearancePresets).mockResolvedValue(presets as never);
    vi.mocked(GetFavoritesStatus).mockResolvedValue([] as never);
});

describe('AppearanceTab preset carousel', () => {
    it('mounts cards around a visible 3D ring and synchronizes the slider', async () => {
        render(<AppearanceTab charIndex={0} onMutate={vi.fn()} />);

        expect(await screen.findByTestId('appearance-carousel')).toBeInTheDocument();
        expect(screen.getByTestId('carousel-ring')).toBeInTheDocument();
        expect(screen.getByTestId('preset-card-0')).toHaveAttribute('data-active', 'true');
        expect(screen.getByTestId('preset-card-0')).toHaveClass('bottom-4');
        expect(screen.getByTestId('preset-card-0')).not.toHaveClass('-translate-x-1/2');
        expect(screen.getByTestId('preset-card-0')).toHaveStyle({transform: 'translateX(-50%) rotateY(0deg) translateZ(390px) scale(1.08)'});
        expect(screen.getByTestId('preset-card-0')).toHaveStyle({transformOrigin: 'center bottom'});
        expect(screen.getByTestId('preset-card-0')).not.toHaveClass('grayscale');
        expect(screen.getByTestId('preset-card-1')).toHaveStyle({transform: 'translateX(-50%) rotateY(34deg) translateZ(390px) scale(0.82)'});
        expect(screen.getByTestId('preset-card-1')).toHaveClass('grayscale', 'saturate-0');

        fireEvent.change(screen.getByRole('slider', { name: 'Appearance preset' }), { target: { value: '2' } });
        expect(screen.getByTestId('preset-card-2')).toHaveAttribute('data-active', 'true');
        expect(screen.getByTestId('preset-card-2')).not.toHaveClass('grayscale');
        expect(screen.getByTestId('preset-card-0')).toHaveClass('grayscale');
    });

    it('ignores the mouse wheel and moves with horizontal drag', async () => {
        render(<AppearanceTab charIndex={0} onMutate={vi.fn()} />);
        const carousel = await screen.findByTestId('appearance-carousel');

        fireEvent.wheel(carousel, { deltaY: 100 });
        expect(screen.getByTestId('preset-card-0')).toHaveAttribute('data-active', 'true');

        fireEvent.pointerDown(carousel, { pointerId: 1, clientX: 200 });
        fireEvent.pointerMove(carousel, { pointerId: 1, clientX: 120 });
        fireEvent.pointerUp(carousel, { pointerId: 1, clientX: 120 });
        await waitFor(() => expect(screen.getByTestId('preset-card-1')).toHaveAttribute('data-active', 'true'));
    });

    it('persists favorite stars, keeps their clicks out of drag, and filters the ring', async () => {
        const {unmount} = render(<AppearanceTab charIndex={0} onMutate={vi.fn()} readOnly />);
        const carousel = await screen.findByTestId('appearance-carousel');
        const star = screen.getByRole('button', { name: 'Add Preset Two to favorites' });

        fireEvent.pointerDown(star, {pointerId: 1, clientX: 200});
        fireEvent.pointerMove(carousel, {pointerId: 1, clientX: 100});
        fireEvent.pointerUp(star, {pointerId: 1, clientX: 100});
        fireEvent.click(star);

        expect(screen.getByTestId('preset-card-0')).toHaveAttribute('data-active', 'true');
        expect(JSON.parse(localStorage.getItem('favorites:appearance-presets')!)).toEqual(['Preset Two']);

        unmount();
        render(<AppearanceTab charIndex={0} onMutate={vi.fn()} readOnly />);
        await screen.findByTestId('appearance-carousel');
        fireEvent.click(screen.getByRole('button', { name: 'Show favorite appearance presets only' }));

        expect(screen.getByText('1/3')).toBeInTheDocument();
        expect(screen.getByAltText('Preset Two')).toBeInTheDocument();
        expect(screen.queryByAltText('Preset One')).not.toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Remove Preset Two from favorites' })).toBeInTheDocument();
    });

    it('renders the image preview above the entire app through a body portal', async () => {
        render(<AppearanceTab charIndex={0} onMutate={vi.fn()} readOnly />);
        await screen.findByTestId('appearance-carousel');

        fireEvent.click(screen.getByRole('button', {name: 'Preview Preset One'}));

        const dialog = screen.getByRole('dialog', {name: 'Appearance preview'});
        expect(dialog.parentElement).toBe(document.body);
        expect(dialog).toHaveClass('fixed', 'inset-0', 'z-[100]', 'bg-black/85');
    });
});
