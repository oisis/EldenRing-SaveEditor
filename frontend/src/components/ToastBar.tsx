import {useEffect, useState, useCallback} from 'react';

export type LogLevel = 'info' | 'warn' | 'error';
export type LogEntry = { time: string; level: LogLevel; message: string };

let globalLogFn: ((level: LogLevel, message: string) => void) | null = null;

export function sfLog(level: LogLevel, message: string) {
    globalLogFn?.(level, message);
}

interface ToastBarProps {
    sidebarWidth?: number;
}

export function ToastBar({ sidebarWidth = 256 }: ToastBarProps) {
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [consoleOpen, setConsoleOpen] = useState(false);
    const [lastMessage, setLastMessage] = useState<LogEntry | null>(null);

    const addLog = useCallback((level: LogLevel, message: string) => {
        const entry: LogEntry = {
            time: new Date().toLocaleTimeString('en-GB', { hour12: false }),
            level,
            message,
        };
        setLogs(prev => [...prev, entry]);
        setLastMessage(entry);
    }, []);

    useEffect(() => {
        globalLogFn = addLog;
        addLog('info', 'SaveForge session started');
        return () => { globalLogFn = null; };
    }, [addLog]);

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

    const levelColor = (l: LogLevel) => {
        switch (l) {
            case 'info': return 'text-info';
            case 'warn': return 'text-warning';
            case 'error': return 'text-destructive';
        }
    };

    return (
        <>
            {/* Toast Bar — single line at bottom */}
            <div
                className="fixed bottom-0 left-1/2 -translate-x-1/2 z-40 cursor-pointer"
                style={{ width: '30%', minWidth: '300px' }}
                onClick={() => setConsoleOpen(v => !v)}
            >
                <div className="backdrop-blur-md border border-b-0 rounded-t-lg px-3 py-1.5 flex items-center gap-2"
                    style={{ background: 'var(--sf-toast-bg)', borderColor: 'var(--sf-console-border)' }}>
                    <div className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse flex-shrink-0" />
                    <span className="text-[11px] font-mono truncate flex-1" style={{ color: 'var(--sf-console-text-dim)' }}>
                        {lastMessage ? lastMessage.message : 'Ready'}
                    </span>
                    <span className="text-[9px] flex-shrink-0" style={{ color: 'var(--sf-console-text-dim)', opacity: 0.5 }}>
                        {consoleOpen ? '▼' : '▲'} `
                    </span>
                </div>
            </div>

            {/* Quake Console */}
            {consoleOpen && (
                <div
                    className="fixed bottom-[30px] z-30 backdrop-blur-md rounded-t-lg flex flex-col"
                    style={{
                        left: `${sidebarWidth}px`,
                        right: `${sidebarWidth}px`,
                        height: '45vh',
                        background: 'var(--sf-console-bg)',
                        border: '1px solid var(--sf-console-border)',
                    }}
                >
                    <div className="flex items-center justify-between px-3 py-1.5" style={{ borderBottom: '1px solid var(--sf-console-border)' }}>
                        <span className="text-[9px] font-black uppercase tracking-[0.2em]" style={{ color: 'var(--sf-console-text-dim)' }}>Console</span>
                        <div className="flex items-center gap-2">
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
                    <div className="flex-1 overflow-y-auto custom-scrollbar px-3 py-2 font-mono text-[11px] space-y-0.5">
                        {logs.map((entry, i) => (
                            <div key={i} className="flex gap-2">
                                <span className="flex-shrink-0" style={{ color: 'var(--sf-console-text-dim)', opacity: 0.5 }}>{entry.time}</span>
                                <span className={`uppercase font-bold flex-shrink-0 w-10 ${levelColor(entry.level)}`}>
                                    {entry.level}
                                </span>
                                <span style={{ color: 'var(--sf-console-text)' }}>{entry.message}</span>
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
