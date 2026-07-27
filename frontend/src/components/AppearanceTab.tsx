import {useCallback, useEffect, useMemo, useRef, useState, type PointerEvent} from 'react';
import {createPortal} from 'react-dom';
import toast from '../lib/toast';
import {
    ApplyMirrorFavoriteToCharacter,
    ApplyPresetToCharacter,
    GetFavoritesStatus,
    ListAppearancePresets,
    RemoveFavoritePreset,
    WriteSelectedToFavorites,
} from '../../wailsjs/go/main/App';
import {application as main} from '../../wailsjs/go/models';

const APPEARANCE_FAVORITES_KEY = 'favorites:appearance-presets';
const RING_RADIUS = 390;
const RING_STEP_DEGREES = 34;

interface Props {
    charIndex: number;
    onMutate: () => void;
    readOnly?: boolean;
    embedded?: boolean;
}

function loadFavoriteNames(): Set<string> {
    try {
        const raw = localStorage.getItem(APPEARANCE_FAVORITES_KEY);
        const parsed = raw ? JSON.parse(raw) : [];
        return new Set(Array.isArray(parsed) ? parsed.filter((name): name is string => typeof name === 'string') : []);
    } catch {
        return new Set();
    }
}

export function AppearanceTab({charIndex, onMutate, readOnly = false, embedded = false}: Props) {
    const [presets, setPresets] = useState<main.PresetInfo[]>([]);
    const [favSlots, setFavSlots] = useState<main.FavoriteSlotInfo[]>([]);
    const [favoriteNames, setFavoriteNames] = useState(loadFavoriteNames);
    const [showOnlyFavorites, setShowOnlyFavorites] = useState(false);
    const [search, setSearch] = useState('');
    const [showMale, setShowMale] = useState(true);
    const [showFemale, setShowFemale] = useState(true);
    const [addingPreset, setAddingPreset] = useState<string | null>(null);
    const [applyingPreset, setApplyingPreset] = useState<string | null>(null);
    const [zoomed, setZoomed] = useState<string | null>(null);
    const [activePreset, setActivePreset] = useState(0);
    const dragLastX = useRef<number | null>(null);

    const refreshFavStatus = useCallback(() => {
        if (!readOnly) GetFavoritesStatus().then(setFavSlots).catch(() => {});
    }, [readOnly]);

    useEffect(() => {
        ListAppearancePresets().then(setPresets).catch(e => toast.error('' + e));
        refreshFavStatus();
    }, [refreshFavStatus]);

    useEffect(() => {
        try {
            localStorage.setItem(APPEARANCE_FAVORITES_KEY, JSON.stringify([...favoriteNames]));
        } catch {
            // Favorites remain available for this session when storage is unavailable.
        }
    }, [favoriteNames]);

    const visiblePresets = useMemo(() => presets.filter(preset => {
        if (preset.bodyType === 'Type A' && !showMale) return false;
        if (preset.bodyType === 'Type B' && !showFemale) return false;
        if (showOnlyFavorites && !favoriteNames.has(preset.name)) return false;
        return !search || preset.name.toLowerCase().includes(search.toLowerCase());
    }), [favoriteNames, presets, search, showFemale, showMale, showOnlyFavorites]);

    useEffect(() => {
        setActivePreset(current => visiblePresets.length === 0 ? 0 : Math.min(current, visiblePresets.length - 1));
    }, [visiblePresets.length]);

    const freeSlots = favSlots.filter(slot => slot.safe && !slot.active).length;
    const usedSafeSlots = favSlots.filter(slot => slot.safe && slot.active);

    const toggleFavorite = (name: string) => {
        setFavoriteNames(current => {
            const next = new Set(current);
            if (next.has(name)) next.delete(name);
            else next.add(name);
            try {
                localStorage.setItem(APPEARANCE_FAVORITES_KEY, JSON.stringify([...next]));
            } catch {
                // Keep the change in memory when persistent storage is unavailable.
            }
            return next;
        });
    };

    const moveCarousel = useCallback((direction: -1 | 1) => {
        if (visiblePresets.length < 2) return;
        setActivePreset(current => (current + direction + visiblePresets.length) % visiblePresets.length);
    }, [visiblePresets.length]);

    const carouselOffset = (index: number) => {
        if (visiblePresets.length < 2) return 0;
        let offset = index - activePreset;
        const half = visiblePresets.length / 2;
        if (offset > half) offset -= visiblePresets.length;
        if (offset < -half) offset += visiblePresets.length;
        return offset;
    };

    const handlePointerDown = (event: PointerEvent<HTMLDivElement>) => {
        if ((event.target as HTMLElement).closest('button, input, label')) return;
        dragLastX.current = event.clientX;
        event.currentTarget.setPointerCapture?.(event.pointerId);
    };

    const handlePointerMove = (event: PointerEvent<HTMLDivElement>) => {
        if (dragLastX.current == null) return;
        const delta = event.clientX - dragLastX.current;
        if (Math.abs(delta) < 55) return;
        moveCarousel(delta < 0 ? 1 : -1);
        dragLastX.current = event.clientX;
    };

    const handlePointerEnd = (event: PointerEvent<HTMLDivElement>) => {
        dragLastX.current = null;
        if (event.currentTarget.hasPointerCapture?.(event.pointerId)) {
            event.currentTarget.releasePointerCapture?.(event.pointerId);
        }
    };

    const handleAddPreset = async (name: string) => {
        if (freeSlots === 0 || addingPreset !== null) return;
        setAddingPreset(name);
        try {
            await WriteSelectedToFavorites(charIndex, [name]);
            toast.success(`Added "${name.split(',')[0].trim()}" to Mirror Favorites`);
            refreshFavStatus();
        } catch (e) {
            toast.error('' + e);
        } finally {
            setAddingPreset(null);
        }
    };

    const handleApplyPreset = async (name: string) => {
        if (applyingPreset !== null) return;
        setApplyingPreset(name);
        try {
            await ApplyPresetToCharacter(charIndex, name);
            toast.success(`Applied "${name.split(',')[0].trim()}" to character`);
            onMutate();
        } catch (e) {
            toast.error('Apply failed: ' + e);
        } finally {
            setApplyingPreset(null);
        }
    };

    const handleRemoveFav = async (slotIndex: number) => {
        try {
            await RemoveFavoritePreset(slotIndex);
            toast.success(`Cleared Favorites slot ${slotIndex + 1}`);
            refreshFavStatus();
        } catch (e) {
            toast.error('' + e);
        }
    };

    const handleApplyFromMirror = async (slotIndex: number) => {
        try {
            await ApplyMirrorFavoriteToCharacter(charIndex, slotIndex);
            toast.success(`Applied Mirror slot ${slotIndex + 1} to character`);
            onMutate();
        } catch (e) {
            toast.error('' + e);
        }
    };

    return (
        <div className={embedded ? 'space-y-4' : 'space-y-6 p-4'}>
            {!embedded && (
                <div className="flex items-center space-x-3">
                    <div className="h-5 w-1 rounded-full bg-primary" />
                    <h3 className="text-sm font-black uppercase tracking-[0.15em]">Appearance Presets</h3>
                    <span className="text-[9px] font-medium uppercase tracking-wider text-muted-foreground">
                        {presets.length} presets
                    </span>
                </div>
            )}

            <div className="card space-y-3 p-4">
                <p className="text-[10px] leading-relaxed text-muted-foreground">
                    <strong>Click the active image</strong> to preview. Use the star to keep presets in local favorites.
                    {!readOnly && <> <strong>Apply</strong> changes the current character; <strong>Add</strong> writes to in-game Mirror Favorites.</>}
                </p>
                <div className="flex flex-wrap items-center gap-3">
                    <input
                        type="text"
                        aria-label="Search appearance presets"
                        placeholder="Search…"
                        value={search}
                        onChange={event => { setSearch(event.target.value); setActivePreset(0); }}
                        className="min-w-[180px] flex-1 rounded-md border border-border bg-muted/20 px-3 py-1.5 text-xs outline-none transition-all focus:ring-1 focus:ring-primary/30"
                    />
                    <label className="flex cursor-pointer select-none items-center gap-1.5">
                        <input type="checkbox" checked={showMale} onChange={event => { setShowMale(event.target.checked); setActivePreset(0); }} className="accent-primary" />
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Male</span>
                    </label>
                    <label className="flex cursor-pointer select-none items-center gap-1.5">
                        <input type="checkbox" checked={showFemale} onChange={event => { setShowFemale(event.target.checked); setActivePreset(0); }} className="accent-primary" />
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Female</span>
                    </label>
                    <button
                        type="button"
                        aria-pressed={showOnlyFavorites}
                        aria-label="Show favorite appearance presets only"
                        title="Show favorite appearance presets only"
                        onClick={() => { setShowOnlyFavorites(value => !value); setActivePreset(0); }}
                        className={`flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-[10px] font-black uppercase tracking-wider transition-colors ${
                            showOnlyFavorites ? 'border-amber-500/50 bg-amber-500/15 text-amber-500' : 'border-border text-muted-foreground hover:text-amber-500'
                        }`}
                    >
                        <span aria-hidden="true">★</span>
                        Favorites only
                    </button>
                    <span className="text-[9px] font-bold tabular-nums text-muted-foreground">
                        {visiblePresets.length}/{presets.length}
                    </span>
                </div>
                {!readOnly && (
                    <div className="text-[9px] font-bold uppercase tracking-wider text-primary">
                        {freeSlots} free mirror slots
                    </div>
                )}
            </div>

            <div
                role="region"
                aria-label="Appearance preset carousel"
                data-testid="appearance-carousel"
                className="relative h-[440px] select-none overflow-hidden rounded-xl border border-border/50 bg-gradient-to-b from-muted/20 via-background to-muted/10 touch-pan-y"
                style={{perspective: '1200px'}}
                onPointerDown={handlePointerDown}
                onPointerMove={handlePointerMove}
                onPointerUp={handlePointerEnd}
                onPointerCancel={handlePointerEnd}
            >
                <div
                    aria-hidden="true"
                    data-testid="carousel-ring"
                    className="pointer-events-none absolute left-1/2 top-[330px] h-[780px] w-[780px] rounded-full border-2 border-primary/25 shadow-[0_0_35px_hsl(var(--primary)/0.12)]"
                    style={{transform: 'translate(-50%, -50%) rotateX(76deg)'}}
                />
                <div
                    className="absolute inset-0"
                    style={{transform: `translateZ(-${RING_RADIUS}px)`, transformStyle: 'preserve-3d'}}
                >
                    {visiblePresets.map((preset, index) => {
                        const offset = carouselOffset(index);
                        const distance = Math.abs(offset);
                        const angle = offset * RING_STEP_DEGREES;
                        const isActive = offset === 0;
                        const isFavorite = favoriteNames.has(preset.name);
                        const visible = distance <= 4;
                        const isAdding = addingPreset === preset.name;
                        const isApplying = applyingPreset === preset.name;
                        return (
                            <article
                                key={preset.name}
                                data-testid={`preset-card-${index}`}
                                data-active={isActive || undefined}
                                data-ring-angle={angle}
                                aria-hidden={!visible}
                                className={`group absolute bottom-4 left-1/2 w-[220px] rounded-xl border bg-card shadow-2xl transition-[transform,opacity,filter] duration-500 ease-out ${
                                    isActive ? 'border-primary/80 ring-2 ring-primary/25' : 'border-border/60 grayscale saturate-0'
                                }`}
                                style={{
                                    opacity: visible ? Math.max(0.18, 1 - distance * 0.17) : 0,
                                    pointerEvents: visible ? 'auto' : 'none',
                                    zIndex: 30 - distance,
                                    transform: `translateX(-50%) rotateY(${angle}deg) translateZ(${RING_RADIUS}px) scale(${isActive ? 1.08 : 0.82})`,
                                    transformStyle: 'preserve-3d',
                                    transformOrigin: 'center bottom',
                                    backfaceVisibility: 'hidden',
                                }}
                                onClick={() => { if (!isActive) setActivePreset(index); }}
                            >
                                <div className="overflow-hidden rounded-xl">
                                    <button
                                        type="button"
                                        aria-label={`${isFavorite ? 'Remove' : 'Add'} ${preset.name} ${isFavorite ? 'from' : 'to'} favorites`}
                                        title={isFavorite ? 'Remove from favorites' : 'Add to favorites'}
                                        tabIndex={visible ? 0 : -1}
                                        onClick={event => { event.stopPropagation(); toggleFavorite(preset.name); }}
                                        className={`absolute right-2 top-2 z-20 flex h-7 w-7 items-center justify-center rounded-full border bg-black/55 text-lg shadow-lg transition-all ${
                                            isFavorite ? 'border-amber-400 text-amber-400' : 'border-white/35 text-white/70 hover:border-amber-400 hover:text-amber-400'
                                        }`}
                                    >
                                        <span aria-hidden="true">{isFavorite ? '★' : '☆'}</span>
                                    </button>
                                    <button
                                        type="button"
                                        aria-label={isActive ? `Preview ${preset.name}` : `Show ${preset.name}`}
                                        tabIndex={visible ? 0 : -1}
                                        className="relative block aspect-[3/4] w-full overflow-hidden bg-muted/30 text-left"
                                        onClick={event => {
                                            event.stopPropagation();
                                            if (isActive) setZoomed(preset.image ? `presets/${preset.image}` : null);
                                            else setActivePreset(index);
                                        }}
                                    >
                                        {preset.image ? (
                                            <img
                                                src={`presets/${preset.image}`}
                                                alt={preset.name}
                                                draggable={false}
                                                className="h-full w-full object-cover object-top transition-transform duration-500 group-hover:scale-105"
                                            />
                                        ) : (
                                            <span className="flex h-full w-full items-center justify-center">
                                                <svg className="h-10 w-10 text-muted-foreground/30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
                                                </svg>
                                            </span>
                                        )}
                                        <span className="absolute inset-0 bg-gradient-to-t from-black/65 via-transparent to-transparent" />
                                    </button>
                                    <div className="bg-background p-2.5">
                                        <div className={`truncate text-[10px] font-black uppercase tracking-wider ${isFavorite ? 'text-amber-500' : 'text-foreground'}`}>{preset.name}</div>
                                        <div className="mt-1 flex items-center justify-between gap-1">
                                            <span className="text-[8px] font-medium uppercase tracking-widest text-muted-foreground">{preset.bodyType}</span>
                                            {!readOnly && (
                                                <div className="flex gap-1">
                                                    <button
                                                        type="button"
                                                        onClick={event => { event.stopPropagation(); handleApplyPreset(preset.name); }}
                                                        disabled={applyingPreset !== null}
                                                        title="Apply appearance to current character"
                                                        className="rounded border border-blue-700/50 px-2 py-0.5 text-[9px] font-black uppercase tracking-wider text-blue-700 transition-all hover:bg-blue-700/10 disabled:cursor-not-allowed disabled:opacity-40"
                                                    >
                                                        {isApplying ? '…' : 'Apply'}
                                                    </button>
                                                    <button
                                                        type="button"
                                                        onClick={event => { event.stopPropagation(); handleAddPreset(preset.name); }}
                                                        disabled={freeSlots === 0 || addingPreset !== null}
                                                        title={freeSlots === 0 ? 'No free Mirror slots' : 'Add to Mirror Favorites'}
                                                        className="rounded border border-primary/40 px-2 py-0.5 text-[9px] font-black uppercase tracking-wider text-primary transition-all hover:bg-primary/10 disabled:cursor-not-allowed disabled:opacity-40"
                                                    >
                                                        {isAdding ? '…' : 'Add'}
                                                    </button>
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            </article>
                        );
                    })}
                </div>

                {visiblePresets.length === 0 && (
                    <div className="flex h-full items-center justify-center text-[10px] font-black uppercase tracking-widest text-muted-foreground">
                        {showOnlyFavorites ? 'No favorite appearance presets' : 'No appearance presets'}
                    </div>
                )}
            </div>

            {visiblePresets.length > 1 && (
                <div className="flex items-center gap-3 px-4">
                    <span className="w-10 text-right text-[9px] font-black tabular-nums text-muted-foreground">{activePreset + 1}</span>
                    <input
                        type="range"
                        aria-label="Appearance preset"
                        min={0}
                        max={visiblePresets.length - 1}
                        value={activePreset}
                        onChange={event => setActivePreset(Number(event.target.value))}
                        className="h-1.5 flex-1 cursor-pointer accent-primary"
                    />
                    <span className="w-10 text-[9px] font-black tabular-nums text-muted-foreground">{visiblePresets.length}</span>
                </div>
            )}

            {!readOnly && usedSafeSlots.length > 0 && (
                <div className="space-y-2 border-t border-border pt-4">
                    <div className="flex items-center space-x-3">
                        <div className="h-5 w-1 rounded-full bg-amber-500" />
                        <h3 className="text-sm font-black uppercase tracking-[0.15em]">Mirror Favorites</h3>
                        <span className="text-[9px] font-medium uppercase tracking-wider text-muted-foreground">
                            {usedSafeSlots.length} used · {freeSlots} free
                        </span>
                    </div>
                    <div className="flex flex-wrap gap-2">
                        {usedSafeSlots.map(slot => (
                            <div key={slot.index} className="flex items-center gap-2 rounded-md bg-muted/30 px-3 py-1.5">
                                {slot.image && <img src={`presets/${slot.image}`} alt={slot.name} className="h-10 w-8 rounded object-cover object-top" />}
                                <div className="flex min-w-[40px] flex-col leading-tight">
                                    <span className="text-[10px] font-bold uppercase tracking-wider">{slot.name ? slot.name.split(',')[0].trim() : 'In-game favorite'}</span>
                                    <span className="text-[9px] text-muted-foreground">Slot {slot.index + 1}</span>
                                </div>
                                <button type="button" onClick={() => handleApplyFromMirror(slot.index)} className="text-primary transition-colors hover:text-primary/80" title="Apply this preset to character">
                                    <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M5 13l4 4L19 7" />
                                    </svg>
                                </button>
                                <button type="button" onClick={() => handleRemoveFav(slot.index)} className="text-red-400 transition-colors hover:text-red-300" title="Remove from Favorites">
                                    <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                                    </svg>
                                </button>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {zoomed && createPortal(
                <div
                    role="dialog"
                    aria-label="Appearance preview"
                    className="fixed inset-0 z-[100] flex cursor-pointer items-center justify-center bg-black/85 backdrop-blur-sm"
                    onClick={() => setZoomed(null)}
                >
                    <img src={zoomed} alt="Preview" className="max-h-[85vh] max-w-[85vw] rounded-xl object-contain shadow-2xl animate-in zoom-in-90 duration-300" onClick={event => event.stopPropagation()} />
                    <button type="button" aria-label="Close preview" onClick={() => setZoomed(null)} className="absolute right-6 top-6 text-white/70 transition-colors hover:text-white">
                        <svg className="h-8 w-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>,
                document.body,
            )}
        </div>
    );
}
