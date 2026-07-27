import {useEffect, useState, useMemo, useRef, useDeferredValue} from 'react';
import toast from '../lib/toast';
import {useVirtualizer} from '@tanstack/react-virtual';
import {GetCharacter, SaveCharacter, RemoveItemsFromCharacter, GetItemDetail} from '../../wailsjs/go/main/App';
import {vm, db, editor} from '../../wailsjs/go/models';
import {CategorySelect} from './CategorySelect';
import {RiskBadge} from './RiskBadge';
import {useFavorites} from '../state/favorites';
import {ItemDetailPanel} from './ItemDetailPanel';
import {ItemCapacityBadge} from './ItemCapacityBadge';
import {useInventoryWorkspace, type ContainerKind} from '../hooks/useInventoryWorkspace';
import {WeaponEditModal} from './WeaponEditModal';
import {adaptForWeaponModal} from './weaponPatch';
import {loadSafetyProfile, usesTechnicalCaps, SAFETY_PROFILE_EVENT, type SafetyProfile} from '../state/safetyProfile';

// Categories with sub-groupings — drives the Sub-Category column visibility.
const CATEGORIES_WITH_SUBGROUPS = new Set([
    'tools', 'bolstering_materials', 'key_items',
    'melee_armaments', 'ranged_and_catalysts', 'arrows_and_bolts', 'shields', 'info',
]);

// Display labels for main categories (used as fallback when 'all' is selected).
const CATEGORY_LABEL: Record<string, string> = {
    tools: 'Tools', ashes: 'Ashes', crafting_materials: 'Crafting Materials',
    bolstering_materials: 'Bolstering Materials', key_items: 'Key Items',
    sorceries: 'Sorceries', incantations: 'Incantations', ashes_of_war: 'Ashes of War',
    melee_armaments: 'Melee Armaments', ranged_and_catalysts: 'Ranged Weapons / Catalysts',
    arrows_and_bolts: 'Arrows / Bolts', shields: 'Shields',
    head: 'Head', chest: 'Chest', arms: 'Arms', legs: 'Legs',
    talismans: 'Talismans', info: 'Info',
};

interface InventoryTabProps {
    charIndex: number;
    inventoryVersion: number;
    columnVisibility: {
        id: boolean;
        category: boolean;
    };
    showFlaggedItems: boolean;
    category: string;
    setCategory: (value: string) => void;
    onMutate?: () => void;
    showOnlyFavorites?: boolean;
    onToggleFavorites?: () => void;
}

