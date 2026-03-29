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
        <div className="space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-10">
                {/* Identity Card */}
                <div className="space-y-6">
                    <div className="flex items-center space-x-2 px-1">
                        <div className="w-1 h-3 bg-primary rounded-full" />
                        <h3 className="text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Identity & Profile</h3>
                    </div>
                    
                    <div className="card p-6 space-y-6">
                        <div className="space-y-2">
                            <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider">Character Name</label>
                            <div className="relative">
                                <input 
                                    type="text" 
                                    value={char.name} 
                                    onChange={e => setChar(vm.CharacterViewModel.createFrom({...char, name: e.target.value}))}
                                    className="w-full bg-muted/30 border border-border rounded-md px-3 py-2.5 text-sm font-semibold focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                                    maxLength={16}
                                />
                                <div className="absolute right-3 top-1/2 -translate-y-1/2 text-[9px] font-bold text-muted-foreground/50">
                                    {char.name.length}/16
                                </div>
                            </div>
                        </div>

                        <div className="space-y-2">
                            <label className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider">Runes (Souls)</label>
                            <div className="relative">
                                <input 
                                    type="number" 
                                    value={char.souls} 
                                    onChange={e => setChar(vm.CharacterViewModel.createFrom({...char, souls: parseInt(e.target.value) || 0}))}
                                    className="w-full bg-muted/30 border border-border rounded-md px-3 py-2.5 text-sm font-black focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all font-mono tracking-tight"
                                />
                                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                                    <svg className="w-4 h-4 text-primary/50" fill="currentColor" viewBox="0 0 24 24">
                                        <path d="M12 2L4.5 20.29l.71.71L12 18l6.79 3 .71-.71L12 2z" />
                                    </svg>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Level Summary Card */}
                <div className="space-y-6">
                    <div className="flex items-center space-x-2 px-1">
                        <div className="w-1 h-3 bg-zinc-500 rounded-full" />
                        <h3 className="text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Level Analysis</h3>
                    </div>
                    <div className="card p-8 flex flex-col items-center justify-center text-center h-[214px] relative overflow-hidden group">
                        <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-primary to-indigo-600 opacity-50" />
                        <div className="absolute -right-8 -bottom-8 w-32 h-32 bg-primary/5 rounded-full blur-3xl group-hover:bg-primary/10 transition-all duration-1000" />
                        
                        <span className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.3em] mb-4">Calculated Level</span>
                        <div className="relative">
                            <span className="text-7xl font-black tracking-tighter bg-clip-text text-transparent bg-gradient-to-b from-foreground to-foreground/70">
                                {char.level}
                            </span>
                            <div className="absolute -top-2 -right-4 w-2 h-2 bg-primary rounded-full animate-ping opacity-20" />
                        </div>
                        <p className="mt-6 text-[9px] font-bold text-muted-foreground uppercase tracking-widest opacity-40">
                            Verified via attribute summation
                        </p>
                    </div>
                </div>
            </div>

            <div className="pt-10 border-t border-border flex justify-end items-center space-x-6">
                <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest italic opacity-50">
                    Changes are staged in memory
                </p>
                <button 
                    onClick={handleSave}
                    className="bg-foreground text-background hover:scale-[1.02] active:scale-[0.98] transition-all font-black px-8 py-3 rounded-md text-[11px] shadow-xl uppercase tracking-[0.2em] ring-offset-2 ring-offset-background focus:ring-2 focus:ring-foreground"
                >
                    Apply Changes
                </button>
            </div>
        </div>
    );
}
