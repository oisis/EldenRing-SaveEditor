import { useEffect, useState, type CSSProperties, type ReactNode, type SyntheticEvent } from 'react';
import { AddItemsToCharacter, GetCharacter, GetEquipmentSnapshot, SaveEquipment, SaveEquippedSpells, SaveQuickPouchItems } from '../../wailsjs/go/main/App';
import type { main, vm } from '../../wailsjs/go/models';
import { EquipmentItemPickerModal, type EquipmentPickerSelection } from './EquipmentItemPickerModal';

type EquippedItem = main.EquipmentSlotView;

const equipmentSlotByLabel: Record<string, number> = {
    'Ranged slot 1': 0,
    'Weapon slot 1': 1,
    'Ranged slot 2': 2,
    'Weapon slot 2': 3,
    'Ranged slot 3': 4,
    'Weapon slot 3': 5,
    'Arrow slot 1': 6,
    'Bolt slot 1': 7,
    'Arrow slot 2': 8,
    'Bolt slot 2': 9,
    'Knight Helm': 10,
    'Knight Armor': 11,
    'Knight Gauntlets': 12,
    'Knight Greaves': 13,
    'Axe Talisman': 14,
    'Claw Talisman': 15,
    'Companion Jar': 16,
    'Gold Scarab': 17,
};

const emptyEquipmentItem = (): EquippedItem => ({ occupied: false, rawId: 0, handle: 0, quantity: 0, name: '', iconPath: '', resolved: false });

// Item icon paths in the DB are stored without a leading slash; the public
// assets are served from the root, so normalize to an absolute path.
const iconSrc = (path: string) => (path.startsWith('/') ? path : `/${path}`);

const slotSurface: CSSProperties = {
    backgroundColor: 'var(--eq-slot-bg)',
    backgroundImage: 'var(--eq-slot-gradient)',
    boxShadow: 'var(--eq-slot-shadow)',
};

const slotClass = 'group relative flex h-[82px] w-[82px] items-center justify-center overflow-visible rounded-lg border border-[color:var(--eq-slot-border)] p-[5px] transition-colors hover:border-[color:var(--eq-slot-hover-border)]';
const ghostClass = 'h-[62px] w-[62px] object-contain opacity-[var(--eq-ghost-opacity)] grayscale contrast-75';
const toolsPlaceholder = '/equipment/tools-slot-placeholder.webp';

type SlotProps = {
    label: string;
    eligibleItems: string;
    onOpen: (label: string) => void;
    selected?: boolean;
    active?: boolean;
    readOnly?: boolean;
    showRemove?: boolean;
    item?: EquippedItem;
    quantity?: number;
    onRemove?: () => void;
    children: ReactNode;
};

function SlotTooltip({ eligibleItems }: { eligibleItems: string }) {
    return (
        <span
            role="tooltip"
            className="pointer-events-none absolute bottom-[calc(100%+7px)] left-1/2 z-30 w-max max-w-[170px] -translate-x-1/2 rounded-md bg-[color:var(--eq-tooltip-bg)] px-2 py-1 text-center text-[9px] font-bold leading-tight text-[color:var(--eq-tooltip-text)] opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100"
        >
            {eligibleItems}
        </span>
    );
}

function SlotRemoveIcon({ label, onRemove, showRemove = true }: { label: string; onRemove?: () => void; showRemove?: boolean }) {
    if (!showRemove) return null;
    if (!onRemove) return <span data-testid="slot-remove-icon" aria-hidden="true" className="pointer-events-none absolute bottom-0.5 left-1 z-20 text-lg font-black leading-none text-red-600 drop-shadow-sm">×</span>;
    const remove = (event: SyntheticEvent) => { event.stopPropagation(); onRemove(); };
    return <span data-testid="slot-remove-icon" role="button" tabIndex={0} aria-label={`Remove ${label}`} onClick={remove} onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') remove(event); }} className="absolute bottom-0.5 left-1 z-20 cursor-pointer text-lg font-black leading-none text-red-600 drop-shadow-sm hover:text-red-500">×</span>;
}

function EquipmentSlot({ label, eligibleItems, onOpen, selected = false, active = false, readOnly = false, showRemove = true, item, quantity, onRemove, children }: SlotProps) {
    // Occupied + resolved: real icon and item name. Occupied + unknown:
    // placeholder with the raw-ID name. Empty: placeholder + eligibility text.
    const tooltip = item?.occupied ? item.name : eligibleItems;
    return (
        <button
            type="button"
            aria-label={label}
            data-active-spell={active || undefined}
            onClick={() => { if (!readOnly) onOpen(label); }}
            style={selected || active ? { ...slotSurface, boxShadow: 'var(--eq-slot-selected-shadow)' } : slotSurface}
            className={`${slotClass} ${selected ? 'border-2 border-[color:var(--eq-slot-selected-border)]' : ''} ${active ? 'border-2 border-amber-400 ring-2 ring-amber-400/60' : ''}`}
        >
            <span className="pointer-events-none absolute inset-[5px] border border-[color:var(--eq-slot-inset-border)]" />
            <SlotTooltip eligibleItems={tooltip} />
            <SlotRemoveIcon label={label} onRemove={onRemove} showRemove={showRemove} />
            {quantity != null && <span className="pointer-events-none absolute left-1.5 top-1 z-20 text-xs font-black text-foreground">{quantity}</span>}
            {item?.resolved ? <ItemIcon src={iconSrc(item.iconPath)} alt={item.name} /> : children}
        </button>
    );
}

