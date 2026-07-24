import { useEffect, useMemo, useState } from 'react';
import { createPortal } from 'react-dom';
import {
    GetArmsSlotEligibleItems,
    GetArrowSlotEligibleItems,
    GetBoltSlotEligibleItems,
    GetCharacter,
    GetChestSlotEligibleItems,
    GetHandArmamentEligibleItems,
    GetHeadSlotEligibleItems,
    GetItemList,
    GetLegsSlotEligibleItems,
    GetPhysickEligibleItems,
    GetPouchEligibleItems,
    GetQuickItemEligibleItems,
    GetTalismanSlotEligibleItems,
} from '../../wailsjs/go/main/App';
import type { db, vm } from '../../wailsjs/go/models';
import { loadSafetyProfile, revealsRiskyItems, SAFETY_PROFILE_EVENT, type SafetyProfile } from '../state/safetyProfile';

type PickerSource = 'inventory' | 'database';
type PickerView = 'icons' | 'list';
type PickerSort = 'alphabetical' | 'weight' | 'category' | 'acquisition';

const RISKY_ITEM_FLAGS = ['cut_content', 'ban_risk', 'pre_order', 'dlc_duplicate'];

type PickerItem = {
    entryKey: string;
    id: number;
    name: string;
    category: string;
    iconPath: string;
    quantity?: number;
    stackable: boolean;
    weight?: number;
    acquisitionOrder?: number;
};

export type EquipmentItemPickerModalProps = {
    slotLabel: string;
    charIdx?: number;
    onClose: () => void;
};

const iconSrc = (path: string) => (path.startsWith('/') ? path : `/${path}`);

function eligibleItemsForSlot(label: string): Promise<db.ItemEntry[]> {
    if (label.startsWith('Weapon slot') || label.startsWith('Ranged slot')) return GetHandArmamentEligibleItems();
    if (label.startsWith('Arrow slot')) return GetArrowSlotEligibleItems();
    if (label.startsWith('Bolt slot')) return GetBoltSlotEligibleItems();
    if (label === 'Knight Helm') return GetHeadSlotEligibleItems();
    if (label === 'Knight Armor') return GetChestSlotEligibleItems();
    if (label === 'Knight Gauntlets') return GetArmsSlotEligibleItems();
    if (label === 'Knight Greaves') return GetLegsSlotEligibleItems();
    if (['Axe Talisman', 'Claw Talisman', 'Companion Jar', 'Gold Scarab'].includes(label)) return GetTalismanSlotEligibleItems();
    if (label.startsWith('Quick item')) return GetQuickItemEligibleItems();
    if (label.startsWith('Quick pouch')) return GetPouchEligibleItems();
    if (label.startsWith('Physick tear')) return GetPhysickEligibleItems();
    if (label.startsWith('Spell slot')) {
        return Promise.all([GetItemList('sorceries'), GetItemList('incantations')]).then(([sorceries, incantations]) => [...sorceries, ...incantations]);
    }
    return Promise.resolve([]);
}

function toPickerItem(item: db.ItemEntry): PickerItem {
    return {
        entryKey: `database-${item.id}`,
        id: item.id,
        name: item.name,
        category: item.category,
        iconPath: item.iconPath,
        stackable: item.maxInventory > 1,
        weight: item.weight,
    };
}

function toOwnedPickerItem(item: vm.ItemViewModel, eligibleItem: db.ItemEntry | undefined, acquisitionOrder: number): PickerItem {
    return {
        entryKey: `inventory-${acquisitionOrder}-${item.handle}`,
        id: item.baseId || item.id,
        name: item.name,
        category: item.category,
        iconPath: item.iconPath,
        quantity: item.quantity,
        stackable: item.maxInventory > 1,
        weight: eligibleItem?.weight,
        acquisitionOrder,
    };
}

function matchesSearch(item: PickerItem, query: string) {
    const normalized = query.trim().toLowerCase();
    return !normalized || item.name.toLowerCase().includes(normalized) || item.category.toLowerCase().includes(normalized);
}

function sortItems(items: PickerItem[], sort: PickerSort): PickerItem[] {
    return [...items].sort((a, b) => {
        if (sort === 'weight') {
            return (a.weight ?? Number.POSITIVE_INFINITY) - (b.weight ?? Number.POSITIVE_INFINITY)
                || a.name.localeCompare(b.name);
        }
        if (sort === 'category') return a.category.localeCompare(b.category) || a.name.localeCompare(b.name);
        if (sort === 'acquisition') {
            return (a.acquisitionOrder ?? a.id) - (b.acquisitionOrder ?? b.id)
                || a.name.localeCompare(b.name);
        }
        return a.name.localeCompare(b.name);
    });
}

