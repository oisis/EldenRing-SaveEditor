import {useState, useEffect} from 'react';
import {SelectAndOpenSave, GetActiveSlots, SetSlotActivity, GetCharacterNames, WriteSave} from '../wailsjs/go/main/App';
import {GeneralTab} from './components/GeneralTab';
import {InventoryTab} from './components/InventoryTab';
import {WorldProgressTab} from './components/WorldProgressTab';
import {CharacterImporter} from './components/CharacterImporter';
import {SettingsTab} from './components/SettingsTab';

type Theme = 'light' | 'dark' | 'system';

function App() {
    const [platform, setPlatform] = useState<string | null>(null);
    const [activeSlots, setActiveSlots] = useState<boolean[]>([]);
    const [charNames, setCharacterNames] = useState<string[]>([]);
    const [selectedChar, setSelectedChar] = useState<number>(0);
    const [activeTab, setActiveTab] = useState('character');
    const [theme, setTheme] = useState<Theme>('light');
    const [columnVisibility, setColumnVisibility] = useState({
        id: false,
        category: true
    });

    const tabs = ['character', 'inventory', 'world progress', 'importer', 'settings'];

    useEffect(() => {
        const root = document.documentElement;
        if (theme === 'system') {
            const systemTheme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
            root.classList.toggle('dark', systemTheme === 'dark');
        } else {
            root.classList.toggle('dark', theme === 'dark');
        }
    }, [theme]);

    const handleOpenSave = async () => {
        try {
            const plat = await SelectAndOpenSave();
            setPlatform(plat);
            refreshSlots();
        } catch (err) {
            alert(err);
        }
    };

    const refreshSlots = async () => {
        const slots = await GetActiveSlots();
        const names = await GetCharacterNames();
        setActiveSlots(slots);
        setCharacterNames(names);
    };

    const toggleSlot = async (idx: number) => {
        await SetSlotActivity(idx, !activeSlots[idx]);
        refreshSlots();
    };

    return (
        <div className="flex h-screen bg-background text-foreground overflow-hidden font-sans selection:bg-primary/30 transition-colors duration-300">
            {/* Sidebar */}
            <aside className="w-64 border-r border-border bg-muted/5 flex flex-col z-20">
                <div className="p-5 space-y-6 flex-1 overflow-y-auto custom-scrollbar">
                    <div className="flex items-center justify-between px-1">
                        <div className="flex items-center space-x-3">
                            <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center shadow-lg shadow-primary/20">
                                <span className="text-primary-foreground font-black text-lg tracking-tighter">ER</span>
                            </div>
                            <h1 className="text-[10px] font-black uppercase tracking-[0.2em] leading-none">Editor</h1>
                        </div>
                    </div>

                    <button 
                        onClick={handleOpenSave}
                        className="w-full bg-primary text-primary-foreground font-black py-3 rounded-lg text-[9px] uppercase tracking-[0.2em] shadow-xl shadow-primary/20 hover:brightness-110 active:scale-95 transition-all"
                    >
                        {platform ? `Change Save (${platform})` : 'Open Save File'}
                    </button>

                    {platform && (
                        <div className="space-y-4 animate-in fade-in slide-in-from-left-2 duration-500">
                            <div className="flex items-center justify-between px-1">
                                <h2 className="text-[9px] font-black uppercase tracking-[0.2em] text-muted-foreground">Characters</h2>
                                <span className="text-[8px] font-bold bg-muted/30 px-2 py-0.5 rounded-full text-muted-foreground">{activeSlots.filter(s => s).length}/10</span>
                            </div>
                            <div className="space-y-1">
                                {charNames.map((name, idx) => (
                                    <div 
                                        key={idx}
                                        onClick={() => setSelectedChar(idx)}
                                        className={`group relative p-2.5 rounded-lg border transition-all cursor-pointer ${selectedChar === idx ? 'bg-muted/30 border-primary/40 ring-1 ring-primary/10 shadow-lg' : 'bg-transparent border-border/30 hover:border-border hover:bg-muted/10'}`}
                                    >
                                        <div className="flex items-center justify-between relative z-10">
                                            <div className="flex items-center space-x-2.5">
                                                <div className={`w-1.5 h-1.5 rounded-full ${activeSlots[idx] ? 'bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.5)]' : 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.3)]'}`} />
                                                <span className={`text-[10px] font-bold uppercase tracking-tight transition-colors ${selectedChar === idx ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`}>
                                                    {name}
                                                </span>
                                            </div>
                                            <button 
                                                onClick={(e) => { e.stopPropagation(); toggleSlot(idx); }}
                                                className={`p-1 rounded-md transition-all ${activeSlots[idx] ? 'text-green-500 hover:bg-green-500/10' : 'text-red-500 hover:bg-red-500/10'}`}
                                            >
                                                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3" d={activeSlots[idx] ? "M5 13l4 4L19 7" : "M6 18L18 6M6 6l12 12"}></path></svg>
                                            </button>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
                
                <div className="p-4 border-t border-border bg-muted/5 space-y-4">
                    <div className="flex items-center justify-between text-[8px] font-bold text-muted-foreground uppercase tracking-widest opacity-50">
                        <span>v0.1.0 Alpha</span>
                        <span>System Ready</span>
                    </div>
                </div>
            </aside>

            {/* Main Content */}
            <main className="flex-1 flex flex-col relative z-10 bg-background overflow-hidden">
                <header className="h-14 border-b border-border flex items-center justify-between px-8 bg-background/50 backdrop-blur-md sticky top-0 z-30">
                    <nav className="flex space-x-1">
                        {tabs.map(tab => (
                            <button
                                key={tab}
                                onClick={() => setActiveTab(tab)}
                                className={`px-4 py-1.5 rounded-full text-[9px] font-black uppercase tracking-[0.2em] transition-all ${activeTab === tab ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20' : 'text-muted-foreground hover:text-foreground hover:bg-muted/30'}`}
                            >
                                {tab}
                            </button>
                        ))}
                    </nav>
                    <div className="flex items-center space-x-4">
                        <div className="text-right">
                            <p className="text-[9px] font-black uppercase tracking-widest text-foreground leading-none mb-1">
                                {charNames[selectedChar] || 'No Slot'}
                            </p>
                            <p className="text-[7px] font-bold text-muted-foreground uppercase tracking-[0.2em]">
                                Slot {selectedChar + 1}
                            </p>
                        </div>
                    </div>
                </header>

                <div className="flex-1 flex flex-col min-h-0 relative">
                    <div className="w-full h-full p-6 flex flex-col min-h-0">
                        {activeTab === 'settings' ? (
                            <div className="animate-in fade-in slide-in-from-bottom-2 duration-500 overflow-y-auto custom-scrollbar pr-2">
                                <SettingsTab 
                                    theme={theme} 
                                    setTheme={setTheme} 
                                    columnVisibility={columnVisibility} 
                                    setColumnVisibility={setColumnVisibility}
                                    platform={platform}
                                    setPlatform={setPlatform}
                                    refreshSlots={refreshSlots}
                                />
                            </div>
                        ) : !platform ? (
                            <div className="flex-1 flex flex-col items-center justify-center text-center space-y-6 animate-in fade-in zoom-in-95 duration-700">
                                <div className="w-16 h-16 bg-muted/10 rounded-2xl flex items-center justify-center border border-border/50">
                                    <svg className="w-8 h-8 text-muted-foreground/30" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
                                </div>
                                <h2 className="text-sm font-black uppercase tracking-[0.3em] text-foreground/60">No Save File</h2>
                                <button 
                                    onClick={handleOpenSave}
                                    className="px-6 py-2 bg-primary text-primary-foreground rounded-full text-[9px] font-black uppercase tracking-[0.2em] transition-all shadow-lg shadow-primary/20"
                                >
                                    Open Save
                                </button>
                            </div>
                        ) : (
                            <div className="flex-1 flex flex-col min-h-0 animate-in fade-in slide-in-from-bottom-2 duration-500">
                                {activeTab === 'character' && <GeneralTab charIndex={selectedChar} />}
                                {activeTab === 'inventory' && <InventoryTab charIndex={selectedChar} columnVisibility={columnVisibility} />}
                                {activeTab === 'world progress' && <WorldProgressTab />}
                                {activeTab === 'importer' && <CharacterImporter destSlot={selectedChar} onComplete={refreshSlots} />}
                            </div>
                        )}
                    </div>
                </div>
            </main>
        </div>
    );
}

export default App;
