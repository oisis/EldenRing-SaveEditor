import {useState, ReactNode, useEffect} from 'react';

interface AccordionSectionProps {
    id?: string;
    title: string;
    defaultOpen?: boolean;
    badge?: string | number;
    progress?: { current: number; total: number };
    summary?: string;
    actions?: ReactNode;
    headerRight?: ReactNode;
    children: ReactNode;
    className?: string;
}

export function AccordionSection({
    id,
    title,
    defaultOpen = false,
    badge,
    progress,
    summary,
    actions,
    headerRight,
    children,
    className = '',
}: AccordionSectionProps) {
    const storageKey = id ? `accordion:${id}` : null;

    const [open, setOpen] = useState(() => {
        if (storageKey) {
            const saved = localStorage.getItem(storageKey);
            if (saved !== null) return saved === '1';
        }
        return defaultOpen;
    });

    useEffect(() => {
        if (storageKey) localStorage.setItem(storageKey, open ? '1' : '0');
    }, [open, storageKey]);

    const pct = progress ? Math.round((progress.current / Math.max(progress.total, 1)) * 100) : null;

    return (
        <div className={`border border-border rounded-lg overflow-hidden ${className}`}>
            {/* Header */}
            <button
                onClick={() => setOpen(v => !v)}
                className="w-full flex items-center gap-2 px-3 py-2 bg-muted/10 hover:bg-muted/20 transition-all text-left"
            >
                <svg
                    className={`w-3 h-3 text-muted-foreground transition-transform duration-200 flex-shrink-0 ${open ? 'rotate-90' : ''}`}
                    fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M9 5l7 7-7 7" />
                </svg>

                <span className="text-[10px] font-black uppercase tracking-[0.15em] text-foreground/80 flex-shrink-0">
                    {title}
                </span>

                {badge !== undefined && (
                    <span className="text-[8px] font-bold bg-primary/10 text-primary px-1.5 py-0.5 rounded-full flex-shrink-0">
                        {badge}
                    </span>
                )}

                {!open && pct !== null && (
                    <div className="flex items-center gap-2 flex-1 min-w-0 ml-2">
                        <div className="flex-1 h-1.5 bg-muted/30 rounded-full overflow-hidden">
                            <div
                                className="h-full bg-primary rounded-full transition-all duration-300"
                                style={{ width: `${pct}%` }}
                            />
                        </div>
                        <span className="text-[9px] font-mono text-muted-foreground flex-shrink-0">
                            {progress!.current}/{progress!.total}
                        </span>
                    </div>
                )}

                {!open && summary && !progress && (
                    <span className="text-[9px] text-muted-foreground font-medium ml-2 truncate">
                        {summary}
                    </span>
                )}

                {open && <div className="flex-1" />}

                {open && actions && (
                    <div className="flex items-center gap-1 flex-shrink-0" onClick={e => e.stopPropagation()}>
                        {actions}
                    </div>
                )}

                {headerRight && (
                    <div className="flex items-center flex-shrink-0 ml-auto" onClick={e => e.stopPropagation()}>
                        {headerRight}
                    </div>
                )}
            </button>

            {open && (
                <div className="px-3 py-2 border-t border-border/50">
                    {children}
                </div>
            )}
        </div>
    );
}
