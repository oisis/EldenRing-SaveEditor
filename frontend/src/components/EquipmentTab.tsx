import { useState, type CSSProperties, type ReactNode } from 'react';

const slotSurface: CSSProperties = {
    backgroundColor: '#f8fafc',
    backgroundImage: 'radial-gradient(ellipse at 50% 45%, rgba(116,132,147,.13), transparent 62%), repeating-linear-gradient(34deg, rgba(94,109,126,.025) 0 1px, transparent 1px 5px), linear-gradient(145deg, #ffffff, #edf2f6 75%)',
    boxShadow: 'inset 0 0 0 2px rgba(255,255,255,.65), inset 0 0 12px rgba(111,127,142,.09), 0 2px 4px #d4d9de',
};

const slotClass = 'group relative flex h-[82px] w-[82px] items-center justify-center overflow-visible rounded-lg border border-[#d2dce6] p-[5px] transition-colors hover:border-[#75c994]';
const ghostClass = 'h-[62px] w-[62px] object-contain opacity-[.22] grayscale contrast-75';
const toolsPlaceholder = '/equipment/tools-slot-placeholder.webp';

type SlotProps = {
    label: string;
    eligibleItems: string;
    onOpen: (label: string) => void;
    selected?: boolean;
    children: ReactNode;
};

function SlotTooltip({ eligibleItems }: { eligibleItems: string }) {
    return (
        <span
            role="tooltip"
            className="pointer-events-none absolute bottom-[calc(100%+7px)] left-1/2 z-30 w-max max-w-[170px] -translate-x-1/2 rounded-md bg-[#303a47] px-2 py-1 text-center text-[9px] font-bold leading-tight text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100"
        >
            {eligibleItems}
        </span>
    );
}

function EquipmentSlot({ label, eligibleItems, onOpen, selected = false, children }: SlotProps) {
    return (
        <button
            type="button"
            aria-label={label}
            onClick={() => onOpen(label)}
            style={slotSurface}
            className={`${slotClass} ${selected ? 'border-2 border-[#37b66c] shadow-[inset_0_0_0_2px_rgba(255,255,255,.65),inset_0_0_12px_rgba(111,127,142,.09),0_0_0_2px_#d3f5de]' : ''}`}
        >
            <span className="pointer-events-none absolute inset-[5px] border border-[#8e9ba9]/15" />
            <SlotTooltip eligibleItems={eligibleItems} />
            {children}
        </button>
    );
}

function GhostIcon({ src, alt = '', mirrored = false }: { src: string; alt?: string; mirrored?: boolean }) {
    return <img className={`relative z-10 ${ghostClass} ${mirrored ? '-scale-x-100' : ''}`} src={src} alt={alt} />;
}

function DPad({ active }: { active: 'up' | 'right' | 'down' | 'left' }) {
    const pieceClass = (direction: 'up' | 'right' | 'down' | 'left', position: string) =>
        `absolute h-[6px] w-[6px] rounded-[1px] border border-[#596776] ${position} ${active === direction ? 'bg-[#596776]' : 'bg-transparent'}`;

    return (
        <span className="pointer-events-none absolute left-1 top-1 z-20 h-5 w-5 drop-shadow-[0_1px_1px_#c1c8d0]" aria-hidden="true">
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
            <span className="pointer-events-none absolute left-1/2 top-[27px] h-[35px] w-[38px] -translate-x-1/2 rounded-[9px_9px_13px_13px] bg-[#667586] opacity-20 shadow-[inset_0_5px_0_rgba(255,255,255,.18),inset_0_-5px_0_rgba(56,68,81,.2)]" />
            <span className="pointer-events-none absolute left-1/2 top-[19px] z-10 h-[13px] w-[24px] -translate-x-1/2 rounded-t-[13px] border-[4px] border-b-0 border-[#667586]/25" />
        </>
    );
}

