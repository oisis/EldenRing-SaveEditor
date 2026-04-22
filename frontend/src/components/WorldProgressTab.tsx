import {useEffect, useState} from 'react';
import {GetGraces, SetGraceVisited, GetBosses, SetBossDefeated, GetSummoningPools, SetSummoningPoolActivated, GetColosseums, SetColosseumUnlocked, GetMapProgress, SetMapFlag, SetMapRegionFlags, RevealAllMap, ResetMapExploration, RemoveFogOfWar} from '../../wailsjs/go/main/App';
import {db} from '../../wailsjs/go/models';

interface WorldProgressTabProps {
    charIdx: number;
    onMutate?: () => void;
}

// Shared compact checkbox + label
const Chk = ({checked, onChange}: {checked: boolean; onChange: (v: boolean) => void}) => (
    <div className="relative flex items-center justify-center">
        <input type="checkbox" checked={checked} onChange={e => onChange(e.target.checked)}
            className="peer appearance-none w-3.5 h-3.5 rounded border border-border bg-background checked:bg-primary checked:border-primary transition-all cursor-pointer" />
        <svg className="absolute w-2 h-2 text-white pointer-events-none hidden peer-checked:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3.5" d="M5 13l4 4L19 7"></path>
        </svg>
    </div>
);

const ChkX = ({checked, onChange}: {checked: boolean; onChange: (v: boolean) => void}) => (
    <div className="relative flex items-center justify-center">
        <input type="checkbox" checked={checked} onChange={e => onChange(e.target.checked)}
            className="peer appearance-none w-3.5 h-3.5 rounded border border-border bg-background checked:bg-red-500 checked:border-red-500 transition-all cursor-pointer" />
        <svg className="absolute w-2 h-2 text-white pointer-events-none hidden peer-checked:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3.5" d="M6 18L18 6M6 6l12 12"></path>
        </svg>
    </div>
);

const Arrow = ({open}: {open: boolean}) => (
    <div className={`transition-transform duration-200 ${open ? 'rotate-90 text-primary' : 'text-muted-foreground'}`}>
        <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M9 5l7 7-7 7"></path></svg>
    </div>
);

const Badge = ({count, total, activeCls}: {count: number; total: number; activeCls?: string}) => (
    <span className={`text-[8px] font-black uppercase tracking-widest px-1.5 py-0.5 rounded border ${count === total ? (activeCls || 'text-primary border-primary/50 bg-primary/10') : 'text-muted-foreground bg-muted/50 border-border'}`}>
        {count}/{total}
    </span>
);

const btnSm = "text-[8px] font-black uppercase tracking-widest text-muted-foreground border border-border/50 px-2 py-0.5 rounded transition-all";

