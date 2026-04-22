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
    columnVisibility: { id: boolean; category: boolean };
    setColumnVisibility: (visibility: { id: boolean; category: boolean }) => void;
    showFlaggedItems: boolean;
    setShowFlaggedItems: (value: boolean) => void;
    debugMode: boolean;
    setDebugMode: (value: boolean) => void;
    platform: string | null;
    setPlatform: (platform: string | null) => void;
    refreshSlots: () => void;
}

const EMPTY_SSH_TARGET: deploy.Target = new deploy.Target({
    type: 'ssh', name: '', host: '', port: 22, user: 'deck', keyPath: '~/.ssh/id_rsa',
    savePath: '/home/deck/.local/share/Steam/steamapps/compatdata/1245620/pfx/drive_c/users/steamuser/AppData/Roaming/EldenRing/{STEAM_ID}/ER0000.sl2',
    gameStartCmd: 'steam steam://rungameid/1245620',
    gameStopCmd: 'pkill -TERM -f eldenring.exe',
});

const EMPTY_LOCAL_TARGET: deploy.Target = new deploy.Target({
    type: 'local', name: '', host: '', port: 22, user: '', keyPath: '',
    savePath: '',
    gameStartCmd: '',
    gameStopCmd: '',
});

export function SettingsTab({
    theme, setTheme, columnVisibility, setColumnVisibility,
    showFlaggedItems, setShowFlaggedItems, debugMode, setDebugMode,
    platform, setPlatform, refreshSlots
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
    const [editTarget, setEditTarget] = useState<deploy.Target>(new deploy.Target(EMPTY_SSH_TARGET));
    const [showForm, setShowForm] = useState(false);
    const [deploying, setDeploying] = useState(false);

    const loadTargets = useCallback(() => {
        GetDeployTargets().then(t => setTargets(t || [])).catch(() => setTargets([]));
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
        setSteamIdApplying(true); setSteamIdError('');
        try { await SetSteamIDFromString(steamIdInput); setSteamIdSaved(steamIdInput); }
        catch (e) { setSteamIdError(String(e)); }
        finally { setSteamIdApplying(false); }
    };

    const handleImport = async () => {
        setImporting(true);
        try { const plat = await SelectAndOpenSave(); setPlatform(plat); refreshSlots(); toast.success("Save imported"); }
        catch (err) { toast.error(String(err)); }
        finally { setImporting(false); }
    };

    const handleExport = async () => {
        setExporting(true);
        try { await WriteSave(targetPlatform); toast.success(`Exported as ${targetPlatform}`); }
        catch (err) { toast.error(String(err)); }
        finally { setExporting(false); }
    };

    // Deploy handlers
    const handleSaveTarget = async () => {
        if (!editTarget.name || !editTarget.host) { toast.error('Name and host required'); return; }
        try { await SaveDeployTarget(editTarget); toast.success(`Target "${editTarget.name}" saved`); loadTargets(); setShowForm(false); setSelectedTarget(editTarget.name); }
        catch (e) { toast.error(String(e)); }
    };
    const handleDeleteTarget = async (name: string) => {
        try { await DeleteDeployTarget(name); toast.success(`Deleted "${name}"`); if (selectedTarget === name) setSelectedTarget(''); loadTargets(); }
        catch (e) { toast.error(String(e)); }
    };
    const handleTestConnection = async () => {
        if (!selectedTarget) return;
        const tid = toast.loading('Testing...');
        try { toast.success(await TestSSHConnection(selectedTarget), { id: tid }); }
        catch (e) { toast.error(String(e), { id: tid }); }
    };
    const handleUpload = async () => {
        if (!selectedTarget) return; setDeploying(true);
        const tid = toast.loading('Uploading save...');
        try { const msg = await DeploySave(selectedTarget); toast.success(msg, { id: tid }); }
        catch (e) { toast.error(String(e), { id: tid }); }
        finally { setDeploying(false); }
    };
    const handleDownload = async () => {
        if (!selectedTarget) return; setDeploying(true);
        const tid = toast.loading('Downloading...');
        try { const plat = await DownloadRemoteSave(selectedTarget); setPlatform(plat); refreshSlots(); toast.success('Downloaded & loaded', { id: tid }); }
        catch (e) { toast.error(String(e), { id: tid }); }
        finally { setDeploying(false); }
    };
    const handleLaunch = async () => {
        if (!selectedTarget) return;
        try { const msg = await LaunchRemoteGame(selectedTarget); toast.success(msg || 'Game launch sent'); } catch (e) { toast.error(String(e)); }
    };
    const handleClose = async () => {
        if (!selectedTarget) return;
        try { const msg = await CloseRemoteGame(selectedTarget); toast.success(msg || 'Game close sent'); } catch (e) { toast.error(String(e)); }
    };
    const handleDeployAndLaunch = async () => {
        if (!selectedTarget) return; setDeploying(true);
        const tid = toast.loading('Close → Upload → Launch...');
        try { await DeployAndLaunch(selectedTarget); toast.success('Deploy complete', { id: tid }); }
        catch (e) { toast.error(String(e), { id: tid }); }
        finally { setDeploying(false); }
    };

    const inputCls = "w-full bg-background border border-border/50 rounded px-2.5 py-1.5 text-[11px] font-mono focus:outline-none focus:ring-1 focus:ring-primary/20 focus:border-primary transition-all";
    const labelCls = "text-[8px] font-black uppercase tracking-widest text-muted-foreground";
    const btnSm = "px-2.5 py-1 rounded text-[8px] font-black uppercase tracking-widest transition-all disabled:opacity-50";
    const sectionHdr = "flex items-center space-x-3 px-1";
    const dot = "w-1 h-5 bg-primary rounded-full shadow-[0_0_6px_rgba(var(--primary),0.3)]";
    const hdrText = "text-[10px] font-black uppercase tracking-[0.25em] text-foreground/80";

    return (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
            {/* Appearance */}
            <section className="space-y-3">
                <div className={sectionHdr}><div className={dot} /><h2 className={hdrText}>Appearance</h2></div>
                <div className="card px-4 py-3">
                    <div className="flex items-center justify-between gap-3">
                        <p className="text-[10px] font-bold text-foreground">Theme</p>
                        <div className="flex bg-muted/30 p-0.5 rounded border border-border">
                            {(['light', 'dark', 'system'] as const).map(t => (
                                <button key={t} onClick={() => setTheme(t)}
                                    className={`px-4 py-1 rounded text-[9px] font-black uppercase tracking-widest transition-all ${theme === t ? 'bg-primary text-primary-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}`}
                                >{t}</button>
                            ))}
                        </div>
                    </div>
                </div>
            </section>

            {/* File Operations */}
            <section className="space-y-3">
                <div className={sectionHdr}><div className={dot} /><h2 className={hdrText}>File Operations</h2></div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div className="card px-4 py-3 space-y-2">
                        <p className="text-[10px] font-bold text-foreground">Import Save</p>
                        <button onClick={handleImport} disabled={importing}
                            className="w-full bg-muted/30 hover:bg-muted/50 text-foreground font-black py-2 rounded text-[8px] uppercase tracking-[0.15em] border border-border transition-all flex items-center justify-center space-x-1.5">
                            {importing ? <div className="w-3 h-3 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" /> :
                            <><svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path></svg><span>Select File</span></>}
                        </button>
                    </div>
                    <div className="card px-4 py-3 space-y-2">
                        <p className="text-[10px] font-bold text-foreground">Export Save</p>
                        <div className="flex bg-muted/30 p-0.5 rounded border border-border mb-1.5">
                            {(['PC', 'PS4'] as const).map(p => (
                                <button key={p} onClick={() => setTargetPlatform(p)}
                                    className={`flex-1 py-1 rounded text-[8px] font-black uppercase tracking-widest transition-all ${targetPlatform === p ? 'bg-background text-foreground shadow-sm ring-1 ring-border' : 'text-muted-foreground hover:text-foreground'}`}
                                >{p}</button>
                            ))}
                        </div>
                        <button onClick={handleExport} disabled={!platform || exporting}
                            className="w-full bg-primary text-primary-foreground font-black py-2 rounded text-[8px] uppercase tracking-[0.15em] shadow-lg shadow-primary/20 hover:brightness-110 active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center space-x-1.5">
                            {exporting ? <div className="w-3 h-3 border-2 border-primary-foreground/20 border-t-primary-foreground rounded-full animate-spin" /> :
                            <><svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4"></path></svg><span>Export as {targetPlatform}</span></>}
                        </button>
                    </div>
                </div>
            </section>

            {/* Deploy */}
            <section className="space-y-3">
                <div className={sectionHdr}><div className={dot} /><h2 className={hdrText}>Deploy</h2></div>
                <div className="card px-4 py-3 space-y-3">
                    <div className="flex items-center gap-2">
                        <select value={selectedTarget} onChange={e => setSelectedTarget(e.target.value)}
                            className="flex-1 bg-background border border-border/50 rounded px-2.5 py-1.5 text-[11px] font-mono focus:outline-none focus:ring-1 focus:ring-primary/20 transition-all">
                            <option value="">Select target...</option>
                            {targets.map(t => <option key={t.name} value={t.name}>{t.name} ({t.type === 'local' ? 'local' : t.host})</option>)}
                        </select>
                        <button onClick={() => { setEditTarget(new deploy.Target(EMPTY_SSH_TARGET)); setShowForm(true); }}
                            className={`${btnSm} bg-primary text-primary-foreground shadow-sm hover:brightness-110`}>+ Add</button>
                        {selectedTarget && <>
                            <button onClick={() => { const t = targets.find(x => x.name === selectedTarget); if (t) { setEditTarget(new deploy.Target(t)); setShowForm(true); } }}
                                className={`${btnSm} bg-muted/30 text-foreground border border-border hover:bg-muted/50`}>Edit</button>
                            <button onClick={() => handleDeleteTarget(selectedTarget)}
                                className={`${btnSm} bg-red-500/10 text-red-400 border border-red-500/20 hover:bg-red-500/20`}>Del</button>
                        </>}
                    </div>
                    {selectedTarget && (
                        <div className="flex flex-wrap gap-1.5">
                            <button onClick={handleTestConnection} disabled={deploying} className={`${btnSm} bg-muted/30 text-foreground border border-border hover:bg-muted/50`}>Test</button>
                            <button onClick={handleUpload} disabled={deploying || !platform} className={`${btnSm} bg-blue-500/10 text-blue-400 border border-blue-500/20 hover:bg-blue-500/20`}>Upload</button>
                            <button onClick={handleDownload} disabled={deploying} className={`${btnSm} bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 hover:bg-cyan-500/20`}>Download</button>
                            <button onClick={handleLaunch} disabled={deploying} className={`${btnSm} bg-green-500/10 text-green-400 border border-green-500/20 hover:bg-green-500/20`}>Launch</button>
                            <button onClick={handleClose} disabled={deploying} className={`${btnSm} bg-orange-500/10 text-orange-400 border border-orange-500/20 hover:bg-orange-500/20`}>Close</button>
                            <button onClick={handleDeployAndLaunch} disabled={deploying || !platform} className={`${btnSm} bg-primary text-primary-foreground shadow-sm hover:brightness-110`}>Deploy & Launch</button>
                        </div>
                    )}
                    {showForm && (
                        <div className="border border-border/50 rounded p-3 space-y-2.5 bg-muted/10">
                            <div className="flex items-center justify-between">
                                <p className="text-[10px] font-bold text-foreground">{editTarget.name && targets.some(t => t.name === editTarget.name) ? 'Edit Target' : 'Add Target'}</p>
                                <div className="flex bg-muted/30 p-0.5 rounded border border-border">
                                    {(['ssh', 'local'] as const).map(tp => (
                                        <button key={tp} onClick={() => {
                                            const base = tp === 'local' ? EMPTY_LOCAL_TARGET : EMPTY_SSH_TARGET;
                                            setEditTarget(new deploy.Target({...base, name: editTarget.name, savePath: editTarget.savePath} as deploy.Target));
                                        }}
                                            className={`px-3 py-0.5 rounded text-[8px] font-black uppercase tracking-widest transition-all ${editTarget.type === tp ? 'bg-primary text-primary-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}`}
                                        >{tp}</button>
                                    ))}
                                </div>
                            </div>
                            <div className={`grid gap-2 ${editTarget.type === 'local' ? 'grid-cols-1' : 'grid-cols-2 md:grid-cols-4'}`}>
                                <div className="space-y-0.5"><label className={labelCls}>Name</label><input value={editTarget.name} onChange={e => setEditTarget({...editTarget, name: e.target.value} as deploy.Target)} placeholder={editTarget.type === 'local' ? 'Local PC' : 'Steam Deck'} className={inputCls} /></div>
                                {editTarget.type === 'ssh' && <>
                                    <div className="space-y-0.5"><label className={labelCls}>Host</label><input value={editTarget.host} onChange={e => setEditTarget({...editTarget, host: e.target.value} as deploy.Target)} placeholder="192.168.1.100" className={inputCls} /></div>
                                    <div className="space-y-0.5"><label className={labelCls}>Port</label><input type="number" value={editTarget.port} onChange={e => setEditTarget({...editTarget, port: parseInt(e.target.value) || 22} as deploy.Target)} className={inputCls} /></div>
                                    <div className="space-y-0.5"><label className={labelCls}>User</label><input value={editTarget.user} onChange={e => setEditTarget({...editTarget, user: e.target.value} as deploy.Target)} placeholder="deck" className={inputCls} /></div>
                                </>}
                            </div>
                            <div className={`grid gap-2 ${editTarget.type === 'local' ? 'grid-cols-1' : 'grid-cols-1 md:grid-cols-2'}`}>
                                {editTarget.type === 'ssh' && (
                                    <div className="space-y-0.5"><label className={labelCls}>SSH Key Path</label><input value={editTarget.keyPath} onChange={e => setEditTarget({...editTarget, keyPath: e.target.value} as deploy.Target)} className={inputCls} /></div>
                                )}
                                <div className="space-y-0.5"><label className={labelCls}>Save Path</label><input value={editTarget.savePath} onChange={e => setEditTarget({...editTarget, savePath: e.target.value} as deploy.Target)} placeholder={editTarget.type === 'local' ? 'C:\\Users\\...\\EldenRing\\{STEAM_ID}\\ER0000.sl2' : ''} className={inputCls} /></div>
                            </div>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                                <div className="space-y-0.5"><label className={labelCls}>Start Command <span className="text-muted-foreground/50">(empty = auto-detect)</span></label><input value={editTarget.gameStartCmd} onChange={e => setEditTarget({...editTarget, gameStartCmd: e.target.value} as deploy.Target)} className={inputCls} /></div>
                                <div className="space-y-0.5"><label className={labelCls}>Stop Command <span className="text-muted-foreground/50">(empty = auto-detect)</span></label><input value={editTarget.gameStopCmd} onChange={e => setEditTarget({...editTarget, gameStopCmd: e.target.value} as deploy.Target)} className={inputCls} /></div>
                            </div>
                            <div className="flex gap-1.5 pt-1">
                                <button onClick={handleSaveTarget} className={`${btnSm} bg-primary text-primary-foreground shadow-sm hover:brightness-110`}>Save</button>
                                <button onClick={() => setShowForm(false)} className={`${btnSm} bg-muted/30 text-foreground border border-border hover:bg-muted/50`}>Cancel</button>
                            </div>
                        </div>
                    )}
                </div>
            </section>

            {/* SteamID */}
            <section className="space-y-3">
                <div className={sectionHdr}><div className={dot} /><h2 className={hdrText}>Steam ID</h2></div>
                <div className="card px-4 py-3">
                    {platform !== 'PC' ? (
                        <p className="text-[10px] text-muted-foreground font-medium">{platform ? 'PS4 saves do not contain a SteamID.' : 'Load a PC save to edit SteamID.'}</p>
                    ) : (
                        <div className="space-y-2">
                            <p className="text-[9px] text-muted-foreground font-medium">17-digit Steam account ID embedded in the save file.</p>
                            <div className="flex items-center gap-2">
                                <input type="text" value={steamIdInput} onChange={e => { setSteamIdInput(e.target.value); setSteamIdError(''); }}
                                    maxLength={17} placeholder="76561198XXXXXXXXX"
                                    className="flex-1 bg-background border border-border/50 rounded px-3 py-1.5 text-[11px] font-mono focus:outline-none focus:ring-1 focus:ring-primary/20 transition-all" />
                                <button onClick={handleApplySteamId} disabled={steamIdApplying || steamIdInput === steamIdSaved}
                                    className="px-4 py-1.5 bg-primary text-primary-foreground rounded text-[9px] font-black uppercase tracking-widest shadow-sm hover:brightness-110 transition-all disabled:opacity-50">
                                    {steamIdApplying ? '...' : 'Apply'}
                                </button>
                            </div>
                            {steamIdError && <p className="text-[9px] text-red-400 font-bold">{steamIdError}</p>}
                            {steamIdSaved && steamIdInput === steamIdSaved && <p className="text-[9px] text-green-500 font-bold">Current: {steamIdSaved}</p>}
                        </div>
                    )}
                </div>
            </section>

            {/* UI Customization */}
            <section className="space-y-3">
                <div className={sectionHdr}><div className={dot} /><h2 className={hdrText}>UI Customization</h2></div>
                <div className="card px-4 py-3 space-y-3">
                    <div className="space-y-2">
                        <p className="text-[10px] font-bold text-foreground">Inventory Columns</p>
                        <div className="grid grid-cols-2 gap-2">
                            <label className="flex items-center justify-between p-2.5 rounded bg-muted/20 border border-border/50 cursor-pointer hover:bg-muted/30 transition-all">
                                <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">ID (HEX)</span>
                                <input type="checkbox" checked={columnVisibility.id} onChange={e => setColumnVisibility({ ...columnVisibility, id: e.target.checked })} className="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20" />
                            </label>
                            <label className="flex items-center justify-between p-2.5 rounded bg-muted/20 border border-border/50 cursor-pointer hover:bg-muted/30 transition-all">
                                <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Category</span>
                                <input type="checkbox" checked={columnVisibility.category} onChange={e => setColumnVisibility({ ...columnVisibility, category: e.target.checked })} className="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20" />
                            </label>
                        </div>
                    </div>
                    <div className="border-t border-border/40 pt-2.5">
                        <label className="flex items-center justify-between p-2.5 rounded bg-muted/20 border border-border/50 cursor-pointer hover:bg-muted/30 transition-all">
                            <div><span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Cut & Ban-Risk Items</span><p className="text-[8px] text-muted-foreground/60 font-medium mt-0.5">Show flagged items in Database/Inventory.</p></div>
                            <input type="checkbox" checked={showFlaggedItems} onChange={e => setShowFlaggedItems(e.target.checked)} className="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20 shrink-0 ml-3" />
                        </label>
                    </div>
                    <div className="border-t border-border/40 pt-2.5">
                        <label className="flex items-center justify-between p-2.5 rounded bg-muted/20 border border-border/50 cursor-pointer hover:bg-muted/30 transition-all">
                            <div><span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">Debug Mode</span><p className="text-[8px] text-muted-foreground/60 font-medium mt-0.5">Show all parser warnings.</p></div>
                            <input type="checkbox" checked={debugMode} onChange={e => setDebugMode(e.target.checked)} className="w-3.5 h-3.5 rounded border-border text-primary focus:ring-primary/20 shrink-0 ml-3" />
                        </label>
                    </div>
                </div>
            </section>
        </div>
    );
}
