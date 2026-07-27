import { useCallback, useEffect, useMemo, useState } from 'react';
import { createPortal } from 'react-dom';
import {
    GetAoWAvailability,
    GetInfuseTypes,
    GetItemList,
} from '../../wailsjs/go/main/App';
import { db, editor, application as main } from '../../wailsjs/go/models';
import { aowApplyPatch, infusionPatch, upgradePatch } from './weaponPatch';
import { WEP_TYPE_TO_BITS, AOW_HEURISTIC_WEPTYPES } from '../data/aowCompat.generated';

// The modal routes upgrade / infusion / AoW edits through the in-memory
// inventory workspace via updateWeapon(uid, patch). pending* state from the
// returned EditableItem is surfaced to the user separately from the read-only
// current* values.
export interface WeaponEditModalWorkspace {
    sessionID: string;
    updateWeapon: (uid: string, patch: editor.WeaponPatch) => Promise<editor.EditableItem | null>;
}

interface Props {
    charIndex: number;
    item: main.InventoryOrderItem;
    source: 'inventory' | 'storage';
    onClose: () => void;
    // The modal operates in workspace mode for the weapon identified by `uid`.
    // The EditableItem snapshot supplies current/pending AoW state without an
    // extra round-trip.
    workspace: WeaponEditModalWorkspace;
    workspaceItem: editor.EditableItem;
}

type AoWCompatStatus = 'compatible' | 'incompatible' | 'unknown';

// Mirrors backend db.IsAoWCompatibleWithWepType. WEP_TYPE_TO_BITS and
// AOW_HEURISTIC_WEPTYPES are generated (single source with the Go backend) —
// see frontend/src/data/aowCompat.generated.ts. Two layers, both fail-closed:
//   1. direct 44-bit mask (base-game and DLC canMountWep fields),
//   2. legacy heuristic fallback for an input with no direct bit.
export function getAoWCompatStatus(aowId: number, aowCompatBitmask: number, wepType: number): AoWCompatStatus {
    if (wepType === 0) return 'unknown';
    const bitPositions = WEP_TYPE_TO_BITS[wepType];
    if (bitPositions && bitPositions.length > 0 && aowCompatBitmask !== 0) {
        const mask = BigInt(aowCompatBitmask);
        if (bitPositions.some(bitPos => ((mask >> BigInt(bitPos)) & BigInt(1)) === BigInt(1))) {
            return 'compatible';
        }
    }
    const heuristic = AOW_HEURISTIC_WEPTYPES[aowId];
    if (heuristic) {
        return heuristic.includes(wepType) ? 'compatible' : 'incompatible';
    }
    if (aowCompatBitmask === 0) return 'unknown';
    if (!bitPositions || bitPositions.length === 0) return 'unknown';
    return 'incompatible';
}

interface AshOfWarOption {
    id: number;
    name: string;
    iconPath: string;
    aowCompatBitmask: number;
}

interface AoWAvailabilityEntry {
    itemId: number;
    totalCopies: number;
    availableCopies: number;
    usedCopies: number;
    usedByWeaponHandles: number[];
    isMissing: boolean;
    hasSharedHandleConflict: boolean;
}

type AoWStatus = 'current' | 'available' | 'in_use' | 'missing' | 'conflict';