function GhostIcon({ src, alt = '', mirrored = false }: { src: string; alt?: string; mirrored?: boolean }) {
    return <img className={`relative z-10 ${ghostClass} ${mirrored ? '-scale-x-100' : ''}`} src={src} alt={alt} />;
}

function ItemIcon({ src, alt }: { src: string; alt: string }) {
    const [failed, setFailed] = useState(false);
    if (failed) return null;
    return <img className="relative z-10 h-[62px] w-[62px] object-contain drop-shadow-sm" src={src} alt={alt} onError={() => setFailed(true)} />;
}

function DPad({ active }: { active: 'up' | 'right' | 'down' | 'left' }) {
    const pieceClass = (direction: 'up' | 'right' | 'down' | 'left', position: string) =>
        `absolute h-[6px] w-[6px] rounded-[1px] border border-[color:var(--eq-dpad-color)] ${position} ${active === direction ? 'bg-[color:var(--eq-dpad-color)]' : 'bg-transparent'}`;

    return (
        <span className="pointer-events-none absolute left-1 top-1 z-20 h-5 w-5 drop-shadow-[0_1px_1px_var(--eq-dpad-shadow)]" aria-hidden="true">
            <i className={pieceClass('up', 'left-[7px] top-[1px]')} />
            <i className={pieceClass('right', 'right-0 top-[7px]')} />
            <i className={pieceClass('down', 'bottom-[1px] left-[7px]')} />
            <i className={pieceClass('left', 'left-0 top-[7px]')} />
        </span>
    );
}

function PouchPlaceholder() {
    return (
        <>
            <span style={{ boxShadow: 'var(--eq-pouch-shadow)' }} className="pointer-events-none absolute left-1/2 top-[27px] h-[35px] w-[38px] -translate-x-1/2 rounded-[9px_9px_13px_13px] bg-[color:var(--eq-pouch-fill)] opacity-20" />
            <span className="pointer-events-none absolute left-1/2 top-[19px] z-10 h-[13px] w-[24px] -translate-x-1/2 rounded-t-[13px] border-[4px] border-b-0 border-[color:var(--eq-pouch-line)]" />
        </>
    );
}

function PouchSlot({ label, active, onOpen, item, onRemove }: { label: string; active?: 'up' | 'right' | 'down' | 'left'; onOpen: (label: string) => void; item?: EquippedItem; onRemove?: () => void }) {
    const tooltip = item?.occupied ? item.name : 'Tools and Spirit Ashes';
    return (
        <button
            type="button"
            aria-label={label}
            onClick={() => onOpen(label)}
            style={slotSurface}
            className="group relative flex h-[82px] w-[82px] items-center justify-center overflow-visible rounded-lg border border-[color:var(--eq-slot-border)] p-[5px] hover:border-[color:var(--eq-slot-hover-border)]"
        >
            <span className="pointer-events-none absolute inset-[5px] border border-[color:var(--eq-slot-inset-border)]" />
            <SlotTooltip eligibleItems={tooltip} />
            <SlotRemoveIcon label={label} onRemove={onRemove} />
            {item?.occupied && <span className="pointer-events-none absolute right-1.5 top-1 z-20 text-xs font-black text-foreground">{item.quantity}</span>}
            {active && <DPad active={active} />}
            {item?.resolved ? <ItemIcon src={iconSrc(item.iconPath)} alt={item.name} /> : <PouchPlaceholder />}
        </button>
    );
}

function PhysickSlot({ label, onOpen, item }: { label: string; onOpen: (label: string) => void; item?: EquippedItem }) {
    // Occupied + resolved: real icon and tear name. Occupied + unknown: the raw-ID
    // name, never the empty placeholder (the native empty-mixture encoding is
    // unconfirmed). Empty: placeholder + eligibility text.
    const occupied = item?.occupied ?? false;
    const resolved = item?.resolved ?? false;
    const tooltip = occupied ? item!.name : 'Crystal Tears';
    return (
        <button
            type="button"
            aria-label={label}
            onClick={() => onOpen(label)}
            style={slotSurface}
            className="group relative flex h-[82px] w-[82px] flex-col rounded-lg border border-[color:var(--eq-slot-border)] p-[5px] text-[color:var(--eq-physick-text)] hover:border-[color:var(--eq-slot-hover-border)]"
        >
            <SlotTooltip eligibleItems={tooltip} />
            <SlotRemoveIcon label={label} />
            <span className="line-clamp-2 min-h-[21px] text-center text-[8px] font-extrabold leading-[1.15]">{occupied ? item!.name : 'Physick tear'}</span>
            <span className="flex h-[51px] items-center justify-center">
                {resolved ? (
                    <ItemIcon src={iconSrc(item!.iconPath)} alt={item!.name} />
                ) : occupied ? (
                    <span data-testid="physick-unknown" className="flex h-[51px] w-[51px] items-center justify-center text-2xl font-black text-[color:var(--eq-physick-text)]">?</span>
                ) : (
                    <img className="h-[51px] w-[51px] object-contain opacity-[var(--eq-ghost-opacity)]" src="/equipment/physick-tear-placeholder.png" alt="" />
                )}
            </span>
        </button>
    );
}