function PouchSlot({ label, active, onOpen }: { label: string; active?: 'up' | 'right' | 'down' | 'left'; onOpen: (label: string) => void }) {
    return (
        <button
            type="button"
            aria-label={label}
            onClick={() => onOpen(label)}
            style={slotSurface}
            className="group relative flex h-[82px] w-[82px] items-center justify-center overflow-visible rounded-lg border border-[#d2dce6] p-[5px] hover:border-[#75c994]"
        >
            <span className="pointer-events-none absolute inset-[5px] border border-[#8e9ba9]/15" />
            <SlotTooltip eligibleItems="Tools and Spirit Ashes" />
            {active && <DPad active={active} />}
            <PouchPlaceholder />
        </button>
    );
}

function PhysickSlot({ label, onOpen }: { label: string; onOpen: (label: string) => void }) {
    return (
        <button
            type="button"
            aria-label={label}
            onClick={() => onOpen(label)}
            style={slotSurface}
            className="group relative flex h-[82px] w-[82px] flex-col rounded-lg border border-[#d2dce6] p-[5px] text-[#15803d] hover:border-[#75c994]"
        >
            <SlotTooltip eligibleItems="Crystal Tears" />
            <span className="line-clamp-2 min-h-[21px] text-center text-[8px] font-extrabold leading-[1.15]">Physick tear</span>
            <span className="flex h-[51px] items-center justify-center">
                <img className="h-[51px] w-[51px] object-contain opacity-25" src="/equipment/physick-tear-placeholder.png" alt="" />
            </span>
        </button>
    );
}

function EmptyEquipmentModal({ onClose }: { onClose: () => void }) {
    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm" onClick={onClose}>
            <div role="dialog" aria-modal="true" aria-label="Equipment slot" className="flex h-36 w-full max-w-sm items-end justify-end gap-2 rounded-xl border border-border bg-card p-4 shadow-2xl" onClick={(event) => event.stopPropagation()}>
                <button type="button" onClick={onClose} className="rounded border border-border px-4 py-2 text-[10px] font-black uppercase tracking-widest text-muted-foreground hover:bg-muted/40">
                    Cancel
                </button>
                <button type="button" onClick={onClose} className="rounded bg-green-700/80 px-4 py-2 text-[10px] font-black uppercase tracking-widest text-white hover:bg-green-700">
                    Ok
                </button>
            </div>
        </div>
    );
}

