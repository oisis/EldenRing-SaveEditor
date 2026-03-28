import {useState} from 'react';
import {OpenSave} from '../wailsjs/go/main/App';
import {GeneralTab} from './components/GeneralTab';
import {InventoryTab} from './components/InventoryTab';
import './App.css';

function App() {
    const [selectedChar, setSelectedChar] = useState(0);
    const [activeTab, setActiveTab] = useState('general');
    const [platform, setPlatform] = useState<string | null>(null);
    const [isLoaded, setIsLoaded] = useState(false);

    const tabs = ['General', 'Stats', 'Equipment', 'Inventory', 'World Progress'];

    const handleOpen = async () => {
        try {
            const res = await OpenSave("tmp/save/ER0000.sl2"); 
            setPlatform(res);
            setIsLoaded(true);
        } catch (e) {
            console.error(e);
            alert("Failed to open save file. Check console for details.");
        }
    };

    return (
        <div id="App" className="flex h-screen bg-er-dark text-gray-200 font-sans select-none">
            {/* Sidebar - Character List */}
            <div className="w-72 bg-er-gray border-r border-gray-800 flex flex-col shadow-2xl z-10">
                <div className="p-6 border-b border-gray-800">
                    <div className="text-er-gold font-serif text-xl tracking-tighter mb-1">ELDEN RING</div>
                    <div className="text-gray-500 uppercase text-[10px] font-bold tracking-[0.2em]">Save Editor</div>
                </div>
                
                <div className="p-4 bg-black/20">
                    <div className="text-gray-500 uppercase text-[10px] font-bold mb-3 px-2">Characters</div>
                    <div className="space-y-1 overflow-y-auto max-h-[calc(100vh-250px)] custom-scrollbar">
                        {[...Array(10)].map((_, i) => (
                            <button
                                key={i}
                                onClick={() => setSelectedChar(i)}
                                disabled={!isLoaded}
                                className={`w-full text-left px-4 py-3 rounded transition-all text-sm flex items-center space-x-3 group ${
                                    !isLoaded ? 'opacity-30 cursor-not-allowed' : 
                                    selectedChar === i ? 'bg-er-gold/10 text-er-gold border border-er-gold/30' : 'text-gray-400 hover:bg-white/5 hover:text-gray-200'
                                }`}
                            >
                                <span className={`w-1.5 h-1.5 rounded-full ${selectedChar === i ? 'bg-er-gold shadow-[0_0_8px_#c1a35f]' : 'bg-gray-700 group-hover:bg-gray-500'}`}></span>
                                <span>Slot {i + 1}</span>
                            </button>
                        ))}
                    </div>
                </div>
                
                <div className="mt-auto p-6 border-t border-gray-800 space-y-4">
                    <button 
                        onClick={handleOpen}
                        className="w-full bg-er-gold hover:bg-yellow-600 text-er-dark font-bold py-2.5 rounded shadow-lg transition-all active:scale-95 text-sm"
                    >
                        {isLoaded ? 'Change Save' : 'Open Save File'}
                    </button>
                    <div className="flex justify-between items-center text-[10px] text-gray-600 font-bold uppercase tracking-widest">
                        <span>v0.1.0 Alpha</span>
                        <span className={platform ? 'text-er-gold' : ''}>{platform || 'No File'}</span>
                    </div>
                </div>
            </div>

            {/* Main Content */}
            <div className="flex-1 flex flex-col bg-[#0f0f0f] relative overflow-hidden">
                {/* Decorative Background Element */}
                <div className="absolute top-0 right-0 w-96 h-96 bg-er-gold/5 blur-[120px] rounded-full -mr-48 -mt-48"></div>

                {/* Navbar - Tabs */}
                <div className="h-16 bg-er-gray/50 backdrop-blur-md border-b border-gray-800 flex items-center px-8 space-x-10 z-10">
                    {tabs.map(tab => (
                        <button
                            key={tab}
                            disabled={!isLoaded}
                            onClick={() => setActiveTab(tab.toLowerCase())}
                            className={`text-xs font-bold uppercase tracking-[0.2em] transition-all relative py-5 ${
                                !isLoaded ? 'opacity-20 cursor-not-allowed' :
                                activeTab === tab.toLowerCase() ? 'text-er-gold' : 'text-gray-500 hover:text-gray-300'
                            }`}
                        >
                            {tab}
                            {activeTab === tab.toLowerCase() && (
                                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-er-gold shadow-[0_0_10px_#c1a35f]"></div>
                            )}
                        </button>
                    ))}
                </div>

                {/* Content Area */}
                <div className="flex-1 p-10 overflow-y-auto relative z-10">
                    {!isLoaded ? (
                        <div className="h-full flex flex-col items-center justify-center text-center space-y-6">
                            <div className="w-24 h-24 border-2 border-er-gold/20 rounded-full flex items-center justify-center animate-[spin_10s_linear_infinite]">
                                <div className="w-16 h-16 border-2 border-er-gold/40 rounded-full flex items-center justify-center animate-[spin_5s_linear_infinite] direction-reverse">
                                    <div className="w-8 h-8 bg-er-gold/60 rounded-full animate-pulse"></div>
                                </div>
                            </div>
                            <div className="space-y-2">
                                <h2 className="text-er-gold font-serif text-2xl">No Save File Loaded</h2>
                                <p className="text-gray-500 text-sm max-w-xs">Please select an Elden Ring save file (.sl2 or decrypted PS4) to begin editing.</p>
                            </div>
                        </div>
                    ) : (
                        <div className="max-w-5xl animate-in fade-in slide-in-from-bottom-4 duration-700">
                            <header className="mb-10">
                                <div className="text-er-gold uppercase text-[10px] font-bold tracking-[0.3em] mb-2">Character Slot {selectedChar + 1}</div>
                                <h1 className="text-4xl font-serif text-white capitalize">{activeTab}</h1>
                            </header>
                            
                            {activeTab === 'general' && <GeneralTab charIndex={selectedChar} />}
                            {activeTab === 'inventory' && <InventoryTab />}
                            
                            {['stats', 'equipment', 'world progress'].includes(activeTab) && (
                                <div className="bg-er-gray/50 p-12 rounded-lg border border-gray-800 text-center space-y-4">
                                    <div className="text-er-gold/40 text-5xl font-serif">Coming Soon</div>
                                    <p className="text-gray-500 text-sm">The {activeTab} editor is currently under development.</p>
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}

export default App
