import {useState, useEffect, useCallback} from 'react';
import toast from 'react-hot-toast';
import {
    SelectAndOpenSave, WriteSave, GetSteamIDString, SetSteamIDFromString,
    GetDeployTargets, SaveDeployTarget, DeleteDeployTarget,
    TestSSHConnection, DeploySave, DownloadRemoteSave,
    LaunchRemoteGame, CloseRemoteGame, DeployAndLaunch,
} from '../../wailsjs/go/main/App';
import {deploy} from '../../wailsjs/go/models';

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
    debugMode: boolean;
    setDebugMode: (value: boolean) => void;
    platform: string | null;
    setPlatform: (platform: string | null) => void;
    refreshSlots: () => void;
}

const EMPTY_TARGET: deploy.Target = new deploy.Target({
    name: '', host: '', port: 22, user: 'deck', keyPath: '~/.ssh/id_rsa',
    savePath: '/home/deck/.local/share/Steam/steamapps/compatdata/1245620/pfx/drive_c/users/steamuser/AppData/Roaming/EldenRing/{STEAM_ID}/ER0000.sl2',
    gameStartCmd: 'steam steam://rungameid/1245620',
    gameStopCmd: 'pkill -TERM -f eldenring.exe',
});

export function SettingsTab({
    theme,
    setTheme,
    columnVisibility,
    setColumnVisibility,
    showFlaggedItems,
    setShowFlaggedItems,
    debugMode,
    setDebugMode,
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

    // Deploy state
    const [targets, setTargets] = useState<deploy.Target[]>([]);
    const [selectedTarget, setSelectedTarget] = useState<string>('');
    const [editTarget, setEditTarget] = useState<deploy.Target>(new deploy.Target(EMPTY_TARGET));
    const [showForm, setShowForm] = useState(false);
    const [deploying, setDeploying] = useState(false);

    const loadTargets = useCallback(() => {
        GetDeployTargets().then(t => {
            setTargets(t || []);
        }).catch(() => setTargets([]));
    }, []);

    useEffect(() => { loadTargets(); }, [loadTargets]);

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
            toast.success("Save imported successfully!");
        } catch (err) {
            toast.error(String(err));
        } finally {
            setImporting(false);
        }
    };

    const handleExport = async () => {
        setExporting(true);
        try {
            await WriteSave(targetPlatform);
            toast.success(`Save exported successfully as ${targetPlatform}!`);
        } catch (err) {
            toast.error(String(err));
        } finally {
            setExporting(false);
        }
    };

    // Deploy handlers
    const handleSaveTarget = async () => {
        if (!editTarget.name || !editTarget.host) {
            toast.error('Name and host are required'); return;
        }
        try {
            await SaveDeployTarget(editTarget);
            toast.success(`Target "${editTarget.name}" saved`);
            loadTargets();
            setShowForm(false);
            setSelectedTarget(editTarget.name);
        } catch (e) { toast.error(String(e)); }
    };

    const handleDeleteTarget = async (name: string) => {
        try {
            await DeleteDeployTarget(name);
            toast.success(`Target "${name}" deleted`);
            if (selectedTarget === name) setSelectedTarget('');
            loadTargets();
        } catch (e) { toast.error(String(e)); }
    };

    const handleTestConnection = async () => {
        if (!selectedTarget) return;
        const tid = toast.loading('Testing connection...');
        try {
            const msg = await TestSSHConnection(selectedTarget);
            toast.success(msg, { id: tid });
        } catch (e) { toast.error(String(e), { id: tid }); }
    };

    const handleUpload = async () => {
        if (!selectedTarget) return;
        setDeploying(true);
        const tid = toast.loading('Uploading save...');
        try {
            await DeploySave(selectedTarget);
            toast.success('Save uploaded successfully', { id: tid });
        } catch (e) { toast.error(String(e), { id: tid }); }
        finally { setDeploying(false); }
    };

    const handleDownload = async () => {
        if (!selectedTarget) return;
        setDeploying(true);
        const tid = toast.loading('Downloading save...');
        try {
            const plat = await DownloadRemoteSave(selectedTarget);
            setPlatform(plat);
            refreshSlots();
            toast.success('Save downloaded and loaded', { id: tid });
        } catch (e) { toast.error(String(e), { id: tid }); }
        finally { setDeploying(false); }
    };

    const handleLaunch = async () => {
        if (!selectedTarget) return;
        try {
            await LaunchRemoteGame(selectedTarget);
            toast.success('Game launched');
        } catch (e) { toast.error(String(e)); }
    };

    const handleClose = async () => {
        if (!selectedTarget) return;
        try {
            await CloseRemoteGame(selectedTarget);
            toast.success('Game closed');
        } catch (e) { toast.error(String(e)); }
    };

    const handleDeployAndLaunch = async () => {
        if (!selectedTarget) return;
        setDeploying(true);
        const tid = toast.loading('Deploying: Close → Upload → Launch...');
        try {
            await DeployAndLaunch(selectedTarget);
            toast.success('Deploy complete — game launched', { id: tid });
        } catch (e) { toast.error(String(e), { id: tid }); }
        finally { setDeploying(false); }
    };

    const inputCls = "w-full bg-background border border-border/50 rounded-md px-3 py-2 text-[11px] font-mono focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all";
    const labelCls = "text-[9px] font-black uppercase tracking-widest text-muted-foreground";
    const btnSmall = "px-3 py-1.5 rounded-md text-[9px] font-black uppercase tracking-widest transition-all disabled:opacity-50";

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

            {/* Remote Deploy Section */}
            <section className="space-y-6">
                <div className="flex items-center space-x-4 px-1">
                    <div className="w-1.5 h-6 bg-primary rounded-full shadow-[0_0_8px_rgba(var(--primary),0.4)]" />
                    <h2 className="text-[11px] font-black uppercase tracking-[0.3em] text-foreground/80">Remote Deploy</h2>
                </div>

                <div className="card p-6 space-y-6">
                    {/* Target selector + actions */}
                    <div className="flex flex-col gap-4">
                        <div className="flex items-center gap-3">
                            <select
                                value={selectedTarget}
                                onChange={e => setSelectedTarget(e.target.value)}
                                className="flex-1 bg-background border border-border/50 rounded-md px-3 py-2.5 text-[11px] font-mono focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                            >
                                <option value="">Select target...</option>
                                {targets.map(t => (
                                    <option key={t.name} value={t.name}>{t.name} ({t.host})</option>
                                ))}
                            </select>
                            <button
                                onClick={() => { setEditTarget(new deploy.Target(EMPTY_TARGET)); setShowForm(true); }}
                                className={`${btnSmall} bg-primary text-primary-foreground shadow-sm shadow-primary/20 hover:brightness-110`}
                            >
                                + Add
                            </button>
                            {selectedTarget && (
                                <>
                                    <button
                                        onClick={() => {
                                            const t = targets.find(x => x.name === selectedTarget);
                                            if (t) { setEditTarget(new deploy.Target(t)); setShowForm(true); }
                                        }}
                                        className={`${btnSmall} bg-muted/30 text-foreground border border-border hover:bg-muted/50`}
                                    >
                                        Edit
                                    </button>
                                    <button
                                        onClick={() => handleDeleteTarget(selectedTarget)}
                                        className={`${btnSmall} bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20`}
                                    >
                                        Delete
                                    </button>
                                </>
                            )}
                        </div>

                        {/* Action buttons */}
                        {selectedTarget && (
                            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-2">
                                <button onClick={handleTestConnection} disabled={deploying} className={`${btnSmall} bg-muted/30 text-foreground border border-border hover:bg-muted/50`}>
                                    Test
                                </button>
                                <button onClick={handleUpload} disabled={deploying || !platform} className={`${btnSmall} bg-blue-500/10 text-blue-400 border border-blue-500/20 hover:bg-blue-500/20`}>
                                    Upload
                                </button>
                                <button onClick={handleDownload} disabled={deploying} className={`${btnSmall} bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 hover:bg-cyan-500/20`}>
                                    Download
                                </button>
                                <button onClick={handleLaunch} disabled={deploying} className={`${btnSmall} bg-green-500/10 text-green-400 border border-green-500/20 hover:bg-green-500/20`}>
                                    Launch
                                </button>
                                <button onClick={handleClose} disabled={deploying} className={`${btnSmall} bg-orange-500/10 text-orange-400 border border-orange-500/20 hover:bg-orange-500/20`}>
                                    Close
                                </button>
                                <button onClick={handleDeployAndLaunch} disabled={deploying || !platform} className={`${btnSmall} bg-primary text-primary-foreground shadow-sm shadow-primary/20 hover:brightness-110`}>
                                    Deploy & Launch
                                </button>
                            </div>
                        )}
                    </div>

                    {/* Target form (add/edit) */}
                    {showForm && (
                        <div className="border border-border/50 rounded-lg p-5 space-y-4 bg-muted/10">
                            <p className="text-xs font-bold text-foreground">
                                {editTarget.name && targets.some(t => t.name === editTarget.name) ? 'Edit Target' : 'Add Target'}
                            </p>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div className="space-y-1">
                                    <label className={labelCls}>Name</label>
                                    <input value={editTarget.name} onChange={e => setEditTarget({...editTarget, name: e.target.value} as deploy.Target)} placeholder="Steam Deck" className={inputCls} />
                                </div>
                                <div className="space-y-1">
                                    <label className={labelCls}>Host</label>
                                    <input value={editTarget.host} onChange={e => setEditTarget({...editTarget, host: e.target.value} as deploy.Target)} placeholder="192.168.1.100" className={inputCls} />
                                </div>
                                <div className="space-y-1">
                                    <label className={labelCls}>Port</label>
                                    <input type="number" value={editTarget.port} onChange={e => setEditTarget({...editTarget, port: parseInt(e.target.value) || 22} as deploy.Target)} className={inputCls} />
                                </div>
                                <div className="space-y-1">
                                    <label className={labelCls}>User</label>
                                    <input value={editTarget.user} onChange={e => setEditTarget({...editTarget, user: e.target.value} as deploy.Target)} placeholder="deck" className={inputCls} />
                                </div>
                                <div className="space-y-1 md:col-span-2">
                                    <label className={labelCls}>SSH Key Path</label>
                                    <input value={editTarget.keyPath} onChange={e => setEditTarget({...editTarget, keyPath: e.target.value} as deploy.Target)} placeholder="~/.ssh/id_rsa" className={inputCls} />
                                </div>
                                <div className="space-y-1 md:col-span-2">
                                    <label className={labelCls}>Remote Save Path</label>
                                    <input value={editTarget.savePath} onChange={e => setEditTarget({...editTarget, savePath: e.target.value} as deploy.Target)} className={inputCls} />
                                </div>
                                <div className="space-y-1">
                                    <label className={labelCls}>Game Start Command</label>
                                    <input value={editTarget.gameStartCmd} onChange={e => setEditTarget({...editTarget, gameStartCmd: e.target.value} as deploy.Target)} className={inputCls} />
                                </div>
                                <div className="space-y-1">
                                    <label className={labelCls}>Game Stop Command</label>
                                    <input value={editTarget.gameStopCmd} onChange={e => setEditTarget({...editTarget, gameStopCmd: e.target.value} as deploy.Target)} className={inputCls} />
                                </div>
                            </div>
                            <div className="flex gap-2 pt-2">
                                <button onClick={handleSaveTarget} className={`${btnSmall} bg-primary text-primary-foreground shadow-sm shadow-primary/20 hover:brightness-110`}>
                                    Save Target
                                </button>
                                <button onClick={() => setShowForm(false)} className={`${btnSmall} bg-muted/30 text-foreground border border-border hover:bg-muted/50`}>
                                    Cancel
                                </button>
                            </div>
                        </div>
                    )}
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

                    <div className="space-y-4 border-t border-border/40 pt-6">
                        <p className="text-xs font-bold text-foreground">Debug</p>
                        <label className="flex items-center justify-between p-4 rounded-lg bg-muted/20 border border-border/50 cursor-pointer hover:bg-muted/30 transition-all">
                            <div className="space-y-1">
                                <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Debug Mode</span>
                                <p className="text-[9px] text-muted-foreground/60 font-medium">Show all parser warnings including non-critical diagnostic messages (e.g. PS4 dynamic size clamps).</p>
                            </div>
                            <input
                                type="checkbox"
                                checked={debugMode}
                                onChange={e => setDebugMode(e.target.checked)}
                                className="w-4 h-4 rounded border-border text-primary focus:ring-primary/20 shrink-0 ml-4"
                            />
                        </label>
                    </div>
                </div>
            </section>
        </div>
    );
}
