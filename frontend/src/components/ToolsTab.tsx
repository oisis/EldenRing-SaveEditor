import {useState} from 'react';
import {CharacterImporter} from './CharacterImporter';
import {DatabaseTab} from './DatabaseTab';
import {AccordionSection} from './AccordionSection';
import type {AddSettings} from '../App';
import {db} from '../../wailsjs/go/models';

interface ToolsTabProps {
    charIndex: number;
    onComplete: () => void;
    platform: string | null;
    inventoryVersion: number;
    setInventoryVersion: (fn: (v: number) => number) => void;
    addSettings: AddSettings;
    setAddSettings: (s: AddSettings) => void;
    columnVisibility: { id: boolean; category: boolean };
    showFlaggedItems: boolean;
    category: string;
    setCategory: (c: string) => void;
    infuseTypes: db.InfuseType[];
}

type ToolView = 'overview' | 'importer' | 'database';

export function ToolsTab({
    charIndex, onComplete, platform, inventoryVersion, setInventoryVersion,
    addSettings, setAddSettings, columnVisibility, showFlaggedItems, category, setCategory, infuseTypes,
}: ToolsTabProps) {
    const [view, setView] = useState<ToolView>('overview');

    if (view === 'importer') {
        return (
            <div className="space-y-3 animate-in fade-in duration-300">
                <button onClick={() => setView('overview')}
                    className="flex items-center gap-1.5 text-[9px] font-black uppercase tracking-widest text-muted-foreground hover:text-foreground transition-colors">
                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M15 19l-7-7 7-7" />
                    </svg>
                    Back to Tools
                </button>
                <CharacterImporter destSlot={charIndex} onComplete={onComplete} />
            </div>
        );
    }

    if (view === 'database') {
        return (
            <div className="flex-1 flex flex-col min-h-0 space-y-3 animate-in fade-in duration-300">
                <button onClick={() => setView('overview')}
                    className="flex items-center gap-1.5 text-[9px] font-black uppercase tracking-widest text-muted-foreground hover:text-foreground transition-colors flex-shrink-0">
                    <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M15 19l-7-7 7-7" />
                    </svg>
                    Back to Tools
                </button>
                <AccordionSection id="tools-add-settings" title="Add Settings"
                    summary={`+${addSettings.upgrade25} / +${addSettings.upgrade10} / Ash +${addSettings.upgradeAsh}`}>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-x-10 gap-y-4 py-1">
                        <div className="flex items-center space-x-3">
                            <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground w-24 shrink-0">Weapon +25</span>
                            <input type="range" min={0} max={25} value={addSettings.upgrade25} onChange={e => setAddSettings({...addSettings, upgrade25: parseInt(e.target.value)})}
                                className="flex-1 h-1.5 rounded-lg appearance-none cursor-pointer [&::-webkit-slider-runnable-track]:bg-border [&::-webkit-slider-runnable-track]:rounded-lg" />
                            <span className="text-[10px] font-mono font-bold text-primary w-6 text-right">+{addSettings.upgrade25}</span>
                        </div>
                        <div className="flex items-center space-x-3">
                            <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground w-24 shrink-0">Weapon +10</span>
                            <input type="range" min={0} max={10} value={addSettings.upgrade10} onChange={e => setAddSettings({...addSettings, upgrade10: parseInt(e.target.value)})}
                                className="flex-1 h-1.5 rounded-lg appearance-none cursor-pointer [&::-webkit-slider-runnable-track]:bg-border [&::-webkit-slider-runnable-track]:rounded-lg" />
                            <span className="text-[10px] font-mono font-bold text-primary w-5 text-right">+{addSettings.upgrade10}</span>
                        </div>
                        <div className="flex items-center space-x-3">
                            <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground w-24 shrink-0">Infuse</span>
                            <select value={addSettings.infuseOffset} onChange={e => setAddSettings({...addSettings, infuseOffset: parseInt(e.target.value)})}
                                className="flex-1 bg-muted/20 border border-border rounded-md px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider focus:ring-1 focus:ring-primary/30 outline-none transition-all cursor-pointer">
                                {infuseTypes.map(t => <option key={t.offset} value={t.offset}>{t.name}</option>)}
                            </select>
                        </div>
                        <div className="flex items-center space-x-3">
                            <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground w-24 shrink-0">Spirit Ash</span>
                            <input type="range" min={0} max={10} value={addSettings.upgradeAsh} onChange={e => setAddSettings({...addSettings, upgradeAsh: parseInt(e.target.value)})}
                                className="flex-1 h-1.5 rounded-lg appearance-none cursor-pointer [&::-webkit-slider-runnable-track]:bg-border [&::-webkit-slider-runnable-track]:rounded-lg" />
                            <span className="text-[10px] font-mono font-bold text-primary w-5 text-right">+{addSettings.upgradeAsh}</span>
                        </div>
                    </div>
                </AccordionSection>
                <DatabaseTab
                    columnVisibility={columnVisibility}
                    platform={platform}
                    charIndex={charIndex}
                    onItemsAdded={() => setInventoryVersion(v => v + 1)}
                    addSettings={addSettings}
                    showFlaggedItems={showFlaggedItems}
                    category={category}
                    setCategory={setCategory}
                />
            </div>
        );
    }

    return (
        <div className="space-y-6 animate-in fade-in duration-500 max-w-4xl mx-auto">
            <div className="flex items-center space-x-2">
                <div className="w-1 h-3 bg-primary rounded-full" />
                <h3 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Tools</h3>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {/* Item Database */}
                <button onClick={() => setView('database')}
                    className="card p-5 text-left hover:border-primary/40 hover:bg-muted/10 transition-all group">
                    <div className="flex items-start gap-3">
                        <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0 group-hover:bg-primary/20 transition-colors">
                            <svg className="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
                            </svg>
                        </div>
                        <div>
                            <h4 className="text-[11px] font-black uppercase tracking-wider text-foreground">Item Database</h4>
                            <p className="text-[9px] text-muted-foreground mt-1">Browse all game items and add them to character inventory</p>
                        </div>
                    </div>
                </button>

                {/* Character Importer */}
                <button onClick={() => setView('importer')}
                    className="card p-5 text-left hover:border-primary/40 hover:bg-muted/10 transition-all group">
                    <div className="flex items-start gap-3">
                        <div className="w-10 h-10 rounded-lg bg-info/10 flex items-center justify-center flex-shrink-0 group-hover:bg-info/20 transition-colors">
                            <svg className="w-5 h-5 text-info" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                            </svg>
                        </div>
                        <div>
                            <h4 className="text-[11px] font-black uppercase tracking-wider text-foreground">Character Importer</h4>
                            <p className="text-[9px] text-muted-foreground mt-1">Import character from another save file into the selected slot</p>
                        </div>
                    </div>
                </button>

                {/* Save Comparison — placeholder */}
                <div className="card p-5 text-left opacity-50 cursor-not-allowed">
                    <div className="flex items-start gap-3">
                        <div className="w-10 h-10 rounded-lg bg-warning/10 flex items-center justify-center flex-shrink-0">
                            <svg className="w-5 h-5 text-warning" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                            </svg>
                        </div>
                        <div>
                            <h4 className="text-[11px] font-black uppercase tracking-wider text-foreground">Save Comparison</h4>
                            <p className="text-[9px] text-muted-foreground mt-1">Compare two save files side by side (coming soon)</p>
                        </div>
                    </div>
                </div>

                {/* Diagnostics — placeholder */}
                <div className="card p-5 text-left opacity-50 cursor-not-allowed">
                    <div className="flex items-start gap-3">
                        <div className="w-10 h-10 rounded-lg bg-destructive/10 flex items-center justify-center flex-shrink-0">
                            <svg className="w-5 h-5 text-destructive" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                            </svg>
                        </div>
                        <div>
                            <h4 className="text-[11px] font-black uppercase tracking-wider text-foreground">Diagnostics</h4>
                            <p className="text-[9px] text-muted-foreground mt-1">Detect and repair save file corruption (coming soon)</p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
