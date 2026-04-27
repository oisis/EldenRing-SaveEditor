// Ban-risk awareness dictionary.
// Phase 1: per-flag baseline (4 entries). Per-action keys are added in later phases.
//
// Editorial guidance:
//   - Frame statements as community-reported, not officially confirmed by FromSoftware.
//   - "Why ban?" explains the detection mechanism in plain English.
//   - "Reports" cites volume/recency without inventing numbers.
//   - "Mitigation" gives the user a concrete way to reduce risk.
//   - Source URLs are optional — leave empty when not verified.

export type RiskLevel = 'low' | 'medium' | 'high';
export type RiskTier = 0 | 1 | 2;

export interface RiskSource {
    label: string;
    url?: string;
}

export interface RiskEntry {
    tier: RiskTier;
    level: RiskLevel;
    title: string;
    whyBan: string;
    reports: string;
    mitigation: string;
    sources: RiskSource[];
}

export type RiskKey =
    | 'cut_content'
    | 'pre_order'
    | 'dlc_duplicate'
    | 'ban_risk';

export const RISK_INFO: Record<RiskKey, RiskEntry> = {
    cut_content: {
        tier: 2,
        level: 'high',
        title: 'Cut Content',
        whyBan:
            'These item IDs exist in the game data but were never released to retail. They cannot be obtained through normal play. Easy Anti-Cheat treats their presence as injected content because no legitimate progression can produce them.',
        reports:
            'Multiple ban reports across r/Eldenring and Discord communities (2022-2024) for cut armor sets, prototype talismans, and unfinished key items. Bans typically follow the first online connection after the edit.',
        mitigation:
            'Use only on save copies for offline experimentation. Remove cut items from inventory and storage before going online — for Bell Bearings and key items also clear the matching event flag, otherwise the acquisition record persists.',
        sources: [
            {label: 'r/Eldenring ban discussion threads (2022-2024)'},
            {label: 'Fextralife wiki — cut content notes'},
        ],
    },
    pre_order: {
        tier: 2,
        level: 'medium',
        title: 'Pre-Order Bonus',
        whyBan:
            'Items granted only with the pre-order edition of the game (or a specific bundle). Easy Anti-Cheat checks the account entitlement when these items appear in the save. If the account does not own the corresponding entitlement, the item is treated as injected.',
        reports:
            'Lower volume than cut content but consistently reported. Pre-order rings (e.g. Ring of Miquella variants) and the Carian Oath gesture are the most common offenders.',
        mitigation:
            'Safe if the account owns the pre-order entitlement. Otherwise do not add. Removing the item from inventory may not clear the acquisition flag — verify before going online.',
        sources: [
            {label: 'r/Eldenring pre-order item discussions'},
        ],
    },
    dlc_duplicate: {
        tier: 2,
        level: 'medium',
        title: 'DLC Duplicate ID',
        whyBan:
            'Some IDs were duplicated when Shadow of the Erdtree integrated, leaving two variants of the same item with different internal codes. Using the wrong variant (legacy ID without DLC, or DLC variant on a non-DLC account) can produce a save state inconsistent with the player\'s entitlements.',
        reports:
            'Sporadic. Mostly affects gestures (e.g. Ring of Miquella alternate slot) and a few duplicated key items.',
        mitigation:
            'Prefer the DLC-active variant if you own Shadow of the Erdtree, otherwise use the base game variant. When in doubt, do not add and pick the canonical equivalent instead.',
        sources: [
            {label: 'er-save-manager DLC duplicate notes'},
        ],
    },
    ban_risk: {
        tier: 2,
        level: 'high',
        title: 'Ban Risk (Generic)',
        whyBan:
            'This item or action has been associated with ban reports for reasons that may include cut content, illegal stat values, impossible game states, or detection rules whose exact mechanism is not publicly documented.',
        reports:
            'Aggregate flag used when the specific cause is unclear or overlaps multiple categories. Treat as worst-case until a more specific entry is available.',
        mitigation:
            'High-risk by default. Use only offline. Remove or revert the change before connecting online.',
        sources: [
            {label: 'Aggregate community ban reports'},
        ],
    },
};
