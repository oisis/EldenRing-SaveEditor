import {useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import {RISK_INFO, RiskKey, RiskLevel, CONFIDENCE_STYLE} from '../data/riskInfo';

interface Props {
    riskKey: RiskKey;
    className?: string;
}

const ICON_TRI_COLOR: Record<RiskLevel, string> = {
    low: 'text-yellow-400 hover:text-yellow-300',
    medium: 'text-orange-400 hover:text-orange-300',
    high: 'text-red-400 hover:text-red-300',
};

const DOTS_COLOR: Record<RiskLevel, string> = {
    low: 'text-yellow-400',
    medium: 'text-orange-400',
    high: 'text-red-400',
};

const DOTS: Record<RiskLevel, string> = {
    low: '● ○ ○',
    medium: '● ● ○',
    high: '● ● ●',
};

const POPOVER_WIDTH = 320;
const POPOVER_OFFSET = 6;
const VIEWPORT_PADDING = 12;

export function RiskInfoIcon({riskKey, className = ''}: Props) {
    const [open, setOpen] = useState(false);
    const [pos, setPos] = useState<{top: number; left: number} | null>(null);
    const buttonRef = useRef<HTMLButtonElement>(null);
    const popoverRef = useRef<HTMLDivElement>(null);
    const entry = RISK_INFO[riskKey];

    useEffect(() => {
        if (!open) return;

        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape') setOpen(false);
        };
        const onClick = (e: MouseEvent) => {
            const target = e.target as Node;
            if (buttonRef.current?.contains(target)) return;
            if (popoverRef.current?.contains(target)) return;
            setOpen(false);
        };

        document.addEventListener('keydown', onKey);
        document.addEventListener('mousedown', onClick);
        return () => {
            document.removeEventListener('keydown', onKey);
            document.removeEventListener('mousedown', onClick);
        };
    }, [open]);

    if (!entry) return null;

    const handleToggle = (e: React.MouseEvent) => {
        e.stopPropagation();
        e.preventDefault();
        if (open) {
            setOpen(false);
            return;
        }
        const rect = buttonRef.current?.getBoundingClientRect();
        if (!rect) return;
        let left = rect.left;
        if (left + POPOVER_WIDTH > window.innerWidth - VIEWPORT_PADDING) {
            left = window.innerWidth - POPOVER_WIDTH - VIEWPORT_PADDING;
        }
        if (left < VIEWPORT_PADDING) left = VIEWPORT_PADDING;
        setPos({top: rect.bottom + POPOVER_OFFSET, left});
        setOpen(true);
    };

    return (
        <>
            <button
                ref={buttonRef}
                type="button"
                onClick={handleToggle}
                onMouseDown={(e) => e.stopPropagation()}
                aria-label={`Why is this risky? — ${entry.title}`}
                className={`inline-flex items-center justify-center text-[11px] leading-none transition-all hover:scale-125 cursor-pointer ${ICON_TRI_COLOR[entry.level]} ${className}`}
            >
                ⚠
            </button>
            {open && pos && createPortal(
                <div
                    ref={popoverRef}
                    role="dialog"
                    aria-label={entry.title}
                    style={{position: 'fixed', top: pos.top, left: pos.left, width: POPOVER_WIDTH, zIndex: 9999}}
                    className="rounded-lg border border-border bg-popover text-foreground shadow-2xl p-4 animate-in fade-in zoom-in-95 duration-150"
                >
                    <div className="flex items-start justify-between gap-3 mb-3">
                        <h4 className="text-[10px] font-black uppercase tracking-[0.15em]">{entry.title}</h4>
                        <span
                            className={`text-[10px] font-mono leading-none whitespace-nowrap ${DOTS_COLOR[entry.level]}`}
                            title={`Risk level: ${entry.level}`}
                        >
                            {DOTS[entry.level]}
                        </span>
                    </div>
                    <div className="flex items-center gap-1.5 mb-3 flex-wrap">
                        <span className="text-[8px] font-black uppercase tracking-widest px-1.5 py-0.5 rounded border border-border/50 text-muted-foreground">
                            Tier {entry.tier}
                        </span>
                        <span
                            className={`text-[8px] font-black uppercase tracking-widest px-1.5 py-0.5 rounded border ${CONFIDENCE_STYLE[entry.confidence].classes}`}
                            title="Confidence in detection rule — see spec/35"
                        >
                            {CONFIDENCE_STYLE[entry.confidence].label}
                        </span>
                    </div>
                    <div className="space-y-3 text-[10px] leading-relaxed text-muted-foreground">
                        <Section heading="Why is this flagged?" body={entry.whyBan} />
                        <Section heading="Community reports" body={entry.reports} />
                        <Section heading="How to mitigate" body={entry.mitigation} />
                        {entry.sources.length > 0 && (
                            <div>
                                <p className="text-[8px] font-black uppercase tracking-[0.15em] text-foreground/80 mb-1">Sources</p>
                                <ul className="space-y-0.5 text-[10px]">
                                    {entry.sources.map((s, i) => (
                                        <li key={i}>
                                            {s.url ? (
                                                <a
                                                    href={s.url}
                                                    target="_blank"
                                                    rel="noreferrer"
                                                    className="text-primary/80 hover:text-primary underline decoration-primary/40 underline-offset-2"
                                                >
                                                    {s.label}
                                                </a>
                                            ) : (
                                                <span>{s.label}</span>
                                            )}
                                        </li>
                                    ))}
                                </ul>
                            </div>
                        )}
                    </div>
                    <p className="mt-3 pt-3 border-t border-border/40 text-[8px] uppercase tracking-widest text-muted-foreground/70">
                        Press Esc to close
                    </p>
                </div>,
                document.body
            )}
        </>
    );
}

function Section({heading, body}: {heading: string; body: string}) {
    return (
        <div>
            <p className="text-[8px] font-black uppercase tracking-[0.15em] text-foreground/80 mb-1">{heading}</p>
            <p>{body}</p>
        </div>
    );
}
