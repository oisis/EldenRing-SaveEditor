import {useState, useEffect} from 'react';
import {SelectAndOpenSave, GetActiveSlots, SetSlotActivity, GetCharacterNames, WriteSave, CloneSlot, DeleteSlot} from '../wailsjs/go/main/App';
import {GeneralTab} from './components/GeneralTab';
import {InventoryTab} from './components/InventoryTab';
import {WorldProgressTab} from './components/WorldProgressTab';
import {CharacterImporter} from './components/CharacterImporter';
import {SettingsTab} from './components/SettingsTab';
import {DatabaseTab} from './components/DatabaseTab';

type Theme = 'light' | 'dark' | 'system';

function App() {
    const [platform, setPlatform] = useState<string | null>(null);
    const [activeSlots, setActiveSlots] = useState<boolean[]>([]);
    const [charNames, setCharacterNames] = useState<string[]>([]);
    const [selectedChar, setSelectedChar] = useState<number>(0);
    const [activeTab, setActiveTab] = useState('database');
    const [inventoryVersion, setInventoryVersion] = useState(0);
    const [theme, setTheme] = useState<Theme>('light');
    const [cloneModal, setCloneModal] = useState<{srcIdx: number} | null>(null);
    const [columnVisibility, setColumnVisibility] = useState({
        id: false,
        category: true
    });

    const tabs = ['database', 'character', 'inventory', 'world progress', 'importer', 'settings'];

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

    const handleClone = async (srcIdx: number, destIdx: number) => {
        try {
            await CloneSlot(srcIdx, destIdx);
            refreshSlots();
            setCloneModal(null);
        } catch (err) {
            alert(err);
        }
    };

    const handleDelete = async (idx: number) => {
        const name = charNames[idx];
        if (!confirm(`Delete "${name}"? This cannot be undone.`)) return;
        try {
            await DeleteSlot(idx);
            if (selectedChar > 0 && selectedChar >= idx) setSelectedChar(selectedChar - 1);
            refreshSlots();
        } catch (err) {
            alert(err);
        }
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
                                            <div className="flex items-center space-x-2.5 min-w-0">
                                                <div className={`w-1.5 h-1.5 flex-shrink-0 rounded-full ${activeSlots[idx] ? 'bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.5)]' : 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.3)]'}`} />
                                                <span className={`text-[10px] font-bold uppercase tracking-tight truncate transition-colors ${selectedChar === idx ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`}>
                                                    {name}
                                                </span>
                                            </div>
                                            <div className="flex items-center gap-0.5 flex-shrink-0 ml-1">
                                                {activeSlots[idx] && (<>
                                                    <button
                                                        onClick={(e) => { e.stopPropagation(); setCloneModal({srcIdx: idx}); }}
                                                        title="Clone character"
                                                        className="p-1 rounded-md opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-primary hover:bg-primary/10"
                                                    >
                                                        <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>
                                                    </button>
                                                    <button
                                                        onClick={(e) => { e.stopPropagation(); handleDelete(idx); }}
                                                        title="Delete character"
                                                        className="p-1 rounded-md opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-red-500 hover:bg-red-500/10"
                                                    >
                                                        <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                                                    </button>
                                                </>)}
                                                <button
                                                    onClick={(e) => { e.stopPropagation(); toggleSlot(idx); }}
                                                    className={`p-1 rounded-md transition-all ${activeSlots[idx] ? 'text-green-500 hover:bg-green-500/10' : 'text-red-500 hover:bg-red-500/10'}`}
                                                >
                                                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3" d={activeSlots[idx] ? "M5 13l4 4L19 7" : "M6 18L18 6M6 6l12 12"}></path></svg>
                                                </button>
                                            </div>
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
                                onClick={() => {
                                    if (tab === 'inventory') setInventoryVersion(v => v + 1);
                                    setActiveTab(tab);
                                }}
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
                        {activeTab === 'database' ? (
                            <div className="flex-1 flex flex-col min-h-0 animate-in fade-in slide-in-from-bottom-2 duration-500">
                                <DatabaseTab
                                    columnVisibility={columnVisibility}
                                    platform={platform}
                                    charIndex={selectedChar}
                                    onItemsAdded={() => setInventoryVersion(v => v + 1)}
                                />
                            </div>
                        ) : activeTab === 'settings' ? (
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
                                {activeTab === 'character' && <GeneralTab charIndex={selectedChar} onNameChange={refreshSlots} />}
                                {activeTab === 'inventory' && <InventoryTab charIndex={selectedChar} inventoryVersion={inventoryVersion} columnVisibility={columnVisibility} />}
                                {activeTab === 'world progress' && <WorldProgressTab charIdx={selectedChar} />}
                                {activeTab === 'importer' && <CharacterImporter destSlot={selectedChar} onComplete={refreshSlots} />}
                            </div>
                        )}
                    </div>
                </div>
            </main>
        {cloneModal && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setCloneModal(null)}>
                <div className="bg-background border border-border rounded-xl p-6 w-72 space-y-4 shadow-2xl" onClick={(e) => e.stopPropagation()}>
                    <div className="flex items-center space-x-2">
                        <div className="w-1 h-4 bg-primary rounded-full" />
                        <h3 className="text-[10px] font-black uppercase tracking-widest">Clone to Slot</h3>
                    </div>
                    <p className="text-[9px] text-muted-foreground uppercase tracking-wide">
                        Cloning: <span className="text-foreground font-bold">{charNames[cloneModal.srcIdx]}</span>
                    </p>
                    <div className="space-y-1">
                        {charNames.map((_, idx) => {
                            if (activeSlots[idx]) return null;
                            return (
                                <button
                                    key={idx}
                                    onClick={() => handleClone(cloneModal.srcIdx, idx)}
                                    className="w-full text-left px-3 py-2.5 rounded-lg border border-border/50 hover:border-primary/40 hover:bg-muted/20 transition-all text-[10px] font-bold uppercase tracking-wider"
                                >
                                    Slot {idx + 1} — Empty
                                </button>
                            );
                        })}
                        {charNames.every((_, idx) => activeSlots[idx]) && (
                            <p className="text-[10px] text-muted-foreground text-center py-4">No empty slots available</p>
                        )}
                    </div>
                    <button
                        onClick={() => setCloneModal(null)}
                        className="w-full py-2 text-[9px] font-black uppercase tracking-widest text-muted-foreground hover:text-foreground transition-colors"
                    >
                        Cancel
                    </button>
                </div>
            </div>
        )}
        </div>
    );
}

export default App;
