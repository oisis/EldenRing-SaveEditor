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
                <div className="bg-background/80 backdrop-blur-sm border border-border/50 border-b-0 rounded-t-lg px-3 py-1.5 flex items-center gap-2">
                    <div className="w-1.5 h-1.5 rounded-full bg-primary animate-pulse flex-shrink-0" />
                    <span className="text-[11px] font-mono text-muted-foreground truncate flex-1">
                        {lastMessage ? lastMessage.message : 'Ready'}
                    </span>
                    <span className="text-[9px] text-muted-foreground/50 flex-shrink-0">
                        {consoleOpen ? '▼' : '▲'} `
                    </span>
                </div>
            </div>

            {/* Quake Console */}
            {consoleOpen && (
                <div
                    className="fixed bottom-[30px] z-30 bg-background/95 backdrop-blur-md border border-border rounded-t-lg flex flex-col"
                    style={{
                        left: `${sidebarWidth}px`,
                        right: `${sidebarWidth}px`,
                        height: '45vh',
                    }}
                >
                    <div className="flex items-center justify-between px-3 py-1.5 border-b border-border/50">
                        <span className="text-[9px] font-black uppercase tracking-[0.2em] text-muted-foreground">Console</span>
                        <div className="flex items-center gap-2">
                            <button
                                onClick={() => setLogs([])}
                                className="text-[8px] font-bold uppercase tracking-widest text-muted-foreground hover:text-foreground transition-colors"
                            >
                                Clear
                            </button>
                            <button
                                onClick={() => setConsoleOpen(false)}
                                className="text-muted-foreground hover:text-foreground transition-colors"
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
                                <span className="text-muted-foreground/50 flex-shrink-0">{entry.time}</span>
                                <span className={`uppercase font-bold flex-shrink-0 w-10 ${levelColor(entry.level)}`}>
                                    {entry.level}
                                </span>
                                <span className="text-foreground/80">{entry.message}</span>
                            </div>
                        ))}
                        {logs.length === 0 && (
                            <div className="text-muted-foreground/30 text-center py-8">No log entries</div>
                        )}
                    </div>
                </div>
            )}
        </>
    );
}
