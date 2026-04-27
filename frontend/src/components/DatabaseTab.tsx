import {useEffect, useState, useMemo, useRef} from 'react';
import toast from '../lib/toast';
import {useVirtualizer} from '@tanstack/react-virtual';
import {GetItemList, GetInfuseTypes, AddItemsToCharacter, GetCharacter} from '../../wailsjs/go/main/App';
import {db, vm} from '../../wailsjs/go/models';
import type {AddSettings} from '../App';
import {CategorySelect} from './CategorySelect';
import {RiskBadge} from './RiskBadge';

interface DatabaseTabProps {
    columnVisibility: {
        id: boolean;
        category: boolean;
    };
    platform: string | null;
    charIndex: number;
    inventoryVersion: number;
    onItemsAdded?: () => void;
    addSettings: AddSettings;
    showFlaggedItems: boolean;
    category: string;
    setCategory: (value: string) => void;
    onSelectItem?: (item: db.ItemEntry | null) => void;
    readOnly?: boolean;
}

// Determine if ALL selected items are non-stackable (max qty == 1)
function allNonStackable(items: db.ItemEntry[]): boolean {
    return items.every(i => i.maxInventory <= 1);
}

export function DatabaseTab({columnVisibility, platform, charIndex, inventoryVersion, onItemsAdded, addSettings, showFlaggedItems, category, setCategory, onSelectItem, readOnly = false}: DatabaseTabProps) {
    const {upgrade25, upgrade10, infuseOffset, upgradeAsh} = addSettings;
    const [search, setSearch] = useState('');
    const [dbItems, setDbItems] = useState<db.ItemEntry[]>([]);
    const [loading, setLoading] = useState(false);
    const [infuseTypes, setInfuseTypes] = useState<db.InfuseType[]>([]);

    // Sorting
    const [sortCol, setSortCol] = useState<string>('name');
    const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');

    // Selection
    const [selectedDbItems, setSelectedDbItems] = useState<Set<number>>(new Set());

    // Modal state
    const [confirmModal, setConfirmModal] = useState<db.ItemEntry[] | null>(null);
    const [banRiskWarning, setBanRiskWarning] = useState<db.ItemEntry[] | null>(null);
    const [ignoreBanRisk, setIgnoreBanRisk] = useState<boolean>(() => {
        return localStorage.getItem('setting:ignoreBanRiskWarning') === 'true';
    });
    const [isSaving, setIsSaving] = useState(false);
    const [brokenIcons, setBrokenIcons] = useState<Set<string>>(new Set());

    // Quantity state for modal
    const [addToInv, setAddToInv] = useState(true);
    const [invMax, setInvMax] = useState(false);
    const [invQtyVal, setInvQtyVal] = useState(1);
    const [addToStorage, setAddToStorage] = useState(false);
    const [storageMax, setStorageMax] = useState(false);
    const [storageQtyVal, setStorageQtyVal] = useState(1);

    // Icon preview
    const [selectedIcon, setSelectedIcon] = useState<{name: string, path: string} | null>(null);

    // Detail drawer
    const [detailItem, setDetailItem] = useState<db.ItemEntry | null>(null);

    // Owned items (for "Inventory" / "Storage" columns)
    const [charInventory, setCharInventory] = useState<vm.ItemViewModel[]>([]);
    const [charStorage, setCharStorage] = useState<vm.ItemViewModel[]>([]);

    useEffect(() => {
        GetInfuseTypes().then(res => setInfuseTypes(res || []));
    }, []);

    useEffect(() => {
        GetCharacter(charIndex).then(res => {
            setCharInventory(res?.inventory || []);
            setCharStorage(res?.storage || []);
        }).catch(() => {
            setCharInventory([]);
            setCharStorage([]);
        });
    }, [charIndex, inventoryVersion]);

    // Map BaseID → {inv, storage} owned counts.
    // Stackable items contribute their stack quantity; non-stackable contribute one per copy.
    // Matching by BaseID groups all upgrade/infusion variants of the same weapon.
    const ownedByBaseID = useMemo(() => {
        const m = new Map<number, {inv: number; storage: number}>();
        const bump = (baseId: number, key: 'inv' | 'storage', n: number) => {
            const cur = m.get(baseId) ?? {inv: 0, storage: 0};
            cur[key] += n;
            m.set(baseId, cur);
        };
        charInventory.forEach(it => {
            const qty = it.maxInventory <= 1 ? 1 : it.quantity;
            bump(it.baseId || it.id, 'inv', qty);
        });
        charStorage.forEach(it => {
            const qty = it.maxInventory <= 1 ? 1 : it.quantity;
            bump(it.baseId || it.id, 'storage', qty);
        });
        return m;
    }, [charInventory, charStorage]);

    useEffect(() => {
        setLoading(true);
        setSelectedDbItems(new Set());
        GetItemList(category).then(res => {
            setDbItems(res || []);
            setLoading(false);
        }).catch(err => {
            console.error("Failed to load items:", err);
            setLoading(false);
        });
    }, [category]);

    const filteredItems = useMemo(() => dbItems.filter(item => {
        // "Cut & Ban-Risk" toggle hides only risky-flagged items, not informational flags
        // (dlc, stackable) which are now present on most entries.
        const RISKY_FLAGS = ['cut_content', 'ban_risk', 'pre_order', 'dlc_duplicate'];
        if (!showFlaggedItems && item.flags?.some(f => RISKY_FLAGS.includes(f))) return false;
        return item.name.toLowerCase().includes(search.toLowerCase()) ||
            item.id.toString(16).includes(search.toLowerCase());
    }).sort((a, b) => {
        const aVal = a[sortCol as keyof db.ItemEntry] ?? '';
        const bVal = b[sortCol as keyof db.ItemEntry] ?? '';
        if (aVal < bVal) return sortDir === 'asc' ? -1 : 1;
        if (aVal > bVal) return sortDir === 'asc' ? 1 : -1;
        return 0;
    }), [dbItems, search, sortCol, sortDir, showFlaggedItems]);

    const handleSort = (col: string) => {
        if (sortCol === col) setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
        else { setSortCol(col); setSortDir('asc'); }
    };

    const toggleItem = (id: number) => {
        const next = new Set(selectedDbItems);
        if (next.has(id)) next.delete(id); else next.add(id);
        setSelectedDbItems(next);
    };

    const toggleAll = () => {
        if (selectedDbItems.size === filteredItems.length && filteredItems.length > 0)
            setSelectedDbItems(new Set());
        else
            setSelectedDbItems(new Set(filteredItems.map(i => i.id)));
    };

    const handleAdd = async () => {
        if (!confirmModal || isSaving) return;
        setIsSaving(true);
        try {
            const baseIds = confirmModal.map(i => i.id);

            if (modalNonStackable) {
                // Non-stackable: each copy needs its own GaItem handle in the backend.
                // When adding exactly 1 copy to both inv and storage, use a single call so
                // both locations share the same handle (prevents duplicate GaItem records).
                const bothActive = addToInv && invQtyVal > 0 && addToStorage && storageQtyVal > 0;
                if (bothActive && invQtyVal === 1 && storageQtyVal === 1) {
                    await AddItemsToCharacter(charIndex, baseIds, upgrade25, upgrade10, infuseOffset, upgradeAsh, 1, 1);
                } else {
                    if (addToInv && invQtyVal > 0) {
                        const ids = invQtyVal > 1
                            ? confirmModal.flatMap(i => Array<number>(invQtyVal).fill(i.id))
                            : baseIds;
                        await AddItemsToCharacter(charIndex, ids, upgrade25, upgrade10, infuseOffset, upgradeAsh, 1, 0);
                    }
                    if (addToStorage && storageQtyVal > 0) {
                        const ids = storageQtyVal > 1
                            ? confirmModal.flatMap(i => Array<number>(storageQtyVal).fill(i.id))
                            : baseIds;
                        await AddItemsToCharacter(charIndex, ids, upgrade25, upgrade10, infuseOffset, upgradeAsh, 0, 1);
                    }
                }
            } else {
                // Stackable: single call with qty values.
                const invQty = !addToInv ? 0 : invMax ? -1 : invQtyVal;
                const storQty = !addToStorage ? 0 : storageMax ? -1 : storageQtyVal;
                await AddItemsToCharacter(charIndex, baseIds, upgrade25, upgrade10, infuseOffset, upgradeAsh, invQty, storQty);
            }

            setConfirmModal(null);
            setSelectedDbItems(new Set());
            onItemsAdded?.();
        } catch (err) {
            toast.error('Failed to add items: ' + err);
        } finally {
            setIsSaving(false);
        }
    };

    const openModal = (items: db.ItemEntry[]) => {
        // Gate: if any selected item is ban_risk-flagged AND user hasn't opted out, show warning first.
        // The warning modal's "Add Anyway" calls openConfirmModal directly to bypass this check.
        if (!ignoreBanRisk && items.some(i => i.flags?.includes('ban_risk'))) {
            setBanRiskWarning(items);
            return;
        }
        openConfirmModal(items);
    };

    const openConfirmModal = (items: db.ItemEntry[]) => {
        setAddToInv(true);
        setInvMax(false);
        setInvQtyVal(1);
        setAddToStorage(true);
        setStorageMax(false);
        setStorageQtyVal(1);
        setConfirmModal(items);
    };

    const handleIgnoreBanRiskChange = (checked: boolean) => {
        setIgnoreBanRisk(checked);
        localStorage.setItem('setting:ignoreBanRiskWarning', String(checked));
    };

    const handleImageError = (iconPath: string) => {
        setBrokenIcons(prev => new Set(prev).add(iconPath));
    };

    const selectedInfuseName = infuseTypes.find(t => t.offset === infuseOffset)?.name ?? 'Standard';

    const scrollRef = useRef<HTMLDivElement>(null);
    const rowVirtualizer = useVirtualizer({
        count: filteredItems.length,
        getScrollElement: () => scrollRef.current,
        estimateSize: () => 52,
        overscan: 20,
    });

    // Header has: (optional checkbox) + icon + name (2 fixed) + optional ID + optional Category + Inv/Storage (always 2)
    const colCount = 4
        + (readOnly ? 0 : 1)
        + (columnVisibility.id ? 1 : 0)
        + (category === 'all' || columnVisibility.category ? 1 : 0);

    // Whether the modal items are all non-stackable (weapons/armor/talismans)
    const modalNonStackable = confirmModal ? allNonStackable(confirmModal) : true;
    const modalMaxInv = confirmModal ? Math.min(...confirmModal.map(i => i.maxInventory)) : 1;
    const modalMaxStorage = confirmModal ? Math.min(...confirmModal.map(i => i.maxStorage)) : 1;
    const modalMixedMaxes = confirmModal && confirmModal.length > 1 && !modalNonStackable &&
        (new Set(confirmModal.map(i => i.maxInventory)).size > 1 || new Set(confirmModal.map(i => i.maxStorage)).size > 1);

    return (
        <div className="flex-1 flex flex-col min-h-0 space-y-3">
            {/* Ban-risk Warning Modal — gates confirmModal when any selected item has ban_risk flag */}
            {banRiskWarning && (() => {
                const banRiskItems = banRiskWarning.filter(i => i.flags?.includes('ban_risk'));
                return (
                    <div className="fixed inset-0 z-[120] flex items-center justify-center bg-background/80 backdrop-blur-sm animate-in fade-in duration-300">
                        <div className="bg-card p-8 rounded-2xl border-2 border-red-500/40 flex flex-col space-y-5 max-w-md w-full mx-4 shadow-2xl shadow-red-500/20 animate-in zoom-in-95 duration-300">
                            {/* Header */}
                            <div className="flex items-center space-x-3">
                                <div className="w-10 h-10 rounded-full bg-red-500/15 border border-red-500/40 flex items-center justify-center">
                                    <svg className="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                                    </svg>
                                </div>
                                <div>
                                    <h3 className="text-sm font-black uppercase tracking-[0.15em] text-red-500">Ban Risk Warning</h3>
                                    <p className="text-[9px] font-bold text-muted-foreground uppercase tracking-widest">Cut content / cheat-flagged item</p>
                                </div>
                            </div>

                            {/* Warning text */}
                            <p className="text-[11px] text-foreground leading-relaxed">
                                {banRiskItems.length === 1 ? (
                                    <>
                                        <strong>{banRiskItems[0].name}</strong> is flagged as <strong>ban-risk</strong>.
                                        Adding it to your save may trigger Easy Anti-Cheat detection if you go online.
                                    </>
                                ) : (
                                    <>
                                        <strong>{banRiskItems.length}</strong> of the selected items are flagged as <strong>ban-risk</strong>.
                                        Adding them to your save may trigger Easy Anti-Cheat detection if you go online.
                                    </>
                                )}
                            </p>

                            {/* List of ban-risk items */}
                            {banRiskItems.length > 1 && (
                                <div className="bg-red-500/5 border border-red-500/20 rounded-md p-3 max-h-32 overflow-y-auto custom-scrollbar">
                                    <ul className="text-[10px] text-red-500/90 list-disc list-inside space-y-0.5">
                                        {banRiskItems.map(i => <li key={i.id}>{i.name}</li>)}
                                    </ul>
                                </div>
                            )}

                            {/* Ignore checkbox */}
                            <label className="flex items-center justify-between p-2.5 rounded-md bg-muted/20 border border-border/50 cursor-pointer hover:bg-muted/30 transition-all">
                                <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">
                                    Ignore all ban risk warnings
                                </span>
                                <input
                                    type="checkbox"
                                    checked={ignoreBanRisk}
                                    onChange={e => handleIgnoreBanRiskChange(e.target.checked)}
                                    className="w-3.5 h-3.5 rounded border-border text-red-500 focus:ring-red-500/20"
                                />
                            </label>

                            {/* Actions */}
                            <div className="flex space-x-2">
                                <button
                                    onClick={() => setBanRiskWarning(null)}
                                    className="flex-1 px-4 py-2.5 bg-muted/30 text-muted-foreground rounded-md text-[10px] font-black uppercase tracking-widest border border-border hover:bg-muted/50 transition-all"
                                >
                                    Cancel
                                </button>
                                <button
                                    onClick={() => {
                                        const items = banRiskWarning;
                                        setBanRiskWarning(null);
                                        openConfirmModal(items);
                                    }}
                                    className="flex-1 px-4 py-2.5 bg-red-500 text-white rounded-md text-[10px] font-black uppercase tracking-widest hover:brightness-110 active:scale-95 transition-all"
                                >
                                    Add Anyway
                                </button>
                            </div>
                        </div>
                    </div>
                );
            })()}

            {/* Confirm Modal */}
            {confirmModal && (
                <div className="fixed inset-0 z-[110] flex items-center justify-center bg-background/80 backdrop-blur-sm animate-in fade-in duration-300">
                    <div className="bg-card p-8 rounded-2xl border border-primary/20 flex flex-col space-y-6 max-w-sm w-full mx-4 shadow-2xl shadow-primary/20 animate-in zoom-in-95 duration-300">
                        {/* Header */}
                        <div className="flex items-center space-x-4">
                            {confirmModal.length === 1 ? (
                                <>
                                    <div className="w-12 h-12 rounded bg-muted/30 border border-border/50 flex items-center justify-center overflow-hidden">
                                        {brokenIcons.has(confirmModal[0].iconPath) ? (
                                            <span className="text-[10px] font-black text-muted-foreground/30 select-none">?</span>
                                        ) : (
                                            <img src={confirmModal[0].iconPath} alt="" className="w-8 h-8 object-contain" onError={() => handleImageError(confirmModal[0].iconPath)} />
                                        )}
                                    </div>
                                    <div>
                                        <h3 className="text-sm font-black uppercase tracking-widest text-foreground">{confirmModal[0].name}</h3>
                                        <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">{confirmModal[0].category}</p>
                                    </div>
                                </>
                            ) : (
                                <>
                                    <div className="w-12 h-12 rounded bg-primary/10 border border-primary/30 flex items-center justify-center">
                                        <span className="text-lg font-black text-primary">{confirmModal.length}</span>
                                    </div>
                                    <div>
                                        <h3 className="text-sm font-black uppercase tracking-widest text-foreground">Add {confirmModal.length} Items</h3>
                                        <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Bulk Action</p>
                                    </div>
                                </>
                            )}
                        </div>

                        {/* Inventory row */}
                        <div className="space-y-3">
                            <div className="flex items-center space-x-3">
                                <div
                                    onClick={() => setAddToInv(!addToInv)}
                                    className={`w-5 h-5 rounded border flex items-center justify-center transition-all cursor-pointer shrink-0 ${addToInv ? 'bg-primary border-primary' : 'bg-muted/30 border-border hover:border-primary/50'}`}
                                >
                                    {addToInv && <svg className="w-3.5 h-3.5 text-primary-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                </div>
                                <span className="text-[11px] font-bold uppercase tracking-widest text-foreground/80 w-20 shrink-0">Inventory</span>
                                <input
                                    type="number"
                                    min={1}
                                    max={modalNonStackable ? 99 : modalMaxInv}
                                    value={invMax ? modalMaxInv : invQtyVal}
                                    disabled={!addToInv || invMax}
                                    onChange={e => setInvQtyVal(Math.max(1, Math.min(modalNonStackable ? 99 : modalMaxInv, parseInt(e.target.value) || 1)))}
                                    className="w-20 bg-background border border-border/50 rounded px-2 py-1 text-[10px] font-mono text-center focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all disabled:opacity-40"
                                />
                                {modalNonStackable && (
                                    <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Copies</span>
                                )}
                                {!modalNonStackable && modalMaxInv > 1 && (
                                    <div
                                        onClick={() => addToInv && setInvMax(!invMax)}
                                        className={`flex items-center space-x-1.5 cursor-pointer group ${!addToInv ? 'opacity-40 pointer-events-none' : ''}`}
                                    >
                                        <div className={`w-4 h-4 rounded border flex items-center justify-center transition-all ${invMax ? 'bg-primary border-primary' : 'bg-muted/30 border-border group-hover:border-primary/50'}`}>
                                            {invMax && <svg className="w-2.5 h-2.5 text-primary-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                        </div>
                                        <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Max ({modalMaxInv})</span>
                                    </div>
                                )}
                            </div>

                            {/* Storage row */}
                            <div className="flex items-center space-x-3">
                                <div
                                    onClick={() => setAddToStorage(!addToStorage)}
                                    className={`w-5 h-5 rounded border flex items-center justify-center transition-all cursor-pointer shrink-0 ${addToStorage ? 'bg-primary border-primary' : 'bg-muted/30 border-border hover:border-primary/50'}`}
                                >
                                    {addToStorage && <svg className="w-3.5 h-3.5 text-primary-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                </div>
                                <span className="text-[11px] font-bold uppercase tracking-widest text-foreground/80 w-20 shrink-0">Storage</span>
                                <input
                                    type="number"
                                    min={1}
                                    max={modalNonStackable ? 99 : modalMaxStorage}
                                    value={storageMax ? modalMaxStorage : storageQtyVal}
                                    disabled={!addToStorage || storageMax}
                                    onChange={e => setStorageQtyVal(Math.max(1, Math.min(modalNonStackable ? 99 : modalMaxStorage, parseInt(e.target.value) || 1)))}
                                    className="w-20 bg-background border border-border/50 rounded px-2 py-1 text-[10px] font-mono text-center focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all disabled:opacity-40"
                                />
                                {modalNonStackable && (
                                    <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Copies</span>
                                )}
                                {!modalNonStackable && modalMaxStorage > 1 && (
                                    <div
                                        onClick={() => addToStorage && setStorageMax(!storageMax)}
                                        className={`flex items-center space-x-1.5 cursor-pointer group ${!addToStorage ? 'opacity-40 pointer-events-none' : ''}`}
                                    >
                                        <div className={`w-4 h-4 rounded border flex items-center justify-center transition-all ${storageMax ? 'bg-primary border-primary' : 'bg-muted/30 border-border group-hover:border-primary/50'}`}>
                                            {storageMax && <svg className="w-2.5 h-2.5 text-primary-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                        </div>
                                        <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Max ({modalMaxStorage})</span>
                                    </div>
                                )}
                            </div>
                        </div>

                        {modalMixedMaxes && (
                            <p className="text-[9px] font-bold text-amber-500 bg-amber-500/10 border border-amber-500/20 rounded px-3 py-1.5">
                                Qty capped to lowest max: Inv {modalMaxInv}, Storage {modalMaxStorage}
                            </p>
                        )}

                        <div className="flex space-x-3 pt-2">
                            <button onClick={handleAdd} disabled={isSaving || (!addToInv && !addToStorage)} className="flex-1 px-4 py-2.5 bg-primary text-primary-foreground rounded-md text-[10px] font-black uppercase tracking-widest shadow-lg shadow-primary/20 hover:scale-[1.02] active:scale-[0.98] transition-all disabled:opacity-50">
                                {isSaving ? 'Adding...' : 'Add'}
                            </button>
                            <button onClick={() => setConfirmModal(null)} className="flex-1 px-4 py-2.5 bg-muted/30 text-muted-foreground rounded-md text-[10px] font-black uppercase tracking-widest border border-border hover:bg-muted/50 transition-all">
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Search / Category bar — filter BEFORE search */}
            <div className="flex items-center justify-between bg-muted/10 p-4 rounded-xl border border-border/50 backdrop-blur-sm sticky top-0 z-20">
                <div className="flex items-center space-x-4 flex-1">
                    {/* Filter (first) */}
                    <CategorySelect value={category} onChange={setCategory} className="w-56 shrink-0" />

                    {/* Search (second) */}
                    <div className="relative flex-1 max-w-md group">
                        <div className="absolute inset-y-0 left-3 flex items-center pointer-events-none">
                            <svg className="w-3.5 h-3.5 text-muted-foreground group-focus-within:text-primary transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>
                        </div>
                        <input
                            type="text"
                            placeholder="Search by name or ID..."
                            value={search}
                            onChange={e => setSearch(e.target.value)}
                            className="w-full bg-background border border-border/50 rounded-lg py-2 pl-10 pr-4 text-[10px] font-bold uppercase tracking-wider focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                        />
                    </div>

                    {!readOnly && selectedDbItems.size > 0 && (
                        <button
                            onClick={() => openModal(dbItems.filter(i => selectedDbItems.has(i.id)))}
                            disabled={!platform}
                            className="px-6 py-2 bg-primary text-primary-foreground rounded-lg text-[9px] font-black uppercase tracking-[0.2em] shadow-xl shadow-primary/20 hover:brightness-110 active:scale-95 transition-all animate-in zoom-in-95 duration-300 disabled:opacity-50 disabled:grayscale disabled:cursor-not-allowed"
                        >
                            Add Selected ({selectedDbItems.size})
                        </button>
                    )}
                </div>
                <div className="flex items-center space-x-2 ml-4">
                    <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground bg-muted/20 px-3 py-1.5 rounded-full border border-border/30">
                        {filteredItems.length} Items
                    </span>
                </div>
            </div>

            {/* Table */}
            <div className="flex-1 bg-muted/5 rounded-xl border border-border/50 overflow-hidden flex flex-col relative">
                {loading && (
                    <div className="absolute inset-0 bg-background/50 backdrop-blur-[2px] z-30 flex items-center justify-center">
                        <div className="flex flex-col items-center space-y-4">
                            <div className="w-10 h-10 border-4 border-primary/20 border-t-primary rounded-full animate-spin" />
                            <span className="text-[10px] font-black uppercase tracking-[0.2em] text-primary animate-pulse">Loading Database...</span>
                        </div>
                    </div>
                )}

                <div ref={scrollRef} className="flex-1 overflow-y-auto custom-scrollbar">
                    <table className="w-full text-left border-collapse">
                        <thead className="sticky top-0 z-20 bg-muted/80 backdrop-blur-md border-b border-border shadow-sm">
                            <tr className="text-[9px] font-black uppercase tracking-[0.15em] text-muted-foreground">
                                {!readOnly && (
                                    <th className="p-4 w-10">
                                        <div
                                            onClick={toggleAll}
                                            className={`w-4 h-4 rounded border flex items-center justify-center transition-all cursor-pointer ${selectedDbItems.size === filteredItems.length && filteredItems.length > 0 ? 'bg-primary border-primary' : 'bg-muted/30 border-border hover:border-primary/50'}`}
                                        >
                                            {selectedDbItems.size === filteredItems.length && filteredItems.length > 0 &&
                                                <svg className="w-3 h-3 text-primary-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                        </div>
                                    </th>
                                )}
                                <th className="p-4 w-12">Icon</th>
                                <th className="p-4 cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('name')}>
                                    Name {sortCol === 'name' && (sortDir === 'asc' ? '↑' : '↓')}
                                </th>
                                {columnVisibility.id && (
                                    <th className="p-4 cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('id')}>
                                        ID {sortCol === 'id' && (sortDir === 'asc' ? '↑' : '↓')}
                                    </th>
                                )}
                                {(category === 'all' || columnVisibility.category) && (
                                    <th className="p-4 cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('category')}>
                                        Category {sortCol === 'category' && (sortDir === 'asc' ? '↑' : '↓')}
                                    </th>
                                )}
                                <th className="p-4 text-center w-32">Inventory</th>
                                <th className="p-4 text-center w-32">Storage</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border/30">
                            {rowVirtualizer.getVirtualItems().length > 0 && rowVirtualizer.getVirtualItems()[0].start > 0 && (
                                <tr><td colSpan={colCount} style={{ height: rowVirtualizer.getVirtualItems()[0].start, padding: 0, border: 'none' }} /></tr>
                            )}
                            {rowVirtualizer.getVirtualItems().map(virtualRow => {
                                const item = filteredItems[virtualRow.index];
                                const isUpgradeable = item.maxUpgrade > 0;
                                const isAsh = item.category === 'ashes';
                                const hasInfuse = item.maxUpgrade === 25 && infuseOffset !== 0;
                                const levelVal = isAsh ? upgradeAsh : (item.maxUpgrade === 25 ? upgrade25 : item.maxUpgrade === 10 ? upgrade10 : 0);
                                const showPreview = isUpgradeable && (levelVal > 0 || hasInfuse);
                                const previewParts: string[] = [];
                                if (hasInfuse) previewParts.push(selectedInfuseName);
                                if (levelVal > 0) previewParts.push(`+${levelVal}`);

                                return (
                                    <tr key={item.id} data-index={virtualRow.index} ref={node => { if (node) rowVirtualizer.measureElement(node); }} className={`group hover:bg-primary/[0.03] transition-colors ${selectedDbItems.has(item.id) ? 'bg-primary/[0.02]' : ''}`}>
                                        {!readOnly && (
                                            <td className="p-4">
                                                <div
                                                    onClick={() => toggleItem(item.id)}
                                                    className={`w-4 h-4 rounded border flex items-center justify-center transition-all cursor-pointer ${selectedDbItems.has(item.id) ? 'bg-primary border-primary' : 'bg-muted/30 border-border group-hover:border-primary/50'}`}
                                                >
                                                    {selectedDbItems.has(item.id) &&
                                                        <svg className="w-3 h-3 text-primary-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                                </div>
                                            </td>
                                        )}
                                        <td className="px-4 py-0.5">
                                            <div
                                                className="w-12 h-12 bg-muted/20 rounded-lg border border-border/50 flex items-center justify-center overflow-hidden group-hover:border-primary/30 transition-all cursor-zoom-in"
                                                onClick={() => setSelectedIcon({name: item.name, path: item.iconPath})}
                                            >
                                                {brokenIcons.has(item.iconPath) ? (
                                                    <span className="text-[10px] font-black text-muted-foreground/30 select-none">?</span>
                                                ) : (
                                                    <img
                                                        src={item.iconPath}
                                                        alt={item.name}
                                                        className="w-full h-full p-0.5 object-contain drop-shadow-md group-hover:scale-110 transition-transform duration-300"
                                                        onError={() => handleImageError(item.iconPath)}
                                                    />
                                                )}
                                            </div>
                                        </td>
                                        <td className="p-4">
                                            <div className="flex flex-col gap-0.5">
                                                <div className="flex items-center gap-1.5 flex-wrap">
                                                    <span
                                                        className="text-[13px] font-semibold text-foreground group-hover:text-primary transition-colors cursor-pointer hover:underline decoration-primary/40 underline-offset-2"
                                                        onClick={e => { e.stopPropagation(); onSelectItem ? onSelectItem(item) : setDetailItem(item); }}
                                                    >{item.name}</span>
                                                    {item.flags?.includes('cut_content') && (
                                                        <RiskBadge flag="cut_content" />
                                                    )}
                                                    {item.flags?.includes('ban_risk') && (
                                                        <RiskBadge flag="ban_risk" />
                                                    )}
                                                </div>
                                                {showPreview && (
                                                    <span className="text-[8px] font-mono font-bold text-primary/60 uppercase tracking-tight">
                                                        {previewParts.join(' ')}
                                                    </span>
                                                )}
                                            </div>
                                        </td>
                                        {columnVisibility.id && (
                                            <td className="p-4 text-[10px] font-mono text-muted-foreground">0x{item.id.toString(16).toUpperCase()}</td>
                                        )}
                                        {(category === 'all' || columnVisibility.category) && (
                                            <td className="p-4">
                                                <span className="text-[8px] font-black uppercase tracking-widest px-2 py-1 bg-muted/30 rounded-md text-muted-foreground border border-border/20">
                                                    {item.category === 'arrows_and_bolts' ? 'Arrows & Bolts' : item.category.replace(/_/g, ' ')}
                                                </span>
                                            </td>
                                        )}
                                        {(() => {
                                            const owned = ownedByBaseID.get(item.id) ?? {inv: 0, storage: 0};
                                            const cellClass = (have: number, max: number): string => {
                                                if (have === 0) return 'text-muted-foreground/50 bg-muted/20 border-border/30';
                                                if (max > 0 && have >= max) return 'text-amber-500 bg-amber-500/10 border-amber-500/30';
                                                return 'text-green-500 bg-green-500/10 border-green-500/30';
                                            };
                                            return (
                                                <>
                                                    <td className="p-4 text-center">
                                                        <span className={`inline-block text-[10px] font-black tabular-nums px-2 py-1 rounded border ${cellClass(owned.inv, item.maxInventory)}`}>
                                                            {owned.inv} / {item.maxInventory}
                                                        </span>
                                                    </td>
                                                    <td className="p-4 text-center">
                                                        <span className={`inline-block text-[10px] font-black tabular-nums px-2 py-1 rounded border ${cellClass(owned.storage, item.maxStorage)}`}>
                                                            {owned.storage} / {item.maxStorage}
                                                        </span>
                                                    </td>
                                                </>
                                            );
                                        })()}
                                    </tr>
                                );
                            })}
                            {(() => {
                                const virtualItems = rowVirtualizer.getVirtualItems();
                                const paddingBottom = virtualItems.length > 0
                                    ? rowVirtualizer.getTotalSize() - virtualItems[virtualItems.length - 1].end
                                    : 0;
                                return paddingBottom > 0 ? <tr><td colSpan={colCount} style={{ height: paddingBottom, padding: 0, border: 'none' }} /></tr> : null;
                            })()}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Item Detail Drawer */}
            {detailItem && (
                <div className="fixed inset-0 z-[100] flex justify-end animate-in fade-in duration-200" onClick={() => setDetailItem(null)}>
                    <div className="absolute inset-0 bg-background/60 backdrop-blur-sm" />
                    <div
                        className="relative w-full max-w-md h-full bg-card border-l border-border shadow-2xl overflow-y-auto animate-in slide-in-from-right duration-300"
                        onClick={e => e.stopPropagation()}
                    >
                        {/* Header */}
                        <div className="sticky top-0 z-10 bg-card/95 backdrop-blur-md border-b border-border p-6 flex items-start gap-4">
                            <div className="w-16 h-16 rounded-lg bg-muted/30 border border-border/50 flex items-center justify-center overflow-hidden shrink-0">
                                {brokenIcons.has(detailItem.iconPath) ? (
                                    <span className="text-xl font-black text-muted-foreground/30">?</span>
                                ) : (
                                    <img src={detailItem.iconPath} alt="" className="w-12 h-12 object-contain drop-shadow-md" onError={() => handleImageError(detailItem.iconPath)} />
                                )}
                            </div>
                            <div className="flex-1 min-w-0">
                                <h3 className="text-sm font-black uppercase tracking-widest text-foreground truncate">{detailItem.name}</h3>
                                <p className="text-[9px] font-bold text-muted-foreground uppercase tracking-widest mt-0.5">
                                    {detailItem.category.replace(/_/g, ' ')}
                                </p>
                                <p className="text-[9px] font-mono text-muted-foreground/60 mt-0.5">
                                    0x{detailItem.id.toString(16).toUpperCase()}
                                </p>
                            </div>
                            <button
                                onClick={() => setDetailItem(null)}
                                className="p-1.5 rounded-md hover:bg-muted/50 text-muted-foreground hover:text-foreground transition-all shrink-0"
                            >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M6 18L18 6M6 6l12 12"/></svg>
                            </button>
                        </div>

                        <div className="p-6 space-y-6">
                            {/* Weight */}
                            {(detailItem.weight || detailItem.weapon?.Weight || detailItem.armor?.Weight) ? (
                                <div className="flex items-center gap-2">
                                    <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Weight</span>
                                    <span className="text-xs font-bold text-foreground">
                                        {detailItem.weapon?.Weight ?? detailItem.armor?.Weight ?? detailItem.weight ?? 0}
                                    </span>
                                </div>
                            ) : null}

                            {/* Description */}
                            {detailItem.description && (
                                <div className="space-y-2">
                                    <h4 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Description</h4>
                                    <p className="text-[11px] leading-relaxed text-foreground/80 whitespace-pre-line">
                                        {detailItem.description}
                                    </p>
                                </div>
                            )}

                            {/* Weapon Stats */}
                            {detailItem.weapon && (
                                <div className="space-y-3">
                                    <h4 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Attack Power</h4>
                                    <div className="grid grid-cols-2 gap-2">
                                        {[
                                            ['Physical', detailItem.weapon.PhysDamage],
                                            ['Magic', detailItem.weapon.MagDamage],
                                            ['Fire', detailItem.weapon.FireDamage],
                                            ['Lightning', detailItem.weapon.LitDamage],
                                            ['Holy', detailItem.weapon.HolyDamage],
                                        ].filter(([, v]) => v > 0).map(([label, val]) => (
                                            <div key={label as string} className="flex items-center justify-between bg-muted/20 rounded px-3 py-1.5 border border-border/30">
                                                <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground">{label}</span>
                                                <span className="text-[11px] font-black text-foreground">{val}</span>
                                            </div>
                                        ))}
                                    </div>

                                    <h4 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground pt-2">Scaling</h4>
                                    <div className="flex gap-2 flex-wrap">
                                        {[
                                            ['STR', detailItem.weapon.ScaleStr],
                                            ['DEX', detailItem.weapon.ScaleDex],
                                            ['INT', detailItem.weapon.ScaleInt],
                                            ['FAI', detailItem.weapon.ScaleFai],
                                        ].filter(([, v]) => v > 0).map(([label, val]) => (
                                            <div key={label as string} className="flex flex-col items-center bg-muted/20 rounded px-3 py-1.5 border border-border/30 min-w-[50px]">
                                                <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">{label}</span>
                                                <span className="text-[11px] font-black text-foreground">{val}</span>
                                            </div>
                                        ))}
                                    </div>

                                    <h4 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground pt-2">Requirements</h4>
                                    <div className="flex gap-2 flex-wrap">
                                        {[
                                            ['STR', detailItem.weapon.ReqStr],
                                            ['DEX', detailItem.weapon.ReqDex],
                                            ['INT', detailItem.weapon.ReqInt],
                                            ['FAI', detailItem.weapon.ReqFai],
                                            ['ARC', detailItem.weapon.ReqArc],
                                        ].filter(([, v]) => v > 0).map(([label, val]) => (
                                            <div key={label as string} className="flex flex-col items-center bg-muted/20 rounded px-3 py-1.5 border border-border/30 min-w-[50px]">
                                                <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">{label}</span>
                                                <span className="text-[11px] font-black text-foreground">{val}</span>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* Armor Stats */}
                            {detailItem.armor && (
                                <div className="space-y-3">
                                    <h4 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Damage Negation</h4>
                                    <div className="grid grid-cols-2 gap-2">
                                        {[
                                            ['Physical', detailItem.armor.Physical],
                                            ['Strike', detailItem.armor.Strike],
                                            ['Slash', detailItem.armor.Slash],
                                            ['Pierce', detailItem.armor.Pierce],
                                            ['Magic', detailItem.armor.Magic],
                                            ['Fire', detailItem.armor.Fire],
                                            ['Lightning', detailItem.armor.Lightning],
                                            ['Holy', detailItem.armor.Holy],
                                        ].map(([label, val]) => (
                                            <div key={label as string} className="flex items-center justify-between bg-muted/20 rounded px-3 py-1.5 border border-border/30">
                                                <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground">{label}</span>
                                                <span className="text-[11px] font-black text-foreground">{(val as number).toFixed(1)}%</span>
                                            </div>
                                        ))}
                                    </div>

                                    <h4 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground pt-2">Resistance</h4>
                                    <div className="grid grid-cols-2 gap-2">
                                        {[
                                            ['Immunity', detailItem.armor.Immunity],
                                            ['Robustness', detailItem.armor.Robustness],
                                            ['Focus', detailItem.armor.Focus],
                                            ['Vitality', detailItem.armor.Vitality],
                                        ].map(([label, val]) => (
                                            <div key={label as string} className="flex items-center justify-between bg-muted/20 rounded px-3 py-1.5 border border-border/30">
                                                <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground">{label}</span>
                                                <span className="text-[11px] font-black text-foreground">{val}</span>
                                            </div>
                                        ))}
                                    </div>

                                    {detailItem.armor.Poise > 0 && (
                                        <div className="flex items-center gap-2 pt-1">
                                            <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Poise</span>
                                            <span className="text-xs font-bold text-foreground">{detailItem.armor.Poise.toFixed(1)}</span>
                                        </div>
                                    )}
                                </div>
                            )}

                            {/* Spell Stats */}
                            {detailItem.spell && (
                                <div className="space-y-3">
                                    <h4 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Spell Info</h4>
                                    <div className="grid grid-cols-2 gap-2">
                                        <div className="flex items-center justify-between bg-muted/20 rounded px-3 py-1.5 border border-border/30">
                                            <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground">FP Cost</span>
                                            <span className="text-[11px] font-black text-foreground">{detailItem.spell.FPCost}</span>
                                        </div>
                                        <div className="flex items-center justify-between bg-muted/20 rounded px-3 py-1.5 border border-border/30">
                                            <span className="text-[9px] font-bold uppercase tracking-wider text-muted-foreground">Slots</span>
                                            <span className="text-[11px] font-black text-foreground">{detailItem.spell.Slots}</span>
                                        </div>
                                    </div>

                                    {(detailItem.spell.ReqInt > 0 || detailItem.spell.ReqFai > 0 || detailItem.spell.ReqArc > 0) && (
                                        <>
                                            <h4 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground pt-2">Requirements</h4>
                                            <div className="flex gap-2 flex-wrap">
                                                {[
                                                    ['INT', detailItem.spell.ReqInt],
                                                    ['FAI', detailItem.spell.ReqFai],
                                                    ['ARC', detailItem.spell.ReqArc],
                                                ].filter(([, v]) => v > 0).map(([label, val]) => (
                                                    <div key={label as string} className="flex flex-col items-center bg-muted/20 rounded px-3 py-1.5 border border-border/30 min-w-[50px]">
                                                        <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">{label}</span>
                                                        <span className="text-[11px] font-black text-foreground">{val}</span>
                                                    </div>
                                                ))}
                                            </div>
                                        </>
                                    )}
                                </div>
                            )}

                            {/* Item info */}
                            <div className="space-y-2 pt-2 border-t border-border/30">
                                <h4 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Item Info</h4>
                                <div className="grid grid-cols-2 gap-2 text-[10px]">
                                    <div className="flex justify-between bg-muted/10 rounded px-2 py-1">
                                        <span className="text-muted-foreground font-bold">Max Inventory</span>
                                        <span className="font-black text-foreground">{detailItem.maxInventory}</span>
                                    </div>
                                    <div className="flex justify-between bg-muted/10 rounded px-2 py-1">
                                        <span className="text-muted-foreground font-bold">Max Storage</span>
                                        <span className="font-black text-foreground">{detailItem.maxStorage}</span>
                                    </div>
                                    {detailItem.maxUpgrade > 0 && (
                                        <div className="flex justify-between bg-muted/10 rounded px-2 py-1">
                                            <span className="text-muted-foreground font-bold">Max Upgrade</span>
                                            <span className="font-black text-foreground">+{detailItem.maxUpgrade}</span>
                                        </div>
                                    )}
                                </div>
                            </div>

                            {/* No data fallback */}
                            {!detailItem.description && !detailItem.weapon && !detailItem.armor && !detailItem.spell && (
                                <p className="text-[10px] text-muted-foreground/60 italic">No description or stats available for this item.</p>
                            )}
                        </div>
                    </div>
                </div>
            )}

            {/* Icon Preview Modal */}
            {selectedIcon && (
                <div
                    className="fixed inset-0 bg-background/80 backdrop-blur-xl z-[100] flex items-center justify-center p-8 animate-in fade-in duration-300"
                    onClick={() => setSelectedIcon(null)}
                >
                    <div className="relative max-w-2xl w-full flex flex-col items-center space-y-8 animate-in zoom-in-95 duration-300">
                        <div className="w-64 h-64 bg-muted/20 rounded-3xl border border-border/50 flex items-center justify-center shadow-2xl shadow-primary/10 relative group">
                            <div className="absolute inset-0 bg-primary/5 rounded-3xl blur-3xl group-hover:bg-primary/10 transition-all duration-500" />
                            {brokenIcons.has(selectedIcon.path) ? (
                                <span className="text-3xl font-black text-muted-foreground/30 select-none">?</span>
                            ) : (
                                <img src={selectedIcon.path} alt={selectedIcon.name} className="w-48 h-48 object-contain drop-shadow-2xl relative z-10" onError={() => handleImageError(selectedIcon.path)} />
                            )}
                        </div>
                        <div className="text-center space-y-2">
                            <h3 className="text-2xl font-black uppercase tracking-[0.2em] text-foreground">{selectedIcon.name}</h3>
                            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-[0.3em]">{selectedIcon.path}</p>
                        </div>
                        <button className="px-8 py-3 bg-primary text-primary-foreground rounded-full text-[10px] font-black uppercase tracking-[0.2em] shadow-xl shadow-primary/20 hover:scale-105 active:scale-95 transition-all">
                            Close Preview
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
