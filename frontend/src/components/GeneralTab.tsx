import {useEffect, useState} from 'react';
import {GetCharacter, SaveCharacter} from '../../wailsjs/go/main/App';
import {vm} from '../../wailsjs/go/models';

interface Props {
    charIndex: number;
}

export function GeneralTab({charIndex}: Props) {
    const [char, setChar] = useState<vm.CharacterViewModel | null>(null);
    const [loading, setLoading] = useState(false);

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

    const handleSave = () => {
        if (char) {
            SaveCharacter(charIndex, char)
                .then(() => alert('Character data updated in memory'))
                .catch(err => alert('Error: ' + err));
        }
    };

    if (loading) return (
        <div className="py-20 flex flex-col items-center justify-center space-y-4">
            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
            <p className="text-xs font-medium text-muted-foreground">Loading character...</p>
        </div>
    );

    if (!char) return (
        <div className="py-20 text-center border border-dashed border-border rounded-lg">
            <p className="text-sm text-muted-foreground">No data found in this slot.</p>
        </div>
    );

    return (
        <div className="space-y-10 animate-in fade-in duration-500">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-12">
                {/* Identity */}
                <div className="space-y-6">
                    <div className="flex items-center space-x-2">
                        <div className="w-1 h-4 bg-blue-500 rounded-full" />
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Identity</h3>
                    </div>
                    
                    <div className="space-y-5">
                        <div className="space-y-2">
                            <label className="block text-xs font-medium text-muted-foreground">Character Name</label>
                            <input 
                                type="text" 
                                value={char.name} 
                                onChange={e => setChar({...char, name: e.target.value})}
                                className="w-full bg-muted/50 border border-border rounded px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
                                maxLength={16}
                            />
                        </div>

                        <div className="space-y-2">
                            <label className="block text-xs font-medium text-muted-foreground">Runes (Souls)</label>
                            <input 
                                type="number" 
                                value={char.souls} 
                                onChange={e => setChar({...char, souls: parseInt(e.target.value) || 0})}
                                className="w-full bg-muted/50 border border-border rounded px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all font-mono"
                            />
                        </div>
                    </div>
                </div>

                {/* Level Card */}
                <div className="space-y-6">
                    <div className="flex items-center space-x-2">
                        <div className="w-1 h-4 bg-zinc-500 rounded-full" />
                        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Summary</h3>
                    </div>
                    <div className="bg-muted/30 border border-border rounded-lg p-8 flex flex-col items-center justify-center text-center h-[180px] relative overflow-hidden">
                        <div className="absolute top-0 left-0 w-full h-0.5 bg-blue-500/50" />
                        <span className="text-xs font-medium text-muted-foreground uppercase tracking-widest mb-2">Calculated Level</span>
                        <span className="text-6xl font-bold tracking-tighter">{char.level}</span>
                        <p className="mt-4 text-[10px] font-medium text-muted-foreground uppercase tracking-tight opacity-60">
                            Derived from attributes
                        </p>
                    </div>
                </div>
            </div>

            <div className="pt-8 border-t border-border flex justify-end">
                <button 
                    onClick={handleSave}
                    className="bg-foreground text-background hover:opacity-90 transition-opacity font-semibold px-6 py-2 rounded text-xs shadow-sm uppercase tracking-wider"
                >
                    Apply Changes
                </button>
            </div>
        </div>
    );
}
