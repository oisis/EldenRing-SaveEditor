import {useState} from 'react';
import {ImportSlot} from '../../wailsjs/go/main/App';

interface Props {
    destSlot: number;
    onComplete: () => void;
}

export function CharacterImporter({destSlot, onComplete}: Props) {
    const [sourcePath, setSourcePath] = useState('tmp/save/ER0000.sl2'); // Default for testing
    const [sourceSlot, setSourceSlot] = useState(0);
    const [importing, setImporting] = useState(false);

    const handleImport = async () => {
        if (!sourcePath) return alert('Please select a source save file.');
        
        setImporting(true);
        try {
            await ImportSlot(sourcePath, sourceSlot, destSlot);
            alert('Character imported successfully into memory. Remember to save the file to apply changes to disk.');
            onComplete();
        } catch (e) {
            console.error(e);
            alert('Failed to import character. Check console for details.');
        } finally {
            setImporting(false);
        }
    };

    return (
        <div className="bg-er-gray p-8 rounded-lg border border-er-gold/20 shadow-2xl space-y-8 max-w-2xl mx-auto animate-in zoom-in-95 duration-300">
            <div className="text-center space-y-2">
                <h2 className="text-3xl font-serif text-er-gold">Character Importer</h2>
                <p className="text-gray-500 text-sm">Transfer a character slot from an external save file.</p>
            </div>
            
            <div className="grid grid-cols-1 gap-6">
                <div className="space-y-3">
                    <label className="block text-xs font-bold text-gray-400 uppercase tracking-widest">Source Save File</label>
                    <div className="flex space-x-2">
                        <input 
                            type="text" 
                            value={sourcePath}
                            onChange={e => setSourcePath(e.target.value)}
                            placeholder="Path to .sl2 or decrypted PS4 save"
                            className="flex-1 bg-er-dark border border-gray-700 rounded px-4 py-2.5 text-sm outline-none focus:border-er-gold transition-all"
                        />
                    </div>
                    <p className="text-[10px] text-gray-600 italic">Example: /Users/name/AppData/Roaming/EldenRing/ID/ER0000.sl2</p>
                </div>
                
                <div className="space-y-3">
                    <label className="block text-xs font-bold text-gray-400 uppercase tracking-widest">Select Source Slot</label>
                    <div className="grid grid-cols-5 gap-2">
                        {[...Array(10)].map((_, i) => (
                            <button
                                key={i}
                                onClick={() => setSourceSlot(i)}
                                className={`py-2 rounded text-xs font-bold border transition-all ${
                                    sourceSlot === i 
                                    ? 'bg-er-gold text-er-dark border-er-gold' 
                                    : 'bg-er-dark text-gray-500 border-gray-700 hover:border-gray-500'
                                }`}
                            >
                                Slot {i + 1}
                            </button>
                        ))}
                    </div>
                </div>
            </div>

            <div className="bg-black/20 p-4 rounded border border-gray-800 space-y-2">
                <div className="text-[10px] text-gray-500 uppercase font-bold">Target Destination</div>
                <div className="text-sm text-er-gold font-serif">Currently editing Slot {destSlot + 1}</div>
                <p className="text-[10px] text-gray-600">The character in this slot will be completely overwritten by the source character.</p>
            </div>

            <button 
                onClick={handleImport}
                disabled={importing}
                className={`w-full py-4 rounded font-bold uppercase tracking-widest shadow-xl transition-all transform active:scale-95 ${
                    importing 
                    ? 'bg-gray-700 text-gray-500 cursor-not-allowed' 
                    : 'bg-er-gold hover:bg-yellow-600 text-er-dark'
                }`}
            >
                {importing ? 'Importing...' : 'Confirm & Import Character'}
            </button>
        </div>
    );
}
