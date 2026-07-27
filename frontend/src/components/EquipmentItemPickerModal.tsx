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
    GetInfuseTypes,
    GetItemList,
    GetLegsSlotEligibleItems,
    GetPhysickEligibleItems,
    GetPouchEligibleItems,
    GetQuickItemEligibleItems,
} from '../../wailsjs/go/main/App';
import type { db, vm } from '../../wailsjs/go/models';
import { loadSafetyProfile, revealsRiskyItems, SAFETY_PROFILE_EVENT, type SafetyProfile } from '../state/safetyProfile';

export type PickerSource = 'inventory' | 'database';
type PickerView = 'icons' | 'list';
type PickerSort = 'alphabetical' | 'weight' | 'category' | 'acquisition';

const RISKY_ITEM_FLAGS = ['cut_content', 'ban_risk', 'pre_order', 'dlc_duplicate'];
const WEAPON_CATEGORIES = new Set(['melee_armaments', 'ranged_and_catalysts', 'shields']);
const ARMOR_CATEGORIES = new Set(['head', 'chest', 'arms', 'legs']);
const AMMO_CATEGORIES = new Set(['arrows_and_bolts']);
const QUICK_EQUIP_CATEGORIES = new Set(['tools', 'ashes']);
const SPELL_CATEGORIES = new Set(['sorceries', 'incantations']);
const PHYSICK_TEAR_CANONICAL_IDS = new Map<number, number>([
	[0x40002AFC, 0x40002AFD], // Cerulean Crystal Tear variant
    [0x40002B08, 0x40002B09], // Ruptured Crystal Tear variant
]);

const isWeaponCategory = (category: string) => WEAPON_CATEGORIES.has(category);
const isArmorCategory = (category: string) => ARMOR_CATEGORIES.has(category);
const isAmmoCategory = (category: string) => AMMO_CATEGORIES.has(category);
const isQuickEquipCategory = (category: string) => QUICK_EQUIP_CATEGORIES.has(category);
const isSpellCategory = (category: string) => SPELL_CATEGORIES.has(category);
const isWeaponSlot = (label: string) => label.startsWith('Weapon slot') || label.startsWith('Ranged slot');

type PickerItem = {
	entryKey: string;
	id: number;
	handle?: number;
    name: string;
    category: string;
    iconPath: string;
    quantity?: number;
    maxInventory: number;
    stackable: boolean;
    weight?: number;
    acquisitionOrder?: number;
    isWeapon: boolean;
    isArmor: boolean;
    isAmmo: boolean;
    isQuickEquipItem: boolean;
    isSpell: boolean;
    memorySlots?: number;
    upgradeLevel?: number;
    infusionName?: string;
    aowName?: string;
};

export type EquipmentItemPickerModalProps = {
    slotLabel: string;
    charIdx?: number;
    initialSelection?: EquipmentPickerSelection;
	disabledItemIDs?: number[];
	disabledItemHandles?: number[];
    // Spell picker only: the Memory Slot capacity (Y) and the cost already spent
    // by spells in OTHER slots (draft total minus the edited slot's own cost).
    // A candidate spell is blocked when spellUsedExcludingSelected + its cost
    // exceeds spellCapacity, or when its cost is unknown. Undefined for every
    // non-spell slot family, which disables the capacity check entirely.
    spellCapacity?: number;
    spellUsedExcludingSelected?: number;
    onConfirm?: (item: EquipmentPickerSelection) => void | Promise<void>;
    onClear?: () => void;
    onClose: () => void;
};

export type EquipmentPickerSelection = {
	id: number;
	handle?: number;
    name: string;
    iconPath: string;
    quantity?: number;
    memorySlots?: number;
    source: PickerSource;
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
    if (label === 'Axe Talisman' || label === 'Claw Talisman' || label === 'Companion Jar' || label === 'Gold Scarab') return GetItemList('talismans');
    if (label.startsWith('Quick item')) return GetQuickItemEligibleItems();
    if (label.startsWith('Quick pouch')) return GetPouchEligibleItems();
    if (label.startsWith('Physick tear')) return GetPhysickEligibleItems();
    if (label.startsWith('Spell slot')) {
        return Promise.all([GetItemList('sorceries'), GetItemList('incantations')]).then(([sorceries, incantations]) => [...sorceries, ...incantations]);
    }
    return Promise.resolve([]);
}

