import {useState} from 'react';
import './App.css';
import {OpenSaveFile, GetCharacters, GetCharacterDetails, SaveCharacterDetails, GetSteamID, SaveSteamID, GetGraces, GetBosses, SetEventFlag, AddBulkItems, ImportCharacter} from "../wailsjs/go/backend/App";
import {backend} from "../wailsjs/go/models";
import {translations, Language} from "./i18n";

interface CharacterInfo {
    slotIndex: number;
    name: string;
    level: number;
    isActive: boolean;
}

function App() {
    const [lang, setLang] = useState<Language>("en");
    const t = translations[lang];

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

    async function handleImport(slotIndex: number) {
        try {
            await ImportCharacter(slotIndex);
            refreshCharacters();
            alert(t.importSuccess);
        } catch (err) {
            setError(String(err));
        }
    }

    async function handleSaveSteamID() {
        try {
            await SaveSteamID(BigInt(steamID) as any);
            alert(t.saveSuccess);
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
            alert(`Added ${count} items!`);
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
            alert(t.saveSuccess);
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
                <div className="header-top">
                    <h1>{t.title}</h1>
                    <div className="lang-switch">
                        <button className={`lang-btn ${lang === 'en' ? 'active' : ''}`} onClick={() => setLang('en')}>EN</button>
                        <button className={`lang-btn ${lang === 'pl' ? 'active' : ''}`} onClick={() => setLang('pl')}>PL</button>
                    </div>
                </div>
                {!editingChar && !isSettingsOpen && (
                    <div className="header-actions">
                        <button className="btn" onClick={handleOpenFile}>{t.openFile}</button>
                        {filePath && <button className="btn btn-secondary" onClick={() => setIsSettingsOpen(true)}>{t.settings}</button>}
                    </div>
                )}
                {(editingChar || isSettingsOpen) && (
                    <button className="btn btn-back" onClick={() => {setEditingChar(null); setIsSettingsOpen(false);}}>{t.back}</button>
                )}
                {filePath && !editingChar && !isSettingsOpen && <p className="file-path">{t.loaded}: {filePath}</p>}
                {error && <p className="error">{error}</p>}
            </header>

            <main className="main">
                {isSettingsOpen ? (
                    <div className="settings-view">
                        <h2>{t.steamIdTitle}</h2>
                        <div className="stat-item">
                            <label>{t.steamIdLabel}</label>
                            <input type="text" value={steamID} onChange={(e) => setSteamID(e.target.value)} />
                            <p className="help-text">{t.steamIdHelp}</p>
                        </div>
                        <div className="edit-actions">
                            <button className="btn btn-save" onClick={handleSaveSteamID}>{t.saveSteamId}</button>
                        </div>
                    </div>
                ) : !editingChar ? (
                    <div className="character-list">
                        {characters.map((char) => (
                            <div key={char.slotIndex} className={`character-card ${char.isActive ? 'active' : 'empty'}`}>
                                <div className="slot-id">Slot {char.slotIndex}</div>
                                <div className="char-info">
                                    <h3>{char.isActive ? char.name : t.emptySlot}</h3>
                                    {char.isActive && <p>{t.level}: {char.level}</p>}
                                </div>
                                {char.isActive ? (
                                    <button className="btn-edit" onClick={() => handleEdit(char.slotIndex)}>{t.edit}</button>
                                ) : (
                                    <button className="btn-import" onClick={() => handleImport(char.slotIndex)}>{t.import}</button>
                                )}
                            </div>
                        ))}
                    </div>
                ) : (
                    <div className="edit-view">
                        <div className="edit-header">
                            <h2>{editingChar.name}</h2>
                            <div className="tabs">
                                <button className={`tab-btn ${activeTab === 'stats' ? 'active' : ''}`} onClick={() => setActiveTab('stats')}>{t.stats}</button>
                                <button className={`tab-btn ${activeTab === 'inventory' ? 'active' : ''}`} onClick={() => setActiveTab('inventory')}>{t.inventory}</button>
                                <button className={`tab-btn ${activeTab === 'graces' ? 'active' : ''}`} onClick={() => setActiveTab('graces')}>{t.graces}</button>
                                <button className={`tab-btn ${activeTab === 'bosses' ? 'active' : ''}`} onClick={() => setActiveTab('bosses')}>{t.bosses}</button>
                            </div>
                        </div>

                        {activeTab === 'stats' && (
                            <div className="tab-content">
                                <div className="stats-grid">
                                    <div className="stat-item">
                                        <label>{t.souls}</label>
                                        <input type="number" value={editingChar.souls} onChange={(e) => updateStat('souls', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>{t.vigor}</label>
                                        <input type="number" value={editingChar.vigor} onChange={(e) => updateStat('vigor', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>{t.mind}</label>
                                        <input type="number" value={editingChar.mind} onChange={(e) => updateStat('mind', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>{t.endurance}</label>
                                        <input type="number" value={editingChar.endurance} onChange={(e) => updateStat('endurance', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>{t.strength}</label>
                                        <input type="number" value={editingChar.strength} onChange={(e) => updateStat('strength', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>{t.dexterity}</label>
                                        <input type="number" value={editingChar.dexterity} onChange={(e) => updateStat('dexterity', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>{t.intelligence}</label>
                                        <input type="number" value={editingChar.intelligence} onChange={(e) => updateStat('intelligence', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>{t.faith}</label>
                                        <input type="number" value={editingChar.faith} onChange={(e) => updateStat('faith', e.target.value)} />
                                    </div>
                                    <div className="stat-item">
                                        <label>{t.arcane}</label>
                                        <input type="number" value={editingChar.arcane} onChange={(e) => updateStat('arcane', e.target.value)} />
                                    </div>
                                </div>
                                <div className="edit-actions">
                                    <button className="btn btn-save" onClick={handleSave}>{t.saveChanges}</button>
                                </div>
                            </div>
                        )}

                        {activeTab === 'inventory' && (
                            <div className="tab-content inventory-view">
                                <h3>{t.bulkAddTitle}</h3>
                                <p className="help-text">{t.bulkAddHelp}</p>
                                <div className="bulk-actions">
                                    <button className="btn btn-secondary" onClick={() => handleBulkAdd('Talismans')}>{t.addTalismans}</button>
                                    <button className="btn btn-secondary" onClick={() => handleBulkAdd('Weapons')}>{t.addWeapons}</button>
                                    <button className="btn btn-secondary" onClick={() => handleBulkAdd('Armors')}>{t.addArmors}</button>
                                    <button className="btn btn-secondary" onClick={() => handleBulkAdd('Items')}>{t.addConsumables}</button>
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