export function EquipmentTab({ charIdx, saveLoadKey, equipmentRevision, onMutate }: { charIdx?: number; saveLoadKey?: number; equipmentRevision?: number; onMutate?: () => void } = {}) {
    const [selectedSlot, setSelectedSlot] = useState('Weapon slot 1');
    const [modalOpen, setModalOpen] = useState(false);
    const [snapshot, setSnapshot] = useState<main.EquipmentSnapshot | null>(null);
    const [ammoQuantities, setAmmoQuantities] = useState<Record<number, number>>({});
    const [draftEquipment, setDraftEquipment] = useState<Record<number, EquippedItem>>({});
    const [equipmentDirty, setEquipmentDirty] = useState(false);
    const [draftQuickPouch, setDraftQuickPouch] = useState<Record<number, EquippedItem>>({});
    const [quickPouchDirty, setQuickPouchDirty] = useState(false);
    const [draftSpells, setDraftSpells] = useState<EquippedItem[] | null>(null);
    const [spellsDirty, setSpellsDirty] = useState(false);
    const [saveError, setSaveError] = useState('');
    const [saveRevision, setSaveRevision] = useState(0);
    const openSlot = (label: string) => {
        setSelectedSlot(label);
        setModalOpen(true);
    };
    const selected = (label: string) => selectedSlot === label;

    // Read-only load of the equipped items for the selected character. Guard
    // against stale updates when the character changes or the tab unmounts.
    useEffect(() => {
        if (charIdx == null) return;
        let active = true;
        setSnapshot(null);
        setAmmoQuantities({});
        setDraftEquipment({});
        setEquipmentDirty(false);
        setDraftQuickPouch({});
        setQuickPouchDirty(false);
        setDraftSpells(null);
        setSpellsDirty(false);
        setSaveError('');
        (async () => {
            try {
                const [snap, character] = await Promise.all([GetEquipmentSnapshot(charIdx), GetCharacter(charIdx)]);
                if (!active) return;
                const quantities: Record<number, number> = {};
                for (const item of character.inventory ?? []) {
                    if (item.subCategory !== 'arrows_and_bolts') continue;
                    const quantity = item.quantity ?? 0;
                    quantities[item.id] = (quantities[item.id] ?? 0) + quantity;
                    if (item.baseId && item.baseId !== item.id) quantities[item.baseId] = (quantities[item.baseId] ?? 0) + quantity;
                }
                setSnapshot(snap);
                setDraftSpells(Array.from(snap.spells ?? []));
                setAmmoQuantities(quantities);
            } catch {
                if (active) {
                    setSnapshot(null);
                    setAmmoQuantities({});
                }
            }
        })();
        return () => { active = false; };
    }, [charIdx, saveLoadKey, equipmentRevision, saveRevision]);

    const spellIndexForLabel = (label: string) => {
        const match = /^Spell slot (\d+)$/.exec(label);
        return match ? Number(match[1]) - 1 : -1;
    };

    const ensureDatabaseItemInInventory = async (itemID: number, quantity: number, excludedHandles: number[] = [], requireHandle = false) => {
        if (charIdx == null) throw new Error('Select a character before adding an item.');
        const findOwnedItem = (inventory: vm.ItemViewModel[]) => inventory?.find(item => {
            const canonicalID = item.baseId || item.id;
            if (canonicalID !== itemID || (item.quantity ?? 0) <= 0) return false;
            return !requireHandle || (item.handle !== 0 && !excludedHandles.includes(item.handle));
        });

        const result = await AddItemsToCharacter(charIdx, [itemID], 0, 0, 0, 0, quantity, 0);
        if (result.capHit) throw new Error(result.capHit);
        const character = await GetCharacter(charIdx);
        const ownedItem = findOwnedItem(character.inventory);
        if (!ownedItem) throw new Error('The item could not be added to Inventory.');
        return ownedItem;
    };

    const setSpellSelection = async (selection: EquipmentPickerSelection) => {
        const index = spellIndexForLabel(selectedSlot);
        if (index < 0) return;
        if (selection.source === 'database') await ensureDatabaseItemInInventory(selection.id, selection.quantity ?? 1);
        setDraftSpells(current => {
            const next = [...(current ?? Array.from(snapshot?.spells ?? []))];
            next[index] = { occupied: true, rawId: selection.id & 0x0FFFFFFF, handle: 0, quantity: 1, name: selection.name, iconPath: selection.iconPath, resolved: true };
            return next;
        });
        setSpellsDirty(true);
        setSaveError('');
    };
    const removeSpell = (index: number) => {
        setDraftSpells(current => {
            const next = [...(current ?? Array.from(snapshot?.spells ?? []))];
            const compact = next.filter((item, itemIndex) => itemIndex !== index && item?.occupied);
            while (compact.length < next.length) compact.push(emptyEquipmentItem());
            return compact;
        });
        setSpellsDirty(true);
        setSaveError('');
    };
    const selectedEquipmentSlot = equipmentSlotByLabel[selectedSlot];
    const equipmentView = (slot: number, fallback: EquippedItem | undefined): EquippedItem | undefined =>
        Object.prototype.hasOwnProperty.call(draftEquipment, slot) ? draftEquipment[slot] : fallback;
    const removeEquipment = (slot: number) => {
        setDraftEquipment(current => ({ ...current, [slot]: emptyEquipmentItem() }));
        setEquipmentDirty(true);
        setSaveError('');
    };
    const quickPouchSlotForLabel = (label: string) => {
        const quick = /^Quick item (\d+)$/.exec(label);
        if (quick) return Number(quick[1]) - 1;
        const pouch: Record<string, number> = {
            'Quick pouch up': 10,
            'Quick pouch right': 11,
            'Quick pouch left': 12,
            'Quick pouch down': 13,
            'Quick pouch slot 5': 14,
            'Quick pouch slot 6': 15,
        };
        return pouch[label] ?? -1;
    };
    const quickPouchSnapshotForSlot = (slot: number): EquippedItem | undefined =>
        slot < 10 ? snapshot?.quickItems[slot] : snapshot?.pouch[slot - 10];
    const quickPouchView = (slot: number): EquippedItem | undefined =>
        Object.prototype.hasOwnProperty.call(draftQuickPouch, slot) ? draftQuickPouch[slot] : quickPouchSnapshotForSlot(slot);
    const removeQuickPouch = (slot: number) => {
        setDraftQuickPouch(current => ({ ...current, [slot]: emptyEquipmentItem() }));
        setQuickPouchDirty(true);
        setSaveError('');
    };
    const saveChanges = async () => {
        if (charIdx == null) return;
        try {
            if (equipmentDirty) {
                const changes = Object.entries(draftEquipment).map(([slot, item]) => ({
                    slot: Number(slot),
                    handle: item.occupied ? item.handle : 0,
                }));
                await SaveEquipment(charIdx, changes);
                setEquipmentDirty(false);
            }
            if (spellsDirty && draftSpells) {
                const spellIDs = draftSpells.filter(item => item.occupied).map(item => item.rawId | 0x40000000);
                await SaveEquippedSpells(charIdx, spellIDs);
                setSpellsDirty(false);
            }
            if (quickPouchDirty) {
                const changes = Object.entries(draftQuickPouch).map(([slot, item]) => ({
                    slot: Number(slot),
                    handle: item.occupied ? item.handle : 0,
                }));
                await SaveQuickPouchItems(charIdx, changes);
                setQuickPouchDirty(false);
            }
            setSaveError('');
            setSaveRevision(value => value + 1);
            onMutate?.();
        } catch (error) {
            setSaveError(error instanceof Error ? error.message : 'Unable to save equipment changes.');
        }
    };

    const weaponSlots = ['Weapon slot 1', 'Weapon slot 2', 'Weapon slot 3'];
    const rangedSlots = ['Ranged slot 1', 'Ranged slot 2', 'Ranged slot 3'];
    const armorSlots = [
        ['Knight Helm', '/items/head/knight_helm.png'],
        ['Knight Armor', '/items/chest/knight_armor.png'],
        ['Knight Gauntlets', '/items/arms/knight_gauntlets.png'],
        ['Knight Greaves', '/items/legs/knight_greaves.png'],
    ] as const;
    const talismanSlots = [
        ['Axe Talisman', '/items/talismans/axe_talisman.png'],
        ['Claw Talisman', '/items/talismans/claw_talisman.png'],
        ['Companion Jar', '/items/talismans/companion_jar.png'],
        ['Gold Scarab', '/items/talismans/gold_scarab.png'],
    ] as const;
    // Mixed sorceries and incantations give every empty spell field a distinct
    // in-game silhouette while keeping the icon in the same subdued placeholder
    // treatment as the rest of the Equipment screen.
    const spellSlots = [
        ['Spell slot 1', '/items/sorceries/comet_azur.png'],
        ['Spell slot 2', '/items/incantations/dragonfire.png'],
        ['Spell slot 3', '/items/sorceries/carian_slicer.png'],
        ['Spell slot 4', '/items/incantations/scarlet_aeonia.png'],
        ['Spell slot 5', '/items/sorceries/rock_sling.png'],
        ['Spell slot 6', '/items/incantations/lightning_spear.png'],
        ['Spell slot 7', '/items/sorceries/rennalas_full_moon.png'],
        ['Spell slot 8', '/items/incantations/black_flame.png'],
        ['Spell slot 9', '/items/sorceries/oracle_bubbles.png'],
        ['Spell slot 10', '/items/incantations/heal.png'],
        ['Spell slot 11', '/items/sorceries/founding_rain_of_stars.png'],
        ['Spell slot 12', '/items/incantations/frenzied_burst.png'],
    ] as const;
    // A save can unlock one to four talisman slots. Until the read-only snapshot
    // arrives, retain the mockup's four-slot layout; once loaded, locked slots
    // disappear from the right and cannot be opened.
    const activeTalismanSlots = snapshot?.activeTalismanSlots ?? talismanSlots.length;
    const activeSpellSlots = snapshot?.activeSpellSlots ?? 10;
    const primarySpellSlots = spellSlots.slice(0, Math.min(activeSpellSlots, 10));
    const moonSpellSlots = activeSpellSlots > 10 ? spellSlots.slice(10, activeSpellSlots) : [];
    const currentEquipLoad = snapshot?.equipLoadKnown ? snapshot.currentEquipLoad.toFixed(1) : 'N';
    const maxEquipLoad = snapshot ? snapshot.maxEquipLoad.toFixed(1) : 'N';
    const equipLoadClass = snapshot?.equipLoadKnown ? snapshot.equipLoadClass : 'Unknown';
    const equipLoadClassStyle = {
        Light: 'text-emerald-600',
        Medium: 'text-orange-500',
        Heavy: 'text-red-600',
        Overloaded: 'text-red-600 font-black',
    }[equipLoadClass] ?? 'text-muted-foreground';
    const spellViews = draftSpells ?? Array.from(snapshot?.spells ?? []);
    const equippedSpellCount = spellViews.filter(item => item.occupied).length;
    const activeSpellIndex = snapshot && equippedSpellCount > 0
        ? (snapshot.activeSpellIndex >= equippedSpellCount ? 0 : snapshot.activeSpellIndex)
        : -1;
    const selectedSpellIndex = spellIndexForLabel(selectedSlot);
    const selectedSpell = selectedSpellIndex >= 0 ? spellViews[selectedSpellIndex] : undefined;
    const selectedSpellSelection = selectedSpell?.occupied ? {
        id: selectedSpell.rawId | 0x40000000,
        name: selectedSpell.name,
        iconPath: selectedSpell.iconPath,
        source: 'inventory' as const,
    } : undefined;
    const disabledSpellIDs = spellViews
        .filter((item, index) => index !== selectedSpellIndex && item.occupied)
        .map(item => item.rawId | 0x40000000);
    const snapshotEquipmentForSlot = (slot: number): EquippedItem | undefined => {
        if (slot >= 0 && slot <= 5) {
            const handIndex = Math.floor(slot / 2);
            return slot % 2 === 0 ? snapshot?.leftHandArmaments[handIndex] : snapshot?.rightHandArmaments[handIndex];
        }
        if (slot === 6) return snapshot?.arrows[0];
        if (slot === 8) return snapshot?.arrows[1];
        if (slot === 7) return snapshot?.bolts[0];
        if (slot === 9) return snapshot?.bolts[1];
        if (slot >= 10 && slot <= 13) return snapshot?.armor[slot - 10];
        if (slot >= 14 && slot <= 17) return snapshot?.talismans[slot - 14];
        return undefined;
    };
    const selectedEquipment = selectedEquipmentSlot == null ? undefined : equipmentView(selectedEquipmentSlot, snapshotEquipmentForSlot(selectedEquipmentSlot));
    const selectedEquipmentSelection = selectedEquipment?.occupied ? {
        id: selectedEquipment.rawId,
        handle: selectedEquipment.handle,
        name: selectedEquipment.name,
        iconPath: selectedEquipment.iconPath,
        source: 'inventory' as const,
    } : undefined;
    const disabledEquipmentHandles = Array.from({ length: 18 }, (_, slot) => slot)
        .filter(slot => slot !== selectedEquipmentSlot)
        .map(slot => equipmentView(slot, snapshotEquipmentForSlot(slot)))
        .filter((item): item is EquippedItem => Boolean(item?.occupied && item.handle))
        .map(item => item.handle);
    const disabledTalismanIDs = selectedEquipmentSlot != null && selectedEquipmentSlot >= 14 && selectedEquipmentSlot <= 17
        ? Array.from({ length: activeTalismanSlots }, (_, index) => index + 14)
            .filter(slot => slot !== selectedEquipmentSlot)
            .map(slot => equipmentView(slot, snapshotEquipmentForSlot(slot)))
            .filter((item): item is EquippedItem => Boolean(item?.occupied))
            .map(item => item.rawId)
        : [];
    const selectedQuickPouchSlot = quickPouchSlotForLabel(selectedSlot);
    const selectedQuickPouch = selectedQuickPouchSlot >= 0 ? quickPouchView(selectedQuickPouchSlot) : undefined;
    const selectedQuickPouchSelection = selectedQuickPouch?.occupied ? {
        id: 0x40000000 | (selectedQuickPouch.rawId & 0x0FFFFFFF),
        handle: selectedQuickPouch.handle,
        name: selectedQuickPouch.name,
        iconPath: selectedQuickPouch.iconPath,
        quantity: selectedQuickPouch.quantity,
        source: 'inventory' as const,
    } : undefined;
    const quickPouchFamilyStart = selectedQuickPouchSlot < 10 ? 0 : 10;
    const quickPouchFamilyEnd = selectedQuickPouchSlot < 10 ? 10 : 16;
    const disabledQuickPouchItems = selectedQuickPouchSlot >= 0
        ? Array.from({ length: quickPouchFamilyEnd - quickPouchFamilyStart }, (_, index) => quickPouchFamilyStart + index)
            .filter(slot => slot !== selectedQuickPouchSlot)
            .map(slot => quickPouchView(slot))
            .filter((item): item is EquippedItem => Boolean(item?.occupied))
        : [];
    const disabledQuickPouchHandles = disabledQuickPouchItems.map(item => item.handle).filter(Boolean);
    const disabledQuickPouchIDs = disabledQuickPouchItems.map(item => 0x40000000 | (item.rawId & 0x0FFFFFFF));
    const disabledPickerItemIDs = selectedSpellIndex >= 0 ? disabledSpellIDs : selectedQuickPouchSlot >= 0 ? disabledQuickPouchIDs : disabledTalismanIDs;
    const disabledPickerItemHandles = selectedQuickPouchSlot >= 0 ? disabledQuickPouchHandles : disabledEquipmentHandles;
    const setEquipmentSelection = async (selection: EquipmentPickerSelection) => {
        if (selectedEquipmentSlot == null) return;
        let handle = selection.handle;
        let quantity = selection.quantity ?? 1;
        if (selection.source === 'database') {
            const ownedItem = await ensureDatabaseItemInInventory(selection.id, quantity, disabledEquipmentHandles, true);
            handle = ownedItem.handle;
            quantity = ownedItem.quantity;
        }
        if (handle == null || handle === 0) throw new Error('The selected item has no writable Inventory handle.');
        setDraftEquipment(current => ({
            ...current,
            [selectedEquipmentSlot]: {
                occupied: true,
                rawId: selection.id,
                handle,
                quantity,
                name: selection.name,
                iconPath: selection.iconPath,
                resolved: true,
            },
        }));
        setEquipmentDirty(true);
        setSaveError('');
    };
    const setQuickPouchSelection = async (selection: EquipmentPickerSelection) => {
        if (selectedQuickPouchSlot < 0) return;
        let handle = selection.handle;
        let quantity = selection.quantity ?? 1;
        if (selection.source === 'database') {
            const ownedItem = await ensureDatabaseItemInInventory(selection.id, quantity, disabledQuickPouchHandles, true);
            handle = ownedItem.handle;
            quantity = ownedItem.quantity;
        }
        if (handle == null || handle === 0) throw new Error('The selected item has no writable Inventory handle.');
        const writableHandle = handle;
        setDraftQuickPouch(current => ({
            ...current,
            [selectedQuickPouchSlot]: {
                occupied: true,
                rawId: writableHandle,
                handle: writableHandle,
                quantity,
                name: selection.name,
                iconPath: selection.iconPath,
                resolved: true,
            },
        }));
        setQuickPouchDirty(true);
        setSaveError('');
    };

    return (
        <section className="w-full shrink-0 overflow-auto rounded-xl border border-border bg-card text-card-foreground shadow-sm custom-scrollbar">
            <div className="mx-auto grid w-fit grid-cols-[499px_255px_199px] px-5 py-5">
                <div>
                    <h2 className="mb-3 text-center text-[10px] font-black uppercase tracking-[0.15em] text-muted-foreground">Equipment slots</h2>
                    <div className="grid gap-[10px]">
                        <div className="grid grid-cols-[repeat(3,82px)_18px_repeat(2,82px)] gap-[9px]">
                            {weaponSlots.map((label, index) => {
                                const slot = 1 + index * 2;
                                const item = equipmentView(slot, snapshot?.rightHandArmaments[index]);
                                return <EquipmentSlot key={label} label={label} eligibleItems="Weapons, shields, staves, seals and torches" selected={selected(label)} onOpen={openSlot} item={item} onRemove={item?.occupied ? () => removeEquipment(slot) : undefined}><GhostIcon src="/equipment/weapon-slot-placeholder.png" /></EquipmentSlot>;
                            })}
                            <span aria-hidden="true" />
                            {['Arrow slot 1', 'Arrow slot 2'].map((label, index) => {
                                const slot = index === 0 ? 6 : 8;
                                const item = equipmentView(slot, snapshot?.arrows[index]);
                                return <EquipmentSlot key={label} label={label} eligibleItems="Arrows and greatarrows" selected={selected(label)} onOpen={openSlot} item={item} quantity={item?.occupied ? (item.quantity || ammoQuantities[item.rawId]) : undefined} onRemove={item?.occupied ? () => removeEquipment(slot) : undefined}><GhostIcon src="/items/arrows_and_bolts/arrow.png" mirrored /></EquipmentSlot>;
                            })}
                        </div>
                        <div className="-mt-[7px] grid grid-cols-[repeat(3,82px)_18px_repeat(2,82px)] gap-[9px]">
                            {rangedSlots.map((label, index) => {
                                const slot = index * 2;
                                const item = equipmentView(slot, snapshot?.leftHandArmaments[index]);
                                return <EquipmentSlot key={label} label={label} eligibleItems="Weapons, shields, staves, seals and torches" selected={selected(label)} onOpen={openSlot} item={item} onRemove={item?.occupied ? () => removeEquipment(slot) : undefined}><GhostIcon src="/equipment/ranged-slot-placeholder.png" /></EquipmentSlot>;
                            })}
                            <span aria-hidden="true" />
                            {['Bolt slot 1', 'Bolt slot 2'].map((label, index) => {
                                const slot = index === 0 ? 7 : 9;
                                const item = equipmentView(slot, snapshot?.bolts[index]);
                                return <EquipmentSlot key={label} label={label} eligibleItems="Bolts and greatbolts" selected={selected(label)} onOpen={openSlot} item={item} quantity={item?.occupied ? (item.quantity || ammoQuantities[item.rawId]) : undefined} onRemove={item?.occupied ? () => removeEquipment(slot) : undefined}><GhostIcon src="/items/arrows_and_bolts/bolt.png" /></EquipmentSlot>;
                            })}
                        </div>
                        <div className="mt-[5px] grid grid-cols-[repeat(4,82px)] gap-[9px]">
                            {armorSlots.map(([label, src], index) => {
                                const slot = 10 + index;
                                const item = equipmentView(slot, snapshot?.armor[index]);
                                return <EquipmentSlot key={label} label={label} eligibleItems={['Helms', 'Chest armor', 'Gauntlets', 'Leg armor'][index]} selected={selected(label)} onOpen={openSlot} item={item} onRemove={item?.occupied ? () => removeEquipment(slot) : undefined}><GhostIcon src={src} /></EquipmentSlot>;
                            })}
                        </div>
                        <div className="grid grid-cols-[repeat(4,82px)] gap-[9px]">
                            {talismanSlots.slice(0, activeTalismanSlots).map(([label, src], index) => {
                                const slot = 14 + index;
                                const item = equipmentView(slot, snapshot?.talismans[index]);
                                return <EquipmentSlot key={label} label={label} eligibleItems="Talismans" selected={selected(label)} onOpen={openSlot} item={item} onRemove={item?.occupied ? () => removeEquipment(slot) : undefined}><GhostIcon src={src} /></EquipmentSlot>;
                            })}
                        </div>
                        {[0, 1].map((row) => (
                            <div key={row} className="grid grid-cols-[repeat(5,82px)] gap-[9px]">
                                {Array.from({ length: 5 }, (_, index) => {
                                    const slotIndex = row * 5 + index;
                                    const label = `Quick item ${slotIndex + 1}`;
                                    const item = quickPouchView(slotIndex);
                                    return <EquipmentSlot key={label} label={label} eligibleItems="Tools and Spirit Ashes" selected={selected(label)} onOpen={openSlot} item={item} quantity={item?.occupied ? item.quantity : undefined} onRemove={item?.occupied ? () => removeQuickPouch(slotIndex) : undefined}><GhostIcon src={toolsPlaceholder} /></EquipmentSlot>;
                                })}
                            </div>
                        ))}
                    </div>
                </div>

                <div className="border-l border-border pl-[26px]">
                    <h2 className="mb-3 w-[173px] text-center text-[10px] font-black uppercase tracking-[0.15em] text-muted-foreground">Quick pouch</h2>
                    <div className="grid grid-cols-[82px_82px] gap-[9px]">
                        <PouchSlot label="Quick pouch up" active="up" onOpen={openSlot} item={quickPouchView(10)} onRemove={quickPouchView(10)?.occupied ? () => removeQuickPouch(10) : undefined} />
                        <PouchSlot label="Quick pouch right" active="right" onOpen={openSlot} item={quickPouchView(11)} onRemove={quickPouchView(11)?.occupied ? () => removeQuickPouch(11) : undefined} />
                        <PouchSlot label="Quick pouch left" active="left" onOpen={openSlot} item={quickPouchView(12)} onRemove={quickPouchView(12)?.occupied ? () => removeQuickPouch(12) : undefined} />
                        <PouchSlot label="Quick pouch down" active="down" onOpen={openSlot} item={quickPouchView(13)} onRemove={quickPouchView(13)?.occupied ? () => removeQuickPouch(13) : undefined} />
                        <PouchSlot label="Quick pouch slot 5" onOpen={openSlot} item={quickPouchView(14)} onRemove={quickPouchView(14)?.occupied ? () => removeQuickPouch(14) : undefined} />
                        <PouchSlot label="Quick pouch slot 6" onOpen={openSlot} item={quickPouchView(15)} onRemove={quickPouchView(15)?.occupied ? () => removeQuickPouch(15) : undefined} />
                    </div>
                    <div aria-hidden="true" className="mt-[6px] h-[14px]" />
                    <h3 className="mb-3 w-[173px] whitespace-pre-line text-center text-[10px] font-black uppercase tracking-[0.1em] text-muted-foreground">{'Wondrous\nPhysick flask'}</h3>
                    <div className="grid grid-cols-[82px_82px] gap-[9px]">
                        <PhysickSlot label="Physick tear 1" onOpen={openSlot} item={snapshot?.physick[0]} />
                        <PhysickSlot label="Physick tear 2" onOpen={openSlot} item={snapshot?.physick[1]} />
                    </div>
                </div>

                <div className="flex h-full flex-col border-l border-border pl-[26px]">
                    <h2 className="mb-3 w-[173px] text-center text-[10px] font-black uppercase tracking-[0.15em] text-muted-foreground">Spell slots</h2>
                    <div data-testid="spell-slot-area" className={`flex flex-1 flex-col ${moonSpellSlots.length ? 'justify-start' : 'justify-center'}`}>
                        <div data-testid="spell-primary-grid" className="grid grid-flow-col grid-cols-[repeat(2,82px)] grid-rows-[repeat(5,82px)] gap-[9px]">
                            {primarySpellSlots.map(([label, src], index) => <EquipmentSlot key={label} label={label} eligibleItems="Sorceries and Incantations" selected={selected(label)} active={index === activeSpellIndex} onOpen={openSlot} item={spellViews[index]} onRemove={spellViews[index]?.occupied ? () => removeSpell(index) : undefined}><GhostIcon src={src} /></EquipmentSlot>)}
                        </div>
                        {moonSpellSlots.length > 0 && (
                            <div className="mt-[9px] grid grid-cols-[repeat(2,82px)] gap-[9px]">
                                {moonSpellSlots.map(([label, src], offset) => {
                                    const index = offset + 10;
                                    return <EquipmentSlot key={label} label={label} eligibleItems="Sorceries and Incantations" selected={selected(label)} active={index === activeSpellIndex} onOpen={openSlot} item={spellViews[index]} onRemove={spellViews[index]?.occupied ? () => removeSpell(index) : undefined}><GhostIcon src={src} /></EquipmentSlot>;
                                })}
                            </div>
                        )}
                    </div>
                </div>
            </div>
            <div className="mx-5 flex items-center justify-between border-t border-border px-0 pb-4 pt-3">
                <span className="text-[12px] font-extrabold tracking-[.04em] text-muted-foreground">Equip Load (<span className={equipLoadClassStyle}>{equipLoadClass}</span>): <strong className="text-foreground">{currentEquipLoad} / {maxEquipLoad}</strong></span>
                <strong className="text-base text-red-600">Expermiental</strong>
                <span role="status" className="text-xs text-red-600">{saveError}</span>
                <button type="button" disabled={(!spellsDirty && !equipmentDirty && !quickPouchDirty) || charIdx == null} onClick={saveChanges} className="rounded-md bg-primary px-4 py-2 text-[10px] font-black uppercase tracking-[.13em] text-primary-foreground hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50">Save changes</button>
            </div>
            {modalOpen && <EquipmentItemPickerModal
                slotLabel={selectedSlot}
                charIdx={charIdx}
                initialSelection={selectedSlot.startsWith('Spell slot') ? selectedSpellSelection : selectedQuickPouchSlot >= 0 ? selectedQuickPouchSelection : selectedEquipmentSelection}
                disabledItemIDs={disabledPickerItemIDs}
                disabledItemHandles={disabledPickerItemHandles}
                onConfirm={selectedSlot.startsWith('Spell slot') ? setSpellSelection : selectedQuickPouchSlot >= 0 ? setQuickPouchSelection : selectedEquipmentSlot != null ? setEquipmentSelection : undefined}
                onClear={selectedSlot.startsWith('Spell slot') ? (selectedSpellIndex >= 0 ? () => removeSpell(selectedSpellIndex) : undefined) : selectedQuickPouchSlot >= 0 ? () => removeQuickPouch(selectedQuickPouchSlot) : selectedEquipmentSlot != null ? () => removeEquipment(selectedEquipmentSlot) : undefined}
                onClose={() => setModalOpen(false)}
            />}
        </section>
    );
}
