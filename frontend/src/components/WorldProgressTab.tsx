import {useEffect, useState} from 'react';
import {GetGraces, SetGraceVisited, GetBosses, SetBossDefeated} from '../../wailsjs/go/main/App';
import {db} from '../../wailsjs/go/models';

interface WorldProgressTabProps {
    charIdx: number;
    onMutate?: () => void;
}

export function WorldProgressTab({charIdx, onMutate}: WorldProgressTabProps) {
    const [graces, setGraces] = useState<db.GraceEntry[]>([]);
    const [bosses, setBosses] = useState<db.BossEntry[]>([]);
    const [loading, setLoading] = useState(false);
    const [expandedRegions, setExpandedRegions] = useState<Record<string, boolean>>({});
    const [expandedBossRegions, setExpandedBossRegions] = useState<Record<string, boolean>>({});
    const [selectedMap, setSelectedMap] = useState<{name: string, path: string} | null>(null);
    const [bossFilter, setBossFilter] = useState<'all' | 'main' | 'field'>('all');
    const [bossSort, setBossSort] = useState<'name' | 'defeated'>('name');
    const [activeSection, setActiveSection] = useState<'graces' | 'bosses'>('graces');

    const loadData = () => {
        setLoading(true);
        Promise.all([
            GetGraces(charIdx).then(res => setGraces(res || [])),
            GetBosses(charIdx).then(res => setBosses(res || []))
        ]).finally(() => setLoading(false));
    };

    useEffect(() => {
        loadData();
    }, [charIdx]);

    // --- Grace logic ---
    const regions = graces.reduce((acc, grace) => {
        const region = grace.region || 'Unknown';
        if (!acc[region]) acc[region] = [];
        acc[region].push(grace);
        return acc;
    }, {} as Record<string, db.GraceEntry[]>);

    const toggleRegion = (region: string) => {
        setExpandedRegions(prev => ({...prev, [region]: !prev[region]}));
    };

    const handleGraceToggle = async (grace: db.GraceEntry, visited: boolean) => {
        await SetGraceVisited(charIdx, grace.id, visited);
        setGraces(prev => prev.map(g => g.id === grace.id ? {...g, visited} : g));
        onMutate?.();
    };

    const handleUnlockAll = async (regionGraces: db.GraceEntry[]) => {
        await Promise.all(
            regionGraces
                .filter(g => !g.visited)
                .map(g => SetGraceVisited(charIdx, g.id, true))
        );
        const ids = new Set(regionGraces.map(g => g.id));
        setGraces(prev => prev.map(g => ids.has(g.id) ? {...g, visited: true} : g));
        onMutate?.();
    };

    // --- Boss logic ---
    const filteredBosses = bosses.filter(b => bossFilter === 'all' || b.type === bossFilter);

    const sortedFilteredBosses = [...filteredBosses].sort((a, b) => {
        if (bossSort === 'defeated') {
            if (a.defeated !== b.defeated) return a.defeated ? -1 : 1;
        }
        return a.name.localeCompare(b.name);
    });

    const bossRegions = sortedFilteredBosses.reduce((acc, boss) => {
        const region = boss.region || 'Unknown';
        if (!acc[region]) acc[region] = [];
        acc[region].push(boss);
        return acc;
    }, {} as Record<string, db.BossEntry[]>);

    const toggleBossRegion = (region: string) => {
        setExpandedBossRegions(prev => ({...prev, [region]: !prev[region]}));
    };

    const handleBossToggle = async (boss: db.BossEntry, defeated: boolean) => {
        await SetBossDefeated(charIdx, boss.id, defeated);
        setBosses(prev => prev.map(b => b.id === boss.id ? {...b, defeated} : b));
        onMutate?.();
    };

    const handleKillAll = async (regionBosses: db.BossEntry[]) => {
        await Promise.all(
            regionBosses
                .filter(b => !b.defeated)
                .map(b => SetBossDefeated(charIdx, b.id, true))
        );
        const ids = new Set(regionBosses.map(b => b.id));
        setBosses(prev => prev.map(b => ids.has(b.id) ? {...b, defeated: true} : b));
        onMutate?.();
    };

    const handleRespawnAll = async (regionBosses: db.BossEntry[]) => {
        await Promise.all(
            regionBosses
                .filter(b => b.defeated)
                .map(b => SetBossDefeated(charIdx, b.id, false))
        );
        const ids = new Set(regionBosses.map(b => b.id));
        setBosses(prev => prev.map(b => ids.has(b.id) ? {...b, defeated: false} : b));
        onMutate?.();
    };

    // --- Global boss actions ---
    const handleGlobalKillAll = async () => {
        const alive = filteredBosses.filter(b => !b.defeated);
        if (alive.length === 0) return;
        await Promise.all(alive.map(b => SetBossDefeated(charIdx, b.id, true)));
        const ids = new Set(alive.map(b => b.id));
        setBosses(prev => prev.map(b => ids.has(b.id) ? {...b, defeated: true} : b));
        onMutate?.();
    };

    const handleGlobalRespawnAll = async () => {
        const dead = filteredBosses.filter(b => b.defeated);
        if (dead.length === 0) return;
        await Promise.all(dead.map(b => SetBossDefeated(charIdx, b.id, false)));
        const ids = new Set(dead.map(b => b.id));
        setBosses(prev => prev.map(b => ids.has(b.id) ? {...b, defeated: false} : b));
        onMutate?.();
    };

    // --- Map helpers ---
    const REGION_MAP_ALIASES: Record<string, string | null> = {
        'limgrave': 'limgrave',
        'limgrave, west': 'limgrave',
        'limgrave, east': 'limgrave',
        'liurnia of the lakes': 'liurnia_of_the_lakes',
        'liurnia, north': 'liurnia_of_the_lakes',
        'liurnia, east': 'liurnia_of_the_lakes',
        'liurnia, west': 'liurnia_of_the_lakes',
        'weeping peninsula': null,
        'crumbling farum azula': null,
        "miquella's haligtree": null,
        'shadow of the erdtree': null,
    };

    const getRegionMapPath = (region: string): string | null => {
        const keyNorm = region.toLowerCase();
        if (keyNorm in REGION_MAP_ALIASES) {
            const val = REGION_MAP_ALIASES[keyNorm];
            return val ? `maps/${val}.jpg` : null;
        }
        const cleanName = region.toLowerCase()
            .replace(/'/g, '')
            .replace(/,/g, '')
            .replace(/\s+/g, '_');
        return `maps/${cleanName}.jpg`;
    };

    // --- Stats ---
    const totalGraces = graces.length;
    const visitedGraces = graces.filter(g => g.visited).length;
    const totalBosses = bosses.length;
    const defeatedBosses = bosses.filter(b => b.defeated).length;
    const mainBosses = bosses.filter(b => b.type === 'main');
    const defeatedMain = mainBosses.filter(b => b.defeated).length;

    if (loading) return (
        <div className="py-20 flex flex-col items-center justify-center space-y-4">
            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Scanning world data...</p>
        </div>
    );

    return (
        <div className="flex-1 min-h-0 space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-12 overflow-y-auto custom-scrollbar pr-2">
            {/* Map Popover */}
            {selectedMap && (
                <div
                    className="fixed inset-0 z-50 flex items-center justify-center bg-background/90 backdrop-blur-sm animate-in fade-in duration-300 p-4 md:p-12"
                    onClick={() => setSelectedMap(null)}
                >
                    <div className="relative max-w-5xl w-full h-full flex flex-col items-center justify-center animate-in zoom-in-95 duration-300">
                        <img
                            src={selectedMap.path}
                            alt={selectedMap.name}
                            className="max-w-full max-h-full object-contain rounded-lg shadow-2xl shadow-primary/20 border border-border/50"
                            onError={(e) => (e.currentTarget.src = '/src/assets/images/logo-universal.png')}
                        />
                        <div className="absolute bottom-4 left-1/2 -translate-x-1/2 bg-background/80 backdrop-blur-md px-6 py-3 rounded-full border border-border/50 shadow-xl">
                            <h3 className="text-sm font-black uppercase tracking-widest text-foreground text-center">{selectedMap.name}</h3>
                            <p className="text-[9px] font-bold text-muted-foreground uppercase tracking-[0.3em] text-center mt-1">Click anywhere to close</p>
                        </div>
                    </div>
                </div>
            )}

            {/* Section Tabs + Stats */}
            <div className="flex items-center justify-between">
                <div className="flex items-center space-x-1">
                    <button
                        onClick={() => setActiveSection('graces')}
                        className={`px-4 py-1.5 rounded-full text-[9px] font-black uppercase tracking-[0.15em] transition-all ${activeSection === 'graces' ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20' : 'text-muted-foreground hover:text-foreground hover:bg-muted/30'}`}
                    >
                        Sites of Grace
                    </button>
                    <button
                        onClick={() => setActiveSection('bosses')}
                        className={`px-4 py-1.5 rounded-full text-[9px] font-black uppercase tracking-[0.15em] transition-all ${activeSection === 'bosses' ? 'bg-primary text-primary-foreground shadow-lg shadow-primary/20' : 'text-muted-foreground hover:text-foreground hover:bg-muted/30'}`}
                    >
                        Bosses
                    </button>
                </div>

                {activeSection === 'graces' ? (
                    <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">
                        {visitedGraces}/{totalGraces} discovered
                    </span>
                ) : (
                    <div className="flex items-center space-x-3">
                        <button
                            onClick={handleGlobalKillAll}
                            className="text-[9px] font-black uppercase tracking-widest text-muted-foreground hover:text-red-400 border border-border/50 hover:border-red-400/50 px-2.5 py-1 rounded transition-all"
                            title="Defeat All Bosses"
                        >
                            Kill All
                        </button>
                        <button
                            onClick={handleGlobalRespawnAll}
                            className="text-[9px] font-black uppercase tracking-widest text-muted-foreground hover:text-green-400 border border-border/50 hover:border-green-400/50 px-2.5 py-1 rounded transition-all"
                            title="Respawn All Bosses"
                        >
                            Respawn All
                        </button>
                        <div className="w-px h-4 bg-border/50" />
                        <div className="flex items-center space-x-1">
                            {(['all', 'main', 'field'] as const).map(f => (
                                <button
                                    key={f}
                                    onClick={() => setBossFilter(f)}
                                    className={`px-2.5 py-1 rounded text-[8px] font-black uppercase tracking-widest transition-all ${bossFilter === f ? 'bg-muted text-foreground border border-border' : 'text-muted-foreground hover:text-foreground'}`}
                                >
                                    {f}
                                </button>
                            ))}
                        </div>
                        <div className="w-px h-4 bg-border/50" />
                        <div className="flex items-center space-x-1">
                            {(['name', 'defeated'] as const).map(s => (
                                <button
                                    key={s}
                                    onClick={() => setBossSort(s)}
                                    className={`px-2.5 py-1 rounded text-[8px] font-black uppercase tracking-widest transition-all ${bossSort === s ? 'bg-muted text-foreground border border-border' : 'text-muted-foreground hover:text-foreground'}`}
                                >
                                    {s}
                                </button>
                            ))}
                        </div>
                        <div className="w-px h-4 bg-border/50" />
                        <span className="text-[9px] font-black uppercase tracking-widest text-muted-foreground">
                            {defeatedMain}/{mainBosses.length} main | {defeatedBosses}/{totalBosses} total
                        </span>
                    </div>
                )}
            </div>

            {/* Sites of Grace Section */}
            {activeSection === 'graces' && (
                <div className="grid grid-cols-1 gap-4 animate-in fade-in duration-300">
                    {Object.entries(regions).sort().map(([region, regionGraces]) => {
                        const visitedCount = regionGraces.filter(g => g.visited).length;
                        const total = regionGraces.length;
                        const allVisited = visitedCount === total;

                        return (
                            <div key={region} className="card overflow-hidden">
                                <div className={`w-full px-5 py-4 flex justify-between items-center transition-all ${expandedRegions[region] ? 'bg-muted/30 border-b border-border' : 'hover:bg-muted/10'}`}>
                                    <button
                                        onClick={() => toggleRegion(region)}
                                        className="flex-1 flex items-center space-x-4 text-left"
                                    >
                                        <div className={`transition-transform duration-300 ${expandedRegions[region] ? 'rotate-90 text-primary' : 'text-muted-foreground'}`}>
                                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M9 5l7 7-7 7"></path>
                                            </svg>
                                        </div>
                                        <h2 className="text-xs font-black uppercase tracking-widest text-foreground">{region}</h2>
                                    </button>

                                    <div className="flex items-center space-x-3">
                                        {!allVisited && (
                                            <button
                                                onClick={(e) => { e.stopPropagation(); handleUnlockAll(regionGraces); }}
                                                className="text-[9px] font-black uppercase tracking-widest text-muted-foreground hover:text-primary border border-border/50 hover:border-primary/50 px-2 py-0.5 rounded transition-all"
                                                title="Unlock All Graces in Region"
                                            >
                                                Unlock All
                                            </button>
                                        )}
                                        {(() => {
                                            const mapPath = getRegionMapPath(region);
                                            if (!mapPath) return null;
                                            return (
                                                <button
                                                    onClick={(e) => {
                                                        e.stopPropagation();
                                                        setSelectedMap({ name: region, path: mapPath });
                                                    }}
                                                    className="w-10 h-10 rounded bg-muted/50 border border-border/50 flex items-center justify-center overflow-hidden hover:border-primary/50 hover:scale-110 transition-all group"
                                                    title={`View ${region} Map`}
                                                >
                                                    <img
                                                        src={mapPath}
                                                        alt="Map"
                                                        className="w-full h-full object-cover opacity-60 group-hover:opacity-100 transition-opacity"
                                                        onError={(e) => (e.currentTarget.style.display = 'none')}
                                                    />
                                                </button>
                                            );
                                        })()}
                                        <span className={`text-[9px] font-black uppercase tracking-widest px-2 py-0.5 rounded border ${allVisited ? 'text-primary border-primary/50 bg-primary/10' : 'text-muted-foreground bg-muted/50 border-border'}`}>
                                            {visitedCount}/{total}
                                        </span>
                                    </div>
                                </div>

                                {expandedRegions[region] && (
                                    <div className="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-x-8 gap-y-3 animate-in slide-in-from-top-2 duration-300">
                                        {regionGraces.map(grace => (
                                            <label key={grace.id} className="flex items-center space-x-3 group cursor-pointer py-1.5 px-2 rounded-md hover:bg-muted/40 transition-all">
                                                <div className="relative flex items-center justify-center">
                                                    <input
                                                        type="checkbox"
                                                        checked={grace.visited}
                                                        onChange={(e) => handleGraceToggle(grace, e.target.checked)}
                                                        className="peer appearance-none w-4 h-4 rounded border border-border bg-background checked:bg-primary checked:border-primary transition-all cursor-pointer focus:ring-2 focus:ring-primary/20"
                                                    />
                                                    <svg className="absolute w-2.5 h-2.5 text-white pointer-events-none hidden peer-checked:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3.5" d="M5 13l4 4L19 7"></path>
                                                    </svg>
                                                </div>
                                                <span className={`text-[11px] transition-colors truncate font-semibold ${grace.visited ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`} title={grace.name}>
                                                    {grace.name}
                                                </span>
                                            </label>
                                        ))}
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            )}

            {/* Bosses Section */}
            {activeSection === 'bosses' && (
                <div className="grid grid-cols-1 gap-4 animate-in fade-in duration-300">
                    {Object.entries(bossRegions).sort().map(([region, regionBosses]) => {
                        const defeatedCount = regionBosses.filter(b => b.defeated).length;
                        const total = regionBosses.length;
                        const allDefeated = defeatedCount === total;
                        const noneDefeated = defeatedCount === 0;
                        const hasRemembrance = regionBosses.some(b => b.remembrance);

                        return (
                            <div key={region} className="card overflow-hidden">
                                <div className={`w-full px-5 py-4 flex justify-between items-center transition-all ${expandedBossRegions[region] ? 'bg-muted/30 border-b border-border' : 'hover:bg-muted/10'}`}>
                                    <button
                                        onClick={() => toggleBossRegion(region)}
                                        className="flex-1 flex items-center space-x-4 text-left"
                                    >
                                        <div className={`transition-transform duration-300 ${expandedBossRegions[region] ? 'rotate-90 text-primary' : 'text-muted-foreground'}`}>
                                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2.5" d="M9 5l7 7-7 7"></path>
                                            </svg>
                                        </div>
                                        <div className="flex items-center space-x-2">
                                            <h2 className="text-xs font-black uppercase tracking-widest text-foreground">{region}</h2>
                                            {hasRemembrance && (
                                                <span className="text-[8px] font-black uppercase tracking-widest text-amber-500/80 bg-amber-500/10 border border-amber-500/20 px-1.5 py-0.5 rounded">
                                                    Remembrance
                                                </span>
                                            )}
                                        </div>
                                    </button>

                                    <div className="flex items-center space-x-3">
                                        {!allDefeated && (
                                            <button
                                                onClick={(e) => { e.stopPropagation(); handleKillAll(regionBosses); }}
                                                className="text-[9px] font-black uppercase tracking-widest text-muted-foreground hover:text-red-400 border border-border/50 hover:border-red-400/50 px-2 py-0.5 rounded transition-all"
                                                title="Defeat All Bosses in Region"
                                            >
                                                Kill All
                                            </button>
                                        )}
                                        {!noneDefeated && (
                                            <button
                                                onClick={(e) => { e.stopPropagation(); handleRespawnAll(regionBosses); }}
                                                className="text-[9px] font-black uppercase tracking-widest text-muted-foreground hover:text-green-400 border border-border/50 hover:border-green-400/50 px-2 py-0.5 rounded transition-all"
                                                title="Respawn All Bosses in Region"
                                            >
                                                Respawn All
                                            </button>
                                        )}
                                        <span className={`text-[9px] font-black uppercase tracking-widest px-2 py-0.5 rounded border ${allDefeated ? 'text-red-400 border-red-400/50 bg-red-400/10' : noneDefeated ? 'text-muted-foreground bg-muted/50 border-border' : 'text-amber-400 border-amber-400/50 bg-amber-400/10'}`}>
                                            {defeatedCount}/{total}
                                        </span>
                                    </div>
                                </div>

                                {expandedBossRegions[region] && (
                                    <div className="p-6 grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-3 animate-in slide-in-from-top-2 duration-300">
                                        {regionBosses.map(boss => (
                                            <label key={boss.id} className="flex items-center space-x-3 group cursor-pointer py-2 px-3 rounded-md hover:bg-muted/40 transition-all">
                                                <div className="relative flex items-center justify-center">
                                                    <input
                                                        type="checkbox"
                                                        checked={boss.defeated}
                                                        onChange={(e) => handleBossToggle(boss, e.target.checked)}
                                                        className="peer appearance-none w-4 h-4 rounded border border-border bg-background checked:bg-red-500 checked:border-red-500 transition-all cursor-pointer focus:ring-2 focus:ring-red-500/20"
                                                    />
                                                    <svg className="absolute w-2.5 h-2.5 text-white pointer-events-none hidden peer-checked:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3.5" d="M6 18L18 6M6 6l12 12"></path>
                                                    </svg>
                                                </div>
                                                <div className="flex items-center space-x-2 min-w-0">
                                                    <span className={`text-[11px] transition-colors truncate font-semibold ${boss.defeated ? 'text-foreground line-through opacity-60' : 'text-muted-foreground group-hover:text-foreground'}`} title={boss.name}>
                                                        {boss.name}
                                                    </span>
                                                    {boss.remembrance && (
                                                        <span className="flex-shrink-0 text-[8px] font-black text-amber-500/70" title="Remembrance Boss">
                                                            R
                                                        </span>
                                                    )}
                                                    {boss.type === 'main' && !boss.remembrance && (
                                                        <span className="flex-shrink-0 text-[8px] font-black text-primary/70" title="Main Boss">
                                                            M
                                                        </span>
                                                    )}
                                                </div>
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