export function WeaponEditModal({ charIndex, item, source, onClose, workspace, workspaceItem }: Props) {
    const [imgError, setImgError] = useState(false);
    const [pendingAoWName, setPendingAoWName] = useState<string>(workspaceItem.pendingAoWName ?? '');
    const [pendingAoWClear, setPendingAoWClear] = useState<boolean>(workspaceItem.pendingAoWClear ?? false);
    const [pendingAoWItemID, setPendingAoWItemID] = useState<number>(workspaceItem.pendingAoWItemID ?? 0);

    useEffect(() => {
        setPendingAoWName(workspaceItem.pendingAoWName ?? '');
        setPendingAoWClear(workspaceItem.pendingAoWClear ?? false);
        setPendingAoWItemID(workspaceItem.pendingAoWItemID ?? 0);
    }, [workspaceItem.pendingAoWName, workspaceItem.pendingAoWClear, workspaceItem.pendingAoWItemID]);

    // Live working state — starts from props but tracks Apply results so the
    // modal can show the new level / itemId / infusion without being closed.
    const [currentLevel, setCurrentLevel] = useState<number>(item.currentUpgrade ?? 0);
    const [currentInfusionName, setCurrentInfusionName] = useState<string>(item.infusionName ?? '');
    const maxUpgrade = item.maxUpgrade ?? 0;

    // Seed the selector with a VALID level. If the stored level is out of range
    // (e.g. a +25 somber weapon that should cap at +10), clamp the seed to the max
    // so the dropdown shows a real option and "Apply Level" is immediately usable
    // to repair it — instead of rendering a blank/0 and a disabled button.
    const [selectedLevel, setSelectedLevel] = useState<number>(
        Math.min(Math.max(item.currentUpgrade ?? 0, 0), item.maxUpgrade ?? 0),
    );
    const [applying, setApplying] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);

    // Infusion options loaded from backend (small static list, 13 entries).
    const [infuseTypes, setInfuseTypes] = useState<db.InfuseType[]>([]);
    const currentInfuseOffset = useMemo(() => {
        const name = currentInfusionName || 'Standard';
        return infuseTypes.find(t => t.name === name)?.offset ?? 0;
    }, [infuseTypes, currentInfusionName]);
    const [selectedInfuseOffset, setSelectedInfuseOffset] = useState<number>(0);

    useEffect(() => {
        GetInfuseTypes().then(types => {
            setInfuseTypes(types ?? []);
        }).catch(() => setInfuseTypes([]));
    }, []);

    useEffect(() => {
        setSelectedInfuseOffset(currentInfuseOffset);
    }, [currentInfuseOffset]);

    // ─── Ash of War state ─────────────────────────────────────────────────────
    const [ashesOfWar, setAshesOfWar] = useState<AshOfWarOption[]>([]);
    const [aowAvailability, setAowAvailability] = useState<Map<number, AoWAvailabilityEntry>>(new Map());
    // The AoW-mount metadata (currentAoWId / canMountAoW / wepType) comes
    // straight from the editable workspace item — reading the *save* state
    // would drift from the workspace (added items have no save-side handle,
    // prior Saves may re-allocate handles).
    const [currentAoWId, setCurrentAoWId] = useState<number>(workspaceItem.currentAoWItemID ?? 0);
    const [canMountAoW, setCanMountAoW] = useState<boolean>(workspaceItem.canMountAoW ?? false);
    const [wepType, setWepType] = useState<number>(workspaceItem.wepType ?? 0);
    const [selectedAoW, setSelectedAoW] = useState<number | null>(null);
    const [aowSearch, setAowSearch] = useState('');
    const [showUnavailable, setShowUnavailable] = useState(false);

    // Keep AoW state synced with the latest workspaceItem snapshot — every
    // UpdateWeapon call returns a fresh EditableItem with updated CurrentAoW*
    // (post-save) and Pending* fields.
    useEffect(() => {
        setCurrentAoWId(workspaceItem.currentAoWItemID ?? 0);
        setCanMountAoW(workspaceItem.canMountAoW ?? false);
        setWepType(workspaceItem.wepType ?? 0);
    }, [
        workspaceItem.currentAoWItemID,
        workspaceItem.canMountAoW,
        workspaceItem.wepType,
    ]);

    // Load AoW item list (one-shot).
    useEffect(() => {
        GetItemList('ashes_of_war').then(items => {
            const list: AshOfWarOption[] = (items ?? []).map((it: any) => ({
                id: it.id,
                name: it.name,
                iconPath: it.iconPath ?? '',
                aowCompatBitmask: it.aowCompatBitmask ?? 0,
            })).sort((a, b) => a.name.localeCompare(b.name));
            setAshesOfWar(list);
        }).catch(() => setAshesOfWar([]));
    }, []);

    const refreshAvailability = useCallback(async () => {
        try {
            const entries = await GetAoWAvailability(charIndex);
            const m = new Map<number, AoWAvailabilityEntry>();
            (entries ?? []).forEach((e: any) => m.set(e.itemId as number, e as AoWAvailabilityEntry));
            setAowAvailability(m);
        } catch {
            setAowAvailability(new Map());
        }
    }, [charIndex]);

    useEffect(() => {
        refreshAvailability();
    }, [refreshAvailability]);

    // ─── Derived ──────────────────────────────────────────────────────────────
    // NOTE: fail-closed on unknown compatibility. This modal does NOT pass through
    // 'unknown' compat because there is no backend compatibility API on this
    // branch beyond the existing apply guard, which silently allows unknown when
    // CanWeaponMountAoW==true. Pending merge of research/aow-weapon-compatibility,
    // we treat unknown as blocked here for safety.

    const getStatus = (aowId: number): AoWStatus => {
        if (currentAoWId !== 0 && aowId === currentAoWId) return 'current';
        const avail = aowAvailability.get(aowId);
        if (!avail) return 'missing';
        if (avail.hasSharedHandleConflict) return 'conflict';
        if (avail.availableCopies > 0) return 'available';
        if (avail.usedCopies > 0) return 'in_use';
        return 'missing';
    };

    const filteredAoW = useMemo(() => {
        const q = aowSearch.trim().toLowerCase();
        return ashesOfWar.filter(a => {
 if (q && !a.name.toLowerCase().includes(q)) return false;
 if (!showUnavailable) {
 const compat = getAoWCompatStatus(a.id, a.aowCompatBitmask, wepType);
 // Fail-closed default view: hide incompatible AND unknown.
 if (compat !== 'compatible') return false;
            }
            return true;
        });
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [ashesOfWar, aowSearch, showUnavailable, wepType, currentAoWId, aowAvailability]);

    const currentAoWOption = useMemo(() => {
        if (currentAoWId !== 0) {
            return ashesOfWar.find(a => a.id === currentAoWId) ?? null;
        }
        const defaultName = workspaceItem.defaultAoWName?.trim();
        if (!defaultName) return null;
        return ashesOfWar.find(a => a.name === defaultName) ?? null;
    }, [ashesOfWar, currentAoWId, workspaceItem.defaultAoWName]);
    const currentAoWDisplay = currentAoWId === 0
        ? workspaceItem.defaultAoWName || (workspaceItem.defaultAoWID ? `Skill #${workspaceItem.defaultAoWID}` : 'Built-in skill')
        : currentAoWOption?.name ?? `Unknown (0x${currentAoWId.toString(16).toUpperCase()})`;

    const selectedAoWEntry = selectedAoW !== null && selectedAoW !== 0
        ? ashesOfWar.find(a => a.id === selectedAoW) ?? null
        : null;
    const selectedAoWStatus: AoWStatus | null =
        selectedAoW !== null && selectedAoW !== 0 ? getStatus(selectedAoW) : null;
    const selectedAoWCompat: AoWCompatStatus | null =
        selectedAoWEntry ? getAoWCompatStatus(selectedAoWEntry.id, selectedAoWEntry.aowCompatBitmask, wepType) : null;

    const aowChanged = selectedAoW !== null && selectedAoW !== currentAoWId;
    // Remove (selectedAoW===0) is always allowed when there is a current AoW.
 // Assign requires compat === 'compatible'. Missing/free-copy status is
 // informational because the backend can allocate a fresh AoW GaItem.
    const canApplyAoW = canMountAoW
        && aowChanged
        && !applying
        && (selectedAoW === 0
 || selectedAoWCompat === 'compatible');

    const canRemoveAoW = canMountAoW && currentAoWId !== 0 && !applying;

    // ─── Level / Infusion gates ────────────────────────────────────────────────
    const canEditLevel = maxUpgrade > 0;
    // Stored level above the weapon's real cap = invalid data (e.g. an old
    // out-of-range add). Surfaced so the user can fix it manually.
    const levelOutOfRange = canEditLevel && currentLevel > maxUpgrade;
    const levelChanged = selectedLevel !== currentLevel;
    const levelInRange = selectedLevel >= 0 && selectedLevel <= maxUpgrade;
    const canApplyLevel = canEditLevel && levelChanged && levelInRange && !applying;

    // Affinity support is independent from the +25 upgrade path. Bows and some
    // staffs or torches upgrade normally but explicitly block affinity changes
    // in EquipParamWeapon.disableGemAttr.
    const canEditInfusion = workspaceItem.canChangeAffinity ?? false;
    const infusionChanged = selectedInfuseOffset !== currentInfuseOffset;
    const canApplyInfusion =
        canEditInfusion && infusionChanged && infuseTypes.length > 0 && !applying;

    // ─── Display helpers ──────────────────────────────────────────────────────
    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose();
        };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, [onClose]);

    const levelOptions = useMemo(() => {
        if (maxUpgrade === 0) return [];
        const arr: number[] = [];
        for (let i = 0; i <= maxUpgrade; i++) arr.push(i);
        return arr;
    }, [maxUpgrade]);

    const showIcon = !!item.iconPath && !imgError;
    const upgradeLabel =
        currentLevel > 0
            ? currentInfusionName
                ? `${currentInfusionName} +${currentLevel}`
                : `+${currentLevel}`
            : currentInfusionName || '+0';

    // ─── Apply handlers ────────────────────────────────────────────────────────
    const onApplyLevel = () => {
        if (!canApplyLevel) return;
        setApplying(true);
        setError(null);
        setSuccess(null);
        const patch = upgradePatch(selectedLevel);
        workspace.updateWeapon(workspaceItem.uid, patch)
            .then(updated => {
                if (!updated) {
                    setError('Failed to update upgrade — see notification.');
                    return;
                }
                setCurrentLevel(updated.currentUpgrade);
                setCurrentInfusionName(updated.infusionName ?? '');
                setSuccess(`Level updated to +${updated.currentUpgrade} (pending save)`);
            })
            .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
            .finally(() => setApplying(false));
    };

    const onApplyInfusion = () => {
        if (!canApplyInfusion) return;
        setApplying(true);
        setError(null);
        setSuccess(null);
        const newName = infuseTypes.find(t => t.offset === selectedInfuseOffset)?.name ?? 'Standard';
        const patch = infusionPatch(newName);
        workspace.updateWeapon(workspaceItem.uid, patch)
            .then(updated => {
                if (!updated) {
                    setError('Failed to update infusion — see notification.');
                    return;
                }
                setCurrentLevel(updated.currentUpgrade);
                setCurrentInfusionName(updated.infusionName ?? '');
                setSuccess(`Infusion updated to ${newName} (pending save)`);
            })
            .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
            .finally(() => setApplying(false));
    };

    const applyAoW = (newAoWItemID: number, label: string) => {
        setApplying(true);
        setError(null);
        setSuccess(null);
        const patch = aowApplyPatch(newAoWItemID);
        workspace.updateWeapon(workspaceItem.uid, patch)
            .then(updated => {
                if (!updated) {
                    setError('Failed to update Ash of War — see notification.');
                    return;
                }
                setSelectedAoW(null);
                setSuccess(`${label} (pending save)`);
                setPendingAoWName(updated.pendingAoWName ?? '');
                setPendingAoWClear(updated.pendingAoWClear ?? false);
                setPendingAoWItemID(updated.pendingAoWItemID ?? 0);
            })
            .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
            .finally(() => setApplying(false));
    };

    const onApplyAoW = () => {
        if (!canApplyAoW || selectedAoW === null) return;
        const name = selectedAoW === 0
            ? 'none'
            : ashesOfWar.find(a => a.id === selectedAoW)?.name ?? 'Ash of War';
        applyAoW(selectedAoW, `Ash of War updated to ${name}`);
    };

    const onRemoveAoW = () => {
        if (!canRemoveAoW) return;
        applyAoW(0, 'Custom Ash of War removed — built-in skill restored');
    };

    const statusBadge = (status: AoWStatus) => {
        const map: Record<AoWStatus, { label: string; cls: string }> = {
            current: { label: 'Current', cls: 'bg-blue-500/15 text-blue-700 dark:text-blue-300 border-blue-500/30' },
            available: { label: 'Available', cls: 'bg-green-500/15 text-black border-green-500/30' },
            in_use: { label: 'In use', cls: 'bg-red-200 text-black border-red-300' },
            missing: { label: 'Missing', cls: 'bg-muted/30 text-muted-foreground border-border/40' },
            conflict: { label: 'Conflict', cls: 'bg-red-500/15 text-red-700 dark:text-red-400 border-red-500/30' },
        };
        const m = map[status];
        return (
            <span className={`whitespace-nowrap text-[8px] font-black uppercase tracking-wide border px-1 py-0.5 rounded ${m.cls}`}>
                {m.label}
            </span>
        );
    };

    const compatBadge = (compat: AoWCompatStatus) => {
        if (compat === 'compatible') return null;
        const map: Record<Exclude<AoWCompatStatus, 'compatible'>, { label: string; cls: string }> = {
            incompatible: { label: 'Incompatible', cls: 'bg-red-500/10 text-red-700 dark:text-red-400 border-red-500/30' },
            unknown: { label: 'Unknown', cls: 'bg-muted/30 text-muted-foreground border-border/40' },
        };
        const m = map[compat];
        return (
            <span className={`whitespace-nowrap text-[8px] font-black uppercase tracking-wide border px-1 py-0.5 rounded ${m.cls}`}>
                {m.label}
            </span>
        );
    };

    return createPortal(
        <div
            className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 p-4"
            onClick={onClose}
        >
            <div
                role="dialog"
                aria-modal="true"
                aria-label={`Edit ${item.name}`}
                className="w-full max-w-3xl bg-card border border-border/60 rounded-xl shadow-2xl max-h-[92vh] flex flex-col"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-start justify-between gap-3 p-4 border-b border-border/40 shrink-0">
                    <div className="flex items-center gap-3 min-w-0">
                        <div className="w-14 h-14 rounded-lg bg-muted/20 border border-border/50 flex items-center justify-center shrink-0 overflow-hidden">
                            {showIcon ? (
                                <img
                                    src={item.iconPath}
                                    alt=""
                                    className="w-full h-full object-contain p-1"
                                    onError={() => setImgError(true)}
                                />
                            ) : (
                                <span className="text-xl font-black text-muted-foreground/40 select-none">
                                    {item.name.charAt(0).toUpperCase()}
                                </span>
                            )}
                        </div>
                        <div className="min-w-0">
                            <h2 className="text-sm font-black uppercase tracking-wider text-foreground truncate">
                                {item.name}
                            </h2>
                            <div className="flex items-center flex-wrap gap-1.5 mt-1">
                                <span className="text-[9px] font-black text-primary bg-primary/10 border border-primary/20 px-1.5 py-0.5 rounded">
                                    {upgradeLabel}
                                </span>
                                {source === 'inventory' ? (
                                    <span className="text-[8px] font-black uppercase bg-blue-500/10 text-blue-500 border border-blue-500/20 px-1.5 py-0.5 rounded">
                                        Inventory
                                    </span>
                                ) : (
                                    <span className="text-[8px] font-black uppercase bg-muted/30 text-muted-foreground border border-border/30 px-1.5 py-0.5 rounded">
                                        Storage
                                    </span>
                                )}
                            </div>
                        </div>
                    </div>
                    <button
                        onClick={onClose}
                        title="Close (Esc)"
                        className="shrink-0 text-muted-foreground hover:text-foreground transition-colors p-1 rounded hover:bg-muted/30"
                    >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>

                {/* Body */}
                <div className="p-5 space-y-4 overflow-y-auto">
                    {/* Compact weapon value editors */}
                    <section
                        data-testid="weapon-value-editors"
                        className="grid grid-cols-2 gap-3 rounded-lg border border-border/50 bg-muted/10 p-3"
                    >
                        <div className="min-w-0 space-y-1.5">
                            <div className="flex items-center justify-between gap-2">
                                <span className="text-[10px] font-black uppercase tracking-[0.18em] text-muted-foreground">
                                    Upgrade Level
                                </span>
                                {canEditLevel && (
                                    <span className={`text-[9px] font-mono ${levelOutOfRange ? 'font-black text-red-500' : 'text-muted-foreground/70'}`}>
                                        +{currentLevel} / +{maxUpgrade}
                                    </span>
                                )}
                            </div>
                            {!canEditLevel ? (
                                <p className="text-[9px] italic text-muted-foreground/70">Cannot be upgraded.</p>
                            ) : (
                                <div className="flex items-center gap-1.5">
                                    <select
                                        aria-label="Upgrade level"
                                        value={selectedLevel}
                                        onChange={(e) => setSelectedLevel(Number(e.target.value))}
                                        disabled={applying}
                                        className="h-8 min-w-0 flex-1 rounded-md border border-border/50 bg-background/60 px-2 text-[10px] font-mono focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary/30 disabled:cursor-not-allowed disabled:opacity-50"
                                    >
                                        {levelOptions.map(lvl => <option key={lvl} value={lvl}>+{lvl}</option>)}
                                    </select>
                                    <button
                                        onClick={onApplyLevel}
                                        disabled={!canApplyLevel}
                                        aria-label="Apply Level"
                                        className={`h-8 shrink-0 rounded px-2 text-[9px] font-black uppercase tracking-wide transition-all ${
                                            canApplyLevel
                                                ? 'bg-green-700/80 text-white shadow-sm hover:bg-green-700'
                                                : 'cursor-not-allowed bg-muted/30 text-muted-foreground opacity-40'
                                        }`}
                                        title={!levelChanged ? 'No level change' : applying ? 'Applying…' : 'Apply new upgrade level'}
                                    >
                                        {applying ? 'Applying…' : 'Apply'}
                                    </button>
                                </div>
                            )}
                            {levelOutOfRange && (
                                <p className="text-[9px] leading-snug text-red-500">
                                    Stored +{currentLevel} exceeds the +{maxUpgrade} limit. Select a valid level to repair it.
                                </p>
                            )}
                        </div>

                        <div className="min-w-0 space-y-1.5">
                            <div className="flex items-center justify-between gap-2">
                                <span className="text-[10px] font-black uppercase tracking-[0.18em] text-muted-foreground">
                                    Infusion
                                </span>
                                {canEditInfusion && (
                                    <span className="truncate text-[9px] font-mono text-muted-foreground/70">
                                        {currentInfusionName || 'Standard'}
                                    </span>
                                )}
                            </div>
                            {!canEditInfusion ? (
                                <p className="text-[9px] italic text-muted-foreground/70">Affinity changes unavailable.</p>
                            ) : (
                                <div className="flex items-center gap-1.5">
                                    <select
                                        aria-label="Infusion"
                                        value={selectedInfuseOffset}
                                        onChange={(e) => setSelectedInfuseOffset(Number(e.target.value))}
                                        disabled={applying || infuseTypes.length === 0}
                                        className="h-8 min-w-0 flex-1 rounded-md border border-border/50 bg-background/60 px-2 text-[10px] font-mono focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary/30 disabled:cursor-not-allowed disabled:opacity-50"
                                    >
                                        {infuseTypes.map(t => <option key={t.offset} value={t.offset}>{t.name}</option>)}
                                    </select>
                                    <button
                                        onClick={onApplyInfusion}
                                        disabled={!canApplyInfusion}
                                        aria-label="Apply Infusion"
                                        className={`h-8 shrink-0 rounded px-2 text-[9px] font-black uppercase tracking-wide transition-all ${
                                            canApplyInfusion
                                                ? 'bg-green-700/80 text-white shadow-sm hover:bg-green-700'
                                                : 'cursor-not-allowed bg-muted/30 text-muted-foreground opacity-40'
                                        }`}
                                        title={!infusionChanged ? 'No infusion change' : applying ? 'Applying…' : 'Apply new infusion'}
                                    >
                                        {applying ? 'Applying…' : 'Apply'}
                                    </button>
                                </div>
                            )}
                        </div>
                    </section>

                    {/* Ash of War edit section */}
                    <section className="rounded-lg border border-border/50 bg-muted/10 p-3 space-y-2">
                        <div className="flex items-center justify-between gap-2">
                            <span className="text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">
                                Ash of War
                            </span>
                        </div>

                        {(pendingAoWClear || pendingAoWItemID !== 0) && (
                            <div className="text-[11px] px-3 py-2 rounded border bg-blue-500/10 border-blue-500/30 text-blue-300 leading-snug">
                                {pendingAoWClear
                                    ? 'Pending save: built-in skill will be restored.'
                                    : `Pending save: ${pendingAoWName || `0x${pendingAoWItemID.toString(16).toUpperCase()}`}`}
                            </div>
                        )}

                        {!canMountAoW ? (
                            <p className="text-[11px] text-muted-foreground/70 italic">
                                This weapon does not support Ash of War changes.
                            </p>
                        ) : (
                            <>
                                <div className="flex items-center gap-3 rounded-lg border border-border/40 bg-background/45 px-3 py-2.5">
                                    <div
                                        data-testid="current-aow-icon"
                                        className="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-md border border-border/40 bg-muted/20"
                                    >
                                        {currentAoWOption?.iconPath ? (
                                            <img
                                                src={currentAoWOption.iconPath}
                                                alt=""
                                                className="h-full w-full object-contain p-0.5 drop-shadow-sm"
                                            />
                                        ) : (
                                            <span className="text-base font-black text-muted-foreground/40">
                                                {currentAoWDisplay.charAt(0) || '?'}
                                            </span>
                                        )}
                                    </div>
                                    <div className="min-w-0 flex-1">
                                        <div className="text-[8px] font-black uppercase tracking-[0.18em] text-muted-foreground/70">
                                            Current Ash of War
                                        </div>
                                        <div className="text-sm font-black text-foreground truncate" title={currentAoWDisplay}>
                                            {currentAoWDisplay}
                                        </div>
                                        <div className="text-[10px] text-muted-foreground/70">
                                            {currentAoWId === 0 ? 'Weapon is using its built-in skill.' : 'Custom Ash of War is attached.'}
                                        </div>
                                    </div>
                                    <button
                                        onClick={onRemoveAoW}
                                        disabled={!canRemoveAoW}
                                        title={canRemoveAoW ? "Remove custom Ash of War — weapon will use its built-in skill" : "No custom Ash of War attached"}
                                        aria-label={canRemoveAoW ? "Remove custom Ash of War — weapon will use its built-in skill" : "No custom Ash of War attached"}
                                        className={`px-3 py-2 text-[10px] font-black uppercase tracking-wider rounded border transition-all ${
                                            canRemoveAoW
                                                ? 'text-red-300 bg-red-500/10 border-red-500/30 hover:bg-red-500/20'
                                                : 'opacity-40 cursor-not-allowed text-muted-foreground bg-muted/20 border-border/30'
                                        }`}
                                    >
                                        Remove
                                    </button>
                                </div>

                                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                                    <div className="relative flex-1">
                                        <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                                        </svg>
                                        <input
                                            type="text"
                                            placeholder="Search Ashes of War..."
                                            value={aowSearch}
                                            onChange={(e) => setAowSearch(e.target.value)}
                                            disabled={applying}
                                            className="h-9 w-full bg-background/60 border border-border/50 rounded-md pl-9 pr-3 text-[12px] focus:outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary disabled:opacity-50"
                                        />
                                    </div>
                                    <label className="flex h-9 shrink-0 items-center gap-2 rounded-md border border-border/40 bg-background/40 px-3 text-[11px] text-muted-foreground/80 cursor-pointer select-none">
                                        <input
                                            type="checkbox"
                                            checked={showUnavailable}
                                            onChange={(e) => setShowUnavailable(e.target.checked)}
                                            className="accent-primary"
                                        />
                                        Show unavailable
                                    </label>
                                </div>

                                <div
                                    data-testid="aow-icon-grid"
                                    className="grid max-h-72 grid-cols-[repeat(auto-fill,minmax(104px,1fr))] gap-2 overflow-y-auto rounded border border-border/40 bg-background/30 p-2"
                                >
                                    {ashesOfWar.length === 0 ? (
                                        <p className="col-span-full text-[10px] text-muted-foreground/60 italic p-3 text-center">Loading…</p>
                                    ) : filteredAoW.length === 0 ? (
                                        <p className="col-span-full text-[10px] text-muted-foreground/60 italic p-3 text-center">
                                            {aowSearch ? 'No matching Ashes of War.' : 'No compatible Ashes of War available.'}
                                        </p>
                                    ) : (
                                        filteredAoW.map(aow => {
                                            const status = getStatus(aow.id);
                                            const compat = getAoWCompatStatus(aow.id, aow.aowCompatBitmask, wepType);
                                            const isSelected = selectedAoW === aow.id;
 const selectable = compat === 'compatible';
                                            return (
                                                <button
                                                    key={aow.id}
                                                    type="button"
                                                    disabled={applying}
                                                    data-aow-icon-card
                                                    data-aow-status={status}
                                                    data-aow-compat={compat}
                                                    aria-label={`Select ${aow.name}`}
                                                    aria-pressed={isSelected}
                                                    title={aow.name}
                                                    onClick={() => setSelectedAoW(isSelected ? null : aow.id)}
                                                    className={`relative flex min-h-[128px] flex-col items-center justify-start gap-1 rounded-lg border p-2 text-center transition-all disabled:cursor-not-allowed ${
                                                        isSelected
                                                            ? 'border-primary bg-primary/15 ring-1 ring-primary/50'
                                                            : selectable
                                                              ? 'border-border/50 bg-card/50 hover:border-primary/60 hover:bg-primary/[0.06]'
                                                              : 'border-border/30 bg-muted/10 opacity-60 grayscale-[0.65] hover:bg-muted/20'
                                                    }`}
                                                >
                                                    <div className="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-md border border-border/40 bg-muted/20">
                                                        {aow.iconPath ? (
                                                            <img src={aow.iconPath} alt="" className="h-full w-full object-contain p-0.5 drop-shadow-sm" />
                                                        ) : (
                                                            <span className="text-lg font-black text-muted-foreground/40">{aow.name.charAt(0)}</span>
                                                        )}
                                                    </div>
                                                    <span className="line-clamp-2 min-h-7 w-full text-[10px] font-bold leading-tight text-foreground/85">
                                                        {aow.name}
                                                    </span>
                                                    <div className="mt-auto flex max-w-full flex-wrap items-center justify-center gap-0.5">
                                                        {compatBadge(compat)}
                                                        {statusBadge(status)}
                                                    </div>
                                                </button>
                                            );
                                        })
                                    )}
                                </div>

                                {selectedAoW !== null && selectedAoW !== 0 && selectedAoWCompat === 'unknown' && (
                                    <p className="text-[9px] text-muted-foreground/80 italic leading-snug">
                                        Unknown compatibility data — blocked for safety. Will be unblocked once weapon AoW compatibility API is merged.
                                    </p>
                                )}
                                {selectedAoW !== null && selectedAoW !== 0 && selectedAoWCompat === 'incompatible' && (
                                    <p className="text-[9px] text-red-400/85 italic leading-snug">
                                        This Ash of War is not compatible with this weapon type.
                                    </p>
                                )}
                                {selectedAoW !== null && selectedAoW !== 0 && selectedAoWStatus !== 'available' && selectedAoWStatus !== 'current' && (
                                    <p className="text-[9px] text-muted-foreground/80 italic leading-snug">
 No free copy is available in the save; applying will create a fresh Ash of War record.
                                    </p>
                                )}

                            </>
                        )}
                    </section>

                    {error && (
                        <p className="text-[10px] font-mono text-red-400/90 leading-snug break-words">
                            {error}
                        </p>
                    )}
                    {success && !error && (
                        <p className="text-[10px] font-bold text-green-400/90">
                            {success}
                        </p>
                    )}

                </div>

                {/* Footer */}
                <div className="flex items-center justify-end gap-2 p-3 border-t border-border/40 shrink-0">
                    <button
                        onClick={onApplyAoW}
                        disabled={!canApplyAoW}
                        title={
                            !canMountAoW
                                ? 'Weapon does not support Ash of War'
                                : !aowChanged
                                    ? 'No Ash of War change'
                                    : selectedAoW !== 0 && selectedAoWCompat !== 'compatible'
                                        ? selectedAoWCompat === 'unknown'
                                            ? 'Unknown compatibility — blocked for safety'
                                            : 'Incompatible Ash of War'
                                        : applying
                                            ? 'Applying…'
                                            : 'Apply new Ash of War'
                        }
                        className={`px-4 py-1.5 text-[10px] font-black uppercase tracking-wider rounded transition-all ${
                            canApplyAoW
                                ? 'bg-green-700/80 text-white hover:bg-green-700 shadow-sm'
                                : 'opacity-40 cursor-not-allowed bg-muted/30 text-muted-foreground'
                        }`}
                    >
                        {applying ? 'Applying…' : 'Apply Ash of War'}
                    </button>
                    <button
                        onClick={onClose}
                        className="px-3 py-1.5 text-[10px] font-black uppercase tracking-wider rounded text-muted-foreground hover:text-foreground hover:bg-muted/40 transition-all"
                    >
                        Close
                    </button>
                </div>
            </div>
        </div>,
        document.body,
    );
}
