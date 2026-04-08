import {useEffect, useState} from 'react';
import {GetCharacter, SaveCharacter, GetInfuseTypes} from '../../wailsjs/go/main/App';
import {vm, db} from '../../wailsjs/go/models';
import type {AddSettings} from '../App';

interface Props {
    charIndex: number;
    onNameChange?: () => void;
    addSettings: AddSettings;
    setAddSettings: (s: AddSettings) => void;
}

export function GeneralTab({charIndex, onNameChange, addSettings, setAddSettings}: Props) {
    const {upgrade25, upgrade10, infuseOffset, upgradeAsh} = addSettings;
    const setUpgrade25 = (v: number) => setAddSettings({...addSettings, upgrade25: v});
    const setUpgrade10 = (v: number) => setAddSettings({...addSettings, upgrade10: v});
    const setInfuseOffset = (v: number) => setAddSettings({...addSettings, infuseOffset: v});
    const setUpgradeAsh = (v: number) => setAddSettings({...addSettings, upgradeAsh: v});
    const [char, setChar] = useState<vm.CharacterViewModel | null>(null);
    const [loading, setLoading] = useState(false);
    const [infuseTypes, setInfuseTypes] = useState<db.InfuseType[]>([]);

    const attributes = [
        { id: 'vigor', label: 'Vigor' },
        { id: 'mind', label: 'Mind' },
        { id: 'endurance', label: 'Endurance' },
        { id: 'strength', label: 'Strength' },
        { id: 'dexterity', label: 'Dexterity' },
        { id: 'intelligence', label: 'Intelligence' },
        { id: 'faith', label: 'Faith' },
        { id: 'arcane', label: 'Arcane' }
    ];

    useEffect(() => {
        GetInfuseTypes().then(res => setInfuseTypes(res || []));
    }, []);

    useEffect(() => {
        setLoading(true);
        GetCharacter(charIndex)
            .then(res => {
                setChar(res);
                setLoading(false);
            })
            .catch(err => {
                console.error(err);
                setLoading(false);
            });
    }, [charIndex]);

    const updateStat = (key: string, val: number) => {
        if (!char) return;
        const clampedVal = Math.min(99, Math.max(1, val));
        const updatedData = {...char, [key]: clampedVal} as any;
        const sum = updatedData.vigor + updatedData.mind + updatedData.endurance + updatedData.strength + 
                    updatedData.dexterity + updatedData.intelligence + updatedData.faith + updatedData.arcane;
        updatedData.level = Math.max(1, sum - 79);
        setChar(vm.CharacterViewModel.createFrom(updatedData));
    };

    const handleSave = () => {
        if (char) {
            SaveCharacter(charIndex, char)
                .then(() => {
                    alert('Character data updated in memory');
                    onNameChange?.();
                })
                .catch(err => alert('Error: ' + err));
        }
    };

    if (loading) return (
        <div className="py-10 flex flex-col items-center justify-center space-y-3">
            <div className="w-5 h-5 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Loading...</p>
        </div>
    );

    if (!char) return (
        <div className="py-10 text-center border border-dashed border-border rounded-lg">
            <p className="text-xs text-muted-foreground">No character data.</p>
        </div>
    );

    return (
        <div className="space-y-6 animate-in fade-in duration-500 max-w-5xl mx-auto overflow-y-auto custom-scrollbar">
            <div className="grid grid-cols-1 md:grid-cols-12 gap-6">
                {/* Left Column: Identity & Level */}
                <div className="md:col-span-4 space-y-6">
                    <div className="card p-5 space-y-4">
                        <div className="flex items-center space-x-2 mb-2">
                            <div className="w-1 h-3 bg-primary rounded-full" />
                            <h3 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Profile</h3>
                        </div>
                        
                        <div className="space-y-1.5">
                            <label className="text-[9px] font-bold text-muted-foreground uppercase tracking-tight ml-1">Character Name</label>
                            <input 
                                type="text" 
                                value={char.name} 
                                onChange={e => setChar(vm.CharacterViewModel.createFrom({...char, name: e.target.value}))}
                                className="w-full bg-muted/20 border border-border rounded-md px-3 py-2 text-xs font-bold focus:ring-1 focus:ring-primary/30 outline-none transition-all"
                                maxLength={16}
                            />
                        </div>

                        <div className="space-y-1.5">
                            <label className="text-[9px] font-bold text-muted-foreground uppercase tracking-tight ml-1">Runes</label>
                            <input 
                                type="number" 
                                value={char.souls} 
                                onChange={e => setChar(vm.CharacterViewModel.createFrom({...char, souls: parseInt(e.target.value) || 0}))}
                                className="w-full bg-muted/20 border border-border rounded-md px-3 py-2 text-xs font-black font-mono focus:ring-1 focus:ring-primary/30 outline-none transition-all"
                            />
                        </div>
                    </div>

                    <div className="card p-6 flex flex-col items-center justify-center text-center relative overflow-hidden">
                        <div className="absolute top-0 left-0 w-full h-0.5 bg-primary/50" />
                        <span className="text-[9px] font-black text-muted-foreground uppercase tracking-[0.2em] mb-1">Current Level</span>
                        <span className="text-5xl font-black tracking-tighter text-foreground">{char.level}</span>
                    </div>
                </div>

                {/* Right Column: Attributes Grid */}
                <div className="md:col-span-8 card p-5">
                    <div className="flex items-center space-x-2 mb-6">
                        <div className="w-1 h-3 bg-zinc-500 rounded-full" />
                        <h3 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Attributes</h3>
                    </div>
                    
                    <div className="grid grid-cols-2 gap-x-8 gap-y-4">
                        {attributes.map(stat => (
                            <div key={stat.id} className="flex items-center justify-between group py-1 border-b border-border/30 hover:border-primary/30 transition-all">
                                <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider group-hover:text-foreground transition-colors">
                                    {stat.label}
                                </label>
                                <input 
                                    type="number" 
                                    min="1" max="99"
                                    value={(char as any)[stat.id]} 
                                    onChange={e => updateStat(stat.id, parseInt(e.target.value) || 1)}
                                    className="w-12 bg-muted/30 border border-border rounded text-center text-xs font-black py-1 focus:ring-1 focus:ring-primary/30 outline-none"
                                />
                            </div>
                        ))}
                    </div>

                    <div className="mt-8 pt-6 border-t border-border/50 flex justify-end items-center space-x-4">
                        <p className="text-[8px] font-bold text-muted-foreground uppercase tracking-widest italic opacity-50">
                            Staged in memory
                        </p>
                        <button 
                            onClick={handleSave}
                            className="bg-primary text-primary-foreground hover:brightness-110 active:scale-95 transition-all font-black px-6 py-2 rounded-md text-[10px] uppercase tracking-widest shadow-lg shadow-primary/20"
                        >
                            Apply Changes
                        </button>
                    </div>
                </div>
            </div>

            {/* Add Settings */}
            <div className="card p-5 space-y-4">
                <div className="flex items-center space-x-2">
                    <div className="w-1 h-3 bg-primary/60 rounded-full" />
                    <h3 className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Add Settings</h3>
                    <span className="text-[8px] font-bold text-muted-foreground/50 uppercase tracking-widest">— applied when adding items from Database</span>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-x-10 gap-y-4 pt-1">
                    {/* Weapon +25 */}
                    <div className="flex items-center space-x-3">
                        <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground w-24 shrink-0">Weapon +25</span>
                        <input
                            type="range" min={0} max={25} value={upgrade25}
                            onChange={e => setUpgrade25(parseInt(e.target.value))}
                            className="flex-1 h-1.5 bg-muted rounded-lg appearance-none cursor-pointer accent-primary"
                        />
                        <span className="text-[10px] font-mono font-bold text-primary w-6 text-right">+{upgrade25}</span>
                    </div>

                    {/* Weapon +10 */}
                    <div className="flex items-center space-x-3">
                        <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground w-24 shrink-0">Weapon +10</span>
                        <input
                            type="range" min={0} max={10} value={upgrade10}
                            onChange={e => setUpgrade10(parseInt(e.target.value))}
                            className="flex-1 h-1.5 bg-muted rounded-lg appearance-none cursor-pointer accent-primary"
                        />
                        <span className="text-[10px] font-mono font-bold text-primary w-5 text-right">+{upgrade10}</span>
                    </div>

                    {/* Infuse */}
                    <div className="flex items-center space-x-3">
                        <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground w-24 shrink-0">Infuse</span>
                        <select
                            value={infuseOffset}
                            onChange={e => setInfuseOffset(parseInt(e.target.value))}
                            className="flex-1 bg-muted/20 border border-border rounded-md px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider focus:ring-1 focus:ring-primary/30 outline-none transition-all cursor-pointer"
                        >
                            {infuseTypes.map(t => (
                                <option key={t.offset} value={t.offset}>{t.name}</option>
                            ))}
                        </select>
                    </div>

                    {/* Spirit Ash */}
                    <div className="flex items-center space-x-3">
                        <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground w-24 shrink-0">Spirit Ash</span>
                        <input
                            type="range" min={0} max={10} value={upgradeAsh}
                            onChange={e => setUpgradeAsh(parseInt(e.target.value))}
                            className="flex-1 h-1.5 bg-muted rounded-lg appearance-none cursor-pointer accent-primary"
                        />
                        <span className="text-[10px] font-mono font-bold text-primary w-5 text-right">+{upgradeAsh}</span>
                    </div>
                </div>
            </div>
        </div>
    );
}
