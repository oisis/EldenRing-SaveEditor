import {useEffect, useRef, useState} from 'react';
import toast from '../lib/toast';
import {GetCharacter, SaveCharacter, GetStartingClasses, SetCharacterGender, GetCharacterAppearancePreset, SetOwnedWeaponLevels} from '../../wailsjs/go/main/App';
import {vm, main, db} from '../../wailsjs/go/models';
import {AccordionSection} from './AccordionSection';
import {AppearanceTab} from './AppearanceTab';
import {RiskInfoIcon} from './RiskInfoIcon';
import {getRunesRiskKey} from '../data/riskInfo';
import {useSafetyMode} from '../state/safetyMode';
import type {AddSettings} from '../App';

const RUNES_LEGAL_MAX = 999_999_999;

function runesCostForLevel(level: number): number {
    let total = 0;
    for (let n = 2; n <= level; n++) {
        const cost = Math.floor(0.02 * n * n * n + 3.06 * n * n + 105.6 * n - 895);
        if (cost > 0) total += cost;
    }
    return Math.min(total, 4_294_967_295);
}

interface Props {
    charIndex: number;
    onNameChange?: () => void;
    onMutate: () => void;
    refreshKey?: number;
    addSettings: AddSettings;
    onAddSettingsChange: (s: AddSettings) => void;
    infuseTypes: db.InfuseType[];
}

const ATTRIBUTES = [
    { id: 'vigor', label: 'Vigor', abbr: 'Vig' },
    { id: 'mind', label: 'Mind', abbr: 'Min' },
    { id: 'endurance', label: 'Endurance', abbr: 'End' },
    { id: 'strength', label: 'Strength', abbr: 'Str' },
    { id: 'dexterity', label: 'Dexterity', abbr: 'Dex' },
    { id: 'intelligence', label: 'Intelligence', abbr: 'Int' },
    { id: 'faith', label: 'Faith', abbr: 'Fai' },
    { id: 'arcane', label: 'Arcane', abbr: 'Arc' },
];

