import {useEffect, useState} from 'react';
import {GetCharacter, SaveCharacter} from '../../wailsjs/go/main/App';
import {vm} from '../../wailsjs/go/models';

interface Props {
    charIndex: number;
}

export function StatsTab({charIndex}: Props) {
    const [char, setChar] = useState<vm.CharacterViewModel | null>(null);
    const [loading, setLoading] = useState(false);

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
                .then(() => alert('Attributes updated in memory'))
                .catch(err => alert('Error: ' + err));
        }
    };

    if (loading) return (
        <div className="py-20 flex flex-col items-center justify-center space-y-4">
            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
            <p className="text-xs font-medium text-muted-foreground">Calculating stats...</p>
        </div>
    );

    if (!char) return null;

    return (
        <div className="space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700">
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-10 items-start">
                {/* Attributes */}
                <div className="lg:col-span-2 space-y-6">
                    <div className="flex items-center space-x-2 px-1">
                        <div className="w-1 h-3 bg-primary rounded-full" />
                        <h3 className="text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Primary Attributes</h3>
                    </div>
                    
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                        {attributes.map(stat => (
                            <div key={stat.id} className="card p-4 hover:border-primary/30 transition-all group">
                                <div className="flex items-center justify-between mb-4">
                                    <label className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest group-hover:text-foreground transition-colors">
                                        {stat.label}
                                    </label>
                                    <input 
                                        type="number" 
                                        min="1" max="99"
                                        value={(char as any)[stat.id]} 
                                        onChange={e => updateStat(stat.id, parseInt(e.target.value) || 1)}
                                        className="w-10 bg-muted/30 border border-border rounded text-center text-xs font-black focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all py-1"
                                    />
                                </div>
                                <input 
                                    type="range"
                                    min="1" max="99"
                                    value={(char as any)[stat.id]}
                                    onChange={e => updateStat(stat.id, parseInt(e.target.value))}
                                    className="w-full h-1 bg-muted rounded-full appearance-none cursor-pointer accent-primary"
                                />
                            </div>
                        ))}
                    </div>
                </div>

                {/* Summary */}
                <div className="sticky top-20 space-y-6">
                    <div className="flex items-center space-x-2 px-1">
                        <div className="w-1 h-3 bg-zinc-500 rounded-full" />
                        <h3 className="text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Summary</h3>
                    </div>
                    <div className="card p-8 flex flex-col items-center justify-center text-center relative overflow-hidden group">
                        <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-primary to-indigo-600 opacity-50" />
                        
                        <div className="w-20 h-20 rounded-full bg-muted/30 border border-border/50 flex items-center justify-center overflow-hidden mb-6 group-hover:border-primary/50 transition-all shadow-inner">
                            <img 
                                src="items/armor/raging_wolf_armor.png" 
                                alt="Character" 
                                className="w-14 h-14 object-contain opacity-80 group-hover:opacity-100 group-hover:scale-110 transition-all"
                            />
                        </div>

                        <span className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.3em] mb-4">Calculated Level</span>
                        <div className="relative mb-8">
                            <span className="text-7xl font-black tracking-tighter bg-clip-text text-transparent bg-gradient-to-b from-foreground to-foreground/70">
                                {char.level}
                            </span>
                        </div>

                        <div className="w-full space-y-4">
                            <button 
                                onClick={handleSave}
                                className="w-full bg-foreground text-background hover:scale-[1.02] active:scale-[0.98] transition-all font-black py-3 rounded-md text-[11px] shadow-xl uppercase tracking-[0.2em]"
                            >
                                Apply Attributes
                            </button>
                            <p className="text-[9px] font-bold text-muted-foreground uppercase tracking-tight opacity-40">
                                Level = Σ(Attributes) - 79
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