export function EquipmentTab() {
    const [selectedSlot, setSelectedSlot] = useState('Weapon slot 1');
    const [modalOpen, setModalOpen] = useState(false);
    const openSlot = (label: string) => {
        setSelectedSlot(label);
        setModalOpen(true);
    };
    const selected = (label: string) => selectedSlot === label;

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

    return (
        <section className="w-full shrink-0 overflow-auto rounded-xl border border-[#d6e0ea] bg-white shadow-sm custom-scrollbar">
            <div className="mx-auto grid w-fit grid-cols-[499px_255px] px-5 py-5">
                <div>
                    <h2 className="mb-3 text-center text-[10px] font-black uppercase tracking-[0.15em] text-[#5a6574]">Equipment slots</h2>
                    <div className="grid gap-[10px]">
                        <div className="grid grid-cols-[repeat(3,82px)_18px_repeat(2,82px)] gap-[9px]">
                            {weaponSlots.map((label) => <EquipmentSlot key={label} label={label} eligibleItems="Weapons, shields, staves, seals and torches" selected={selected(label)} onOpen={openSlot}><GhostIcon src="/equipment/weapon-slot-placeholder.png" /></EquipmentSlot>)}
                            <span aria-hidden="true" />
                            {['Arrow slot 1', 'Arrow slot 2'].map((label) => <EquipmentSlot key={label} label={label} eligibleItems="Arrows and greatarrows" selected={selected(label)} onOpen={openSlot}><GhostIcon src="/items/arrows_and_bolts/arrow.png" mirrored /></EquipmentSlot>)}
                        </div>
                        <div className="-mt-[7px] grid grid-cols-[repeat(3,82px)_18px_repeat(2,82px)] gap-[9px]">
                            {rangedSlots.map((label) => <EquipmentSlot key={label} label={label} eligibleItems="Weapons, shields, staves, seals and torches" selected={selected(label)} onOpen={openSlot}><GhostIcon src="/equipment/ranged-slot-placeholder.png" /></EquipmentSlot>)}
                            <span aria-hidden="true" />
                            {['Bolt slot 1', 'Bolt slot 2'].map((label) => <EquipmentSlot key={label} label={label} eligibleItems="Bolts and greatbolts" selected={selected(label)} onOpen={openSlot}><GhostIcon src="/items/arrows_and_bolts/bolt.png" /></EquipmentSlot>)}
                        </div>
                        <div className="mt-[5px] grid grid-cols-[repeat(4,82px)] gap-[9px]">
                            {armorSlots.map(([label, src], index) => <EquipmentSlot key={label} label={label} eligibleItems={['Helms', 'Chest armor', 'Gauntlets', 'Leg armor'][index]} selected={selected(label)} onOpen={openSlot}><GhostIcon src={src} /></EquipmentSlot>)}
                        </div>
                        <div className="grid grid-cols-[repeat(4,82px)] gap-[9px]">
                            {talismanSlots.map(([label, src]) => <EquipmentSlot key={label} label={label} eligibleItems="Talismans" selected={selected(label)} onOpen={openSlot}><GhostIcon src={src} /></EquipmentSlot>)}
                        </div>
                        {[0, 1].map((row) => (
                            <div key={row} className="grid grid-cols-[repeat(5,82px)] gap-[9px]">
                                {Array.from({ length: 5 }, (_, index) => {
                                    const label = `Quick item ${row * 5 + index + 1}`;
                                    return <EquipmentSlot key={label} label={label} eligibleItems="Tools and Spirit Ashes" selected={selected(label)} onOpen={openSlot}><GhostIcon src={toolsPlaceholder} /></EquipmentSlot>;
                                })}
                            </div>
                        ))}
                    </div>
                </div>

                <div className="border-l border-[#d6e0ea] pl-[26px]">
                    <h2 className="mb-3 text-center text-[10px] font-black uppercase tracking-[0.15em] text-[#5a6574]">Quick pouch</h2>
                    <div className="grid grid-cols-[82px_82px] gap-[9px]">
                        <PouchSlot label="Quick pouch up" active="up" onOpen={openSlot} />
                        <PouchSlot label="Quick pouch right" active="right" onOpen={openSlot} />
                        <PouchSlot label="Quick pouch left" active="left" onOpen={openSlot} />
                        <PouchSlot label="Quick pouch down" active="down" onOpen={openSlot} />
                        <PouchSlot label="Quick pouch slot 5" onOpen={openSlot} />
                        <PouchSlot label="Quick pouch slot 6" onOpen={openSlot} />
                    </div>
                    <div aria-hidden="true" className="mt-[6px] h-[14px]" />
                    <h3 className="mb-3 text-center text-[10px] font-black uppercase tracking-[0.1em] text-[#5a6574]">Wondrous Physick flask</h3>
                    <div className="grid grid-cols-[82px_82px] gap-[9px]">
                        <PhysickSlot label="Physick tear 1" onOpen={openSlot} />
                        <PhysickSlot label="Physick tear 2" onOpen={openSlot} />
                    </div>
                </div>
            </div>
            <div className="mx-5 flex items-center justify-between border-t border-[#d6e0ea] px-0 pb-4 pt-3">
                <span className="text-[11px] font-extrabold tracking-[.04em] text-[#687484]">Equip Load <strong className="text-[#313944]">N / N</strong> | Medium</span>
                <button type="button" className="rounded-md bg-green-700 px-4 py-2 text-[10px] font-black uppercase tracking-[.13em] text-white hover:bg-green-800">Save changes</button>
            </div>
            {modalOpen && <EmptyEquipmentModal onClose={() => setModalOpen(false)} />}
        </section>
    );
}
