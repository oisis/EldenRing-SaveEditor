import {RiskInfoIcon} from './RiskInfoIcon';
import {RiskKey} from '../data/riskInfo';

interface Props {
    flag: RiskKey;
    showInfoIcon?: boolean;
    className?: string;
}

interface BadgeStyle {
    label: string;
    classes: string;
}

// Only the per-flag RiskKeys have a dedicated badge label. Per-action keys
// (stat_above_99, runes_above_999m, ...) surface only the (?) icon, not a badge.
const STYLE: Partial<Record<RiskKey, BadgeStyle>> = {
    cut_content: {
        label: 'CUT',
        classes: 'bg-amber-500/15 text-amber-400 border-amber-500/30',
    },
    pre_order: {
        label: 'PRE-ORDER',
        classes: 'bg-orange-500/15 text-orange-400 border-orange-500/30',
    },
    dlc_duplicate: {
        label: 'DLC DUP',
        classes: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
    },
    ban_risk: {
        label: '⚠ BAN',
        classes: 'bg-red-500/15 text-red-400 border-red-500/30',
    },
};

export function RiskBadge({flag, showInfoIcon = true, className = ''}: Props) {
    const style = STYLE[flag];
    if (!style) return null;
    return (
        <span className={`inline-flex items-center gap-1 ${className}`}>
            <span className={`text-[8px] font-black uppercase tracking-widest px-1.5 py-0.5 rounded border ${style.classes}`}>
                {style.label}
            </span>
            {showInfoIcon && <RiskInfoIcon riskKey={flag} />}
        </span>
    );
}
