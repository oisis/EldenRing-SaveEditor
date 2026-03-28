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
                .then(() => console.log('Character data updated in memory'))
                .catch(console.error);
        }
    };

    if (loading) return <div className="text-er-gold italic animate-pulse">Loading character data...</div>;
    if (!char) return <div className="text-gray-500 italic">No character data available. Please open a save file.</div>;

    return (
        <div className="space-y-8 animate-in fade-in duration-500">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                {/* Identity Section */}
                <div className="bg-er-gray p-6 rounded-lg border border-gray-700 shadow-lg space-y-4">
                    <h2 className="text-lg font-serif text-er-gold border-b border-gray-700 pb-2 mb-4">Identity</h2>
                    
                    <div className="space-y-2">
                        <label className="block text-xs font-bold text-gray-500 uppercase tracking-wider">Character Name</label>
                        <input 
                            type="text" 
                            value={char.name} 
                            onChange={e => setChar({...char, name: e.target.value})}
                            className="w-full bg-er-dark border border-gray-700 rounded px-4 py-2.5 text-sm focus:border-er-gold outline-none transition-all"
                            maxLength={16}
                        />
                    </div>

                    <div className="space-y-2">
                        <label className="block text-xs font-bold text-gray-500 uppercase tracking-wider">Runes (Souls)</label>
                        <input 
                            type="number" 
                            value={char.souls} 
                            onChange={e => setChar({...char, souls: parseInt(e.target.value) || 0})}
                            className="w-full bg-er-dark border border-gray-700 rounded px-4 py-2.5 text-sm focus:border-er-gold outline-none transition-all"
                        />
                    </div>
                </div>

                {/* Level Summary Section */}
                <div className="bg-er-gray p-6 rounded-lg border border-gray-700 shadow-lg flex flex-col justify-center items-center space-y-2">
                    <div className="text-gray-500 uppercase text-xs font-bold tracking-widest">Current Level</div>
                    <div className="text-6xl font-serif text-er-gold">{char.level}</div>
                    <div className="text-gray-400 text-xs italic mt-4 text-center">
                        Level is automatically calculated based on your attributes in the Stats tab.
                    </div>
                </div>
            </div>

            <div className="flex justify-end">
                <button 
                    onClick={handleSave}
                    className="bg-er-gold hover:bg-yellow-600 text-er-dark font-bold px-8 py-3 rounded shadow-lg transition-all transform active:scale-95"
                >
                    Apply Changes to Slot
                </button>
            </div>
        </div>
    );
}
