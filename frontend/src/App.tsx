import {useState} from 'react';
import './App.css';

function App() {
    const [selectedChar, setSelectedChar] = useState(0);
    const [activeTab, setActiveTab] = useState('general');

    const tabs = ['General', 'Stats', 'Equipment', 'Inventory', 'World Progress'];

    return (
        <div id="App" className="flex h-screen bg-er-dark text-gray-200 font-sans">
            {/* Sidebar - Character List */}
            <div className="w-64 bg-er-gray border-r border-gray-700 flex flex-col">
                <div className="p-4 border-b border-gray-700 font-bold text-er-gold uppercase tracking-wider text-sm">
                    Characters
                </div>
                <div className="flex-1 overflow-y-auto">
                    {[...Array(10)].map((_, i) => (
                        <button
                            key={i}
                            onClick={() => setSelectedChar(i)}
                            className={`w-full text-left p-3 hover:bg-gray-700 transition-colors text-sm ${selectedChar === i ? 'bg-gray-700 border-l-4 border-er-gold text-white' : 'text-gray-400'}`}
                        >
                            Character {i + 1}
                        </button>
                    ))}
                </div>
                <div className="p-4 border-t border-gray-700 text-xs text-gray-500">
                    ER Save Editor v0.1.0
                </div>
            </div>

            {/* Main Content */}
            <div className="flex-1 flex flex-col">
                {/* Header / Toolbar */}
                <div className="h-14 bg-er-gray border-b border-gray-700 flex items-center justify-between px-6">
                    <div className="flex space-x-4">
                        <button className="bg-gray-700 hover:bg-gray-600 px-4 py-1.5 rounded text-sm font-medium transition-colors border border-gray-600">
                            Open Save
                        </button>
                        <button className="bg-er-gold/20 hover:bg-er-gold/30 text-er-gold px-4 py-1.5 rounded text-sm font-medium transition-colors border border-er-gold/50">
                            Save Changes
                        </button>
                    </div>
                    <div className="text-xs font-mono text-gray-500">
                        Platform: <span className="text-gray-300">PC (Detected)</span>
                    </div>
                </div>

                {/* Navbar - Tabs */}
                <div className="h-12 bg-er-dark border-b border-gray-800 flex items-center px-6 space-x-8">
                    {tabs.map(tab => (
                        <button
                            key={tab}
                            onClick={() => setActiveTab(tab.toLowerCase())}
                            className={`text-sm font-medium transition-all relative py-3 ${activeTab === tab.toLowerCase() ? 'text-er-gold' : 'text-gray-500 hover:text-gray-300'}`}
                        >
                            {tab}
                            {activeTab === tab.toLowerCase() && (
                                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-er-gold"></div>
                            )}
                        </button>
                    ))}
                </div>

                {/* Content Area */}
                <div className="flex-1 p-8 overflow-y-auto bg-[#121212]">
                    <div className="max-w-4xl">
                        <h1 className="text-2xl font-serif text-er-gold mb-6 capitalize">{activeTab} Settings</h1>
                        <div className="bg-er-gray p-6 rounded-lg border border-gray-700 shadow-xl">
                            <p className="text-gray-400 italic">
                                Editing character {selectedChar + 1}. Data binding with Go backend in progress...
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default App