export function InventoryTab({ charIndex, inventoryVersion, columnVisibility, showFlaggedItems, category, setCategory, onMutate, showOnlyFavorites = false, onToggleFavorites }: InventoryTabProps) {
    const {isFav, toggle: toggleFav} = useFavorites();
    const [search, setSearch] = useState('');
    const [subCategory, setSubCategory] = useState('all');
    const [viewMode, setViewMode] = useState<'table' | 'grid'>('table');
    const [charInventory, setCharInventory] = useState<vm.ItemViewModel[]>([]);
    const [charStorage, setCharStorage] = useState<vm.ItemViewModel[]>([]);
    const [charAttachedItems, setCharAttachedItems] = useState<vm.ItemViewModel[]>([]);
    const [loading, setLoading] = useState(false);
    const workspace = useInventoryWorkspace();
    const [weaponEditor, setWeaponEditor] = useState<{uid: string; source: ContainerKind} | null>(null);
    const [safetyProfile, setSafetyProfile] = useState<SafetyProfile>(() => loadSafetyProfile());
    const useTechnicalCapsMode = usesTechnicalCaps(safetyProfile);
    
    // Sorting state
    const [sortCol, setSortCol] = useState<string>('name');
    const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');

    const [selectedIcon, setSelectedIcon] = useState<{name: string, path: string} | null>(null);
    const [isSaving, setIsSaving] = useState(false);

    // Local state for edited quantities
    const [editedInv, setEditedInv] = useState<Record<number, number>>({});
    const [editedStorage, setEditedStorage] = useState<Record<number, number>>({});

    // Selection + remove
    const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
    const [removeModal, setRemoveModal] = useState<{handles: number[], names: string[]} | null>(null);
    const [isRemoving, setIsRemoving] = useState(false);
    const [brokenIcons, setBrokenIcons] = useState<Set<string>>(new Set());
    const [detailItem, setDetailItem] = useState<db.ItemEntry | null>(null);
    const [hoverTooltip, setHoverTooltip] = useState<{name: string, path: string, x: number, y: number} | null>(null);

    useEffect(() => {
        const handler = (event: Event) => setSafetyProfile((event as CustomEvent<SafetyProfile>).detail);
        window.addEventListener(SAFETY_PROFILE_EVENT, handler);
        return () => window.removeEventListener(SAFETY_PROFILE_EVENT, handler);
    }, []);

    useEffect(() => {
        workspace.start(charIndex).catch(() => { /* surfaced through workspace.lastError */ });
        setWeaponEditor(null);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [charIndex, inventoryVersion]);

    const toggleSelect = (key: string) => {
        setSelectedKeys(prev => {
            const next = new Set(prev);
            if (next.has(key)) next.delete(key); else next.add(key);
            return next;
        });
    };

    const toggleSelectAll = () => {
        const selectable = filteredOwnedItems.filter(item => !item.readOnly);
        if (selectedKeys.size === selectable.length && selectable.length > 0) {
            setSelectedKeys(new Set());
        } else {
            setSelectedKeys(new Set(selectable.map(i => rowKey(i))));
        }
    };

    const handleRemoveSelected = () => {
        const items = filteredOwnedItems.filter(i => !i.readOnly && selectedKeys.has(rowKey(i)));
        const handles = items.flatMap(i => [i.invHandle, i.storageHandle].filter(h => h !== 0));
        setRemoveModal({ handles, names: items.map(i => i.name) });
    };

    const confirmRemove = async (fromInventory: boolean, fromStorage: boolean) => {
        if (!removeModal || isRemoving) return;
        setIsRemoving(true);
        try {
            await RemoveItemsFromCharacter(charIndex, removeModal.handles, fromInventory, fromStorage);
            setSelectedKeys(new Set());
            setRemoveModal(null);
            // Reload inventory
            const char = await GetCharacter(charIndex);
            if (char) {
                setCharInventory(char.inventory || []);
                setCharStorage(char.storage || []);
                setCharAttachedItems(char.attachedItems || []);
            }
            onMutate?.();
        } catch (err) {
            toast.error('Remove failed: ' + err);
        } finally {
            setIsRemoving(false);
        }
    };

    const handleQtyChange = (handle: number, value: string, type: 'inv' | 'storage', max: number) => {
        const qty = parseInt(value) || 0;
        const safeQty = Math.min(Math.max(0, qty), max);
        
        if (type === 'inv') {
            setEditedInv(prev => ({ ...prev, [handle]: safeQty }));
        } else {
            setEditedStorage(prev => ({ ...prev, [handle]: safeQty }));
        }
    };

    const saveChanges = async () => {
        if (isSaving || workspace.saving) return;
        setIsSaving(true);
        try {
            // Persist the weapon workspace first. Quantity edits are then applied
            // to a fresh VM so SaveCharacter cannot overwrite the AoW/infusion
            // state with a snapshot captured before the workspace save.
            if (workspace.dirty) {
                const saved = await workspace.save();
                if (!saved) {
                    throw new Error(workspace.lastError || 'Weapon changes could not be saved');
                }
            }

            const hasQuantityEdits = Object.keys(editedInv).length > 0 || Object.keys(editedStorage).length > 0;
            if (!hasQuantityEdits) {
                const updatedChar = await GetCharacter(charIndex);
                setCharInventory(updatedChar?.inventory || []);
                setCharStorage(updatedChar?.storage || []);
                setCharAttachedItems(updatedChar?.attachedItems || []);
                setWeaponEditor(null);
                onMutate?.();
                return;
            }

            const char = await GetCharacter(charIndex);
            if (!char) return;
            char.useTechnicalItemCaps = useTechnicalCapsMode;

            // Apply local edits to the character VM
            char.inventory = char.inventory.map(item => ({
                ...item,
                quantity: editedInv[item.handle] !== undefined ? editedInv[item.handle] : item.quantity
            }));
            char.storage = char.storage.map(item => ({
                ...item,
                quantity: editedStorage[item.handle] !== undefined ? editedStorage[item.handle] : item.quantity
            }));

            await SaveCharacter(charIndex, char);

            // Refresh data
            const updatedChar = await GetCharacter(charIndex);
            setCharInventory(updatedChar?.inventory || []);
            setCharStorage(updatedChar?.storage || []);
            setCharAttachedItems(updatedChar?.attachedItems || []);
            setEditedInv({});
            setEditedStorage({});
            setWeaponEditor(null);
            onMutate?.();
            
        } catch (err) {
            toast.error('Save failed: ' + err);
        } finally {
            setIsSaving(false);
        }
    };

    type MergedItem = {
        id: number; baseId: number; name: string; category: string; subCategory: string; subGroup: string;
        separateInstances: boolean; inInventory: boolean; inStorage: boolean;
        invHandle: number; storageHandle: number; attachedHandle: number;
        invQty: number; storageQty: number;
        maxInv: number; maxStorage: number; maxUpgrade: number; currentUpgrade: number; iconPath: string;
        flags: string[];
        readOnly: boolean;
        isEquippedAoW: boolean;
        equippedByWeaponHandle: number;
        equippedByWeaponName: string;
        aowHandleShared: boolean;
        uid: string;
    };

    // uid is a per-record unique React key. Talismans store N copies as N separate
    // records sharing one id-derived handle, so handle alone is NOT unique — a
    // collision breaks React reconciliation (sort/search/order glitch). uid is
    // assigned at build time with an occurrence counter to disambiguate copies.
    const rowKey = (item: MergedItem) => item.uid;

    // Build item list for display.
    // Instance-backed items: separate row for each physical record in inventory and storage.
    //   Each copy can have different upgrade/infuse/AoW, so merging by handle is wrong.
    // Quantity stacks: merged by item ID — one row with both inv and storage qty.
    const mergedOwnedItems = useMemo(() => {
        const separateInstanceList: MergedItem[] = [];
        const stackableMap = new Map<number, MergedItem>();
        const recordMode = (item: vm.ItemViewModel) =>
            item.recordMode || (item.category === 'Item' || item.subCategory === 'arrows_and_bolts'
                ? 'quantity_stack'
                : 'separate_instances');
        const cap = (item: vm.ItemViewModel, kind: 'inv' | 'storage') => {
            if (useTechnicalCapsMode) {
                if (kind === 'inv' && item.gameMaxInventoryKnown) return item.gameMaxInventory;
                if (kind === 'storage' && item.gameMaxStorageKnown) return item.gameMaxStorage;
            }
            return kind === 'inv' ? item.maxInventory : item.maxStorage;
        };

        charInventory.forEach(item => {
            if (recordMode(item) === 'separate_instances') {
                separateInstanceList.push({
                    id: item.id, baseId: item.baseId || item.id, name: item.name,
                    category: item.category, subCategory: item.subCategory, subGroup: item.subGroup ?? '',
                    separateInstances: true, inInventory: true, inStorage: false,
                    invHandle: item.handle, storageHandle: 0, attachedHandle: 0,
                    invQty: 1, storageQty: 0,
                    maxInv: cap(item, 'inv'), maxStorage: cap(item, 'storage'),
                    maxUpgrade: item.maxUpgrade, currentUpgrade: item.currentUpgrade ?? 0, iconPath: item.iconPath,
                    flags: item.flags ?? [],
                    readOnly: item.readOnly ?? false,
                    isEquippedAoW: item.isEquippedAoW ?? false,
                    equippedByWeaponHandle: item.equippedByWeaponHandle ?? 0,
                    equippedByWeaponName: item.equippedByWeaponName ?? '',
                    aowHandleShared: item.aowHandleShared ?? false,
                    uid: '',
                });
            } else {
                stackableMap.set(item.id, {
                    id: item.id, baseId: item.baseId || item.id, name: item.name,
                    category: item.category, subCategory: item.subCategory, subGroup: item.subGroup ?? '',
                    separateInstances: false, inInventory: true, inStorage: false,
                    invHandle: item.handle, storageHandle: 0, attachedHandle: 0,
                    invQty: item.quantity, storageQty: 0,
                    maxInv: cap(item, 'inv'), maxStorage: cap(item, 'storage'),
                    maxUpgrade: item.maxUpgrade, currentUpgrade: item.currentUpgrade ?? 0, iconPath: item.iconPath,
                    flags: item.flags ?? [],
                    readOnly: item.readOnly ?? false,
                    isEquippedAoW: false,
                    equippedByWeaponHandle: 0,
                    equippedByWeaponName: '',
                    aowHandleShared: false,
                    uid: '',
                });
            }
        });

        charStorage.forEach(item => {
            if (recordMode(item) === 'separate_instances') {
                separateInstanceList.push({
                    id: item.id, baseId: item.baseId || item.id, name: item.name,
                    category: item.category, subCategory: item.subCategory, subGroup: item.subGroup ?? '',
                    separateInstances: true, inInventory: false, inStorage: true,
                    invHandle: 0, storageHandle: item.handle, attachedHandle: 0,
                    invQty: 0, storageQty: 1,
                    maxInv: cap(item, 'inv'), maxStorage: cap(item, 'storage'),
                    maxUpgrade: item.maxUpgrade, currentUpgrade: item.currentUpgrade ?? 0, iconPath: item.iconPath,
                    flags: item.flags ?? [],
                    readOnly: item.readOnly ?? false,
                    isEquippedAoW: item.isEquippedAoW ?? false,
                    equippedByWeaponHandle: item.equippedByWeaponHandle ?? 0,
                    equippedByWeaponName: item.equippedByWeaponName ?? '',
                    aowHandleShared: item.aowHandleShared ?? false,
                    uid: '',
                });
            } else {
                const existing = stackableMap.get(item.id);
                if (existing) {
                    existing.inStorage = true;
                    existing.storageHandle = item.handle;
                    existing.storageQty = item.quantity;
                } else {
                    stackableMap.set(item.id, {
                        id: item.id, baseId: item.baseId || item.id, name: item.name,
                        category: item.category, subCategory: item.subCategory, subGroup: item.subGroup ?? '',
                        separateInstances: false, inInventory: false, inStorage: true,
                        invHandle: 0, storageHandle: item.handle, attachedHandle: 0,
                        invQty: 0, storageQty: item.quantity,
                        maxInv: cap(item, 'inv'), maxStorage: cap(item, 'storage'),
                        maxUpgrade: item.maxUpgrade, currentUpgrade: item.currentUpgrade ?? 0, iconPath: item.iconPath,
                        flags: item.flags ?? [],
                        readOnly: item.readOnly ?? false,
                        isEquippedAoW: false,
                        equippedByWeaponHandle: 0,
                        equippedByWeaponName: '',
                        aowHandleShared: false,
                        uid: '',
                    });
                }
            }
        });

        charAttachedItems.forEach(item => {
            separateInstanceList.push({
                id: item.id, baseId: item.baseId || item.id, name: item.name,
                category: item.category, subCategory: item.subCategory, subGroup: item.subGroup ?? '',
                separateInstances: true, inInventory: false, inStorage: false,
                invHandle: 0, storageHandle: 0, attachedHandle: item.handle,
                invQty: 0, storageQty: 0,
                maxInv: cap(item, 'inv'), maxStorage: cap(item, 'storage'),
                maxUpgrade: item.maxUpgrade, currentUpgrade: item.currentUpgrade ?? 0, iconPath: item.iconPath,
                flags: item.flags ?? [],
                readOnly: true,
                isEquippedAoW: item.isEquippedAoW ?? true,
                equippedByWeaponHandle: item.equippedByWeaponHandle ?? 0,
                equippedByWeaponName: item.equippedByWeaponName ?? '',
                aowHandleShared: item.aowHandleShared ?? false,
                uid: '',
            });
        });

        const all = [...separateInstanceList, ...Array.from(stackableMap.values())];
        const seen = new Map<string, number>();
        for (const it of all) {
            const base = it.separateInstances
                ? `h-${it.inInventory ? 'i' : it.inStorage ? 's' : 'a'}-${it.invHandle || it.storageHandle || it.attachedHandle}`
                : `s-${it.id}`;
            const n = seen.get(base) ?? 0;
            seen.set(base, n + 1);
            it.uid = n === 0 ? base : `${base}#${n}`;
        }
        return all;
    }, [charInventory, charStorage, charAttachedItems, useTechnicalCapsMode]);

    const handleImageError = (iconPath: string) => {
        setBrokenIcons(prev => new Set(prev).add(iconPath));
    };

    const handleItemClick = async (item: MergedItem) => {
        try {
            const detail = await GetItemDetail(item.baseId);
            if (detail) setDetailItem(detail);
        } catch (err) {
            console.error('Failed to load item detail:', err);
        }
    };

    useEffect(() => {
        setLoading(true);
        GetCharacter(charIndex).then(res => {
            setCharInventory(res?.inventory || []);
            setCharStorage(res?.storage || []);
            setCharAttachedItems(res?.attachedItems || []);
        }).finally(() => setLoading(false));
    }, [charIndex, inventoryVersion]);

    const handleSort = (col: string) => {
        if (sortCol === col) {
            setSortDir(sortDir === 'asc' ? 'desc' : 'asc');
        } else {
            setSortCol(col);
            setSortDir('asc');
        }
    };

    const sortItems = (items: MergedItem[]) => {
        return [...items].sort((a, b) => {
            let valA: string | number = 0;
            let valB: string | number = 0;

            if (sortCol === 'maxUpgrade') {
                valA = a.maxUpgrade || 0;
                valB = b.maxUpgrade || 0;
            } else if (sortCol === 'currentUpgrade') {
                valA = a.currentUpgrade || 0;
                valB = b.currentUpgrade || 0;
            } else if (sortCol === 'invOwned') {
                // Numeric ownership shown in the Inventory cell: non-stackable = 0/1
                // presence, stackable = base quantity (edits don't re-sort mid-typing).
                valA = a.separateInstances ? (a.inInventory ? 1 : 0) : a.invQty;
                valB = b.separateInstances ? (b.inInventory ? 1 : 0) : b.invQty;
            } else if (sortCol === 'storageOwned') {
                valA = a.separateInstances ? (a.inStorage ? 1 : 0) : a.storageQty;
                valB = b.separateInstances ? (b.inStorage ? 1 : 0) : b.storageQty;
            } else {
                const rawA = a[sortCol as keyof MergedItem];
                const rawB = b[sortCol as keyof MergedItem];
                if (typeof rawA === 'string' && typeof rawB === 'string') {
                    valA = rawA.toLowerCase();
                    valB = rawB.toLowerCase();
                } else if (typeof rawA === 'number' && typeof rawB === 'number') {
                    valA = rawA;
                    valB = rawB;
                }
            }

            if (valA < valB) return sortDir === 'asc' ? -1 : 1;
            if (valA > valB) return sortDir === 'asc' ? 1 : -1;
            return 0;
        });
    };

    const deferredSearch = useDeferredValue(search);

    // Reset the subcategory filter whenever the category changes.
    useEffect(() => { setSubCategory('all'); }, [category]);

    // Subcategories (item.subGroup) derived from the owned items in the active
    // category scope — never a hardcoded list. Empty values are dropped.
    const availableSubCategories = useMemo(() => {
        const scope = category === 'all' ? mergedOwnedItems : mergedOwnedItems.filter(i => i.subCategory === category);
        return Array.from(new Set(scope.map(i => i.subGroup).filter(Boolean))).sort();
    }, [mergedOwnedItems, category]);
    const hasSubCategories = availableSubCategories.length > 0;

    const filteredOwnedItems = useMemo(() => sortItems(mergedOwnedItems.filter(item => {
        if (showOnlyFavorites && !isFav(item.id)) return false;
        // "Cut & Ban-Risk" toggle hides only risky-flagged items, not informational flags
        // (dlc, stackable) which are now present on most entries.
        const RISKY_FLAGS = ['cut_content', 'ban_risk', 'pre_order', 'dlc_duplicate'];
        if (!showFlaggedItems && item.flags?.some(f => RISKY_FLAGS.includes(f))) return false;
        // Category (item.subCategory) then subcategory (item.subGroup) scope, before search.
        if (category !== 'all' && item.subCategory !== category) return false;
        if (subCategory !== 'all' && item.subGroup !== subCategory) return false;
        const q = deferredSearch.toLowerCase();
        return !q ||
            item.name.toLowerCase().includes(q) ||
            item.id.toString(16).toLowerCase().includes(q);
    })), [mergedOwnedItems, deferredSearch, category, subCategory, sortCol, sortDir, showFlaggedItems, showOnlyFavorites, isFav]);
    const selectableOwnedCount = useMemo(() => filteredOwnedItems.filter(item => !item.readOnly).length, [filteredOwnedItems]);

    // Owned count scoped to the selected category/subcategory — derived from
    // mergedOwnedItems (not filtered) so text search never affects it.
    const ownedCount = useMemo(() =>
        mergedOwnedItems.filter(i => {
            if (category !== 'all' && i.subCategory !== category) return false;
            if (subCategory !== 'all' && i.subGroup !== subCategory) return false;
            return true;
        }).length,
    [mergedOwnedItems, category, subCategory]);

    const showSubGroupColumn = category === 'all' || CATEGORIES_WITH_SUBGROUPS.has(category);

    // Combined upgrade/max-upgrade column — hidden when the whole current table
    // has no upgradeable rows (avoids empty columns on consumables/armor).
    const showUpgradeColumn = useMemo(() => filteredOwnedItems.some(i => i.maxUpgrade > 0), [filteredOwnedItems]);
    const showWeaponColumns = useMemo(() => filteredOwnedItems.some(i => i.category === 'Weapon'), [filteredOwnedItems]);

    const workspaceItemsByRecord = useMemo(() => {
        const result = new Map<string, editor.EditableItem>();
        for (const item of workspace.inventoryItems) {
            if (item.isWeapon) result.set(`inventory:${item.originalHandle}`, item);
        }
        for (const item of workspace.storageItems) {
            if (item.isWeapon) result.set(`storage:${item.originalHandle}`, item);
        }
        return result;
    }, [workspace.inventoryItems, workspace.storageItems]);

    const getWorkspaceWeapon = (item: MergedItem): {item: editor.EditableItem; source: ContainerKind} | null => {
        if (item.category !== 'Weapon') return null;
        const source: ContainerKind = item.inInventory ? 'inventory' : 'storage';
        const handle = item.inInventory ? item.invHandle : item.storageHandle;
        const workspaceItem = workspaceItemsByRecord.get(`${source}:${handle}`);
        return workspaceItem ? {item: workspaceItem, source} : null;
    };

    const activeWeaponEditor = useMemo(() => {
        if (!weaponEditor) return null;
        const pool = weaponEditor.source === 'inventory' ? workspace.inventoryItems : workspace.storageItems;
        const item = pool.find(candidate => candidate.uid === weaponEditor.uid);
        return item ? {item, source: weaponEditor.source} : null;
    }, [weaponEditor, workspace.inventoryItems, workspace.storageItems]);

    const tableColumnCount = 6 +
        (columnVisibility.id ? 1 : 0) +
        (columnVisibility.category && showSubGroupColumn ? 1 : 0) +
        (showUpgradeColumn ? 1 : 0) +
        (showWeaponColumns ? 2 : 0);

    const SortIndicator = ({ col }: { col: string }) => {
        if (sortCol !== col) return <span className="ml-1 opacity-20">↕</span>;
        return <span className="ml-1 text-primary">{sortDir === 'asc' ? '↑' : '↓'}</span>;
    };

    const scrollRef = useRef<HTMLDivElement>(null);
    const rowVirtualizer = useVirtualizer({
        count: filteredOwnedItems.length,
        getScrollElement: () => scrollRef.current,
        estimateSize: () => 52,
        overscan: 20,
    });

    return (
        <div className="flex-1 flex flex-col min-h-0 space-y-3 animate-in fade-in slide-in-from-bottom-4 duration-700">
            {activeWeaponEditor && (
                <WeaponEditModal
                    charIndex={charIndex}
                    item={adaptForWeaponModal(activeWeaponEditor.item)}
                    source={activeWeaponEditor.source}
                    onClose={() => setWeaponEditor(null)}
                    workspace={{
                        sessionID: workspace.sessionID,
                        updateWeapon: (uid, patch) => workspace.updateWeapon(uid, patch),
                    }}
                    workspaceItem={activeWeaponEditor.item}
                />
            )}

            {/* Remove Confirm Modal */}
            {removeModal && (
                <div className="fixed inset-0 z-[110] flex items-center justify-center bg-background/80 backdrop-blur-sm animate-in fade-in duration-300">
                    <div className="bg-card p-8 rounded-2xl border border-red-500/20 flex flex-col space-y-6 max-w-sm w-full mx-4 shadow-2xl shadow-red-500/10 animate-in zoom-in-95 duration-300">
                        <div className="flex items-center space-x-4">
                            <div className="w-12 h-12 rounded-xl bg-red-500/10 border border-red-500/30 flex items-center justify-center shrink-0">
                                <svg className="w-6 h-6 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                            </div>
                            <div>
                                <h3 className="text-sm font-black uppercase tracking-widest text-foreground">Remove {removeModal.handles.length} item{removeModal.handles.length > 1 ? 's' : ''}?</h3>
                                <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest mt-0.5">Choose what to remove</p>
                            </div>
                        </div>
                        {removeModal.names.length <= 5 && (
                            <ul className="space-y-1">
                                {removeModal.names.map((n, i) => (
                                    <li key={i} className="text-[11px] text-muted-foreground font-medium truncate">— {n}</li>
                                ))}
                            </ul>
                        )}
                        <div className="flex flex-col space-y-2 pt-1">
                            <button onClick={() => confirmRemove(true, true)} disabled={isRemoving} className="w-full px-4 py-2.5 bg-red-500 text-white rounded-md text-[10px] font-black uppercase tracking-widest hover:bg-red-600 active:scale-95 transition-all disabled:opacity-50">
                                {isRemoving ? 'Removing...' : 'Remove from Inventory & Storage'}
                            </button>
                            <button onClick={() => confirmRemove(true, false)} disabled={isRemoving} className="w-full px-4 py-2.5 bg-muted/40 text-foreground rounded-md text-[10px] font-black uppercase tracking-widest border border-border hover:bg-muted/60 active:scale-95 transition-all disabled:opacity-50">
                                Inventory only
                            </button>
                            <button onClick={() => confirmRemove(false, true)} disabled={isRemoving} className="w-full px-4 py-2.5 bg-muted/40 text-foreground rounded-md text-[10px] font-black uppercase tracking-widest border border-border hover:bg-muted/60 active:scale-95 transition-all disabled:opacity-50">
                                Storage only
                            </button>
                            <button onClick={() => setRemoveModal(null)} disabled={isRemoving} className="w-full px-4 py-2.5 bg-transparent text-muted-foreground rounded-md text-[10px] font-black uppercase tracking-widest hover:text-foreground transition-all">
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {/* Icon Popover */}
            {selectedIcon && (
                <div 
                    className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm animate-in fade-in duration-300"
                    onClick={() => setSelectedIcon(null)}
                >
                    <div className="card p-8 flex flex-col items-center space-y-6 max-w-sm w-full mx-4 shadow-2xl shadow-primary/20 border-primary/20 animate-in zoom-in-95 duration-300">
                        <div className="relative group">
                            <div className="absolute -inset-4 bg-primary/10 rounded-full blur-2xl group-hover:bg-primary/20 transition-all duration-500" />
                            <img 
                                src={selectedIcon.path} 
                                alt={selectedIcon.name}
                                className="w-48 h-48 object-contain relative z-10 drop-shadow-[0_0_15px_rgba(var(--primary),0.3)]"
                                onError={(e) => (e.currentTarget.src = '/src/assets/images/logo-universal.png')}
                            />
                        </div>
                        <div className="text-center space-y-2">
                            <h3 className="text-lg font-black uppercase tracking-widest text-foreground">{selectedIcon.name}</h3>
                            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-[0.3em]">Item Preview</p>
                        </div>
                        <button className="text-[10px] font-black uppercase tracking-widest text-muted-foreground hover:text-foreground transition-colors">Click anywhere to close</button>
                    </div>
                </div>
            )}

            {/* Hover Icon Tooltip */}
            {hoverTooltip && (
                <div className="fixed z-[60] pointer-events-none" style={{left: hoverTooltip.x, top: hoverTooltip.y - 8, transform: 'translate(-50%, -100%)'}}>
                    <div className="bg-card border border-border rounded-lg shadow-xl p-2">
                        <img src={hoverTooltip.path} alt="" className="w-24 h-24 object-contain drop-shadow-md" onError={(e) => (e.currentTarget.style.display = 'none')} />
                    </div>
                </div>
            )}

            {/* Capacity bar moved to App.tsx (header consolidation, spec/36) */}

            {/* Top Bar: [Category] [Owned/Total badge] [Search] */}
            <div className="flex flex-col md:flex-row gap-4 shrink-0 items-center">
                <CategorySelect value={category} onChange={setCategory} className="w-full md:w-56 shrink-0" />

                {/* Subcategory filter — disabled placeholder keeps the layout stable
                    when the active category has no subcategories. */}
                <div className="relative w-48 shrink-0">
                    <select
                        aria-label="Subcategory"
                        value={subCategory}
                        disabled={!hasSubCategories}
                        onChange={e => setSubCategory(e.target.value)}
                        className="w-full appearance-none bg-muted/30 border border-border rounded-md px-4 py-2.5 pr-10 text-[10px] font-black uppercase tracking-widest text-muted-foreground outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                        <option value="all">{hasSubCategories ? 'All Subcategories' : 'No Subcategories'}</option>
                        {availableSubCategories.map(s => (
                            <option key={s} value={s}>{s}</option>
                        ))}
                    </select>
                    <div className="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none text-muted-foreground">
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M19 9l-7 7-7-7"></path></svg>
                    </div>
                </div>

                <div className="flex items-center gap-2 px-3 py-2 rounded-md bg-muted/20 border border-border/50">
                    <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Owned:</span>
                    <span className="text-[10px] font-bold tabular-nums text-foreground">{ownedCount}</span>
                </div>

                {(Object.keys(editedInv).length > 0 || Object.keys(editedStorage).length > 0 || workspace.dirty) && (
                    <button
                        onClick={saveChanges}
                        disabled={isSaving || workspace.saving}
                        className="px-6 py-2.5 bg-primary text-primary-foreground rounded-md text-[10px] font-black uppercase tracking-widest shadow-lg shadow-primary/20 hover:scale-105 active:scale-95 transition-all disabled:opacity-50 disabled:scale-100"
                    >
                        {isSaving || workspace.saving ? 'Saving...' : 'Save Changes'}
                    </button>
                )}
                {selectedKeys.size > 0 && (
                    <button
                        onClick={handleRemoveSelected}
                        className="px-6 py-2.5 bg-red-500/10 text-red-400 border border-red-500/30 rounded-md text-[10px] font-black uppercase tracking-widest hover:bg-red-500/20 active:scale-95 transition-all animate-in zoom-in-95 duration-200"
                    >
                        Remove ({selectedKeys.size})
                    </button>
                )}

                <div className="flex-1" />

                <div className="flex items-center gap-1 shrink-0">
                    {onToggleFavorites && (
                        <>
                            <button
                                onClick={onToggleFavorites}
                                className={`p-1.5 rounded transition-all ${showOnlyFavorites ? 'bg-amber-500/20 text-amber-500' : 'text-amber-700/60 hover:text-amber-500'}`}
                                title="Show favorites only"
                            >
                                <svg className="w-4 h-4" fill={showOnlyFavorites ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
                                </svg>
                            </button>
                            <div className="w-px h-4 bg-border/50 mx-0.5" />
                        </>
                    )}
                    <button onClick={() => setViewMode('table')} className={`p-1.5 rounded transition-all ${viewMode === 'table' ? 'bg-primary/20 text-primary' : 'text-muted-foreground/40 hover:text-muted-foreground'}`} title="Table view">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2"><path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" /></svg>
                    </button>
                    <button onClick={() => setViewMode('grid')} className={`p-1.5 rounded transition-all ${viewMode === 'grid' ? 'bg-primary/20 text-primary' : 'text-muted-foreground/40 hover:text-muted-foreground'}`} title="Grid view">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2"><path strokeLinecap="round" strokeLinejoin="round" d="M4 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm10 0a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zM4 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1v-4zm10 0a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" /></svg>
                    </button>
                </div>

                <div className="relative w-full max-w-xs shrink-0">
                    <input
                        type="text"
                        placeholder="Search owned items..."
                        value={search}
                        onChange={e => setSearch(e.target.value)}
                        className="w-full bg-muted/30 border border-border rounded-md px-10 py-2.5 text-xs font-semibold focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                    />
                    <div className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                    </div>
                </div>
            </div>

            {/* Content — Table or Grid + Detail Panel */}
            <div className="card overflow-hidden flex flex-col flex-1 min-h-0">
                <div className="flex-1 flex min-h-0 relative">
                <div className={`flex-1 min-w-0 flex flex-col ${detailItem ? 'max-w-[60%]' : ''}`}>
                {viewMode === 'grid' ? (
                    <div className="overflow-y-auto flex-1 custom-scrollbar p-3">
                        {loading ? (
                            <div className="flex flex-col items-center justify-center py-16 space-y-4">
                                <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
                                <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Accessing data...</p>
                            </div>
                        ) : filteredOwnedItems.length === 0 ? (
                            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground/50">
                                <svg className="w-8 h-8 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-2.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"/></svg>
                                <span className="text-[10px] font-black uppercase tracking-widest">No items found</span>
                            </div>
                        ) : (
                            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-2">
                                {filteredOwnedItems.map(item => {
                                    const key = rowKey(item);
                                    return (
                                        <div key={key}
                                            className={`relative rounded-xl border bg-card p-1.5 flex flex-col items-center gap-1 transition-all hover:border-primary/40 hover:bg-primary/[0.03] group ${
                                                item.isEquippedAoW
                                                    ? 'border-border/50 bg-green-500/[0.08]'
                                                    : selectedKeys.has(key)
                                                        ? 'border-red-500/50 bg-red-500/[0.03]'
                                                        : 'border-border/50'
                                            }`}
                                            title={item.isEquippedAoW ? `Equipped on ${item.equippedByWeaponName || 'weapon'}` : undefined}
                                        >
                                            <button onClick={e => { e.stopPropagation(); toggleFav(item.id); }} className="absolute top-2 right-2 p-0.5 transition-all hover:scale-125 z-10">
                                                <svg className={`w-3.5 h-3.5 ${isFav(item.id) ? 'text-amber-500 fill-amber-500' : 'text-muted-foreground/20 fill-none hover:text-amber-500/50'}`} stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                                                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
                                                </svg>
                                            </button>
                                            {!item.readOnly && (
                                                <button onClick={() => toggleSelect(key)} className="absolute top-2 left-2 z-10">
                                                    <div className={`w-4 h-4 rounded border flex items-center justify-center transition-all ${selectedKeys.has(key) ? 'bg-red-500 border-red-500' : 'bg-muted/30 border-border hover:border-red-400/50'}`}>
                                                        {selectedKeys.has(key) && <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                                    </div>
                                                </button>
                                            )}
                                            <div className="w-20 h-20 rounded-lg bg-muted/30 border border-border/50 flex items-center justify-center overflow-hidden cursor-pointer"
                                                onClick={() => handleItemClick(item)}
                                                onMouseEnter={(e) => { const r = e.currentTarget.getBoundingClientRect(); setHoverTooltip({name: item.name, path: item.iconPath, x: r.left + r.width / 2, y: r.top}); }}
                                                onMouseLeave={() => setHoverTooltip(null)}>
                                                {brokenIcons.has(item.iconPath)
                                                    ? <span className="text-[10px] font-black text-muted-foreground/30">?</span>
                                                    : <img src={item.iconPath} alt="" className="w-full h-full p-0.5 object-contain drop-shadow-md group-hover:scale-110 transition-transform duration-300" onError={() => handleImageError(item.iconPath)} />
                                                }
                                            </div>
                                            <div className="text-center w-full cursor-pointer" onClick={() => handleItemClick(item)}>
                                                <div className="text-[10px] font-bold text-foreground truncate group-hover:text-primary transition-colors" title={item.name}>{item.name}</div>
                                                {item.maxUpgrade > 0 && (
                                                    <span className="text-[8px] font-mono font-bold text-primary/60">+{item.currentUpgrade}/{item.maxUpgrade}</span>
                                                )}
                                            </div>
                                            <div className="flex items-center gap-2">
                                                <ItemCapacityBadge owned={item.separateInstances ? (item.inInventory ? 1 : 0) : item.invQty} max={item.maxInv} label="Inventory" prefix="I" compact mode={item.separateInstances ? 'instance' : 'quantity'} />
                                                <ItemCapacityBadge owned={item.separateInstances ? (item.inStorage ? 1 : 0) : item.storageQty} max={item.maxStorage} label="Storage" prefix="S" compact mode={item.separateInstances ? 'instance' : 'quantity'} />
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        )}
                    </div>
                ) : (
                <div ref={scrollRef} className="overflow-y-auto flex-1 custom-scrollbar">
                    <table className="w-full text-left text-sm border-collapse">
                        <thead className="bg-muted/30 text-[10px] font-black text-muted-foreground uppercase tracking-[0.2em] sticky top-0 z-10 backdrop-blur-md border-b border-border">
                            <tr>
                                <th className="pl-3 pr-1 py-4 w-6">
                                    <div
                                        onClick={toggleSelectAll}
                                        className={`w-4 h-4 rounded border flex items-center justify-center transition-all cursor-pointer ${selectedKeys.size === selectableOwnedCount && selectableOwnedCount > 0 ? 'bg-red-500 border-red-500' : 'bg-muted/30 border-border hover:border-red-400/50'}`}
                                    >
                                        {selectedKeys.size === selectableOwnedCount && selectableOwnedCount > 0 &&
                                            <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                    </div>
                                </th>
                                <th className="px-1 py-4 w-6"></th>
                                {showWeaponColumns && <th className="px-1 py-4 w-6"></th>}
                                <th className="pl-1 pr-2 py-4 w-14">Icon</th>
                                <th className="pl-2 pr-6 py-4 w-full cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('name')}>
                                    Name <SortIndicator col="name" />
                                </th>
                                {columnVisibility.id && (
                                    <th className="px-3 py-4 whitespace-nowrap cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('id')}>
                                        ID <SortIndicator col="id" />
                                    </th>
                                )}
                                {columnVisibility.category && showSubGroupColumn && (
                                    <th className="px-3 py-4 whitespace-nowrap cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('subGroup')}>
                                        Sub-Category <SortIndicator col="subGroup" />
                                    </th>
                                )}
                                {showUpgradeColumn && (
                                    <th className="px-4 py-4 text-center whitespace-nowrap">
                                        <span
                                            onClick={() => handleSort('currentUpgrade')}
                                            className="cursor-pointer hover:text-primary transition-colors"
                                            title="Sort by current upgrade"
                                            aria-label="Sort by current upgrade"
                                        >Up<SortIndicator col="currentUpgrade" /></span>
                                        <span className="mx-1 text-muted-foreground/40">/</span>
                                        <span
                                            onClick={() => handleSort('maxUpgrade')}
                                            className="cursor-pointer hover:text-primary transition-colors"
                                            title="Sort by max upgrade"
                                            aria-label="Sort by max upgrade"
                                        >MaxUp<SortIndicator col="maxUpgrade" /></span>
                                    </th>
                                )}
                                {showWeaponColumns && (
                                    <th className="px-3 py-4 text-left whitespace-nowrap">Ash of War</th>
                                )}
                                <th className="px-3 py-4 text-center whitespace-nowrap cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('invOwned')}>
                                    Inventory <SortIndicator col="invOwned" />
                                </th>
                                <th className="px-3 py-4 text-center whitespace-nowrap cursor-pointer hover:text-primary transition-colors" onClick={() => handleSort('storageOwned')}>
                                    Storage <SortIndicator col="storageOwned" />
                                </th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-border/30">
                            {loading ? (
                                <tr>
                                    <td colSpan={tableColumnCount} className="px-6 py-24 text-center">
                                        <div className="flex flex-col items-center justify-center space-y-4">
                                            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
                                            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Accessing data...</p>
                                        </div>
                                    </td>
                                </tr>
                            ) : filteredOwnedItems.length > 0 ? (
                                <>
                                {rowVirtualizer.getVirtualItems().length > 0 && rowVirtualizer.getVirtualItems()[0].start > 0 && (
                                    <tr><td colSpan={tableColumnCount} style={{ height: rowVirtualizer.getVirtualItems()[0].start, padding: 0, border: 'none' }} /></tr>
                                )}
                                {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                                    const item = filteredOwnedItems[virtualRow.index];
                                    const key = rowKey(item);
                                    return (
                                    <tr
                                        key={key}
                                        data-index={virtualRow.index}
                                        ref={node => { if (node) rowVirtualizer.measureElement(node); }}
                                        className={`group hover:bg-primary/[0.02] transition-colors ${
                                            item.isEquippedAoW
                                                ? 'bg-green-500/[0.06] hover:bg-green-500/[0.09]'
                                                : selectedKeys.has(key) ? 'bg-red-500/[0.03]' : ''
                                        }`}
                                        title={item.isEquippedAoW ? `Equipped on ${item.equippedByWeaponName || 'weapon'}` : undefined}
                                    >
                                        <td className="pl-3 pr-1 py-0.5">
                                            {item.readOnly ? (
                                                <div
                                                    className="w-4 h-4 rounded border border-border/20 bg-muted/10 flex items-center justify-center"
                                                    title={item.isEquippedAoW
                                                        ? `Attached to ${item.equippedByWeaponName || 'weapon'}. Edit it from the weapon.`
                                                        : 'Managed by World tab'}
                                                >
                                                    <svg className="w-2.5 h-2.5 text-muted-foreground/30" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z" clipRule="evenodd"/></svg>
                                                </div>
                                            ) : (
                                                <div
                                                    onClick={() => toggleSelect(key)}
                                                    className={`w-4 h-4 rounded border flex items-center justify-center transition-all cursor-pointer ${selectedKeys.has(key) ? 'bg-red-500 border-red-500' : 'bg-muted/30 border-border group-hover:border-red-400/50'}`}
                                                >
                                                    {selectedKeys.has(key) &&
                                                        <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M5 13l4 4L19 7"/></svg>}
                                                </div>
                                            )}
                                        </td>
                                        <td className="px-1 text-center">
                                            <button onClick={e => { e.stopPropagation(); toggleFav(item.id); }} className="p-0.5 transition-all hover:scale-125">
                                                <svg className={`w-4 h-4 ${isFav(item.id) ? 'text-amber-500 fill-amber-500' : 'text-muted-foreground/20 fill-none hover:text-amber-500/50'}`} stroke="currentColor" viewBox="0 0 24 24" strokeWidth="2">
                                                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
                                                </svg>
                                            </button>
                                        </td>
                                        {showWeaponColumns && (
                                            <td className="px-1 text-center">
                                                {(() => {
                                                    const weapon = getWorkspaceWeapon(item);
                                                    return weapon ? (
                                                        <button
                                                            type="button"
                                                            onClick={event => {
                                                                event.stopPropagation();
                                                                setWeaponEditor({uid: weapon.item.uid, source: weapon.source});
                                                            }}
                                                            title="Edit weapon"
                                                            aria-label={`Edit ${item.name}`}
                                                            className="w-5 h-5 inline-flex items-center justify-center rounded bg-red-700/85 hover:bg-red-600 text-white shadow ring-1 ring-red-900/40 transition-colors"
                                                        >
                                                            <svg className="w-3 h-3" fill="none" stroke="currentColor" strokeWidth="2.5" viewBox="0 0 24 24" aria-hidden="true">
                                                                <path strokeLinecap="round" strokeLinejoin="round" d="M14.7 6.3l3 3M4 20l3.5-1 9.8-9.8a2.1 2.1 0 0 0 0-3l-.5-.5a2.1 2.1 0 0 0-3 0L4 16.5 4 20z" />
                                                            </svg>
                                                        </button>
                                                    ) : null;
                                                })()}
                                            </td>
                                        )}
                                        <td className="pl-1 pr-2 py-0.5">
                                            <div
                                                className="w-12 h-12 rounded bg-muted/30 border border-border/50 flex items-center justify-center overflow-hidden group-hover:border-primary/30 transition-all cursor-pointer"
                                                onClick={() => handleItemClick(item)}
                                                onMouseEnter={(e) => { const r = e.currentTarget.getBoundingClientRect(); setHoverTooltip({name: item.name, path: item.iconPath, x: r.left + r.width / 2, y: r.top}); }}
                                                onMouseLeave={() => setHoverTooltip(null)}
                                            >
                                                {brokenIcons.has(item.iconPath) ? (
                                                    <span className="text-[10px] font-black text-muted-foreground/30 select-none">?</span>
                                                ) : (
                                                    <img
                                                        src={item.iconPath}
                                                        alt=""
                                                        className="w-full h-full object-contain drop-shadow-md group-hover:scale-110 transition-transform duration-500"
                                                        onError={() => handleImageError(item.iconPath)}
                                                    />
                                                )}
                                            </div>
                                        </td>
                                        <td className="pl-2 pr-6 py-4 w-full">
                                            <div className="flex flex-col gap-0.5">
                                                <div className="flex items-center gap-1.5 flex-wrap">
                                                    <span className="text-[13px] font-semibold text-foreground group-hover:text-primary transition-colors cursor-pointer hover:underline decoration-primary/40 underline-offset-2" onClick={() => handleItemClick(item)}>{item.name}</span>
                                                    {item.flags?.includes('cut_content') && (
                                                        <RiskBadge flag="cut_content" />
                                                    )}
                                                    {item.flags?.includes('ban_risk') && (
                                                        <RiskBadge flag="ban_risk" />
                                                    )}
                                                </div>
                                            </div>
                                        </td>
                                        {columnVisibility.id && (
                                            <td className="px-3 py-4 font-mono text-[10px] text-muted-foreground whitespace-nowrap">0x{item.id.toString(16).toUpperCase()}</td>
                                        )}
                                        {columnVisibility.category && showSubGroupColumn && (
                                            <td className="px-3 py-4 whitespace-nowrap">
                                                <span className="text-[8px] font-black uppercase tracking-widest px-2 py-1 bg-muted/50 rounded border border-border/50 text-muted-foreground">
                                                    {category === 'all'
                                                        ? (CATEGORY_LABEL[item.subCategory] ?? item.subCategory.replace(/_/g, ' '))
                                                        : (item.subGroup || '—')}
                                                </span>
                                            </td>
                                        )}
                                        {showUpgradeColumn && (
                                            <td className="px-4 py-4 text-center">
                                                {item.maxUpgrade > 0 ? (
                                                    <span className="text-[10px] font-black tabular-nums whitespace-nowrap bg-primary/5 px-2 py-1 rounded border border-primary/10" title={`Upgrade +${item.currentUpgrade} of max +${item.maxUpgrade}`}>
                                                        <span className="text-primary">+{item.currentUpgrade}</span>
                                                        <span className="text-muted-foreground/50 mx-0.5">/</span>
                                                        <span className="text-muted-foreground">+{item.maxUpgrade}</span>
                                                    </span>
                                                ) : (
                                                    <span className="text-muted-foreground/50 text-[10px] font-black">—</span>
                                                )}
                                            </td>
                                        )}
                                        {showWeaponColumns && (
                                            <td className="px-3 py-4 whitespace-nowrap">
                                                {(() => {
                                                    const weapon = getWorkspaceWeapon(item)?.item;
                                                    if (!weapon) return <span className="text-muted-foreground/30">—</span>;
                                                    if (weapon.pendingAoWItemID && weapon.pendingAoWName) {
                                                        return <span className="text-[11px] font-semibold text-green-500" title="Pending save">{weapon.pendingAoWName}</span>;
                                                    }
                                                    if (weapon.pendingAoWClear) {
                                                        return <span className="text-[11px] font-semibold text-muted-foreground" title="Built-in Ash of War (pending save)">{weapon.defaultAoWName || 'Built-in skill'}</span>;
                                                    }
                                                    if (weapon.hasCurrentAoW && weapon.currentAoWName) {
                                                        return <span className="text-[11px] font-semibold text-green-500">{weapon.currentAoWName}</span>;
                                                    }
                                                    return <span className="text-[11px] font-semibold text-muted-foreground" title="Built-in Ash of War">{weapon.defaultAoWName || 'Built-in skill'}</span>;
                                                })()}
                                            </td>
                                        )}
                                        <td className="px-3 py-4 text-center whitespace-nowrap">
                                            {item.maxInv <= 1 || item.readOnly ? (
                                                <ItemCapacityBadge owned={item.inInventory ? (item.separateInstances ? 1 : item.invQty) : 0} max={item.maxInv} label="Inventory" mode={item.separateInstances ? 'instance' : 'quantity'} />
                                            ) : item.inInventory ? (
                                                <div className="flex items-center justify-center space-x-2">
                                                    <input
                                                        type="number"
                                                        min={0}
                                                        max={item.maxInv}
                                                        value={editedInv[item.invHandle] !== undefined ? editedInv[item.invHandle] : item.invQty}
                                                        onChange={e => handleQtyChange(item.invHandle, e.target.value, 'inv', item.maxInv)}
                                                        className={`w-16 bg-muted/20 border rounded px-2 py-1 text-center text-xs font-bold outline-none focus:ring-1 focus:ring-primary/30 transition-all ${editedInv[item.invHandle] !== undefined ? 'border-primary/50 text-primary bg-primary/5' : 'border-border/50 text-foreground'}`}
                                                    />
                                                    <span className="text-[9px] font-bold text-muted-foreground/65 uppercase tracking-tighter whitespace-nowrap shrink-0">/ {item.maxInv}</span>
                                                </div>
                                            ) : (
                                                <span className="text-muted-foreground/60 text-[10px] font-black">—</span>
                                            )}
                                        </td>
                                        <td className="px-3 py-4 text-center whitespace-nowrap">
                                            {item.maxStorage <= 1 || item.readOnly ? (
                                                <ItemCapacityBadge owned={item.inStorage ? (item.separateInstances ? 1 : item.storageQty) : 0} max={item.maxStorage} label="Storage" mode={item.separateInstances ? 'instance' : 'quantity'} />
                                            ) : item.inStorage ? (
                                                <div className="flex items-center justify-center space-x-2">
                                                    <input
                                                        type="number"
                                                        min={0}
                                                        max={item.maxStorage}
                                                        value={editedStorage[item.storageHandle] !== undefined ? editedStorage[item.storageHandle] : item.storageQty}
                                                        onChange={e => handleQtyChange(item.storageHandle, e.target.value, 'storage', item.maxStorage)}
                                                        className={`w-16 bg-muted/20 border rounded px-2 py-1 text-center text-xs font-bold outline-none focus:ring-1 focus:ring-primary/30 transition-all ${editedStorage[item.storageHandle] !== undefined ? 'border-primary/50 text-primary bg-primary/5' : 'border-border/50 text-foreground'}`}
                                                    />
                                                    <span className="text-[9px] font-bold text-muted-foreground/65 uppercase tracking-tighter whitespace-nowrap shrink-0">/ {item.maxStorage}</span>
                                                </div>
                                            ) : (
                                                <span className="text-muted-foreground/60 text-[10px] font-black">—</span>
                                            )}
                                        </td>
                                    </tr>
                                    );})}
                                {(() => {
                                    const virtualItems = rowVirtualizer.getVirtualItems();
                                    const paddingBottom = virtualItems.length > 0
                                        ? rowVirtualizer.getTotalSize() - virtualItems[virtualItems.length - 1].end
                                        : 0;
                                    return paddingBottom > 0 ? <tr><td colSpan={tableColumnCount} style={{ height: paddingBottom, padding: 0, border: 'none' }} /></tr> : null;
                                })()}
                                </>
                            ) : (
                                <tr>
                                    <td colSpan={tableColumnCount} className="px-6 py-24 text-center">
                                        <p className="text-xs text-muted-foreground font-medium italic">Nothing found in this section.</p>
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>
                )}
                </div>

                {detailItem && (
                    <div className="absolute top-0 right-0 bottom-0 w-[40%] animate-in slide-in-from-right duration-200">
                        <ItemDetailPanel item={detailItem} onClose={() => setDetailItem(null)} />
                    </div>
                )}
                </div>
            </div>
        </div>
    );
}
