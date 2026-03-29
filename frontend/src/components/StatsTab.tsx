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
        const newChar = {...char, [key]: clampedVal} as any;
        const sum = newChar.vigor + newChar.mind + newChar.endurance + newChar.strength + 
                    newChar.dexterity + newChar.intelligence + newChar.faith + newChar.arcane;
        newChar.level = Math.max(1, sum - 79);
        setChar(newChar);
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
        <div className="space-y-10 animate-in fade-in duration-500">
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-12 items-start">
                {/* Attributes */}
                <div className="lg:col-span-2 space-y-6">
                    <div className="flex items-center space-x-2">
                        <div className="w-1 h-4 bg-blue-500 rounded-full" />
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Primary Attributes</h3>
                    </div>
                    
                    <div className="grid grid-cols-1 gap-2">
                        {attributes.map(stat => (
                            <div key={stat.id} className="flex items-center justify-between bg-muted/20 border border-border rounded px-4 py-3 hover:bg-muted/40 transition-colors group">
                                <div className="min-w-[100px]">
                                    <label className="text-xs font-semibold text-muted-foreground uppercase tracking-tight group-hover:text-foreground transition-colors">
                                        {stat.label}
                                    </label>
                                </div>
                                <div className="flex items-center space-x-6 flex-1 max-w-md">
                                    <input 
                                        type="range"
                                        min="1" max="99"
                                        value={(char as any)[stat.id]}
                                        onChange={e => updateStat(stat.id, parseInt(e.target.value))}
                                        className="flex-1 h-1 bg-zinc-200 dark:bg-zinc-800 rounded-full appearance-none cursor-pointer accent-foreground"
                                    />
                                    <input 
                                        type="number" 
                                        min="1" max="99"
                                        value={(char as any)[stat.id]} 
                                        onChange={e => updateStat(stat.id, parseInt(e.target.value) || 1)}
                                        className="w-12 bg-background border border-border rounded py-1 text-center text-sm font-bold focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
                                    />
                                </div>
                            </div>
                        ))}
                    </div>
                </div>

                {/* Summary */}
                <div className="sticky top-8 space-y-6">
                    <div className="flex items-center space-x-2">
                        <div className="w-1 h-4 bg-zinc-500 rounded-full" />
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Summary</h3>
                    </div>
                    <div className="bg-muted/30 border border-border rounded-lg p-8 flex flex-col items-center justify-center text-center relative overflow-hidden">
                        <div className="absolute top-0 left-0 w-full h-0.5 bg-blue-500/50" />
                        <span className="text-xs font-medium text-muted-foreground uppercase tracking-widest mb-2">Calculated Level</span>
                        <span className="text-7xl font-bold tracking-tighter">{char.level}</span>
                        <div className="mt-8 pt-6 border-t border-border w-full">
                            <button 
                                onClick={handleSave}
                                className="w-full bg-foreground text-background hover:opacity-90 transition-opacity font-semibold py-2.5 rounded text-xs shadow-sm uppercase tracking-wider"
                            >
                                Apply Attributes
                            </button>
                            <p className="mt-4 text-[10px] font-medium text-muted-foreground uppercase tracking-tight opacity-60">
                                Level = Σ(Attributes) - 79
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
