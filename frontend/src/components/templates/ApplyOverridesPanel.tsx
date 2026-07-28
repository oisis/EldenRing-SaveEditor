import { ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { GetCharacter, GetStartingClasses } from '../../../wailsjs/go/main/App';
import { db, vm } from '../../../wailsjs/go/models';
import { StatSlider } from '../StatSlider';
import {
    calculateLevel,
    minimumSoulMemoryForLevel,
    STAT_KEYS,
    StatBlock,
} from '../../lib/characterProgression';
import { useModalEscape } from './useModalEscape';
import { WeaponLevelOverridePanel, WeaponOverridePayload } from './WeaponLevelOverridePanel';

export type ProfileOverrideKey =
    | 'name'
    | 'runes'
    | 'soulMemory'
    | 'clearCount'
    | 'scadutreeBlessing'
    | 'shadowRealmBlessing'
    | 'talismanSlots';

export type StatsOverrideKey = keyof StatBlock;

type NumericRange = { min: number; max: number };

interface FieldMeta<K extends string> {
    key: K;
    label: string;
    kind: 'integer' | 'text';
    range?: NumericRange;
    softCap?: number;
    hint?: string;
}

export const OVERRIDABLE_PROFILE_FIELDS: ReadonlyArray<FieldMeta<ProfileOverrideKey>> = [
    { key: 'name', label: 'Name', kind: 'text', hint: 'UTF-16 ≤ 16 code units (backend enforces).' },
    {
        key: 'runes',
        label: 'Runes',
        kind: 'integer',
        range: { min: 0, max: 4_294_967_295 },
        softCap: 999_000_000,
        hint: 'Above the soft cap is unusual for vanilla saves.',
    },
    {
        key: 'soulMemory',
        label: 'Soul Memory',
        kind: 'integer',
        range: { min: 0, max: 4_294_967_295 },
    },
    { key: 'clearCount', label: 'NG+ Cycle', kind: 'integer', range: { min: 0, max: 7 } },
    { key: 'scadutreeBlessing', label: 'Scadutree Blessing', kind: 'integer', range: { min: 0, max: 20 } },
    { key: 'shadowRealmBlessing', label: 'Shadow Realm Blessing', kind: 'integer', range: { min: 0, max: 10 } },
    { key: 'talismanSlots', label: 'Talisman Slots', kind: 'integer', range: { min: 0, max: 3 } },
];

export const OVERRIDABLE_STATS_FIELDS: ReadonlyArray<FieldMeta<StatsOverrideKey>> = [
    { key: 'vigor', label: 'Vigor', kind: 'integer', range: { min: 1, max: 99 } },
    { key: 'mind', label: 'Mind', kind: 'integer', range: { min: 1, max: 99 } },
    { key: 'endurance', label: 'Endurance', kind: 'integer', range: { min: 1, max: 99 } },
    { key: 'strength', label: 'Strength', kind: 'integer', range: { min: 1, max: 99 } },
    { key: 'dexterity', label: 'Dexterity', kind: 'integer', range: { min: 1, max: 99 } },
    { key: 'intelligence', label: 'Intelligence', kind: 'integer', range: { min: 1, max: 99 } },
    { key: 'faith', label: 'Faith', kind: 'integer', range: { min: 1, max: 99 } },
    { key: 'arcane', label: 'Arcane', kind: 'integer', range: { min: 1, max: 99 } },
];

interface FieldDraft {
    enabled: boolean;
    value: string;
    presentInitially: boolean;
    initialValue: string;
}

interface ClassDraft {
    enabled: boolean;
    classID: number;
    initialEnabled: boolean;
    initialClassID: number;
}

export interface DraftState {
    profile: Record<string, FieldDraft>;
    stats: Record<string, FieldDraft>;
    class?: ClassDraft;
}

interface ParsedTemplate {
    sections?: { profile?: Record<string, unknown>; stats?: Record<string, unknown> } & Record<string, unknown>;
    selection?: { profile?: Record<string, unknown>; stats?: Record<string, unknown> } & Record<string, unknown>;
    [k: string]: unknown;
}

export interface CharacterOverridePayload {
    deriveLevelFromStats: true;
    classOverride?: { classID: number };
    calculatedLevel: number;
    effectiveSoulMemory: number;
}

export interface MutatedOverrideResult {
    json: string | null;
    hasInvalid: boolean;
    hasOverrides: boolean;
    fieldErrors: Record<string, string>;
}

interface OverrideContext {
    calculatedLevel?: number;
    minimumSoulMemory?: number;
    classes?: db.ClassStats[];
}

function safeParse(canonicalJSON: string): ParsedTemplate | null {
    try {
        const obj = JSON.parse(canonicalJSON);
        if (obj === null || typeof obj !== 'object' || Array.isArray(obj)) return null;
        return obj as ParsedTemplate;
    } catch {
        return null;
    }
}

function rawToString(raw: unknown): string {
    if (raw === null || raw === undefined) return '';
    if (typeof raw === 'string') return raw;
    if (typeof raw === 'number') return String(raw);
    return '';
}

function fieldIsPresent(section: Record<string, unknown> | undefined, key: string): boolean {
    if (!section) return false;
    return section[key] !== undefined && section[key] !== null;
}

function initDraft(
    parsed: ParsedTemplate,
    fields: ReadonlyArray<FieldMeta<string>>,
    sectionKey: 'profile' | 'stats',
    fallbacks: Record<string, string> = {},
): Record<string, FieldDraft> {
    const sec = parsed.sections?.[sectionKey] as Record<string, unknown> | undefined;
    const sel = parsed.selection?.[sectionKey] as Record<string, unknown> | undefined;
    const out: Record<string, FieldDraft> = {};
    for (const field of fields) {
        const present = fieldIsPresent(sec, field.key);
        const initial = present ? rawToString(sec?.[field.key]) : (fallbacks[field.key] ?? '');
        out[field.key] = {
            enabled: !!sel?.[field.key] && present,
            value: initial,
            presentInitially: present,
            initialValue: initial,
        };
    }
    return out;
}

function classByName(classes: db.ClassStats[], name: string): db.ClassStats | undefined {
    return classes.find(candidate => candidate.name === name);
}

function initClassDraft(
    parsed: ParsedTemplate,
    character: vm.CharacterViewModel,
    classes: db.ClassStats[],
): ClassDraft {
    const profile = parsed.sections?.profile as Record<string, unknown> | undefined;
    const selection = parsed.selection?.profile as Record<string, unknown> | undefined;
    const templateName = typeof profile?.class === 'string' ? profile.class : '';
    const templateClass = classByName(classes, templateName);
    const classID = templateClass?.id ?? character.class;
    const enabled = !!selection?.class && !!templateClass;
    return {
        enabled,
        classID,
        initialEnabled: enabled,
        initialClassID: classID,
    };
}

function parseInteger(text: string): number | null {
    const trimmed = text.trim();
    if (!/^-?\d+$/.test(trimmed)) return null;
    const value = Number(trimmed);
    return Number.isSafeInteger(value) ? value : null;
}

function validateField(meta: FieldMeta<string>, draft: FieldDraft): { value: number | string | null; error?: string } {
    if (!draft.enabled) return { value: null };
    if (meta.kind === 'text') {
        if (draft.value.trim() === '') return { value: null, error: 'Value required.' };
        return { value: draft.value };
    }
    const value = parseInteger(draft.value);
    if (value === null) return { value: null, error: 'Integer required.' };
    if (meta.range && (value < meta.range.min || value > meta.range.max)) {
        return { value: null, error: `Must be ${meta.range.min}–${meta.range.max}.` };
    }
    return { value };
}

function cleanupSection(next: ParsedTemplate, sectionKey: 'profile' | 'stats') {
    const sec = next.sections?.[sectionKey] as Record<string, unknown> | undefined;
    const sel = next.selection?.[sectionKey] as Record<string, unknown> | undefined;
    if (sec && Object.keys(sec).length === 0) delete (next.sections as Record<string, unknown>)[sectionKey];
    if (sel && Object.keys(sel).length === 0) delete (next.selection as Record<string, unknown>)[sectionKey];
}

// Applies only the canonical profile/stats edits. Level is deliberately
// removed: Apply-with-overrides derives it in the backend from final stats.
export function applyOverridesToCanonical(
    canonicalJSON: string,
    draft: DraftState,
    context: OverrideContext = {},
): MutatedOverrideResult {
    const parsed = safeParse(canonicalJSON);
    if (!parsed) return { json: null, hasInvalid: true, hasOverrides: false, fieldErrors: {} };

    const fieldErrors: Record<string, string> = {};
    const validated: Record<'profile' | 'stats', Record<string, number | string | null>> = {
        profile: {},
        stats: {},
    };

    for (const [section, fields] of [
        ['profile', OVERRIDABLE_PROFILE_FIELDS],
        ['stats', OVERRIDABLE_STATS_FIELDS],
    ] as const) {
        for (const meta of fields) {
            const current = draft[section][meta.key];
            if (!current) continue;
            const result = validateField(meta, current);
            validated[section][meta.key] = result.value;
            if (result.error) fieldErrors[`${section}.${meta.key}`] = result.error;
        }
    }

    const soulMemory = draft.profile.soulMemory;
    const soulMemoryValue = soulMemory?.enabled ? parseInteger(soulMemory.value) : null;
    if (
        soulMemory?.enabled &&
        soulMemoryValue !== null &&
        context.minimumSoulMemory !== undefined &&
        soulMemoryValue < context.minimumSoulMemory
    ) {
        fieldErrors['profile.soulMemory'] =
            `Must be at least ${context.minimumSoulMemory} for calculated level ${context.calculatedLevel}.`;
    }

    if (draft.class?.enabled && !context.classes?.some(candidate => candidate.id === draft.class?.classID)) {
        fieldErrors['profile.class'] = 'Select a valid starting class.';
    }
    if (Object.keys(fieldErrors).length > 0) {
        return { json: null, hasInvalid: true, hasOverrides: false, fieldErrors };
    }

    const next = JSON.parse(canonicalJSON) as ParsedTemplate;
    next.sections = (next.sections ?? {}) as ParsedTemplate['sections'];
    next.selection = (next.selection ?? {}) as ParsedTemplate['selection'];
    let hasOverrides = false;

    const writeSection = (section: 'profile' | 'stats', fields: ReadonlyArray<FieldMeta<string>>) => {
        const sec = ((next.sections![section] as Record<string, unknown> | undefined) ?? {});
        const sel = ((next.selection![section] as Record<string, unknown> | undefined) ?? {});
        let touched = false;
        for (const meta of fields) {
            const current = draft[section][meta.key];
            if (!current) continue;
            if (current.enabled) {
                sec[meta.key] = validated[section][meta.key];
                sel[meta.key] = true;
                touched = true;
                if (!current.presentInitially || current.value !== current.initialValue) hasOverrides = true;
            } else if (meta.key in sec || meta.key in sel) {
                delete sec[meta.key];
                delete sel[meta.key];
                touched = true;
                if (current.presentInitially) hasOverrides = true;
            }
        }
        if (touched) {
            (next.sections as Record<string, unknown>)[section] = sec;
            (next.selection as Record<string, unknown>)[section] = sel;
        }
    };

    writeSection('profile', OVERRIDABLE_PROFILE_FIELDS);
    writeSection('stats', OVERRIDABLE_STATS_FIELDS);

    // Manual level is not an Apply-with-overrides input or source of truth.
    const profileSec = ((next.sections!.profile as Record<string, unknown> | undefined) ?? {});
    const profileSel = ((next.selection!.profile as Record<string, unknown> | undefined) ?? {});
    if ('level' in profileSec || 'level' in profileSel) hasOverrides = true;
    delete profileSec.level;
    delete profileSel.level;

    if (draft.class) {
        const selectedClass = context.classes?.find(candidate => candidate.id === draft.class?.classID);
        if (draft.class.enabled && selectedClass) {
            profileSec.class = selectedClass.name;
            profileSel.class = true;
        } else {
            delete profileSec.class;
            delete profileSel.class;
        }
        if (
            draft.class.enabled !== draft.class.initialEnabled ||
            draft.class.classID !== draft.class.initialClassID
        ) {
            hasOverrides = true;
        }
    }
    (next.sections as Record<string, unknown>).profile = profileSec;
    (next.selection as Record<string, unknown>).profile = profileSel;
    cleanupSection(next, 'profile');
    cleanupSection(next, 'stats');

    return {
        json: JSON.stringify(next),
        hasInvalid: false,
        hasOverrides,
        fieldErrors,
    };
}

interface ApplyOverridesPanelProps {
    canonicalJSON: string;
    character: vm.CharacterViewModel;
    startingClasses: db.ClassStats[];
    onMutatedChange: (
        mutated: string | null,
        hasInvalid: boolean,
        fieldErrors: Record<string, string>,
        runtime: CharacterOverridePayload,
    ) => void;
    disabled?: boolean;
}

function characterStats(character: vm.CharacterViewModel): StatBlock {
    return {
        vigor: character.vigor,
        mind: character.mind,
        endurance: character.endurance,
        strength: character.strength,
        dexterity: character.dexterity,
        intelligence: character.intelligence,
        faith: character.faith,
        arcane: character.arcane,
    };
}

function statFallbacks(character: vm.CharacterViewModel): Record<string, string> {
    const out: Record<string, string> = {};
    for (const key of STAT_KEYS) out[key] = String(character[key]);
    return out;
}

export function ApplyOverridesPanel({
    canonicalJSON,
    character,
    startingClasses,
    onMutatedChange,
    disabled,
}: ApplyOverridesPanelProps) {
    const parsed = useMemo(() => safeParse(canonicalJSON), [canonicalJSON]);
    const makeDraft = useCallback((): DraftState => {
        if (!parsed) return { profile: {}, stats: {} };
        return {
            profile: initDraft(parsed, OVERRIDABLE_PROFILE_FIELDS, 'profile', {
                name: character.name,
                runes: String(character.souls),
                soulMemory: String(character.soulMemory),
                clearCount: String(character.clearCount),
                scadutreeBlessing: String(character.scadutreeBlessing),
                shadowRealmBlessing: String(character.shadowRealmBlessing),
                talismanSlots: String(character.talismanSlots),
            }),
            stats: initDraft(parsed, OVERRIDABLE_STATS_FIELDS, 'stats', statFallbacks(character)),
            class: initClassDraft(parsed, character, startingClasses),
        };
    }, [parsed, character, startingClasses]);

    const [draft, setDraft] = useState<DraftState>(makeDraft);
    useEffect(() => setDraft(makeDraft()), [makeDraft]);

    const selectedClass = useMemo(() => {
        const classID = draft.class?.enabled ? draft.class.classID : character.class;
        return startingClasses.find(candidate => candidate.id === classID);
    }, [draft.class, character.class, startingClasses]);

    const effectiveStats = useMemo(() => {
        const current = characterStats(character);
        const out = { ...current };
        for (const key of STAT_KEYS) {
            const field = draft.stats[key];
            if (!field?.enabled) continue;
            const value = parseInteger(field.value);
            if (value !== null) out[key] = value;
        }
        return out;
    }, [draft.stats, character]);

    const calculatedLevel = calculateLevel(effectiveStats);
    const minimumSoulMemory = minimumSoulMemoryForLevel(calculatedLevel);
    const selectedSoulMemory = draft.profile.soulMemory?.enabled
        ? parseInteger(draft.profile.soulMemory.value)
        : null;
    const effectiveSoulMemory = Math.max(
        selectedSoulMemory ?? character.soulMemory,
        minimumSoulMemory,
    );
    const liveResult = useMemo(() => applyOverridesToCanonical(canonicalJSON, draft, {
        calculatedLevel,
        minimumSoulMemory,
        classes: startingClasses,
    }), [canonicalJSON, draft, calculatedLevel, minimumSoulMemory, startingClasses]);
    const runtime = useMemo<CharacterOverridePayload>(() => ({
        deriveLevelFromStats: true,
        calculatedLevel,
        effectiveSoulMemory,
        ...(draft.class?.enabled ? { classOverride: { classID: draft.class.classID } } : {}),
    }), [calculatedLevel, effectiveSoulMemory, draft.class]);

    useEffect(() => {
        onMutatedChange(liveResult.json, liveResult.hasInvalid, liveResult.fieldErrors, runtime);
    }, [
        liveResult.json,
        liveResult.hasInvalid,
        liveResult.fieldErrors,
        runtime,
        onMutatedChange,
    ]);

    const onToggleProfile = useCallback((key: string) => {
        setDraft(previous => {
            const current = previous.profile[key];
            if (!current) return previous;
            return {
                ...previous,
                profile: {
                    ...previous.profile,
                    [key]: { ...current, enabled: !current.enabled },
                },
            };
        });
    }, []);

    const onProfileValue = useCallback((key: string, value: string) => {
        setDraft(previous => {
            const current = previous.profile[key];
            if (!current) return previous;
            return {
                ...previous,
                profile: { ...previous.profile, [key]: { ...current, value } },
            };
        });
    }, []);

    const onToggleStat = useCallback((key: StatsOverrideKey) => {
        setDraft(previous => {
            const current = previous.stats[key];
            if (!current) return previous;
            return {
                ...previous,
                stats: { ...previous.stats, [key]: { ...current, enabled: !current.enabled } },
            };
        });
    }, []);

    const onStatValue = useCallback((key: StatsOverrideKey, value: number) => {
        const minimum = selectedClass?.[key] ?? 1;
        const normalized = Math.min(99, Math.max(minimum, value));
        setDraft(previous => {
            const current = previous.stats[key];
            if (!current) return previous;
            return {
                ...previous,
                stats: { ...previous.stats, [key]: { ...current, value: String(normalized) } },
            };
        });
    }, [selectedClass]);

    const applyClassMinimums = useCallback((classID: number, enableClass: boolean) => {
        const nextClass = startingClasses.find(candidate => candidate.id === classID);
        if (!nextClass) return;
        setDraft(previous => {
            const stats = { ...previous.stats };
            for (const key of STAT_KEYS) {
                const current = stats[key];
                if (!current) continue;
                const effective = current.enabled
                    ? (parseInteger(current.value) ?? character[key])
                    : character[key];
                const minimum = nextClass[key];
                if (effective < minimum) {
                    stats[key] = { ...current, enabled: true, value: String(minimum) };
                }
            }
            return {
                ...previous,
                class: {
                    ...(previous.class ?? {
                        classID,
                        initialClassID: character.class,
                        initialEnabled: false,
                    }),
                    enabled: enableClass,
                    classID,
                },
                stats,
            };
        });
    }, [startingClasses, character]);

    if (!parsed) {
        return (
            <div
                data-testid="apply-overrides-panel"
                data-state="invalid-json"
                className="px-3 py-3 text-[11px] text-red-300"
            >
                Could not parse template JSON — overrides cannot be edited.
            </div>
        );
    }

    return (
        <div data-testid="apply-overrides-panel" className="space-y-4 text-[12px]">
            <ProfileGrid
                drafts={draft.profile}
                errors={liveResult.fieldErrors}
                disabled={disabled}
                minimumSoulMemory={minimumSoulMemory}
                calculatedLevel={calculatedLevel}
                effectiveSoulMemory={effectiveSoulMemory}
                onToggle={onToggleProfile}
                onValue={onProfileValue}
            >
                <ClassOverrideRow
                    draft={draft.class!}
                    classes={startingClasses}
                    error={liveResult.fieldErrors['profile.class']}
                    disabled={disabled}
                    onEnabledChange={enabled => {
                        if (enabled) applyClassMinimums(draft.class!.classID, true);
                        else setDraft(previous => ({
                            ...previous,
                            class: { ...previous.class!, enabled: false },
                        }));
                    }}
                    onClassChange={classID => applyClassMinimums(classID, true)}
                />
            </ProfileGrid>

            <section aria-label="Stats overrides" className="space-y-1.5">
                <div className="flex items-center justify-between gap-3">
                    <h3 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                        Stats
                    </h3>
                    <div
                        data-testid="apply-overrides-calculated-level"
                        className="text-[11px] font-bold text-primary"
                    >
                        Calculated Level: {calculatedLevel}
                    </div>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-x-6">
                    {OVERRIDABLE_STATS_FIELDS.map(meta => {
                        const field = draft.stats[meta.key];
                        const minimum = selectedClass?.[meta.key] ?? 1;
                        const requiredByClass =
                            !!draft.class?.enabled && character[meta.key] < minimum;
                        return (
                            <StatSlider
                                key={meta.key}
                                label={meta.label}
                                value={parseInteger(field.value) ?? character[meta.key]}
                                min={minimum}
                                disabled={disabled || !field.enabled}
                                onChange={value => onStatValue(meta.key, value)}
                                rangeTestId={`apply-overrides-stats-range-${meta.key}`}
                                numberTestId={`apply-overrides-stats-input-${meta.key}`}
                                leading={
                                    <input
                                        type="checkbox"
                                        data-testid={`apply-overrides-stats-toggle-${meta.key}`}
                                        checked={field.enabled}
                                        onChange={() => onToggleStat(meta.key)}
                                        disabled={disabled || requiredByClass}
                                        title={requiredByClass
                                            ? `${meta.label} is required by the selected class minimum.`
                                            : undefined}
                                        aria-label={`Apply ${meta.label}`}
                                    />
                                }
                            />
                        );
                    })}
                </div>
            </section>
        </div>
    );
}

interface ProfileGridProps {
    drafts: Record<string, FieldDraft>;
    errors: Record<string, string>;
    disabled?: boolean;
    minimumSoulMemory: number;
    calculatedLevel: number;
    effectiveSoulMemory: number;
    onToggle: (key: string) => void;
    onValue: (key: string, value: string) => void;
    children: ReactNode;
}

function ProfileGrid({
    drafts,
    errors,
    disabled,
    minimumSoulMemory,
    calculatedLevel,
    effectiveSoulMemory,
    onToggle,
    onValue,
    children,
}: ProfileGridProps) {
    return (
        <section aria-label="Profile overrides" className="space-y-1.5">
            <h3 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Profile</h3>
            <ul className="grid grid-cols-1 gap-1.5">
                {OVERRIDABLE_PROFILE_FIELDS.map(meta => {
                    const field = drafts[meta.key];
                    const error = errors[`profile.${meta.key}`];
                    const dynamicSoulMemoryHint = meta.key === 'soulMemory'
                        ? `Minimum ${minimumSoulMemory} for calculated level ${calculatedLevel}. Effective: ${effectiveSoulMemory}.`
                        : '';
                    const rangeHint = dynamicSoulMemoryHint || (meta.range
                        ? `${meta.range.min}–${meta.range.max}`
                        : meta.hint ?? '');
                    const parsedValue = parseInteger(field.value);
                    const softWarning =
                        field.enabled && meta.softCap && !error && parsedValue !== null && parsedValue > meta.softCap
                            ? `Above the soft cap (${meta.softCap}).`
                            : '';
                    return (
                        <li
                            key={meta.key}
                            data-testid={`apply-overrides-profile-row-${meta.key}`}
                            className="grid grid-cols-[18px_140px_1fr_170px] items-center gap-2"
                        >
                            <input
                                type="checkbox"
                                data-testid={`apply-overrides-profile-toggle-${meta.key}`}
                                checked={field.enabled}
                                onChange={() => onToggle(meta.key)}
                                disabled={disabled}
                                aria-label={`Apply ${meta.label}`}
                            />
                            <label htmlFor={`apply-overrides-profile-input-${meta.key}`}>
                                {meta.label}
                            </label>
                            <input
                                id={`apply-overrides-profile-input-${meta.key}`}
                                data-testid={`apply-overrides-profile-input-${meta.key}`}
                                type={meta.kind === 'integer' ? 'text' : 'text'}
                                inputMode={meta.kind === 'integer' ? 'numeric' : 'text'}
                                value={field.value}
                                onChange={event => onValue(meta.key, event.target.value)}
                                disabled={disabled || !field.enabled}
                                aria-invalid={!!error}
                                className={`rounded border px-2 py-1 text-foreground bg-background/40 ${
                                    error ? 'border-red-500/60' : 'border-border/60'
                                } disabled:opacity-40`}
                            />
                            <div className="text-[10px] text-muted-foreground">
                                {rangeHint && (
                                    <div data-testid={`apply-overrides-profile-range-${meta.key}`}>
                                        {rangeHint}
                                    </div>
                                )}
                                {error && (
                                    <div
                                        data-testid={`apply-overrides-profile-error-${meta.key}`}
                                        className="text-red-300"
                                    >
                                        {error}
                                    </div>
                                )}
                                {!error && softWarning && (
                                    <div
                                        data-testid={`apply-overrides-profile-soft-warning-${meta.key}`}
                                        className="text-warning-foreground"
                                    >
                                        {softWarning}
                                    </div>
                                )}
                            </div>
                        </li>
                    );
                })}
            </ul>
            {children}
        </section>
    );
}

interface ClassOverrideRowProps {
    draft: ClassDraft;
    classes: db.ClassStats[];
    error?: string;
    disabled?: boolean;
    onEnabledChange: (enabled: boolean) => void;
    onClassChange: (classID: number) => void;
}

function ClassOverrideRow({
    draft,
    classes,
    error,
    disabled,
    onEnabledChange,
    onClassChange,
}: ClassOverrideRowProps) {
    return (
        <div
            data-testid="apply-overrides-profile-class"
            className="grid grid-cols-[18px_140px_1fr_170px] items-center gap-2"
        >
            <input
                type="checkbox"
                data-testid="apply-overrides-profile-toggle-class"
                checked={draft.enabled}
                onChange={event => onEnabledChange(event.target.checked)}
                disabled={disabled}
                aria-label="Apply Class"
            />
            <label htmlFor="apply-overrides-profile-input-class">Class</label>
            <select
                id="apply-overrides-profile-input-class"
                data-testid="apply-overrides-profile-input-class"
                value={draft.classID}
                onChange={event => onClassChange(Number(event.target.value))}
                disabled={disabled || !draft.enabled}
                aria-invalid={!!error}
                className={`rounded border px-2 py-1 bg-background/40 ${
                    error ? 'border-red-500/60' : 'border-border/60'
                } disabled:opacity-40`}
            >
                {classes.map(candidate => (
                    <option key={candidate.id} value={candidate.id}>{candidate.name}</option>
                ))}
            </select>
            <div className="text-[10px] text-muted-foreground">
                {error ?? 'Changing class raises any lower stats to its minimums.'}
            </div>
        </div>
    );
}

interface ApplyOverridesModalProps {
    sourceLabel: string;
    canonicalJSON: string;
    charIndex?: number;
    character?: vm.CharacterViewModel;
    startingClasses?: db.ClassStats[];
    onCancel: () => void;
    onConfirm: (
        mutatedJSON: string,
        weaponOverride: WeaponOverridePayload,
        characterOverride: CharacterOverridePayload,
    ) => void | Promise<void>;
    applying?: boolean;
}

export function ApplyOverridesModal({
    sourceLabel,
    canonicalJSON,
    charIndex,
    character: suppliedCharacter,
    startingClasses: suppliedClasses,
    onCancel,
    onConfirm,
    applying,
}: ApplyOverridesModalProps) {
    const dialogRef = useRef<HTMLDivElement | null>(null);
    const [character, setCharacter] = useState<vm.CharacterViewModel | null>(suppliedCharacter ?? null);
    const [startingClasses, setStartingClasses] = useState<db.ClassStats[]>(suppliedClasses ?? []);
    const [contextLoading, setContextLoading] = useState(!suppliedCharacter || !suppliedClasses);
    const [contextError, setContextError] = useState('');
    const [mutatedJSON, setMutatedJSON] = useState<string | null>(null);
    const [hasInvalid, setHasInvalid] = useState(false);
    const [errorCount, setErrorCount] = useState(0);
    const [characterOverride, setCharacterOverride] = useState<CharacterOverridePayload | null>(null);
    const [weaponOverride, setWeaponOverride] = useState<WeaponOverridePayload>(undefined);
    const [weaponInvalid, setWeaponInvalid] = useState(false);
    const onDialogKeyDown = useModalEscape(onCancel, !!applying);

    useEffect(() => {
        dialogRef.current?.focus();
    }, []);

    useEffect(() => {
        if (suppliedCharacter && suppliedClasses) {
            setCharacter(suppliedCharacter);
            setStartingClasses(suppliedClasses);
            setContextLoading(false);
            setContextError('');
            return;
        }
        if (charIndex === undefined) {
            setContextLoading(false);
            setContextError('No target character is available for overrides.');
            return;
        }
        let cancelled = false;
        setContextLoading(true);
        Promise.all([GetCharacter(charIndex), GetStartingClasses()])
            .then(([loadedCharacter, loadedClasses]) => {
                if (cancelled) return;
                setCharacter(loadedCharacter);
                setStartingClasses(loadedClasses);
                setContextError(
                    loadedClasses.length > 0
                        ? ''
                        : 'Starting-class data is unavailable for overrides.',
                );
            })
            .catch(error => {
                if (!cancelled) setContextError(`Could not load character context: ${String(error)}`);
            })
            .finally(() => {
                if (!cancelled) setContextLoading(false);
            });
        return () => { cancelled = true; };
    }, [charIndex, suppliedCharacter, suppliedClasses]);

    const showWeaponOverride = useMemo(() => {
        const parsed = safeParse(canonicalJSON);
        const selection = parsed?.selection as Record<string, unknown> | undefined;
        if (!selection) return false;
        const active = (raw: unknown): boolean => {
            if (typeof raw === 'boolean') return raw;
            if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
                const obj = raw as Record<string, unknown>;
                return obj.all === true || Object.keys(obj).length > 0;
            }
            return false;
        };
        return active(selection['inventory.workspace']) || active(selection.items);
    }, [canonicalJSON]);

    const hasItemsSelection = useMemo(() => {
        const selection = safeParse(canonicalJSON)?.selection as Record<string, unknown> | undefined;
        const raw = selection?.items;
        if (typeof raw === 'boolean') return raw;
        return !!raw && typeof raw === 'object' && !Array.isArray(raw) && Object.keys(raw as object).length > 0;
    }, [canonicalJSON]);

    const hasLayoutSelection = useMemo(() => {
        const selection = safeParse(canonicalJSON)?.selection as Record<string, unknown> | undefined;
        return !!selection?.inventoryLayout || !!selection?.storageLayout;
    }, [canonicalJSON]);

    const handleMutated = useCallback((
        json: string | null,
        invalid: boolean,
        fieldErrors: Record<string, string>,
        runtime: CharacterOverridePayload,
    ) => {
        setMutatedJSON(json);
        setHasInvalid(invalid);
        setErrorCount(Object.keys(fieldErrors).length);
        setCharacterOverride(runtime);
    }, []);

    const handleWeaponChange = useCallback((override: WeaponOverridePayload, invalid: boolean) => {
        setWeaponOverride(override);
        setWeaponInvalid(invalid);
    }, []);

    const canApply =
        !applying &&
        !contextLoading &&
        contextError === '' &&
        !hasInvalid &&
        !weaponInvalid &&
        mutatedJSON !== null &&
        characterOverride !== null;

    return (
        <div
            data-testid="apply-overrides-modal"
            role="dialog"
            aria-modal="true"
            aria-label="Apply Build Template with Overrides"
            ref={dialogRef}
            tabIndex={-1}
            onKeyDown={onDialogKeyDown}
            className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4"
        >
            <div className="w-full max-w-2xl rounded-lg bg-card border border-border/60 shadow-xl flex flex-col max-h-[85vh]">
                <div className="px-4 py-3 border-b border-border/60">
                    <h2 className="text-sm font-black uppercase tracking-wider">Apply with overrides</h2>
                    <p data-testid="apply-overrides-source-label" className="mt-1 text-[11px] text-muted-foreground break-all">
                        {sourceLabel}
                    </p>
                    <p className="mt-1 text-[11px] text-muted-foreground">
                        Choose the final class and attributes. Level is calculated from the eight effective stats;
                        the backend validates class minimums and Soul Memory before any mutation.
                    </p>
                </div>

                <div className="px-4 py-3 overflow-y-auto space-y-4">
                    {contextLoading && (
                        <div data-testid="apply-overrides-context-loading" className="text-[11px] text-muted-foreground">
                            Loading character context…
                        </div>
                    )}
                    {contextError && (
                        <div data-testid="apply-overrides-context-error" className="text-[11px] text-red-300">
                            {contextError}
                        </div>
                    )}
                    {character && startingClasses.length > 0 && (
                        <ApplyOverridesPanel
                            canonicalJSON={canonicalJSON}
                            character={character}
                            startingClasses={startingClasses}
                            onMutatedChange={handleMutated}
                            disabled={applying}
                        />
                    )}
                    {hasItemsSelection && (
                        <section
                            data-testid="apply-overrides-items-mode"
                            aria-label="Items apply mode"
                            className="space-y-1 rounded border border-border/40 bg-background/30 px-3 py-2"
                        >
                            <h3 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                                Items mode
                            </h3>
                            <div className="text-[11px]">
                                <span className="font-bold">Add missing only</span>
                                <span className="text-muted-foreground">
                                    {' '}— inserts template items the character does not already own. Nothing is deleted or replaced.
                                </span>
                            </div>
                            {hasLayoutSelection && (
                                <div
                                    data-testid="apply-overrides-items-layout-ignored"
                                    className="text-[10px] text-warning-foreground/90"
                                >
                                    Inventory / storage layout sections will be ignored — layout apply is not supported in Phase 8D.2.
                                </div>
                            )}
                        </section>
                    )}
                    {showWeaponOverride && (
                        <WeaponLevelOverridePanel onChange={handleWeaponChange} disabled={applying} />
                    )}
                </div>

                <div className="px-4 py-3 border-t border-border/60 flex items-center justify-between gap-2">
                    <div data-testid="apply-overrides-status" className="text-[10px] text-muted-foreground">
                        {contextLoading
                            ? 'Loading character context.'
                            : contextError
                              ? 'Character context is unavailable.'
                              : hasInvalid
                                ? `${errorCount} field${errorCount === 1 ? '' : 's'} need attention.`
                                : weaponInvalid
                                  ? 'Fix weapon level override to apply.'
                                  : 'Ready to apply.'}
                    </div>
                    <div className="flex items-center gap-2">
                        <button
                            type="button"
                            data-testid="apply-overrides-cancel"
                            onClick={onCancel}
                            disabled={applying}
                            className="px-3 py-1 text-[10px] font-black uppercase tracking-wider rounded border border-border/60 text-muted-foreground hover:text-foreground hover:bg-muted/40 transition-all disabled:opacity-40"
                        >
                            Cancel
                        </button>
                        <button
                            type="button"
                            data-testid="apply-overrides-apply"
                            onClick={() => {
                                if (canApply && mutatedJSON && characterOverride) {
                                    void onConfirm(mutatedJSON, weaponOverride, characterOverride);
                                }
                            }}
                            disabled={!canApply}
                            title={canApply ? 'Apply with current values.' : 'Fix invalid values to apply.'}
                            aria-label={canApply ? 'Apply with current values.' : 'Fix invalid values to apply.'}
                            className={`px-3 py-1 text-[10px] font-black uppercase tracking-wider rounded transition-all ${
                                canApply
                                    ? 'bg-green-700/80 text-white hover:bg-green-700 shadow-sm'
                                    : 'opacity-40 cursor-not-allowed bg-muted/20 text-muted-foreground'
                            }`}
                        >
                            {applying ? 'Applying…' : 'Apply to character'}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
}
