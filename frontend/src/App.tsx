import {useState, useEffect} from 'react';
import {SelectAndOpenSave, GetActiveSlots, SetSlotActivity, GetCharacterNames} from '../wailsjs/go/main/App';
import {GeneralTab} from './components/GeneralTab';
import {StatsTab} from './components/StatsTab';
import {InventoryTab} from './components/InventoryTab';
import {WorldProgressTab} from './components/WorldProgressTab';
import {CharacterImporter} from './components/CharacterImporter';
import './App.css';

type Theme = 'light' | 'dark' | 'system';

function App() {
    const [selectedChar, setSelectedChar] = useState(0);
    const [activeTab, setActiveTab] = useState('general');
    const [platform, setPlatform] = useState<string | null>(null);
    const [isLoaded, setIsLoaded] = useState(false);
    const [activeSlots, setActiveSlots] = useState<boolean[]>(new Array(10).fill(false));
    const [charNames, setCharNames] = useState<string[]>(new Array(10).fill('Empty Slot'));
    const [theme, setTheme] = useState<Theme>(() => (localStorage.getItem('theme') as Theme) || 'system');

    useEffect(() => {
        const root = window.document.documentElement;
        const applyTheme = (t: 'light' | 'dark') => {
            root.classList.remove('light', 'dark');
            root.classList.add(t);
        };

        if (theme === 'system') {
            const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
            applyTheme(systemTheme);
        } else {
            applyTheme(theme);
        }
        localStorage.setItem('theme', theme);
    }, [theme]);

    const refreshSlots = async () => {
        try {
            const [slots, names] = await Promise.all([
                GetActiveSlots(),
                GetCharacterNames()
            ]);
            setActiveSlots(slots || new Array(10).fill(false));
            setCharNames(names || new Array(10).fill('Empty Slot'));
        } catch (e) {
            console.error("Failed to refresh slots:", e);
        }
    };

    const handleOpen = async () => {
        try {
            const res = await SelectAndOpenSave(); 
            if (res) {
                setPlatform(res);
                setIsLoaded(true);
                await refreshSlots();
            }
        } catch (e) {
            alert("Error opening file: " + e);
        }
    };

    const handleToggleSlot = async (idx: number) => {
        try {
            await SetSlotActivity(idx, !activeSlots[idx]);
            await refreshSlots();
        } catch (e) {
            console.error("Failed to toggle slot:", e);
        }
    };

    return (
        <div className="flex h-screen bg-background text-foreground font-sans overflow-hidden selection:bg-primary/20">
            {/* Sidebar */}
            <aside className="w-64 border-r border-border bg-muted/20 flex flex-col flex-shrink-0 z-20">
                <div className="p-5">
                    <div className="flex items-center space-x-3">
                        <div className="w-9 h-9 bg-foreground text-background rounded-lg flex items-center justify-center shadow-md">
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 11c0 3.517-1.009 6.799-2.753 9.571m-3.44-2.04l.054-.09A10.003 10.003 0 0012 3c1.268 0 2.39.234 3.468.657m-3.42 5.692L12 11m0 0l-1.036-3.242M12 11l3.964-3.036" />
                            </svg>
                        </div>
                        <div>
                            <h1 className="text-sm font-bold tracking-tight">ER Save Editor</h1>
                            <p className="text-[10px] text-muted-foreground font-semibold uppercase tracking-widest opacity-70">Professional Tool</p>
                        </div>
                    </div>
                </div>
                
                <div className="px-4 mb-4">
                    <button 
                        onClick={handleOpen} 
                        className="w-full bg-foreground text-background hover:opacity-90 transition-all font-semibold py-2 px-4 rounded-md text-[11px] shadow-sm uppercase tracking-wider"
                    >
                        {isLoaded ? 'Switch Save' : 'Open Save File'}
                    </button>
                </div>

                <div className="px-3 mb-2">
                    <div className="h-px bg-border/50 w-full" />
                </div>

                <nav className="flex-1 overflow-y-auto px-2 space-y-1.5 custom-scrollbar">
                    <p className="px-3 text-[10px] font-bold text-muted-foreground uppercase tracking-[0.15em] mb-3 mt-2">Character Slots</p>
                    {activeSlots.map((isActive, i) => (
                        <button 
                            key={i} 
                            disabled={!isLoaded}
                            onClick={() => setSelectedChar(i)}
                            className={`
                                w-full group px-3 py-2.5 rounded-md transition-all flex items-center justify-between border text-foreground
                                ${!isLoaded ? 'opacity-30 cursor-not-allowed' : 'cursor-pointer'}
                                ${isActive 
                                    ? 'bg-slot-active-bg border-slot-active-border hover:opacity-80' 
                                    : 'bg-slot-empty-bg border-slot-empty-border hover:opacity-80'}
                                ${selectedChar === i ? 'ring-2 ring-primary shadow-sm border-transparent' : ''}
                            `}
                        >
                            <div className="flex items-center space-x-3 overflow-hidden">
                                <div className={`w-2 h-2 rounded-full flex-shrink-0 ${isActive ? 'bg-green-500' : 'bg-red-500'}`} />
                                <span className="text-xs font-bold tracking-tight truncate">{charNames[i]}</span>
                            </div>
                            {isLoaded && (
                                <div 
                                    onClick={(e) => { e.stopPropagation(); handleToggleSlot(i); }}
                                    className="opacity-0 group-hover:opacity-100 p-1 hover:text-primary transition-all"
                                >
                                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
                                    </svg>
                                </div>
                            )}
                        </button>
                    ))}
                </nav>
                
                <div className="p-5 border-t border-border bg-muted/10 mt-auto">
                    <div className="flex items-center justify-between bg-background/50 border border-border p-1 rounded-lg mb-4">
                        {(['light', 'dark', 'system'] as Theme[]).map((t) => (
                            <button
                                key={t}
                                onClick={() => setTheme(t)}
                                className={`
                                    flex-1 py-1.5 px-2 rounded-md text-[10px] font-bold uppercase tracking-tighter transition-all
                                    ${theme === t ? 'bg-background text-foreground shadow-sm ring-1 ring-border' : 'text-muted-foreground hover:text-foreground'}
                                `}
                            >
                                {t}
                            </button>
                        ))}
                    </div>
                    <div className="flex justify-between items-center text-[10px] px-1 pb-2">
                        <span className="text-muted-foreground font-medium uppercase tracking-widest">Platform</span>
                        <span className="font-black text-foreground uppercase tracking-widest bg-muted/50 px-2 py-0.5 rounded">{platform || 'None'}</span>
                    </div>
                </div>
            </aside>

            {/* Main Content */}
            <main className="flex-1 flex flex-col min-w-0 bg-background">
                {/* Tabs Header */}
                <header className="border-b border-border px-8 flex items-center h-14 flex-shrink-0 glass sticky top-0 z-10">
                    <div className="flex space-x-8 h-full">
                        {['General', 'Stats', 'Inventory', 'World Progress', 'Importer'].map(tab => (
                            <button 
                                key={tab}
                                disabled={!isLoaded}
                                onClick={() => setActiveTab(tab.toLowerCase())}
                                className={`
                                    h-full px-1 text-[11px] font-bold uppercase tracking-widest transition-all relative flex items-center
                                    ${!isLoaded ? 'opacity-30 cursor-not-allowed' : 'hover:text-foreground'}
                                    ${activeTab === tab.toLowerCase() ? 'text-foreground' : 'text-muted-foreground'}
                                `}
                            >
                                {tab}
                                {activeTab === tab.toLowerCase() && (
                                    <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary shadow-[0_0_8px_rgba(59,130,246,0.5)]" />
                                )}
                            </button>
                        ))}
                    </div>
                </header>

                {/* Content Area */}
                <div className="flex-1 overflow-y-auto p-8 custom-scrollbar">
                    <div className="max-w-5xl mx-auto w-full">
                        {!isLoaded ? (
                            <div className="h-[70vh] flex flex-col items-center justify-center text-center">
                                <div className="w-16 h-16 bg-muted/50 rounded-2xl flex items-center justify-center mb-8 border border-border shadow-inner">
                                    <svg className="w-8 h-8 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                    </svg>
                                </div>
                                <h3 className="text-xl font-bold tracking-tight">Ready to begin</h3>
                                <p className="text-muted-foreground text-sm mt-2 max-w-xs mx-auto font-medium">
                                    Connect your Elden Ring save file to unlock character editing tools.
                                </p>
                                <button onClick={handleOpen} className="mt-8 bg-foreground text-background hover:opacity-90 font-bold text-[11px] px-8 py-3 rounded-md shadow-lg transition-all uppercase tracking-widest">
                                    Open Save File
                                </button>
                            </div>
                        ) : (
                            <div className="animate-in fade-in slide-in-from-bottom-3 duration-700">
                                <div className="flex items-end justify-between mb-12 border-b border-border pb-8">
                                    <div>
                                        <h2 className="text-3xl font-black tracking-tighter capitalize">{activeTab}</h2>
                                    </div>
                                    <div className="flex items-center space-x-3 bg-muted/20 px-4 py-2 rounded-lg border border-border shadow-sm">
                                        <div className="flex flex-col items-end">
                                            <span className="text-[9px] font-bold text-muted-foreground uppercase tracking-widest">Platform</span>
                                            <span className="text-xs font-black uppercase">{platform}</span>
                                        </div>
                                        <div className="w-px h-8 bg-border" />
                                        <div className="w-2 h-2 bg-primary rounded-full animate-pulse shadow-[0_0_8px_rgba(59,130,246,0.5)]" />
                                    </div>
                                </div>
                                
                                <div className="min-h-[400px]">
                                    {activeTab === 'general' && <GeneralTab charIndex={selectedChar} />}
                                    {activeTab === 'stats' && <StatsTab charIndex={selectedChar} />}
                                    {activeTab === 'inventory' && <InventoryTab charIndex={selectedChar} />}
                                    {activeTab === 'world progress' && <WorldProgressTab />}
                                    {activeTab === 'importer' && <CharacterImporter destSlot={selectedChar} onComplete={refreshSlots} />}
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            </main>
        </div>
    );
}

export default App;