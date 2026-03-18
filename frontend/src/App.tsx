import {useState} from 'react';
import './App.css';
import {OpenSaveFile, GetCharacters, GetCharacterDetails, SaveCharacterDetails, GetSteamID, SaveSteamID, GetGraces, GetBosses, SetEventFlag, AddBulkItems} from "../wailsjs/go/backend/App";
import {backend} from "../wailsjs/go/models";

interface CharacterInfo {
    slotIndex: number;
    name: string;
    level: number;
    isActive: boolean;
}

function App() {
    const [filePath, setFilePath] = useState("");
    const [characters, setCharacters] = useState<CharacterInfo[]>([]);
    const [editingChar, setEditingChar] = useState<backend.CharacterDetails | null>(null);
    const [graces, setGraces] = useState<backend.EventItem[]>([]);
    const [bosses, setBosses] = useState<backend.EventItem[]>([]);
    const [activeTab, setActiveTab] = useState<"stats" | "graces" | "bosses" | "inventory">("stats");
    const [steamID, setSteamID] = useState<string>("0");
    const [isSettingsOpen, setIsSettingsOpen] = useState(false);
    const [error, setError] = useState("");

    async function handleOpenFile() {
        try {
            const path = await OpenSaveFile();
            if (path) {
                setFilePath(path);
                refreshCharacters();
                const sid = await GetSteamID();
                setSteamID(sid.toString());
                setError("");
            }
        } catch (err) {
            setError(String(err));
        }
    }

    async function handleSaveSteamID() {
        try {
            await SaveSteamID(BigInt(steamID) as any);
            alert("SteamID saved successfully!");
            setIsSettingsOpen(false);
        } catch (err) {
            setError(String(err));
        }
    }

    async function refreshCharacters() {
        const chars = await GetCharacters();
        setCharacters(chars);
    }

    async function handleEdit(slotIndex: number) {
        try {
            const details = await GetCharacterDetails(slotIndex);
            setEditingChar(details);
            const g = await GetGraces(slotIndex);
            setGraces(g);
            const b = await GetBosses(slotIndex);
            setBosses(b);
            setActiveTab("stats");
            setIsSettingsOpen(false);
        } catch (err) {
            setError(String(err));
        }
    }

    async function handleToggleEvent(flagID: number, enabled: boolean) {
        if (!editingChar) return;
        try {
            await SetEventFlag(editingChar.slotIndex, flagID, enabled);
            if (activeTab === "graces") {
                setGraces(graces.map(g => g.id === flagID ? {...g, enabled} : g));
            } else {
                setBosses(bosses.map(b => b.id === flagID ? {...b, enabled} : b));
            }
        } catch (err) {
            setError(String(err));
        }
    }

    async function handleBulkAdd(category: string) {
        if (!editingChar) return;
        try {
            const count = await AddBulkItems(editingChar.slotIndex, category);
            alert(`Added ${count} items from ${category} category!`);
        } catch (err) {
            setError(String(err));
        }
    }

    async function handleSave() {
        if (!editingChar) return;
        try {
            await SaveCharacterDetails(editingChar);
            setEditingChar(null);
            refreshCharacters();
            setError("");
            alert("Save successful!");
        } catch (err) {
            setError(String(err));
        }
    }

    const updateStat = (stat: keyof backend.CharacterDetails, value: string) => {
        if (!editingChar) return;
        setEditingChar({
            ...editingChar,
            [stat]: parseInt(value) || 0
        });
    };

    return (
        <div id="App">
            <header className="header">
                <h1>ER Save Editor</h1>
                {!editingChar && !isSettingsOpen && (
                    <div className="header-actions">
                        <button className="btn" onClick={handleOpenFile}>Open Save File</button>
                        {filePath && <button className="btn btn-secondary" onClick={() => setIsSettingsOpen(true)}>Settings</button>}
                    </div>
                )}
                {(editingChar || isSettingsOpen) && (
                    <button className="btn btn-back" onClick={() => {setEditingChar(null); setIsSettingsOpen(false);}}>Back to List</button>
                )}
                {filePath && !editingChar && !isSettingsOpen && <p className="file-path">Loaded: {filePath}</p>}
                {error && <p className="error">{error}</p>}
            </header>

            <main className="main">
                {isSettingsOpen ? (
                    <div className="settings-view">
                        <h2>Account Settings</h2>
                        <div className="stat-item">
                            <label>SteamID (64-bit)</label>
                            <input type="text" value={steamID} onChange={(e) => setSteamID(e.target.value)} />
                            <p className="help-text">Change this to match your Steam Account ID to use this save on another account.</p>
                        </div>
                        <div className="edit-actions">
                            <button className="btn btn-save" onClick={handleSaveSteamID}>Save SteamID</button>
                        </div>
                    </div>
                ) : !editingChar ? (
                    <div className="character-list">
                        {characters.map((char) => (
                            <div key={char.slotIndex} className={`character-card ${char.isActive ? 'active' : 'empty'}`}>
                                <div className="slot-id">Slot {char.slotIndex}</div>
                                <div className="char-info">
                                    <h3>{char.isActive ? char.name : "Empty Slot"}</h3>
                                    {char.isActive && <p>Level: {char.level}</p>}
                                </div>
                                {char.isActive && <button className="btn-edit" onClick={() => handleEdit(char.slotIndex)}>Edit</button>}
                            </div>
                        ))}
                    </div>
                ) : (
                    <div className="edit-view">
                        <div className="edit-header">
                            <h2>Editing: {editingChar.name}</h2>
                            <div className="tabs">
                                <button className={`tab-btn ${activeTab === 'stats' ? 'active' : ''}`} onClick={() => setActiveTab('stats')}>Stats</button>
                                <button className={`tab-btn ${activeTab === 'inventory' ? 'active' : ''}`} onClick={() => setActiveTab('inventory')}>Inventory</button>
                                <button className={`tab-btn ${activeTab === 'graces' ? 'active' : ''}`} onClick={() => setActiveTab('graces')}>Graces</button>
                                <button className={`tab-btn ${activeTab === 'bosses' ? 'active' : ''}`} onClick={() => setActiveTab('bosses')}>Bosses</button>
                            </div>
                        </div>

                        {activeTab === 'stats' && (
                            <div className="tab-content">
                                <div className="stats-grid">
                                    <div className="stat-item">
                                        <label>Souls</label>
                                        <input type="number" value={editingChar.souls} onChange={(e) => updateStat('souls', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>Vigor</label>
                                        <input type="number" value={editingChar.vigor} onChange={(e) => updateStat('vigor', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>Mind</label>
                                        <input type="number" value={editingChar.mind} onChange={(e) => updateStat('mind', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>Endurance</label>
                                        <input type="number" value={editingChar.endurance} onChange={(e) => updateStat('endurance', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>Strength</label>
                                        <input type="number" value={editingChar.strength} onChange={(e) => updateStat('strength', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>Dexterity</label>
                                        <input type="number" value={editingChar.dexterity} onChange={(e) => updateStat('dexterity', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>Intelligence</label>
                                        <input type="number" value={editingChar.intelligence} onChange={(e) => updateStat('intelligence', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>Faith</label>
                                        <input type="number" value={editingChar.faith} onChange={(e) => updateStat('faith', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>Arcane</label>
                                        <input type="number" value={editingChar.arcane} onChange={(e) => updateStat('arcane', e.target.value)} />
                                    </div>
                                </div>
                                <div className="edit-actions">
                                    <button className="btn btn-save" onClick={handleSave}>Save Changes</button>
                                </div>
                            </div>
                        )}

                        {activeTab === 'inventory' && (
                            <div className="tab-content inventory-view">
                                <h3>Bulk Add Items</h3>
                                <p className="help-text">Add all items from a category that you don't already have.</p>
                                <div className="bulk-actions">
                                    <button className="btn btn-secondary" onClick={() => handleBulkAdd('Talismans')}>Add All Talismans</button>
                                    <button className="btn btn-secondary" onClick={() => handleBulkAdd('Weapons')}>Add All Weapons</button>
                                    <button className="btn btn-secondary" onClick={() => handleBulkAdd('Armors')}>Add All Armors</button>
                                    <button className="btn btn-secondary" onClick={() => handleBulkAdd('Items')}>Add All Consumables</button>
                                </div>
                            </div>
                        )}

                        {activeTab === 'graces' && (
                            <div className="tab-content event-list">
                                {graces.map(g => (
                                    <div key={g.id} className="event-item">
                                        <input type="checkbox" checked={g.enabled} onChange={(e) => handleToggleEvent(g.id, e.target.checked)} />
                                        <span>{g.name}</span>
                                    </div>
                                ))}
                            </div>
                        )}

                        {activeTab === 'bosses' && (
                            <div className="tab-content event-list">
                                {bosses.map(b => (
                                    <div key={b.id} className="event-item">
                                        <input type="checkbox" checked={b.enabled} onChange={(e) => handleToggleEvent(b.id, e.target.checked)} />
                                        <span>{b.name}</span>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                )}
            </main>
        </div>
    )
}

export default App
