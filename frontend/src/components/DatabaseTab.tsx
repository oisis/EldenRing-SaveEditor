import {useEffect, useState} from 'react';
import {GetItemList, GetInfuseTypes, AddItemsToCharacter} from '../../wailsjs/go/main/App';
import {db} from '../../wailsjs/go/models';
import type {AddSettings} from '../App';
import {CategorySelect} from './CategorySelect';

interface DatabaseTabProps {
    columnVisibility: {
        id: boolean;
        category: boolean;
    };
    platform: string | null;
    charIndex: number;
    onItemsAdded?: () => void;
    addSettings: AddSettings;
    showFlaggedItems: boolean;
    category: string;
    setCategory: (value: string) => void;
}

// Determine if ALL selected items are non-stackable (max qty == 1)
function allNonStackable(items: db.ItemEntry[]): boolean {
    return items.every(i => i.maxInventory <= 1);
}

export function DatabaseTab({columnVisibility, platform, charIndex, onItemsAdded, addSettings, showFlaggedItems, category, setCategory}: DatabaseTabProps) {
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
    const [isSaving, setIsSaving] = useState(false);

    // Quantity state for modal
    const [addToInv, setAddToInv] = useState(true);
    const [invMax, setInvMax] = useState(false);
    const [invQtyVal, setInvQtyVal] = useState(1);
    const [addToStorage, setAddToStorage] = useState(false);
    const [storageMax, setStorageMax] = useState(false);
    const [storageQtyVal, setStorageQtyVal] = useState(1);

    // Icon preview
    const [selectedIcon, setSelectedIcon] = useState<{name: string, path: string} | null>(null);

    useEffect(() => {
        GetInfuseTypes().then(res => setInfuseTypes(res || []));
    }, []);

    useEffect(() => {
        setLoading(true);
        GetItemList(category).then(res => {
            setDbItems(res || []);
            setLoading(false);
        }).catch(() => setLoading(false));
    }, [category]);

    const filteredItems = dbItems.filter(item => {
        if (!showFlaggedItems && item.flags?.length > 0) return false;
        const matchesSearch = item.name.toLowerCase().includes(search.toLowerCase()) ||
            item.id.toString(16).includes(search.toLowerCase());
        if (category === 'all') return matchesSearch;
        return item.category === category && matchesSearch;
    }).sort((a, b) => {
        const aVal = a[sortCol as keyof db.ItemEntry];
        const bVal = b[sortCol as keyof db.ItemEntry];
        if (aVal < bVal) return sortDir === 'asc' ? -1 : 1;
        if (aVal > bVal) return sortDir === 'asc' ? 1 : -1;
        return 0;
    });

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
                // Non-stackable: separate calls for inv and storage (different copy counts).
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
            alert('Failed to add items: ' + err);
        } finally {
            setIsSaving(false);
        }
    };

    const openModal = (items: db.ItemEntry[]) => {
        setAddToInv(true);
        setInvMax(false);
        setInvQtyVal(1);
        setAddToStorage(true);
        setStorageMax(false);
        setStorageQtyVal(1);
        setConfirmModal(items);
    };

    const handleImageError = (e: React.SyntheticEvent<HTMLImageElement>) => {
        const target = e.currentTarget;
        target.style.display = 'none';
        const parent = target.parentElement;
        if (parent) {
            const ph = document.createElement('div');
            ph.className = 'text-[10px] font-black text-muted-foreground/30 select-none';
            ph.innerText = '?';
            parent.appendChild(ph);
        }
    };

    const selectedInfuseName = infuseTypes.find(t => t.offset === infuseOffset)?.name ?? 'Standard';

    // Whether the modal items are all non-stackable (weapons/armor/talismans)
    const modalNonStackable = confirmModal ? allNonStackable(confirmModal) : true;
    const modalMaxInv = confirmModal ? Math.min(...confirmModal.map(i => i.maxInventory)) : 1;
    const modalMaxStorage = confirmModal ? Math.min(...confirmModal.map(i => i.maxStorage)) : 1;
    const modalMixedMaxes = confirmModal && confirmModal.length > 1 && !modalNonStackable &&
        (new Set(confirmModal.map(i => i.maxInventory)).size > 1 || new Set(confirmModal.map(i => i.maxStorage)).size > 1);

    return (
        <div className="flex-1 flex flex-col min-h-0 space-y-3">
            {/* Confirm Modal */}
            {confirmModal && (
                <div className="fixed inset-0 z-[110] flex items-center justify-center bg-background/80 backdrop-blur-sm animate-in fade-in duration-300">
                    <div className="bg-card p-8 rounded-2xl border border-primary/20 flex flex-col space-y-6 max-w-sm w-full mx-4 shadow-2xl shadow-primary/20 animate-in zoom-in-95 duration-300">
                        {/* Header */}
                        <div className="flex items-center space-x-4">
                            {confirmModal.length === 1 ? (
                                <>
                                    <div className="w-12 h-12 rounded bg-muted/30 border border-border/50 flex items-center justify-center overflow-hidden">
                                        <img src={confirmModal[0].iconPath} alt="" className="w-8 h-8 object-contain" onError={handleImageError} />
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

                    {selectedDbItems.size > 0 && (
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

                <div className="flex-1 overflow-y-auto custom-scrollbar">
                    <table className="w-full text-left border-collapse">
                        <thead className="sticky top-0 z-20 bg-muted/80 backdrop-blur-md border-b border-border shadow-sm">
                            <tr className="text-[9px] font-black uppercase tracking-[0.15em] text-muted-foreground">
                                <th className="p-4 w-10">
                                    <div
                                        onClick={toggleAll}
                                        className={`w-4 h-4 rounded border flex items-center justify-center transition-all cursor-pointer ${selectedDbItems.size === filteredItems.length && filteredItems.length > 0 ? 'bg-primary border-primary' : 'bg-muted/30 border-border hover:border-primary/50'}`}
                                    >
                                        {selectedDbItems.size === filteredItems.length && filteredItems.length > 0 &&
                                            <svg className="w-3 h-3 text-primary-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                    </div>
                                </th>
                                <th className="p-4 w-12">Icon</th>
                                <th className="p-4 cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('name')}>
                                    Name {sortCol === 'name' && (sortDir === 'asc' ? '↑' : '↓')}
                                </th>
                                {columnVisibility.id && (
                                    <th className="p-4 cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('id')}>
                                        ID {sortCol === 'id' && (sortDir === 'asc' ? '↑' : '↓')}
                                    </th>
                                )}
                                {columnVisibility.category && (
                                    <th className="p-4 cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('category')}>
                                        Category {sortCol === 'category' && (sortDir === 'asc' ? '↑' : '↓')}
                                    </th>
                                )}
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border/30">
                            {filteredItems.map(item => {
                                const isUpgradeable = item.maxUpgrade > 0;
                                const isAsh = item.category === 'ashes';
                                const hasInfuse = item.maxUpgrade === 25 && infuseOffset !== 0;
                                const levelVal = isAsh ? upgradeAsh : (item.maxUpgrade === 25 ? upgrade25 : item.maxUpgrade === 10 ? upgrade10 : 0);
                                const showPreview = isUpgradeable && (levelVal > 0 || hasInfuse);
                                const previewParts: string[] = [];
                                if (hasInfuse) previewParts.push(selectedInfuseName);
                                if (levelVal > 0) previewParts.push(`+${levelVal}`);

                                return (
                                    <tr key={item.id} className={`group hover:bg-primary/[0.03] transition-colors ${selectedDbItems.has(item.id) ? 'bg-primary/[0.02]' : ''}`}>
                                        <td className="p-4">
                                            <div
                                                onClick={() => toggleItem(item.id)}
                                                className={`w-4 h-4 rounded border flex items-center justify-center transition-all cursor-pointer ${selectedDbItems.has(item.id) ? 'bg-primary border-primary' : 'bg-muted/30 border-border group-hover:border-primary/50'}`}
                                            >
                                                {selectedDbItems.has(item.id) &&
                                                    <svg className="w-3 h-3 text-primary-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                            </div>
                                        </td>
                                        <td className="px-4 py-0.5">
                                            <div
                                                className="w-12 h-12 bg-muted/20 rounded-lg border border-border/50 flex items-center justify-center overflow-hidden group-hover:border-primary/30 transition-all cursor-zoom-in"
                                                onClick={() => setSelectedIcon({name: item.name, path: item.iconPath})}
                                            >
                                                <img
                                                    src={item.iconPath}
                                                    alt={item.name}
                                                    className="w-full h-full p-0.5 object-contain drop-shadow-md group-hover:scale-110 transition-transform duration-300"
                                                    onError={handleImageError}
                                                />
                                            </div>
                                        </td>
                                        <td className="p-4">
                                            <div className="flex flex-col gap-0.5">
                                                <div className="flex items-center gap-1.5 flex-wrap">
                                                    <span className="text-[13px] font-semibold text-foreground group-hover:text-primary transition-colors">{item.name}</span>
                                                    {item.flags?.includes('cut_content') && (
                                                        <span className="text-[8px] font-black uppercase tracking-widest px-1.5 py-0.5 rounded bg-amber-500/15 text-amber-400 border border-amber-500/30">CUT</span>
                                                    )}
                                                    {item.flags?.includes('ban_risk') && (
                                                        <span className="text-[8px] font-black uppercase tracking-widest px-1.5 py-0.5 rounded bg-red-500/15 text-red-400 border border-red-500/30">⚠ BAN</span>
                                                    )}
                                                </div>
                                                {showPreview ? (
                                                    <span className="text-[8px] font-mono font-bold text-primary/60 uppercase tracking-tight">
                                                        {previewParts.join(' ')}
                                                    </span>
                                                ) : (
                                                    <span className="text-[8px] font-mono text-muted-foreground/50 uppercase tracking-tighter">
                                                        0x{item.id.toString(16).toUpperCase()}
                                                    </span>
                                                )}
                                            </div>
                                        </td>
                                        {columnVisibility.id && (
                                            <td className="p-4 text-[10px] font-mono text-muted-foreground">0x{item.id.toString(16).toUpperCase()}</td>
                                        )}
                                        {columnVisibility.category && (
                                            <td className="p-4">
                                                <span className="text-[8px] font-black uppercase tracking-widest px-2 py-1 bg-muted/30 rounded-md text-muted-foreground border border-border/20">
                                                    {item.category === 'arrows_and_bolts' ? 'Arrows & Bolts' : item.category.replace(/_/g, ' ')}
                                                </span>
                                            </td>
                                        )}
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Icon Preview Modal */}
            {selectedIcon && (
                <div
                    className="fixed inset-0 bg-background/80 backdrop-blur-xl z-[100] flex items-center justify-center p-8 animate-in fade-in duration-300"
                    onClick={() => setSelectedIcon(null)}
                >
                    <div className="relative max-w-2xl w-full flex flex-col items-center space-y-8 animate-in zoom-in-95 duration-300">
                        <div className="w-64 h-64 bg-muted/20 rounded-3xl border border-border/50 flex items-center justify-center shadow-2xl shadow-primary/10 relative group">
                            <div className="absolute inset-0 bg-primary/5 rounded-3xl blur-3xl group-hover:bg-primary/10 transition-all duration-500" />
                            <img src={selectedIcon.path} alt={selectedIcon.name} className="w-48 h-48 object-contain drop-shadow-2xl relative z-10" onError={handleImageError} />
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