function ItemCard({ item, source, view, selected, onSelect }: {
    item: PickerItem;
    source: PickerSource;
    view: PickerView;
    selected: boolean;
    onSelect: (item: PickerItem) => void;
}) {
    const selectionClass = selected ? 'border-primary ring-1 ring-primary/50' : 'border-border hover:border-primary/60';

    if (view === 'list') {
        return (
            <div className={`flex min-h-16 items-center gap-3 rounded-lg border p-2 transition-colors ${selectionClass}`}>
                <button type="button" aria-label={`Select ${item.name}`} className="flex min-w-0 flex-1 items-center gap-3 text-left" onClick={() => onSelect(item)}>
                    <img className="h-12 w-12 shrink-0 object-contain" src={iconSrc(item.iconPath)} alt="" />
                    <span className="min-w-0">
                        <span className="block truncate text-xs font-bold text-foreground">{item.name}</span>
                        <span className="block truncate text-[10px] text-muted-foreground">{item.category}{item.quantity != null ? ` · ${item.quantity}` : ''}</span>
                    </span>
                </button>
            </div>
        );
    }

    return (
        <div className={`relative flex min-h-32 flex-col rounded-lg border p-2 transition-colors ${selectionClass}`}>
            {item.quantity != null && item.stackable && <span className="absolute right-2 top-2 text-xs font-black text-foreground">×{item.quantity}</span>}
            <button type="button" aria-label={`Select ${item.name}`} className="flex min-h-0 flex-1 flex-col items-center justify-center gap-1" onClick={() => onSelect(item)}>
                <img className={`${source === 'database' ? 'h-20 w-20' : 'h-16 w-16'} object-contain`} src={iconSrc(item.iconPath)} alt={item.name} />
                <span className="line-clamp-2 text-center text-[10px] font-bold leading-tight text-foreground">{item.name}</span>
                {item.quantity == null && <span className="text-[9px] text-muted-foreground">{item.category}</span>}
            </button>
        </div>
    );
}

