import {useState, useEffect} from 'react';
import {SelectAndOpenSave, WriteSave, GetSteamIDString, SetSteamIDFromString} from '../../wailsjs/go/main/App';

interface SettingsTabProps {
    theme: 'light' | 'dark' | 'system';
    setTheme: (theme: 'light' | 'dark' | 'system') => void;
    columnVisibility: {
        id: boolean;
        category: boolean;
    };
    setColumnVisibility: (visibility: { id: boolean; category: boolean }) => void;
    showFlaggedItems: boolean;
    setShowFlaggedItems: (value: boolean) => void;
    platform: string | null;
    setPlatform: (platform: string | null) => void;
    refreshSlots: () => void;
}

export function SettingsTab({
    theme,
    setTheme,
    columnVisibility,
    setColumnVisibility,
    showFlaggedItems,
    setShowFlaggedItems,
    platform,
    setPlatform,
    refreshSlots
}: SettingsTabProps) {
    const [targetPlatform, setTargetPlatform] = useState<string>('PC');
    const [exporting, setExporting] = useState(false);
    const [importing, setImporting] = useState(false);

    const [steamIdInput, setSteamIdInput] = useState('');
    const [steamIdSaved, setSteamIdSaved] = useState('');
    const [steamIdError, setSteamIdError] = useState('');
    const [steamIdApplying, setSteamIdApplying] = useState(false);

    useEffect(() => {
        if (platform !== 'PC') { setSteamIdInput(''); setSteamIdSaved(''); return; }
        GetSteamIDString().then(id => { setSteamIdInput(id); setSteamIdSaved(id); });
    }, [platform]);

    const validateSteamId = (val: string) => {
        if (!/^\d{17}$/.test(val)) return 'SteamID must be exactly 17 digits.';
        if (!val.startsWith('7656119')) return 'SteamID must start with 7656119.';
        return '';
    };

    const handleApplySteamId = async () => {
        const err = validateSteamId(steamIdInput);
        if (err) { setSteamIdError(err); return; }
        setSteamIdApplying(true);
        setSteamIdError('');
        try {
            await SetSteamIDFromString(steamIdInput);
            setSteamIdSaved(steamIdInput);
        } catch (e) {
            setSteamIdError(String(e));
        } finally {
            setSteamIdApplying(false);
        }
    };

    const handleImport = async () => {
        setImporting(true);
        try {
            const plat = await SelectAndOpenSave();
            setPlatform(plat);
            refreshSlots();
            alert("Save imported successfully!");
        } catch (err) {
            alert(err);
        } finally {
            setImporting(false);
        }
    };

    const handleExport = async () => {
        setExporting(true);
        try {
            await WriteSave(targetPlatform);
            alert(`Save exported successfully as ${targetPlatform}!`);
        } catch (err) {
            alert(err);
        } finally {
            setExporting(false);
        }
    };

    return (
        <div className="space-y-12 animate-in fade-in slide-in-from-bottom-4 duration-700">
            {/* Appearance Section */}
            <section className="space-y-6">
                <div className="flex items-center space-x-4 px-1">
                    <div className="w-1.5 h-6 bg-primary rounded-full shadow-[0_0_8px_rgba(var(--primary),0.4)]" />
                    <h2 className="text-[11px] font-black uppercase tracking-[0.3em] text-foreground/80">Appearance</h2>
                </div>
                
                <div className="card p-6 space-y-6">
                    <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                        <div className="space-y-1">
                            <p className="text-xs font-bold text-foreground">Application Theme</p>
                            <p className="text-[10px] text-muted-foreground font-medium">Choose between light, dark or system default.</p>
                        </div>
                        <div className="flex bg-muted/30 p-1 rounded-lg border border-border w-full md:w-auto">
                            {(['light', 'dark', 'system'] as const).map(t => (
                                <button
                                    key={t}
                                    onClick={() => setTheme(t)}
                                    className={`px-6 py-2 rounded-md text-[10px] font-black uppercase tracking-widest transition-all ${theme === t ? 'bg-primary text-primary-foreground shadow-sm shadow-primary/20' : 'text-muted-foreground hover:text-foreground'}`}
                                >
                                    {t}
                                </button>
                            ))}
                        </div>
                    </div>
                </div>
            </section>

            {/* File Operations Section */}
            <section className="space-y-6">
                <div className="flex items-center space-x-4 px-1">
                    <div className="w-1.5 h-6 bg-primary rounded-full shadow-[0_0_8px_rgba(var(--primary),0.4)]" />
                    <h2 className="text-[11px] font-black uppercase tracking-[0.3em] text-foreground/80">File Operations</h2>
                </div>
                
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div className="card p-6 space-y-4">
                        <div className="space-y-1">
                            <p className="text-xs font-bold text-foreground">Import Save</p>
                            <p className="text-[10px] text-muted-foreground font-medium">Load an existing Elden Ring save file.</p>
                        </div>
                        <button 
                            onClick={handleImport}
                            disabled={importing}
                            className="w-full bg-muted/30 hover:bg-muted/50 text-foreground font-black py-3 rounded-lg text-[9px] uppercase tracking-[0.2em] border border-border transition-all flex items-center justify-center space-x-2"
                        >
                            {importing ? (
                                <div className="w-3 h-3 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
                            ) : (
                                <>
                                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path></svg>
                                    <span>Select File</span>
                                </>
                            )}
                        </button>
                    </div>

                    <div className="card p-6 space-y-4">
                        <div className="space-y-1">
                            <p className="text-xs font-bold text-foreground">Export Save</p>
                            <p className="text-[10px] text-muted-foreground font-medium">Save current changes to a new file.</p>
                        </div>
                        <div className="flex bg-muted/30 p-1 rounded-lg border border-border mb-2">
                            {(['PC', 'PS4'] as const).map(p => (
                                <button
                                    key={p}
                                    onClick={() => setTargetPlatform(p)}
                                    className={`flex-1 py-1.5 rounded-md text-[9px] font-black uppercase tracking-widest transition-all ${targetPlatform === p ? 'bg-background text-foreground shadow-sm ring-1 ring-border' : 'text-muted-foreground hover:text-foreground'}`}
                                >
                                    {p}
                                </button>
                            ))}
                        </div>
                        <button 
                            onClick={handleExport}
                            disabled={!platform || exporting}
                            className="w-full bg-primary text-primary-foreground font-black py-3 rounded-lg text-[9px] uppercase tracking-[0.2em] shadow-lg shadow-primary/20 hover:brightness-110 active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center space-x-2"
                        >
                            {exporting ? (
                                <div className="w-3 h-3 border-2 border-primary-foreground/20 border-t-primary-foreground rounded-full animate-spin" />
                            ) : (
                                <>
                                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path></svg>
                                    <span>Export as {targetPlatform}</span>
                                </>
                            )}
                        </button>
                    </div>
                </div>
            </section>

            {/* SteamID Section */}
            <section className="space-y-6">
                <div className="flex items-center space-x-4 px-1">
                    <div className="w-1.5 h-6 bg-primary rounded-full shadow-[0_0_8px_rgba(var(--primary),0.4)]" />
                    <h2 className="text-[11px] font-black uppercase tracking-[0.3em] text-foreground/80">Steam ID</h2>
                </div>

                <div className="card p-6 space-y-4">
                    {platform !== 'PC' ? (
                        <p className="text-[10px] text-muted-foreground font-medium">
                            {platform ? 'PS4 saves do not contain a SteamID.' : 'Load a PC save file to edit the SteamID.'}
                        </p>
                    ) : (
                        <>
                            <div className="space-y-1">
                                <p className="text-xs font-bold text-foreground">Steam ID</p>
                                <p className="text-[10px] text-muted-foreground font-medium">17-digit Steam account ID embedded in the save file. Required for the save to load on the correct account.</p>
                            </div>
                            <div className="flex items-center gap-3">
                                <input
                                    type="text"
                                    value={steamIdInput}
                                    onChange={e => { setSteamIdInput(e.target.value); setSteamIdError(''); }}
                                    maxLength={17}
                                    placeholder="76561198XXXXXXXXX"
                                    className="flex-1 bg-background border border-border/50 rounded-md px-4 py-2.5 text-[11px] font-mono focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                                />
                                <button
                                    onClick={handleApplySteamId}
                                    disabled={steamIdApplying || steamIdInput === steamIdSaved}
                                    className="px-5 py-2.5 bg-primary text-primary-foreground rounded-md text-[10px] font-black uppercase tracking-widest shadow-lg shadow-primary/20 hover:brightness-110 active:scale-95 transition-all disabled:opacity-50 disabled:scale-100"
                                >
                                    {steamIdApplying ? 'Applying...' : 'Apply'}
                                </button>
                            </div>
                            {steamIdError && (
                                <p className="text-[10px] text-red-400 font-bold">{steamIdError}</p>
                            )}
                            {steamIdSaved && steamIdInput === steamIdSaved && (
                                <p className="text-[10px] text-green-500 font-bold">Current: {steamIdSaved}</p>
                            )}
                        </>
                    )}
                </div>
            </section>

            {/* UI Customization Section */}
            <section className="space-y-6">
                <div className="flex items-center space-x-4 px-1">
                    <div className="w-1.5 h-6 bg-primary rounded-full shadow-[0_0_8px_rgba(var(--primary),0.4)]" />
                    <h2 className="text-[11px] font-black uppercase tracking-[0.3em] text-foreground/80">UI Customization</h2>
                </div>
                
                <div className="card p-6 space-y-6">
                    <div className="space-y-4">
                        <p className="text-xs font-bold text-foreground">Inventory Table Columns</p>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <label className="flex items-center justify-between p-4 rounded-lg bg-muted/20 border border-border/50 cursor-pointer hover:bg-muted/30 transition-all">
                                <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Show ID (HEX)</span>
                                <input
                                    type="checkbox"
                                    checked={columnVisibility.id}
                                    onChange={e => setColumnVisibility({ ...columnVisibility, id: e.target.checked })}
                                    className="w-4 h-4 rounded border-border text-primary focus:ring-primary/20"
                                />
                            </label>
                            <label className="flex items-center justify-between p-4 rounded-lg bg-muted/20 border border-border/50 cursor-pointer hover:bg-muted/30 transition-all">
                                <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Show Category</span>
                                <input
                                    type="checkbox"
                                    checked={columnVisibility.category}
                                    onChange={e => setColumnVisibility({ ...columnVisibility, category: e.target.checked })}
                                    className="w-4 h-4 rounded border-border text-primary focus:ring-primary/20"
                                />
                            </label>
                        </div>
                    </div>

                    <div className="space-y-4 border-t border-border/40 pt-6">
                        <p className="text-xs font-bold text-foreground">Flagged Content</p>
                        <label className="flex items-center justify-between p-4 rounded-lg bg-muted/20 border border-border/50 cursor-pointer hover:bg-muted/30 transition-all">
                            <div className="space-y-1">
                                <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Show Cut &amp; Ban-Risk Items</span>
                                <p className="text-[9px] text-muted-foreground/60 font-medium">Display items marked as cut content or ban risk in Database and Inventory.</p>
                            </div>
                            <input
                                type="checkbox"
                                checked={showFlaggedItems}
                                onChange={e => setShowFlaggedItems(e.target.checked)}
                                className="w-4 h-4 rounded border-border text-primary focus:ring-primary/20 shrink-0 ml-4"
                            />
                        </label>
                    </div>
                </div>
            </section>
        </div>
    );
}
