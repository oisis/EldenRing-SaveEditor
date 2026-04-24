import {useEffect, useState, useCallback, useRef} from 'react';

export type LogLevel = 'info' | 'warn' | 'error';
export type LogEntry = { time: string; level: LogLevel; message: string; loading?: boolean };

let globalLogFn: ((level: LogLevel, message: string) => void) | null = null;
let globalLoadingFn: ((id: string, message: string) => void) | null = null;
let globalDoneFn: ((id: string) => void) | null = null;

export function sfLog(level: LogLevel, message: string) {
    globalLogFn?.(level, message);
}

export function sfLoading(id: string, message: string) {
    globalLoadingFn?.(id, message);
}

export function sfDone(id: string) {
    globalDoneFn?.(id);
}

function LoadingDots() {
    return (
        <span className="inline-flex ml-0.5">
            <span className="animate-dot-1">.</span>
            <span className="animate-dot-2">.</span>
            <span className="animate-dot-3">.</span>
        </span>
    );
}

interface ToastBarProps {
    sidebarWidth?: number;
}

const MIN_WIDTH = 400;
const MIN_HEIGHT = 150;

export function ToastBar({ sidebarWidth = 256 }: ToastBarProps) {
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [consoleOpen, setConsoleOpen] = useState(false);
    const [lastMessage, setLastMessage] = useState<LogEntry | null>(null);

    // Console dimensions (persisted in localStorage)
    const [consoleWidth, setConsoleWidth] = useState<number>(() => {
        const saved = localStorage.getItem('console:width');
        return saved ? parseInt(saved) : 0; // 0 = auto (full width minus margins)
    });
    const [consoleHeight, setConsoleHeight] = useState<number>(() => {
        const saved = localStorage.getItem('console:height');
        return saved ? parseInt(saved) : Math.round(window.innerHeight * 0.45);
    });

    const consoleRef = useRef<HTMLDivElement>(null);
    const resizingRef = useRef<{ edge: 'top' | 'left' | 'right' | 'topleft' | 'topright'; startX: number; startY: number; startW: number; startH: number } | null>(null);
    const initRef = useRef(false);

    const addLog = useCallback((level: LogLevel, message: string) => {
        const entry: LogEntry = {
            time: new Date().toLocaleTimeString('en-GB', { hour12: false }),
            level,
            message,
        };
        setLogs(prev => [...prev, entry]);
        setLastMessage(entry);
    }, []);

    const startLoading = useCallback((id: string, message: string) => {
        const entry: LogEntry = {
            time: new Date().toLocaleTimeString('en-GB', { hour12: false }),
            level: 'info',
            message,
            loading: true,
        };
        setLogs(prev => {
            const idx = prev.findIndex(e => (e as any)._loadId === id);
            if (idx >= 0) {
                const next = [...prev];
                next[idx] = Object.assign(entry, { _loadId: id });
                return next;
            }
            return [...prev, Object.assign(entry, { _loadId: id })];
        });
        setLastMessage(entry);
    }, []);

    const finishLoading = useCallback((id: string) => {
        setLogs(prev => prev.map(e =>
            (e as any)._loadId === id ? { ...e, loading: false } : e
        ));
        setLastMessage(prev => prev && (prev as any)._loadId === id ? { ...prev, loading: false } : prev);
    }, []);

    useEffect(() => {
        globalLogFn = addLog;
        globalLoadingFn = startLoading;
        globalDoneFn = finishLoading;
        if (!initRef.current) {
            initRef.current = true;
            addLog('info', 'SaveForge session started');
        }
        return () => { globalLogFn = null; globalLoadingFn = null; globalDoneFn = null; };
    }, [addLog, startLoading, finishLoading]);

    // Keyboard toggle
    useEffect(() => {
        const handler = (e: KeyboardEvent) => {
            if (e.key === '`' && !e.ctrlKey && !e.metaKey) {
                const tag = (e.target as HTMLElement).tagName;
                if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;
                e.preventDefault();
                setConsoleOpen(v => !v);
            }
        };
        window.addEventListener('keydown', handler);
        return () => window.removeEventListener('keydown', handler);
    }, []);

    // Click outside to close
    useEffect(() => {
        if (!consoleOpen) return;
        const handler = (e: MouseEvent) => {
            if (resizingRef.current) return; // don't close while resizing
            if (consoleRef.current && !consoleRef.current.contains(e.target as Node)) {
                setConsoleOpen(false);
            }
        };
        // Delay to avoid immediate close from the click that opened it
        const timer = setTimeout(() => {
            window.addEventListener('mousedown', handler);
        }, 100);
        return () => {
            clearTimeout(timer);
            window.removeEventListener('mousedown', handler);
        };
    }, [consoleOpen]);

    // Persist dimensions
    useEffect(() => {
        if (consoleWidth > 0) localStorage.setItem('console:width', String(consoleWidth));
        localStorage.setItem('console:height', String(consoleHeight));
    }, [consoleWidth, consoleHeight]);

    // Resize handlers
    const startResize = useCallback((e: React.MouseEvent, edge: 'top' | 'left' | 'right' | 'topleft' | 'topright') => {
        e.preventDefault();
        e.stopPropagation();
        const rect = consoleRef.current?.getBoundingClientRect();
        if (!rect) return;
        resizingRef.current = {
            edge,
            startX: e.clientX,
            startY: e.clientY,
            startW: rect.width,
            startH: rect.height,
        };

        const onMove = (ev: MouseEvent) => {
            if (!resizingRef.current) return;
            const { edge, startX, startY, startW, startH } = resizingRef.current;
            const dx = ev.clientX - startX;
            const dy = ev.clientY - startY;
            const maxW = window.innerWidth - sidebarWidth * 2;
            const maxH = window.innerHeight - 60;

            if (edge === 'top' || edge === 'topleft' || edge === 'topright') {
                setConsoleHeight(Math.max(MIN_HEIGHT, Math.min(maxH, startH - dy)));
            }
            if (edge === 'left' || edge === 'topleft') {
                setConsoleWidth(Math.max(MIN_WIDTH, Math.min(maxW, startW - dx * 2))); // *2 because centered
            }
            if (edge === 'right' || edge === 'topright') {
                setConsoleWidth(Math.max(MIN_WIDTH, Math.min(maxW, startW + dx * 2)));
            }
        };

        const onUp = () => {
            resizingRef.current = null;
            window.removeEventListener('mousemove', onMove);
            window.removeEventListener('mouseup', onUp);
        };

        window.addEventListener('mousemove', onMove);
        window.addEventListener('mouseup', onUp);
    }, [sidebarWidth]);

    const levelColor = (l: LogLevel) => {
        switch (l) {
            case 'info': return 'text-info';
            case 'warn': return 'text-warning';
            case 'error': return 'text-destructive';
        }
    };

    // Compute console positioning
    const availableWidth = typeof window !== 'undefined' ? window.innerWidth - sidebarWidth * 2 : 800;
    const effectiveWidth = consoleWidth > 0 ? Math.min(consoleWidth, availableWidth) : availableWidth;
    const horizontalOffset = consoleWidth > 0
        ? Math.max(sidebarWidth, (window.innerWidth - effectiveWidth) / 2)
        : sidebarWidth;

    return (
        <>
            {/* Toast Bar — only visible when console is closed */}
            {!consoleOpen && (
                <div
                    className="fixed bottom-0 left-1/2 -translate-x-1/2 z-40 cursor-pointer"
                    style={{ width: '30%', minWidth: '300px' }}
                    onClick={() => setConsoleOpen(true)}
                >
                    <div className="backdrop-blur-md border border-b-0 rounded-t-lg px-3 py-1.5 flex items-center gap-2"
                        style={{ background: 'var(--sf-toast-bg)', borderColor: 'var(--sf-console-border)' }}>
                        <div className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${lastMessage?.loading ? 'bg-warning animate-spin-slow' : 'bg-primary animate-pulse'}`} />
                        <span className="text-[11px] font-mono truncate flex-1" style={{ color: 'var(--sf-console-text-dim)' }}>
                            {lastMessage ? lastMessage.message : 'Ready'}
                            {lastMessage?.loading && <LoadingDots />}
                        </span>
                        <span className="text-[9px] flex-shrink-0" style={{ color: 'var(--sf-console-text-dim)', opacity: 0.5 }}>
                            ▲ `
                        </span>
                    </div>
                </div>
            )}

            {/* Quake Console — expanded */}
            {consoleOpen && (
                <div
                    ref={consoleRef}
                    className="fixed bottom-0 z-30 backdrop-blur-md rounded-t-lg flex flex-col"
                    style={{
                        left: `${horizontalOffset}px`,
                        right: `${horizontalOffset}px`,
                        width: consoleWidth > 0 ? `${effectiveWidth}px` : undefined,
                        height: `${consoleHeight}px`,
                        background: 'var(--sf-console-bg)',
                        border: '1px solid var(--sf-console-border)',
                        borderBottom: 'none',
                    }}
                >
                    {/* Resize handles */}
                    {/* Top edge */}
                    <div className="absolute -top-1 left-3 right-3 h-2 cursor-ns-resize z-10"
                        onMouseDown={e => startResize(e, 'top')} />
                    {/* Left edge */}
                    <div className="absolute top-3 -left-1 w-2 bottom-0 cursor-ew-resize z-10"
                        onMouseDown={e => startResize(e, 'left')} />
                    {/* Right edge */}
                    <div className="absolute top-3 -right-1 w-2 bottom-0 cursor-ew-resize z-10"
                        onMouseDown={e => startResize(e, 'right')} />
                    {/* Top-left corner */}
                    <div className="absolute -top-1 -left-1 w-4 h-4 cursor-nwse-resize z-20"
                        onMouseDown={e => startResize(e, 'topleft')} />
                    {/* Top-right corner */}
                    <div className="absolute -top-1 -right-1 w-4 h-4 cursor-nesw-resize z-20"
                        onMouseDown={e => startResize(e, 'topright')} />

                    {/* Header */}
                    <div className="flex items-center justify-between px-3 py-1.5 shrink-0" style={{ borderBottom: '1px solid var(--sf-console-border)' }}>
                        <span className="text-[9px] font-black uppercase tracking-[0.2em]" style={{ color: 'var(--sf-console-text-dim)' }}>Console</span>
                        <div className="flex items-center gap-2">
                            {consoleWidth > 0 && (
                                <button
                                    onClick={() => setConsoleWidth(0)}
                                    className="text-[8px] font-bold uppercase tracking-widest transition-colors"
                                    style={{ color: 'var(--sf-console-text-dim)' }}
                                    title="Reset size"
                                >
                                    Reset
                                </button>
                            )}
                            <button
                                onClick={() => setLogs([])}
                                className="text-[8px] font-bold uppercase tracking-widest transition-colors"
                                style={{ color: 'var(--sf-console-text-dim)' }}
                            >
                                Clear
                            </button>
                            <button
                                onClick={() => setConsoleOpen(false)}
                                className="transition-colors"
                                style={{ color: 'var(--sf-console-text-dim)' }}
                            >
                                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                                </svg>
                            </button>
                        </div>
                    </div>

                    {/* Log body */}
                    <div className="flex-1 overflow-y-auto custom-scrollbar px-3 py-2 font-mono text-[11px] space-y-0.5">
                        {logs.map((entry, i) => (
                            <div key={i} className="flex gap-2">
                                <span className="flex-shrink-0" style={{ color: 'var(--sf-console-text-dim)', opacity: 0.7 }}>{entry.time}</span>
                                <span className={`uppercase font-bold flex-shrink-0 w-10 ${levelColor(entry.level)}`}>
                                    {entry.level}
                                </span>
                                <span style={{ color: 'var(--sf-console-text)' }}>
                                    {entry.message}
                                    {entry.loading && <LoadingDots />}
                                </span>
                            </div>
                        ))}
                        {logs.length === 0 && (
                            <div className="text-center py-8" style={{ color: 'var(--sf-console-text-dim)', opacity: 0.3 }}>No log entries</div>
                        )}
                    </div>
                </div>
            )}
        </>
    );
}