export function WorldProgressTab({charIdx, onMutate}: WorldProgressTabProps) {
    const [graces, setGraces] = useState<db.GraceEntry[]>([]);
    const [bosses, setBosses] = useState<db.BossEntry[]>([]);
    const [pools, setPools] = useState<db.SummoningPoolEntry[]>([]);
    const [colosseums, setColosseums] = useState<db.ColosseumEntry[]>([]);
    const [mapEntries, setMapEntries] = useState<db.MapEntry[]>([]);
    const [loading, setLoading] = useState(false);
    const [expandedRegions, setExpandedRegions] = useState<Record<string, boolean>>({});
    const [expandedBossRegions, setExpandedBossRegions] = useState<Record<string, boolean>>({});
    const [expandedPoolRegions, setExpandedPoolRegions] = useState<Record<string, boolean>>({});
    const [selectedMap, setSelectedMap] = useState<{name: string, path: string} | null>(null);
    const [bossFilter, setBossFilter] = useState<'all' | 'main' | 'field'>('all');
    const [bossSort, setBossSort] = useState<'name' | 'defeated'>('name');
    const [activeSection, setActiveSection] = useState<'graces' | 'bosses' | 'pools' | 'colosseums' | 'map'>('graces');
    const [expandedMapAreas, setExpandedMapAreas] = useState<Record<string, boolean>>({});
    const [skipBossArenas, setSkipBossArenas] = useState(true);

    const loadData = () => {
        setLoading(true);
        Promise.all([
            GetGraces(charIdx).then(res => setGraces(res || [])),
            GetBosses(charIdx).then(res => setBosses(res || [])),
            GetSummoningPools(charIdx).then(res => setPools(res || [])),
            GetColosseums(charIdx).then(res => setColosseums(res || [])),
            GetMapProgress(charIdx).then(res => setMapEntries(res || [])),
        ]).finally(() => setLoading(false));
    };
    useEffect(() => { loadData(); }, [charIdx]);

    // --- Grace logic ---
    const regions = graces.reduce((acc, g) => { const r = g.region || 'Unknown'; (acc[r] ??= []).push(g); return acc; }, {} as Record<string, db.GraceEntry[]>);
    const handleGraceToggle = async (grace: db.GraceEntry, visited: boolean) => { await SetGraceVisited(charIdx, grace.id, visited); setGraces(prev => prev.map(g => g.id === grace.id ? {...g, visited} : g)); onMutate?.(); };
    const handleUnlockRegionGraces = async (rg: db.GraceEntry[]) => { await Promise.all(rg.filter(g => !g.visited).map(g => SetGraceVisited(charIdx, g.id, true))); const ids = new Set(rg.map(g => g.id)); setGraces(prev => prev.map(g => ids.has(g.id) ? {...g, visited: true} : g)); onMutate?.(); };
    const handleUnlockAllGraces = async () => { const u = graces.filter(g => !g.visited && (!skipBossArenas || !g.isBossArena)); if (!u.length) return; await Promise.all(u.map(g => SetGraceVisited(charIdx, g.id, true))); const ids = new Set(u.map(g => g.id)); setGraces(prev => prev.map(g => ids.has(g.id) ? {...g, visited: true} : g)); onMutate?.(); };
    const handleLockAllGraces = async () => { const u = graces.filter(g => g.visited); if (!u.length) return; await Promise.all(u.map(g => SetGraceVisited(charIdx, g.id, false))); const ids = new Set(u.map(g => g.id)); setGraces(prev => prev.map(g => ids.has(g.id) ? {...g, visited: false} : g)); onMutate?.(); };

    // --- Boss logic ---
    const filteredBosses = bosses.filter(b => bossFilter === 'all' || b.type === bossFilter);
    const sortedFilteredBosses = [...filteredBosses].sort((a, b) => { if (bossSort === 'defeated' && a.defeated !== b.defeated) return a.defeated ? -1 : 1; return a.name.localeCompare(b.name); });
    const bossRegions = sortedFilteredBosses.reduce((acc, b) => { const r = b.region || 'Unknown'; (acc[r] ??= []).push(b); return acc; }, {} as Record<string, db.BossEntry[]>);
    const handleBossToggle = async (boss: db.BossEntry, defeated: boolean) => { await SetBossDefeated(charIdx, boss.id, defeated); setBosses(prev => prev.map(b => b.id === boss.id ? {...b, defeated} : b)); onMutate?.(); };
    const handleKillAll = async (rb: db.BossEntry[]) => { await Promise.all(rb.filter(b => !b.defeated).map(b => SetBossDefeated(charIdx, b.id, true))); const ids = new Set(rb.map(b => b.id)); setBosses(prev => prev.map(b => ids.has(b.id) ? {...b, defeated: true} : b)); onMutate?.(); };
    const handleRespawnAll = async (rb: db.BossEntry[]) => { await Promise.all(rb.filter(b => b.defeated).map(b => SetBossDefeated(charIdx, b.id, false))); const ids = new Set(rb.map(b => b.id)); setBosses(prev => prev.map(b => ids.has(b.id) ? {...b, defeated: false} : b)); onMutate?.(); };
    const handleGlobalKillAll = async () => { const a = filteredBosses.filter(b => !b.defeated); if (!a.length) return; await Promise.all(a.map(b => SetBossDefeated(charIdx, b.id, true))); const ids = new Set(a.map(b => b.id)); setBosses(prev => prev.map(b => ids.has(b.id) ? {...b, defeated: true} : b)); onMutate?.(); };
    const handleGlobalRespawnAll = async () => { const d = filteredBosses.filter(b => b.defeated); if (!d.length) return; await Promise.all(d.map(b => SetBossDefeated(charIdx, b.id, false))); const ids = new Set(d.map(b => b.id)); setBosses(prev => prev.map(b => ids.has(b.id) ? {...b, defeated: false} : b)); onMutate?.(); };

    // --- Pool logic ---
    const poolRegions = pools.reduce((acc, p) => { const r = p.region || 'Unknown'; (acc[r] ??= []).push(p); return acc; }, {} as Record<string, db.SummoningPoolEntry[]>);
    const handlePoolToggle = async (pool: db.SummoningPoolEntry, activated: boolean) => { await SetSummoningPoolActivated(charIdx, pool.id, activated); setPools(prev => prev.map(p => p.id === pool.id ? {...p, activated} : p)); onMutate?.(); };
    const handleActivateAllPools = async (rp: db.SummoningPoolEntry[]) => { await Promise.all(rp.filter(p => !p.activated).map(p => SetSummoningPoolActivated(charIdx, p.id, true))); const ids = new Set(rp.map(p => p.id)); setPools(prev => prev.map(p => ids.has(p.id) ? {...p, activated: true} : p)); onMutate?.(); };
    const handleGlobalActivateAllPools = async () => { const i = pools.filter(p => !p.activated); if (!i.length) return; await Promise.all(i.map(p => SetSummoningPoolActivated(charIdx, p.id, true))); const ids = new Set(i.map(p => p.id)); setPools(prev => prev.map(p => ids.has(p.id) ? {...p, activated: true} : p)); onMutate?.(); };

    // --- Colosseum logic ---
    const handleColosseumToggle = async (c: db.ColosseumEntry, unlocked: boolean) => { await SetColosseumUnlocked(charIdx, c.id, unlocked); setColosseums(prev => prev.map(x => x.id === c.id ? {...x, unlocked} : x)); onMutate?.(); };
    const handleUnlockAllColosseums = async () => { const l = colosseums.filter(c => !c.unlocked); if (!l.length) return; await Promise.all(l.map(c => SetColosseumUnlocked(charIdx, c.id, true))); setColosseums(prev => prev.map(c => ({...c, unlocked: true}))); onMutate?.(); };

    // --- Map logic ---
    const mapRegionEntries = mapEntries.filter(e => e.category === 'visible');
    const mapSystemEntries = mapEntries.filter(e => e.category === 'system');
    const mapAreas = mapRegionEntries.reduce((acc, e) => { const a = e.area || 'Unknown'; (acc[a] ??= []).push(e); return acc; }, {} as Record<string, db.MapEntry[]>);
    const handleMapRegionToggle = async (entry: db.MapEntry, enabled: boolean) => {
        await SetMapRegionFlags(charIdx, entry.id, enabled);
        if (enabled) await RemoveFogOfWar(charIdx);
        const acquiredId = entry.id + 1000;
        setMapEntries(prev => prev.map(e => { if (e.id === entry.id) return {...e, enabled}; if (e.id === acquiredId && e.category === 'acquired') return {...e, enabled}; return e; }));
        onMutate?.();
    };
    const handleSystemFlagToggle = async (entry: db.MapEntry, enabled: boolean) => { await SetMapFlag(charIdx, entry.id, enabled); setMapEntries(prev => prev.map(e => e.id === entry.id ? {...e, enabled} : e)); onMutate?.(); };
    const handleRevealAllMap = async () => { await RevealAllMap(charIdx); await RemoveFogOfWar(charIdx); setMapEntries(prev => prev.map(e => ({...e, enabled: true}))); onMutate?.(); };
    const handleResetMap = async () => { await ResetMapExploration(charIdx); loadData(); onMutate?.(); };

    const totalMapRegions = mapRegionEntries.length;
    const enabledMapRegions = mapRegionEntries.filter(e => e.enabled).length;

    // Map image aliases
    const REGION_MAP_ALIASES: Record<string, string | null> = {
        'limgrave': 'limgrave', 'limgrave, west': 'limgrave', 'limgrave, east': 'limgrave',
        'liurnia of the lakes': 'liurnia_of_the_lakes', 'liurnia, north': 'liurnia_of_the_lakes', 'liurnia, east': 'liurnia_of_the_lakes', 'liurnia, west': 'liurnia_of_the_lakes',
        'weeping peninsula': null, 'crumbling farum azula': null, "miquella's haligtree": null, 'shadow of the erdtree': null,
    };
    const getRegionMapPath = (region: string): string | null => {
        const k = region.toLowerCase();
        if (k in REGION_MAP_ALIASES) { const v = REGION_MAP_ALIASES[k]; return v ? `maps/${v}.jpg` : null; }
        return `maps/${k.replace(/'/g, '').replace(/,/g, '').replace(/\s+/g, '_')}.jpg`;
    };

    // Stats
    const visitedGraces = graces.filter(g => g.visited).length;
    const defeatedBosses = bosses.filter(b => b.defeated).length;
    const mainBosses = bosses.filter(b => b.type === 'main');
    const defeatedMain = mainBosses.filter(b => b.defeated).length;
    const activatedPools = pools.filter(p => p.activated).length;
    const unlockedColosseums = colosseums.filter(c => c.unlocked).length;

    if (loading) return (
        <div className="py-16 flex flex-col items-center justify-center space-y-3">
            <div className="w-5 h-5 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
            <p className="text-[9px] font-bold text-muted-foreground uppercase tracking-widest">Scanning...</p>
        </div>
    );

    return (
        <div className="flex-1 min-h-0 space-y-3 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-8 overflow-y-auto custom-scrollbar pr-2">
            {/* Map Popover */}
            {selectedMap && (
                <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/90 backdrop-blur-sm animate-in fade-in duration-300 p-4 md:p-8" onClick={() => setSelectedMap(null)}>
                    <div className="relative max-w-4xl w-full h-full flex flex-col items-center justify-center animate-in zoom-in-95 duration-300">
                        <img src={selectedMap.path} alt={selectedMap.name} className="max-w-full max-h-full object-contain rounded-lg shadow-2xl border border-border/50" onError={e => (e.currentTarget.src = '/src/assets/images/logo-universal.png')} />
                        <div className="absolute bottom-3 left-1/2 -translate-x-1/2 bg-background/80 backdrop-blur-md px-4 py-2 rounded-full border border-border/50 shadow-xl">
                            <h3 className="text-xs font-black uppercase tracking-widest text-foreground text-center">{selectedMap.name}</h3>
                        </div>
                    </div>
                </div>
            )}

            {/* Tabs + toolbar */}
            <div className="flex items-center justify-between flex-wrap gap-1.5">
                <div className="flex items-center space-x-0.5">
                    {(['graces', 'bosses', 'pools', 'colosseums', 'map'] as const).map(s => (
                        <button key={s} onClick={() => setActiveSection(s)}
                            className={`px-3 py-1 rounded-full text-[8px] font-black uppercase tracking-[0.12em] transition-all ${activeSection === s ? 'bg-primary text-primary-foreground shadow-md shadow-primary/20' : 'text-muted-foreground hover:text-foreground hover:bg-muted/30'}`}>
                            {s === 'graces' ? 'Sites of Grace' : s === 'pools' ? 'Summoning Pools' : s === 'map' ? 'Map Discovery' : s.charAt(0).toUpperCase() + s.slice(1)}
                        </button>
                    ))}
                </div>

                {activeSection === 'graces' && (
                    <div className="flex items-center space-x-2">
                        <button onClick={handleUnlockAllGraces} className={`${btnSm} hover:text-primary hover:border-primary/50`}>Unlock All</button>
                        <button onClick={handleLockAllGraces} className={`${btnSm} hover:text-red-400 hover:border-red-400/50`}>Lock All</button>
                        <label className="flex items-center space-x-1 cursor-pointer">
                            <Chk checked={skipBossArenas} onChange={setSkipBossArenas} />
                            <span className="text-[7px] font-black uppercase tracking-widest text-muted-foreground">Skip Bosses</span>
                        </label>
                        <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">{visitedGraces}/{graces.length}</span>
                    </div>
                )}
                {activeSection === 'bosses' && (
                    <div className="flex items-center space-x-1.5">
                        <button onClick={handleGlobalKillAll} className={`${btnSm} hover:text-red-400 hover:border-red-400/50`}>Kill All</button>
                        <button onClick={handleGlobalRespawnAll} className={`${btnSm} hover:text-green-400 hover:border-green-400/50`}>Respawn</button>
                        <div className="w-px h-3 bg-border/50" />
                        {(['all', 'main', 'field'] as const).map(f => (
                            <button key={f} onClick={() => setBossFilter(f)}
                                className={`px-2 py-0.5 rounded text-[7px] font-black uppercase tracking-widest transition-all ${bossFilter === f ? 'bg-muted text-foreground border border-border' : 'text-muted-foreground hover:text-foreground'}`}>{f}</button>
                        ))}
                        <div className="w-px h-3 bg-border/50" />
                        {(['name', 'defeated'] as const).map(s => (
                            <button key={s} onClick={() => setBossSort(s)}
                                className={`px-2 py-0.5 rounded text-[7px] font-black uppercase tracking-widest transition-all ${bossSort === s ? 'bg-muted text-foreground border border-border' : 'text-muted-foreground hover:text-foreground'}`}>{s}</button>
                        ))}
                        <div className="w-px h-3 bg-border/50" />
                        <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">{defeatedMain}/{mainBosses.length}m | {defeatedBosses}/{bosses.length}</span>
                    </div>
                )}
                {activeSection === 'pools' && (
                    <div className="flex items-center space-x-2">
                        <button onClick={handleGlobalActivateAllPools} className={`${btnSm} hover:text-primary hover:border-primary/50`}>Activate All</button>
                        <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">{activatedPools}/{pools.length}</span>
                    </div>
                )}
                {activeSection === 'colosseums' && (
                    <div className="flex items-center space-x-2">
                        <button onClick={handleUnlockAllColosseums} className={`${btnSm} hover:text-primary hover:border-primary/50`}>Unlock All</button>
                        <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">{unlockedColosseums}/{colosseums.length}</span>
                    </div>
                )}
                {activeSection === 'map' && (
                    <div className="flex items-center space-x-2">
                        <button onClick={handleRevealAllMap} className={`${btnSm} hover:text-primary hover:border-primary/50`}>Reveal All</button>
                        <button onClick={handleResetMap} className={`${btnSm} hover:text-red-400 hover:border-red-400/50`}>Reset</button>
                        <span className="text-[8px] font-black uppercase tracking-widest text-muted-foreground">{enabledMapRegions}/{totalMapRegions}</span>
                    </div>
                )}
            </div>

            {/* Sites of Grace */}
            {activeSection === 'graces' && (
                <div className="grid grid-cols-1 gap-1.5 animate-in fade-in duration-200">
                    {Object.entries(regions).sort().map(([region, rg]) => {
                        const vc = rg.filter(g => g.visited).length;
                        const mapPath = getRegionMapPath(region);
                        return (
                            <div key={region} className="card overflow-hidden">
                                <div className={`w-full px-3 py-2 flex justify-between items-center transition-all ${expandedRegions[region] ? 'bg-muted/30 border-b border-border' : 'hover:bg-muted/10'}`}>
                                    <button onClick={() => setExpandedRegions(p => ({...p, [region]: !p[region]}))} className="flex-1 flex items-center space-x-2.5 text-left">
                                        <Arrow open={!!expandedRegions[region]} />
                                        <h2 className="text-[10px] font-black uppercase tracking-widest text-foreground">{region}</h2>
                                    </button>
                                    <div className="flex items-center space-x-2">
                                        {vc < rg.length && <button onClick={e => { e.stopPropagation(); handleUnlockRegionGraces(rg); }} className={`${btnSm} hover:text-primary hover:border-primary/50`}>Unlock</button>}
                                        {mapPath && (
                                            <button onClick={e => { e.stopPropagation(); setSelectedMap({name: region, path: mapPath}); }}
                                                className="w-7 h-7 rounded bg-muted/50 border border-border/50 flex items-center justify-center overflow-hidden hover:border-primary/50 hover:scale-110 transition-all group">
                                                <img src={mapPath} alt="Map" className="w-full h-full object-cover opacity-60 group-hover:opacity-100 transition-opacity" onError={e => (e.currentTarget.style.display = 'none')} />
                                            </button>
                                        )}
                                        <Badge count={vc} total={rg.length} />
                                    </div>
                                </div>
                                {expandedRegions[region] && (
                                    <div className="px-3 py-2 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-0.5 animate-in slide-in-from-top-1 duration-200">
                                        {rg.map(g => (
                                            <label key={g.id} className="flex items-center space-x-2 group cursor-pointer py-0.5 px-1.5 rounded hover:bg-muted/40 transition-all">
                                                <Chk checked={g.visited} onChange={v => handleGraceToggle(g, v)} />
                                                {g.isBossArena && <span className="text-amber-500 text-[10px] flex-shrink-0" title="Boss Arena">⚔</span>}
                                                {g.dungeonType === 'catacomb' && <span className="flex-shrink-0 text-violet-500" title="Catacomb"><svg className="w-2.5 h-2.5" viewBox="0 0 24 24" fill="currentColor"><path d="M3 21V7l9-5 9 5v14H3zm2-2h14V8l-7-3.9L5 8v11zm5-1v-4h4v4h-4zm0-6V8h4v4h-4z"/></svg></span>}
                                                {g.dungeonType === 'hero_grave' && <span className="flex-shrink-0 text-slate-400" title="Hero's Grave"><svg className="w-2.5 h-2.5" viewBox="0 0 24 24" fill="currentColor"><path d="M10 2h4v6h4l-6 6-6-6h4V2zm-7 18v-2h18v2H3z"/></svg></span>}
                                                <span className={`text-[10px] truncate font-semibold ${g.visited ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`} title={g.name}>
                                                    {g.name}
                                                </span>
                                                {g.isBossArena && <span className="flex-shrink-0 text-[7px] font-black uppercase tracking-wider px-1 py-px rounded bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30" title="Site of Grace appears after defeating a boss">Boss Arena</span>}
                                                {g.dungeonType === 'catacomb' && <span className="flex-shrink-0 text-[7px] font-black uppercase tracking-wider px-1 py-px rounded bg-violet-500/15 text-violet-600 dark:text-violet-400 border border-violet-500/30" title="Catacomb — sealed entrance doors">Catacomb</span>}
                                                {g.dungeonType === 'hero_grave' && <span className="flex-shrink-0 text-[7px] font-black uppercase tracking-wider px-1 py-px rounded bg-slate-500/15 text-slate-600 dark:text-slate-400 border border-slate-500/30" title="Hero's Grave — sealed entrance doors">Hero's Grave</span>}
                                            </label>
                                        ))}
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            )}

            {/* Bosses */}
            {activeSection === 'bosses' && (
                <div className="grid grid-cols-1 gap-1.5 animate-in fade-in duration-200">
                    {Object.entries(bossRegions).sort().map(([region, rb]) => {
                        const dc = rb.filter(b => b.defeated).length;
                        return (
                            <div key={region} className="card overflow-hidden">
                                <div className={`w-full px-3 py-2 flex justify-between items-center transition-all ${expandedBossRegions[region] ? 'bg-muted/30 border-b border-border' : 'hover:bg-muted/10'}`}>
                                    <button onClick={() => setExpandedBossRegions(p => ({...p, [region]: !p[region]}))} className="flex-1 flex items-center space-x-2.5 text-left">
                                        <Arrow open={!!expandedBossRegions[region]} />
                                        <h2 className="text-[10px] font-black uppercase tracking-widest text-foreground">{region}</h2>
                                        {rb.some(b => b.remembrance) && <span className="text-[7px] font-black uppercase text-amber-500/80 bg-amber-500/10 border border-amber-500/20 px-1 py-0 rounded">R</span>}
                                    </button>
                                    <div className="flex items-center space-x-2">
                                        {dc < rb.length && <button onClick={e => { e.stopPropagation(); handleKillAll(rb); }} className={`${btnSm} hover:text-red-400 hover:border-red-400/50`}>Kill</button>}
                                        {dc > 0 && <button onClick={e => { e.stopPropagation(); handleRespawnAll(rb); }} className={`${btnSm} hover:text-green-400 hover:border-green-400/50`}>Respawn</button>}
                                        <Badge count={dc} total={rb.length} activeCls={dc === rb.length ? 'text-red-400 border-red-400/50 bg-red-400/10' : dc > 0 ? 'text-amber-400 border-amber-400/50 bg-amber-400/10' : undefined} />
                                    </div>
                                </div>
                                {expandedBossRegions[region] && (
                                    <div className="px-3 py-2 grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-0.5 animate-in slide-in-from-top-1 duration-200">
                                        {rb.map(b => (
                                            <label key={b.id} className="flex items-center space-x-2 group cursor-pointer py-0.5 px-1.5 rounded hover:bg-muted/40 transition-all">
                                                <ChkX checked={b.defeated} onChange={v => handleBossToggle(b, v)} />
                                                <span className={`text-[10px] truncate font-semibold ${b.defeated ? 'text-foreground line-through opacity-60' : 'text-muted-foreground group-hover:text-foreground'}`} title={b.name}>
                                                    {b.name}
                                                </span>
                                                {b.remembrance && <span className="flex-shrink-0 text-[7px] font-black text-amber-500/70">R</span>}
                                                {b.type === 'main' && !b.remembrance && <span className="flex-shrink-0 text-[7px] font-black text-primary/70">M</span>}
                                            </label>
                                        ))}
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            )}

            {/* Summoning Pools */}
            {activeSection === 'pools' && (
                <div className="grid grid-cols-1 gap-1.5 animate-in fade-in duration-200">
                    {Object.entries(poolRegions).sort().map(([region, rp]) => {
                        const ac = rp.filter(p => p.activated).length;
                        return (
                            <div key={region} className="card overflow-hidden">
                                <div className={`w-full px-3 py-2 flex justify-between items-center transition-all ${expandedPoolRegions[region] ? 'bg-muted/30 border-b border-border' : 'hover:bg-muted/10'}`}>
                                    <button onClick={() => setExpandedPoolRegions(p => ({...p, [region]: !p[region]}))} className="flex-1 flex items-center space-x-2.5 text-left">
                                        <Arrow open={!!expandedPoolRegions[region]} />
                                        <h2 className="text-[10px] font-black uppercase tracking-widest text-foreground">{region}</h2>
                                    </button>
                                    <div className="flex items-center space-x-2">
                                        {ac < rp.length && <button onClick={e => { e.stopPropagation(); handleActivateAllPools(rp); }} className={`${btnSm} hover:text-primary hover:border-primary/50`}>Activate</button>}
                                        <Badge count={ac} total={rp.length} />
                                    </div>
                                </div>
                                {expandedPoolRegions[region] && (
                                    <div className="px-3 py-2 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-0.5 animate-in slide-in-from-top-1 duration-200">
                                        {rp.map(p => (
                                            <label key={p.id} className="flex items-center space-x-2 group cursor-pointer py-0.5 px-1.5 rounded hover:bg-muted/40 transition-all">
                                                <Chk checked={p.activated} onChange={v => handlePoolToggle(p, v)} />
                                                <span className={`text-[10px] truncate font-semibold ${p.activated ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`} title={p.name}>{p.name}</span>
                                            </label>
                                        ))}
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            )}

            {/* Colosseums */}
            {activeSection === 'colosseums' && (
                <div className="animate-in fade-in duration-200">
                    <div className="card px-3 py-2">
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
                            {colosseums.map(c => (
                                <label key={c.id} className="flex items-center space-x-3 group cursor-pointer py-2 px-3 rounded border border-border hover:border-primary/40 hover:bg-muted/30 transition-all">
                                    <Chk checked={c.unlocked} onChange={v => handleColosseumToggle(c, v)} />
                                    <div className="min-w-0">
                                        <p className={`text-[11px] font-black uppercase tracking-wide ${c.unlocked ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`}>{c.name}</p>
                                        <p className="text-[8px] font-bold text-muted-foreground uppercase tracking-widest">{c.region}</p>
                                    </div>
                                </label>
                            ))}
                        </div>
                    </div>
                </div>
            )}

            {/* Map Discovery */}
            {activeSection === 'map' && (
                <div className="grid grid-cols-1 gap-1.5 animate-in fade-in duration-200">
                    {mapSystemEntries.length > 0 && (
                        <div className="card px-3 py-2">
                            <div className="flex items-center flex-wrap gap-x-4 gap-y-1">
                                {mapSystemEntries.map(e => (
                                    <label key={e.id} className="flex items-center space-x-1.5 group cursor-pointer">
                                        <Chk checked={e.enabled} onChange={v => handleSystemFlagToggle(e, v)} />
                                        <span className={`text-[9px] font-bold uppercase tracking-widest ${e.enabled ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`}>{e.name}</span>
                                    </label>
                                ))}
                            </div>
                        </div>
                    )}
                    {Object.entries(mapAreas).sort(([a], [b]) => a.localeCompare(b)).map(([area, ae]) => {
                        const ec = ae.filter(e => e.enabled).length;
                        return (
                            <div key={area} className="card overflow-hidden">
                                <div className={`w-full px-3 py-2 flex justify-between items-center transition-all ${expandedMapAreas[area] ? 'bg-muted/30 border-b border-border' : 'hover:bg-muted/10'}`}>
                                    <button onClick={() => setExpandedMapAreas(p => ({...p, [area]: !p[area]}))} className="flex-1 flex items-center space-x-2.5 text-left">
                                        <Arrow open={!!expandedMapAreas[area]} />
                                        <h2 className="text-[10px] font-black uppercase tracking-widest text-foreground">{area}</h2>
                                    </button>
                                    <Badge count={ec} total={ae.length} />
                                </div>
                                {expandedMapAreas[area] && (
                                    <div className="px-3 py-2 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-0.5 animate-in slide-in-from-top-1 duration-200">
                                        {ae.map(e => (
                                            <label key={e.id} className="flex items-center space-x-2 group cursor-pointer py-0.5 px-1.5 rounded hover:bg-muted/40 transition-all">
                                                <Chk checked={e.enabled} onChange={v => handleMapRegionToggle(e, v)} />
                                                <span className={`text-[10px] font-semibold truncate ${e.enabled ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`}>{e.name}</span>
                                            </label>
                                        ))}
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
