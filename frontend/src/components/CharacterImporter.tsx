import {useState} from 'react';
import {SelectAndOpenSourceSave, GetSourceActiveSlots, ImportCharacter} from '../../wailsjs/go/main/App';

interface Props {
    destSlot: number;
    onComplete: () => void;
}

export function CharacterImporter({destSlot, onComplete}: Props) {
    const [sourceLoaded, setSourceLoaded] = useState(false);
    const [sourceSlots, setSourceSlots] = useState<boolean[]>(new Array(10).fill(false));
    const [selectedSourceSlot, setSelectedSourceSlot] = useState<number | null>(null);
    const [loading, setLoading] = useState(false);

    const handleOpenSource = async () => {
        try {
            const res = await SelectAndOpenSourceSave();
            if (res) {
                const slots = await GetSourceActiveSlots();
                setSourceSlots(slots || new Array(10).fill(false));
                setSourceLoaded(true);
            }
        } catch (e) {
            alert("Error: " + e);
        }
    };

    const handleImport = async () => {
        if (selectedSourceSlot === null) return;
        setLoading(true);
        try {
            await ImportCharacter(selectedSourceSlot, destSlot);
            alert("Character imported successfully!");
            onComplete();
        } catch (e) {
            alert("Import failed: " + e);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="space-y-8 animate-in fade-in duration-500">
            <div className="bg-muted/30 border border-border rounded-lg p-10 text-center max-w-2xl mx-auto relative overflow-hidden">
                <div className="absolute top-0 right-0 w-32 h-32 bg-blue-500/5 rounded-full -mr-16 -mt-16" />
                
                <div className="relative space-y-6">
                    <div className="w-12 h-12 bg-background border border-border rounded flex items-center justify-center mx-auto shadow-sm">
                        <svg className="w-6 h-6 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
                        </svg>
                    </div>
                    
                    <div className="space-y-2">
                        <h3 className="text-lg font-semibold tracking-tight">Character Importer</h3>
                        <p className="text-sm text-muted-foreground leading-relaxed max-w-sm mx-auto">
                            Transfer a character profile from an external save file into your current session.
                        </p>
                    </div>
                    
                    {!sourceLoaded ? (
                        <div className="pt-4">
                            <button 
                                onClick={handleOpenSource}
                                className="bg-foreground text-background hover:opacity-90 transition-opacity font-semibold px-8 py-2.5 rounded text-xs uppercase tracking-widest shadow-sm"
                            >
                                Select Source File
                            </button>
                        </div>
                    ) : (
                        <div className="space-y-8 pt-4 animate-in slide-in-from-bottom-2 duration-300">
                            <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
                                {sourceSlots.map((isActive, i) => (
                                    <button
                                        key={i}
                                        disabled={!isActive}
                                        onClick={() => setSelectedSourceSlot(i)}
                                        className={`
                                            p-4 rounded border transition-all flex flex-col items-center space-y-2 relative
                                            ${!isActive ? 'opacity-30 grayscale cursor-not-allowed border-border bg-muted/50' : 
                                              selectedSourceSlot === i ? 'border-blue-500 bg-blue-500/5 ring-1 ring-blue-500' : 
                                              'border-border bg-background hover:border-zinc-400 dark:hover:border-zinc-500'}
                                        `}
                                    >
                                        <span className={`text-[10px] font-bold uppercase tracking-wider ${selectedSourceSlot === i ? 'text-blue-600 dark:text-blue-400' : 'text-muted-foreground'}`}>Slot {i + 1}</span>
                                        <div className={`w-1.5 h-1.5 rounded-full ${isActive ? 'bg-green-500' : 'bg-zinc-300 dark:bg-zinc-700'}`} />
                                        {selectedSourceSlot === i && (
                                            <div className="absolute top-1 right-1">
                                                <svg className="w-3 h-3 text-blue-500" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd"></path></svg>
                                            </div>
                                        )}
                                    </button>
                                ))}
                            </div>

                            <div className="flex items-center justify-center space-x-4 pt-2">
                                <button 
                                    onClick={() => setSourceLoaded(false)}
                                    className="text-muted-foreground hover:text-foreground text-xs font-semibold uppercase tracking-widest transition-colors px-4 py-2"
                                >
                                    Cancel
                                </button>
                                <button 
                                    disabled={selectedSourceSlot === null || loading}
                                    onClick={handleImport}
                                    className={`
                                        bg-foreground text-background hover:opacity-90 transition-opacity font-semibold px-8 py-2.5 rounded text-xs uppercase tracking-widest shadow-sm
                                        ${(selectedSourceSlot === null || loading) ? 'opacity-50 cursor-not-allowed' : ''}
                                    `}
                                >
                                    {loading ? 'Importing...' : `Import to Slot ${destSlot + 1}`}
                                </button>
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
