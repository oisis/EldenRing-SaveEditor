import { ReactNode } from 'react';

// StatSlider — the single shared attribute-row control used by both
// Character → Attributes and the Templates "Apply with overrides" panel.
// Renders a label + range slider (with a red-zone gradient marking the
// class minimum) + a compact number input, exactly as the Character tab
// has always looked. Extracted so the two call sites cannot drift into
// two divergent implementations.
//
// `leading` lets a caller prepend a control (e.g. the overrides "apply
// this stat" checkbox) inside the same flex row without duplicating the
// row markup. Character → Attributes passes no leading node.

interface StatSliderProps {
    label: string;
    value: number;
    // min is the dynamic lower bound (starting-class minimum). It drives
    // the number input's min attribute and the red-zone gradient width.
    min: number;
    max?: number;
    onChange: (value: number) => void;
    disabled?: boolean;
    leading?: ReactNode;
    title?: string;
    rangeTestId?: string;
    numberTestId?: string;
}

export function StatSlider({
    label,
    value,
    min,
    max = 99,
    onChange,
    disabled,
    leading,
    title,
    rangeTestId,
    numberTestId,
}: StatSliderProps) {
    const redZonePct = ((min - 1) / 98) * 100;
    return (
        <div className="flex items-center gap-3 py-1.5 border-b border-border/30">
            {leading}
            <span
                className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider w-20 flex-shrink-0"
                title={title ?? `Base: ${min}`}
            >
                {label}
            </span>
            <input
                type="range"
                min={1}
                max={max}
                value={value}
                disabled={disabled}
                data-testid={rangeTestId}
                onChange={e => onChange(parseInt(e.target.value))}
                className="flex-1 h-1.5 rounded-lg appearance-none cursor-pointer disabled:opacity-40"
                style={{
                    background: `linear-gradient(to right, rgb(239 68 68 / 0.4) 0%, rgb(239 68 68 / 0.4) ${redZonePct}%, hsl(var(--border)) ${redZonePct}%, hsl(var(--border)) 100%)`,
                }}
            />
            <input
                type="number"
                min={min}
                max={max}
                value={value}
                disabled={disabled}
                data-testid={numberTestId}
                onChange={e => onChange(parseInt(e.target.value) || min)}
                className="w-12 bg-muted/30 border border-border rounded text-center text-xs py-1 focus:ring-1 focus:ring-primary/30 outline-none disabled:opacity-40"
            />
        </div>
    );
}