function toPickerItem(item: db.ItemEntry): PickerItem {
    const weapon = isWeaponCategory(item.category);
    const armor = isArmorCategory(item.category);
    const ammo = isAmmoCategory(item.category);
    const quickEquipItem = isQuickEquipCategory(item.category);
    const spell = isSpellCategory(item.category);
    return {
        entryKey: `database-${item.id}`,
        id: item.id,
        name: item.name,
        category: item.category,
        iconPath: item.iconPath,
        maxInventory: Math.max(1, item.maxInventory || 1),
        stackable: item.maxInventory > 1,
        weight: item.weight,
        isWeapon: weapon,
        isArmor: armor,
        isAmmo: ammo,
        isQuickEquipItem: quickEquipItem,
        isSpell: spell,
        memorySlots: spell ? item.memorySlots : undefined,
        upgradeLevel: weapon ? 0 : undefined,
        infusionName: weapon ? 'Standard' : undefined,
        aowName: weapon ? '—' : undefined,
    };
}

function toOwnedPickerItem(
    item: vm.ItemViewModel,
    eligibleItem: db.ItemEntry | undefined,
    acquisitionOrder: number,
    infuseTypes: db.InfuseType[],
    ashesOfWar: Map<number, string>,
    canonicalID?: number,
): PickerItem {
    const weapon = isWeaponCategory(eligibleItem?.category ?? item.category);
    const armor = isArmorCategory(eligibleItem?.category ?? item.category);
    const ammo = isAmmoCategory(eligibleItem?.category ?? item.category);
    const quickEquipItem = isQuickEquipCategory(eligibleItem?.category ?? item.category);
    const spell = isSpellCategory(eligibleItem?.category ?? item.category);
    const infusionOffset = item.id - item.baseId - item.currentUpgrade;
	return {
		entryKey: `inventory-${acquisitionOrder}-${item.handle}`,
		id: canonicalID ?? (item.baseId || item.id),
		handle: item.handle,
        name: canonicalID != null ? (eligibleItem?.name ?? item.name) : item.name,
        category: canonicalID != null ? (eligibleItem?.category ?? item.category) : item.category,
        iconPath: canonicalID != null ? (eligibleItem?.iconPath ?? item.iconPath) : item.iconPath,
        quantity: item.quantity,
        maxInventory: Math.max(1, item.maxInventory || 1),
        stackable: item.maxInventory > 1,
        weight: eligibleItem?.weight,
        acquisitionOrder,
        isWeapon: weapon,
        isArmor: armor,
        isAmmo: ammo,
        isQuickEquipItem: quickEquipItem,
        isSpell: spell,
        memorySlots: spell ? eligibleItem?.memorySlots : undefined,
        upgradeLevel: weapon ? item.currentUpgrade : undefined,
        infusionName: weapon ? (infuseTypes.find(type => type.offset === infusionOffset)?.name ?? 'Standard') : undefined,
        aowName: weapon ? (item.aowId ? (ashesOfWar.get(item.aowId) ?? 'Unknown Ash of War') : '—') : undefined,
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

function ItemCard({ item, source, view, weaponList, physickPicker, selected, disabled, blockReason, onSelect, onActivate }: {
    item: PickerItem;
    source: PickerSource;
    view: PickerView;
    weaponList: boolean;
    physickPicker: boolean;
    selected: boolean;
    disabled: boolean;
    blockReason?: string;
    onSelect: (item: PickerItem) => void;
    onActivate: (item: PickerItem) => void;
}) {
    const selectionClass = selected ? 'border-emerald-500 ring-2 ring-emerald-500/60' : disabled ? 'border-border opacity-40 grayscale' : 'border-border hover:border-primary/60';
    const prominentListName = item.isArmor || item.isAmmo || item.isQuickEquipItem || item.isSpell || physickPicker;
    // Multi-slot spells advertise their real Memory Slot cost; a blocked spell
    // (over capacity or unknown cost) shows why so the disabled state is legible.
    const costLabel = item.isSpell && item.memorySlots != null && item.memorySlots > 1 ? `${item.memorySlots} slots` : null;
    // Every spell shows its exact Memory Slot requirement as a persistent line,
    // including N=1 — distinct from the compact ×N badge above.
    const memoryLabel = item.isSpell && item.memorySlots != null ? `Required memory slots: ${item.memorySlots}` : null;

    if (view === 'list') {
        if (item.isWeapon) {
            return (
                <div data-picker-selected={selected || undefined} className={`col-span-4 grid min-h-16 items-center gap-x-3 rounded-lg border p-2 transition-colors ${selectionClass}`} style={weaponList ? { gridTemplateColumns: 'subgrid' } : undefined}>
                    <button type="button" disabled={disabled} aria-label={`Select ${item.name}`} className="flex min-w-0 items-center gap-3 text-left disabled:cursor-not-allowed" onClick={() => onSelect(item)} onDoubleClick={() => onActivate(item)}>
                        <img className="h-12 w-12 shrink-0 object-contain" src={iconSrc(item.iconPath)} alt="" />
                        <span className="block truncate text-sm font-bold text-foreground">{item.name}</span>
                    </button>
                    <span className="shrink-0 text-right text-xs font-bold text-muted-foreground">+{item.upgradeLevel ?? 0}</span>
                    <span className="shrink-0 text-right text-xs text-muted-foreground">{item.infusionName ?? '—'}</span>
                    <span className="max-w-48 shrink-0 truncate text-right text-xs text-muted-foreground">{item.aowName ?? '—'}</span>
                </div>
            );
        }
        return (
            <div data-picker-selected={selected || undefined} className={`flex min-h-16 items-center gap-3 rounded-lg border p-2 transition-colors ${selectionClass}`}>
                <button type="button" disabled={disabled} aria-label={`Select ${item.name}`} className="flex min-w-0 flex-1 items-center gap-3 text-left disabled:cursor-not-allowed" onClick={() => onSelect(item)} onDoubleClick={() => onActivate(item)}>
                    <img className="h-12 w-12 shrink-0 object-contain" src={iconSrc(item.iconPath)} alt="" />
                    <span className="min-w-0">
                        <span className={`block truncate font-bold text-foreground ${prominentListName ? 'text-sm' : 'text-xs'}`}>{item.name}</span>
                        {!prominentListName && <span className="block truncate text-[10px] text-muted-foreground">{item.category}{item.quantity != null ? ` · ${item.quantity}` : ''}</span>}
                        {memoryLabel && <span className="block truncate text-[10px] font-bold text-muted-foreground">{memoryLabel}</span>}
                        {blockReason && <span className="block truncate text-[10px] font-bold text-muted-foreground">{blockReason}</span>}
                    </span>
                </button>
            </div>
        );
    }

    return (
        <div data-picker-selected={selected || undefined} className={`relative flex min-h-32 flex-col rounded-lg border p-2 transition-colors ${selectionClass}`}>
            {item.quantity != null && item.stackable && <span className="absolute right-2 top-2 text-xs font-black text-foreground">×{item.quantity}</span>}
            {costLabel && <span className="absolute left-2 top-2 rounded bg-muted px-1 text-[9px] font-black text-muted-foreground">{costLabel}</span>}
            <button type="button" disabled={disabled} aria-label={`Select ${item.name}`} className="flex min-h-0 flex-1 flex-col items-center justify-center gap-1 disabled:cursor-not-allowed" onClick={() => onSelect(item)} onDoubleClick={() => onActivate(item)}>
                <img className={`${source === 'database' ? 'h-20 w-20' : 'h-16 w-16'} object-contain`} src={iconSrc(item.iconPath)} alt={item.name} />
                <span className="line-clamp-2 text-center text-[10px] font-bold leading-tight text-foreground">{item.name}</span>
                {item.isWeapon && <span className="text-[10px] font-bold text-muted-foreground">+{item.upgradeLevel ?? 0}</span>}
                {!item.isWeapon && !item.isSpell && item.quantity == null && !blockReason && <span className="text-[9px] text-muted-foreground">{item.category}</span>}
                {memoryLabel && <span className="text-center text-[9px] font-bold text-muted-foreground">{memoryLabel}</span>}
                {blockReason && <span className="text-center text-[9px] font-bold text-muted-foreground">{blockReason}</span>}
            </button>
        </div>
    );
}

export function EquipmentItemPickerModal({ slotLabel, charIdx, initialSelection, disabledItemIDs = [], disabledItemHandles = [], spellCapacity, spellUsedExcludingSelected = 0, onConfirm, onClear, onClose }: EquipmentItemPickerModalProps) {
    const [source, setSource] = useState<PickerSource>('inventory');
    const [view, setView] = useState<PickerView>('icons');
    const [sort, setSort] = useState<PickerSort>('alphabetical');
    const [search, setSearch] = useState('');
    const [eligible, setEligible] = useState<db.ItemEntry[]>([]);
    const [owned, setOwned] = useState<vm.ItemViewModel[]>([]);
    const [loading, setLoading] = useState(true);
    const [selected, setSelected] = useState<PickerItem | null>(null);
    const [safetyProfile, setSafetyProfile] = useState<SafetyProfile>(() => loadSafetyProfile());
    const [infuseTypes, setInfuseTypes] = useState<db.InfuseType[]>([]);
    const [ashesOfWar, setAshesOfWar] = useState<db.ItemEntry[]>([]);
    const [submitting, setSubmitting] = useState(false);
    const [submissionError, setSubmissionError] = useState('');
    const [quantityItem, setQuantityItem] = useState<PickerItem | null>(null);
    const [quantity, setQuantity] = useState(1);

    useEffect(() => {
        let active = true;
        setLoading(true);
		setSelected(initialSelection ? {
			entryKey: `current-${initialSelection.id}`,
			id: initialSelection.id,
			handle: initialSelection.handle,
            name: initialSelection.name,
            category: '',
            iconPath: initialSelection.iconPath,
            maxInventory: 1,
            stackable: false,
            isWeapon: false,
            isArmor: false,
            isAmmo: false,
            isQuickEquipItem: false,
            isSpell: slotLabel.startsWith('Spell slot'),
            memorySlots: initialSelection.memorySlots,
        } : null);
        setSearch('');
		setSource('inventory');
        setSubmitting(false);
        setSubmissionError('');
        setQuantityItem(null);
        setQuantity(1);

        const weaponSlot = isWeaponSlot(slotLabel);
        Promise.all([
            eligibleItemsForSlot(slotLabel),
            charIdx == null ? Promise.resolve<vm.CharacterViewModel | null>(null) : GetCharacter(charIdx),
            weaponSlot ? GetInfuseTypes() : Promise.resolve<db.InfuseType[]>([]),
            weaponSlot ? GetItemList('ashes_of_war') : Promise.resolve<db.ItemEntry[]>([]),
        ]).then(([items, character, infusions, ashes]) => {
            if (!active) return;
            setEligible(items);
            setOwned(character?.inventory ?? []);
            setInfuseTypes(infusions);
            setAshesOfWar(ashes);
            setLoading(false);
        }).catch(() => {
            if (!active) return;
            setEligible([]);
            setOwned([]);
            setInfuseTypes([]);
            setAshesOfWar([]);
            setLoading(false);
        });

        return () => { active = false; };
	}, [slotLabel, charIdx, initialSelection?.id, initialSelection?.handle, initialSelection?.memorySlots]);

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent) => {
            if (event.key !== 'Escape') return;
            if (quantityItem) {
                setQuantityItem(null);
                return;
            }
            onClose();
        };
        window.addEventListener('keydown', onKeyDown);
        return () => window.removeEventListener('keydown', onKeyDown);
    }, [onClose, quantityItem]);

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
        const aowNames = new Map(ashesOfWar.map(item => [item.id, item.name]));
        return owned
            .map(item => {
                const ownedID = item.baseId || item.id;
                const canonicalID = slotLabel.startsWith('Physick tear')
                    ? (PHYSICK_TEAR_CANONICAL_IDS.get(ownedID) ?? ownedID)
                    : ownedID;
                return { item, canonicalID, canonicalized: canonicalID !== ownedID };
            })
            .filter(({ canonicalID }) => eligibleByID.has(canonicalID))
            .map(({ item, canonicalID, canonicalized }, index) =>
                toOwnedPickerItem(item, eligibleByID.get(canonicalID), index, infuseTypes, aowNames, canonicalized ? canonicalID : undefined));
    }, [ashesOfWar, infuseTypes, owned, visibleEligible]);

    const items = useMemo(
        () => sortItems(
            (source === 'inventory' ? inventoryItems : visibleEligible.map(toPickerItem))
                .filter(item => matchesSearch(item, search)),
            sort,
        ),
        [inventoryItems, search, sort, source, visibleEligible],
    );
    const weaponList = view === 'list' && isWeaponSlot(slotLabel);
	const physickPicker = slotLabel.startsWith('Physick tear');
	const spellPicker = slotLabel.startsWith('Spell slot');
	// Spell capacity block: a candidate is unselectable when its Memory Slot cost
	// (added to what other slots already spend) would exceed capacity, or when its
	// cost is unknown — never silently assumed to be 1.
	const spellBlockReason = (item: PickerItem): string | undefined => {
		if (!spellPicker || spellCapacity == null) return undefined;
		if (item.memorySlots == null || item.memorySlots < 1) return 'No memory-cost data';
		const free = spellCapacity - spellUsedExcludingSelected;
		if (item.memorySlots > free) return `Needs ${item.memorySlots}, ${Math.max(0, free)} free`;
		return undefined;
	};
	const isDisabled = (item: PickerItem) =>
		disabledItemIDs.includes(item.id) ||
		(item.handle != null && disabledItemHandles.includes(item.handle)) ||
		spellBlockReason(item) != null;
	const selectItem = (item: PickerItem) => {
		const isCurrent = source === 'database'
			? initialSelection?.id === item.id
			: (spellPicker && initialSelection?.id === item.id) || (initialSelection?.handle != null && initialSelection.handle === item.handle);
		if (isCurrent) {
			onClear?.();
            onClose();
            return;
        }
        setSelected(item);
    };
    const submitSelection = async (item: PickerItem, selectedQuantity?: number) => {
        if (submitting) return;
        setSubmitting(true);
        setSubmissionError('');
        try {
            await onConfirm?.({
                id: item.id,
                handle: item.handle,
                name: item.name,
                iconPath: item.iconPath,
                quantity: selectedQuantity ?? item.quantity,
                memorySlots: item.memorySlots,
                source,
            });
            onClose();
        } catch (error) {
            setSubmissionError(error instanceof Error ? error.message : 'Unable to select this item.');
        } finally {
            setSubmitting(false);
        }
    };
    const commitSelection = (item: PickerItem) => {
        if (source === 'database' && item.stackable) {
            setQuantityItem(item);
            setQuantity(1);
            return;
        }
        void submitSelection(item, source === 'database' ? 1 : item.quantity);
    };
    const confirmSelection = () => {
        if (selected) commitSelection(selected);
    };
    const confirmQuantity = () => {
        if (!quantityItem) return;
        const normalized = Math.max(1, Math.min(quantityItem.maxInventory, Math.trunc(quantity || 1)));
        setQuantity(normalized);
        void submitSelection(quantityItem, normalized);
    };

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
                        {/* Physick tears are owned crystal tears only — never add one from the
                            Item Database — so the source is locked to Inventory for that slot. */}
                        {!physickPicker && <button type="button" aria-pressed={source === 'database'} onClick={() => setSource('database')} className={`h-full rounded px-3 text-[10px] font-black uppercase tracking-wider ${source === 'database' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'}`}>Item Database</button>}
                    </div>
                </div>

                <main className="min-h-0 flex-1 overflow-y-auto p-4 custom-scrollbar">
                    {loading ? <p className="py-12 text-center text-sm text-muted-foreground">Loading items…</p> : items.length === 0 ? <p className="py-12 text-center text-sm text-muted-foreground">No matching items.</p> : (
                        <div className={weaponList ? 'grid grid-cols-[minmax(0,1fr)_max-content_max-content_max-content] gap-x-3 gap-y-2' : view === 'icons' ? 'grid grid-cols-[repeat(auto-fill,minmax(125px,1fr))] gap-2' : 'space-y-2'}>
                            {weaponList && <><span className="px-2 text-[10px] font-black uppercase tracking-wider text-muted-foreground">Weapon</span><span className="text-right text-[10px] font-black uppercase tracking-wider text-muted-foreground">Level</span><span className="text-right text-[10px] font-black uppercase tracking-wider text-muted-foreground">Infuse</span><span className="text-right text-[10px] font-black uppercase tracking-wider text-muted-foreground">Ashes of War</span></>}
							{items.map(item => <ItemCard key={item.entryKey} item={item} source={source} view={view} weaponList={weaponList} physickPicker={physickPicker} selected={source === 'database' ? selected?.id === item.id : selected?.handle != null ? selected.handle === item.handle : selected?.id === item.id} disabled={isDisabled(item)} blockReason={spellBlockReason(item)} onSelect={selectItem} onActivate={commitSelection} />)}
                        </div>
                    )}
                </main>

                <div className="relative flex items-center justify-between gap-4 border-t border-border p-4">
                    <span className="min-w-0 truncate text-sm text-muted-foreground">{submissionError || (selected ? `Selected: ${selected.name}` : 'Select an item to preview it.')}</span>
                    {source === 'database' && <span className="absolute left-1/2 -translate-x-1/2 text-center text-xs font-bold text-primary">Items from Item Database will be added to Inventory before equipping.</span>}
                    <div className="flex shrink-0 gap-2">
                        <button type="button" disabled={submitting} onClick={onClose} className="rounded border border-border px-4 py-2 text-[10px] font-black uppercase tracking-widest text-muted-foreground hover:bg-muted/40 disabled:opacity-50">Cancel</button>
                        <button type="button" disabled={submitting || !selected} onClick={confirmSelection} className="rounded bg-primary px-4 py-2 text-[10px] font-black uppercase tracking-widest text-primary-foreground hover:opacity-90 disabled:opacity-50">Ok</button>
                    </div>
                </div>
            </div>
            {quantityItem && (
                <div className="fixed inset-0 z-[110] flex items-center justify-center bg-black/65 p-4" onMouseDown={() => setQuantityItem(null)}>
                    <div role="dialog" aria-modal="true" aria-label="Select item quantity" className="w-full max-w-sm rounded-xl border border-border bg-card p-5 text-card-foreground shadow-2xl" onMouseDown={(event) => event.stopPropagation()}>
                        <h3 className="text-base font-black">Add stackable item</h3>
                        <p className="mt-1 text-sm text-muted-foreground">Choose how many {quantityItem.name} items to add to Inventory.</p>
                        <label className="mt-4 block text-xs font-bold text-muted-foreground" htmlFor="equipment-item-quantity">Quantity</label>
                        <input
                            id="equipment-item-quantity"
                            aria-label="Item quantity"
                            type="number"
                            min={1}
                            max={quantityItem.maxInventory}
                            value={quantity}
                            autoFocus
                            onChange={(event) => setQuantity(Number(event.target.value))}
                            onKeyDown={(event) => { if (event.key === 'Enter') confirmQuantity(); }}
                            className="mt-1 h-10 w-full rounded-md border border-border bg-background px-3 text-foreground outline-none focus:border-primary"
                        />
                        <p className="mt-1 text-[11px] text-muted-foreground">Allowed range: 1–{quantityItem.maxInventory}</p>
                        {submissionError && <p role="alert" className="mt-2 text-xs font-bold text-red-600">{submissionError}</p>}
                        <div className="mt-5 flex justify-end gap-2">
                            <button type="button" disabled={submitting} onClick={() => setQuantityItem(null)} className="rounded border border-border px-4 py-2 text-[10px] font-black uppercase tracking-widest text-muted-foreground hover:bg-muted/40 disabled:opacity-50">Cancel</button>
                            <button type="button" disabled={submitting} onClick={confirmQuantity} className="rounded bg-primary px-4 py-2 text-[10px] font-black uppercase tracking-widest text-primary-foreground hover:opacity-90 disabled:opacity-50">Add and equip</button>
                        </div>
                    </div>
                </div>
            )}
        </div>,
        document.body,
    );
}
