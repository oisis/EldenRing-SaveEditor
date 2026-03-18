import {useState} from 'react';
import './App.css';
import {OpenSaveFile, GetCharacters, GetCharacterDetails, SaveCharacterDetails} from "../wailsjs/go/backend/App";
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
    const [error, setError] = useState("");

    async function handleOpenFile() {
        try {
            const path = await OpenSaveFile();
            if (path) {
                setFilePath(path);
                refreshCharacters();
                setError("");
            }
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
                {!editingChar && <button className="btn" onClick={handleOpenFile}>Open Save File</button>}
                {editingChar && <button className="btn btn-back" onClick={() => setEditingChar(null)}>Back to List</button>}
                {filePath && !editingChar && <p className="file-path">Loaded: {filePath}</p>}
                {error && <p className="error">{error}</p>}
            </header>

            <main className="main">
                {!editingChar ? (
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
                        <h2>Editing: {editingChar.name}</h2>
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
            </main>
        </div>
    )
}

export default App
