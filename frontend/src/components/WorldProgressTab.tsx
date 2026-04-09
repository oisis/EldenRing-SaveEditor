import {useEffect, useState} from 'react';
import {GetGraces, SetGraceVisited} from '../../wailsjs/go/main/App';
import {db} from '../../wailsjs/go/models';

interface WorldProgressTabProps {
    charIdx: number;
}

export function WorldProgressTab({charIdx}: WorldProgressTabProps) {
    const [graces, setGraces] = useState<db.GraceEntry[]>([]);
    const [loading, setLoading] = useState(false);
    const [expandedRegions, setExpandedRegions] = useState<Record<string, boolean>>({});
    const [selectedMap, setSelectedMap] = useState<{name: string, path: string} | null>(null);

    const loadGraces = () => {
        setLoading(true);
        GetGraces(charIdx).then(res => {
            setGraces(res || []);
            setLoading(false);
        }).catch(err => {
            console.error("Failed to load graces:", err);
            setLoading(false);
        });
    };

    useEffect(() => {
        loadGraces();
    }, [charIdx]);

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
    };

    const handleUnlockAll = async (regionGraces: db.GraceEntry[]) => {
        await Promise.all(
            regionGraces
                .filter(g => !g.visited)
                .map(g => SetGraceVisited(charIdx, g.id, true))
        );
        const ids = new Set(regionGraces.map(g => g.id));
        setGraces(prev => prev.map(g => ids.has(g.id) ? {...g, visited: true} : g));
    };

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
        // Generic: keep period before underscore (e.g. "Mt. Gelmir" → "mt._gelmir")
        const cleanName = region.toLowerCase()
            .replace(/'/g, '')
            .replace(/,/g, '')
            .replace(/\s+/g, '_');
        return `maps/${cleanName}.jpg`;
    };

    if (loading) return (
        <div className="py-20 flex flex-col items-center justify-center space-y-4">
            <div className="w-6 h-6 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
            <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Scanning world data...</p>
        </div>
    );

    return (
        <div className="flex-1 min-h-0 space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700 pb-12 overflow-y-auto custom-scrollbar pr-2">
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

            <div className="flex items-center space-x-2 px-1">
                <div className="w-1 h-3 bg-primary rounded-full" />
                <h3 className="text-[10px] font-black uppercase tracking-[0.2em] text-muted-foreground">Sites of Grace</h3>
            </div>

            <div className="grid grid-cols-1 gap-4">
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
        </div>
    );
}
