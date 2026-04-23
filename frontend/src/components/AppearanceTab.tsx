import {useState, useEffect} from 'react';
import toast from 'react-hot-toast';
import {ListAppearancePresets, ApplyAppearancePreset} from '../../wailsjs/go/main/App';
import {main} from '../../wailsjs/go/models';

interface Props {
    charIndex: number;
    onMutate: () => void;
}

export function AppearanceTab({charIndex, onMutate}: Props) {
    const [presets, setPresets] = useState<main.PresetInfo[]>([]);
    const [selected, setSelected] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const [confirmName, setConfirmName] = useState<string | null>(null);

    useEffect(() => {
        ListAppearancePresets().then(setPresets).catch(e => toast.error("" + e));
    }, []);

    const handleApply = async (name: string) => {
        setLoading(true);
        try {
            await ApplyAppearancePreset(charIndex, name);
            toast.success(`Applied "${name}" appearance (face shape, body, skin)`);
            setConfirmName(null);
            setSelected(name);
            onMutate();
        } catch (e) {
            toast.error("Failed: " + e);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="space-y-6 p-4 animate-in fade-in slide-in-from-bottom-4 duration-700">
            <div className="flex items-center space-x-3">
                <div className="w-1 h-5 bg-primary rounded-full" />
                <h3 className="text-sm font-black uppercase tracking-[0.15em]">Appearance Presets</h3>
                <span className="text-[9px] text-muted-foreground font-medium uppercase tracking-wider">
                    {presets.length} available
                </span>
            </div>

            <div className="card p-4 space-y-2">
                <p className="text-[10px] text-muted-foreground leading-relaxed">
                    Applies face shape, body proportions, and skin/cosmetics from the preset.
                    Hair style, beard, and bone structure are kept from the current character.
                    Use Undo (Ctrl+Z) to revert.
                </p>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
                {presets.map(p => (
                    <div
                        key={p.name}
                        className={`
                            group relative rounded-lg border overflow-hidden transition-all cursor-pointer
                            ${selected === p.name
                                ? 'border-primary ring-1 ring-primary shadow-lg shadow-primary/10'
                                : 'border-border hover:border-primary/30'}
                        `}
                        onClick={() => { setSelected(p.name); setConfirmName(null); }}
                    >
                        <div className="relative aspect-[3/4] bg-muted/30 overflow-hidden">
                            {p.image ? (
                                <img
                                    src={`presets/${p.image}`}
                                    alt={p.name}
                                    className={`w-full h-full object-cover object-top transition-all duration-500 ${
                                        selected === p.name ? 'scale-105' : 'group-hover:scale-105'
                                    }`}
                                />
                            ) : (
                                <div className="w-full h-full flex items-center justify-center">
                                    <svg className="w-10 h-10 text-muted-foreground/30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
                                    </svg>
                                </div>
                            )}
                            <div className={`absolute inset-0 transition-all ${
                                selected === p.name ? 'bg-primary/10' : 'bg-gradient-to-t from-black/60 via-transparent to-transparent'
                            }`} />
                        </div>
                        <div className={`p-2.5 text-center transition-colors ${
                            selected === p.name ? 'bg-primary/5' : 'bg-background'
                        }`}>
                            <div className={`text-[10px] font-black uppercase tracking-wider leading-tight ${
                                selected === p.name ? 'text-primary' : 'text-foreground'
                            }`}>{p.name}</div>
                            <div className="text-[8px] text-muted-foreground font-medium uppercase tracking-widest mt-0.5">{p.bodyType}</div>
                        </div>
                        {selected === p.name && (
                            <div className="absolute top-2 right-2 bg-primary rounded-full p-0.5 shadow-lg">
                                <svg className="w-3 h-3 text-primary-foreground" fill="currentColor" viewBox="0 0 20 20">
                                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                                </svg>
                            </div>
                        )}
                    </div>
                ))}
            </div>

            {selected && (
                <div className="flex items-center justify-center pt-2 animate-in slide-in-from-bottom-2 duration-300">
                    {confirmName === selected ? (
                        <div className="flex items-center space-x-4">
                            <span className="text-[10px] text-amber-500 font-bold uppercase tracking-wider">
                                Overwrite face shape, body &amp; skin?
                            </span>
                            <button onClick={() => setConfirmName(null)}
                                className="text-muted-foreground hover:text-foreground text-[10px] font-black uppercase tracking-[0.2em] transition-colors px-4 py-2">
                                Cancel
                            </button>
                            <button disabled={loading} onClick={() => handleApply(selected)}
                                className={`bg-primary text-primary-foreground hover:scale-[1.02] active:scale-[0.98] transition-all
                                    font-black px-8 py-2.5 rounded-md text-[10px] shadow-lg shadow-primary/20 uppercase tracking-[0.2em]
                                    ${loading ? 'opacity-50 cursor-not-allowed' : ''}`}>
                                {loading ? 'Applying...' : 'Confirm'}
                            </button>
                        </div>
                    ) : (
                        <button onClick={() => setConfirmName(selected)}
                            className="bg-foreground text-background hover:scale-[1.02] active:scale-[0.98] transition-all font-black px-10 py-3 rounded-md text-[11px] shadow-xl uppercase tracking-[0.2em]">
                            Apply {selected}
                        </button>
                    )}
                </div>
            )}
        </div>
    );
}
