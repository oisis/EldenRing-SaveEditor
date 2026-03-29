import {useState, useEffect} from 'react';
import {SelectAndOpenSave, GetActiveSlots, SetSlotActivity} from '../wailsjs/go/main/App';
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
            const res = await GetActiveSlots();
            setActiveSlots(res || new Array(10).fill(false));
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
        <div className="flex h-screen bg-background text-foreground font-sans overflow-hidden selection:bg-blue-500/20">
            {/* Sidebar */}
            <aside className="w-64 border-r border-border bg-muted/30 flex flex-col flex-shrink-0 z-20">
                <div className="p-6">
                    <div className="flex items-center space-x-2.5">
                        <div className="w-8 h-8 bg-foreground rounded flex items-center justify-center shadow-sm">
                            <svg className="w-5 h-5 text-background" fill="currentColor" viewBox="0 0 24 24">
                                <path d="M12 2L4.5 20.29l.71.71L12 18l6.79 3 .71-.71L12 2z" />
                            </svg>
                        </div>
                        <div>
                            <h1 className="text-sm font-semibold tracking-tight leading-none">ER Editor</h1>
                            <p className="text-[11px] text-muted-foreground mt-1 font-medium">v1.0.0</p>
                        </div>
                    </div>
                </div>
                
                <div className="px-4 mb-6">
                    <button 
                        onClick={handleOpen} 
                        className="w-full bg-foreground text-background hover:opacity-90 transition-opacity font-medium py-2 px-4 rounded text-xs shadow-sm"
                    >
                        {isLoaded ? 'Switch Save' : 'Open Save File'}
                    </button>
                </div>

                <nav className="flex-1 overflow-y-auto px-3 space-y-0.5 custom-scrollbar">
                    <p className="px-3 text-[10px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">Slots</p>
                    {activeSlots.map((isActive, i) => (
                        <div 
                            key={i} 
                            onClick={() => isLoaded && setSelectedChar(i)}
                            className={`
                                group px-3 py-2 rounded cursor-pointer transition-all flex items-center justify-between
                                ${!isLoaded ? 'opacity-40 cursor-not-allowed' : ''}
                                ${selectedChar === i ? 'bg-accent text-accent-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'}
                            `}
                        >
                            <div className="flex items-center space-x-3">
                                <div className={`w-1.5 h-1.5 rounded-full ${isActive ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.4)]' : 'bg-zinc-300 dark:bg-zinc-700'}`} />
                                <span className="text-xs font-medium">Slot {i + 1}</span>
                            </div>
                            {isLoaded && (
                                <button 
                                    onClick={(e) => { e.stopPropagation(); handleToggleSlot(i); }}
                                    className="opacity-0 group-hover:opacity-100 text-[10px] font-semibold hover:text-blue-500 transition-all"
                                >
                                    {isActive ? 'OFF' : 'ON'}
                                </button>
                            )}
                        </div>
                    ))}
                </nav>
                
                <div className="p-4 border-t border-border space-y-4">
                    <div className="flex items-center justify-between bg-accent/50 p-1 rounded">
                        {(['light', 'dark', 'system'] as Theme[]).map((t) => (
                            <button
                                key={t}
                                onClick={() => setTheme(t)}
                                className={`
                                    flex-1 py-1 px-2 rounded text-[10px] font-medium capitalize transition-all
                                    ${theme === t ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}
                                `}
                            >
                                {t}
                            </button>
                        ))}
                    </div>
                    <div className="flex justify-between items-center text-[10px] px-1 text-muted-foreground">
                        <span>Platform</span>
                        <span className="font-semibold text-foreground uppercase">{platform || 'None'}</span>
                    </div>
                </div>
            </aside>

            {/* Main Content */}
            <main className="flex-1 flex flex-col min-w-0">
                {/* Tabs Header */}
                <header className="border-b border-border px-8 flex items-center h-12 flex-shrink-0">
                    <div className="flex space-x-6 h-full">
                        {['General', 'Stats', 'Inventory', 'World Progress', 'Importer'].map(tab => (
                            <button 
                                key={tab}
                                disabled={!isLoaded}
                                onClick={() => setActiveTab(tab.toLowerCase())}
                                className={`
                                    h-full px-1 text-xs font-medium transition-all relative flex items-center
                                    ${!isLoaded ? 'opacity-30 cursor-not-allowed' : 'hover:text-foreground'}
                                    ${activeTab === tab.toLowerCase() ? 'text-foreground' : 'text-muted-foreground'}
                                `}
                            >
                                {tab}
                                {activeTab === tab.toLowerCase() && (
                                    <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-foreground" />
                                )}
                            </button>
                        ))}
                    </div>
                </header>

                {/* Content Area */}
                <div className="flex-1 overflow-y-auto p-10 custom-scrollbar">
                    <div className="max-w-4xl mx-auto w-full">
                        {!isLoaded ? (
                            <div className="h-[60vh] flex flex-col items-center justify-center text-center">
                                <div className="w-12 h-12 bg-muted rounded flex items-center justify-center mb-6 border border-border">
                                    <svg className="w-6 h-6 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                    </svg>
                                </div>
                                <h3 className="text-lg font-semibold">No active session</h3>
                                <p className="text-muted-foreground text-sm mt-1.5 max-w-xs mx-auto">
                                    Select an Elden Ring save file to begin editing.
                                </p>
                                <button onClick={handleOpen} className="mt-6 bg-foreground text-background hover:opacity-90 font-medium text-xs px-5 py-2 rounded shadow-sm transition-opacity">
                                    Open Save File
                                </button>
                            </div>
                        ) : (
                            <div className="animate-in fade-in slide-in-from-bottom-2 duration-500">
                                <div className="flex items-end justify-between mb-10">
                                    <div>
                                        <h2 className="text-2xl font-semibold tracking-tight capitalize">{activeTab}</h2>
                                        <p className="text-sm text-muted-foreground mt-1">Slot {selectedChar + 1}</p>
                                    </div>
                                    <div className="flex items-center space-x-2 bg-muted/50 px-2.5 py-1 rounded border border-border">
                                        <span className="w-1.5 h-1.5 bg-green-500 rounded-full" />
                                        <span className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Connected</span>
                                    </div>
                                </div>
                                
                                <div className="bg-background">
                                    {activeTab === 'general' && <GeneralTab charIndex={selectedChar} />}
                                    {activeTab === 'stats' && <StatsTab charIndex={selectedChar} />}
                                    {activeTab === 'inventory' && <InventoryTab />}
                                    {activeTab === 'world progress' && <WorldProgressTab />}
                                    {activeTab === 'importer' && <CharacterImporter destSlot={selectedChar} onComplete={refreshSlots} />}
                                </div>
                            </div>
                        )}
                    </div>
                </div>
            </main>
        </div>
    )
}

export default App
