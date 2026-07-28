interface ItemCapacityBadgeProps {
    owned: number;
    max: number;
    label: 'Inventory' | 'Storage';
    prefix?: 'I' | 'S';
    compact?: boolean;
    mode?: 'quantity' | 'instance' | 'instance-count';
}

export function ItemCapacityBadge({ owned, max, label, prefix, compact = false, mode = 'quantity' }: ItemCapacityBadgeProps) {
    const common = compact
        ? 'inline-flex min-w-[34px] items-center justify-center rounded border px-1.5 py-0.5 text-[8px] font-black tabular-nums'
        : 'inline-flex min-w-[48px] items-center justify-center rounded border px-2 py-1 text-[10px] font-black tabular-nums whitespace-nowrap';
    const prefixText = prefix ? `${prefix}:` : '';

    // Instance-backed items (weapons, armor, talismans and Ashes of War) use
    // one physical record per copy. A per-record quantity cap of 1 is not a
    // total ownership cap, so this badge reports only whether this row/catalog
    // item is present in the container.
    if (mode === 'instance') {
        if (owned > 0) {
            return (
                <span
                    aria-label={`${label} present`}
                    title={`${label}: ${owned} ${owned === 1 ? 'copy' : 'copies'}`}
                    className={`${common} border-green-500/30 bg-green-500/10 text-green-500`}
                >
                    {prefixText}<span aria-hidden="true">✓</span>
                </span>
            );
        }
        return (
            <span
                aria-label={`${label} not present`}
                title={`${label}: not present`}
                className={`${common} border-border/30 bg-muted/20 text-muted-foreground/50`}
            >
                {prefixText}<span aria-hidden="true">—</span>
            </span>
        );
    }

    // Catalog counter for instance-backed items: shows the real number of
    // physical copies over the per-record quantity cap of 1 (e.g. "3 / 1").
    // The denominator is always 1 — it is the single-record limit, not the
    // container's technical storage cap.
    if (mode === 'instance-count') {
        const tone = owned > 0
            ? 'border-green-500/30 bg-green-500/10 text-green-500'
            : 'border-border/30 bg-muted/20 text-muted-foreground/50';
        return (
            <span aria-label={`${label} ${owned} of 1`} title={`${label}: ${owned} / 1`} className={`${common} ${tone}`}>
                {compact ? `${prefixText}${owned}/1` : `${owned} / 1`}
            </span>
        );
    }

    if (max <= 0) {
        return (
            <span
                aria-label={`${label} unavailable`}
                title={`${label}: this item cannot be placed here`}
                className={`${common} border-red-500/40 bg-red-500/10 text-red-500`}
            >
                {prefixText}<span aria-hidden="true">×</span>
            </span>
        );
    }

    if (max === 1 && owned >= 1) {
        return (
            <span
                aria-label={`${label} 1 of 1`}
                title={`${label}: 1 / 1`}
                className={`${common} border-green-500/30 bg-green-500/10 text-green-500`}
            >
                {prefixText}<span aria-hidden="true">✓</span>
            </span>
        );
    }

    const tone = owned > 0
        ? 'border-green-500/30 bg-green-500/10 text-green-500'
        : 'border-border/30 bg-muted/20 text-muted-foreground/50';
    return (
        <span aria-label={`${label} ${owned} of ${max}`} className={`${common} ${tone}`}>
            {compact ? `${prefixText}${owned}` : `${owned} / ${max}`}
        </span>
    );
}