export function EquipmentItemPickerModal({ slotLabel, charIdx, onClose }: EquipmentItemPickerModalProps) {
    const [source, setSource] = useState<PickerSource>('inventory');
    const [view, setView] = useState<PickerView>('icons');
    const [sort, setSort] = useState<PickerSort>('alphabetical');
    const [search, setSearch] = useState('');
    const [eligible, setEligible] = useState<db.ItemEntry[]>([]);
    const [owned, setOwned] = useState<vm.ItemViewModel[]>([]);
    const [loading, setLoading] = useState(true);
    const [selected, setSelected] = useState<PickerItem | null>(null);
    const [safetyProfile, setSafetyProfile] = useState<SafetyProfile>(() => loadSafetyProfile());

    useEffect(() => {
        let active = true;
        setLoading(true);
        setSelected(null);
        setSearch('');
        setSource('inventory');

        Promise.all([
            eligibleItemsForSlot(slotLabel),
            charIdx == null ? Promise.resolve<vm.CharacterViewModel | null>(null) : GetCharacter(charIdx),
        ]).then(([items, character]) => {
            if (!active) return;
            setEligible(items);
            setOwned(character?.inventory ?? []);
            setLoading(false);
        }).catch(() => {
            if (!active) return;
            setEligible([]);
            setOwned([]);
            setLoading(false);
        });

        return () => { active = false; };
    }, [slotLabel, charIdx]);

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') onClose();
        };
        window.addEventListener('keydown', onKeyDown);
        return () => window.removeEventListener('keydown', onKeyDown);
    }, [onClose]);

    useEffect(() => {
        const onSafetyProfileChanged = (event: Event) => setSafetyProfile((event as CustomEvent<SafetyProfile>).detail);
        window.addEventListener(SAFETY_PROFILE_EVENT, onSafetyProfileChanged);
        return () => window.removeEventListener(SAFETY_PROFILE_EVENT, onSafetyProfileChanged);
    }, []);

    const visibleEligible = useMemo(() => {
        if (revealsRiskyItems(safetyProfile)) return eligible;
        return eligible.filter(item => !item.flags?.some(flag => RISKY_ITEM_FLAGS.includes(flag)));
    }, [eligible, safetyProfile]);

    const inventoryItems = useMemo(() => {
        const eligibleByID = new Map(visibleEligible.map(item => [item.id, item]));
        return owned
            .filter(item => eligibleByID.has(item.baseId || item.id))
            .map((item, index) => toOwnedPickerItem(item, eligibleByID.get(item.baseId || item.id), index));
    }, [owned, visibleEligible]);

    const items = useMemo(
        () => sortItems(
            (source === 'inventory' ? inventoryItems : visibleEligible.map(toPickerItem))
                .filter(item => matchesSearch(item, search)),
            sort,
        ),
        [inventoryItems, search, sort, source, visibleEligible],
    );

    return createPortal(
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 p-4" onMouseDown={onClose}>
            <div role="dialog" aria-modal="true" aria-label="Select equipment item" className="flex h-[min(680px,calc(100vh-2rem))] w-full max-w-5xl flex-col overflow-hidden rounded-xl border border-border bg-card text-card-foreground shadow-2xl" onMouseDown={(event) => event.stopPropagation()}>
                <div data-testid="equipment-picker-toolbar" className="flex flex-wrap items-center justify-start gap-2 border-b border-border p-4">
                    <input
                        aria-label="Search items"
                        className="h-[32px] w-[448px] max-w-full rounded-md border border-border bg-background px-3 text-sm text-foreground outline-none placeholder:text-muted-foreground focus:border-primary"
                        placeholder="Search items..."
                        value={search}
                        onChange={(event) => setSearch(event.target.value)}
                    />
                    <div className="flex h-[32px] rounded-md border border-border p-0.5" aria-label="View mode">
                        <button type="button" aria-label="Icon view" aria-pressed={view === 'icons'} onClick={() => setView('icons')} className={`h-full rounded px-3 text-[10px] font-black uppercase tracking-wider ${view === 'icons' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'}`}>Icons</button>
                        <button type="button" aria-label="List view" aria-pressed={view === 'list'} onClick={() => setView('list')} className={`h-full rounded px-3 text-[10px] font-black uppercase tracking-wider ${view === 'list' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'}`}>List</button>
                    </div>
                    <select aria-label="Sort items" value={sort} onChange={(event) => setSort(event.target.value as PickerSort)} className="h-[32px] rounded-md border border-border bg-background px-2 text-[10px] font-black uppercase tracking-wider text-foreground outline-none focus:border-primary">
                        <option value="alphabetical">Alphabetical</option>
                        <option value="weight">Weight</option>
                        <option value="category">Category</option>
                        <option value="acquisition">Acquisition order</option>
                    </select>
                    <div className="ml-auto flex h-[32px] rounded-md border border-border p-0.5" aria-label="Item source">
                        <button type="button" aria-pressed={source === 'inventory'} onClick={() => setSource('inventory')} className={`h-full rounded px-3 text-[10px] font-black uppercase tracking-wider ${source === 'inventory' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'}`}>Inventory</button>
                        <button type="button" aria-pressed={source === 'database'} onClick={() => setSource('database')} className={`h-full rounded px-3 text-[10px] font-black uppercase tracking-wider ${source === 'database' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'}`}>Item Database</button>
                    </div>
                </div>

                <main className="min-h-0 flex-1 overflow-y-auto p-4 custom-scrollbar">
                    {loading ? <p className="py-12 text-center text-sm text-muted-foreground">Loading items…</p> : items.length === 0 ? <p className="py-12 text-center text-sm text-muted-foreground">No matching items.</p> : (
                        <div className={view === 'icons' ? 'grid grid-cols-[repeat(auto-fill,minmax(125px,1fr))] gap-2' : 'space-y-2'}>
                            {items.map(item => <ItemCard key={item.entryKey} item={item} source={source} view={view} selected={selected?.id === item.id} onSelect={setSelected} />)}
                        </div>
                    )}
                </main>

                <div className="relative flex items-center justify-between gap-4 border-t border-border p-4">
                    <span className="min-w-0 truncate text-sm text-muted-foreground">{selected ? `Selected: ${selected.name}` : 'Select an item to preview it.'}</span>
                    {source === 'database' && <span className="absolute left-1/2 -translate-x-1/2 text-center text-xs font-bold text-primary">Items from Item Database will be added to Inventory before equipping.</span>}
                    <div className="flex shrink-0 gap-2">
                        <button type="button" onClick={onClose} className="rounded border border-border px-4 py-2 text-[10px] font-black uppercase tracking-widest text-muted-foreground hover:bg-muted/40">Cancel</button>
                        <button type="button" onClick={onClose} className="rounded bg-primary px-4 py-2 text-[10px] font-black uppercase tracking-widest text-primary-foreground hover:opacity-90">Ok</button>
                    </div>
                </div>
            </div>
        </div>,
        document.body,
    );
}