export function CharacterTab({charIndex, onNameChange, onMutate, refreshKey, addSettings, onAddSettingsChange, infuseTypes}: Props) {
    const safetyMode = useSafetyMode();
    const [char, setChar] = useState<vm.CharacterViewModel | null>(null);
    const [loading, setLoading] = useState(false);
    const [startingClasses, setStartingClasses] = useState<db.ClassStats[]>([]);
    const isDirty = useRef(false);
    const [characterDirty, setCharacterDirty] = useState(false);
    const [applyOwnedWeaponLevels, setApplyOwnedWeaponLevels] = useState(false);
    const [applyingChanges, setApplyingChanges] = useState(false);
    const prevCharIndex = useRef(charIndex);

    // Appearance state
    const [matchedPreset, setMatchedPreset] = useState<main.PresetInfo | null>(null);

    const refreshMatch = () => {
        GetCharacterAppearancePreset(charIndex).then(setMatchedPreset).catch(() => setMatchedPreset(null));
    };

    useEffect(() => {
        GetStartingClasses().then(setStartingClasses).catch(e => toast.error("" + e));
    }, []);

    // Refresh the exact appearance match on initial load, character change, and
    // whenever a mutation bumps refreshKey (direct Apply, Apply from Mirror).
    useEffect(() => { refreshMatch(); }, [charIndex, refreshKey]);

    useEffect(() => {
        const charChanged = prevCharIndex.current !== charIndex;
        prevCharIndex.current = charIndex;
        if (charChanged) {
            isDirty.current = false;
            setCharacterDirty(false);
            setApplyOwnedWeaponLevels(false);
        } else if (isDirty.current) {
            return;
        }
        setLoading(true);
        GetCharacter(charIndex)
            .then(res => { setChar(res); setLoading(false); })
            .catch(() => setLoading(false));
    }, [charIndex, refreshKey]);

    const getStatMin = (statId: string): number => {
        return char?.classBaseStats?.[statId] || 1;
    };

    const updateStat = (key: string, val: number) => {
        if (!char) return;
        const min = getStatMin(key);
        const clampedVal = Math.min(99, Math.max(min, val));
        const updatedData = {...char, [key]: clampedVal} as any;
        const sum = updatedData.vigor + updatedData.mind + updatedData.endurance + updatedData.strength +
                    updatedData.dexterity + updatedData.intelligence + updatedData.faith + updatedData.arcane;
        updatedData.level = Math.max(1, sum - 79);
        isDirty.current = true;
        setCharacterDirty(true);
        setChar(vm.CharacterViewModel.createFrom(updatedData));
    };

    const handleClassChange = (classId: number) => {
        if (!char) return;
        const nc = startingClasses.find(c => c.id === classId);
        if (!nc) return;
        const vigor        = Math.max(char.vigor,        nc.vigor);
        const mind         = Math.max(char.mind,         nc.mind);
        const endurance    = Math.max(char.endurance,    nc.endurance);
        const strength     = Math.max(char.strength,     nc.strength);
        const dexterity    = Math.max(char.dexterity,    nc.dexterity);
        const intelligence = Math.max(char.intelligence, nc.intelligence);
        const faith        = Math.max(char.faith,        nc.faith);
        const arcane       = Math.max(char.arcane,       nc.arcane);
        const level = Math.max(1, vigor + mind + endurance + strength + dexterity + intelligence + faith + arcane - 79);
        isDirty.current = true;
        setCharacterDirty(true);
        setChar(vm.CharacterViewModel.createFrom({
            ...char,
            class: classId,
            className: nc.name,
            classBaseStats: { vigor: nc.vigor, mind: nc.mind, endurance: nc.endurance, strength: nc.strength, dexterity: nc.dexterity, intelligence: nc.intelligence, faith: nc.faith, arcane: nc.arcane },
            vigor, mind, endurance, strength, dexterity, intelligence, faith, arcane, level,
        }));
    };

    const handleGenderChange = async (targetGender: number) => {
        if (!char) return;
        try {
            await SetCharacterGender(charIndex, targetGender);
            const updated = await GetCharacter(charIndex);
            setChar(updated);
            refreshMatch();
            const label = targetGender === 1 ? 'Type A (Male) — Geralt defaults applied' : 'Type B (Female) — Ciri defaults applied';
            toast.success(label);
        } catch (e) {
            toast.error('Gender change failed: ' + e);
        }
    };

    const handleSave = async () => {
        if (!char || (!characterDirty && !applyOwnedWeaponLevels)) return;
        setApplyingChanges(true);
        try {
            // Apply the optional weapon batch first: a batch failure leaves the
            // separately staged character fields untouched and retryable.
            if (applyOwnedWeaponLevels) {
                const changed = await SetOwnedWeaponLevels(charIndex, addSettings.upgrade25, addSettings.upgrade10);
                toast.success(changed === 0
                    ? 'All owned weapons already match these levels'
                    : `Set ${changed} owned weapon${changed === 1 ? '' : 's'} to +${addSettings.upgrade25} / +${addSettings.upgrade10}`);
                setApplyOwnedWeaponLevels(false);
            }
            if (characterDirty) {
                await SaveCharacter(charIndex, char);
                isDirty.current = false;
                setCharacterDirty(false);
                toast.success('Character data updated in memory');
                onNameChange?.();
                GetCharacter(charIndex).then(updated => { if (updated) setChar(updated); }).catch(() => {});
            }
            onMutate();
        } catch (err) {
            toast.error('Apply changes failed: ' + err);
        } finally {
            setApplyingChanges(false);
        }
    };

    const handleFixSoulMemory = () => {
        if (!char) return;
        const minRequired = runesCostForLevel(char.level);
        const buffered = Math.min(Math.floor(minRequired * 1.1), 4_294_967_295);
        const updated = vm.CharacterViewModel.createFrom({...char, soulMemory: buffered});
        setChar(updated);
        SaveCharacter(charIndex, updated)
            .then(() => {
                isDirty.current = false;
                setCharacterDirty(false);
                toast.success('Soul Memory corrected');
                onNameChange?.();
                onMutate();
                GetCharacter(charIndex).then(res => { if (res) setChar(res); }).catch(() => {});
            })
            .catch(err => toast.error('Fix failed: ' + err));
    };

    const handleAppearanceMutate = () => {
        GetCharacter(charIndex).then(updated => { if (updated) setChar(updated); }).catch(() => {});
        refreshMatch();
        onMutate();
    };

    // Summaries for collapsed sections
    const profileSummary = char
        ? <span className="flex items-center gap-2 text-center">
            <span className="text-xs font-black text-primary">{char.name}</span>
            <span className="text-[11px] text-muted-foreground font-medium">RL {char.level} | NG+{char.clearCount || 0} | {(char.souls || 0).toLocaleString()} Runes</span>
          </span>
        : undefined;

    const attrSummary = char
        ? ATTRIBUTES.map(a => `${a.abbr} ${(char as any)[a.id]}`).join(' | ')
        : '';

    if (loading) return (
        <div className="py-10 flex flex-col items-center justify-center space-y-3">
            <div className="w-5 h-5 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Loading...</p>
        </div>
    );

    if (!char) return (
        <div className="py-10 text-center border border-dashed border-border rounded-lg">
            <p className="text-xs text-muted-foreground">No character data.</p>
        </div>
    );

    return (
        <div className="space-y-3 animate-in fade-in duration-500 max-w-5xl mx-auto">
            {/* ═══ PROFILE ═══ */}
            <AccordionSection
                id="char-profile"
                title="Profile"
                summary={profileSummary}
                headerRight={
                    <div className="flex items-center gap-1.5">
                        <span className="text-[11px] font-black text-muted-foreground uppercase tracking-[0.2em]">RL</span>
                        <span className="text-lg font-black tracking-tighter text-primary leading-none">{char.level}</span>
                    </div>
                }
            >
                <div className="space-y-4">
                    {matchedPreset && (
                        <div className="flex items-center gap-3 rounded-lg border border-primary/30 bg-primary/5 px-3 py-2" data-testid="matched-appearance">
                            {matchedPreset.image && (
                                <img src={`presets/${matchedPreset.image}`} alt={matchedPreset.name}
                                    className="w-10 h-12 object-cover object-top rounded" />
                            )}
                            <div className="flex flex-col leading-tight">
                                <span className="text-[9px] font-bold text-muted-foreground uppercase tracking-widest">Matched appearance</span>
                                <span className="text-xs font-black text-primary">{matchedPreset.name}</span>
                            </div>
                        </div>
                    )}
                    <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold text-muted-foreground uppercase tracking-tight ml-1">Character Name</label>
                            <input type="text" value={char.name} maxLength={16}
                                onChange={e => { isDirty.current = true; setCharacterDirty(true); setChar(vm.CharacterViewModel.createFrom({...char, name: e.target.value})); }}
                                className="w-full bg-muted/20 border border-border rounded-md px-3 py-2 text-xs focus:ring-1 focus:ring-primary/30 outline-none transition-all" />
                        </div>
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold text-muted-foreground uppercase tracking-tight ml-1">Starting Class</label>
                            <select value={char.class ?? 0}
                                onChange={e => handleClassChange(parseInt(e.target.value))}
                                className="w-full bg-muted/20 border border-border rounded-md px-3 py-2 text-xs font-black text-primary focus:ring-1 focus:ring-primary/30 outline-none transition-all cursor-pointer h-[34px]">
                                {startingClasses.map(c => (
                                    <option key={c.id} value={c.id}>{c.name}</option>
                                ))}
                            </select>
                        </div>
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold text-muted-foreground uppercase tracking-tight ml-1">Body Type</label>
                            <select
                                value={char.gender ?? 1}
                                onChange={e => handleGenderChange(parseInt(e.target.value))}
                                className="w-full bg-muted/20 border border-border rounded-md px-3 py-2 text-xs font-black text-primary focus:ring-1 focus:ring-primary/30 outline-none transition-all cursor-pointer h-[34px]">
                                <option value={1}>Type A (Male)</option>
                                <option value={0}>Type B (Female)</option>
                            </select>
                        </div>
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold text-muted-foreground uppercase tracking-tight ml-1">
                                Talisman Slots <span className="text-primary font-mono">{1 + (char.talismanSlots || 0)}/4</span>
                            </label>
                            <input type="number" min={0} max={3} value={char.talismanSlots || 0}
                                onChange={e => {
                                    const v = Math.min(3, Math.max(0, parseInt(e.target.value) || 0));
                                    isDirty.current = true;
                                    setCharacterDirty(true);
                                    setChar(vm.CharacterViewModel.createFrom({...char, talismanSlots: v}));
                                }}
                                className="w-full bg-muted/20 border border-border rounded-md px-3 py-2 text-xs font-mono focus:ring-1 focus:ring-primary/30 outline-none transition-all" />
                        </div>
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold text-muted-foreground uppercase tracking-tight ml-1">
                                Memory Stones <span className="text-primary font-mono">{char.memoryStones || 0}/8</span>
                            </label>
                            <input type="number" min={0} max={8} value={char.memoryStones || 0}
                                onChange={e => {
                                    const v = Math.min(8, Math.max(0, parseInt(e.target.value) || 0));
                                    isDirty.current = true;
                                    setCharacterDirty(true);
                                    setChar(vm.CharacterViewModel.createFrom({...char, memoryStones: v}));
                                }}
                                className="w-full bg-muted/20 border border-border rounded-md px-3 py-2 text-xs font-mono focus:ring-1 focus:ring-primary/30 outline-none transition-all" />
                        </div>
                    </div>

                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold text-muted-foreground uppercase tracking-tight ml-1">
                                NG+ Cycle <span className="text-primary font-mono">{char.clearCount || 0}/7</span>
                            </label>
                            <input type="number" min={0} max={7} value={char.clearCount || 0}
                                onChange={e => {
                                    const v = Math.min(7, Math.max(0, parseInt(e.target.value) || 0));
                                    isDirty.current = true;
                                    setCharacterDirty(true);
                                    setChar(vm.CharacterViewModel.createFrom({...char, clearCount: v}));
                                }}
                                className="w-full bg-muted/20 border border-border rounded-md px-3 py-2 text-xs font-mono focus:ring-1 focus:ring-primary/30 outline-none transition-all" />
                        </div>
                        <div className="space-y-1.5">
                            <label className="text-[11px] font-bold text-muted-foreground uppercase tracking-tight ml-1 flex items-center gap-1.5">
                                <span>Runes</span>
                                {getRunesRiskKey(char.souls) && <RiskInfoIcon riskKey={getRunesRiskKey(char.souls)!} />}
                            </label>
                            <input type="number" value={char.souls}
                                onChange={e => {
                                    let v = parseInt(e.target.value) || 0;
                                    if (safetyMode.enabled && v > RUNES_LEGAL_MAX) {
                                        v = RUNES_LEGAL_MAX;
                                        toast.error(`Online Safety Mode: clamped to legal max ${RUNES_LEGAL_MAX.toLocaleString()}`);
                                    }
                                    isDirty.current = true;
                                    setCharacterDirty(true);
                                    setChar(vm.CharacterViewModel.createFrom({...char, souls: v}));
                                }}
                                title={safetyMode.enabled ? `Online Safety Mode caps Runes at ${RUNES_LEGAL_MAX.toLocaleString()}` : undefined}
                                className={
                                    getRunesRiskKey(char.souls)
                                        ? 'w-full bg-red-500/10 border-2 border-red-500 rounded-md px-3 py-2 text-xs font-mono text-red-300 focus:ring-2 focus:ring-red-500/40 outline-none transition-all'
                                        : 'w-full bg-muted/20 border border-border rounded-md px-3 py-2 text-xs font-mono focus:ring-1 focus:ring-primary/30 outline-none transition-all'
                                } />
                        </div>
                        {(() => {
                            const minSM = runesCostForLevel(char.level);
                            const consistent = (char.soulMemory || 0) >= minSM;
                            return (
                                <div className="space-y-1.5">
                                    <label className="text-[11px] font-bold text-muted-foreground uppercase tracking-tight ml-1 flex items-center gap-1.5">
                                        <span>Soul Memory</span>
                                        <span className={consistent ? 'text-green-400 font-black' : 'text-red-400 font-black'}>
                                            {consistent ? '✓' : '✗'}
                                        </span>
                                    </label>
                                    <div className="flex items-center gap-1.5">
                                        <span className="flex-1 bg-muted/20 border border-border rounded-md px-3 py-2 text-xs font-mono text-muted-foreground truncate">
                                            {(char.soulMemory || 0).toLocaleString()}
                                        </span>
                                        {!consistent && (
                                            <button onClick={handleFixSoulMemory}
                                                title={`Set to ${Math.min(Math.floor(minSM * 1.1), 4_294_967_295).toLocaleString()} (+10% buffer)`}
                                                className="px-2 py-2 text-[10px] font-black uppercase tracking-widest bg-red-500/20 border border-red-500/50 text-red-300 rounded-md hover:bg-red-500/30 transition-colors whitespace-nowrap">
                                                Fix
                                            </button>
                                        )}
                                    </div>
                                </div>
                            );
                        })()}
                    </div>
                </div>
            </AccordionSection>

            {/* ═══ ATTRIBUTES ═══ */}
            <AccordionSection
                id="char-attributes"
                title="Attributes"
                summary={attrSummary}
            >
                <div className="grid grid-cols-1 md:grid-cols-2 gap-x-6">
                    {ATTRIBUTES.map(stat => {
                        const statMin = getStatMin(stat.id);
                        const redZonePct = ((statMin - 1) / 98) * 100;
                        return (
                            <div key={stat.id} className="flex items-center gap-3 py-1.5 border-b border-border/30">
                                <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider w-20 flex-shrink-0"
                                    title={`Base: ${statMin}`}>
                                    {stat.label}
                                </span>
                                <input
                                    type="range" min={1} max={99}
                                    value={(char as any)[stat.id]}
                                    onChange={e => updateStat(stat.id, parseInt(e.target.value))}
                                    className="flex-1 h-1.5 rounded-lg appearance-none cursor-pointer"
                                    style={{
                                        background: `linear-gradient(to right, rgb(239 68 68 / 0.4) 0%, rgb(239 68 68 / 0.4) ${redZonePct}%, hsl(var(--border)) ${redZonePct}%, hsl(var(--border)) 100%)`,
                                    }}
                                />
                                <input
                                    type="number" min={statMin} max={99}
                                    value={(char as any)[stat.id]}
                                    onChange={e => updateStat(stat.id, parseInt(e.target.value) || statMin)}
                                    className="w-12 bg-muted/30 border border-border rounded text-center text-xs py-1 focus:ring-1 focus:ring-primary/30 outline-none"
                                />
                            </div>
                        );
                    })}
                </div>
            </AccordionSection>

            {/* ═══ ADD SETTINGS ═══ */}
            <AccordionSection
                id="char-add-settings"
                title="Add Settings"
                summary={`+${addSettings.upgrade25} · +${addSettings.upgrade10} · ${infuseTypes.find(t => t.offset === addSettings.infuseOffset)?.name ?? 'Standard'} · Ash +${addSettings.upgradeAsh}`}
            >
                {(() => {
                    const set = (patch: Partial<AddSettings>) => onAddSettingsChange({...addSettings, ...patch});
                    return (
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-x-10 gap-y-5 py-2">
                            <div className="flex items-center space-x-3">
                                <span className="text-[11px] font-normal uppercase tracking-widest text-foreground w-24 shrink-0">Weapon +25</span>
                                <input type="range" min={0} max={25} value={addSettings.upgrade25} onChange={e => set({upgrade25: parseInt(e.target.value)})}
                                    className="flex-1 h-1.5 rounded-lg appearance-none cursor-pointer"
                                    style={{background: 'hsl(var(--border))'}} />
                                <span className="text-[10px] font-mono font-bold text-primary w-6 text-right">+{addSettings.upgrade25}</span>
                            </div>
                            <div className="flex items-center space-x-3">
                                <span className="text-[11px] font-normal uppercase tracking-widest text-foreground w-24 shrink-0">Weapon +10</span>
                                <input type="range" min={0} max={10} value={addSettings.upgrade10} onChange={e => set({upgrade10: parseInt(e.target.value)})}
                                    className="flex-1 h-1.5 rounded-lg appearance-none cursor-pointer"
                                    style={{background: 'hsl(var(--border))'}} />
                                <span className="text-[10px] font-mono font-bold text-primary w-5 text-right">+{addSettings.upgrade10}</span>
                            </div>
                            <div className="flex items-center space-x-3">
                                <span className="text-[11px] font-normal uppercase tracking-widest text-foreground w-24 shrink-0">Infuse</span>
                                <select value={addSettings.infuseOffset} onChange={e => set({infuseOffset: parseInt(e.target.value)})}
                                    className="flex-1 bg-muted/20 border border-border rounded-md px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider focus:ring-1 focus:ring-primary/30 outline-none transition-all cursor-pointer">
                                    {infuseTypes.map(t => <option key={t.offset} value={t.offset}>{t.name}</option>)}
                                </select>
                            </div>
                            <div className="flex items-center space-x-3">
                                <span className="text-[11px] font-normal uppercase tracking-widest text-foreground w-24 shrink-0">Spirit Ash</span>
                                <input type="range" min={0} max={10} value={addSettings.upgradeAsh} onChange={e => set({upgradeAsh: parseInt(e.target.value)})}
                                    className="flex-1 h-1.5 rounded-lg appearance-none cursor-pointer"
                                    style={{background: 'hsl(var(--border))'}} />
                                <span className="text-[10px] font-mono font-bold text-primary w-5 text-right">+{addSettings.upgradeAsh}</span>
                            </div>
                            <div className="flex items-center gap-8 md:col-span-2 pt-1 border-t border-border/30">
                                <label title={`Standard +${addSettings.upgrade25} · Special +${addSettings.upgrade10} · Inventory only`} className="flex items-center gap-2 cursor-pointer">
                                    <input type="checkbox" checked={applyOwnedWeaponLevels} onChange={e => setApplyOwnedWeaponLevels(e.target.checked)}
                                        className="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20" />
                                    <span className="text-[11px] font-normal uppercase tracking-widest text-foreground">Set all weapons levels</span>
                                </label>
                                <label title="When enabled, only the highest-tier variant of each talisman family is shown — lower upgrade levels are hidden." className="flex items-center gap-2 cursor-pointer">
                                    <input type="checkbox" checked={addSettings.talismansHighestOnly} onChange={e => set({talismansHighestOnly: e.target.checked})}
                                        className="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20" />
                                    <span className="text-[11px] font-normal uppercase tracking-widest text-foreground">Talismans: highest only</span>
                                </label>
                                <label title="When enabled, 'Unlock All' Sites of Grace in World tab will also include Leyndell, Ashen Capital graces. Disable if you haven't triggered the capital's transformation yet." className="flex items-center gap-2 cursor-pointer">
                                    <input type="checkbox" checked={addSettings.includeAshenCapital} onChange={e => set({includeAshenCapital: e.target.checked})}
                                        className="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20" />
                                    <span className="text-[11px] font-normal uppercase tracking-widest text-foreground">SoG: Leyndell, Ashen Capital</span>
                                </label>
                            </div>
                        </div>
                    );
                })()}
            </AccordionSection>

            {/* ═══ APPEARANCE PRESETS ═══ */}
            <AccordionSection
                id="char-presets"
                title="Appearance Presets"
            >
                <AppearanceTab charIndex={charIndex} onMutate={handleAppearanceMutate} embedded />
            </AccordionSection>

            {/* ═══ APPLY CHANGES ═══ */}
            <div className="flex justify-end items-center space-x-4 pt-4 pb-2 border-t border-border/30">
                <p className="text-[11px] font-bold text-muted-foreground uppercase tracking-widest italic opacity-50">Staged in memory</p>
                <button onClick={handleSave} disabled={!characterDirty && !applyOwnedWeaponLevels || applyingChanges}
                    className="bg-primary text-primary-foreground hover:brightness-110 active:scale-95 transition-all font-black px-6 py-2 rounded-md text-[10px] uppercase tracking-widest shadow-lg shadow-primary/20 disabled:cursor-not-allowed disabled:opacity-50">
                    {applyingChanges ? 'Applying…' : 'Apply Changes'}
                </button>
            </div>
        </div>
    );
}
